// Handler de l'arbre Car Mastery : GET /v1/cars/{id}/mastery. Lecture seule,
// store paramétré. Renvoie la grille de perks (tableau plat ; le client
// reconstruit la grille 4×4 via row/col + prereqPerkId). Mapping DB → contrat.
package handler

import (
	"context"

	"github.com/zinackes/forza-open-api/internal/oas"
)

// GetCarMastery implémente GET /v1/cars/{id}/mastery.
func (h *Handler) GetCarMastery(ctx context.Context, params oas.GetCarMasteryParams) (oas.GetCarMasteryRes, error) {
	rows, err := h.store.ListCarMasteryPerks(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	items := make(oas.GetCarMasteryOKApplicationJSON, 0, len(rows))
	for _, p := range rows {
		items = append(items, oas.CarMasteryPerk{
			ID:                p.ID,
			CarId:             p.CarID,
			Row:               optInt(p.Row),
			Col:               optInt(p.Col),
			Name:              optString(p.Name),
			SpCost:            optInt(p.SPCost),
			EffectDescription: optString(p.EffectDescription),
			EffectType:        optString(p.EffectType),
			EffectValue:       optInt(p.EffectValue),
			PrereqPerkId:      optString(p.PrereqPerkID),
			UnlockedCarId:     optString(p.UnlockedCarID),
			Source:            optString(p.Source),
			LastVerified:      optTime(p.LastVerified),
		})
	}
	return &items, nil
}
