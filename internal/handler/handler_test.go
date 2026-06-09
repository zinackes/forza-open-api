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
// est aussi rejeté en 400 par l'enum du contrat. Couvre aussi les ressources de
// voitures cachées (barn-finds, treasure-cars), qui ne portent que game + region.
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
		{"barn-finds sans game", "/v1/barn-finds", http.StatusBadRequest},
		{"treasure-cars sans game", "/v1/treasure-cars", http.StatusBadRequest},
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

// TestJournalEndpointValidation vérifie que /v1/journal applique la validation du
// contrat AVANT le handler : game requis (multi-jeux) et track hors enum rejeté
// en 400. Avec un store nil, un 400 prouve qu'on n'a jamais touché la DB.
func TestJournalEndpointValidation(t *testing.T) {
	cases := []struct {
		name   string
		target string
		want   int
	}{
		{"journal sans game", "/v1/journal", http.StatusBadRequest},
		{"journal track invalide", "/v1/journal?game=fh6&track=nope", http.StatusBadRequest},
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

// TestRandomCarEndpointValidation vérifie que /v1/cars/random applique la
// validation du contrat AVANT le handler : game est requis (multi-jeux), et les
// filtres bornés par enum (class, drivetrain) rejettent une valeur hors domaine
// en 400. Avec un store nil, un 400 prouve qu'on n'a jamais touché la DB.
func TestRandomCarEndpointValidation(t *testing.T) {
	cases := []struct {
		name   string
		target string
		want   int
	}{
		{"random sans game", "/v1/cars/random", http.StatusBadRequest},
		{"random game invalide", "/v1/cars/random?game=fh99", http.StatusBadRequest},
		{"random class invalide", "/v1/cars/random?game=fh6&class=Z", http.StatusBadRequest},
		{"random drivetrain invalide", "/v1/cars/random?game=fh6&drivetrain=4WD", http.StatusBadRequest},
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

// TestReferenceEndpointValidation vérifie que /v1/reference applique la validation
// du contrat AVANT le handler : game est requis (multi-jeux), sa valeur est bornée
// par l'enum Game. Avec un store nil, un 400 prouve qu'on n'a jamais touché la DB.
func TestReferenceEndpointValidation(t *testing.T) {
	cases := []struct {
		name   string
		target string
		want   int
	}{
		{"reference sans game", "/v1/reference", http.StatusBadRequest},
		{"reference game invalide", "/v1/reference?game=fh99", http.StatusBadRequest},
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

// TestUpgradeEndpointsValidation vérifie que la validation du contrat s'applique
// AVANT le handler. /v1/upgrade-parts exige game (multi-jeux) ; une catégorie hors
// enum est rejetée en 400 sur les deux routes. /v1/cars/{id}/upgrades ne prend pas
// game (déterminé par la voiture). Avec un store nil, un 400 prouve qu'on n'a
// jamais touché la DB.
func TestUpgradeEndpointsValidation(t *testing.T) {
	cases := []struct {
		name   string
		target string
		want   int
	}{
		{"upgrade-parts sans game", "/v1/upgrade-parts", http.StatusBadRequest},
		{"upgrade-parts catégorie invalide", "/v1/upgrade-parts?game=fh6&category=nope", http.StatusBadRequest},
		{"car upgrades catégorie invalide", "/v1/cars/car-1/upgrades?category=nope", http.StatusBadRequest},
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
