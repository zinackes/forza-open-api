// Test du endpoint manifeste : GET /v1/exports. Sème des entrées exports (fh6) et
// vérifie le listing ordonné, le filtre game optionnel (y compris jeu sans archive
// → liste vide, pas null), et l'en-tête de cache moyen. Réutilise le Postgres
// jetable et le serveur ogen de meta_test.
package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type exportResp struct {
	Game        string    `json:"game"`
	Resource    string    `json:"resource"`
	Format      string    `json:"format"`
	URL         string    `json:"url"`
	SizeBytes   int64     `json:"sizeBytes"`
	Etag        string    `json:"etag"`
	GeneratedAt time.Time `json:"generatedAt"`
}

func seedExports(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	const q = `INSERT INTO exports (game, resource, format, url, size_bytes, etag, generated_at) VALUES
		('fh6','cars','json','https://data.example/fh6/cars.json',100,'e-cj',$1),
		('fh6','cars','jsonl','https://data.example/fh6/cars.jsonl',110,'e-cl',$1),
		('fh6','cars','csv','https://data.example/fh6/cars.csv',90,'e-cc',$1),
		('fh6','playlist','json','https://data.example/fh6/playlist.json',50,'e-pj',$1)`
	if _, err := pool.Exec(context.Background(), q, time.Date(2026, 6, 10, 5, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("seed exports: %v", err)
	}
}

func getExports(t *testing.T, srv http.Handler, query string) (*httptest.ResponseRecorder, []exportResp) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/exports"+query, nil)
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200\nbody: %s", rec.Code, rec.Body.String())
	}
	var got []exportResp
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode exports: %v\nbody: %s", err, rec.Body.String())
	}
	return rec, got
}

func TestListExports(t *testing.T) {
	pool := newMetaPool(t)
	seedExports(t, pool)
	srv := newMetaServer(t, pool, "")

	// Sans filtre : les 4 archives, ordonnées resource puis format (cars/csv d'abord).
	rec, all := getExports(t, srv, "")
	if cc := rec.Header().Get("Cache-Control"); cc != "public, max-age=300, stale-while-revalidate=600" {
		t.Errorf("Cache-Control = %q, want cache moyen", cc)
	}
	if len(all) != 4 {
		t.Fatalf("exports = %d, want 4", len(all))
	}
	if all[0].Resource != "cars" || all[0].Format != "csv" {
		t.Errorf("1re entrée = %s/%s, want cars/csv", all[0].Resource, all[0].Format)
	}
	if all[3].Resource != "playlist" {
		t.Errorf("dernière entrée = %s, want playlist", all[3].Resource)
	}
	// Champs propagés tels quels (url/etag/taille du manifeste).
	cj := findExport(all, "cars", "json")
	if cj == nil || cj.URL != "https://data.example/fh6/cars.json" || cj.Etag != "e-cj" || cj.SizeBytes != 100 {
		t.Errorf("entrée cars/json incorrecte: %+v", cj)
	}

	// Filtre game = fh6 → idem (toutes les archives sont fh6).
	_, fh6 := getExports(t, srv, "?game=fh6")
	if len(fh6) != 4 {
		t.Errorf("exports?game=fh6 = %d, want 4", len(fh6))
	}

	// Filtre sur un jeu sans archive → liste vide (et non null).
	_, fh5 := getExports(t, srv, "?game=fh5")
	if len(fh5) != 0 {
		t.Errorf("exports?game=fh5 = %d, want 0", len(fh5))
	}
}

func findExport(rows []exportResp, resource, format string) *exportResp {
	for i := range rows {
		if rows[i].Resource == resource && rows[i].Format == format {
			return &rows[i]
		}
	}
	return nil
}
