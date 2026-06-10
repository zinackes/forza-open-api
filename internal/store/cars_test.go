// Tests d'intégration du catalogue (cars/manufacturers) : Postgres jetable via
// testcontainers-go, schéma db/init.sql, fixtures en SQL paramétré. Couvre les
// filtres scalaires, la pagination, le tri (param sort whitelisté, défaut pi puis
// départage name, id), GetCar et la
// liste des constructeurs. Partage newTestStore/mustExec avec integration_test.go.
//
// Nécessite Docker (cf. .claude/rules/testing.md). En CI, Docker est présent.
package store_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/zinackes/forza-open-api/internal/store"
)

func strptr(s string) *string { return &s }
func intptr(i int) *int       { return &i }

// carIDs projette une page de voitures sur leurs identifiants, dans l'ordre rendu
// par le store (utile pour asserter le tri sans comparer toute la structure).
func carIDs(cars []store.Car) []string {
	out := make([]string, 0, len(cars))
	for _, c := range cars {
		out = append(out, c.ID)
	}
	return out
}

// TestListCarsScalarFilters couvre les filtres simples de ListCars (make, class,
// pi_min/pi_max, drivetrain, q sur name/model, combinés), l'isolation par jeu et
// le cas « aucun résultat ». Les IDs attendus sont ordonnés (ORDER BY pi, name, id).
func TestListCarsScalarFilters(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	mustExec(t, st, `INSERT INTO cars (id, game, name, make, model, class, pi, drivetrain) VALUES
		('audi-r8','fh6','R8','Audi',NULL,'S1',850,'AWD'),
		('ford-gt','fh6','GT','Ford',NULL,'S2',920,'RWD'),
		('honda-civic','fh6','Civic Type R','Honda',NULL,'A',780,'FWD'),
		('mazda-rx7','fh6','RX-7','Mazda','FD','A',780,'RWD'),
		('toyota-ae86','fh6','AE86','Toyota',NULL,'B',600,'RWD'),
		('vw-beetle','fh5','Beetle','Volkswagen',NULL,'D',400,'RWD')`)

	cases := []struct {
		name   string
		filter store.CarFilter
		want   []string // IDs ordonnés
		total  int64
	}{
		{"make=Honda", store.CarFilter{Game: "fh6", Make: strptr("Honda"), Limit: 50}, []string{"honda-civic"}, 1},
		{"class=A", store.CarFilter{Game: "fh6", Class: strptr("A"), Limit: 50}, []string{"honda-civic", "mazda-rx7"}, 2},
		{"pi_min=850", store.CarFilter{Game: "fh6", PIMin: intptr(850), Limit: 50}, []string{"audi-r8", "ford-gt"}, 2},
		{"pi_max=700", store.CarFilter{Game: "fh6", PIMax: intptr(700), Limit: 50}, []string{"toyota-ae86"}, 1},
		{"pi range 750-900", store.CarFilter{Game: "fh6", PIMin: intptr(750), PIMax: intptr(900), Limit: 50}, []string{"honda-civic", "mazda-rx7", "audi-r8"}, 3},
		{"drivetrain=RWD", store.CarFilter{Game: "fh6", Drivetrain: strptr("RWD"), Limit: 50}, []string{"toyota-ae86", "mazda-rx7", "ford-gt"}, 3},
		{"q sur name", store.CarFilter{Game: "fh6", Q: strptr("Civic"), Limit: 50}, []string{"honda-civic"}, 1},
		{"q sur model", store.CarFilter{Game: "fh6", Q: strptr("FD"), Limit: 50}, []string{"mazda-rx7"}, 1},
		{"class=A + drivetrain=RWD", store.CarFilter{Game: "fh6", Class: strptr("A"), Drivetrain: strptr("RWD"), Limit: 50}, []string{"mazda-rx7"}, 1},
		{"isolation par jeu (fh5)", store.CarFilter{Game: "fh5", Limit: 50}, []string{"vw-beetle"}, 1},
		{"aucun résultat", store.CarFilter{Game: "fh6", Make: strptr("Bugatti"), Limit: 50}, []string{}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cars, total, err := st.ListCars(ctx, tc.filter)
			if err != nil {
				t.Fatalf("ListCars: %v", err)
			}
			if total != tc.total {
				t.Errorf("total = %d, want %d", total, tc.total)
			}
			if got := carIDs(cars); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ids = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestListCarsPagination vérifie le découpage LIMIT/OFFSET (total stable d'une page
// à l'autre) et le cas « page hors borne » : au-delà des données la page est vide
// mais le total reste exact (COUNT séparé, indépendant de l'OFFSET).
func TestListCarsPagination(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	// pi distincts → ordre déterministe p1..p5.
	mustExec(t, st, `INSERT INTO cars (id, game, name, make, class, pi, drivetrain) VALUES
		('p1','fh6','A','Make','D',500,'RWD'),
		('p2','fh6','B','Make','C',600,'RWD'),
		('p3','fh6','C','Make','B',700,'RWD'),
		('p4','fh6','D','Make','A',800,'RWD'),
		('p5','fh6','E','Make','S1',850,'RWD')`)

	pages := []struct {
		name   string
		offset int
		want   []string
		total  int64
	}{
		{"page 1 (size 2)", 0, []string{"p1", "p2"}, 5},
		{"page 2 (size 2)", 2, []string{"p3", "p4"}, 5},
		{"page 3 (size 2, partielle)", 4, []string{"p5"}, 5},
		{"page hors borne (total exact)", 100, []string{}, 5},
	}
	for _, tc := range pages {
		t.Run(tc.name, func(t *testing.T) {
			cars, total, err := st.ListCars(ctx, store.CarFilter{Game: "fh6", Limit: 2, Offset: tc.offset})
			if err != nil {
				t.Fatalf("ListCars: %v", err)
			}
			if total != tc.total {
				t.Errorf("total = %d, want %d", total, tc.total)
			}
			if got := carIDs(cars); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ids = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestListCarsSorting vérifie le paramètre sort : chaque clé whitelistée (pi, name,
// year, value) dans les deux sens, le défaut (pi croissant), le départage stable
// name puis id (car-alpha-1/2 : mêmes pi, name, year, value), les NULL renvoyés en
// dernier quel que soit le sens (year/value absents) et le repli sur le défaut pour
// une clé inconnue (impossible via l'API — l'enum du contrat la rejette en 400 —
// mais le store ne doit jamais interpoler une valeur non mappée).
func TestListCarsSorting(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	// Insertion volontairement désordonnée : l'ordre attendu vient du tri SQL.
	// z-low : pi le plus bas, year le plus ancien, value NULL.
	// car-alpha-1 : year NULL. car-alpha-1/2 : même pi/name/value → départage id.
	mustExec(t, st, `INSERT INTO cars (id, game, name, make, class, pi, drivetrain, year, value_cr) VALUES
		('car-beta','fh6','Beta','Make','A',780,'RWD',2005,50000),
		('car-alpha-2','fh6','Alpha','Make','A',780,'RWD',2020,150000),
		('z-low','fh6','Whatever','Make','D',500,'RWD',1998,NULL),
		('car-alpha-1','fh6','Alpha','Make','A',780,'RWD',NULL,150000)`)

	cases := []struct {
		name string
		sort string
		want []string // IDs dans l'ordre attendu
	}{
		{"défaut (pi asc)", "", []string{"z-low", "car-alpha-1", "car-alpha-2", "car-beta"}},
		{"pi", "pi", []string{"z-low", "car-alpha-1", "car-alpha-2", "car-beta"}},
		{"-pi", "-pi", []string{"car-alpha-1", "car-alpha-2", "car-beta", "z-low"}},
		{"name", "name", []string{"car-alpha-1", "car-alpha-2", "car-beta", "z-low"}},
		{"-name", "-name", []string{"z-low", "car-beta", "car-alpha-1", "car-alpha-2"}},
		{"year (NULL dernier)", "year", []string{"z-low", "car-beta", "car-alpha-2", "car-alpha-1"}},
		{"-year (NULL dernier)", "-year", []string{"car-alpha-2", "car-beta", "z-low", "car-alpha-1"}},
		{"value (NULL dernier)", "value", []string{"car-beta", "car-alpha-1", "car-alpha-2", "z-low"}},
		{"-value (NULL dernier)", "-value", []string{"car-alpha-1", "car-alpha-2", "car-beta", "z-low"}},
		{"clé inconnue → défaut", "bogus", []string{"z-low", "car-alpha-1", "car-alpha-2", "car-beta"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cars, _, err := st.ListCars(ctx, store.CarFilter{Game: "fh6", Sort: tc.sort, Limit: 50})
			if err != nil {
				t.Fatalf("ListCars(sort=%q): %v", tc.sort, err)
			}
			if got := carIDs(cars); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("sort=%q : ordre = %v, want %v", tc.sort, got, tc.want)
			}
		})
	}
}

// TestGetCar vérifie la lecture par id : remontée complète des champs (avec mapping
// nullable NULL → nil et JSONB stats brut), et (nil, nil) pour un id inconnu — le
// handler en fait un 404.
func TestGetCar(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	mustExec(t, st, `INSERT INTO cars
		(id, game, name, make, model, year, class, pi, drivetrain, stats, body_type,
		 category, rarity, value_cr, obtain_method, image_url, created_at) VALUES
		('rx7-full','fh6','RX-7','Mazda','FD',1998,'A',780,'RWD','{"speed":7.5}','coupe',
		 'JDM','rare',250000,'Autoshow','https://img/rx7.png','2026-01-01T00:00:00Z'),
		('plain','fh6','Plain','Make',NULL,NULL,'D',500,'FWD',NULL,NULL,NULL,NULL,NULL,
		 NULL,NULL,'2026-01-02T00:00:00Z')`)

	got, err := st.GetCar(ctx, "rx7-full")
	if err != nil {
		t.Fatalf("GetCar: %v", err)
	}
	if got == nil {
		t.Fatal("GetCar rx7-full = nil, want une voiture")
	}
	if got.Game != "fh6" || got.Name != "RX-7" || got.Make != "Mazda" ||
		got.Class != "A" || got.PI != 780 || got.Drivetrain != "RWD" {
		t.Errorf("champs requis incorrects: %+v", got)
	}
	if got.Model == nil || *got.Model != "FD" {
		t.Errorf("model = %v, want FD", got.Model)
	}
	if got.Year == nil || *got.Year != 1998 {
		t.Errorf("year = %v, want 1998", got.Year)
	}
	if got.Category == nil || *got.Category != "JDM" {
		t.Errorf("category = %v, want JDM", got.Category)
	}
	if got.ValueCr == nil || *got.ValueCr != 250000 {
		t.Errorf("value_cr = %v, want 250000", got.ValueCr)
	}
	if got.ImageURL == nil || *got.ImageURL != "https://img/rx7.png" {
		t.Errorf("image_url = %v, want https://img/rx7.png", got.ImageURL)
	}
	if len(got.Stats) == 0 {
		t.Error("stats vide, want JSONB brut")
	}

	// Nullables NULL → nil ; pas de stats.
	plain, err := st.GetCar(ctx, "plain")
	if err != nil {
		t.Fatalf("GetCar plain: %v", err)
	}
	if plain.Model != nil || plain.Year != nil || plain.Category != nil ||
		plain.ValueCr != nil || plain.ImageURL != nil || plain.Stats != nil {
		t.Errorf("plain : nullables non nil: %+v", plain)
	}

	// Id inconnu → (nil, nil).
	none, err := st.GetCar(ctx, "ghost")
	if err != nil {
		t.Fatalf("GetCar ghost: %v", err)
	}
	if none != nil {
		t.Errorf("GetCar ghost = %+v, want nil", none)
	}
}

// TestListManufacturers vérifie l'ordre alphabétique, le car_count via LEFT JOIN
// (constructeur sans voiture → 0), le mapping country NULL → nil et l'isolation
// par jeu (les constructeurs d'un autre jeu n'apparaissent pas).
func TestListManufacturers(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	mustExec(t, st, `INSERT INTO cars (id, game, name, make, class, pi, drivetrain) VALUES
		('m-civic','fh6','Civic','Honda','A',780,'FWD'),
		('m-nsx','fh6','NSX','Honda','S1',850,'RWD'),
		('m-gt','fh6','GT','Ford','S2',920,'RWD'),
		('m-beetle','fh5','Beetle','Volkswagen','D',400,'RWD')`)
	mustExec(t, st, `INSERT INTO manufacturers (game, name, country) VALUES
		('fh6','Honda','Japan'),
		('fh6','Ford','USA'),
		('fh6','Koenigsegg','Sweden'),
		('fh6','Mystery',NULL),
		('fh5','Volkswagen','Germany')`)

	ms, err := st.ListManufacturers(ctx, store.ManufacturerFilter{Game: "fh6"})
	if err != nil {
		t.Fatalf("ListManufacturers: %v", err)
	}

	// Ordre alphabétique ; Volkswagen (fh5) exclu par le filtre game.
	wantOrder := []string{"Ford", "Honda", "Koenigsegg", "Mystery"}
	got := make([]string, len(ms))
	for i, m := range ms {
		got[i] = m.Name
	}
	if !reflect.DeepEqual(got, wantOrder) {
		t.Fatalf("ordre = %v, want %v", got, wantOrder)
	}

	by := make(map[string]store.Manufacturer, len(ms))
	for _, m := range ms {
		by[m.Name] = m
	}
	// car_count via LEFT JOIN : Honda 2, Ford 1, Koenigsegg 0 (aucune voiture sourcée).
	if by["Honda"].CarCount != 2 {
		t.Errorf("Honda car_count = %d, want 2", by["Honda"].CarCount)
	}
	if by["Ford"].CarCount != 1 {
		t.Errorf("Ford car_count = %d, want 1", by["Ford"].CarCount)
	}
	if by["Koenigsegg"].CarCount != 0 {
		t.Errorf("Koenigsegg car_count = %d, want 0", by["Koenigsegg"].CarCount)
	}
	// country : présent pour Ford, NULL → nil pour Mystery.
	if by["Ford"].Country == nil || *by["Ford"].Country != "USA" {
		t.Errorf("Ford country = %v, want USA", by["Ford"].Country)
	}
	if by["Mystery"].Country != nil {
		t.Errorf("Mystery country = %v, want nil", by["Mystery"].Country)
	}

	// Jeu sans constructeurs sourcés → liste vide (pas d'erreur).
	none, err := st.ListManufacturers(ctx, store.ManufacturerFilter{Game: "ghost"})
	if err != nil {
		t.Fatalf("ListManufacturers ghost: %v", err)
	}
	if len(none) != 0 {
		t.Errorf("ghost = %d, want 0", len(none))
	}
}
