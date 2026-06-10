// Tests d'intégration de l'upsert Festival Playlist (UpsertSeries) sur Postgres
// jetable (testcontainers). Vérifie : la série courante + ses rewards/challenges
// atterrissent en DB, l'idempotence (re-run sans doublon, enfants remplacés),
// l'invariant « une seule série courante par jeu », et la fraîcheur via Meta
// (data_changes resource « series »). Nécessite Docker.
package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/zinackes/forza-open-api/internal/store"
)

func TestUpsertSeriesLandsAndIsIdempotent(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	season, saga := "Autumn", "Welcome to Japan"
	start := time.Date(2026, 5, 28, 14, 30, 0, 0, time.UTC)
	end := time.Date(2026, 6, 4, 14, 30, 0, 0, time.UTC)
	p33, p67 := 33, 67
	req, rw := "Win a Cross Country Race", "1 point"

	ser := store.Series{
		ID: "fh6-s01w2", Game: "fh6", Series: 1, Name: &saga, Season: &season,
		Week: 2, StartsAt: start, EndsAt: end, IsCurrent: true,
	}
	rewards := []store.Reward{
		{ID: "fh6-s01w2-rw01", SeriesID: ser.ID, AtPercent: &p33, Type: "series", Item: "2008 Mazda Furai"},
		{ID: "fh6-s01w2-rw02", SeriesID: ser.ID, AtPercent: &p67, Type: "series", Item: "2010 Nissan 370Z"},
	}
	challenges := []store.Challenge{
		{ID: "fh6-s01w2-ch01", SeriesID: ser.ID, Scope: "weekly", Name: "King of the Road", Requirement: &req, Reward: &rw, ExpiresAt: &end},
		{ID: "fh6-s01w2-ch02", SeriesID: ser.ID, Scope: "daily", Name: "Big in Japan", Requirement: &req, Reward: &rw},
	}

	// 1) La série courante atterrit (nouvelle).
	added, err := st.UpsertSeries(ctx, ser, rewards, challenges)
	if err != nil {
		t.Fatalf("UpsertSeries: %v", err)
	}
	if !added {
		t.Error("added = false, want true (série nouvelle)")
	}
	if got := queryInt(t, st, `SELECT count(*) FROM series WHERE game='fh6'`); got != 1 {
		t.Errorf("series count = %d, want 1", got)
	}
	if got := queryBool(t, st, `SELECT is_current FROM series WHERE id='fh6-s01w2'`); !got {
		t.Error("is_current = false, want true")
	}
	if got := queryInt(t, st, `SELECT count(*) FROM rewards WHERE series_id='fh6-s01w2'`); got != 2 {
		t.Errorf("rewards count = %d, want 2", got)
	}
	if got := queryInt(t, st, `SELECT count(*) FROM challenges WHERE series_id='fh6-s01w2'`); got != 2 {
		t.Errorf("challenges count = %d, want 2", got)
	}
	// Le changement est journalisé (resource « series ») → surfacé par GET
	// /v1/changes et la fraîcheur playlist de GET /v1/meta.
	if got := queryInt(t, st, `SELECT count(*) FROM data_changes WHERE game='fh6' AND resource='series'`); got != 1 {
		t.Errorf("data_changes(series) = %d, want 1 (added)", got)
	}

	// 2) Re-run idempotent : pas de doublon, enfants remplacés (pas accumulés).
	added, err = st.UpsertSeries(ctx, ser, rewards, challenges)
	if err != nil {
		t.Fatalf("UpsertSeries re-run: %v", err)
	}
	if added {
		t.Error("added = true au re-run, want false (série existante)")
	}
	if got := queryInt(t, st, `SELECT count(*) FROM rewards WHERE series_id='fh6-s01w2'`); got != 2 {
		t.Errorf("rewards après re-run = %d, want 2 (pas de doublon)", got)
	}
	if got := queryInt(t, st, `SELECT count(*) FROM challenges WHERE series_id='fh6-s01w2'`); got != 2 {
		t.Errorf("challenges après re-run = %d, want 2", got)
	}

	// 3) Source amaigrie : les enfants retirés disparaissent (remplacement en bloc).
	if _, err := st.UpsertSeries(ctx, ser, rewards[:1], challenges[:1]); err != nil {
		t.Fatalf("UpsertSeries amaigri: %v", err)
	}
	if got := queryInt(t, st, `SELECT count(*) FROM rewards WHERE series_id='fh6-s01w2'`); got != 1 {
		t.Errorf("rewards après amaigrissement = %d, want 1", got)
	}
	if got := queryInt(t, st, `SELECT count(*) FROM challenges WHERE series_id='fh6-s01w2'`); got != 1 {
		t.Errorf("challenges après amaigrissement = %d, want 1", got)
	}

	// 4) Une nouvelle série courante éteint l'ancienne (invariant 1 courante/jeu).
	ser2 := ser
	ser2.ID, ser2.Week, ser2.IsCurrent = "fh6-s01w3", 3, true
	if _, err := st.UpsertSeries(ctx, ser2, nil, nil); err != nil {
		t.Fatalf("UpsertSeries série suivante: %v", err)
	}
	if got := queryInt(t, st, `SELECT count(*) FROM series WHERE game='fh6' AND is_current`); got != 1 {
		t.Errorf("séries courantes = %d, want 1", got)
	}
	if got := queryString(t, st, `SELECT id FROM series WHERE game='fh6' AND is_current`); got != "fh6-s01w3" {
		t.Errorf("série courante = %q, want fh6-s01w3", got)
	}
}

// TestSeriesReads couvre les lectures servant /v1/playlist/series, /series/{id}
// et /current : liste (récentes d'abord, sans enfants), détail hydraté, série
// courante, filtrage par jeu, et inconnu → nil.
func TestSeriesReads(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	saga, season1, season2 := "Welcome to Japan", "Summer", "Autumn"
	s1 := time.Date(2026, 5, 21, 14, 30, 0, 0, time.UTC)
	e1 := time.Date(2026, 5, 28, 14, 30, 0, 0, time.UTC)
	s2 := time.Date(2026, 5, 28, 14, 30, 0, 0, time.UTC)
	e2 := time.Date(2026, 6, 4, 14, 30, 0, 0, time.UTC)
	p33 := 33
	req, rw := "Win a Race", "1 point"

	past := store.Series{ID: "fh6-s01w1", Game: "fh6", Series: 1, Name: &saga, Season: &season1, Week: 1, StartsAt: s1, EndsAt: e1, IsCurrent: false}
	cur := store.Series{ID: "fh6-s01w2", Game: "fh6", Series: 1, Name: &saga, Season: &season2, Week: 2, StartsAt: s2, EndsAt: e2, IsCurrent: true}
	other := store.Series{ID: "fh5-s45w1", Game: "fh5", Series: 45, Week: 1, StartsAt: s1, EndsAt: e1, IsCurrent: false}

	if _, err := st.UpsertSeries(ctx, past, []store.Reward{{ID: "fh6-s01w1-rw01", SeriesID: past.ID, AtPercent: &p33, Type: "season", Item: "Car A"}}, nil); err != nil {
		t.Fatalf("upsert past: %v", err)
	}
	if _, err := st.UpsertSeries(ctx, cur,
		[]store.Reward{
			{ID: "fh6-s01w2-rw01", SeriesID: cur.ID, AtPercent: &p33, Type: "series", Item: "Mazda Furai"},
			{ID: "fh6-s01w2-rw02", SeriesID: cur.ID, Type: "event", Item: "LINK Phrase"}, // at_percent NULL
		},
		[]store.Challenge{
			{ID: "fh6-s01w2-ch01", SeriesID: cur.ID, Scope: "weekly", Name: "King of the Road", Requirement: &req, Reward: &rw, ExpiresAt: &e2},
			{ID: "fh6-s01w2-ch02", SeriesID: cur.ID, Scope: "daily", Name: "Big in Japan", Requirement: &req},
		}); err != nil {
		t.Fatalf("upsert cur: %v", err)
	}
	if _, err := st.UpsertSeries(ctx, other, nil, nil); err != nil {
		t.Fatalf("upsert other: %v", err)
	}

	// Liste fh6 : récentes d'abord (w2 puis w1), sans enfants.
	list, err := st.ListSeries(ctx, "fh6")
	if err != nil {
		t.Fatalf("ListSeries: %v", err)
	}
	if len(list) != 2 || list[0].ID != "fh6-s01w2" || list[1].ID != "fh6-s01w1" {
		t.Fatalf("list order = %+v", list)
	}

	// Détail hydraté.
	got, rewards, challenges, err := st.SeriesDetail(ctx, "fh6-s01w2")
	if err != nil || got == nil {
		t.Fatalf("SeriesDetail: got=%v err=%v", got, err)
	}
	if len(rewards) != 2 || rewards[0].ID != "fh6-s01w2-rw01" || rewards[1].AtPercent != nil {
		t.Fatalf("rewards = %+v", rewards)
	}
	if len(challenges) != 2 || challenges[0].Scope != "weekly" || challenges[1].ExpiresAt != nil {
		t.Fatalf("challenges = %+v", challenges)
	}

	// Inconnu → nil sans erreur.
	if s, _, _, err := st.SeriesDetail(ctx, "nope"); err != nil || s != nil {
		t.Fatalf("SeriesDetail(unknown) = %v, %v", s, err)
	}

	// Courante fh6 = w2 ; fh5 n'en a pas → nil.
	cs, csr, _, err := st.CurrentSeries(ctx, "fh6")
	if err != nil || cs == nil || cs.ID != "fh6-s01w2" || len(csr) != 2 {
		t.Fatalf("CurrentSeries(fh6) = %v rewards=%d err=%v", cs, len(csr), err)
	}
	if cs2, _, _, err := st.CurrentSeries(ctx, "fh5"); err != nil || cs2 != nil {
		t.Fatalf("CurrentSeries(fh5) = %v, %v", cs2, err)
	}
}

func queryInt(t *testing.T, st *store.Store, sql string) int64 {
	t.Helper()
	var n int64
	if err := st.DB.QueryRow(context.Background(), sql).Scan(&n); err != nil {
		t.Fatalf("query int: %v\nSQL: %s", err, sql)
	}
	return n
}

func queryBool(t *testing.T, st *store.Store, sql string) bool {
	t.Helper()
	var b bool
	if err := st.DB.QueryRow(context.Background(), sql).Scan(&b); err != nil {
		t.Fatalf("query bool: %v\nSQL: %s", err, sql)
	}
	return b
}

func queryString(t *testing.T, st *store.Store, sql string) string {
	t.Helper()
	var s string
	if err := st.DB.QueryRow(context.Background(), sql).Scan(&s); err != nil {
		t.Fatalf("query string: %v\nSQL: %s", err, sql)
	}
	return s
}
