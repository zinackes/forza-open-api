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
		Game:         string(params.Game),
		Type:         optFilter(params.Type.Set, string(params.Type.Value)),
		Region:       optFilter(params.Region.Set, string(params.Region.Value)),
		Q:            optFilter(params.Q.Set, params.Q.Value),
		UpdatedSince: optTimeFilter(params.UpdatedSince),
		Limit:        size,
		Offset:       (page - 1) * size,
	})
	if err != nil {
		return nil, err
	}

	items := make([]oas.Track, 0, len(rows))
	for _, t := range rows {
		items = append(items, mapTrack(t))
	}
	return &oas.TrackList{Items: items, Total: total, Page: page, PageSize: size}, nil
}

// GetTrack implémente GET /v1/tracks/{id} : le tracé demandé, ou un 404 RFC 9457.
func (h *Handler) GetTrack(ctx context.Context, params oas.GetTrackParams) (oas.GetTrackRes, error) {
	t, err := h.store.GetTrack(ctx, params.ID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, errNotFound("no track with the given id")
	}
	track := mapTrack(*t)
	return &track, nil
}

// GetRandomTrack implémente GET /v1/tracks/random : un tracé au hasard parmi les
// correspondances, 404 si aucun. Réponse jamais cachée (no-store), même règle
// que GET /v1/cars/random.
func (h *Handler) GetRandomTrack(ctx context.Context, params oas.GetRandomTrackParams) (oas.GetRandomTrackRes, error) {
	t, err := h.store.RandomTrack(ctx, store.GeoFilter{
		Game:   string(params.Game),
		Type:   optFilter(params.Type.Set, string(params.Type.Value)),
		Region: optFilter(params.Region.Set, string(params.Region.Value)),
	})
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, errNotFound("no track matches the given filters")
	}
	return &oas.TrackHeaders{
		CacheControl: oas.NewOptString(randomCarCacheControl),
		Response:     mapTrack(*t),
	}, nil
}

// ListPrStunts implémente GET /v1/pr-stunts.
func (h *Handler) ListPrStunts(ctx context.Context, params oas.ListPrStuntsParams) (oas.ListPrStuntsRes, error) {
	page, size := pageParams(params.Page, params.PageSize)
	rows, total, err := h.store.ListPRStunts(ctx, store.GeoFilter{
		Game:   string(params.Game),
		Type:   optFilter(params.Type.Set, string(params.Type.Value)),
		Region: optFilter(params.Region.Set, string(params.Region.Value)),
		Q:      optFilter(params.Q.Set, params.Q.Value),
		Limit:  size,
		Offset: (page - 1) * size,
	})
	if err != nil {
		return nil, err
	}

	items := make([]oas.PRStunt, 0, len(rows))
	for _, p := range rows {
		items = append(items, mapPRStunt(p))
	}
	return &oas.PRStuntList{Items: items, Total: total, Page: page, PageSize: size}, nil
}

// GetPrStunt implémente GET /v1/pr-stunts/{id} : le PR Stunt demandé, ou un 404
// RFC 9457.
func (h *Handler) GetPrStunt(ctx context.Context, params oas.GetPrStuntParams) (oas.GetPrStuntRes, error) {
	p, err := h.store.GetPRStunt(ctx, params.ID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, errNotFound("no pr stunt with the given id")
	}
	stunt := mapPRStunt(*p)
	return &stunt, nil
}

// ListEvents implémente GET /v1/events.
func (h *Handler) ListEvents(ctx context.Context, params oas.ListEventsParams) (oas.ListEventsRes, error) {
	page, size := pageParams(params.Page, params.PageSize)
	rows, total, err := h.store.ListEvents(ctx, store.GeoFilter{
		Game:   string(params.Game),
		Type:   optFilter(params.Type.Set, string(params.Type.Value)),
		Region: optFilter(params.Region.Set, string(params.Region.Value)),
		Q:      optFilter(params.Q.Set, params.Q.Value),
		Limit:  size,
		Offset: (page - 1) * size,
	})
	if err != nil {
		return nil, err
	}

	items := make([]oas.Event, 0, len(rows))
	for _, e := range rows {
		items = append(items, mapEvent(e))
	}
	return &oas.EventList{Items: items, Total: total, Page: page, PageSize: size}, nil
}

// GetEvent implémente GET /v1/events/{id} : l'événement demandé, ou un 404
// RFC 9457.
func (h *Handler) GetEvent(ctx context.Context, params oas.GetEventParams) (oas.GetEventRes, error) {
	e, err := h.store.GetEvent(ctx, params.ID)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, errNotFound("no event with the given id")
	}
	event := mapEvent(*e)
	return &event, nil
}

// mapTrack projette la vue DB d'un tracé sur le modèle du contrat.
func mapTrack(t store.Track) oas.Track {
	return oas.Track{
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
	}
}

// mapPRStunt projette la vue DB d'un PR Stunt sur le modèle du contrat.
func mapPRStunt(p store.PRStunt) oas.PRStunt {
	return oas.PRStunt{
		ID:          p.ID,
		Game:        oas.Game(p.Game),
		Type:        oas.PRStuntType(p.Type),
		Name:        p.Name,
		Region:      optRegion(p.Region),
		Lat:         optFloat(p.Lat),
		Lng:         optFloat(p.Lng),
		TargetScore: optInt(p.TargetScore),
	}
}

// mapEvent projette la vue DB d'un événement sur le modèle du contrat.
func mapEvent(e store.Event) oas.Event {
	return oas.Event{
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
	}
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

// optTimeFilter convertit un paramètre de filtre OptDateTime en *time.Time
// (nil si absent).
func optTimeFilter(p oas.OptDateTime) *time.Time {
	if v, ok := p.Get(); ok {
		return &v
	}
	return nil
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
