// Handlers Festival Playlist : GET /v1/playlist/series (liste), /series/{id}
// (détail), /current (série courante d'un jeu). Lecture seule, store paramétré.
// La liste est légère (séries sans enfants) ; le détail hydrate rewards +
// challenges. Mapping DB → contrat ; 404 RFC 9457 si la série est inconnue.
package handler

import (
	"context"

	"github.com/zinackes/forza-open-api/internal/oas"
	"github.com/zinackes/forza-open-api/internal/store"
)

// playlistCacheControl : la Festival Playlist tourne ~hebdo et is_current bascule
// à l'ingestion. Cache court côté navigateur/bord pour une fraîcheur de quelques
// minutes, mais stale-while-revalidate sert périmé jusqu'à 1 j en revalidant en
// arrière-plan (absorbe redémarrages/fenêtres d'ingestion). Couplé à l'ETag
// (ConditionalGet) + bump DATA_VERSION au refresh → revalidation gratuite (304).
const playlistCacheControl = "public, max-age=300, s-maxage=900, stale-while-revalidate=86400"

// ListSeries implémente GET /v1/playlist/series : les séries connues d'un jeu,
// les plus récentes d'abord. Sans rewards/challenges (liste légère).
func (h *Handler) ListSeries(ctx context.Context, params oas.ListSeriesParams) (oas.ListSeriesRes, error) {
	rows, err := h.store.ListSeries(ctx, string(params.Game))
	if err != nil {
		return nil, err
	}
	out := make([]oas.Series, 0, len(rows))
	for _, ser := range rows {
		out = append(out, mapSeries(ser, nil, nil))
	}
	return &oas.ListSeriesOKHeaders{
		CacheControl: oas.NewOptString(playlistCacheControl),
		Response:     out,
	}, nil
}

// GetSeries implémente GET /v1/playlist/series/{id} : la série demandée avec ses
// rewards et challenges, ou un 404 si l'identifiant est inconnu.
func (h *Handler) GetSeries(ctx context.Context, params oas.GetSeriesParams) (oas.GetSeriesRes, error) {
	ser, rewards, challenges, err := h.store.SeriesDetail(ctx, params.ID)
	if err != nil {
		return nil, err
	}
	if ser == nil {
		return nil, errNotFound("no series with the given id")
	}
	return &oas.SeriesHeaders{
		CacheControl: oas.NewOptString(playlistCacheControl),
		Response:     mapSeries(*ser, rewards, challenges),
	}, nil
}

// GetCurrentPlaylist implémente GET /v1/playlist/current : la série courante du
// jeu (is_current) avec ses rewards et challenges, ou un 404 si aucune.
func (h *Handler) GetCurrentPlaylist(ctx context.Context, params oas.GetCurrentPlaylistParams) (oas.GetCurrentPlaylistRes, error) {
	ser, rewards, challenges, err := h.store.CurrentSeries(ctx, string(params.Game))
	if err != nil {
		return nil, err
	}
	if ser == nil {
		return nil, errNotFound("no current playlist for the given game")
	}
	return &oas.SeriesHeaders{
		CacheControl: oas.NewOptString(playlistCacheControl),
		Response:     mapSeries(*ser, rewards, challenges),
	}, nil
}

// mapSeries projette la vue DB d'une série (+ ses enfants) sur le modèle du
// contrat. rewards/challenges peuvent être nil (liste) : la série est alors
// renvoyée avec des tableaux vides.
func mapSeries(ser store.Series, rewards []store.Reward, challenges []store.Challenge) oas.Series {
	out := oas.Series{
		ID:         ser.ID,
		Game:       oas.Game(ser.Game),
		Series:     ser.Series,
		Name:       derefStr(ser.Name),
		Season:     optString(ser.Season),
		Week:       oas.NewOptInt(ser.Week),
		IsCurrent:  ser.IsCurrent,
		Rewards:    make([]oas.Reward, 0, len(rewards)),
		Challenges: make([]oas.Challenge, 0, len(challenges)),
	}
	if !ser.StartsAt.IsZero() {
		out.StartsAt = oas.NewOptDateTime(ser.StartsAt)
	}
	if !ser.EndsAt.IsZero() {
		out.EndsAt = oas.NewOptDateTime(ser.EndsAt)
	}
	for _, r := range rewards {
		out.Rewards = append(out.Rewards, oas.Reward{
			ID:        r.ID,
			AtPercent: optInt(r.AtPercent),
			Type:      r.Type,
			Item:      r.Item,
		})
	}
	for _, c := range challenges {
		out.Challenges = append(out.Challenges, oas.Challenge{
			ID:          c.ID,
			Scope:       c.Scope,
			Name:        c.Name,
			Requirement: derefStr(c.Requirement),
			Reward:      optString(c.Reward),
			ExpiresAt:   optTime(c.ExpiresAt),
		})
	}
	return out
}

// derefStr rend la valeur d'un *string (vide si nil) pour un champ de contrat
// requis dont la source DB est nullable (name, requirement).
func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
