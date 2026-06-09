// Handler de la référence DLC : GET /v1/dlc-packs. Lecture seule, store paramétré.
package handler

import (
	"context"

	"github.com/zinackes/forza-open-api/internal/oas"
)

// ListDlcPacks implémente GET /v1/dlc-packs.
func (h *Handler) ListDlcPacks(ctx context.Context, params oas.ListDlcPacksParams) (oas.ListDlcPacksRes, error) {
	rows, err := h.store.ListDlcPacks(ctx, string(params.Game))
	if err != nil {
		return nil, err
	}

	items := make(oas.ListDlcPacksOKApplicationJSON, 0, len(rows))
	for _, d := range rows {
		items = append(items, oas.DlcPack{
			ID:           d.ID,
			Game:         oas.Game(d.Game),
			Name:         d.Name,
			Kind:         oas.DlcKind(d.Kind),
			ReleasedAt:   optTime(d.ReleasedAt),
			Description:  optString(d.Description),
			Source:       optString(d.Source),
			LastVerified: optTime(d.LastVerified),
		})
	}
	return &items, nil
}
