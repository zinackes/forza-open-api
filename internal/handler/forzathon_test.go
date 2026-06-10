// Test bout-en-bout des endpoints Forzathon Shop en lecture (/v1/forzathon-shop,
// /v1/forzathon-shop/history) à travers le serveur ogen, sur un Postgres jetable
// (db/init.sql). Vérifie le mapping DB → contrat : rotation courante (semaine la
// plus récente), champs optionnels (fpCost/weekEnd/carId), Cache-Control, et
// historique paginé (rotations récentes d'abord). Nécessite Docker.
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

func newSeededForzathonServer(t *testing.T) http.Handler {
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

	if _, err := pool.Exec(ctx, `INSERT INTO cars (id, game, name, make, class, pi, drivetrain)
		VALUES ('fh6-mazda-furai','fh6','Mazda Furai','Mazda','X',999,'RWD')`); err != nil {
		t.Fatalf("seed car: %v", err)
	}

	w1 := time.Date(2026, 5, 28, 14, 30, 0, 0, time.UTC)
	w2 := time.Date(2026, 6, 4, 14, 30, 0, 0, time.UTC)
	end1, end2 := w1.AddDate(0, 0, 7), w2.AddDate(0, 0, 7)
	now := time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC)
	c750, c75, c150 := 750, 75, 150
	src, carID := "wiki", "fh6-mazda-furai"

	week1 := []store.ForzathonShopItem{
		{ID: "fh6-20260528-old-horn", Game: "fh6", WeekStart: w1, WeekEnd: &end1, Kind: "horn", Name: "Old Horn", FpCost: &c75, Source: &src, LastVerified: &now},
	}
	week2 := []store.ForzathonShopItem{
		{ID: "fh6-20260604-mazda-furai", Game: "fh6", WeekStart: w2, WeekEnd: &end2, Kind: "car", CarID: &carID, Name: "Mazda Furai", FpCost: &c750, Source: &src, LastVerified: &now},
		{ID: "fh6-20260604-festive-horn", Game: "fh6", WeekStart: w2, WeekEnd: &end2, Kind: "horn", Name: "Festive Horn", FpCost: &c75, Source: &src, LastVerified: &now},
		{ID: "fh6-20260604-aviator", Game: "fh6", WeekStart: w2, WeekEnd: &end2, Kind: "clothing", Name: "Aviator Outfit", FpCost: &c150, Source: &src, LastVerified: &now},
	}
	if err := st.UpsertForzathonRotation(ctx, week1); err != nil {
		t.Fatalf("seed week1: %v", err)
	}
	if err := st.UpsertForzathonRotation(ctx, week2); err != nil {
		t.Fatalf("seed week2: %v", err)
	}

	srv, err := oas.NewServer(handler.New(st, ""), handler.SecurityHandler{},
		oas.WithErrorHandler(handler.ProblemErrorHandler))
	if err != nil {
		t.Fatalf("oas.NewServer: %v", err)
	}
	return srv
}

type shopItemJSON struct {
	ID        string `json:"id"`
	Game      string `json:"game"`
	WeekStart string `json:"weekStart"`
	WeekEnd   string `json:"weekEnd"`
	Kind      string `json:"kind"`
	CarID     string `json:"carId"`
	Name      string `json:"name"`
	FpCost    *int   `json:"fpCost"`
}

type shopListJSON struct {
	Items    []shopItemJSON `json:"items"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
}

func TestForzathonShopEndpoints(t *testing.T) {
	srv := newSeededForzathonServer(t)

	// Rotation courante = semaine la plus récente (w2), 3 objets, tri fp_cost croissant.
	code, body := doGET(t, srv, "/v1/forzathon-shop?game=fh6")
	if code != http.StatusOK {
		t.Fatalf("current status = %d\n%s", code, body)
	}
	var cur []shopItemJSON
	if err := json.Unmarshal(body, &cur); err != nil {
		t.Fatalf("decode current: %v\n%s", err, body)
	}
	if len(cur) != 3 {
		t.Fatalf("courante = %d objets, want 3 (w2)", len(cur))
	}
	if !strings.HasPrefix(cur[0].WeekStart, "2026-06-04") {
		t.Errorf("courante weekStart = %q, want 2026-06-04 (w2)", cur[0].WeekStart)
	}
	// Tri fp_cost croissant : Festive Horn (75) avant Aviator (150) avant Furai (750).
	if cur[0].Name != "Festive Horn" || cur[2].Name != "Mazda Furai" {
		t.Errorf("ordre courante = [%s ... %s], want Festive Horn ... Mazda Furai", cur[0].Name, cur[2].Name)
	}
	// Objet kind=car : carId + weekEnd + fpCost présents.
	var furai *shopItemJSON
	for i := range cur {
		if cur[i].Kind == "car" {
			furai = &cur[i]
		}
	}
	if furai == nil || furai.CarID != "fh6-mazda-furai" {
		t.Fatalf("objet car = %+v, want carId fh6-mazda-furai", furai)
	}
	if furai.FpCost == nil || *furai.FpCost != 750 || !strings.HasPrefix(furai.WeekEnd, "2026-06-11") {
		t.Errorf("furai = %+v, want fpCost 750 + weekEnd 2026-06-11", furai)
	}

	// Cache-Control court + SWR posé par le handler.
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/forzathon-shop?game=fh6", nil))
	if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "max-age=300") || !strings.Contains(cc, "stale-while-revalidate=") {
		t.Errorf("Cache-Control = %q, want court + SWR", cc)
	}

	// Jeu sans données → tableau vide (200), pas 404.
	code, body = doGET(t, srv, "/v1/forzathon-shop?game=fh5")
	if code != http.StatusOK {
		t.Fatalf("current fh5 status = %d\n%s", code, body)
	}
	var empty []shopItemJSON
	if err := json.Unmarshal(body, &empty); err != nil || len(empty) != 0 {
		t.Errorf("current fh5 = %v (err=%v), want []", empty, err)
	}

	// Historique paginé : total 4, page 1 (size 2) = 2 objets de w2 (récentes d'abord).
	code, body = doGET(t, srv, "/v1/forzathon-shop/history?game=fh6&page_size=2")
	if code != http.StatusOK {
		t.Fatalf("history status = %d\n%s", code, body)
	}
	var list shopListJSON
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("decode history: %v\n%s", err, body)
	}
	if list.Total != 4 || list.PageSize != 2 || len(list.Items) != 2 {
		t.Fatalf("history = total %d size %d items %d, want total 4 size 2 items 2", list.Total, list.PageSize, len(list.Items))
	}
	for _, it := range list.Items {
		if !strings.HasPrefix(it.WeekStart, "2026-06-04") {
			t.Errorf("history p1 contient une semaine != w2 (tri DESC cassé): %s", it.WeekStart)
		}
	}

	// Page 2 atteint la semaine la plus ancienne (w1).
	code, body = doGET(t, srv, "/v1/forzathon-shop/history?game=fh6&page=2&page_size=2")
	if code != http.StatusOK {
		t.Fatalf("history p2 status = %d\n%s", code, body)
	}
	var p2 shopListJSON
	if err := json.Unmarshal(body, &p2); err != nil {
		t.Fatalf("decode history p2: %v", err)
	}
	if len(p2.Items) != 2 {
		t.Fatalf("history p2 = %d objets, want 2", len(p2.Items))
	}
	if !strings.HasPrefix(p2.Items[1].WeekStart, "2026-05-28") {
		t.Errorf("dernier objet = %q, want w1 2026-05-28", p2.Items[1].WeekStart)
	}
}
