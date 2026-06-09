// Package handler implémente les interfaces générées par ogen (oas.Handler,
// oas.SecurityHandler). Phase 0 : tous les endpoints renvoient 501 Not
// Implemented (via oas.UnimplementedHandler). Les implémentations réelles
// arriveront endpoint par endpoint, en contract-first.
package handler

import (
	"context"

	"github.com/zinackes/forza-open-api/internal/oas"
	"github.com/zinackes/forza-open-api/internal/store"
)

// Handler porte les dépendances de données et satisfait oas.Handler.
// L'embed d'UnimplementedHandler fournit un stub 501 pour chaque opération non
// encore implémentée ; on remplacera méthode par méthode.
type Handler struct {
	oas.UnimplementedHandler
	store *store.Store
}

// New construit le handler avec ses dépendances.
func New(st *store.Store) *Handler {
	return &Handler{store: st}
}

// SecurityHandler implémente oas.SecurityHandler.
type SecurityHandler struct{}

// HandleApiKeyAuth est un no-op en Phase 0 : aucune clé n'est exigée (lecture
// publique). La vérification réelle (hash sha256 en DB, rate-limit, révocation)
// sera branchée ici sans toucher au code généré.
func (SecurityHandler) HandleApiKeyAuth(ctx context.Context, _ oas.OperationName, _ oas.ApiKeyAuth) (context.Context, error) {
	return ctx, nil
}
