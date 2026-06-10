// Handlers Forzathon Shop : GET /v1/forzathon-shop (rotation courante d'un jeu)
// et /v1/forzathon-shop/history (rotations passées, paginé). Lecture seule, store
// paramétré. Mapping DB → contrat ; champs nullables omis si absents.
package handler

import (
	"context"

	"github.com/zinackes/forza-open-api/internal/oas"
	"github.com/zinackes/forza-open-api/internal/store"
)

// forzathonCacheControl : la rotation tourne ~hebdo (reset jeudi 14:30 UTC) et
// l'ingestion bascule la semaine courante. Même politique que la playlist : cache
// court + stale-while-revalidate (sert périmé jusqu'à 1 j en revalidant) couplé à
// l'ETag (ConditionalGet) + bump DATA_VERSION au refresh → revalidation 304.
const forzathonCacheControl = "public, max-age=300, s-maxage=900, stale-while-revalidate=86400"

// GetForzathonShop implémente GET /v1/forzathon-shop : les objets de la rotation
// courante (semaine la plus récente connue) du jeu. Tableau vide si aucune.
func (h *Handler) GetForzathonShop(ctx context.Context, params oas.GetForzathonShopParams) (oas.GetForzathonShopRes, error) {
	rows, err := h.store.CurrentForzathonShop(ctx, string(params.Game))
	if err != nil {
		return nil, err
	}
	out := make([]oas.ForzathonShopItem, 0, len(rows))
	for _, it := range rows {
		out = append(out, mapForzathonItem(it))
	}
	return &oas.GetForzathonShopOKHeaders{
		CacheControl: oas.NewOptString(forzathonCacheControl),
		Response:     out,
	}, nil
}

// ListForzathonShopHistory implémente GET /v1/forzathon-shop/history : page des
// objets de toutes les rotations connues du jeu, les plus récentes d'abord.
func (h *Handler) ListForzathonShopHistory(ctx context.Context, params oas.ListForzathonShopHistoryParams) (oas.ListForzathonShopHistoryRes, error) {
	page, size := pageParams(params.Page, params.PageSize)
	rows, total, err := h.store.ForzathonShopHistory(ctx, string(params.Game), size, (page-1)*size)
	if err != nil {
		return nil, err
	}
	items := make([]oas.ForzathonShopItem, 0, len(rows))
	for _, it := range rows {
		items = append(items, mapForzathonItem(it))
	}
	return &oas.ForzathonShopList{Items: items, Total: total, Page: page, PageSize: size}, nil
}

// mapForzathonItem projette la vue DB d'un objet de shop sur le modèle du contrat.
func mapForzathonItem(it store.ForzathonShopItem) oas.ForzathonShopItem {
	return oas.ForzathonShopItem{
		ID:           it.ID,
		Game:         oas.Game(it.Game),
		WeekStart:    it.WeekStart,
		WeekEnd:      optTime(it.WeekEnd),
		Kind:         oas.ForzathonShopKind(it.Kind),
		CarId:        optString(it.CarID),
		Name:         it.Name,
		FpCost:       optInt(it.FpCost),
		Description:  optString(it.Description),
		ImageUrl:     optURI(it.ImageURL),
		Source:       optString(it.Source),
		LastVerified: optTime(it.LastVerified),
	}
}
