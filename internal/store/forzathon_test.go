// Tests d'intégration du store Forzathon Shop (Postgres jetable). Vérifie l'upsert
// idempotent par (game, week_start, name), la journalisation data_changes
// (added puis updated), la rotation courante (semaine la plus récente), l'historique
// paginé (plus récentes d'abord) et le lien car_id (FK nullable). Nécessite Docker.
package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/zinackes/forza-open-api/internal/store"
)

func fpi(id, game string, week time.Time, kind, name string, cost *int, carID *string) store.ForzathonShopItem {
	end := week.AddDate(0, 0, 7)
	now := time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC)
	src := "wiki"
	return store.ForzathonShopItem{
		ID: id, Game: game, WeekStart: week, WeekEnd: &end, Kind: kind,
		CarID: carID, Name: name, FpCost: cost, Source: &src, LastVerified: &now,
	}
}

func TestForzathonUpsertReadsAndIdempotence(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	// Voiture du catalogue pour le lien car_id (FK).
	mustExec(t, st, `INSERT INTO cars (id, game, name, make, class, pi, drivetrain)
		VALUES ('fh6-mazda-furai','fh6','Mazda Furai','Mazda','X',999,'RWD')`)

	w1 := time.Date(2026, 5, 28, 14, 30, 0, 0, time.UTC)
	w2 := time.Date(2026, 6, 4, 14, 30, 0, 0, time.UTC)
	c750, c75, c150 := 750, 75, 150
	carID := "fh6-mazda-furai"

	week1 := []store.ForzathonShopItem{
		fpi("fh6-20260528-old-horn", "fh6", w1, "horn", "Old Horn", &c75, nil),
		fpi("fh6-20260528-old-suit", "fh6", w1, "clothing", "Old Suit", &c150, nil),
	}
	week2 := []store.ForzathonShopItem{
		fpi("fh6-20260604-mazda-furai", "fh6", w2, "car", "Mazda Furai", &c750, &carID),
		fpi("fh6-20260604-festive-horn", "fh6", w2, "horn", "Festive Horn", &c75, nil),
		fpi("fh6-20260604-aviator", "fh6", w2, "clothing", "Aviator Outfit", &c150, nil),
	}

	if err := st.UpsertForzathonRotation(ctx, week1); err != nil {
		t.Fatalf("upsert week1: %v", err)
	}
	if err := st.UpsertForzathonRotation(ctx, week2); err != nil {
		t.Fatalf("upsert week2: %v", err)
	}
	// Re-run identique de la semaine 2 : rejouable sans doublon.
	if err := st.UpsertForzathonRotation(ctx, week2); err != nil {
		t.Fatalf("re-run week2: %v", err)
	}

	if got := queryInt(t, st, `SELECT count(*) FROM forzathon_shop_items WHERE game='fh6'`); got != 5 {
		t.Errorf("items fh6 = %d, want 5 (pas de doublon)", got)
	}

	// data_changes : w1 added, w2 added, re-run w2 updated → 3 entrées forzathon_shop.
	if got := queryInt(t, st, `SELECT count(*) FROM data_changes WHERE resource='forzathon_shop'`); got != 3 {
		t.Errorf("data_changes forzathon = %d, want 3 (added,added,updated)", got)
	}
	if got := queryInt(t, st, `SELECT count(*) FROM data_changes WHERE resource='forzathon_shop' AND action='added'`); got != 2 {
		t.Errorf("data_changes added = %d, want 2", got)
	}
	if got := queryInt(t, st, `SELECT count(*) FROM data_changes WHERE resource='forzathon_shop' AND action='updated'`); got != 1 {
		t.Errorf("data_changes updated = %d, want 1 (re-run)", got)
	}

	// Rotation courante = semaine la plus récente (w2), 3 objets.
	cur, err := st.CurrentForzathonShop(ctx, "fh6")
	if err != nil {
		t.Fatalf("current: %v", err)
	}
	if len(cur) != 3 {
		t.Fatalf("courante = %d objets, want 3 (w2)", len(cur))
	}
	for _, it := range cur {
		if !it.WeekStart.Equal(w2) {
			t.Errorf("objet courant hors w2: %+v", it)
		}
	}
	// Lien car_id préservé (FK), trié fp_cost croissant → Festive Horn (75) d'abord.
	var furai *store.ForzathonShopItem
	for i := range cur {
		if cur[i].Name == "Mazda Furai" {
			furai = &cur[i]
		}
	}
	if furai == nil || furai.CarID == nil || *furai.CarID != carID {
		t.Errorf("car_id Mazda Furai = %v, want %q", furai, carID)
	}

	// Historique paginé, plus récentes d'abord : page 1 (size 2) = 2 objets de w2.
	hist, total, err := st.ForzathonShopHistory(ctx, "fh6", 2, 0)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if total != 5 {
		t.Errorf("history total = %d, want 5", total)
	}
	if len(hist) != 2 {
		t.Fatalf("history page = %d objets, want 2", len(hist))
	}
	for _, it := range hist {
		if !it.WeekStart.Equal(w2) {
			t.Errorf("history p1 contient une semaine != w2 (tri DESC cassé): %+v", it)
		}
	}
	// Dernière page atteint la semaine la plus ancienne (w1).
	last, _, err := st.ForzathonShopHistory(ctx, "fh6", 2, 4)
	if err != nil {
		t.Fatalf("history last page: %v", err)
	}
	if len(last) != 1 || !last[0].WeekStart.Equal(w1) {
		t.Errorf("dernière page = %+v, want 1 objet de w1", last)
	}

	// FK ON DELETE SET NULL : supprimer la voiture délie l'objet sans le supprimer.
	mustExec(t, st, `DELETE FROM cars WHERE id='fh6-mazda-furai'`)
	if got := queryInt(t, st, `SELECT count(*) FROM forzathon_shop_items WHERE name='Mazda Furai' AND car_id IS NULL`); got != 1 {
		t.Errorf("après suppression voiture, car_id devrait être NULL (objet conservé)")
	}
}
