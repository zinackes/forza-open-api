// Handler de la référence DLC : GET /v1/dlc-packs. Lecture seule, store paramétré.
package handler

import (
	"context"
	"net/http"

	"github.com/zinackes/forza-open-api/internal/oas"
	"github.com/zinackes/forza-open-api/internal/store"
)

// ListDlcPacks implémente GET /v1/dlc-packs.
func (h *Handler) ListDlcPacks(ctx context.Context, params oas.ListDlcPacksParams) (oas.ListDlcPacksRes, error) {
	rows, err := h.store.ListDlcPacks(ctx, string(params.Game),
		optFilter(params.Kind.Set, string(params.Kind.Value)))
	if err != nil {
		return nil, err
	}

	items := make(oas.ListDlcPacksOKApplicationJSON, 0, len(rows))
	for _, d := range rows {
		items = append(items, mapDlcPack(d))
	}
	return &items, nil
}

// GetDlcPack implémente GET /v1/dlc-packs/{id} : le pack demandé, ou un 404
// RFC 9457.
func (h *Handler) GetDlcPack(ctx context.Context, params oas.GetDlcPackParams) (oas.GetDlcPackRes, error) {
	d, err := h.store.GetDlcPack(ctx, params.ID)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return &oas.GetDlcPackNotFound{
			Title:  oas.NewOptString(http.StatusText(http.StatusNotFound)),
			Status: oas.NewOptInt(http.StatusNotFound),
			Detail: oas.NewOptString("no dlc pack with the given id"),
		}, nil
	}
	pack := mapDlcPack(*d)
	return &pack, nil
}

// mapDlcPack projette la vue DB d'un pack DLC sur le modèle du contrat.
func mapDlcPack(d store.DlcPack) oas.DlcPack {
	return oas.DlcPack{
		ID:           d.ID,
		Game:         oas.Game(d.Game),
		Name:         d.Name,
		Kind:         oas.DlcKind(d.Kind),
		ReleasedAt:   optTime(d.ReleasedAt),
		Description:  optString(d.Description),
		Source:       optString(d.Source),
		LastVerified: optTime(d.LastVerified),
	}
}
