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
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/zinackes/forza-open-api/internal/config"
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
	default:
		usage(logger)
	}
}

func usage(logger *slog.Logger) {
	logger.Info("usage: seed tracks <dataset.json|URL> | seed cars [game] | seed playlist [game] [SxxWx] | seed playlist-history [game]")
}

func seedTracks(logger *slog.Logger) {
	if len(os.Args) < 3 {
		logger.Info("usage: seed tracks <dataset.json | URL>")
		os.Exit(1)
	}
	src := os.Args[2]
	ctx := context.Background()

	raw, err := tracks.FetchDataset(ctx, src)
	if err != nil {
		logger.Error("fetch dataset", "src", src, "err", err)
		os.Exit(1)
	}
	parsed, skipped, err := tracks.ParseTracks(raw, time.Now().UTC())
	if err != nil {
		logger.Error("parse dataset", "src", src, "err", err)
		os.Exit(1)
	}

	st := mustStore(ctx, logger)
	defer st.Close()

	if err := st.UpsertTracks(ctx, parsed); err != nil {
		logger.Error("upsert tracks", "err", err)
		os.Exit(1)
	}
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

	logger.Info("seed cars: fetch list", "game", game, "page", src.listPage)
	listWT, err := cars.FetchCarListWikitext(ctx, src.listPage)
	if err != nil {
		logger.Error("fetch car list", "err", err)
		os.Exit(1)
	}
	rows := cars.ParseCarList(listWT, src.listTemplate)
	titles := cars.CarTitles(rows)
	logger.Info("seed cars: list parsed", "rows", len(rows), "distinct_titles", len(titles))

	logger.Info("seed cars: fetch car pages (batched)", "titles", len(titles))
	pages, err := cars.FetchCarPages(ctx, titles)
	if err != nil {
		logger.Error("fetch car pages", "err", err)
		os.Exit(1)
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

	st := mustStore(ctx, logger)
	defer st.Close()

	if err := st.UpsertCars(ctx, catalogue); err != nil {
		logger.Error("upsert cars", "err", err)
		os.Exit(1)
	}

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

	logger.Info("seed playlist: fetch", "game", game, "override", override)
	ev, fd, err := playlist.FetchCurrent(ctx, game, override, now)
	if err != nil {
		logger.Error("fetch playlist", "err", err)
		os.Exit(1)
	}

	// Validation croisée forza.net (primaire) ↔ titre forum (secondaire) : logue
	// toute divergence (filet si forza.net change de structure) et complète les
	// champs d'identité que forza.net ne fournirait plus.
	ev, div := playlist.Reconcile(ev, fd)
	div.Log(logger)

	// Auto-discovery → c'est la série courante (rollover géré par l'upsert).
	// Override d'une semaine précise → courante seulement si la fenêtre couvre now
	// (un re-seed d'historique ne vole pas le flag « courante »).
	isCurrent := override == "" ||
		(!ev.Start.IsZero() && !ev.End.IsZero() && !ev.Start.After(now) && !ev.End.Before(now))

	ser, rewards, challenges := playlist.Normalize(ev, fd, isCurrent)

	st := mustStore(ctx, logger)
	defer st.Close()

	added, err := st.UpsertSeries(ctx, ser, rewards, challenges)
	if err != nil {
		logger.Error("upsert series", "err", err)
		os.Exit(1)
	}

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
		"rewards", len(rewards),
		"challenges", len(challenges),
		"divergences", len(div.Divergences),
		"new", added,
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

	logger.Info("seed playlist-history: backfill", "game", game)
	seasons, err := playlist.BackfillPlaylist(ctx, game, now)
	if err != nil {
		logger.Error("backfill playlist", "err", err)
		os.Exit(1)
	}

	st := mustStore(ctx, logger)
	defer st.Close()

	added, updated := 0, 0
	for _, s := range seasons {
		isNew, err := st.UpsertSeries(ctx, s.Series, s.Rewards, s.Challenges)
		if err != nil {
			logger.Error("upsert series", "id", s.Series.ID, "err", err)
			os.Exit(1)
		}
		if isNew {
			added++
		} else {
			updated++
		}
	}

	logger.Info("seed playlist-history ok",
		"game", game,
		"seasons", len(seasons),
		"added", added,
		"updated", updated,
	)
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
