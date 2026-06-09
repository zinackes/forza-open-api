// Handler de la recherche globale : GET /v1/search. Lecture seule, UNION côté
// store. Pas de pagination (limit borné par le contrat) : pensé autocomplete.
package handler

import (
	"context"

	"github.com/zinackes/forza-open-api/internal/oas"
	"github.com/zinackes/forza-open-api/internal/store"
)

// Search implémente GET /v1/search.
func (h *Handler) Search(ctx context.Context, params oas.SearchParams) (oas.SearchRes, error) {
	limit := 10
	if v, ok := params.Limit.Get(); ok {
		limit = v
	}
	kinds := make([]string, 0, len(params.Kinds))
	for _, k := range params.Kinds {
		kinds = append(kinds, string(k))
	}

	rows, err := h.store.Search(ctx, store.SearchFilter{
		Game:  string(params.Game),
		Q:     params.Q,
		Kinds: kinds,
		Limit: limit,
	})
	if err != nil {
		return nil, err
	}

	items := make([]oas.SearchResult, 0, len(rows))
	for _, r := range rows {
		items = append(items, oas.SearchResult{
			Kind:   oas.SearchResultKind(r.Kind),
			ID:     r.ID,
			Game:   oas.Game(r.Game),
			Name:   r.Name,
			Detail: optString(r.Detail),
		})
	}
	return &oas.SearchResultList{Items: items}, nil
}
