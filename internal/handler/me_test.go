// Tests de GET /v1/me (endpoint authentifié : identité + quota de la clé).
//   - clé valide → 200, name/scopes/rateLimit/createdAt renseignés, no-store ;
//   - sans clé / clé inconnue → 401 application/problem+json ;
//   - sans middleware de rate-limit (Redis hors-jeu) → remaining/resetAt omis ;
//   - avec middleware + Redis → remaining/resetAt présents et cohérents avec les
//     en-têtes X-RateLimit-*.
//
// Réutilise newAuthStore/mustKey (auth_test.go) et newRedisClient (ratelimit_test.go).
// Nécessite Docker (cf. .claude/rules/testing.md).
package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/zinackes/forza-open-api/internal/apikey"
	"github.com/zinackes/forza-open-api/internal/handler"
	"github.com/zinackes/forza-open-api/internal/oas"
)

// meBody décode le corps JSON de /v1/me. remaining/resetAt sont des pointeurs
// pour distinguer « absent » (champ omis) de « zéro ».
type meBody struct {
	Name      string   `json:"name"`
	Scopes    []string `json:"scopes"`
	RateLimit int      `json:"rateLimit"`
	Remaining *int     `json:"remaining"`
	ResetAt   *string  `json:"resetAt"`
	CreatedAt string   `json:"createdAt"`
}

// TestGetMe vérifie l'endpoint à travers le serveur ogen (sécurité requise), sans
// middleware de rate-limit : remaining/resetAt doivent donc être omis.
func TestGetMe(t *testing.T) {
	st := newAuthStore(t)
	ctx := context.Background()

	// Clé avec scopes explicites (round-trip read + submit-ugc) et quota 1000.
	plain, err := apikey.Generate()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	hash := apikey.Hash(plain)
	if err := st.CreateAPIKey(ctx, hash, "mon-bot-discord", 1000, []string{"read", "submit-ugc"}); err != nil {
		t.Fatalf("create: %v", err)
	}

	srv, err := oas.NewServer(handler.New(st, ""), handler.NewSecurityHandler(st),
		oas.WithErrorHandler(handler.ProblemErrorHandler))
	if err != nil {
		t.Fatalf("oas.NewServer: %v", err)
	}

	t.Run("clé valide", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
		req.Header.Set("X-API-Key", plain)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200\nbody: %s", rec.Code, rec.Body.String())
		}
		if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
			t.Errorf("Cache-Control = %q, want no-store", cc)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}

		var body meBody
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body.Name != "mon-bot-discord" {
			t.Errorf("name = %q, want mon-bot-discord", body.Name)
		}
		if want := []string{"read", "submit-ugc"}; !equalStrings(body.Scopes, want) {
			t.Errorf("scopes = %v, want %v", body.Scopes, want)
		}
		if body.RateLimit != 1000 {
			t.Errorf("rateLimit = %d, want 1000", body.RateLimit)
		}
		if body.CreatedAt == "" {
			t.Error("createdAt absent")
		}
		// Sans middleware de rate-limit, le compteur n'a pas tourné → champs omis.
		if body.Remaining != nil {
			t.Errorf("remaining = %v, want omis (pas de middleware)", *body.Remaining)
		}
		if body.ResetAt != nil {
			t.Errorf("resetAt = %v, want omis (pas de middleware)", *body.ResetAt)
		}
	})

	t.Run("sans clé → 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401\nbody: %s", rec.Code, rec.Body.String())
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
			t.Errorf("Content-Type = %q, want application/problem+json", ct)
		}
	})

	t.Run("clé inconnue → 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
		req.Header.Set("X-API-Key", "totally-unknown-key")
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401\nbody: %s", rec.Code, rec.Body.String())
		}
	})
}

// TestGetMeRateLimitState vérifie que remaining/resetAt sont renseignés quand la
// requête traverse le middleware de rate-limit (Redis), et qu'ils sont cohérents
// avec les en-têtes X-RateLimit-* posés sur la même réponse.
func TestGetMeRateLimitState(t *testing.T) {
	st := newAuthStore(t)
	st.Redis = newRedisClient(t)

	const limit = 5
	plain := mustKey(t, st, "rl-me", limit, false)

	sec := handler.NewSecurityHandler(st)
	oasSrv, err := oas.NewServer(handler.New(st, ""), sec, oas.WithErrorHandler(handler.ProblemErrorHandler))
	if err != nil {
		t.Fatalf("oas.NewServer: %v", err)
	}
	h := handler.NewRateLimiter(sec, time.Minute).Middleware(oasSrv)

	req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	req.Header.Set("X-API-Key", plain)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200\nbody: %s", rec.Code, rec.Body.String())
	}

	var body meBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Première requête comptée → remaining = limit-1, et le corps doit égaler
	// l'en-tête X-RateLimit-Remaining (même RateLimitResult).
	if body.Remaining == nil {
		t.Fatal("remaining absent alors que le middleware a tourné")
	}
	if *body.Remaining != limit-1 {
		t.Errorf("remaining = %d, want %d", *body.Remaining, limit-1)
	}
	if hdr := rec.Header().Get("X-RateLimit-Remaining"); hdr != strconv.Itoa(*body.Remaining) {
		t.Errorf("remaining corps = %d, en-tête X-RateLimit-Remaining = %q (doivent coïncider)", *body.Remaining, hdr)
	}
	if body.ResetAt == nil {
		t.Error("resetAt absent alors que le middleware a tourné")
	}
}

// equalStrings compare deux slices de chaînes (ordre significatif).
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
