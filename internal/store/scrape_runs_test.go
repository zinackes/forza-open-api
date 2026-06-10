package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/zinackes/forza-open-api/internal/store"
)

// TestScrapeRunsLatestPerSource : LatestScrapeRuns ne renvoie que le dernier run
// de chaque (source, jeu), avec round-trip des violations/erreur et game NULL pour
// les sources sans jeu (vue dashboard `seed health`).
func TestScrapeRunsLatestPerSource(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	base := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	runs := []store.ScrapeRun{
		{Source: "cars", Game: "fh6", Status: "ok", Records: 600, StartedAt: base, FinishedAt: base},
		{Source: "cars", Game: "fh6", Status: "anomaly", Records: 12, // plus récent → gagne
			Violations: []byte(`[{"rule":"count_floor","detail":"12 < 150","severity":"anomaly"}]`),
			StartedAt:  base.Add(time.Hour), FinishedAt: base.Add(time.Hour)},
		{Source: "tracks", Status: "ok", Records: 340, StartedAt: base, FinishedAt: base}, // game NULL
		{Source: "playlist", Game: "fh6", Status: "failed", Error: "boom", StartedAt: base, FinishedAt: base},
	}
	for _, r := range runs {
		if err := st.InsertScrapeRun(ctx, r); err != nil {
			t.Fatalf("InsertScrapeRun: %v", err)
		}
	}

	got, err := st.LatestScrapeRuns(ctx)
	if err != nil {
		t.Fatalf("LatestScrapeRuns: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("groupes = %d, want 3 (cars/fh6, playlist/fh6, tracks/-)", len(got))
	}

	by := map[string]store.ScrapeRunSummary{}
	for _, r := range got {
		by[r.Source] = r
	}

	if c := by["cars"]; c.Status != "anomaly" || c.Records != 12 || c.Game == nil || *c.Game != "fh6" || len(c.Violations) == 0 {
		t.Errorf("cars latest = %+v, want le run le plus récent (anomaly, records=12, fh6, violations non vides)", c)
	}
	if tr := by["tracks"]; tr.Game != nil {
		t.Errorf("tracks game = %v, want NULL (source sans jeu)", tr.Game)
	}
	if pl := by["playlist"]; pl.Status != "failed" || pl.Error == nil || *pl.Error != "boom" {
		t.Errorf("playlist latest = %+v, want failed + error=boom", pl)
	}
}
