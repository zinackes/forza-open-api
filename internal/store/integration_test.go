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
