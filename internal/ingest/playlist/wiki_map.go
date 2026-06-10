package playlist

// Mapping wiki → modèle store (series/rewards/challenges) et orchestration du
// backfill. Game-agnostic : un wikiGame porte les spécificités par jeu (préfixe de
// titre et de template). Une ligne `series` par saison/semaine (Summer=W1 … Spring=
// W4), comme l'id stable « fh6-s01w2 » de l'ingestion courante.

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zinackes/forza-open-api/internal/store"
)

// wikiGame porte les spécificités d'un jeu sur le wiki Fandom : préfixe des pages
// (« Forza Horizon 6 ») et des templates (« FH6 » → FH6FPEvent, FH6FPReward,
// InfoboxFH6FestivalPlaylist).
type wikiGame struct {
	game        string
	titlePrefix string
	tmplPrefix  string
}

var wikiGames = map[string]wikiGame{
	"fh6": {game: "fh6", titlePrefix: "Forza Horizon 6", tmplPrefix: "FH6"},
	"fh5": {game: "fh5", titlePrefix: "Forza Horizon 5", tmplPrefix: "FH5"},
}

// seasonOrder est l'ordre des saisons d'une série FH (Summer démarre la série).
// L'indice (1-based) est le numéro de semaine.
var seasonOrder = []string{"summer", "autumn", "winter", "spring"}

func weekOf(season string) int {
	for i, s := range seasonOrder {
		if s == strings.ToLower(season) {
			return i + 1
		}
	}
	return 0
}

// Reset hebdomadaire documenté sur la page Festival Playlist du wiki : « each reset
// at 14:30 (UTC) every Thursday ». Le wiki ne donne que la date → on l'horodate à
// ce reset (sourcé, pas inventé) plutôt que d'inventer une heure ou de la laisser
// à minuit.
const (
	resetHour = 14
	resetMin  = 30
)

// SeasonUpsert est une saison prête pour store.UpsertSeries.
type SeasonUpsert struct {
	Series     store.Series
	Rewards    []store.Reward
	Challenges []store.Challenge
}

// BackfillPlaylist découvre, récupère et mappe tout l'historique Festival Playlist
// d'un jeu depuis le wiki, en saisons prêtes à upserter. now sert à décider la
// série courante (now ∈ [start,end]). Idempotent côté store (upsert sur l'id stable).
func BackfillPlaylist(ctx context.Context, game string, now time.Time) ([]SeasonUpsert, error) {
	g, ok := wikiGames[strings.ToLower(game)]
	if !ok {
		return nil, fmt.Errorf("playlist backfill: jeu non supporté %q (fh5, fh6)", game)
	}
	pages, err := discoverSeries(ctx, g)
	if err != nil {
		return nil, fmt.Errorf("discover series (%s): %w", game, err)
	}

	titles := make([]string, 0, len(pages)*5)
	for _, ps := range pages {
		titles = append(titles, ps.seriesTitle)
		for _, st := range ps.seasonTitle {
			titles = append(titles, st)
		}
	}
	wikitext, err := fetchWikitexts(ctx, titles)
	if err != nil {
		return nil, fmt.Errorf("fetch wikitext (%s): %w", game, err)
	}
	return g.buildBackfill(pages, wikitext, now), nil
}

// buildBackfill transforme le wikitext récupéré (titre → contenu) en saisons à
// upserter. Pure (aucun réseau) → testable sur fixtures.
func (g wikiGame) buildBackfill(pages []seriesPageset, wikitext map[string]string, now time.Time) []SeasonUpsert {
	var out []SeasonUpsert
	for _, ps := range pages {
		seriesWT, ok := wikitext[ps.seriesTitle]
		if !ok {
			continue
		}
		si := g.parseSeries(seriesWT)
		if si.series == 0 {
			si.series = ps.series // fallback : numéro depuis le titre
		}
		for _, season := range seasonOrder {
			title, ok := ps.seasonTitle[season]
			if !ok {
				continue
			}
			swt, ok := wikitext[title]
			if !ok {
				continue
			}
			if u, ok := g.buildSeason(si, season, g.parseSeason(swt), now); ok {
				out = append(out, u)
			}
		}
	}
	return out
}

// buildSeason assemble une saison (identité série + détail saison) en SeasonUpsert.
// ok=false si la saison n'a pas de date de début exploitable (on n'insère pas une
// ligne sans repère temporel : précision > exhaustivité).
func (g wikiGame) buildSeason(si seriesInfo, season string, sd seasonDetail, now time.Time) (SeasonUpsert, bool) {
	week := weekOf(season)
	if week == 0 {
		return SeasonUpsert{}, false
	}
	starts := parseWikiDate(sd.start)
	ends := parseWikiDate(sd.end)
	if starts.IsZero() {
		return SeasonUpsert{}, false
	}

	id := fmt.Sprintf("%s-s%02dw%d", g.game, si.series, week)
	ser := store.Series{
		ID:        id,
		Game:      g.game,
		Series:    si.series,
		Name:      strPtr(si.name),
		Season:    strPtr(titleCase(season)),
		Week:      week,
		StartsAt:  starts,
		EndsAt:    ends,
		IsCurrent: !starts.IsZero() && !ends.IsZero() && !starts.After(now) && !ends.Before(now),
	}

	// Récompenses : paliers série (répétés sur chaque semaine, comme la source
	// forum) puis paliers saison. Ids déterministes (rw%02d) → upsert idempotent.
	rewards := make([]store.Reward, 0, len(si.rewards)+len(sd.rewards))
	add := func(r parsedReward) {
		rewards = append(rewards, store.Reward{
			ID: fmt.Sprintf("%s-rw%02d", id, len(rewards)+1), SeriesID: id,
			AtPercent: r.AtPercent, Type: r.Type, Item: r.Item,
		})
	}
	for _, r := range si.rewards {
		add(r)
	}
	for _, r := range sd.rewards {
		add(r)
	}

	// Défis : expires_at = fin de saison pour weekly/seasonal/online/monthly (ils
	// expirent au changement de saison), NULL pour daily (prolongés 7 j).
	challenges := make([]store.Challenge, 0, len(sd.challenges))
	for i, c := range sd.challenges {
		var expires *time.Time
		if c.Scope != "daily" && !ends.IsZero() {
			e := ends
			expires = &e
		}
		challenges = append(challenges, store.Challenge{
			ID: fmt.Sprintf("%s-ch%02d", id, i+1), SeriesID: id,
			Scope: c.Scope, Name: c.Name,
			Requirement: strPtr(c.Requirement), Reward: strPtr(c.Reward),
			ExpiresAt: expires,
		})
	}

	return SeasonUpsert{Series: ser, Rewards: rewards, Challenges: challenges}, true
}

// parseWikiDate lit une date wiki (« May 28, 2026 » ou « December 09 2021 ») et
// l'horodate au reset hebdomadaire (14:30 UTC). Zéro si illisible.
func parseWikiDate(s string) time.Time {
	s = strings.TrimSpace(s)
	for _, layout := range []string{"January 2, 2006", "January 2 2006"} {
		if t, err := time.Parse(layout, s); err == nil {
			return time.Date(t.Year(), t.Month(), t.Day(), resetHour, resetMin, 0, 0, time.UTC)
		}
	}
	return time.Time{}
}

func titleCase(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
