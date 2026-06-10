// Handler des agrégats du catalogue : GET /v1/stats. Lecture seule, store
// paramétré. Compteurs par facette, histogramme PI et classements (top 10) en un
// appel. Réponse fortement cacheable (Cache-Control long) : la donnée ne bouge
// qu'à l'ingestion, qui purge le cache edge (Cloudflare).
package handler

import (
	"context"

	"github.com/zinackes/forza-open-api/internal/oas"
	"github.com/zinackes/forza-open-api/internal/store"
)

// statsCacheControl : même politique que /v1/reference (agrégats stables entre
// deux ingestions). L'invalidation fine est portée par les jobs d'ingestion.
const statsCacheControl = "public, max-age=86400, stale-while-revalidate=604800"

// GetStats implémente GET /v1/stats.
func (h *Handler) GetStats(ctx context.Context, params oas.GetStatsParams) (oas.GetStatsRes, error) {
	st, err := h.store.GetStats(ctx, string(params.Game))
	if err != nil {
		return nil, err
	}

	return &oas.StatsHeaders{
		CacheControl: oas.NewOptString(statsCacheControl),
		Response: oas.Stats{
			Game: params.Game,
			CountsBy: oas.StatsCountsBy{
				Class:        oas.CountMap(st.Class),
				Drivetrain:   oas.CountMap(st.Drivetrain),
				Manufacturer: mapNamedCounts(st.Manufacturer),
				YearDecade:   oas.CountMap(st.YearDecade),
				BodyType:     oas.CountMap(st.BodyType),
				Category:     oas.CountMap(st.Category),
			},
			PiHistogram: mapPiBuckets(st.PiHistogram),
			Top: oas.StatsTop{
				Pi:           mapTopCars(st.TopPI),
				Speed:        mapTopCars(st.TopSpeed),
				Acceleration: mapTopCars(st.TopAcceleration),
			},
		},
	}, nil
}

// mapNamedCounts mappe les volumes par constructeur (Code = make) vers la vue du
// contrat (ordre préservé : plus fréquents d'abord).
func mapNamedCounts(in []store.RefCount) []oas.NamedCount {
	out := make([]oas.NamedCount, 0, len(in))
	for _, r := range in {
		out = append(out, oas.NamedCount{Name: r.Code, Count: r.Count})
	}
	return out
}

// mapPiBuckets mappe les paliers de l'histogramme PI (Code = "lo-hi") vers la vue
// du contrat (ordre croissant préservé).
func mapPiBuckets(in []store.RefCount) []oas.PiBucket {
	out := make([]oas.PiBucket, 0, len(in))
	for _, r := range in {
		out = append(out, oas.PiBucket{Bucket: r.Code, Count: r.Count})
	}
	return out
}

// mapTopCars mappe un classement DB vers la vue du contrat (ordre préservé).
func mapTopCars(in []store.TopCar) []oas.TopCar {
	out := make([]oas.TopCar, 0, len(in))
	for _, c := range in {
		out = append(out, oas.TopCar{CarId: c.CarID, Name: c.Name, Value: c.Value})
	}
	return out
}
