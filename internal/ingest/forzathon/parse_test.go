// Tests du parsing Forzathon Shop sur fixture figée (pas de réseau, cf.
// testing.md) : dérivation de la fenêtre de rotation, extraction + classification
// des objets de la table wikitext, normalisation idempotente, et invariants de
// santé (CheckRun) sur un run sain comme sur une rupture de structure.
package forzathon

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/zinackes/forza-open-api/internal/health"
)

func TestWeekWindow(t *testing.T) {
	cases := []struct {
		name        string
		now         time.Time
		wantStart   time.Time
		description string
	}{
		{
			name:      "milieu de semaine",
			now:       time.Date(2026, 6, 7, 10, 0, 0, 0, time.UTC), // dimanche
			wantStart: time.Date(2026, 6, 4, 14, 30, 0, 0, time.UTC),
		},
		{
			name:      "jeudi juste après le reset",
			now:       time.Date(2026, 6, 4, 15, 0, 0, 0, time.UTC),
			wantStart: time.Date(2026, 6, 4, 14, 30, 0, 0, time.UTC),
		},
		{
			name:      "jeudi avant le reset → rotation de la semaine précédente",
			now:       time.Date(2026, 6, 4, 9, 0, 0, 0, time.UTC),
			wantStart: time.Date(2026, 5, 28, 14, 30, 0, 0, time.UTC),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			start, end := weekWindow(c.now)
			if !start.Equal(c.wantStart) {
				t.Errorf("start = %s, want %s", start.Format(time.RFC3339), c.wantStart.Format(time.RFC3339))
			}
			if !end.Equal(c.wantStart.AddDate(0, 0, 7)) {
				t.Errorf("end = %s, want start+7j", end.Format(time.RFC3339))
			}
		})
	}
}

func TestParseShopTable(t *testing.T) {
	raw := readFixture(t, "shop.wikitext")
	items := parseShopTable(raw)

	if len(items) != 5 {
		t.Fatalf("objets parsés = %d, want 5\n%+v", len(items), items)
	}

	// Lien wikitext déréférencé au libellé, kind classé, coût parsé.
	if items[0].Name != "Mazda Furai Concept" {
		t.Errorf("name[0] = %q, want Mazda Furai Concept (lien déréférencé)", items[0].Name)
	}
	if items[0].Kind != "car" {
		t.Errorf("kind[0] = %q, want car", items[0].Kind)
	}
	if items[0].FpCost == nil || *items[0].FpCost != 750 {
		t.Errorf("fpCost[0] = %v, want 750", items[0].FpCost)
	}
	if items[0].Description == nil || *items[0].Description != "Rare festival exclusive" {
		t.Errorf("desc[0] = %v, want note", items[0].Description)
	}

	wantKinds := []string{"car", "horn", "clothing", "forza_link_phrase", "other"}
	for i, want := range wantKinds {
		if items[i].Kind != want {
			t.Errorf("kind[%d] = %q, want %q", i, items[i].Kind, want)
		}
	}

	// Coût avec séparateur de milliers.
	if items[4].FpCost == nil || *items[4].FpCost != 1000 {
		t.Errorf("fpCost[4] = %v, want 1000 (séparateur retiré)", items[4].FpCost)
	}
	// Coût absent → nil (jamais inventé) : le klaxon n'a pas de note.
	if items[1].Description != nil {
		t.Errorf("desc[1] = %v, want nil (cellule vide)", items[1].Description)
	}
}

func TestNormalizeIdempotentIDs(t *testing.T) {
	raw := readFixture(t, "shop.wikitext")
	parsed := parseShopTable(raw)
	start := time.Date(2026, 6, 4, 14, 30, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 7)
	now := time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC)

	items := normalize("fh6", start, end, parsed, now)
	if len(items) != 5 {
		t.Fatalf("normalisés = %d, want 5", len(items))
	}
	if items[0].ID != "fh6-20260604-mazda-furai-concept" {
		t.Errorf("id[0] = %q, want fh6-20260604-mazda-furai-concept", items[0].ID)
	}
	for _, it := range items {
		if it.Game != "fh6" || !it.WeekStart.Equal(start) || it.WeekEnd == nil || !it.WeekEnd.Equal(end) {
			t.Errorf("item daté incorrectement: %+v", it)
		}
		if it.Source == nil || *it.Source != "wiki" {
			t.Errorf("source = %v, want wiki", it.Source)
		}
		if it.LastVerified == nil || !it.LastVerified.Equal(now) {
			t.Errorf("lastVerified = %v, want %s", it.LastVerified, now)
		}
	}
}

func TestCheckRun(t *testing.T) {
	now := time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC)
	start := time.Date(2026, 6, 4, 14, 30, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 7)

	// Run sain : aucune violation.
	if vs := CheckRun(Result{WeekStart: start, WeekEnd: end, Items: 5, ItemsWithCost: 5}, now); len(vs) != 0 {
		t.Errorf("run sain → violations = %+v, want aucune", vs)
	}

	// 0 objet (page absente / table non parsée) → anomalie.
	vs := CheckRun(Result{WeekStart: start, WeekEnd: end, Items: 0}, now)
	if !hasRule(vs, "no_items", health.SevAnomaly) {
		t.Errorf("0 objet → want anomalie no_items, got %+v", vs)
	}

	// Aucun coût parsé alors qu'il y a des objets → avertissement.
	vs = CheckRun(Result{WeekStart: start, WeekEnd: end, Items: 3, ItemsWithCost: 0}, now)
	if !hasRule(vs, "no_fp_cost", health.SevWarning) {
		t.Errorf("0 coût → want warning no_fp_cost, got %+v", vs)
	}

	// Fenêtre périmée → anomalie.
	vs = CheckRun(Result{WeekStart: start, WeekEnd: end, Items: 5, ItemsWithCost: 5},
		end.AddDate(0, 0, 1))
	if !hasRule(vs, "stale_rotation", health.SevAnomaly) {
		t.Errorf("fenêtre périmée → want anomalie stale_rotation, got %+v", vs)
	}
}

func hasRule(vs []health.Violation, rule string, sev health.Severity) bool {
	for _, v := range vs {
		if v.Rule == rule && v.Severity == sev {
			return true
		}
	}
	return false
}

func readFixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(b)
}
