// Test bout-en-bout du middleware de rate-limit, à travers le serveur ogen :
// clé valide → boucle de requêtes jusqu'au 429 typé, en-têtes de quota (X-* et
// IETF) vérifiés ; requête anonyme → jamais limitée. Postgres jetable pour la
// clé (réutilise newAuthStore/mustKey de auth_test.go) + Redis jetable pour le
// compteur.
//
// Nécessite Docker (cf. .claude/rules/testing.md). En CI, Docker est présent.
package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/zinackes/forza-open-api/internal/handler"
	"github.com/zinackes/forza-open-api/internal/oas"
)

// newRedisClient lève un Redis jetable et renvoie un client branché dessus.
func newRedisClient(t *testing.T) *redis.Client {
	t.Helper()
	ctx := context.Background()

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:7-alpine",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForListeningPort("6379/tcp").WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("démarrage Redis (Docker requis): %v", err)
	}
	t.Cleanup(func() { _ = c.Terminate(context.Background()) })

	host, err := c.Host(ctx)
	if err != nil {
		t.Fatalf("host: %v", err)
	}
	port, err := c.MappedPort(ctx, "6379/tcp")
	if err != nil {
		t.Fatalf("port: %v", err)
	}

	client := redis.NewClient(&redis.Options{Addr: host + ":" + port.Port()})
	t.Cleanup(func() { _ = client.Close() })
	return client
}

// TestRateLimitMiddleware vérifie le quota par clé bout-en-bout sur
// /v1/playlist/series : tant qu'on est sous la limite la réponse n'est PAS un 429
// et porte les en-têtes de quota ; au dépassement, le middleware court-circuite en
// 429 typé (avant d'atteindre le handler).
func TestRateLimitMiddleware(t *testing.T) {
	st := newAuthStore(t)
	st.Redis = newRedisClient(t)

	const limit = 3
	key := mustKey(t, st, "rl", limit, false)

	sec := handler.NewSecurityHandler(st)
	oasSrv, err := oas.NewServer(handler.New(st, ""), sec, oas.WithErrorHandler(handler.ProblemErrorHandler))
	if err != nil {
		t.Fatalf("oas.NewServer: %v", err)
	}
	// Quota anonyme désactivé (0) : ce test couvre le chemin par clé et le
	// passthrough anonyme historique ; le quota par IP a son propre test.
	h := handler.NewRateLimiter(sec, time.Minute, 0).Middleware(oasSrv)

	do := func(withKey bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/v1/playlist/series?game=fh6", nil)
		if withKey {
			req.Header.Set("X-API-Key", key)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	// limit requêtes sous le quota : pas de 429, en-têtes posés, Remaining décroît.
	for i := 1; i <= limit; i++ {
		rec := do(true)
		if rec.Code == http.StatusTooManyRequests {
			t.Fatalf("req %d: 429 prématuré (limite pas atteinte)\nbody: %s", i, rec.Body.String())
		}
		if got := rec.Header().Get("X-RateLimit-Limit"); got != strconv.Itoa(limit) {
			t.Errorf("req %d: X-RateLimit-Limit = %q, attendu %d", i, got, limit)
		}
		if got, want := rec.Header().Get("X-RateLimit-Remaining"), strconv.Itoa(limit-i); got != want {
			t.Errorf("req %d: X-RateLimit-Remaining = %q, attendu %s", i, got, want)
		}
	}

	// Dépassement → 429 RFC 9457 + Retry-After + en-têtes de quota à zéro.
	rec := do(true)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("dépassement: status = %d, attendu 429\nbody: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, attendu application/problem+json", ct)
	}
	if ra := rec.Header().Get("Retry-After"); ra == "" {
		t.Error("Retry-After absent du 429")
	}
	if got := rec.Header().Get("X-RateLimit-Remaining"); got != "0" {
		t.Errorf("X-RateLimit-Remaining = %q, attendu 0", got)
	}
	// En-têtes IETF (draft-11) : RateLimit r=0, RateLimit-Policy q=limit.
	if rl := rec.Header().Get("RateLimit"); !strings.Contains(rl, "r=0") {
		t.Errorf("RateLimit = %q, attendu r=0", rl)
	}
	if rp := rec.Header().Get("RateLimit-Policy"); !strings.Contains(rp, "q="+strconv.Itoa(limit)) {
		t.Errorf("RateLimit-Policy = %q, attendu q=%d", rp, limit)
	}

	// Anonyme (sans clé) : quota par IP désactivé (0) → jamais limité.
	for i := 0; i < limit+2; i++ {
		if rec := do(false); rec.Code == http.StatusTooManyRequests {
			t.Fatalf("anonyme req %d: 429 inattendu (quota anonyme désactivé)", i)
		}
	}
}

// TestRateLimitAnonymousPerIP vérifie le quota par IP du trafic sans clé :
// chaque IP a son seau (même fenêtre glissante Redis que les clés), le
// dépassement rend un 429 typé, une autre IP n'est pas affectée, et une requête
// AVEC clé reste sur le quota de la clé (seaux disjoints).
func TestRateLimitAnonymousPerIP(t *testing.T) {
	st := newAuthStore(t)
	st.Redis = newRedisClient(t)

	const anonLimit = 3
	key := mustKey(t, st, "rl-anon", 100, false)

	sec := handler.NewSecurityHandler(st)
	oasSrv, err := oas.NewServer(handler.New(st, ""), sec, oas.WithErrorHandler(handler.ProblemErrorHandler))
	if err != nil {
		t.Fatalf("oas.NewServer: %v", err)
	}
	h := handler.NewRateLimiter(sec, time.Minute, anonLimit).Middleware(oasSrv)

	do := func(remoteAddr, apiKey string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/v1/playlist/series?game=fh6", nil)
		req.RemoteAddr = remoteAddr
		if apiKey != "" {
			req.Header.Set("X-API-Key", apiKey)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	// anonLimit requêtes anonymes sous le quota : en-têtes posés, pas de 429.
	for i := 1; i <= anonLimit; i++ {
		rec := do("203.0.113.7:4242", "")
		if rec.Code == http.StatusTooManyRequests {
			t.Fatalf("anonyme req %d: 429 prématuré\nbody: %s", i, rec.Body.String())
		}
		if got := rec.Header().Get("X-RateLimit-Limit"); got != strconv.Itoa(anonLimit) {
			t.Errorf("req %d: X-RateLimit-Limit = %q, attendu %d", i, got, anonLimit)
		}
	}

	// Dépassement → 429 RFC 9457 + Retry-After.
	rec := do("203.0.113.7:4242", "")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("dépassement anonyme: status = %d, attendu 429\nbody: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, attendu application/problem+json", ct)
	}
	if ra := rec.Header().Get("Retry-After"); ra == "" {
		t.Error("Retry-After absent du 429 anonyme")
	}

	// Une AUTRE IP a son propre seau : pas affectée par le 429 ci-dessus.
	if rec := do("203.0.113.99:4242", ""); rec.Code == http.StatusTooManyRequests {
		t.Fatalf("autre IP: 429 inattendu (seaux par IP disjoints)\nbody: %s", rec.Body.String())
	}

	// Une requête AVEC clé depuis l'IP saturée suit le quota de la clé, pas
	// celui de l'IP : elle passe.
	if rec := do("203.0.113.7:4242", key); rec.Code == http.StatusTooManyRequests {
		t.Fatalf("clé valide depuis IP saturée: 429 inattendu (seaux disjoints)\nbody: %s", rec.Body.String())
	}
}
