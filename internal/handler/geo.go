// Handlers des ressources « carte » : tracks, pr_stunts, events. Lecture seule,
// store paramétré, pagination (page/page_size). Mapping snake_case DB → vue
// camelCase du contrat. Erreurs DB → 500 RFC 9457 via ProblemErrorHandler.
package handler

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-faster/jx"
	"github.com/zinackes/forza-open-api/internal/oas"
	"github.com/zinackes/forza-open-api/internal/store"
)

// ListTracks implémente GET /v1/tracks.
func (h *Handler) ListTracks(ctx context.Context, params oas.ListTracksParams) (oas.ListTracksRes, error) {
	page, size := pageParams(params.Page, params.PageSize)
	rows, total, err := h.store.ListTracks(ctx, store.GeoFilter{
		Game:   string(params.Game),
		Type:   optFilter(params.Type.Set, string(params.Type.Value)),
		Region: optFilter(params.Region.Set, string(params.Region.Value)),
		Limit:  size,
		Offset: (page - 1) * size,
	})
	if err != nil {
		return nil, err
	}

	items := make([]oas.Track, 0, len(rows))
	for _, t := range rows {
		items = append(items, oas.Track{
			ID:           t.ID,
			Game:         oas.Game(t.Game),
			Name:         t.Name,
			Type:         oas.TrackType(t.Type),
			Region:       optRegion(t.Region),
			LengthM:      optInt(t.LengthM),
			SurfaceMix:   optString(t.SurfaceMix),
			StartLat:     optFloat(t.StartLat),
			StartLng:     optFloat(t.StartLng),
			Source:       optString(t.Source),
			LastVerified: optTime(t.LastVerified),
			CreatedAt:    oas.NewOptDateTime(t.CreatedAt),
			UpdatedAt:    oas.NewOptDateTime(t.UpdatedAt),
		})
	}
	return &oas.TrackList{Items: items, Total: total, Page: page, PageSize: size}, nil
}

// ListPrStunts implémente GET /v1/pr-stunts.
func (h *Handler) ListPrStunts(ctx context.Context, params oas.ListPrStuntsParams) (oas.ListPrStuntsRes, error) {
	page, size := pageParams(params.Page, params.PageSize)
	rows, total, err := h.store.ListPRStunts(ctx, store.GeoFilter{
		Game:   string(params.Game),
		Type:   optFilter(params.Type.Set, string(params.Type.Value)),
		Region: optFilter(params.Region.Set, string(params.Region.Value)),
		Limit:  size,
		Offset: (page - 1) * size,
	})
	if err != nil {
		return nil, err
	}

	items := make([]oas.PRStunt, 0, len(rows))
	for _, p := range rows {
		items = append(items, oas.PRStunt{
			ID:          p.ID,
			Game:        oas.Game(p.Game),
			Type:        oas.PRStuntType(p.Type),
			Name:        p.Name,
			Region:      optRegion(p.Region),
			Lat:         optFloat(p.Lat),
			Lng:         optFloat(p.Lng),
			TargetScore: optInt(p.TargetScore),
		})
	}
	return &oas.PRStuntList{Items: items, Total: total, Page: page, PageSize: size}, nil
}

// ListEvents implémente GET /v1/events.
func (h *Handler) ListEvents(ctx context.Context, params oas.ListEventsParams) (oas.ListEventsRes, error) {
	page, size := pageParams(params.Page, params.PageSize)
	rows, total, err := h.store.ListEvents(ctx, store.GeoFilter{
		Game:   string(params.Game),
		Type:   optFilter(params.Type.Set, string(params.Type.Value)),
		Region: optFilter(params.Region.Set, string(params.Region.Value)),
		Limit:  size,
		Offset: (page - 1) * size,
	})
	if err != nil {
		return nil, err
	}

	items := make([]oas.Event, 0, len(rows))
	for _, e := range rows {
		items = append(items, oas.Event{
			ID:                  e.ID,
			Game:                oas.Game(e.Game),
			Name:                e.Name,
			Type:                oas.EventType(e.Type),
			Region:              optRegion(e.Region),
			StartLat:            optFloat(e.StartLat),
			StartLng:            optFloat(e.StartLng),
			EndLat:              optFloat(e.EndLat),
			EndLng:              optFloat(e.EndLng),
			RouteGeojson:        optRouteGeojson(e.RouteGeojson),
			CarClassRestriction: optString(e.CarClassRestriction),
			LengthM:             optInt(e.LengthM),
		})
	}
	return &oas.EventList{Items: items, Total: total, Page: page, PageSize: size}, nil
}

// pageParams extrait page/page_size (defaults appliqués par ogen : 1 / 50).
func pageParams(page, size oas.OptInt) (int, int) {
	p, s := 1, 50
	if v, ok := page.Get(); ok {
		p = v
	}
	if v, ok := size.Get(); ok {
		s = v
	}
	return p, s
}

// optFilter convertit un paramètre de filtre optionnel en *string (nil si absent).
func optFilter(set bool, v string) *string {
	if !set {
		return nil
	}
	return &v
}

func optString(p *string) oas.OptString {
	if p == nil {
		return oas.OptString{}
	}
	return oas.NewOptString(*p)
}

func optInt(p *int) oas.OptInt {
	if p == nil {
		return oas.OptInt{}
	}
	return oas.NewOptInt(*p)
}

func optFloat(p *float64) oas.OptFloat64 {
	if p == nil {
		return oas.OptFloat64{}
	}
	return oas.NewOptFloat64(*p)
}

func optTime(p *time.Time) oas.OptDateTime {
	if p == nil {
		return oas.OptDateTime{}
	}
	return oas.NewOptDateTime(*p)
}

func optRegion(p *string) oas.OptRegion {
	if p == nil {
		return oas.OptRegion{}
	}
	return oas.NewOptRegion(oas.Region(*p))
}

// optRouteGeojson convertit le JSONB brut en OptEventRouteGeojson. JSON invalide
// ou nil → champ omis (jamais d'erreur exposée pour une donnée annexe).
func optRouteGeojson(raw []byte) oas.OptEventRouteGeojson {
	if len(raw) == 0 {
		return oas.OptEventRouteGeojson{}
	}
	var rm map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rm); err != nil {
		return oas.OptEventRouteGeojson{}
	}
	m := make(oas.EventRouteGeojson, len(rm))
	for k, v := range rm {
		m[k] = jx.Raw(v)
	}
	return oas.NewOptEventRouteGeojson(m)
}
