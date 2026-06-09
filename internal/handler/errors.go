package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/ogen-go/ogen/ogenerrors"
)

// problem est le corps d'erreur RFC 9457 (application/problem+json).
type problem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// ProblemErrorHandler traduit toute erreur non gérée par les handlers (codes
// ogen : validation 400, sécurité 401, non implémenté 501, …) en RFC 9457,
// conformément à la règle api-contract. ogenerrors.ErrorCode fait le mapping
// du statut ; on ne fait que rendre le bon Content-Type et le bon corps.
func ProblemErrorHandler(_ context.Context, w http.ResponseWriter, _ *http.Request, err error) {
	code := ogenerrors.ErrorCode(err)
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(problem{
		Type:   "about:blank",
		Title:  http.StatusText(code),
		Status: code,
		Detail: err.Error(),
	})
}
