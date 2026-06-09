// Handlers des voitures cachées FH6 : barn-finds, treasure-cars. Lecture seule,
// store paramétré, pagination (page/page_size). Mapping snake_case DB → vue
// camelCase du contrat. Erreurs DB → 500 RFC 9457 via ProblemErrorHandler.
package handler

import (
	"context"

	"github.com/zinackes/forza-open-api/internal/oas"
	"github.com/zinackes/forza-open-api/internal/store"
)

// ListBarnFinds implémente GET /v1/barn-finds.
func (h *Handler) ListBarnFinds(ctx context.Context, params oas.ListBarnFindsParams) (oas.ListBarnFindsRes, error) {
	page, size := pageParams(params.Page, params.PageSize)
	rows, total, err := h.store.ListBarnFinds(ctx, store.HiddenCarFilter{
		Game:   string(params.Game),
		Region: optFilter(params.Region.Set, string(params.Region.Value)),
		Limit:  size,
		Offset: (page - 1) * size,
	})
	if err != nil {
		return nil, err
	}

	items := make([]oas.BarnFind, 0, len(rows))
	for _, b := range rows {
		items = append(items, oas.BarnFind{
			ID:                     b.ID,
			Game:                   oas.Game(b.Game),
			CarId:                  b.CarID,
			Region:                 optRegion(b.Region),
			SearchZoneCenterLat:    optFloat(b.SearchZoneCenterLat),
			SearchZoneCenterLng:    optFloat(b.SearchZoneCenterLng),
			SearchZoneRadiusM:      optInt(b.SearchZoneRadiusM),
			PrerequisiteStampLevel: optInt(b.PrerequisiteStampLevel),
			RestorationTimeH:       optInt(b.RestorationTimeH),
			Source:                 optString(b.Source),
			LastVerified:           optTime(b.LastVerified),
		})
	}
	return &oas.BarnFindList{Items: items, Total: total, Page: page, PageSize: size}, nil
}

// ListTreasureCars implémente GET /v1/treasure-cars.
func (h *Handler) ListTreasureCars(ctx context.Context, params oas.ListTreasureCarsParams) (oas.ListTreasureCarsRes, error) {
	page, size := pageParams(params.Page, params.PageSize)
	rows, total, err := h.store.ListTreasureCars(ctx, store.HiddenCarFilter{
		Game:   string(params.Game),
		Region: optFilter(params.Region.Set, string(params.Region.Value)),
		Limit:  size,
		Offset: (page - 1) * size,
	})
	if err != nil {
		return nil, err
	}

	items := make([]oas.TreasureCar, 0, len(rows))
	for _, t := range rows {
		items = append(items, oas.TreasureCar{
			ID:           t.ID,
			Game:         oas.Game(t.Game),
			CarId:        t.CarID,
			Region:       optRegion(t.Region),
			PostcardClue: optString(t.PostcardClueText),
			LocationLat:  optFloat(t.LocationLat),
			LocationLng:  optFloat(t.LocationLng),
			Source:       optString(t.Source),
			LastVerified: optTime(t.LastVerified),
		})
	}
	return &oas.TreasureCarList{Items: items, Total: total, Page: page, PageSize: size}, nil
}
