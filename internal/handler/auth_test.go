// Vérifie la validation X-API-Key du SecurityHandler sur un Postgres jetable
// (testcontainers, Redis nil → cache désactivé, Postgres fait foi) :
//   - clé valide   → AuthInfo (hash + quota) en context, sécurité passée (501 sur stub) ;
//   - clé inconnue → 401 application/problem+json ;
//   - clé révoquée → 401 application/problem+json ;
//   - sans clé     → lecture publique (la sécurité optionnelle du contrat n'entre
//     jamais dans le handler), donc 501 sur le stub, jamais 401.
//
// Nécessite Docker (cf. .claude/rules/testing.md). En CI, Docker est présent.
package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/zinackes/forza-open-api/internal/apikey"
	"github.com/zinackes/forza-open-api/internal/handler"
	"github.com/zinackes/forza-open-api/internal/oas"
	"github.com/zinackes/forza-open-api/internal/store"
)

// newAuthStore lève un Postgres jetable, applique db/init.sql, et renvoie un
// Store branché dessus (Redis nil : le SecurityHandler tombe alors directement
// sur Postgres, source de vérité).
func newAuthStore(t *testing.T) *store.Store {
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

// mustKey crée une clé API (active, ou révoquée si revoke=true) et renvoie sa
// forme en clair (celle qu'un client met dans X-API-Key).
func mustKey(t *testing.T, st *store.Store, name string, rateLimit int, revoke bool) string {
	t.Helper()
	ctx := context.Background()
	plain, err := apikey.Generate()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	hash := apikey.Hash(plain)
	if err := st.CreateAPIKey(ctx, hash, name, rateLimit, nil); err != nil {
		t.Fatalf("create %s: %v", name, err)
	}
	if revoke {
		if _, err := st.RevokeAPIKey(ctx, hash); err != nil {
			t.Fatalf("revoke %s: %v", name, err)
		}
	}
	return plain
}

// TestHandleApiKeyAuth teste la résolution au niveau du handler : une clé valide
// place AuthInfo (hash + quota) dans le context ; une clé inconnue ou révoquée
// renvoie une erreur (que ogen mappe en 401).
func TestHandleApiKeyAuth(t *testing.T) {
	st := newAuthStore(t)
	sh := handler.NewSecurityHandler(st)
	ctx := context.Background()

	validPlain := mustKey(t, st, "active", 4242, false)
	validHash := apikey.Hash(validPlain)
	revokedPlain := mustKey(t, st, "revoked", 0, true)
	unknownPlain, err := apikey.Generate()
	if err != nil {
		t.Fatalf("generate unknown: %v", err)
	}

	// Clé valide : pas d'erreur, AuthInfo (hash + quota) disponible en context.
	gotCtx, err := sh.HandleApiKeyAuth(ctx, oas.GetMetaOperation, oas.ApiKeyAuth{APIKey: validPlain})
	if err != nil {
		t.Fatalf("clé valide: err = %v, want nil", err)
	}
	info, ok := handler.AuthInfoFromContext(gotCtx)
	if !ok {
		t.Fatal("clé valide: AuthInfo absent du context")
	}
	if info.Hash != validHash {
		t.Errorf("AuthInfo.Hash = %q, want %q", info.Hash, validHash)
	}
	if info.RateLimit != 4242 {
		t.Errorf("AuthInfo.RateLimit = %d, want 4242", info.RateLimit)
	}

	// Clé inconnue → erreur (→ 401), pas d'AuthInfo.
	if c, err := sh.HandleApiKeyAuth(ctx, oas.GetMetaOperation, oas.ApiKeyAuth{APIKey: unknownPlain}); err == nil {
		t.Error("clé inconnue: err = nil, want erreur (401)")
	} else if _, ok := handler.AuthInfoFromContext(c); ok {
		t.Error("clé inconnue: AuthInfo ne devrait pas être en context")
	}

	// Clé révoquée → erreur (→ 401).
	if _, err := sh.HandleApiKeyAuth(ctx, oas.GetMetaOperation, oas.ApiKeyAuth{APIKey: revokedPlain}); err == nil {
		t.Error("clé révoquée: err = nil, want erreur (401)")
	}
}

// TestApiKeyAuthHTTP vérifie le comportement bout-en-bout à travers le serveur
// ogen, sur un endpoint encore en stub (/v1/playlist/series, 501 quand la
// sécurité passe). La sécurité s'exécute avant le handler : une clé invalide
// court-circuite en 401 application/problem+json ; sans clé ou avec une clé
// valide, on atteint le stub (501).
func TestApiKeyAuthHTTP(t *testing.T) {
	st := newAuthStore(t)
	validPlain := mustKey(t, st, "http-active", 0, false)
	revokedPlain := mustKey(t, st, "http-revoked", 0, true)

	srv, err := oas.NewServer(handler.New(st, ""), handler.NewSecurityHandler(st),
		oas.WithErrorHandler(handler.ProblemErrorHandler))
	if err != nil {
		t.Fatalf("oas.NewServer: %v", err)
	}

	cases := []struct {
		name string
		key  string
		want int
	}{
		{"sans clé (lecture publique)", "", http.StatusNotImplemented},
		{"clé valide", validPlain, http.StatusNotImplemented},
		{"clé inconnue", "totally-unknown-key", http.StatusUnauthorized},
		{"clé révoquée", revokedPlain, http.StatusUnauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/playlist/series?game=fh6", nil)
			if tc.key != "" {
				req.Header.Set("X-API-Key", tc.key)
			}
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)

			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, tc.want, rec.Body.String())
			}
			if tc.want == http.StatusUnauthorized {
				if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
					t.Errorf("Content-Type = %q, want application/problem+json", ct)
				}
			}
		})
	}
}
