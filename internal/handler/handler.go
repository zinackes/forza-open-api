// Package handler implémente les interfaces générées par ogen (oas.Handler,
// oas.SecurityHandler). Phase 0 : tous les endpoints renvoient 501 Not
// Implemented (via oas.UnimplementedHandler). Les implémentations réelles
// arriveront endpoint par endpoint, en contract-first.
package handler

import (
	"github.com/zinackes/forza-open-api/internal/oas"
	"github.com/zinackes/forza-open-api/internal/store"
)

// Handler porte les dépendances de données et satisfait oas.Handler.
// L'embed d'UnimplementedHandler fournit un stub 501 pour chaque opération non
// encore implémentée ; on remplacera méthode par méthode.
type Handler struct {
	oas.UnimplementedHandler
	store *store.Store
	// dataVersion : version du jeu de données publiée (tampon opérateur, env
	// DATA_VERSION), exposée par GET /v1/meta. Vide → champ omis.
	dataVersion string
}

// New construit le handler avec ses dépendances. dataVersion peut être vide
// (aucune version de data tamponnée → champ meta omis).
func New(st *store.Store, dataVersion string) *Handler {
	return &Handler{store: st, dataVersion: dataVersion}
}
