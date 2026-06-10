// Tests unitaires des middlewares CORS et SecurityHeaders, sans dépendance
// (pas de Docker) : on enveloppe un handler témoin et on inspecte les en-têtes.
// Couvre un fetch cross-origin réel et un preflight OPTIONS, lecture vs write.
package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zinackes/forza-open-api/internal/handler"
)

// okHandler répond 200 et signale qu'il a bien été atteint (aval non court-circuité).
func okHandler(reached *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*reached = true
		w.WriteHeader(http.StatusOK)
	})
}

// TestCORSActualRequest : un fetch GET cross-origin reçoit Allow-Origin et
// Expose-Headers, et l'aval est exécuté (la réponse réelle passe).
func TestCORSActualRequest(t *testing.T) {
	t.Run("wildcard lecture publique", func(t *testing.T) {
		var reached bool
		h := handler.NewCORS([]string{"*"}, nil).Middleware(okHandler(&reached))

		req := httptest.NewRequest(http.MethodGet, "/v1/cars?game=fh6", nil)
		req.Header.Set("Origin", "https://overlay.example")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if !reached {
			t.Fatal("aval non atteint : la requête réelle doit passer")
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
			t.Errorf("Allow-Origin = %q, attendu *", got)
		}
		if exp := rec.Header().Get("Access-Control-Expose-Headers"); !strings.Contains(exp, "X-RateLimit-Remaining") || !strings.Contains(exp, "ETag") {
			t.Errorf("Expose-Headers = %q, doit exposer le quota et l'ETag", exp)
		}
		// Joker → la réponse ne dépend pas de l'origine, pas de Vary: Origin.
		if v := rec.Header().Get("Vary"); strings.Contains(v, "Origin") {
			t.Errorf("Vary = %q, inattendu avec le joker", v)
		}
	})

	t.Run("liste d'origines : écho + Vary", func(t *testing.T) {
		var reached bool
		allowed := "https://app.forza.example"
		h := handler.NewCORS([]string{allowed}, nil).Middleware(okHandler(&reached))

		req := httptest.NewRequest(http.MethodGet, "/v1/cars?game=fh6", nil)
		req.Header.Set("Origin", allowed)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != allowed {
			t.Errorf("Allow-Origin = %q, attendu l'écho %q", got, allowed)
		}
		if v := rec.Header().Get("Vary"); !strings.Contains(v, "Origin") {
			t.Errorf("Vary = %q, doit contenir Origin (réponse dépendante de l'origine)", v)
		}
	})

	t.Run("origine non autorisée : pas d'en-tête CORS", func(t *testing.T) {
		var reached bool
		h := handler.NewCORS([]string{"https://app.forza.example"}, nil).Middleware(okHandler(&reached))

		req := httptest.NewRequest(http.MethodGet, "/v1/cars?game=fh6", nil)
		req.Header.Set("Origin", "https://evil.example")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		// L'aval s'exécute (CORS n'est pas une autorisation serveur), mais sans
		// Allow-Origin le navigateur masque la réponse au JS.
		if !reached {
			t.Fatal("aval non atteint")
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("Allow-Origin = %q, attendu absent pour une origine non listée", got)
		}
	})
}

// TestCORSNoOrigin : sans en-tête Origin (curl, appel serveur), le middleware est
// transparent — aucun en-tête CORS, aval exécuté.
func TestCORSNoOrigin(t *testing.T) {
	var reached bool
	h := handler.NewCORS([]string{"*"}, nil).Middleware(okHandler(&reached))

	req := httptest.NewRequest(http.MethodGet, "/v1/cars?game=fh6", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if !reached {
		t.Fatal("aval non atteint sans Origin")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Allow-Origin = %q, attendu absent hors contexte navigateur", got)
	}
}

// TestCORSPreflight : un preflight OPTIONS pour un GET est court-circuité (204),
// renvoie les en-têtes de négociation et n'atteint jamais l'aval.
func TestCORSPreflight(t *testing.T) {
	var reached bool
	h := handler.NewCORS([]string{"*"}, nil).Middleware(okHandler(&reached))

	req := httptest.NewRequest(http.MethodOptions, "/v1/cars?game=fh6", nil)
	req.Header.Set("Origin", "https://overlay.example")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	req.Header.Set("Access-Control-Request-Headers", "X-API-Key")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if reached {
		t.Fatal("le preflight ne doit pas atteindre l'aval (ni ogen ni quota)")
	}
	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, attendu 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Allow-Origin = %q, attendu *", got)
	}
	if m := rec.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(m, "GET") {
		t.Errorf("Allow-Methods = %q, doit contenir GET", m)
	}
	if hd := rec.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(hd, "X-API-Key") {
		t.Errorf("Allow-Headers = %q, doit autoriser X-API-Key", hd)
	}
	if ma := rec.Header().Get("Access-Control-Max-Age"); ma != "600" {
		t.Errorf("Max-Age = %q, attendu 600", ma)
	}
}

// TestCORSWritePolicy : les writes obéissent à un jeu d'origines distinct, fermé
// par défaut et ouvert seulement aux origines explicitement configurées.
func TestCORSWritePolicy(t *testing.T) {
	t.Run("write fermé par défaut", func(t *testing.T) {
		// Lecture large, aucune origine d'écriture : un preflight POST est refusé.
		h := handler.NewCORS([]string{"*"}, nil).Middleware(okHandler(new(bool)))

		req := httptest.NewRequest(http.MethodOptions, "/v1/tunes", nil)
		req.Header.Set("Origin", "https://overlay.example")
		req.Header.Set("Access-Control-Request-Method", http.MethodPost)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Errorf("status = %d, attendu 204", rec.Code)
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("Allow-Origin = %q, attendu absent (write fermé)", got)
		}
	})

	t.Run("write autorisé pour une origine configurée", func(t *testing.T) {
		trusted := "https://studio.forza.example"
		h := handler.NewCORS([]string{"*"}, []string{trusted}).Middleware(okHandler(new(bool)))

		req := httptest.NewRequest(http.MethodOptions, "/v1/tunes", nil)
		req.Header.Set("Origin", trusted)
		req.Header.Set("Access-Control-Request-Method", http.MethodPost)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != trusted {
			t.Errorf("Allow-Origin = %q, attendu %q", got, trusted)
		}
		if m := rec.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(m, "POST") {
			t.Errorf("Allow-Methods = %q, doit contenir POST", m)
		}
	})

	t.Run("origine de lecture seule ne débloque pas les writes", func(t *testing.T) {
		// L'origine est autorisée en lecture (joker) mais pas dans le set write.
		h := handler.NewCORS([]string{"*"}, []string{"https://studio.forza.example"}).Middleware(okHandler(new(bool)))

		req := httptest.NewRequest(http.MethodOptions, "/v1/tunes", nil)
		req.Header.Set("Origin", "https://overlay.example")
		req.Header.Set("Access-Control-Request-Method", http.MethodPost)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("Allow-Origin = %q, attendu absent (origine hors set write)", got)
		}
	})
}

// TestSecurityHeaders : les en-têtes de sécurité de base sont posés sur la réponse.
func TestSecurityHeaders(t *testing.T) {
	var reached bool
	h := handler.SecurityHeaders(okHandler(&reached))

	req := httptest.NewRequest(http.MethodGet, "/v1/cars?game=fh6", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if !reached {
		t.Fatal("aval non atteint")
	}
	want := map[string]string{
		"X-Content-Type-Options":       "nosniff",
		"X-Frame-Options":              "DENY",
		"Referrer-Policy":              "no-referrer",
		"Cross-Origin-Resource-Policy": "cross-origin",
	}
	for k, v := range want {
		if got := rec.Header().Get(k); got != v {
			t.Errorf("%s = %q, attendu %q", k, got, v)
		}
	}
}
