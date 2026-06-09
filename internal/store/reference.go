// Facettes de référence : agrégats (enums + compteurs) pour GET /v1/reference.
// Lecture seule, requêtes paramétrées only. Les facettes énumérées (classes,
// drivetrains) sont renvoyées dans l'ordre canonique avec count 0 inclus ; les
// facettes libres (body_type, category, pays) seulement pour les valeurs
// présentes, plus fréquentes d'abord. La liste des jeux est globale (hors filtre
// game) pour amorcer un sélecteur côté client.
package store

import (
	"context"
	"fmt"
	"sort"
)

// Ordres canoniques des facettes énumérées (cf. CHECK cars.class / cars.drivetrain
// et les enums CarClass / Drivetrain du contrat). R est en dernier (track-focused,
// FH6). Renvoyés en entier, count 0 compris, pour piloter des filtres complets.
var (
	canonicalClasses     = []string{"D", "C", "B", "A", "S1", "S2", "X", "R"}
	canonicalDrivetrains = []string{"FWD", "RWD", "AWD"}
)

// RefCount est une valeur de facette et son nombre d'occurrences.
type RefCount struct {
	Code  string
	Count int64
}

// GameCount est un jeu disponible et ses volumes (voitures, séries).
type GameCount struct {
	Code        string
	CountCars   int64
	CountSeries int64
}

// Reference agrège les facettes de référence d'un jeu. classes/drivetrains/
// bodyTypes/countries/categories sont scopés au jeu demandé ; games est global.
type Reference struct {
	Classes     []RefCount
	Drivetrains []RefCount
	BodyTypes   []RefCount
	Countries   []RefCount
	Categories  []RefCount
	Games       []GameCount
}

// GetReference renvoie les facettes de référence pour un jeu. Jeu inconnu →
// classes/drivetrains présents (count 0) et le reste vide (pas d'erreur).
func (s *Store) GetReference(ctx context.Context, game string) (Reference, error) {
	carFacets, err := s.carFacetCounts(ctx, game)
	if err != nil {
		return Reference{}, err
	}

	countries, err := s.countryCounts(ctx, game)
	if err != nil {
		return Reference{}, err
	}

	games, err := s.gameCounts(ctx)
	if err != nil {
		return Reference{}, err
	}

	return Reference{
		Classes:     expandCanonical(canonicalClasses, carFacets["class"]),
		Drivetrains: expandCanonical(canonicalDrivetrains, carFacets["drivetrain"]),
		BodyTypes:   presentByCount(carFacets["body_type"]),
		Categories:  presentByCount(carFacets["category"]),
		Countries:   presentByCount(countries),
		Games:       games,
	}, nil
}

// carFacetCounts compte, pour un jeu, les voitures par class/drivetrain/body_type/
// category en une seule requête (UNION ALL). Renvoie facet → code → count ; les
// valeurs NULL sont exclues (jamais de facette « inventée »).
func (s *Store) carFacetCounts(ctx context.Context, game string) (map[string]map[string]int64, error) {
	const q = `
SELECT 'class'      AS facet, class      AS code, count(*) AS n FROM cars WHERE game = $1 AND class      IS NOT NULL GROUP BY class
UNION ALL
SELECT 'drivetrain' AS facet, drivetrain AS code, count(*) AS n FROM cars WHERE game = $1 AND drivetrain IS NOT NULL GROUP BY drivetrain
UNION ALL
SELECT 'body_type'  AS facet, body_type  AS code, count(*) AS n FROM cars WHERE game = $1 AND body_type  IS NOT NULL GROUP BY body_type
UNION ALL
SELECT 'category'   AS facet, category   AS code, count(*) AS n FROM cars WHERE game = $1 AND category   IS NOT NULL GROUP BY category`
	rows, err := s.DB.Query(ctx, q, game)
	if err != nil {
		return nil, fmt.Errorf("query car facets: %w", err)
	}
	defer rows.Close()

	out := map[string]map[string]int64{}
	for rows.Next() {
		var facet, code string
		var n int64
		if err := rows.Scan(&facet, &code, &n); err != nil {
			return nil, fmt.Errorf("scan car facet: %w", err)
		}
		if out[facet] == nil {
			out[facet] = map[string]int64{}
		}
		out[facet][code] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate car facets: %w", err)
	}
	return out, nil
}

// countryCounts compte les constructeurs par pays pour un jeu (pays NULL exclus).
func (s *Store) countryCounts(ctx context.Context, game string) (map[string]int64, error) {
	const q = `
SELECT country, count(*) AS n
FROM manufacturers
WHERE game = $1 AND country IS NOT NULL
GROUP BY country`
	rows, err := s.DB.Query(ctx, q, game)
	if err != nil {
		return nil, fmt.Errorf("query country counts: %w", err)
	}
	defer rows.Close()

	out := map[string]int64{}
	for rows.Next() {
		var country string
		var n int64
		if err := rows.Scan(&country, &n); err != nil {
			return nil, fmt.Errorf("scan country count: %w", err)
		}
		out[country] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate country counts: %w", err)
	}
	return out, nil
}

// gameCounts renvoie tous les jeux présents (cars ∪ series) avec leurs volumes.
// Global : indépendant du filtre game (amorce un sélecteur de jeu côté client).
func (s *Store) gameCounts(ctx context.Context) ([]GameCount, error) {
	const q = `
SELECT COALESCE(c.game, s.game)  AS game,
       COALESCE(c.n, 0)          AS count_cars,
       COALESCE(s.n, 0)          AS count_series
FROM      (SELECT game, count(*) AS n FROM cars   GROUP BY game) c
FULL JOIN (SELECT game, count(*) AS n FROM series GROUP BY game) s ON s.game = c.game
ORDER BY game`
	rows, err := s.DB.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query game counts: %w", err)
	}
	defer rows.Close()

	out := make([]GameCount, 0)
	for rows.Next() {
		var g GameCount
		if err := rows.Scan(&g.Code, &g.CountCars, &g.CountSeries); err != nil {
			return nil, fmt.Errorf("scan game count: %w", err)
		}
		out = append(out, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate game counts: %w", err)
	}
	return out, nil
}

// expandCanonical projette les compteurs sur l'ordre canonique d'une facette
// énumérée : chaque code valide est renvoyé, count 0 s'il est absent des données.
func expandCanonical(order []string, counts map[string]int64) []RefCount {
	out := make([]RefCount, 0, len(order))
	for _, code := range order {
		out = append(out, RefCount{Code: code, Count: counts[code]})
	}
	return out
}

// presentByCount renvoie les facettes libres présentes, triées par count
// décroissant puis code croissant (ordre stable et déterministe).
func presentByCount(counts map[string]int64) []RefCount {
	out := make([]RefCount, 0, len(counts))
	for code, n := range counts {
		out = append(out, RefCount{Code: code, Count: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Code < out[j].Code
	})
	return out
}
