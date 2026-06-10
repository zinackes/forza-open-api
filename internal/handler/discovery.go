// Handlers des contenus Discovery FH6 : stories, tours. Lecture seule, store
// paramétré, pagination (page/page_size). Mapping snake_case DB → vue camelCase
// du contrat. Champs non sourcés → omis.
package handler

import (
	"context"

	"github.com/zinackes/forza-open-api/internal/oas"
	"github.com/zinackes/forza-open-api/internal/store"
)

// ListStories implémente GET /v1/stories.
func (h *Handler) ListStories(ctx context.Context, params oas.ListStoriesParams) (oas.ListStoriesRes, error) {
	page, size := pageParams(params.Page, params.PageSize)
	rows, total, err := h.store.ListStories(ctx, store.DiscoveryFilter{
		Game:   string(params.Game),
		Region: optFilter(params.Region.Set, string(params.Region.Value)),
		Limit:  size,
		Offset: (page - 1) * size,
	})
	if err != nil {
		return nil, err
	}

	items := make([]oas.Story, 0, len(rows))
	for _, st := range rows {
		items = append(items, oas.Story{
			ID:                st.ID,
			Game:              oas.Game(st.Game),
			Name:              st.Name,
			Region:            optRegion(st.Region),
			Description:       optString(st.Description),
			ChaptersCount:     optInt(st.ChaptersCount),
			RewardDescription: optString(st.RewardDescription),
			Source:            optString(st.Source),
			LastVerified:      optTime(st.LastVerified),
		})
	}
	return &oas.StoryList{Items: items, Total: total, Page: page, PageSize: size}, nil
}

// ListTours implémente GET /v1/tours.
func (h *Handler) ListTours(ctx context.Context, params oas.ListToursParams) (oas.ListToursRes, error) {
	page, size := pageParams(params.Page, params.PageSize)
	rows, total, err := h.store.ListTours(ctx, store.DiscoveryFilter{
		Game:   string(params.Game),
		Region: optFilter(params.Region.Set, string(params.Region.Value)),
		Limit:  size,
		Offset: (page - 1) * size,
	})
	if err != nil {
		return nil, err
	}

	items := make([]oas.Tour, 0, len(rows))
	for _, t := range rows {
		items = append(items, oas.Tour{
			ID:           t.ID,
			Game:         oas.Game(t.Game),
			Name:         t.Name,
			Region:       optRegion(t.Region),
			Description:  optString(t.Description),
			Source:       optString(t.Source),
			LastVerified: optTime(t.LastVerified),
		})
	}
	return &oas.TourList{Items: items, Total: total, Page: page, PageSize: size}, nil
}
