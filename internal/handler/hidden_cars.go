// Handlers des voitures cachées FH6 : barn-finds, treasure-cars. Lecture seule,
// store paramétré, pagination (page/page_size). Mapping snake_case DB → vue
// camelCase du contrat. Erreurs DB → 500 RFC 9457 via ProblemErrorHandler.
package handler

import (
	"context"
	"net/http"

	"github.com/zinackes/forza-open-api/internal/oas"
	"github.com/zinackes/forza-open-api/internal/store"
)

// ListBarnFinds implémente GET /v1/barn-finds.
func (h *Handler) ListBarnFinds(ctx context.Context, params oas.ListBarnFindsParams) (oas.ListBarnFindsRes, error) {
	page, size := pageParams(params.Page, params.PageSize)
	rows, total, err := h.store.ListBarnFinds(ctx, store.HiddenCarFilter{
		Game:   string(params.Game),
		Region: optFilter(params.Region.Set, string(params.Region.Value)),
		CarID:  optFilter(params.CarID.Set, params.CarID.Value),
		Limit:  size,
		Offset: (page - 1) * size,
	})
	if err != nil {
		return nil, err
	}

	items := make([]oas.BarnFind, 0, len(rows))
	for _, b := range rows {
		items = append(items, mapBarnFind(b))
	}
	return &oas.BarnFindList{Items: items, Total: total, Page: page, PageSize: size}, nil
}

// GetBarnFind implémente GET /v1/barn-finds/{id} : le Barn Find demandé, ou un
// 404 RFC 9457.
func (h *Handler) GetBarnFind(ctx context.Context, params oas.GetBarnFindParams) (oas.GetBarnFindRes, error) {
	b, err := h.store.GetBarnFind(ctx, params.ID)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return &oas.GetBarnFindNotFound{
			Title:  oas.NewOptString(http.StatusText(http.StatusNotFound)),
			Status: oas.NewOptInt(http.StatusNotFound),
			Detail: oas.NewOptString("no barn find with the given id"),
		}, nil
	}
	bf := mapBarnFind(*b)
	return &bf, nil
}

// ListTreasureCars implémente GET /v1/treasure-cars.
func (h *Handler) ListTreasureCars(ctx context.Context, params oas.ListTreasureCarsParams) (oas.ListTreasureCarsRes, error) {
	page, size := pageParams(params.Page, params.PageSize)
	rows, total, err := h.store.ListTreasureCars(ctx, store.HiddenCarFilter{
		Game:   string(params.Game),
		Region: optFilter(params.Region.Set, string(params.Region.Value)),
		CarID:  optFilter(params.CarID.Set, params.CarID.Value),
		Limit:  size,
		Offset: (page - 1) * size,
	})
	if err != nil {
		return nil, err
	}

	items := make([]oas.TreasureCar, 0, len(rows))
	for _, t := range rows {
		items = append(items, mapTreasureCar(t))
	}
	return &oas.TreasureCarList{Items: items, Total: total, Page: page, PageSize: size}, nil
}

// GetTreasureCar implémente GET /v1/treasure-cars/{id} : la Treasure Car
// demandée, ou un 404 RFC 9457.
func (h *Handler) GetTreasureCar(ctx context.Context, params oas.GetTreasureCarParams) (oas.GetTreasureCarRes, error) {
	t, err := h.store.GetTreasureCar(ctx, params.ID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return &oas.GetTreasureCarNotFound{
			Title:  oas.NewOptString(http.StatusText(http.StatusNotFound)),
			Status: oas.NewOptInt(http.StatusNotFound),
			Detail: oas.NewOptString("no treasure car with the given id"),
		}, nil
	}
	tc := mapTreasureCar(*t)
	return &tc, nil
}

// mapBarnFind projette la vue DB d'un Barn Find sur le modèle du contrat.
func mapBarnFind(b store.BarnFind) oas.BarnFind {
	return oas.BarnFind{
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
	}
}

// mapTreasureCar projette la vue DB d'une Treasure Car sur le modèle du contrat.
func mapTreasureCar(t store.TreasureCar) oas.TreasureCar {
	return oas.TreasureCar{
		ID:           t.ID,
		Game:         oas.Game(t.Game),
		CarId:        t.CarID,
		Region:       optRegion(t.Region),
		PostcardClue: optString(t.PostcardClueText),
		LocationLat:  optFloat(t.LocationLat),
		LocationLng:  optFloat(t.LocationLng),
		Source:       optString(t.Source),
		LastVerified: optTime(t.LastVerified),
	}
}
