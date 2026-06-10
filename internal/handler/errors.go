package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/ogen-go/ogen/ogenerrors"
)

// problem est le corps d'erreur RFC 9457 (application/problem+json). C'est le
// SEUL format d'erreur de l'API : 400, 401, 404, 429 et 5xx le partagent, écrits
// par writeProblem (cf. ci-dessous). Voir le schéma Error du contrat.
type problem struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
}

// apiError porte un statut HTTP et un détail destinés au client, pour qu'un
// handler (signature ogen sans *http.Request) délègue le rendu du corps au
// ProblemErrorHandler, seul à avoir la requête (donc l'instance). C'est le
// mécanisme d'erreur ogen : un handler renvoie (nil, err), le wrapper généré
// appelle cfg.ErrorHandler(ctx, w, r, err).
type apiError struct {
	status int
	detail string
}

func (e *apiError) Error() string { return e.detail }

// errNotFound construit un 404 typé (ressource introuvable). detail est exposé
// au client (libellé métier, jamais de donnée sensible).
func errNotFound(detail string) error {
	return &apiError{status: http.StatusNotFound, detail: detail}
}

// errUnauthorized construit un 401 typé. Utilisé par les endpoints à
// authentification requise (GET /v1/me) quand la clé n'a pu être résolue (clé
// absente sur un endpoint authentifié, ou révoquée entre le cache et le lookup).
func errUnauthorized(detail string) error {
	return &apiError{status: http.StatusUnauthorized, detail: detail}
}

// problemType renvoie le code d'erreur stable (URN) associé à un statut HTTP.
// Indépendant de l'host (le domaine prod n'est pas figé) et stable dans le temps.
// Documenté dans le schéma Error du contrat. Statut inconnu → about:blank (le
// statut HTTP fait alors office de code, conforme RFC 9457).
func problemType(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "urn:forza-open-api:problem:validation"
	case http.StatusUnauthorized:
		return "urn:forza-open-api:problem:unauthorized"
	case http.StatusNotFound:
		return "urn:forza-open-api:problem:not-found"
	case http.StatusTooManyRequests:
		return "urn:forza-open-api:problem:rate-limited"
	case http.StatusInternalServerError:
		return "urn:forza-open-api:problem:internal"
	default:
		return "about:blank"
	}
}

// writeProblem est l'UNIQUE écrivain d'un corps d'erreur de l'API : il garantit
// que 400/401/404/429/5xx ont strictement la même forme (type, title, status,
// detail, instance). instance = chemin de la requête (sans la query : on ne
// reflète pas de chaîne attaquant-contrôlée). L'appelant a déjà pu poser des
// en-têtes (Retry-After, X-RateLimit-*) avant l'appel.
func writeProblem(w http.ResponseWriter, r *http.Request, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problem{
		Type:     problemType(status),
		Title:    http.StatusText(status),
		Status:   status,
		Detail:   detail,
		Instance: r.URL.Path,
	})
}

// ProblemErrorHandler traduit toute erreur d'un handler ou d'ogen en RFC 9457,
// conformément à la règle api-contract. Erreurs reconnues :
//   - *apiError (404 métier renvoyé par un handler) : statut + détail portés ;
//   - erreurs ogen (validation 400, sécurité 401, non implémenté 501…) :
//     ogenerrors.ErrorCode fait le mapping du statut, err.Error() le détail.
//
// Les 500 (erreur interne : DB, bug…) ne doivent jamais exposer err.Error() au
// client (requêtes SQL, hôtes…) : détail générique, erreur réelle loguée.
func ProblemErrorHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
	var status int
	var detail string

	var ae *apiError
	if errors.As(err, &ae) {
		status, detail = ae.status, ae.detail
	} else {
		status = ogenerrors.ErrorCode(err)
		detail = err.Error()
	}

	if status == http.StatusInternalServerError {
		slog.ErrorContext(ctx, "internal error", "method", r.Method, "path", r.URL.Path, "err", err)
		detail = "internal error"
	}

	writeProblem(w, r, status, detail)
}
