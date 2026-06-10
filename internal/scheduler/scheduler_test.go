// Tests du Runner du scheduler sur Postgres jetable (testcontainers). L'ingestion
// réelle (réseau) est remplacée par une IngestFunc qui upsert des fixtures figées
// (cf. testing.md : pas de réseau en test). Vérifie : run manuel + re-run sans
// doublon, bascule is_current sur la nouvelle saison, qu'un échec sur un jeu est
// alerté (webhook) + persisté sans avorter les autres jeux, et qu'une ANOMALIE
// SILENCIEUSE (parse « réussi » mais données suspectes) déclenche aussi l'alerte.
// Nécessite Docker.
package scheduler_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/zinackes/forza-open-api/internal/health"
	"github.com/zinackes/forza-open-api/internal/scheduler"
	"github.com/zinackes/forza-open-api/internal/store"
)

// discardLogger : les tests n'inspectent pas les logs, on les jette.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// seasonFixture construit une saison déterministe (ids stables) prête pour
// UpsertSeries — l'équivalent figé de ce que playlist.IngestCurrent produirait.
func seasonFixture(game string, series, week int, current bool) (store.Series, []store.Reward, []store.Challenge) {
	id := fmt.Sprintf("%s-s%02dw%d", game, series, week)
	season, saga := "Autumn", "Welcome to Japan"
	start := time.Date(2026, 5, 28, 14, 30, 0, 0, time.UTC)
	end := time.Date(2026, 6, 4, 14, 30, 0, 0, time.UTC)
	p33, p67 := 33, 67
	req, rw := "Win a Race", "1 point"

	ser := store.Series{
		ID: id, Game: game, Series: series, Name: &saga, Season: &season,
		Week: week, StartsAt: start, EndsAt: end, IsCurrent: current,
	}
	rewards := []store.Reward{
		{ID: id + "-rw01", SeriesID: id, AtPercent: &p33, Type: "series", Item: "Mazda Furai"},
		{ID: id + "-rw02", SeriesID: id, AtPercent: &p67, Type: "series", Item: "Nissan 370Z"},
	}
	challenges := []store.Challenge{
		{ID: id + "-ch01", SeriesID: id, Scope: "weekly", Name: "King of the Road", Requirement: &req, Reward: &rw, ExpiresAt: &end},
		{ID: id + "-ch02", SeriesID: id, Scope: "daily", Name: "Big in Japan", Requirement: &req, Reward: &rw},
	}
	return ser, rewards, challenges
}

// TestRunPlaylistIdempotentAndCurrentToggle : un run manuel suivi d'un re-run
// identique ne crée pas de doublon, et l'ingestion d'une nouvelle semaine bascule
// is_current sur la nouvelle saison (invariant « une seule courante par jeu »).
func TestRunPlaylistIdempotentAndCurrentToggle(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	// La semaine « courante » servie par la fausse ingestion est pilotable.
	week := 2
	r := &scheduler.Runner{
		Source:  "playlist",
		Games:   []string{"fh6"},
		Logger:  discardLogger(),
		Monitor: &health.Monitor{Store: st, Logger: discardLogger()},
		Ingest: func(ctx context.Context, game string, _ time.Time) (health.Report, error) {
			ser, rw, ch := seasonFixture(game, 1, week, true)
			_, err := st.UpsertSeries(ctx, ser, rw, ch)
			return health.Report{Records: 1}, err
		},
	}
	now := time.Now().UTC()

	// 1) Run manuel.
	if err := r.Run(ctx, now); err != nil {
		t.Fatalf("run #1: %v", err)
	}
	// 2) Re-run identique : rejouable sans doublon.
	if err := r.Run(ctx, now); err != nil {
		t.Fatalf("run #2 (re-run): %v", err)
	}

	if got := queryInt(t, st, `SELECT count(*) FROM series WHERE game='fh6'`); got != 1 {
		t.Errorf("séries fh6 = %d, want 1 (pas de doublon)", got)
	}
	if got := queryInt(t, st, `SELECT count(*) FROM rewards WHERE series_id='fh6-s01w2'`); got != 2 {
		t.Errorf("rewards = %d, want 2 (enfants remplacés, pas accumulés)", got)
	}
	if got := queryInt(t, st, `SELECT count(*) FROM challenges WHERE series_id='fh6-s01w2'`); got != 2 {
		t.Errorf("challenges = %d, want 2", got)
	}
	if got := queryInt(t, st, `SELECT count(*) FROM series WHERE game='fh6' AND is_current`); got != 1 {
		t.Errorf("séries courantes = %d, want 1", got)
	}

	// Chaque run sain est journalisé dans scrape_runs (dashboard de fraîcheur).
	if got := queryInt(t, st, `SELECT count(*) FROM scrape_runs WHERE source='playlist' AND status='ok'`); got != 2 {
		t.Errorf("scrape_runs ok = %d, want 2 (un par run)", got)
	}

	// 3) Reset hebdo : la nouvelle semaine devient courante, l'ancienne s'éteint.
	week = 3
	if err := r.Run(ctx, now); err != nil {
		t.Fatalf("run #3 (rollover): %v", err)
	}
	if got := queryInt(t, st, `SELECT count(*) FROM series WHERE game='fh6'`); got != 2 {
		t.Errorf("séries fh6 = %d, want 2 (w2 conservée + w3)", got)
	}
	if got := queryInt(t, st, `SELECT count(*) FROM series WHERE game='fh6' AND is_current`); got != 1 {
		t.Errorf("séries courantes après rollover = %d, want 1", got)
	}
	if got := queryString(t, st, `SELECT id FROM series WHERE game='fh6' AND is_current`); got != "fh6-s01w3" {
		t.Errorf("série courante = %q, want fh6-s01w3", got)
	}
}

// TestRunPlaylistAlertsOnFailureAndContinues : l'échec dur d'un jeu déclenche une
// alerte webhook (status=failed), est persisté et agrégé dans l'erreur retournée,
// mais n'empêche pas le traitement des autres jeux.
func TestRunPlaylistAlertsOnFailureAndContinues(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	alerts := newAlertSink()
	srv := alerts.server()
	defer srv.Close()

	r := &scheduler.Runner{
		Source:  "playlist",
		Games:   []string{"fh6", "fh5"},
		Logger:  discardLogger(),
		Monitor: &health.Monitor{Store: st, Notifier: health.NewNotifier(srv.URL), Logger: discardLogger()},
		Ingest: func(ctx context.Context, game string, _ time.Time) (health.Report, error) {
			if game == "fh6" {
				return health.Report{}, errors.New("boom: forza.net 500")
			}
			ser, rw, ch := seasonFixture(game, 45, 1, true)
			_, err := st.UpsertSeries(ctx, ser, rw, ch)
			return health.Report{Records: 1}, err
		},
	}

	err := r.Run(ctx, time.Now().UTC())
	if err == nil {
		t.Fatal("Run = nil, want erreur agrégée (fh6 a échoué)")
	}
	if !strings.Contains(err.Error(), "fh6") || !strings.Contains(err.Error(), "boom") {
		t.Errorf("erreur agrégée = %q, want mention fh6 + cause", err)
	}

	// fh5 traité malgré l'échec de fh6 (un jeu en échec n'avorte pas les autres).
	if got := queryInt(t, st, `SELECT count(*) FROM series WHERE id='fh5-s45w1'`); got != 1 {
		t.Errorf("fh5 series = %d, want 1 (traité malgré l'échec fh6)", got)
	}

	// Une seule alerte, pour fh6, avec la cause, le service et le statut.
	got := alerts.all()
	if len(got) != 1 {
		t.Fatalf("alertes = %d, want 1 (fh6 seul)", len(got))
	}
	a := got[0]
	if a.Game != "fh6" || a.Source != "playlist" || a.Status != health.StatusFailed || !strings.Contains(a.Error, "boom") {
		t.Errorf("alerte = %+v, want game=fh6 source=playlist status=failed error~boom", a)
	}

	// Persistance : fh6 failed, fh5 ok.
	if s := queryString(t, st, `SELECT status FROM scrape_runs WHERE source='playlist' AND game='fh6'`); s != "failed" {
		t.Errorf("scrape_runs fh6 status = %q, want failed", s)
	}
	if s := queryString(t, st, `SELECT status FROM scrape_runs WHERE source='playlist' AND game='fh5'`); s != "ok" {
		t.Errorf("scrape_runs fh5 status = %q, want ok", s)
	}
}

// TestRunPlaylistAlertsOnAnomaly : un parse « réussi » mais aux invariants violés
// (ici 0 récompense — signature d'un changement de structure forza.net) déclenche
// une alerte status=anomaly, est persisté, et fait échouer le run (-once non-zéro)
// sans qu'aucune erreur dure n'ait été levée.
func TestRunPlaylistAlertsOnAnomaly(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	alerts := newAlertSink()
	srv := alerts.server()
	defer srv.Close()

	r := &scheduler.Runner{
		Source:  "playlist",
		Games:   []string{"fh6"},
		Logger:  discardLogger(),
		Monitor: &health.Monitor{Store: st, Notifier: health.NewNotifier(srv.URL), Logger: discardLogger()},
		Ingest: func(_ context.Context, _ string, _ time.Time) (health.Report, error) {
			return health.Report{Records: 1, Violations: []health.Violation{
				{Rule: "no_rewards", Detail: "0 récompense", Severity: health.SevAnomaly},
			}}, nil
		},
	}

	err := r.Run(ctx, time.Now().UTC())
	if err == nil || !strings.Contains(err.Error(), "anomalie") {
		t.Fatalf("Run = %v, want erreur mentionnant une anomalie", err)
	}

	got := alerts.all()
	if len(got) != 1 || got[0].Status != health.StatusAnomaly {
		t.Fatalf("alertes = %+v, want 1 alerte status=anomaly", got)
	}
	if got[0].Violations[0].Rule != "no_rewards" {
		t.Errorf("violation = %q, want no_rewards", got[0].Violations[0].Rule)
	}
	if s := queryString(t, st, `SELECT status FROM scrape_runs WHERE source='playlist' AND game='fh6'`); s != "anomaly" {
		t.Errorf("scrape_runs status = %q, want anomaly", s)
	}
}

// alertSink capture les alertes POSTées sur un webhook httptest (concurrence-safe).
type alertSink struct {
	mu   sync.Mutex
	hits []health.Alert
}

func newAlertSink() *alertSink { return &alertSink{} }

func (s *alertSink) server() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var a health.Alert
		_ = json.NewDecoder(r.Body).Decode(&a)
		s.mu.Lock()
		s.hits = append(s.hits, a)
		s.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
}

func (s *alertSink) all() []health.Alert {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]health.Alert(nil), s.hits...)
}

// newTestStore lève un Postgres jetable, applique db/init.sql, et renvoie un Store
// branché dessus (Redis nil : non utilisé par le scheduler).
func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	ctx := context.Background()

	initScript, err := filepath.Abs(filepath.Join("..", "..", "db", "init.sql"))
	if err != nil {
		t.Fatalf("abs init.sql: %v", err)
	}

	pg, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithInitScripts(initScript),
		tcpostgres.WithDatabase("forza"),
		tcpostgres.WithUsername("forza"),
		tcpostgres.WithPassword("forza"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("démarrage Postgres (Docker requis): %v", err)
	}
	t.Cleanup(func() { _ = pg.Terminate(context.Background()) })

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool: %v", err)
	}
	t.Cleanup(pool.Close)

	return &store.Store{DB: pool}
}

func queryInt(t *testing.T, st *store.Store, sql string) int64 {
	t.Helper()
	var n int64
	if err := st.DB.QueryRow(context.Background(), sql).Scan(&n); err != nil {
		t.Fatalf("query int: %v\nSQL: %s", err, sql)
	}
	return n
}

func queryString(t *testing.T, st *store.Store, sql string) string {
	t.Helper()
	var s string
	if err := st.DB.QueryRow(context.Background(), sql).Scan(&s); err != nil {
		t.Fatalf("query string: %v\nSQL: %s", err, sql)
	}
	return s
}
