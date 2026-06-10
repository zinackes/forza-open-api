// Tests internes (package handler) du rendu d'erreur centralisé. Ils prouvent,
// sans dépendance d'infra (DB/Redis), que 400, 401, 404 et 429 produisent
// strictement le même corps RFC 9457 (type, title, status, detail, instance),
// via l'unique écrivain writeProblem. Les chemins bout-en-bout (handler → 404,
// sécurité → 401, middleware → 429) sont couverts par les tests testcontainers.
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zinackes/forza-open-api/internal/store"
)

// decodeProblem décode un corps application/problem+json en problem.
func decodeProblem(t *testing.T, raw []byte) problem {
	t.Helper()
	var p problem
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatalf("décodage problem: %v\nbody: %s", err, raw)
	}
	return p
}

// TestProblemType vérifie le mapping statut → code stable (URN), documenté au
// contrat. Un statut hors liste retombe sur about:blank (le statut HTTP fait
// alors office de code, conforme RFC 9457).
func TestProblemType(t *testing.T) {
	cases := map[int]string{
		http.StatusBadRequest:          "urn:forza-open-api:problem:validation",
		http.StatusUnauthorized:        "urn:forza-open-api:problem:unauthorized",
		http.StatusNotFound:            "urn:forza-open-api:problem:not-found",
		http.StatusTooManyRequests:     "urn:forza-open-api:problem:rate-limited",
		http.StatusInternalServerError: "urn:forza-open-api:problem:internal",
		http.StatusTeapot:              "about:blank",
	}
	for status, want := range cases {
		if got := problemType(status); got != want {
			t.Errorf("problemType(%d) = %q, want %q", status, got, want)
		}
	}
}

// TestWriteProblemUnifiedFormat est le cœur de la garantie « même format » : pour
// chaque code (400/401/404/429), writeProblem doit poser le Content-Type, le
// statut et les 5 champs, l'instance valant le chemin de la requête.
func TestWriteProblemUnifiedFormat(t *testing.T) {
	const path = "/v1/cars/ghost"
	for _, status := range []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusNotFound,
		http.StatusTooManyRequests,
	} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path+"?secret=leak", nil)

			writeProblem(rec, req, status, "boom")

			if rec.Code != status {
				t.Fatalf("status = %d, want %d", rec.Code, status)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
				t.Errorf("Content-Type = %q, want application/problem+json", ct)
			}
			p := decodeProblem(t, rec.Body.Bytes())
			if p.Type != problemType(status) {
				t.Errorf("type = %q, want %q", p.Type, problemType(status))
			}
			if p.Title != http.StatusText(status) {
				t.Errorf("title = %q, want %q", p.Title, http.StatusText(status))
			}
			if p.Status != status {
				t.Errorf("status field = %d, want %d", p.Status, status)
			}
			if p.Detail != "boom" {
				t.Errorf("detail = %q, want %q", p.Detail, "boom")
			}
			// instance = chemin seul, sans la query (on ne reflète pas ?secret=leak).
			if p.Instance != path {
				t.Errorf("instance = %q, want %q", p.Instance, path)
			}
		})
	}
}

// TestProblemErrorHandlerNotFound vérifie le chemin handler : une erreur
// errNotFound est traduite en 404 RFC 9457 avec l'URN not-found et l'instance.
func TestProblemErrorHandlerNotFound(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/cars/ghost", nil)

	ProblemErrorHandler(req.Context(), rec, req, errNotFound("no car with the given id"))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404\nbody: %s", rec.Code, rec.Body.String())
	}
	p := decodeProblem(t, rec.Body.Bytes())
	if p.Type != "urn:forza-open-api:problem:not-found" {
		t.Errorf("type = %q", p.Type)
	}
	if p.Detail != "no car with the given id" {
		t.Errorf("detail = %q", p.Detail)
	}
	if p.Instance != "/v1/cars/ghost" {
		t.Errorf("instance = %q", p.Instance)
	}
}

// TestProblemErrorHandlerInternalHidesDetail vérifie qu'une erreur quelconque
// (non typée) devient un 500 générique : le détail réel n'est jamais exposé au
// client (pas de fuite de requête SQL / hôte), seul "internal error" sort.
func TestProblemErrorHandlerInternalHidesDetail(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/cars", nil)

	ProblemErrorHandler(req.Context(), rec, req, errors.New("pq: relation \"cars\" does not exist"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	p := decodeProblem(t, rec.Body.Bytes())
	if p.Detail != "internal error" {
		t.Errorf("detail = %q, want %q (pas de fuite interne)", p.Detail, "internal error")
	}
	if p.Type != "urn:forza-open-api:problem:internal" {
		t.Errorf("type = %q", p.Type)
	}
}

// TestWriteRateLimited vérifie le 429 du middleware : même corps unifié que les
// autres codes, plus l'en-tête Retry-After (delta en secondes, >= 1).
func TestWriteRateLimited(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/cars?game=fh6", nil)

	writeRateLimited(rec, req, store.RateLimitResult{ResetAfter: 30 * time.Second})

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rec.Code)
	}
	if ra := rec.Header().Get("Retry-After"); ra != "30" {
		t.Errorf("Retry-After = %q, want 30", ra)
	}
	p := decodeProblem(t, rec.Body.Bytes())
	if p.Type != "urn:forza-open-api:problem:rate-limited" {
		t.Errorf("type = %q", p.Type)
	}
	if p.Status != http.StatusTooManyRequests {
		t.Errorf("status field = %d, want 429", p.Status)
	}
	if p.Instance != "/v1/cars" {
		t.Errorf("instance = %q, want /v1/cars", p.Instance)
	}
}
