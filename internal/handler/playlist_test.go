// Test bout-en-bout des endpoints Festival Playlist en lecture
// (/v1/playlist/series, /series/{id}, /current) à travers le serveur ogen, sur un
// Postgres jetable (db/init.sql). Vérifie le mapping DB → contrat : liste sans
// enfants, détail hydraté (rewards/challenges), atPercent optionnel (NULL omis),
// série courante, et 404 RFC 9457 sur un id inconnu. Nécessite Docker.
package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
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

func newSeededPlaylistServer(t *testing.T) http.Handler {
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
	st := &store.Store{DB: pool}

	saga, season := "Welcome to Japan", "Autumn"
	start := time.Date(2026, 5, 28, 14, 30, 0, 0, time.UTC)
	end := time.Date(2026, 6, 4, 14, 30, 0, 0, time.UTC)
	p33 := 33
	req, rw := "Win a Race", "25,000 Credits"
	ser := store.Series{ID: "fh6-s01w2", Game: "fh6", Series: 1, Name: &saga, Season: &season, Week: 2, StartsAt: start, EndsAt: end, IsCurrent: true}
	rewards := []store.Reward{
		{ID: "fh6-s01w2-rw01", SeriesID: ser.ID, AtPercent: &p33, Type: "series", Item: "Mazda Furai"},
		{ID: "fh6-s01w2-rw02", SeriesID: ser.ID, Type: "event", Item: "Forza LINK: shiny"}, // at_percent NULL
	}
	challenges := []store.Challenge{
		{ID: "fh6-s01w2-ch01", SeriesID: ser.ID, Scope: "weekly", Name: "King of the Road", Requirement: &req, Reward: &rw, ExpiresAt: &end},
	}
	if _, err := st.UpsertSeries(ctx, ser, rewards, challenges); err != nil {
		t.Fatalf("seed series: %v", err)
	}

	srv, err := oas.NewServer(handler.New(st, ""), handler.SecurityHandler{},
		oas.WithErrorHandler(handler.ProblemErrorHandler))
	if err != nil {
		t.Fatalf("oas.NewServer: %v", err)
	}
	return srv
}

type seriesJSON struct {
	ID        string `json:"id"`
	Game      string `json:"game"`
	Series    int    `json:"series"`
	Name      string `json:"name"`
	Season    string `json:"season"`
	Week      int    `json:"week"`
	IsCurrent bool   `json:"isCurrent"`
	Rewards   []struct {
		ID        string `json:"id"`
		AtPercent *int   `json:"atPercent"`
		Item      string `json:"item"`
	} `json:"rewards"`
	Challenges []struct {
		ID    string `json:"id"`
		Scope string `json:"scope"`
		Name  string `json:"name"`
	} `json:"challenges"`
}

func doGET(t *testing.T, srv http.Handler, target string) (int, []byte) {
	t.Helper()
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec.Code, rec.Body.Bytes()
}

func TestPlaylistEndpoints(t *testing.T) {
	srv := newSeededPlaylistServer(t)

	// Liste : la série est requêtable, sans enfants (liste légère).
	code, body := doGET(t, srv, "/v1/playlist/series?game=fh6")
	if code != http.StatusOK {
		t.Fatalf("list status = %d\n%s", code, body)
	}
	var list []seriesJSON
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("decode list: %v\n%s", err, body)
	}
	if len(list) != 1 || list[0].ID != "fh6-s01w2" || list[0].Name != "Welcome to Japan" || list[0].Season != "Autumn" {
		t.Fatalf("list = %+v", list)
	}
	if len(list[0].Rewards) != 0 || len(list[0].Challenges) != 0 {
		t.Errorf("list devrait être sans enfants, got rewards=%d challenges=%d", len(list[0].Rewards), len(list[0].Challenges))
	}

	// Détail : hydraté ; atPercent NULL omis sur le reward event.
	code, body = doGET(t, srv, "/v1/playlist/series/fh6-s01w2")
	if code != http.StatusOK {
		t.Fatalf("detail status = %d\n%s", code, body)
	}
	var detail seriesJSON
	if err := json.Unmarshal(body, &detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if len(detail.Rewards) != 2 || len(detail.Challenges) != 1 {
		t.Fatalf("detail children: rewards=%d challenges=%d", len(detail.Rewards), len(detail.Challenges))
	}
	if detail.Rewards[0].AtPercent == nil || *detail.Rewards[0].AtPercent != 33 {
		t.Errorf("reward[0].atPercent = %v, want 33", detail.Rewards[0].AtPercent)
	}
	if detail.Rewards[1].AtPercent != nil {
		t.Errorf("reward[1].atPercent = %v, want omis (NULL)", *detail.Rewards[1].AtPercent)
	}
	if detail.Challenges[0].Scope != "weekly" || detail.Challenges[0].Name != "King of the Road" {
		t.Errorf("challenge = %+v", detail.Challenges[0])
	}

	// Courante.
	code, body = doGET(t, srv, "/v1/playlist/current?game=fh6")
	if code != http.StatusOK {
		t.Fatalf("current status = %d\n%s", code, body)
	}
	var current seriesJSON
	if err := json.Unmarshal(body, &current); err != nil {
		t.Fatalf("decode current: %v", err)
	}
	if current.ID != "fh6-s01w2" || !current.IsCurrent || len(current.Rewards) != 2 {
		t.Fatalf("current = %+v", current)
	}

	// Cache-Control : TTL court + stale-while-revalidate posé par le handler.
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/playlist/current?game=fh6", nil))
	if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "max-age=300") || !strings.Contains(cc, "stale-while-revalidate=") {
		t.Errorf("Cache-Control = %q, want court + SWR", cc)
	}

	// Id inconnu → 404 RFC 9457.
	code, body = doGET(t, srv, "/v1/playlist/series/nope")
	if code != http.StatusNotFound {
		t.Fatalf("unknown status = %d\n%s", code, body)
	}
}
