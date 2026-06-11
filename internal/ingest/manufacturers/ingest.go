// Package manufacturers ingère le catalogue des CONSTRUCTEURS d'un jeu depuis une
// source PROPRE : l'API MediaWiki du wiki Fandom Forza (forza.fandom.com). Jamais le
// jeu (doctrine zéro gris). Source = la sous-catégorie PAR JEU « Category:Manufacturers
// (<TAG>) » (roster du jeu) + l'{{InfoboxMFR}} de chaque page (origin → country).
//
// Le contrat /v1/manufacturers expose name + country + car_count ; car_count est un
// agrégat calculé EN LECTURE (LEFT JOIN cars.make = manufacturers.name), pas stocké.
// L'ingestion ne pose donc que (game, name, country). Aucune donnée inventée : origin
// absent ou code non mappé → country NULL. Upsert idempotent (ON CONFLICT (game,name)).
// Orchestration partagée par cmd/seed (ponctuel) et cmd/scheduler (mensuel).
package manufacturers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/zinackes/forza-open-api/internal/store"
)

// Result résume une passe IngestManufacturers (log de l'appelant + CheckRun).
type Result struct {
	Game            string
	CategoryFound   bool           // sous-catégorie du jeu présente sur le wiki
	Members         int            // titres listés dans la catégorie
	PagesFetched    int            // pages constructeurs effectivement récupérées
	Manufacturers   int            // constructeurs upsertés (= entrées normalisées)
	CountryResolved int            // dont country mappé depuis origin
	NoOrigin        int            // sans champ origin (country NULL)
	UnknownOrigins  map[string]int // codes origin non mappés (country NULL + anomalie)
	MatchedToCars   int            // constructeurs ayant ≥1 voiture au catalogue (car_count > 0)
	Sources         []string
}

// IngestManufacturers exécute une passe complète pour un jeu : liste le roster de la
// sous-catégorie wiki, récupère chaque page (par lots), normalise (origin → country),
// puis upsert. Réseau borné par le ctx fourni. Idempotent et rejouable sans doublon.
// Une catégorie absente est un soft-skip (jeu non documenté) ; seul un échec
// réseau/HTTP ou DB est une erreur dure.
func IngestManufacturers(ctx context.Context, st *store.Store, game string, now time.Time, logger *slog.Logger) (Result, error) {
	res := Result{Game: game, UnknownOrigins: map[string]int{}}

	titles, found, err := fetchCategoryMembers(ctx, game)
	if err != nil {
		return Result{}, fmt.Errorf("fetch category members (%s): %w", game, err)
	}
	res.CategoryFound = found
	res.Members = len(titles)
	if !found {
		return res, nil
	}

	if err := sleep(ctx, politeDelay); err != nil {
		return Result{}, err
	}
	pages, err := fetchManufacturerPages(ctx, titles)
	if err != nil {
		return Result{}, fmt.Errorf("fetch manufacturer pages (%s): %w", game, err)
	}
	res.PagesFetched = len(pages)

	mfrs, rep := Normalize(game, pages)
	res.Manufacturers = len(mfrs)
	res.CountryResolved = rep.CountryResolved
	res.NoOrigin = rep.NoOrigin
	res.UnknownOrigins = rep.UnknownOrigins

	// Signal de santé : combien de constructeurs ont au moins une voiture au catalogue
	// (= car_count > 0 en lecture). 0 alors qu'on a des cars = noms divergents (rupture
	// probable du format des titres). Best-effort : une erreur de lecture n'avorte pas.
	if makes, err := st.DistinctCarMakes(ctx, game); err == nil {
		res.MatchedToCars = countMatched(mfrs, makes)
	} else if logger != nil {
		logger.Warn("manufacturers: lecture makes catalogue échouée (signal de rapprochement ignoré)", "game", game, "err", err)
	}

	if err := st.UpsertManufacturers(ctx, mfrs); err != nil {
		return Result{}, fmt.Errorf("upsert manufacturers (%s): %w", game, err)
	}

	res.Sources = []string{"wiki"}
	return res, nil
}

// countMatched compte les constructeurs dont le nom correspond exactement à un make
// du catalogue (la jointure car_count en lecture est sur make = name).
func countMatched(mfrs []store.Manufacturer, makes []string) int {
	set := make(map[string]bool, len(makes))
	for _, m := range makes {
		set[strings.TrimSpace(m)] = true
	}
	n := 0
	for _, m := range mfrs {
		if set[m.Name] {
			n++
		}
	}
	return n
}
