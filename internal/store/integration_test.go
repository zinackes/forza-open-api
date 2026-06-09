// Tests d'intégration store : Postgres jetable via testcontainers-go, schéma
// appliqué depuis db/init.sql, fixtures insérées en SQL paramétré. Vérifie
// notamment que le filtre dlc= de ListCars ne renvoie que les voitures du pack.
//
// Nécessite Docker (cf. .claude/rules/testing.md). En CI, Docker est présent.
package store_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/zinackes/forza-open-api/internal/store"
)

// newTestStore lève un Postgres jetable, applique db/init.sql, et renvoie un
// Store branché dessus (Redis nil : les lectures testées n'y touchent pas).
func newTestStore(t *testing.T) *store.Store {
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

	return &store.Store{DB: pool}
}

func mustExec(t *testing.T, st *store.Store, sql string) {
	t.Helper()
	if _, err := st.DB.Exec(context.Background(), sql); err != nil {
		t.Fatalf("exec fixture: %v\nSQL: %s", err, sql)
	}
}

// TestListCarsFilterByDlc vérifie qu'un filtre par DLC liste bien (et seulement)
// les voitures liées au pack via car_dlc.
func TestListCarsFilterByDlc(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	// 2 voitures FH6 ; un Car Pass qui ne contient que la première.
	mustExec(t, st, `INSERT INTO cars (id, game, name, make, class, pi, drivetrain) VALUES
		('car-1','fh6','Quartz Regalia','Cindra','S1',850,'AWD'),
		('car-2','fh6','Ford GT','Ford','S2',920,'RWD')`)
	mustExec(t, st, `INSERT INTO dlc_packs (id, game, name, kind, source) VALUES
		('fh6-car-pass','fh6','FH6 Car Pass','car_pass','forza.net')`)
	mustExec(t, st, `INSERT INTO car_dlc (car_id, dlc_id) VALUES ('car-1','fh6-car-pass')`)

	dlc := "fh6-car-pass"
	cars, total, err := st.ListCars(ctx, store.CarFilter{Game: "fh6", Dlc: &dlc, Limit: 50})
	if err != nil {
		t.Fatalf("ListCars dlc: %v", err)
	}
	if total != 1 || len(cars) != 1 {
		t.Fatalf("filtre dlc: total=%d len=%d, want 1/1", total, len(cars))
	}
	if cars[0].ID != "car-1" {
		t.Errorf("voiture du pack = %q, want car-1", cars[0].ID)
	}

	// Sans filtre dlc : les 2 voitures du jeu.
	all, total, err := st.ListCars(ctx, store.CarFilter{Game: "fh6", Limit: 50})
	if err != nil {
		t.Fatalf("ListCars sans filtre: %v", err)
	}
	if total != 2 || len(all) != 2 {
		t.Fatalf("sans filtre: total=%d len=%d, want 2/2", total, len(all))
	}
}

// TestListDlcPacks vérifie le filtre par jeu et l'ordre (sortis d'abord, packs
// planifiés released_at NULL en dernier).
func TestListDlcPacks(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	mustExec(t, st, `INSERT INTO dlc_packs (id, game, name, kind, released_at) VALUES
		('fh6-car-pass','fh6','FH6 Car Pass','car_pass','2026-01-01T00:00:00Z'),
		('fh6-exp-1','fh6','Expansion 1 (planifiée)','expansion',NULL),
		('fh5-hot-wheels','fh5','Hot Wheels','expansion','2021-07-01T00:00:00Z')`)

	packs, err := st.ListDlcPacks(ctx, "fh6")
	if err != nil {
		t.Fatalf("ListDlcPacks: %v", err)
	}
	if len(packs) != 2 {
		t.Fatalf("packs fh6 = %d, want 2 (fh5 exclu par le filtre game)", len(packs))
	}
	// released_at DESC NULLS LAST : le pack daté avant l'expansion planifiée (NULL).
	if packs[0].ID != "fh6-car-pass" || packs[1].ID != "fh6-exp-1" {
		t.Errorf("ordre = [%s, %s], want [fh6-car-pass, fh6-exp-1]", packs[0].ID, packs[1].ID)
	}
}

// TestListBarnFinds vérifie le filtre par game/region, l'ordre par stamp et le
// mapping des champs (coords NULL → nil, stamp/restauration scannés).
func TestListBarnFinds(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	mustExec(t, st, `INSERT INTO cars (id, game, name, make, class, pi, drivetrain) VALUES
		('bf-car-1','fh6','Mazda RX-7','Mazda','A',800,'RWD'),
		('bf-car-2','fh6','Toyota AE86','Toyota','B',600,'RWD')`)
	// Stamp 7 inséré avant stamp 1 : l'ordre attendu vient du tri SQL, pas de l'insert.
	mustExec(t, st, `INSERT INTO barn_finds
		(id, car_id, game, region, search_zone_center_lat, search_zone_center_lng,
		 search_zone_radius_m, prerequisite_stamp_level, restoration_time_h, source) VALUES
		('bf-2','bf-car-2','fh6','Kanto',35.6,139.7,250,7,12,'fandom'),
		('bf-1','bf-car-1','fh6','Kansai',NULL,NULL,NULL,1,6,'gamesradar'),
		('bf-fh5','bf-car-1','fh5','Mexico',NULL,NULL,NULL,3,9,'fandom')`)

	finds, total, err := st.ListBarnFinds(ctx, store.HiddenCarFilter{Game: "fh6", Limit: 50})
	if err != nil {
		t.Fatalf("ListBarnFinds: %v", err)
	}
	if total != 2 || len(finds) != 2 {
		t.Fatalf("fh6: total=%d len=%d, want 2/2 (fh5 exclu)", total, len(finds))
	}
	// ORDER BY prerequisite_stamp_level : bf-1 (stamp 1) avant bf-2 (stamp 7).
	if finds[0].ID != "bf-1" || finds[1].ID != "bf-2" {
		t.Errorf("ordre = [%s, %s], want [bf-1, bf-2]", finds[0].ID, finds[1].ID)
	}
	// bf-1 : coords NULL → nil, stamp/restauration présents.
	if finds[0].SearchZoneCenterLat != nil || finds[0].SearchZoneRadiusM != nil {
		t.Errorf("bf-1 coords/rayon = %v/%v, want nil", finds[0].SearchZoneCenterLat, finds[0].SearchZoneRadiusM)
	}
	if finds[0].PrerequisiteStampLevel == nil || *finds[0].PrerequisiteStampLevel != 1 {
		t.Errorf("bf-1 stamp = %v, want 1", finds[0].PrerequisiteStampLevel)
	}
	if finds[0].CarID != "bf-car-1" {
		t.Errorf("bf-1 car_id = %q, want bf-car-1", finds[0].CarID)
	}
	// bf-2 : coords présentes.
	if finds[1].SearchZoneCenterLat == nil || *finds[1].SearchZoneCenterLat != 35.6 {
		t.Errorf("bf-2 lat = %v, want 35.6", finds[1].SearchZoneCenterLat)
	}

	// Filtre region : seul bf-1 est en Kansai.
	region := "Kansai"
	kansai, total, err := st.ListBarnFinds(ctx, store.HiddenCarFilter{Game: "fh6", Region: &region, Limit: 50})
	if err != nil {
		t.Fatalf("ListBarnFinds region: %v", err)
	}
	if total != 1 || len(kansai) != 1 || kansai[0].ID != "bf-1" {
		t.Fatalf("filtre region: total=%d len=%d id=%v, want 1/1/bf-1", total, len(kansai), kansai)
	}
}

// TestListTreasureCars vérifie le filtre par game/region et le mapping (clue,
// coords nullable).
func TestListTreasureCars(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	mustExec(t, st, `INSERT INTO cars (id, game, name, make, class, pi, drivetrain) VALUES
		('tc-car-1','fh6','Honda NSX','Honda','S1',850,'RWD'),
		('tc-car-2','fh6','Nissan GT-R','Nissan','S1',880,'AWD')`)
	mustExec(t, st, `INSERT INTO treasure_cars
		(id, car_id, game, region, postcard_clue_text, location_lat, location_lng, source) VALUES
		('tc-1','tc-car-1','fh6','Kanto','Near the great torii gate',35.1,139.2,'mitchcactus'),
		('tc-2','tc-car-2','fh6','Kansai',NULL,NULL,NULL,'fandom'),
		('tc-fh5','tc-car-1','fh5','Mexico','By the volcano',NULL,NULL,'fandom')`)

	cars, total, err := st.ListTreasureCars(ctx, store.HiddenCarFilter{Game: "fh6", Limit: 50})
	if err != nil {
		t.Fatalf("ListTreasureCars: %v", err)
	}
	if total != 2 || len(cars) != 2 {
		t.Fatalf("fh6: total=%d len=%d, want 2/2 (fh5 exclu)", total, len(cars))
	}
	// ORDER BY region : Kansai (tc-2) avant Kanto (tc-1).
	if cars[0].ID != "tc-2" || cars[1].ID != "tc-1" {
		t.Errorf("ordre = [%s, %s], want [tc-2, tc-1]", cars[0].ID, cars[1].ID)
	}
	// tc-2 : clue + coords NULL → nil.
	if cars[0].PostcardClueText != nil || cars[0].LocationLat != nil {
		t.Errorf("tc-2 clue/lat = %v/%v, want nil", cars[0].PostcardClueText, cars[0].LocationLat)
	}
	// tc-1 : clue présente.
	if cars[1].PostcardClueText == nil || *cars[1].PostcardClueText != "Near the great torii gate" {
		t.Errorf("tc-1 clue = %v, want 'Near the great torii gate'", cars[1].PostcardClueText)
	}

	// Filtre region.
	region := "Kanto"
	kanto, total, err := st.ListTreasureCars(ctx, store.HiddenCarFilter{Game: "fh6", Region: &region, Limit: 50})
	if err != nil {
		t.Fatalf("ListTreasureCars region: %v", err)
	}
	if total != 1 || len(kanto) != 1 || kanto[0].ID != "tc-1" {
		t.Fatalf("filtre region: total=%d len=%d id=%v, want 1/1/tc-1", total, len(kanto), kanto)
	}
}
