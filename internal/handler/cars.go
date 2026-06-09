// Handler du catalogue voitures : GET /v1/cars. Lecture seule, store paramétré,
// pagination, filtres (make/class/pi/drivetrain/q + dlc). Mapping DB → contrat.
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/zinackes/forza-open-api/internal/oas"
	"github.com/zinackes/forza-open-api/internal/store"
)

// ListCars implémente GET /v1/cars.
func (h *Handler) ListCars(ctx context.Context, params oas.ListCarsParams) (oas.ListCarsRes, error) {
	page, size := pageParams(params.Page, params.PageSize)
	rows, total, err := h.store.ListCars(ctx, store.CarFilter{
		Game:       string(params.Game),
		Make:       optFilter(params.Make.Set, params.Make.Value),
		Class:      optFilter(params.Class.Set, string(params.Class.Value)),
		PIMin:      optIntFilter(params.PiMin),
		PIMax:      optIntFilter(params.PiMax),
		Drivetrain: optFilter(params.Drivetrain.Set, string(params.Drivetrain.Value)),
		Category:   optFilter(params.Category.Set, params.Category.Value),
		Q:          optFilter(params.Q.Set, params.Q.Value),
		Dlc:        optFilter(params.Dlc.Set, params.Dlc.Value),
		Limit:      size,
		Offset:     (page - 1) * size,
	})
	if err != nil {
		return nil, err
	}

	items := make([]oas.Car, 0, len(rows))
	for _, c := range rows {
		items = append(items, mapCar(c))
	}
	return &oas.CarList{Items: items, Total: total, Page: page, PageSize: size}, nil
}

// randomCarCacheControl : un tirage aléatoire ne doit jamais être mis en cache
// (edge Cloudflare ou navigateur), sinon tous les clients verraient la même
// voiture « random ». Chaque appel re-tire → no-store.
const randomCarCacheControl = "no-store"

// GetRandomCar implémente GET /v1/cars/random : une voiture au hasard parmi les
// correspondances, 404 si aucune. Réponse jamais cachée (no-store).
func (h *Handler) GetRandomCar(ctx context.Context, params oas.GetRandomCarParams) (oas.GetRandomCarRes, error) {
	c, err := h.store.RandomCar(ctx, store.CarFilter{
		Game:       string(params.Game),
		Make:       optFilter(params.Make.Set, params.Make.Value),
		Class:      optFilter(params.Class.Set, string(params.Class.Value)),
		PIMin:      optIntFilter(params.PiMin),
		PIMax:      optIntFilter(params.PiMax),
		Drivetrain: optFilter(params.Drivetrain.Set, string(params.Drivetrain.Value)),
		Category:   optFilter(params.Category.Set, params.Category.Value),
	})
	if err != nil {
		return nil, err
	}
	if c == nil {
		return &oas.GetRandomCarNotFound{
			Title:  oas.NewOptString(http.StatusText(http.StatusNotFound)),
			Status: oas.NewOptInt(http.StatusNotFound),
			Detail: oas.NewOptString("no car matches the given filters"),
		}, nil
	}
	return &oas.CarHeaders{
		CacheControl: oas.NewOptString(randomCarCacheControl),
		Response:     mapCar(*c),
	}, nil
}

// GetCar implémente GET /v1/cars/{id} : la voiture demandée, ou un 404 RFC 9457
// (application/problem+json) si l'identifiant est inconnu.
func (h *Handler) GetCar(ctx context.Context, params oas.GetCarParams) (oas.GetCarRes, error) {
	c, err := h.store.GetCar(ctx, params.ID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return &oas.GetCarNotFound{
			Title:  oas.NewOptString(http.StatusText(http.StatusNotFound)),
			Status: oas.NewOptInt(http.StatusNotFound),
			Detail: oas.NewOptString("no car with the given id"),
		}, nil
	}
	car := mapCar(*c)
	return &car, nil
}

// mapCar projette la vue DB d'une voiture sur le modèle du contrat.
func mapCar(c store.Car) oas.Car {
	return oas.Car{
		ID:           c.ID,
		Game:         oas.Game(c.Game),
		Name:         c.Name,
		Make:         c.Make,
		Model:        optString(c.Model),
		Year:         optInt(c.Year),
		Class:        oas.CarClass(c.Class),
		Pi:           c.PI,
		Drivetrain:   oas.Drivetrain(c.Drivetrain),
		Stats:        optCarStats(c.Stats),
		BodyType:     optString(c.BodyType),
		Category:     optString(c.Category),
		Rarity:       optString(c.Rarity),
		ValueCr:      optInt64(c.ValueCr),
		ObtainMethod: optString(c.ObtainMethod),
		ImageUrl:     optURI(c.ImageURL),
		CreatedAt:    oas.NewOptDateTime(c.CreatedAt),
	}
}

// optIntFilter convertit un paramètre de filtre OptInt en *int (nil si absent).
func optIntFilter(p oas.OptInt) *int {
	if v, ok := p.Get(); ok {
		return &v
	}
	return nil
}

func optInt64(p *int64) oas.OptInt64 {
	if p == nil {
		return oas.OptInt64{}
	}
	return oas.NewOptInt64(*p)
}

// optURI parse une URL stockée en texte. URL vide ou invalide → champ omis
// (jamais d'erreur exposée pour une donnée annexe).
func optURI(p *string) oas.OptURI {
	if p == nil || *p == "" {
		return oas.OptURI{}
	}
	u, err := url.Parse(*p)
	if err != nil {
		return oas.OptURI{}
	}
	return oas.NewOptURI(*u)
}

// optCarStats convertit le JSONB brut en OptCarStats : les clés connues vont sur
// les champs nommés, le reste dans AdditionalProps. JSON invalide/nil → omis.
func optCarStats(raw []byte) oas.OptCarStats {
	if len(raw) == 0 {
		return oas.OptCarStats{}
	}
	var m map[string]float64
	if err := json.Unmarshal(raw, &m); err != nil {
		return oas.OptCarStats{}
	}
	cs := oas.CarStats{AdditionalProps: make(oas.CarStatsAdditional)}
	for k, v := range m {
		switch k {
		case "speed":
			cs.Speed = oas.NewOptFloat64(v)
		case "handling":
			cs.Handling = oas.NewOptFloat64(v)
		case "acceleration":
			cs.Acceleration = oas.NewOptFloat64(v)
		case "launch":
			cs.Launch = oas.NewOptFloat64(v)
		case "braking":
			cs.Braking = oas.NewOptFloat64(v)
		case "offroad":
			cs.Offroad = oas.NewOptFloat64(v)
		default:
			cs.AdditionalProps[k] = v
		}
	}
	return oas.NewOptCarStats(cs)
}
