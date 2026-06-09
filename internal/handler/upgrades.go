// Handlers du catalogue d'upgrade : GET /v1/upgrade-parts (catalogue global d'un
// jeu) et GET /v1/cars/{id}/upgrades (upgrades montables sur une voiture).
// Lecture seule, store paramétré, pagination. Mapping DB → contrat.
package handler

import (
	"context"

	"github.com/zinackes/forza-open-api/internal/oas"
	"github.com/zinackes/forza-open-api/internal/store"
)

// ListUpgradeParts implémente GET /v1/upgrade-parts.
func (h *Handler) ListUpgradeParts(ctx context.Context, params oas.ListUpgradePartsParams) (oas.ListUpgradePartsRes, error) {
	page, size := pageParams(params.Page, params.PageSize)
	rows, total, err := h.store.ListUpgradeParts(ctx, store.UpgradePartFilter{
		Game:     string(params.Game),
		Category: optFilter(params.Category.Set, string(params.Category.Value)),
		Limit:    size,
		Offset:   (page - 1) * size,
	})
	if err != nil {
		return nil, err
	}

	items := make([]oas.UpgradePart, 0, len(rows))
	for _, p := range rows {
		items = append(items, upgradePartToOAS(p))
	}
	return &oas.UpgradePartList{Items: items, Total: total, Page: page, PageSize: size}, nil
}

// ListCarUpgrades implémente GET /v1/cars/{id}/upgrades.
func (h *Handler) ListCarUpgrades(ctx context.Context, params oas.ListCarUpgradesParams) (oas.ListCarUpgradesRes, error) {
	page, size := pageParams(params.Page, params.PageSize)
	rows, total, err := h.store.ListCarUpgrades(ctx, store.CarUpgradeFilter{
		CarID:    params.ID,
		Category: optFilter(params.Category.Set, string(params.Category.Value)),
		Limit:    size,
		Offset:   (page - 1) * size,
	})
	if err != nil {
		return nil, err
	}

	items := make([]oas.CarUpgrade, 0, len(rows))
	for _, u := range rows {
		items = append(items, oas.CarUpgrade{
			Part:           upgradePartToOAS(u.Part),
			RequiresPartId: optString(u.RequiresPartID),
			ExclusiveGroup: optString(u.ExclusiveGroup),
		})
	}
	return &oas.CarUpgradeList{Items: items, Total: total, Page: page, PageSize: size}, nil
}

// upgradePartToOAS mappe une pièce DB (snake_case) vers la vue camelCase du contrat.
func upgradePartToOAS(p store.UpgradePart) oas.UpgradePart {
	return oas.UpgradePart{
		ID:            p.ID,
		Game:          oas.Game(p.Game),
		Category:      oas.UpgradePartCategory(p.Category),
		Name:          p.Name,
		Level:         optInt(p.Level),
		PiDelta:       optInt(p.PIDelta),
		WeightDeltaKg: optInt(p.WeightDeltaKg),
		PowerDeltaHp:  optInt(p.PowerDeltaHp),
		TorqueDeltaNm: optInt(p.TorqueDeltaNm),
		Source:        optString(p.Source),
		LastVerified:  optTime(p.LastVerified),
	}
}
