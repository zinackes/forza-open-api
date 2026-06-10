// Handler des constructeurs : GET /v1/manufacturers. Lecture seule, store
// paramétré. car_count vient d'un agrégat GROUP BY côté store.
package handler

import (
	"context"

	"github.com/zinackes/forza-open-api/internal/oas"
	"github.com/zinackes/forza-open-api/internal/store"
)

// ListManufacturers implémente GET /v1/manufacturers : les constructeurs du jeu
// avec leur nombre de voitures (carCount), filtrables par pays et nom.
func (h *Handler) ListManufacturers(ctx context.Context, params oas.ListManufacturersParams) (oas.ListManufacturersRes, error) {
	rows, err := h.store.ListManufacturers(ctx, store.ManufacturerFilter{
		Game:    string(params.Game),
		Country: optFilter(params.Country.Set, params.Country.Value),
		Q:       optFilter(params.Q.Set, params.Q.Value),
	})
	if err != nil {
		return nil, err
	}

	items := make(oas.ListManufacturersOKApplicationJSON, 0, len(rows))
	for _, m := range rows {
		items = append(items, oas.Manufacturer{
			Game:     oas.Game(m.Game),
			Name:     m.Name,
			Country:  optString(m.Country),
			CarCount: m.CarCount,
		})
	}
	return &items, nil
}
