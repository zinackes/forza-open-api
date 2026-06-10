// Tests d'intégration des opérations ajoutées par l'expansion du contrat :
// recherche globale, vue obtain, journal des changements, filtres car_id /
// updated_since, gets par id, facettes geo de la référence, stories/tours.
// Même harnais que integration_test.go (Postgres jetable, db/init.sql).
//
// Nécessite Docker (cf. .claude/rules/testing.md). En CI, Docker est présent.
package store_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/zinackes/forza-open-api/internal/store"
)

// TestUpsertCarsRecordsChanges vérifie le journal data_changes alimenté par
// UpsertCars : added à l'insert, rien sur un re-run identique (no-op grâce au
// garde IS DISTINCT FROM), updated quand la donnée change réellement.
func TestUpsertCarsRecordsChanges(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	cars := []store.Car{
		{ID: "c1", Game: "fh6", Name: "Civic", Make: "Honda", Class: "A", PI: 780, Drivetrain: "FWD"},
		{ID: "c2", Game: "fh6", Name: "GT", Make: "Ford", Class: "S2", PI: 920, Drivetrain: "RWD"},
	}
	if err := st.UpsertCars(ctx, cars); err != nil {
		t.Fatalf("UpsertCars initial: %v", err)
	}

	countByAction := func() map[string]int {
		t.Helper()
		rows, total, err := st.ListChanges(ctx, store.ChangeFilter{Game: "fh6", Limit: 50})
		if err != nil {
			t.Fatalf("ListChanges: %v", err)
		}
		if int64(len(rows)) != total {
			t.Fatalf("len(rows) = %d, total = %d", len(rows), total)
		}
		out := map[string]int{}
		for _, c := range rows {
			if c.Resource != "car" {
				t.Fatalf("resource = %q, want car", c.Resource)
			}
			out[c.Action]++
		}
		return out
	}

	if got := countByAction(); got["added"] != 2 || got["updated"] != 0 {
		t.Fatalf("après insert: %v, want 2 added / 0 updated", got)
	}

	// Re-run strictement identique : aucun nouveau changement (idempotence).
	if err := st.UpsertCars(ctx, cars); err != nil {
		t.Fatalf("UpsertCars re-run: %v", err)
	}
	if got := countByAction(); got["added"] != 2 || got["updated"] != 0 {
		t.Fatalf("après re-run identique: %v, want 2 added / 0 updated", got)
	}

	// Changement réel (PI rebalance) → un updated.
	cars[0].PI = 790
	if err := st.UpsertCars(ctx, cars); err != nil {
		t.Fatalf("UpsertCars modifié: %v", err)
	}
	if got := countByAction(); got["added"] != 2 || got["updated"] != 1 {
		t.Fatalf("après modification: %v, want 2 added / 1 updated", got)
	}

	// Filtre action.
	upd, _, err := st.ListChanges(ctx, store.ChangeFilter{Game: "fh6", Action: strptr("updated"), Limit: 50})
	if err != nil {
		t.Fatalf("ListChanges action=updated: %v", err)
	}
	if len(upd) != 1 || upd[0].ResourceID != "c1" {
		t.Fatalf("changes updated = %+v, want c1 seul", upd)
	}
	if upd[0].Summary == nil || *upd[0].Summary != "Civic" {
		t.Fatalf("summary = %v, want Civic", upd[0].Summary)
	}

	// Filtre since : un instant futur ne renvoie rien.
	future := time.Now().Add(time.Hour)
	none, total, err := st.ListChanges(ctx, store.ChangeFilter{Game: "fh6", Since: &future, Limit: 50})
	if err != nil {
		t.Fatalf("ListChanges since futur: %v", err)
	}
	if len(none) != 0 || total != 0 {
		t.Fatalf("since futur: %d rows / total %d, want 0/0", len(none), total)
	}
}

// TestListCarsUpdatedSince vérifie la sync incrémentale : seules les voitures
// modifiées après l'instant donné remontent.
func TestListCarsUpdatedSince(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	mustExec(t, st, `INSERT INTO cars (id, game, name, make, class, pi, drivetrain, updated_at) VALUES
		('old','fh6','Old','Make','D',500,'RWD','2026-01-01T00:00:00Z'),
		('new','fh6','New','Make','A',800,'RWD','2026-06-01T00:00:00Z')`)

	since := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	cars, total, err := st.ListCars(ctx, store.CarFilter{Game: "fh6", UpdatedSince: &since, Limit: 50})
	if err != nil {
		t.Fatalf("ListCars updated_since: %v", err)
	}
	if total != 1 || len(cars) != 1 || cars[0].ID != "new" {
		t.Fatalf("ids = %v (total %d), want [new] (1)", carIDs(cars), total)
	}
}

// TestSearch vérifie la recherche globale : correspondances multi-ressources,
// restriction par kinds, neutralisation des métacaractères LIKE et limit.
func TestSearch(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	mustExec(t, st, `INSERT INTO cars (id, game, name, make, class, pi, drivetrain) VALUES
		('civic','fh6','Civic Type R','Honda','A',780,'FWD'),
		('beetle','fh5','Beetle','Volkswagen','D',400,'RWD')`)
	mustExec(t, st, `INSERT INTO tracks (id, game, name, type) VALUES
		('shibuya','fh6','Shibuya Civic Circuit','street')`)
	mustExec(t, st, `INSERT INTO manufacturers (game, name, country) VALUES
		('fh6','Honda','Japan')`)
	mustExec(t, st, `INSERT INTO dlc_packs (id, game, name, kind) VALUES
		('pack','fh6','Civic Legends Pack','car_pass')`)

	// Tous types confondus : car + track + dlc_pack matchent « civic ».
	got, err := st.Search(ctx, store.SearchFilter{Game: "fh6", Q: "civic", Limit: 10})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	kinds := make([]string, 0, len(got))
	for _, r := range got {
		kinds = append(kinds, r.Kind)
	}
	if want := []string{"car", "dlc_pack", "track"}; !reflect.DeepEqual(kinds, want) {
		t.Fatalf("kinds = %v, want %v", kinds, want)
	}

	// Restriction kinds.
	onlyCars, err := st.Search(ctx, store.SearchFilter{Game: "fh6", Q: "civic", Kinds: []string{"car"}, Limit: 10})
	if err != nil {
		t.Fatalf("Search kinds=car: %v", err)
	}
	if len(onlyCars) != 1 || onlyCars[0].ID != "civic" {
		t.Fatalf("résultats kinds=car = %+v, want la voiture seule", onlyCars)
	}
	if onlyCars[0].Detail == nil || *onlyCars[0].Detail != "Honda · A 780" {
		t.Fatalf("detail = %v, want Honda · A 780", onlyCars[0].Detail)
	}

	// Isolation par jeu : fh5 ne voit pas les ressources fh6.
	fh5, err := st.Search(ctx, store.SearchFilter{Game: "fh5", Q: "civic", Limit: 10})
	if err != nil {
		t.Fatalf("Search fh5: %v", err)
	}
	if len(fh5) != 0 {
		t.Fatalf("fh5 = %+v, want vide", fh5)
	}

	// Métacaractère LIKE : '%' est une sous-chaîne littérale, pas un wildcard.
	esc, err := st.Search(ctx, store.SearchFilter{Game: "fh6", Q: "%", Limit: 10})
	if err != nil {
		t.Fatalf("Search %%: %v", err)
	}
	if len(esc) != 0 {
		t.Fatalf("recherche %% = %+v, want vide (littéral)", esc)
	}

	// Limit borne le total tous types confondus.
	one, err := st.Search(ctx, store.SearchFilter{Game: "fh6", Q: "civic", Limit: 1})
	if err != nil {
		t.Fatalf("Search limit=1: %v", err)
	}
	if len(one) != 1 {
		t.Fatalf("limit=1 → %d résultats", len(one))
	}
}

// TestGetCarObtain vérifie l'agrégat : toutes les voies présentes remontent,
// les absentes restent vides, et un id inconnu renvoie (nil, nil).
func TestGetCarObtain(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	mustExec(t, st, `INSERT INTO cars (id, game, name, make, class, pi, drivetrain, value_cr, obtain_method) VALUES
		('hidden','fh6','Hidden','Make','A',800,'RWD',NULL,NULL),
		('owner','fh6','Owner','Make','S1',880,'AWD',45000,'autoshow'),
		('plain','fh6','Plain','Make','D',500,'FWD',12000,'autoshow')`)
	mustExec(t, st, `INSERT INTO dlc_packs (id, game, name, kind) VALUES
		('pack','fh6','Car Pass','car_pass')`)
	mustExec(t, st, `INSERT INTO car_dlc (car_id, dlc_id) VALUES ('hidden','pack')`)
	mustExec(t, st, `INSERT INTO barn_finds (id, car_id, game, prerequisite_stamp_level) VALUES
		('bf1','hidden','fh6',3)`)
	mustExec(t, st, `INSERT INTO journal_tiers (id, game, track, level, name, reward_car_id) VALUES
		('jt1','fh6','horizon_festival',7,'Gold','hidden')`)
	mustExec(t, st, `INSERT INTO car_mastery_perks (id, car_id, name, effect_type, unlocked_car_id) VALUES
		('perk1','owner','Hidden unlock','car_unlock','hidden')`)

	o, err := st.GetCarObtain(ctx, "hidden")
	if err != nil {
		t.Fatalf("GetCarObtain: %v", err)
	}
	if o == nil {
		t.Fatal("GetCarObtain hidden = nil")
	}
	if len(o.DlcPacks) != 1 || o.DlcPacks[0].ID != "pack" {
		t.Errorf("dlcPacks = %+v, want [pack]", o.DlcPacks)
	}
	if o.BarnFind == nil || o.BarnFind.ID != "bf1" {
		t.Errorf("barnFind = %+v, want bf1", o.BarnFind)
	}
	if o.TreasureCar != nil {
		t.Errorf("treasureCar = %+v, want nil", o.TreasureCar)
	}
	if len(o.JournalTiers) != 1 || o.JournalTiers[0].ID != "jt1" {
		t.Errorf("journalTiers = %+v, want [jt1]", o.JournalTiers)
	}
	if len(o.MasteryUnlocks) != 1 || o.MasteryUnlocks[0].PerkID != "perk1" ||
		o.MasteryUnlocks[0].OwnerCarID != "owner" {
		t.Errorf("masteryUnlocks = %+v, want [perk1/owner]", o.MasteryUnlocks)
	}

	// Voiture sans voie particulière : agrégat vide mais méthode catalogue présente.
	p, err := st.GetCarObtain(ctx, "plain")
	if err != nil {
		t.Fatalf("GetCarObtain plain: %v", err)
	}
	if p == nil || len(p.DlcPacks) != 0 || p.BarnFind != nil || p.TreasureCar != nil ||
		len(p.JournalTiers) != 0 || len(p.MasteryUnlocks) != 0 {
		t.Errorf("plain = %+v, want voies vides", p)
	}
	if p.Car.ObtainMethod == nil || *p.Car.ObtainMethod != "autoshow" {
		t.Errorf("obtainMethod = %v, want autoshow", p.Car.ObtainMethod)
	}

	// Id inconnu → (nil, nil), le handler en fait un 404.
	ghost, err := st.GetCarObtain(ctx, "ghost")
	if err != nil {
		t.Fatalf("GetCarObtain ghost: %v", err)
	}
	if ghost != nil {
		t.Fatalf("ghost = %+v, want nil", ghost)
	}
}

// TestHiddenCarsCarIDFilter vérifie le lookup inverse car_id sur les deux tables.
func TestHiddenCarsCarIDFilter(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	mustExec(t, st, `INSERT INTO cars (id, game, name, make, class, pi, drivetrain) VALUES
		('a','fh6','A','Make','A',800,'RWD'),
		('b','fh6','B','Make','B',700,'RWD')`)
	mustExec(t, st, `INSERT INTO barn_finds (id, car_id, game) VALUES
		('bf-a','a','fh6'), ('bf-b','b','fh6')`)
	mustExec(t, st, `INSERT INTO treasure_cars (id, car_id, game) VALUES
		('tc-b','b','fh6')`)

	bf, total, err := st.ListBarnFinds(ctx, store.HiddenCarFilter{Game: "fh6", CarID: strptr("a"), Limit: 50})
	if err != nil {
		t.Fatalf("ListBarnFinds car_id: %v", err)
	}
	if total != 1 || len(bf) != 1 || bf[0].ID != "bf-a" {
		t.Fatalf("barn finds = %+v (total %d), want [bf-a] (1)", bf, total)
	}

	tc, total, err := st.ListTreasureCars(ctx, store.HiddenCarFilter{Game: "fh6", CarID: strptr("a"), Limit: 50})
	if err != nil {
		t.Fatalf("ListTreasureCars car_id: %v", err)
	}
	if total != 0 || len(tc) != 0 {
		t.Fatalf("treasure cars = %+v (total %d), want vide", tc, total)
	}
}

// TestGetByIDExpansion vérifie les gets par id ajoutés (trouvé / inconnu → nil).
func TestGetByIDExpansion(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	mustExec(t, st, `INSERT INTO cars (id, game, name, make, class, pi, drivetrain) VALUES
		('car','fh6','Car','Make','A',800,'RWD')`)
	mustExec(t, st, `INSERT INTO tracks (id, game, name, type, region) VALUES
		('trk','fh6','Track','circuit','Tokyo')`)
	mustExec(t, st, `INSERT INTO pr_stunts (id, game, type, name) VALUES
		('stunt','fh6','speed_trap','Trap')`)
	mustExec(t, st, `INSERT INTO events (id, game, name, type) VALUES
		('evt','fh6','Showcase Run','showcase')`)
	mustExec(t, st, `INSERT INTO dlc_packs (id, game, name, kind) VALUES
		('pack','fh6','Pack','expansion')`)
	mustExec(t, st, `INSERT INTO barn_finds (id, car_id, game) VALUES ('bf','car','fh6')`)
	mustExec(t, st, `INSERT INTO treasure_cars (id, car_id, game) VALUES ('tc','car','fh6')`)

	if trk, err := st.GetTrack(ctx, "trk"); err != nil || trk == nil || trk.Name != "Track" {
		t.Errorf("GetTrack = %+v, %v", trk, err)
	}
	if ghost, err := st.GetTrack(ctx, "ghost"); err != nil || ghost != nil {
		t.Errorf("GetTrack ghost = %+v, %v, want nil", ghost, err)
	}
	if p, err := st.GetPRStunt(ctx, "stunt"); err != nil || p == nil || p.Type != "speed_trap" {
		t.Errorf("GetPRStunt = %+v, %v", p, err)
	}
	// showcase accepté par le CHECK events.type (nouveau type FH6).
	if e, err := st.GetEvent(ctx, "evt"); err != nil || e == nil || e.Type != "showcase" {
		t.Errorf("GetEvent = %+v, %v", e, err)
	}
	if d, err := st.GetDlcPack(ctx, "pack"); err != nil || d == nil || d.Kind != "expansion" {
		t.Errorf("GetDlcPack = %+v, %v", d, err)
	}
	if b, err := st.GetBarnFind(ctx, "bf"); err != nil || b == nil || b.CarID != "car" {
		t.Errorf("GetBarnFind = %+v, %v", b, err)
	}
	if tc, err := st.GetTreasureCar(ctx, "tc"); err != nil || tc == nil || tc.CarID != "car" {
		t.Errorf("GetTreasureCar = %+v, %v", tc, err)
	}

	// RandomTrack : seul tracé → toujours lui ; filtre sans correspondance → nil.
	if r, err := st.RandomTrack(ctx, store.GeoFilter{Game: "fh6"}); err != nil || r == nil || r.ID != "trk" {
		t.Errorf("RandomTrack = %+v, %v", r, err)
	}
	if r, err := st.RandomTrack(ctx, store.GeoFilter{Game: "fh6", Type: strptr("dirt")}); err != nil || r != nil {
		t.Errorf("RandomTrack dirt = %+v, %v, want nil", r, err)
	}
}

// TestStoriesTours vérifie les listes Discovery : filtre region, pagination et
// upserts idempotents.
func TestStoriesTours(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	stories := []store.Story{
		{ID: "s1", Game: "fh6", Name: "Taxi Tales", Region: strptr("Tokyo"), ChaptersCount: intptr(8)},
		{ID: "s2", Game: "fh6", Name: "Drift Legends", Region: strptr("Hakone")},
	}
	if err := st.UpsertStories(ctx, stories); err != nil {
		t.Fatalf("UpsertStories: %v", err)
	}
	if err := st.UpsertStories(ctx, stories); err != nil {
		t.Fatalf("UpsertStories re-run: %v", err)
	}
	tours := []store.Tour{{ID: "t1", Game: "fh6", Name: "Tour of Tokyo", Region: strptr("Tokyo")}}
	if err := st.UpsertTours(ctx, tours); err != nil {
		t.Fatalf("UpsertTours: %v", err)
	}

	all, total, err := st.ListStories(ctx, store.DiscoveryFilter{Game: "fh6", Limit: 50})
	if err != nil {
		t.Fatalf("ListStories: %v", err)
	}
	if total != 2 || len(all) != 2 || all[0].Name != "Drift Legends" {
		t.Fatalf("stories = %+v (total %d), want 2 triées par nom", all, total)
	}

	tokyo, total, err := st.ListStories(ctx, store.DiscoveryFilter{Game: "fh6", Region: strptr("Tokyo"), Limit: 50})
	if err != nil {
		t.Fatalf("ListStories region: %v", err)
	}
	if total != 1 || len(tokyo) != 1 || tokyo[0].ID != "s1" {
		t.Fatalf("stories Tokyo = %+v (total %d), want [s1]", tokyo, total)
	}
	if tokyo[0].ChaptersCount == nil || *tokyo[0].ChaptersCount != 8 {
		t.Fatalf("chaptersCount = %v, want 8", tokyo[0].ChaptersCount)
	}

	tr, total, err := st.ListTours(ctx, store.DiscoveryFilter{Game: "fh6", Limit: 50})
	if err != nil {
		t.Fatalf("ListTours: %v", err)
	}
	if total != 1 || len(tr) != 1 || tr[0].ID != "t1" {
		t.Fatalf("tours = %+v (total %d), want [t1]", tr, total)
	}
}

// TestReferenceGeoFacets vérifie les nouvelles facettes : types canoniques avec
// count 0 inclus, régions agrégées sur tracks ∪ events ∪ pr_stunts.
func TestReferenceGeoFacets(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	mustExec(t, st, `INSERT INTO tracks (id, game, name, type, region) VALUES
		('t1','fh6','T1','circuit','Tokyo'),
		('t2','fh6','T2','street','Tokyo')`)
	mustExec(t, st, `INSERT INTO events (id, game, name, type, region) VALUES
		('e1','fh6','E1','showcase','Hakone')`)
	mustExec(t, st, `INSERT INTO pr_stunts (id, game, type, name, region) VALUES
		('p1','fh6','speed_trap','P1','Tokyo')`)

	ref, err := st.GetReference(ctx, "fh6")
	if err != nil {
		t.Fatalf("GetReference: %v", err)
	}

	asMap := func(in []store.RefCount) map[string]int64 {
		out := map[string]int64{}
		for _, r := range in {
			out[r.Code] = r.Count
		}
		return out
	}

	tt := asMap(ref.TrackTypes)
	if len(ref.TrackTypes) != 7 || tt["circuit"] != 1 || tt["street"] != 1 || tt["dirt"] != 0 {
		t.Errorf("trackTypes = %v, want 7 codes canoniques avec circuit/street à 1", tt)
	}
	et := asMap(ref.EventTypes)
	if len(ref.EventTypes) != 10 || et["showcase"] != 1 {
		t.Errorf("eventTypes = %v, want 10 codes canoniques avec showcase à 1", et)
	}
	st2 := asMap(ref.PRStuntTypes)
	if len(ref.PRStuntTypes) != 4 || st2["speed_trap"] != 1 {
		t.Errorf("prStuntTypes = %v, want 4 codes canoniques", st2)
	}
	// Régions : Tokyo = 2 tracks + 1 stunt = 3 ; Hakone = 1 event. Plus fréquentes d'abord.
	if len(ref.Regions) != 2 || ref.Regions[0].Code != "Tokyo" || ref.Regions[0].Count != 3 ||
		ref.Regions[1].Code != "Hakone" || ref.Regions[1].Count != 1 {
		t.Errorf("regions = %v, want [Tokyo 3, Hakone 1]", ref.Regions)
	}
}

// TestReferenceObtainFacet vérifie la facette obtainMethods : ordre canonique
// complet (count 0 inclus), une voiture multi-méthodes comptée dans chacune,
// valeurs couvrant deux tokens sommées (wristband), obtain_method NULL exclu,
// isolation par jeu.
func TestReferenceObtainFacet(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	mustExec(t, st, `INSERT INTO cars (id, game, name, make, class, pi, drivetrain, obtain_method) VALUES
		('rf-1','fh6','C1','Make','A',780,'RWD','Autoshow, Wheelspin'),
		('rf-2','fh6','C2','Make','B',600,'RWD','Yellow Wristband'),
		('rf-3','fh6','C3','Make','C',650,'RWD','Wristband reward'),
		('rf-4','fh6','C4','Make','D',500,'RWD',NULL),
		('rf-5','fh5','C5','Make','D',400,'RWD','Autoshow')`)

	ref, err := st.GetReference(ctx, "fh6")
	if err != nil {
		t.Fatalf("GetReference: %v", err)
	}

	om := map[string]int64{}
	for _, r := range ref.ObtainMethods {
		om[r.Code] = r.Count
	}
	if len(ref.ObtainMethods) != 17 {
		t.Fatalf("obtainMethods = %d codes, want 17 canoniques", len(ref.ObtainMethods))
	}
	if om["autoshow"] != 1 || om["wheelspin"] != 1 {
		t.Errorf("autoshow/wheelspin = %d/%d, want 1/1 (rf-1 compté dans chacune, fh5 exclu)",
			om["autoshow"], om["wheelspin"])
	}
	if om["wristband"] != 2 {
		t.Errorf("wristband = %d, want 2 (Yellow Wristband + Wristband reward)", om["wristband"])
	}
	if om["barn_find"] != 0 {
		t.Errorf("barn_find = %d, want 0", om["barn_find"])
	}
}
