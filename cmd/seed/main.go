// Command seed orchestre l'ingestion du catalogue depuis des sources PROPRES
// (datasets communautaires, exports/API wiki Fandom). Jamais de scraping du jeu,
// lecture mémoire, injection ni botting. Upserts idempotents (ON CONFLICT),
// rejouables sans doublon.
//
// Usage :
//
//	seed tracks   <dataset.json | https://…>  ingère un dataset de tracés.
//	seed cars     [game]                      ingère le catalogue voitures (déf. fh6)
//	                                           depuis l'API MediaWiki du wiki Forza.
//	seed playlist [game] [SxxWx]              ingère la série/saison courante de la
//	                                           Festival Playlist depuis forza.net +
//	                                           forums.forza.net (déf. fh6, courante).
//	seed playlist-history [game]              backfill de TOUT l'historique Festival
//	                                           Playlist (S1 → courante) depuis l'API
//	                                           MediaWiki du wiki Forza (déf. fh6).
//	seed exports  [game]                      régénère les archives téléchargeables
//	                                           du dataset (JSON/CSV/JSONL) par jeu
//	                                           (déf. tous les EXPORTS_GAMES).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"text/tabwriter"
	"time"

	"github.com/zinackes/forza-open-api/internal/config"
	"github.com/zinackes/forza-open-api/internal/export"
	"github.com/zinackes/forza-open-api/internal/health"
	"github.com/zinackes/forza-open-api/internal/ingest/cars"
	"github.com/zinackes/forza-open-api/internal/ingest/playlist"
	"github.com/zinackes/forza-open-api/internal/ingest/tracks"
	"github.com/zinackes/forza-open-api/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if len(os.Args) < 2 {
		usage(logger)
		return
	}
	switch os.Args[1] {
	case "tracks":
		seedTracks(logger)
	case "cars":
		seedCars(logger)
	case "playlist":
		seedPlaylist(logger)
	case "playlist-history":
		seedPlaylistHistory(logger)
	case "exports":
		seedExports(logger)
	case "health":
		seedHealth(logger)
	default:
		usage(logger)
	}
}

func usage(logger *slog.Logger) {
	logger.Info("usage: seed tracks <dataset.json|URL> | seed cars [game] | seed playlist [game] [SxxWx] | seed playlist-history [game] | seed exports [game] | seed health")
}

func seedTracks(logger *slog.Logger) {
	if len(os.Args) < 3 {
		logger.Info("usage: seed tracks <dataset.json | URL>")
		os.Exit(1)
	}
	src := os.Args[2]
	ctx := context.Background()

	st := mustStore(ctx, logger)
	defer st.Close()
	monitor := newMonitor(st, logger)
	started := time.Now()

	raw, err := tracks.FetchDataset(ctx, src)
	if err != nil {
		failRun(ctx, monitor, "tracks", "", started, fmt.Errorf("fetch dataset %q: %w", src, err))
	}
	parsed, skipped, err := tracks.ParseTracks(raw, time.Now().UTC())
	if err != nil {
		failRun(ctx, monitor, "tracks", "", started, fmt.Errorf("parse dataset %q: %w", src, err))
	}
	if err := st.UpsertTracks(ctx, parsed); err != nil {
		failRun(ctx, monitor, "tracks", "", started, fmt.Errorf("upsert tracks: %w", err))
	}

	monitor.Observe(ctx, health.Report{
		Source: "tracks", Records: len(parsed), Violations: tracks.CheckRun(len(parsed), skipped),
	}, started)
	logger.Info("seed tracks ok", "upserted", len(parsed), "skipped", skipped, "src", src)
}

// carSource associe un jeu à sa page liste et au template de ligne du wiki Forza.
type carSource struct {
	listPage     string
	listTemplate string
}

var carSources = map[string]carSource{
	"fh6": {listPage: "Forza Horizon 6/Cars", listTemplate: "CarListStatsFH6"},
	"fh5": {listPage: "Forza Horizon 5/Cars", listTemplate: "CarListStatsFH5"},
}

func seedCars(logger *slog.Logger) {
	game := "fh6"
	if len(os.Args) >= 3 {
		game = os.Args[2]
	}
	src, ok := carSources[game]
	if !ok {
		logger.Error("seed cars: jeu non supporté", "game", game)
		os.Exit(1)
	}

	// Réseau borné : l'ingestion ne doit pas pendre indéfiniment.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	st := mustStore(ctx, logger)
	defer st.Close()
	monitor := newMonitor(st, logger)
	started := time.Now()

	logger.Info("seed cars: fetch list", "game", game, "page", src.listPage)
	listWT, err := cars.FetchCarListWikitext(ctx, src.listPage)
	if err != nil {
		failRun(ctx, monitor, "cars", game, started, fmt.Errorf("fetch car list: %w", err))
	}
	rows := cars.ParseCarList(listWT, src.listTemplate)
	titles := cars.CarTitles(rows)
	logger.Info("seed cars: list parsed", "rows", len(rows), "distinct_titles", len(titles))

	logger.Info("seed cars: fetch car pages (batched)", "titles", len(titles))
	pages, err := cars.FetchCarPages(ctx, titles)
	if err != nil {
		failRun(ctx, monitor, "cars", game, started, fmt.Errorf("fetch car pages: %w", err))
	}
	infoboxes := make(map[string]cars.Infobox, len(pages))
	for title, wt := range pages {
		infoboxes[title] = cars.ParseInfobox(wt)
	}
	logger.Info("seed cars: pages fetched", "pages", len(pages), "missing", len(titles)-len(pages))

	catalogue, rep := cars.Normalize(rows, infoboxes, game)

	// Étape qualité : valide les invariants du contrat (class/pi/drivetrain,
	// champs requis) sans bloquer l'ingestion. Les entrées douteuses sont
	// seedées mais signalées ; les entrées qui violeraient une contrainte DB
	// (CHECK/NOT NULL) sont écartées de l'upsert pour ne pas avorter le batch.
	vrep := cars.Validate(catalogue, game)
	if n := vrep.Count(cars.SevWarning); n > 0 {
		logger.Warn("seed cars: entrées douteuses (seedées, à investiguer)",
			"count", n, "rules", cars.SortedReport(vrep.RuleCounts()))
	}
	if n := vrep.Count(cars.SevBlocking); n > 0 {
		blocking := vrep.BlockingIDs()
		kept := make([]store.Car, 0, len(catalogue))
		for _, c := range catalogue {
			if _, bad := blocking[c.ID]; !bad {
				kept = append(kept, c)
			}
		}
		logger.Error("seed cars: anomalies BLOQUANTES écartées de l'upsert (contrainte violée)",
			"count", n, "dropped_cars", len(catalogue)-len(kept),
			"rules", cars.SortedReport(vrep.RuleCounts()))
		catalogue = kept
	}

	if err := st.UpsertCars(ctx, catalogue); err != nil {
		failRun(ctx, monitor, "cars", game, started, fmt.Errorf("upsert cars: %w", err))
	}

	// Constructeurs dérivés de la même page liste (make + code pays, champ 14) :
	// alimente GET /v1/manufacturers dans la même passe, même idempotence.
	mans := cars.Manufacturers(rows, infoboxes, game)
	if err := st.UpsertManufacturers(ctx, mans); err != nil {
		failRun(ctx, monitor, "cars", game, started, fmt.Errorf("upsert manufacturers: %w", err))
	}
	logger.Info("seed cars: manufacturers upserted", "count", len(mans))

	// Contrôle de santé : counts dans la fourchette attendue, taux de parse, 0
	// anomalie bloquante (rupture de structure wiki = anomaly silencieuse).
	monitor.Observe(ctx, health.Report{
		Source: "cars", Game: game, Records: len(catalogue), Violations: cars.CheckRun(rep, vrep, game),
	}, started)

	logger.Info("seed cars ok",
		"game", game,
		"rows", rep.Rows,
		"imported", rep.Imported,
		"upserted", len(catalogue),
		"distinct_upserted", rep.Distinct,
		"deduped", rep.Imported-rep.Distinct,
		"skipped", rep.Rows-rep.Imported,
		"skip_reasons", cars.SortedReport(rep.SkipReasons),
		"classes", cars.SortedReport(rep.Classes),
		"drivetrains", cars.SortedReport(rep.Drivetrains),
	)
	if len(rep.Collisions) > 0 {
		logger.Info("seed cars: doublons de listing dédupliqués (last-wins)", "ids", cars.SortedReport(rep.Collisions))
	}
	if len(rep.UnknownCodes) > 0 {
		logger.Warn("seed cars: codes non mappés (anomalies)", "codes", cars.SortedReport(rep.UnknownCodes))
	}
}

// seedPlaylist ingère la série Festival Playlist visée : « seed playlist [game]
// [SxxWx] ». Sans SxxWx, la série courante est découverte depuis forza.net/events ;
// avec, une semaine précise est ciblée (ex. S01W2). Upsert idempotent.
func seedPlaylist(logger *slog.Logger) {
	game := "fh6"
	if len(os.Args) >= 3 {
		game = os.Args[2]
	}
	override := ""
	if len(os.Args) >= 4 {
		override = os.Args[3]
	}

	// Réseau borné (deux sources + délai poli) : l'ingestion ne pend pas.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	now := time.Now().UTC()

	st := mustStore(ctx, logger)
	defer st.Close()
	monitor := newMonitor(st, logger)
	started := time.Now()

	logger.Info("seed playlist: fetch", "game", game, "override", override)
	res, err := playlist.IngestCurrent(ctx, st, game, override, now, logger)
	if err != nil {
		failRun(ctx, monitor, "playlist", game, started, fmt.Errorf("ingest playlist: %w", err))
	}

	// Contrôle de santé : nom/dates/récompenses/défis non vides (parse forza.net +
	// forum OK), série courante non périmée.
	monitor.Observe(ctx, health.Report{
		Source: "playlist", Game: game, Records: 1, Violations: playlist.CheckRun(res, now),
	}, started)

	ser := res.Series
	logger.Info("seed playlist ok",
		"game", ser.Game,
		"id", ser.ID,
		"series", ser.Series,
		"week", ser.Week,
		"season", deref(ser.Season),
		"name", deref(ser.Name),
		"starts_at", ser.StartsAt,
		"ends_at", ser.EndsAt,
		"is_current", ser.IsCurrent,
		"rewards", res.Rewards,
		"challenges", res.Challenges,
		"divergences", res.Divergences,
		"new", res.Added,
	)
}

// seedPlaylistHistory backfill TOUT l'historique Festival Playlist d'un jeu depuis
// l'API MediaWiki du wiki Forza : « seed playlist-history [game] » (déf. fh6).
// One-shot idempotent (upsert sur l'id stable de chaque saison) ; ensuite maintenu
// par le scheduler (carte 3.5). is_current est décidé par la fenêtre de dates
// (now ∈ [start,end]) ; UpsertSeries éteint les autres séries courantes du jeu.
func seedPlaylistHistory(logger *slog.Logger) {
	game := "fh6"
	if len(os.Args) >= 3 {
		game = os.Args[2]
	}

	// Réseau borné mais généreux : découverte + des centaines de pages par lots
	// (FH5 ≈ 61 séries × 4 saisons), avec délais polis entre lots.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	now := time.Now().UTC()

	st := mustStore(ctx, logger)
	defer st.Close()
	monitor := newMonitor(st, logger)
	started := time.Now()

	logger.Info("seed playlist-history: backfill", "game", game)
	seasons, err := playlist.BackfillPlaylist(ctx, game, now)
	if err != nil {
		failRun(ctx, monitor, "playlist-history", game, started, fmt.Errorf("backfill playlist: %w", err))
	}

	added, updated := 0, 0
	for _, s := range seasons {
		isNew, err := st.UpsertSeries(ctx, s.Series, s.Rewards, s.Challenges)
		if err != nil {
			failRun(ctx, monitor, "playlist-history", game, started, fmt.Errorf("upsert series %s: %w", s.Series.ID, err))
		}
		if isNew {
			added++
		} else {
			updated++
		}
	}

	// Backfill one-shot : seul invariant utile = découverte non vide (0 saison =
	// rupture de la découverte wiki).
	var vios []health.Violation
	if len(seasons) == 0 {
		vios = []health.Violation{{Rule: "zero_records", Detail: "0 saison backfillée (découverte wiki vide ?)", Severity: health.SevAnomaly}}
	}
	monitor.Observe(ctx, health.Report{
		Source: "playlist-history", Game: game, Records: len(seasons), Violations: vios,
	}, started)

	logger.Info("seed playlist-history ok",
		"game", game,
		"seasons", len(seasons),
		"added", added,
		"updated", updated,
	)
}

// seedExports régénère les archives téléchargeables (GET /v1/exports) : « seed
// exports [game] » (déf. tous les EXPORTS_GAMES). Dump complet par jeu → fichiers
// statiques (R2 si configuré, sinon EXPORTS_DIR) + manifeste. Idempotent, rejouable.
func seedExports(logger *slog.Logger) {
	cfg := config.Load()
	games := cfg.ExportsGames
	if len(os.Args) >= 3 {
		games = []string{os.Args[2]}
	}

	// Borne large : le dump couvre toutes les ressources de chaque jeu.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	now := time.Now().UTC()

	st := mustStore(ctx, logger)
	defer st.Close()
	monitor := newMonitor(st, logger)

	uploader, err := export.UploaderFor(export.R2Config{
		Endpoint:        cfg.R2Endpoint,
		Bucket:          cfg.R2Bucket,
		AccessKeyID:     cfg.R2AccessKeyID,
		SecretAccessKey: cfg.R2SecretAccessKey,
	}, cfg.ExportsDir)
	if err != nil {
		logger.Error("seed exports: backend", "err", err)
		os.Exit(1)
	}

	for _, game := range games {
		started := time.Now()
		res, err := export.Generate(ctx, st, uploader, cfg.ExportsBaseURL, game, now)
		if err != nil {
			failRun(ctx, monitor, "exports", game, started, fmt.Errorf("generate exports: %w", err))
		}
		// Contrôle de santé : dataset non entièrement vide (jeu non seedé = anomalie).
		monitor.Observe(ctx, health.Report{
			Source: "exports", Game: game, Records: res.Artifacts, Violations: export.CheckRun(res, now),
		}, started)
		logger.Info("seed exports ok", "game", game, "artifacts", res.Artifacts, "rows", res.Rows)
	}
}

// deref rend la valeur d'un *string pour le log (vide si nil).
func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func mustStore(ctx context.Context, logger *slog.Logger) *store.Store {
	st, err := store.New(ctx, config.Load())
	if err != nil {
		logger.Error("store init", "err", err)
		os.Exit(1)
	}
	return st
}

// newMonitor construit le moniteur de santé du seed : persiste chaque run dans
// scrape_runs et alerte (webhook ALERT_WEBHOOK_URL si configuré, sinon log seul).
func newMonitor(st *store.Store, logger *slog.Logger) *health.Monitor {
	return &health.Monitor{
		Store:    st,
		Notifier: health.NewNotifier(config.Load().AlertWebhookURL),
		Logger:   logger,
	}
}

// failRun journalise l'échec dur d'un run (log + persistance + alerte via le
// Monitor) puis sort en code != 0. Centralise les chemins d'erreur des seeds pour
// qu'un échec soit toujours tracé dans scrape_runs et alerté.
func failRun(ctx context.Context, m *health.Monitor, source, game string, started time.Time, err error) {
	m.Observe(ctx, health.Report{Source: source, Game: game, Err: err}, started)
	os.Exit(1)
}

// seedHealth imprime le dashboard ops de fraîcheur / échecs par source : le
// dernier run de chaque source/jeu (statut, âge, volume, détail). Sortie tabulaire
// sur stdout (sortie primaire de la commande, distincte des logs slog).
func seedHealth(logger *slog.Logger) {
	ctx := context.Background()
	st := mustStore(ctx, logger)
	defer st.Close()

	runs, err := st.LatestScrapeRuns(ctx)
	if err != nil {
		logger.Error("seed health: query", "err", err)
		os.Exit(1)
	}
	if len(runs) == 0 {
		logger.Info("seed health: aucun run enregistré")
		return
	}

	now := time.Now().UTC()
	tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	// Les erreurs d'écriture du tabwriter sont différées et remontent au Flush.
	_, _ = fmt.Fprintln(tw, "SOURCE\tGAME\tLAST RUN\tSTATUS\tAGE\tRECORDS\tDETAIL")
	for _, r := range runs {
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%d\t%s\n",
			r.Source, gameLabel(r.Game), r.FinishedAt.UTC().Format("2006-01-02 15:04"),
			r.Status, humanAge(now.Sub(r.FinishedAt)), r.Records, runDetail(r))
	}
	_ = tw.Flush()
}

// gameLabel rend le jeu d'un run (« - » pour les sources sans jeu, ex. tracks).
func gameLabel(g *string) string {
	if g == nil || *g == "" {
		return "-"
	}
	return *g
}

// humanAge formate une ancienneté en m/h/j (lecture rapide du dashboard).
func humanAge(d time.Duration) string {
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

// runDetail rend la colonne DETAIL : l'erreur pour un échec, la 1re anomalie pour
// une anomalie, « - » pour un run sain.
func runDetail(r store.ScrapeRunSummary) string {
	if r.Error != nil && *r.Error != "" {
		return truncate(*r.Error, 60)
	}
	if len(r.Violations) > 0 {
		var vs []health.Violation
		if json.Unmarshal(r.Violations, &vs) == nil {
			for _, v := range vs {
				if v.Severity == health.SevAnomaly {
					return truncate(v.Rule+" ("+v.Detail+")", 60)
				}
			}
			if len(vs) > 0 {
				return truncate(vs[0].Rule, 60)
			}
		}
	}
	return "-"
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
