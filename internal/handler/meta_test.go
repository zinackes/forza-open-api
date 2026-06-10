// Test du endpoint méta : GET /v1/meta. Réponse non golden (generatedAt varie) →
// assertions structurelles. Sème deux jeux (fh6, fh5) et un journal data_changes
// à timestamps explicites pour vérifier counts + fraîcheur catalogue/playlist,
// les omissions honnêtes (playlist jamais ingérée), l'en-tête de cache court et
// la version de data optionnelle (env DATA_VERSION).
package handler_test

import (
	"context"
	"encoding/json"
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

// Instants d'ingestion semés, distincts pour distinguer MAX par (jeu, ressource).
var (
	fh6CarT1  = time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	fh6CarT2  = time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC) // plus récent → catalog fh6
	fh6Series = time.Date(2026, 6, 2, 9, 0, 0, 0, time.UTC)  // playlist fh6
	fh5CarT   = time.Date(2026, 5, 20, 8, 0, 0, 0, time.UTC) // catalog fh5
	otherTs   = time.Date(2026, 6, 4, 0, 0, 0, 0, time.UTC)  // resource ignorée (dlc_pack)
)

// newMetaPool lève un Postgres jetable (db/init.sql) et sème cars + data_changes
// pour les deux jeux. fh5 n'a aucun changement de série → playlist omise.
func newMetaPool(t *testing.T) *pgxpool.Pool {
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
		// fh6 : 3 voitures, fh5 : 2 voitures.
		`INSERT INTO cars (id, game, name, make, class, pi, drivetrain) VALUES
			('fh6-a','fh6','A','Make','A',800,'RWD'),
			('fh6-b','fh6','B','Make','A',810,'AWD'),
			('fh6-c','fh6','C','Make','B',700,'FWD'),
			('fh5-a','fh5','A5','Make','A',800,'RWD'),
			('fh5-b','fh5','B5','Make','S1',900,'AWD')`,
		// Journal d'ingestion : catalogue (car) fh6 à T1/T2, playlist (series) fh6,
		// catalogue fh5, et une ressource ignorée (dlc_pack) pour vérifier le filtre.
		`INSERT INTO data_changes (game, resource, resource_id, action, occurred_at) VALUES
			('fh6','car','fh6-a','added',$1),
			('fh6','car','fh6-b','added',$2),
			('fh6','series','fh6-s1','added',$3),
			('fh5','car','fh5-a','added',$4),
			('fh6','dlc_pack','fh6-pack','added',$5)`,
	}
	if _, err := pool.Exec(ctx, seeds[0]); err != nil {
		t.Fatalf("seed cars: %v", err)
	}
	if _, err := pool.Exec(ctx, seeds[1], fh6CarT1, fh6CarT2, fh6Series, fh5CarT, otherTs); err != nil {
		t.Fatalf("seed data_changes: %v", err)
	}
	return pool
}

// metaResp décode le corps JSON de /v1/meta (vue locale, indépendante d'ogen).
type metaResp struct {
	Games []struct {
		Game              string     `json:"game"`
		CarCount          int64      `json:"carCount"`
		CatalogUpdatedAt  *time.Time `json:"catalogUpdatedAt"`
		PlaylistUpdatedAt *time.Time `json:"playlistUpdatedAt"`
	} `json:"games"`
	DataVersion *string   `json:"dataVersion"`
	GeneratedAt time.Time `json:"generatedAt"`
}

func getMeta(t *testing.T, srv http.Handler) (*httptest.ResponseRecorder, metaResp) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/meta", nil)
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200\nbody: %s", rec.Code, rec.Body.String())
	}
	var got metaResp
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode meta: %v\nbody: %s", err, rec.Body.String())
	}
	return rec, got
}

func newMetaServer(t *testing.T, pool *pgxpool.Pool, dataVersion string) http.Handler {
	t.Helper()
	srv, err := oas.NewServer(handler.New(&store.Store{DB: pool}, dataVersion), handler.SecurityHandler{},
		oas.WithErrorHandler(handler.ProblemErrorHandler))
	if err != nil {
		t.Fatalf("oas.NewServer: %v", err)
	}
	return srv
}

func TestMeta(t *testing.T) {
	pool := newMetaPool(t)
	srv := newMetaServer(t, pool, "")

	rec, got := getMeta(t, srv)

	if cc := rec.Header().Get("Cache-Control"); cc != "public, max-age=60, stale-while-revalidate=300" {
		t.Errorf("Cache-Control = %q, want cache court", cc)
	}
	if got.DataVersion != nil {
		t.Errorf("dataVersion = %v, want absent (env non tamponné)", *got.DataVersion)
	}
	if got.GeneratedAt.IsZero() {
		t.Error("generatedAt absent")
	}

	// Tous les jeux supportés (enum Game) sont listés, dans l'ordre du contrat.
	if len(got.Games) != 2 {
		t.Fatalf("games = %d, want 2 (fh6, fh5)", len(got.Games))
	}
	if got.Games[0].Game != "fh6" || got.Games[1].Game != "fh5" {
		t.Fatalf("ordre des jeux = %s,%s, want fh6,fh5", got.Games[0].Game, got.Games[1].Game)
	}

	fh6 := got.Games[0]
	if fh6.CarCount != 3 {
		t.Errorf("fh6 carCount = %d, want 3", fh6.CarCount)
	}
	if fh6.CatalogUpdatedAt == nil || !fh6.CatalogUpdatedAt.Equal(fh6CarT2) {
		t.Errorf("fh6 catalogUpdatedAt = %v, want MAX %v", fh6.CatalogUpdatedAt, fh6CarT2)
	}
	if fh6.PlaylistUpdatedAt == nil || !fh6.PlaylistUpdatedAt.Equal(fh6Series) {
		t.Errorf("fh6 playlistUpdatedAt = %v, want %v", fh6.PlaylistUpdatedAt, fh6Series)
	}

	fh5 := got.Games[1]
	if fh5.CarCount != 2 {
		t.Errorf("fh5 carCount = %d, want 2", fh5.CarCount)
	}
	if fh5.CatalogUpdatedAt == nil || !fh5.CatalogUpdatedAt.Equal(fh5CarT) {
		t.Errorf("fh5 catalogUpdatedAt = %v, want %v", fh5.CatalogUpdatedAt, fh5CarT)
	}
	// fh5 n'a aucun changement de série → playlist omise (rien d'inventé).
	if fh5.PlaylistUpdatedAt != nil {
		t.Errorf("fh5 playlistUpdatedAt = %v, want absent", *fh5.PlaylistUpdatedAt)
	}
}

// TestMetaDataVersion : la version de data tamponnée (env DATA_VERSION) apparaît.
func TestMetaDataVersion(t *testing.T) {
	pool := newMetaPool(t)
	srv := newMetaServer(t, pool, "fh6-2026.06")

	_, got := getMeta(t, srv)

	if got.DataVersion == nil || *got.DataVersion != "fh6-2026.06" {
		t.Errorf("dataVersion = %v, want fh6-2026.06", got.DataVersion)
	}
}
