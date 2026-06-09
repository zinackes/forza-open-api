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

// TestUnimplementedReturns501 vérifie qu'un endpoint non encore implémenté répond
// 501 en RFC 9457 (application/problem+json). game=fh6 est requis pour passer
// la validation du contrat et atteindre le stub. /v1/manufacturers reste un stub
// (cars, tracks, pr-stunts, events, dlc-packs sont eux implémentés).
func TestUnimplementedReturns501(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/manufacturers?game=fh6", nil)

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
// avant le handler : sans le paramètre requis game, on obtient un 400. Vaut pour
// /v1/cars comme pour /v1/dlc-packs (avec store nil, un 400 prouve qu'on n'a
// jamais touché la DB).
func TestMissingGameReturns400(t *testing.T) {
	for _, target := range []string{"/v1/cars", "/v1/dlc-packs"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, target, nil)

		newServer(t).ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400\nbody: %s", target, rec.Code, rec.Body.String())
		}
	}
}

// TestGeoEndpointsRequireGame vérifie que tracks/pr-stunts/events appliquent la
// validation du contrat (game requis) AVANT d'appeler le handler : avec un store
// nil, un 400 prouve que la requête n'a jamais touché la DB. Un type invalide
// est aussi rejeté en 400 par l'enum du contrat.
func TestGeoEndpointsRequireGame(t *testing.T) {
	cases := []struct {
		name   string
		target string
		want   int
	}{
		{"tracks sans game", "/v1/tracks", http.StatusBadRequest},
		{"pr-stunts sans game", "/v1/pr-stunts", http.StatusBadRequest},
		{"events sans game", "/v1/events", http.StatusBadRequest},
		{"tracks type invalide", "/v1/tracks?game=fh6&type=nope", http.StatusBadRequest},
		{"pr-stunts type invalide", "/v1/pr-stunts?game=fh6&type=nope", http.StatusBadRequest},
		{"events type invalide", "/v1/events?game=fh6&type=nope", http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.target, nil)

			newServer(t).ServeHTTP(rec, req)

			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, tc.want, rec.Body.String())
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
				t.Errorf("Content-Type = %q, want application/problem+json", ct)
			}
		})
	}
}
