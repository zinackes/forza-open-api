// Handler des facettes de référence : GET /v1/reference. Lecture seule, store
// paramétré. Agrège enums + compteurs pour amorcer les filtres clients en un
// appel. Réponse fortement cacheable (Cache-Control long) : la donnée ne bouge
// qu'à l'ingestion, qui purge le cache edge (Cloudflare).
package handler

import (
	"context"

	"github.com/zinackes/forza-open-api/internal/oas"
	"github.com/zinackes/forza-open-api/internal/store"
)

// referenceCacheControl : cache long côté origin. L'invalidation fine est portée
// par les jobs d'ingestion (purge Cloudflare) ; stale-while-revalidate absorbe
// les pics pendant un refresh.
const referenceCacheControl = "public, max-age=86400, stale-while-revalidate=604800"

// GetReference implémente GET /v1/reference.
func (h *Handler) GetReference(ctx context.Context, params oas.GetReferenceParams) (oas.GetReferenceRes, error) {
	ref, err := h.store.GetReference(ctx, string(params.Game))
	if err != nil {
		return nil, err
	}

	return &oas.ReferenceHeaders{
		CacheControl: oas.NewOptString(referenceCacheControl),
		Response: oas.Reference{
			Game:          params.Game,
			Classes:       mapRefCounts(ref.Classes),
			Drivetrains:   mapRefCounts(ref.Drivetrains),
			BodyTypes:     mapRefCounts(ref.BodyTypes),
			Countries:     mapRefCounts(ref.Countries),
			Categories:    mapRefCounts(ref.Categories),
			Regions:       mapRefCounts(ref.Regions),
			TrackTypes:    mapRefCounts(ref.TrackTypes),
			EventTypes:    mapRefCounts(ref.EventTypes),
			PrStuntTypes:  mapRefCounts(ref.PRStuntTypes),
			ObtainMethods: mapRefCounts(ref.ObtainMethods),
			Games:         mapGameCounts(ref.Games),
		},
	}, nil
}

// mapRefCounts mappe les facettes DB vers la vue du contrat (ordre préservé).
func mapRefCounts(in []store.RefCount) []oas.RefCount {
	out := make([]oas.RefCount, 0, len(in))
	for _, r := range in {
		out = append(out, oas.RefCount{Code: r.Code, Count: r.Count})
	}
	return out
}

// mapGameCounts mappe les volumes par jeu vers la vue du contrat.
func mapGameCounts(in []store.GameCount) []oas.GameCount {
	out := make([]oas.GameCount, 0, len(in))
	for _, g := range in {
		out = append(out, oas.GameCount{
			Code:        oas.Game(g.Code),
			CountCars:   g.CountCars,
			CountSeries: g.CountSeries,
		})
	}
	return out
}
