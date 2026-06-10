// Handler du journal des changements : GET /v1/changes. Lecture seule, store
// paramétré, pagination (page/page_size), plus récents d'abord.
package handler

import (
	"context"

	"github.com/zinackes/forza-open-api/internal/oas"
	"github.com/zinackes/forza-open-api/internal/store"
)

// ListChanges implémente GET /v1/changes.
func (h *Handler) ListChanges(ctx context.Context, params oas.ListChangesParams) (oas.ListChangesRes, error) {
	page, size := pageParams(params.Page, params.PageSize)
	rows, total, err := h.store.ListChanges(ctx, store.ChangeFilter{
		Game:     string(params.Game),
		Resource: optFilter(params.Resource.Set, params.Resource.Value),
		Action:   optFilter(params.Action.Set, string(params.Action.Value)),
		Since:    optTimeFilter(params.Since),
		Limit:    size,
		Offset:   (page - 1) * size,
	})
	if err != nil {
		return nil, err
	}

	items := make([]oas.Change, 0, len(rows))
	for _, c := range rows {
		items = append(items, oas.Change{
			ID:         c.ID,
			Game:       oas.Game(c.Game),
			Resource:   c.Resource,
			ResourceId: c.ResourceID,
			Action:     oas.ChangeAction(c.Action),
			Summary:    optString(c.Summary),
			OccurredAt: c.OccurredAt,
		})
	}
	return &oas.ChangeList{Items: items, Total: total, Page: page, PageSize: size}, nil
}
