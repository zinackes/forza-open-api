// Package forzathon ingère la rotation hebdomadaire du Forzathon Shop FH (objets
// achetables contre des Forza Points). Sources propres : wiki Fandom (primaire,
// table wikitext) + corroboration forza.net/forums (cf. fetch.go). Orchestration
// partagée par le scheduler hebdomadaire (cmd/scheduler). Idempotente et rejouable :
// l'upsert porte sur la clé naturelle (game, week_start, name) côté store. La
// fenêtre de rotation est dérivée du reset hebdo (déterministe), pas scrappée.
package forzathon

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/zinackes/forza-open-api/internal/store"
)

// Result résume une passe IngestCurrent (pour le log de l'appelant et CheckRun).
type Result struct {
	Game          string
	WeekStart     time.Time
	WeekEnd       time.Time
	Items         int      // objets ingérés
	ItemsWithCost int      // objets dont le coût FP a été parsé
	Sources       []string // sources ayant confirmé la rotation (wiki + corroborations)
}

// IngestCurrent exécute une passe complète d'ingestion de la rotation courante
// d'un jeu : dérive la fenêtre (reset hebdo), récupère + parse la page wiki
// (primaire), corrobore via forza.net/forums (best-effort), normalise et upsert.
// Le réseau est borné par le ctx fourni. Idempotent et rejouable sans doublon.
func IngestCurrent(ctx context.Context, st *store.Store, game string, now time.Time, logger *slog.Logger) (Result, error) {
	start, end := weekWindow(now)

	wikitext, err := FetchWikiShop(ctx, game)
	if err != nil {
		return Result{}, fmt.Errorf("fetch wiki forzathon (%s): %w", game, err)
	}
	raw := parseShopTable(wikitext)

	items := normalize(game, start, end, raw, now)
	if err := st.UpsertForzathonRotation(ctx, items); err != nil {
		return Result{}, fmt.Errorf("upsert forzathon (%s): %w", game, err)
	}

	sources := []string{"wiki"}
	var logf func(string, ...any)
	if logger != nil {
		logf = func(msg string, kv ...any) { logger.Warn(msg, kv...) }
	}
	sources = append(sources, corroborate(ctx, logf)...)

	withCost := 0
	for _, it := range items {
		if it.FpCost != nil {
			withCost++
		}
	}
	return Result{
		Game: game, WeekStart: start, WeekEnd: end,
		Items: len(items), ItemsWithCost: withCost, Sources: sources,
	}, nil
}

// normalize projette les lignes parsées en objets store, datés sur la fenêtre de
// rotation et la passe (last_verified). car_id reste NULL : on ne rapproche pas le
// nom de boutique du catalogue sans correspondance fiable (précision > exhaustivité,
// jamais de lien inventé). id stable dérivé de (game, week_start, name) → idempotent.
func normalize(game string, start, end time.Time, raw []rawItem, now time.Time) []store.ForzathonShopItem {
	src := "wiki"
	out := make([]store.ForzathonShopItem, 0, len(raw))
	seen := map[string]bool{}
	for _, r := range raw {
		id := itemID(game, start, r.Name)
		if seen[id] { // doublon de nom dans la même semaine : on garde le premier
			continue
		}
		seen[id] = true
		endCopy, nowCopy := end, now
		out = append(out, store.ForzathonShopItem{
			ID:           id,
			Game:         game,
			WeekStart:    start,
			WeekEnd:      &endCopy,
			Kind:         r.Kind,
			Name:         r.Name,
			FpCost:       r.FpCost,
			Description:  r.Description,
			Source:       &src,
			LastVerified: &nowCopy,
		})
	}
	return out
}

var reNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// itemID construit un identifiant stable « <game>-<yyyymmdd>-<slug(nom)> », aligné
// sur la clé naturelle d'upsert (game, week_start, name).
func itemID(game string, start time.Time, name string) string {
	slug := reNonAlnum.ReplaceAllString(strings.ToLower(name), "-")
	slug = strings.Trim(slug, "-")
	return fmt.Sprintf("%s-%s-%s", game, start.Format("20060102"), slug)
}
