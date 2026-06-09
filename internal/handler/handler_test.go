package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zinackes/forza-open-api/internal/handler"
	"github.com/zinackes/forza-open-api/internal/oas"
)

// newServer construit le serveur ogen comme cmd/api, mais avec un store nil :
// en Phase 0 les stubs renvoient 501 sans toucher aux dépendances de données.
func newServer(t *testing.T) http.Handler {
	t.Helper()
	srv, err := oas.NewServer(handler.New(nil), handler.SecurityHandler{},
		oas.WithErrorHandler(handler.ProblemErrorHandler))
	if err != nil {
		t.Fatalf("oas.NewServer: %v", err)
	}
	return srv
}

// TestListCarsReturns501 vérifie qu'un endpoint non encore implémenté répond
// 501 en RFC 9457 (application/problem+json). game=fh6 est requis pour passer
// la validation du contrat et atteindre le stub.
func TestListCarsReturns501(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/cars?game=fh6", nil)

	newServer(t).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501\nbody: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", ct)
	}

	var body struct {
		Status int `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode problem body: %v", err)
	}
	if body.Status != http.StatusNotImplemented {
		t.Errorf("problem.status = %d, want 501", body.Status)
	}
}

// TestMissingGameReturns400 documente que la validation du contrat s'applique
// avant le stub : sans le paramètre requis game, on obtient un 400.
func TestMissingGameReturns400(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/cars", nil)

	newServer(t).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400\nbody: %s", rec.Code, rec.Body.String())
	}
}
