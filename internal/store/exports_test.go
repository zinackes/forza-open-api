// Tests d'intégration du manifeste d'exports et des dump readers (Postgres jetable,
// db/init.sql). Valide l'upsert idempotent + le filtre du manifeste, et surtout que
// les requêtes to_jsonb produisent un dump correct contre un vrai Postgres : dump
// plat scopé au jeu (cars, JSONB imbriqué préservé) et dump imbriqué (playlist avec
// rewards/challenges agrégés).
package store_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/zinackes/forza-open-api/internal/store"
)

func exPtr(s string) *string { return &s }

func exFind(rows []store.Export, game, resource, format string) *store.Export {
	for i := range rows {
		if rows[i].Game == game && rows[i].Resource == resource && rows[i].Format == format {
			return &rows[i]
		}
	}
	return nil
}

func exHasCol(cols []string, name string) bool {
	for _, c := range cols {
		if c == name {
			return true
		}
	}
	return false
}

func TestExportsUpsertAndList(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	now := time.Date(2026, 6, 10, 5, 0, 0, 0, time.UTC)

	rows := []store.Export{
		{Game: "fh6", Resource: "cars", Format: "json", URL: "u1", SizeBytes: 10, ETag: "e1", GeneratedAt: now},
		{Game: "fh6", Resource: "cars", Format: "csv", URL: "u2", SizeBytes: 20, ETag: "e2", GeneratedAt: now},
		{Game: "fh5", Resource: "cars", Format: "json", URL: "u3", SizeBytes: 30, ETag: "e3", GeneratedAt: now},
	}
	for _, e := range rows {
		if err := st.UpsertExport(ctx, e); err != nil {
			t.Fatalf("upsert %v: %v", e, err)
		}
	}

	// Sans filtre : tout, ordonné game, resource, format (fh5 < fh6 ; csv < json).
	all, err := st.ListExports(ctx, nil)
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("list all = %d, want 3", len(all))
	}
	if all[0].Game != "fh5" || all[1].Format != "csv" || all[2].Format != "json" {
		t.Errorf("ordre inattendu: %+v", all)
	}

	// Filtre par jeu, puis jeu sans archive → liste vide.
	fh6, _ := st.ListExports(ctx, exPtr("fh6"))
	if len(fh6) != 2 {
		t.Errorf("list fh6 = %d, want 2", len(fh6))
	}
	none, _ := st.ListExports(ctx, exPtr("fh4"))
	if len(none) != 0 {
		t.Errorf("list fh4 = %d, want 0", len(none))
	}

	// Idempotence : ré-upsert même clé (game,resource,format) → mise à jour en place.
	updated := rows[0]
	updated.URL, updated.SizeBytes, updated.ETag = "u1b", 99, "e1b"
	if err := st.UpsertExport(ctx, updated); err != nil {
		t.Fatalf("re-upsert: %v", err)
	}
	all2, _ := st.ListExports(ctx, nil)
	if len(all2) != 3 {
		t.Fatalf("après re-upsert = %d, want 3 (pas de doublon)", len(all2))
	}
	got := exFind(all2, "fh6", "cars", "json")
	if got == nil || got.URL != "u1b" || got.SizeBytes != 99 || got.ETag != "e1b" {
		t.Errorf("upsert n'a pas mis à jour: %+v", got)
	}
}

func TestDumpResourceCarsFlat(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	mustExec(t, st, `INSERT INTO cars (id, game, name, make, class, pi, drivetrain, stats) VALUES
		('c1','fh6','Supra','Toyota','S1',850,'RWD','{"power":382}'),
		('c2','fh6','Civic','Honda','B',600,'FWD',NULL),
		('c3','fh5','Other','Make','A',800,'AWD',NULL)`)

	d, err := st.DumpResource(ctx, "cars", "fh6")
	if err != nil {
		t.Fatalf("dump cars: %v", err)
	}
	// Scopé au jeu : seules les 2 voitures fh6.
	if len(d.Rows) != 2 {
		t.Fatalf("rows = %d, want 2 (fh6 only)", len(d.Rows))
	}
	if !exHasCol(d.Columns, "id") || !exHasCol(d.Columns, "stats") {
		t.Errorf("colonnes incomplètes: %v", d.Columns)
	}

	// 1re ligne (ORDER BY id → c1) : game correct, JSONB stats préservé en objet
	// imbriqué (pas aplati en chaîne).
	var first map[string]any
	if err := json.Unmarshal(d.Rows[0], &first); err != nil {
		t.Fatalf("row JSON: %v", err)
	}
	if first["game"] != "fh6" {
		t.Errorf("game = %v, want fh6", first["game"])
	}
	stats, ok := first["stats"].(map[string]any)
	if !ok || stats["power"] == nil {
		t.Errorf("stats JSONB non imbriqué: %v", first["stats"])
	}
}

func TestDumpResourcePlaylistNested(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	mustExec(t, st, `INSERT INTO series (id, game, series, name, week, is_current) VALUES
		('s1','fh6',1,'Series One',1,true)`)
	mustExec(t, st, `INSERT INTO rewards (id, series_id, at_percent, type, item) VALUES
		('r1','s1',50,'car','Supra'),('r2','s1',80,'car','GTR')`)
	mustExec(t, st, `INSERT INTO challenges (id, series_id, scope, name, requirement, reward) VALUES
		('ch1','s1','weekly','Win a race','race','100pts')`)

	d, err := st.DumpResource(ctx, "playlist", "fh6")
	if err != nil {
		t.Fatalf("dump playlist: %v", err)
	}
	// Ressource imbriquée : pas de Columns (donc pas de CSV en amont).
	if d.Columns != nil {
		t.Errorf("playlist ne doit pas exposer de colonnes: %v", d.Columns)
	}
	if len(d.Rows) != 1 {
		t.Fatalf("rows = %d, want 1 série", len(d.Rows))
	}
	var s struct {
		ID         string            `json:"id"`
		Rewards    []json.RawMessage `json:"rewards"`
		Challenges []json.RawMessage `json:"challenges"`
	}
	if err := json.Unmarshal(d.Rows[0], &s); err != nil {
		t.Fatalf("série JSON: %v", err)
	}
	if s.ID != "s1" || len(s.Rewards) != 2 || len(s.Challenges) != 1 {
		t.Errorf("imbrication = id %s / %d rewards / %d challenges, want s1/2/1",
			s.ID, len(s.Rewards), len(s.Challenges))
	}
}
