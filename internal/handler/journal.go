// Handler du Collection Journal : GET /v1/journal. Lecture seule, store
// paramétré. Renvoie les paliers (wristbands + stamps) d'un jeu, filtrés par
// piste. Mapping snake_case DB → vue camelCase du contrat.
package handler

import (
	"context"

	"github.com/zinackes/forza-open-api/internal/oas"
	"github.com/zinackes/forza-open-api/internal/store"
)

// ListJournalTiers implémente GET /v1/journal.
func (h *Handler) ListJournalTiers(ctx context.Context, params oas.ListJournalTiersParams) (oas.ListJournalTiersRes, error) {
	rows, err := h.store.ListJournalTiers(ctx, store.JournalFilter{
		Game:  string(params.Game),
		Track: optFilter(params.Track.Set, string(params.Track.Value)),
	})
	if err != nil {
		return nil, err
	}

	items := make(oas.ListJournalTiersOKApplicationJSON, 0, len(rows))
	for _, j := range rows {
		items = append(items, oas.JournalTier{
			ID:                 j.ID,
			Game:               oas.Game(j.Game),
			Track:              oas.JournalTrack(j.Track),
			Level:              j.Level,
			Color:              optJournalColor(j.Color),
			Name:               j.Name,
			PointsRequired:     optInt(j.PointsRequired),
			RewardCarId:        optString(j.RewardCarID),
			UnlocksDescription: optString(j.UnlocksDescription),
			Source:             optString(j.Source),
			LastVerified:       optTime(j.LastVerified),
		})
	}
	return &items, nil
}

// optJournalColor convertit la couleur DB (nil pour les stamps Discover Japan)
// en OptJournalTierColor du contrat.
func optJournalColor(p *string) oas.OptJournalTierColor {
	if p == nil {
		return oas.OptJournalTierColor{}
	}
	return oas.NewOptJournalTierColor(oas.JournalTierColor(*p))
}
