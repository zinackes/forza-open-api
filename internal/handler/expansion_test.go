// Tests golden des endpoints ajoutés par l'expansion du contrat : obtain,
// search, stories/tours, gets par id (avec 404 RFC 9457), filtres
// manufacturers et reference enrichie. Même harnais que catalog_test.go
// (Postgres jetable, db/init.sql, goldens via -update).
//
// Nécessite Docker (cf. .claude/rules/testing.md). En CI, Docker est présent.
package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/zinackes/forza-open-api/internal/handler"
	"github.com/zinackes/forza-open-api/internal/oas"
	"github.com/zinackes/forza-open-api/internal/store"
)

// newSeededExpansionServer lève un Postgres jetable et sème un jeu de données
// couvrant toutes les voies d'obtention + les ressources Discovery. Timestamps
// explicites : les goldens restent stables.
func newSeededExpansionServer(t *testing.T) http.Handler {
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

	seeds := []string{
		`INSERT INTO cars (id, game, name, make, class, pi, drivetrain, value_cr, obtain_method) VALUES
			('hidden','fh6','Hidden Legend','Cindra','S1',880,'AWD',NULL,'reward'),
			('owner','fh6','Perk Owner','Cindra','A',800,'RWD',45000,'autoshow'),
			('plain','fh6','Plain Car','Volcan','D',500,'FWD',12000,'autoshow')`,
		`INSERT INTO manufacturers (game, name, country) VALUES
			('fh6','Cindra','Japan'),
			('fh6','Volcan','Italy')`,
		`INSERT INTO dlc_packs (id, game, name, kind, released_at) VALUES
			('pack','fh6','Legends Car Pack','car_pass','2026-05-01T00:00:00Z')`,
		`INSERT INTO car_dlc (car_id, dlc_id) VALUES ('hidden','pack')`,
		`INSERT INTO barn_finds (id, car_id, game, region, prerequisite_stamp_level) VALUES
			('bf1','hidden','fh6','Hakone',3)`,
		`INSERT INTO journal_tiers (id, game, track, level, color, name, points_required, reward_car_id) VALUES
			('jt1','fh6','horizon_festival',7,'gold','Gold',12000,'hidden')`,
		`INSERT INTO car_mastery_perks (id, car_id, name, effect_type, unlocked_car_id) VALUES
			('perk1','owner','Hidden unlock','car_unlock','hidden')`,
		`INSERT INTO tracks (id, game, name, type, region, length_m) VALUES
			('shibuya','fh6','Shibuya Circuit','street','Tokyo',2400)`,
		`INSERT INTO stories (id, game, name, region, chapters_count, reward_description) VALUES
			('story1','fh6','Taxi Tales','Tokyo',8,'Hidden Legend')`,
		`INSERT INTO tours (id, game, name, region) VALUES
			('tour1','fh6','Tour of Tokyo','Tokyo')`,
	}
	for _, q := range seeds {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatalf("seed expansion: %v\nSQL: %s", err, q)
		}
	}

	srv, err := oas.NewServer(handler.New(&store.Store{DB: pool}), handler.SecurityHandler{},
		oas.WithErrorHandler(handler.ProblemErrorHandler))
	if err != nil {
		t.Fatalf("oas.NewServer: %v", err)
	}
	return srv
}

// TestExpansionGolden compare la réponse HTTP de chaque nouvel endpoint à son
// fichier golden (404 RFC 9457 compris).
func TestExpansionGolden(t *testing.T) {
	srv := newSeededExpansionServer(t)

	cases := []struct {
		name   string
		target string
		status int
	}{
		// obtain : toutes les voies, voiture « simple », 404.
		{"obtain_full", "/v1/cars/hidden/obtain", http.StatusOK},
		{"obtain_plain", "/v1/cars/plain/obtain", http.StatusOK},
		{"obtain_not_found", "/v1/cars/ghost/obtain", http.StatusNotFound},
		// search : multi-ressources, restriction kinds, q trop court → 400 ogen.
		{"search_multi", "/v1/search?game=fh6&q=legend", http.StatusOK},
		{"search_kinds", "/v1/search?game=fh6&q=legend&kinds=car", http.StatusOK},
		{"search_q_too_short", "/v1/search?game=fh6&q=l", http.StatusBadRequest},
		// gets par id + 404.
		{"gettrack_found", "/v1/tracks/shibuya", http.StatusOK},
		{"gettrack_not_found", "/v1/tracks/ghost", http.StatusNotFound},
		{"getdlcpack_found", "/v1/dlc-packs/pack", http.StatusOK},
		{"getbarnfind_found", "/v1/barn-finds/bf1", http.StatusOK},
		// filtres ajoutés.
		{"manufacturers_country", "/v1/manufacturers?game=fh6&country=Japan", http.StatusOK},
		{"barnfinds_car_id", "/v1/barn-finds?game=fh6&car_id=hidden", http.StatusOK},
		{"dlcpacks_kind", "/v1/dlc-packs?game=fh6&kind=car_pass", http.StatusOK},
		// Discovery.
		{"stories", "/v1/stories?game=fh6", http.StatusOK},
		{"tours", "/v1/tours?game=fh6", http.StatusOK},
		// reference enrichie (regions + types canoniques, counts 0 inclus).
		{"reference_expansion", "/v1/reference?game=fh6", http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.target, nil)

			srv.ServeHTTP(rec, req)

			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, tc.status, rec.Body.String())
			}
			assertGolden(t, tc.name, rec.Body.Bytes())
		})
	}
}
