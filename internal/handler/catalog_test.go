// Tests golden du catalogue côté handler : serveur ogen branché sur un Postgres
// jetable (testcontainers-go), schéma db/init.sql, catalogue fixe semé une fois.
// Chaque cas table-driven envoie une requête HTTP et compare la réponse à un
// fichier .golden (regénérable via `go test ./internal/handler -run TestCatalogGolden -update`).
// Couvre listCars (filtres, pagination, page hors borne, résultat vide, isolation
// par jeu), getCar (trouvé + 404) et manufacturers.
//
// Nécessite Docker (cf. .claude/rules/testing.md). En CI, Docker est présent.
package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/zinackes/forza-open-api/internal/handler"
	"github.com/zinackes/forza-open-api/internal/oas"
	"github.com/zinackes/forza-open-api/internal/store"
)

var update = flag.Bool("update", false, "met à jour les fichiers .golden")

// timestampRE neutralise les champs timestamp avant la comparaison golden :
// l'instant exact — et surtout son fuseau de sérialisation, qui dépend de la
// machine (pgx scanne les timestamptz en time.Local) — ne fait pas partie du
// contrat qu'on fige ici. On les remplace par un marqueur stable.
var timestampRE = regexp.MustCompile(`"(createdAt|updatedAt|releasedAt|lastVerified|occurredAt)":"[^"]*"`)

// newSeededCatalogServer lève un Postgres jetable, applique db/init.sql, sème un
// catalogue fixe, et renvoie le serveur ogen branché dessus (Redis nil : les
// lectures testées n'y touchent pas). Un seul conteneur pour tous les sous-tests.
func newSeededCatalogServer(t *testing.T) http.Handler {
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

	seedCatalog(ctx, t, pool)

	srv, err := oas.NewServer(handler.New(&store.Store{DB: pool}, ""), handler.SecurityHandler{},
		oas.WithErrorHandler(handler.ProblemErrorHandler))
	if err != nil {
		t.Fatalf("oas.NewServer: %v", err)
	}
	return srv
}

// seedCatalog insère un catalogue déterministe (created_at explicite) : 5 voitures
// fh6 couvrant le tri ORDER BY pi, name, id, une fh5 pour l'isolation, et des
// constructeurs dont deux sans voiture (car_count 0) et un sans pays (NULL → nil).
func seedCatalog(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	const cars = `INSERT INTO cars
		(id, game, name, make, model, year, class, pi, drivetrain, body_type, category, obtain_method, created_at) VALUES
		('audi-r8','fh6','R8','Audi',NULL,NULL,'S1',850,'AWD','coupe','Modern Supercars','Autoshow, Wheelspin','2026-01-01T00:00:00Z'),
		('ford-gt','fh6','GT','Ford',NULL,NULL,'S2',920,'RWD','coupe','Modern Supercars',NULL,'2026-01-01T00:00:00Z'),
		('honda-civic','fh6','Civic Type R','Honda',NULL,NULL,'A',780,'FWD','hatchback','Hot Hatch','Autoshow','2026-01-01T00:00:00Z'),
		('mazda-rx7','fh6','RX-7','Mazda','FD',1998,'A',780,'RWD','coupe',NULL,NULL,'2026-01-01T00:00:00Z'),
		('toyota-ae86','fh6','AE86','Toyota',NULL,NULL,'B',600,'RWD','coupe',NULL,'Wristband reward','2026-01-01T00:00:00Z'),
		('vw-beetle','fh5','Beetle','Volkswagen',NULL,NULL,'D',400,'RWD','hatchback',NULL,NULL,'2026-01-01T00:00:00Z')`
	const manufacturers = `INSERT INTO manufacturers (game, name, country) VALUES
		('fh6','Audi','Germany'),
		('fh6','Ford','USA'),
		('fh6','Honda','Japan'),
		('fh6','Mazda','Japan'),
		('fh6','Toyota','Japan'),
		('fh6','Koenigsegg','Sweden'),
		('fh6','Mystery',NULL),
		('fh5','Volkswagen','Germany')`
	for _, q := range []string{cars, manufacturers} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatalf("seed catalogue: %v", err)
		}
	}
}

// TestCatalogGolden compare la réponse HTTP de chaque cas à son fichier golden.
func TestCatalogGolden(t *testing.T) {
	srv := newSeededCatalogServer(t)

	// carsCC : Cache-Control attendu sur les 200 cacheables du catalogue (cars
	// list / getCar / compare). Posé par les handlers via le wrapper *Headers du
	// contrat. Valeur figée ici comme attendu golden des en-têtes.
	const carsCC = "public, max-age=3600, s-maxage=86400, stale-while-revalidate=86400"

	cases := []struct {
		name   string
		target string
		status int
		cache  string // Cache-Control attendu (vide = non vérifié)
	}{
		// listCars : liste complète triée, filtres, pagination, cas limites.
		{"cars_all", "/v1/cars?game=fh6", http.StatusOK, carsCC},
		{"cars_make", "/v1/cars?game=fh6&make=Honda", http.StatusOK, carsCC},
		{"cars_class_a", "/v1/cars?game=fh6&class=A", http.StatusOK, carsCC},
		{"cars_page2", "/v1/cars?game=fh6&page_size=2&page=2", http.StatusOK, carsCC},
		{"cars_page_out_of_bounds", "/v1/cars?game=fh6&page=99", http.StatusOK, carsCC},
		{"cars_empty_result", "/v1/cars?game=fh6&make=Bugatti", http.StatusOK, carsCC},
		{"cars_game_fh5", "/v1/cars?game=fh5", http.StatusOK, carsCC},
		// obtain : match par token d'obtain_method multi-valeurs ; valeur hors
		// enum rejetée en 400 par ogen.
		{"cars_obtain_wheelspin", "/v1/cars?game=fh6&obtain=wheelspin", http.StatusOK, carsCC},
		{"cars_obtain_invalid", "/v1/cars?game=fh6&obtain=bogus", http.StatusBadRequest, ""},
		// getCar : trouvé puis 404 RFC 9457.
		{"getcar_found", "/v1/cars/honda-civic", http.StatusOK, carsCC},
		{"getcar_not_found", "/v1/cars/ghost", http.StatusNotFound, ""},
		// compareCars : 2 puis 3 voitures alignées sur l'ordre des ids ; bornes
		// 2..3 (ogen → 400) ; id inconnu → 404 (comparaison stricte).
		{"compare_two", "/v1/cars/compare?ids=ford-gt,audi-r8", http.StatusOK, carsCC},
		{"compare_three", "/v1/cars/compare?ids=audi-r8,ford-gt,honda-civic", http.StatusOK, carsCC},
		{"compare_too_few", "/v1/cars/compare?ids=audi-r8", http.StatusBadRequest, ""},
		{"compare_too_many", "/v1/cars/compare?ids=audi-r8,ford-gt,honda-civic,mazda-rx7", http.StatusBadRequest, ""},
		{"compare_missing_id", "/v1/cars/compare?ids=audi-r8,ghost", http.StatusNotFound, ""},
		// manufacturers.
		{"manufacturers", "/v1/manufacturers?game=fh6", http.StatusOK, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.target, nil)

			srv.ServeHTTP(rec, req)

			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, tc.status, rec.Body.String())
			}
			if tc.cache != "" {
				if got := rec.Header().Get("Cache-Control"); got != tc.cache {
					t.Errorf("Cache-Control = %q, want %q", got, tc.cache)
				}
			}
			assertGolden(t, tc.name, rec.Body.Bytes())
		})
	}
}

// assertGolden compare la réponse (normalisée) au fichier testdata/<name>.golden.json.
// Avec -update, (ré)écrit le fichier au lieu de comparer.
func assertGolden(t *testing.T, name string, raw []byte) {
	t.Helper()
	got := normalizeJSON(t, raw)
	path := filepath.Join("testdata", name+".golden.json")

	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatalf("mkdir testdata: %v", err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("écriture golden: %v", err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("lecture golden %s (lancer `go test -run TestCatalogGolden -update` ?): %v", path, err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("réponse != golden %s\n--- got ---\n%s\n--- want ---\n%s", path, got, want)
	}
}

// normalizeJSON neutralise createdAt puis ré-indente : golden lisibles et diffs
// stables, indépendants du fuseau et de l'instant de création.
func normalizeJSON(t *testing.T, raw []byte) []byte {
	t.Helper()
	b := timestampRE.ReplaceAll(raw, []byte(`"$1":"<ts>"`))
	var buf bytes.Buffer
	if err := json.Indent(&buf, b, "", "  "); err != nil {
		t.Fatalf("indent json: %v\nbody: %s", err, b)
	}
	buf.WriteByte('\n')
	return buf.Bytes()
}
