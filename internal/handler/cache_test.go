// Tests du middleware de cache conditionnel (ConditionalGet) : ETag fort sur les
// GET cacheables, court-circuit 304 sur If-None-Match correspondant, respect de
// no-store et des non-GET, sensibilité à dataVersion. Pas de dépendance externe
// (handler aval factice + httptest).
package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zinackes/forza-open-api/internal/handler"
)

// downstream renvoie un handler qui pose cacheControl (si non vide) puis écrit body
// en 200 — imite un handler ogen cacheable.
func downstream(cacheControl, body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if cacheControl != "" {
			w.Header().Set("Cache-Control", cacheControl)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	})
}

func TestConditionalGet(t *testing.T) {
	const cc = "public, max-age=3600"
	mw := handler.ConditionalGet("")

	// Premier GET : ETag posé, corps servi.
	srv := mw(downstream(cc, `{"items":[]}`))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/cars?game=fh6", nil))

	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Fatal("ETag absent sur GET 200 cacheable")
	}
	if rec.Code != http.StatusOK || rec.Body.Len() == 0 {
		t.Fatalf("attendu 200 + corps, got %d len=%d", rec.Code, rec.Body.Len())
	}
	if got := rec.Header().Get("Cache-Control"); got != cc {
		t.Fatalf("Cache-Control = %q, want %q", got, cc)
	}

	t.Run("if_none_match_matches_304", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/cars?game=fh6", nil)
		req.Header.Set("If-None-Match", etag)
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotModified {
			t.Fatalf("status = %d, want 304", rec.Code)
		}
		if rec.Body.Len() != 0 {
			t.Fatalf("304 ne doit pas avoir de corps, got %q", rec.Body.String())
		}
		if rec.Header().Get("ETag") != etag {
			t.Errorf("304 doit conserver l'ETag")
		}
		if rec.Header().Get("Cache-Control") != cc {
			t.Errorf("304 doit conserver Cache-Control")
		}
	})

	t.Run("if_none_match_star_304", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/cars?game=fh6", nil)
		req.Header.Set("If-None-Match", "*")
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotModified {
			t.Fatalf("If-None-Match: * doit donner 304, got %d", rec.Code)
		}
	})

	t.Run("if_none_match_mismatch_200", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/cars?game=fh6", nil)
		req.Header.Set("If-None-Match", `"stale"`)
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK || rec.Body.Len() == 0 {
			t.Fatalf("ETag obsolète doit donner 200 + corps, got %d len=%d", rec.Code, rec.Body.Len())
		}
	})
}

func TestConditionalGetNoStore(t *testing.T) {
	mw := handler.ConditionalGet("")
	srv := mw(downstream("no-store", `{"id":"random"}`))

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/cars/random?game=fh6", nil))

	if rec.Header().Get("ETag") != "" {
		t.Error("no-store ne doit jamais recevoir d'ETag")
	}
	// Même avec un If-None-Match, no-store ne court-circuite pas en 304.
	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/cars/random?game=fh6", nil)
	req.Header.Set("If-None-Match", "*")
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("no-store doit rester 200 même avec If-None-Match, got %d", rec.Code)
	}
}

func TestConditionalGetNonGET(t *testing.T) {
	mw := handler.ConditionalGet("")
	srv := mw(downstream("public, max-age=3600", `{}`))

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/anything", nil))
	if rec.Header().Get("ETag") != "" {
		t.Error("les non-GET ne doivent pas recevoir d'ETag")
	}
}

// TestConditionalGetDataVersion : à corps identique, un bump de dataVersion change
// l'ETag (invalidation au refresh du catalogue).
func TestConditionalGetDataVersion(t *testing.T) {
	const body = `{"items":[1,2,3]}`
	etag := func(version string) string {
		srv := handler.ConditionalGet(version)(downstream("public, max-age=3600", body))
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/cars", nil))
		return rec.Header().Get("ETag")
	}
	if etag("2026-01-01") == etag("2026-02-01") {
		t.Error("un bump de dataVersion doit changer l'ETag à corps constant")
	}
}
