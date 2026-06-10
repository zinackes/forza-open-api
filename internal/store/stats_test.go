// Test d'intégration de GetStats : agrégats du catalogue sur Postgres jetable.
// Vérifie l'isolation par jeu, le count 0 canonique (class/drivetrain), les
// décennies, les volumes par constructeur (ordre count desc / make asc), les
// paliers PI de 50 (ordre croissant) et les classements (pi réel ; speed /
// acceleration sur la note in-game, NULL exclu). Réutilise newTestStore /
// mustExec (integration_test.go, même package store_test).
package store_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/zinackes/forza-open-api/internal/store"
)

func TestGetStats(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	// 5 voitures fh6 (c5 : year/body_type/category/stats NULL → exclue des
	// agrégats correspondants) + 1 fh5 (isolée du scope fh6).
	mustExec(t, st, `INSERT INTO cars (id, game, name, make, class, pi, drivetrain, year, body_type, category, stats) VALUES
		('c1','fh6','RX-7','Mazda','A',780,'RWD',1997,'coupe','Modern Supercars','{"speed":9.0,"acceleration":8.0}'),
		('c2','fh6','RX-8','Mazda','A',760,'AWD',2003,'coupe','Modern Supercars','{"speed":7.0,"acceleration":9.5}'),
		('c3','fh6','Civic','Honda','S1',850,'RWD',1999,'hatchback','Retro Hot Hatch','{"speed":8.0,"acceleration":7.0}'),
		('c4','fh6','GR Track','Toyota','R',998,'AWD',2015,'race car','Track Toys','{"speed":10.0,"acceleration":10.0}'),
		('c5','fh6','Kei','Toyota','D',500,'FWD',NULL,NULL,NULL,NULL),
		('c6','fh5','Skyline','Nissan','B',650,'RWD',1980,'sedan','Classics','{"speed":5.0}')`)

	stats, err := st.GetStats(ctx, "fh6")
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}

	// class : carte canonique complète (8 codes), count 0 inclus ; fh5 (B) exclu.
	wantClass := map[string]int64{"D": 1, "C": 0, "B": 0, "A": 2, "S1": 1, "S2": 0, "X": 0, "R": 1}
	if !reflect.DeepEqual(stats.Class, wantClass) {
		t.Errorf("class = %v\nwant %v", stats.Class, wantClass)
	}

	// drivetrain : carte canonique (3 codes), count 0 inclus.
	wantDrivetrain := map[string]int64{"FWD": 1, "RWD": 2, "AWD": 2}
	if !reflect.DeepEqual(stats.Drivetrain, wantDrivetrain) {
		t.Errorf("drivetrain = %v\nwant %v", stats.Drivetrain, wantDrivetrain)
	}

	// yearDecade : décennie de début ; c5 (year NULL) exclue, décennies vides absentes.
	wantDecade := map[string]int64{"1990": 2, "2000": 1, "2010": 1}
	if !reflect.DeepEqual(stats.YearDecade, wantDecade) {
		t.Errorf("yearDecade = %v\nwant %v", stats.YearDecade, wantDecade)
	}

	// bodyType / category : valeurs présentes (c5 NULL exclue).
	wantBody := map[string]int64{"coupe": 2, "hatchback": 1, "race car": 1}
	if !reflect.DeepEqual(stats.BodyType, wantBody) {
		t.Errorf("bodyType = %v\nwant %v", stats.BodyType, wantBody)
	}
	wantCategory := map[string]int64{"Modern Supercars": 2, "Retro Hot Hatch": 1, "Track Toys": 1}
	if !reflect.DeepEqual(stats.Category, wantCategory) {
		t.Errorf("category = %v\nwant %v", stats.Category, wantCategory)
	}

	// manufacturer : count desc puis make asc (Mazda/Toyota à 2 → Mazda d'abord).
	wantMakes := []store.RefCount{
		{Code: "Mazda", Count: 2}, {Code: "Toyota", Count: 2}, {Code: "Honda", Count: 1},
	}
	if !reflect.DeepEqual(stats.Manufacturer, wantMakes) {
		t.Errorf("manufacturer = %v\nwant %v", stats.Manufacturer, wantMakes)
	}

	// piHistogram : paliers de 50 présents, ordre croissant, borne haute exclue.
	wantHist := []store.RefCount{
		{Code: "500-550", Count: 1}, {Code: "750-800", Count: 2},
		{Code: "850-900", Count: 1}, {Code: "950-1000", Count: 1},
	}
	if !reflect.DeepEqual(stats.PiHistogram, wantHist) {
		t.Errorf("piHistogram = %v\nwant %v", stats.PiHistogram, wantHist)
	}

	// top pi : colonne réelle, desc puis id ; toutes les voitures (pi NOT NULL).
	wantTopPI := []store.TopCar{
		{CarID: "c4", Name: "GR Track", Value: 998},
		{CarID: "c3", Name: "Civic", Value: 850},
		{CarID: "c1", Name: "RX-7", Value: 780},
		{CarID: "c2", Name: "RX-8", Value: 760},
		{CarID: "c5", Name: "Kei", Value: 500},
	}
	if !reflect.DeepEqual(stats.TopPI, wantTopPI) {
		t.Errorf("topPI = %v\nwant %v", stats.TopPI, wantTopPI)
	}

	// top speed : note in-game stats.speed desc ; c5 (stats NULL) exclue.
	wantTopSpeed := []store.TopCar{
		{CarID: "c4", Name: "GR Track", Value: 10},
		{CarID: "c1", Name: "RX-7", Value: 9},
		{CarID: "c3", Name: "Civic", Value: 8},
		{CarID: "c2", Name: "RX-8", Value: 7},
	}
	if !reflect.DeepEqual(stats.TopSpeed, wantTopSpeed) {
		t.Errorf("topSpeed = %v\nwant %v", stats.TopSpeed, wantTopSpeed)
	}

	// top acceleration : note in-game stats.acceleration desc ; c5 exclue.
	wantTopAccel := []store.TopCar{
		{CarID: "c4", Name: "GR Track", Value: 10},
		{CarID: "c2", Name: "RX-8", Value: 9.5},
		{CarID: "c1", Name: "RX-7", Value: 8},
		{CarID: "c3", Name: "Civic", Value: 7},
	}
	if !reflect.DeepEqual(stats.TopAcceleration, wantTopAccel) {
		t.Errorf("topAccel = %v\nwant %v", stats.TopAcceleration, wantTopAccel)
	}

	// Jeu inconnu : class/drivetrain canoniques tout à 0, reste vide (pas d'erreur).
	ghost, err := st.GetStats(ctx, "ghost")
	if err != nil {
		t.Fatalf("GetStats ghost: %v", err)
	}
	if len(ghost.Class) != 8 || len(ghost.Drivetrain) != 3 {
		t.Errorf("ghost canoniques: class=%d drivetrain=%d, want 8/3", len(ghost.Class), len(ghost.Drivetrain))
	}
	for code, n := range ghost.Class {
		if n != 0 {
			t.Errorf("ghost class %s = %d, want 0", code, n)
		}
	}
	if len(ghost.BodyType) != 0 || len(ghost.Category) != 0 || len(ghost.YearDecade) != 0 {
		t.Errorf("ghost facettes libres non vides: body=%d cat=%d decade=%d",
			len(ghost.BodyType), len(ghost.Category), len(ghost.YearDecade))
	}
	if len(ghost.Manufacturer) != 0 || len(ghost.PiHistogram) != 0 {
		t.Errorf("ghost listes non vides: make=%d hist=%d", len(ghost.Manufacturer), len(ghost.PiHistogram))
	}
	if len(ghost.TopPI) != 0 || len(ghost.TopSpeed) != 0 || len(ghost.TopAcceleration) != 0 {
		t.Errorf("ghost tops non vides: pi=%d speed=%d accel=%d",
			len(ghost.TopPI), len(ghost.TopSpeed), len(ghost.TopAcceleration))
	}
}
