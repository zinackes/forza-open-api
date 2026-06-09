// Recherche globale multi-ressources pour GET /v1/search : un UNION ALL sur les
// ressources nommées (cars, tracks, events, pr_stunts, manufacturers, dlc_packs),
// chaque branche gardée par un booléen (kinds). Requêtes paramétrées only ; q est
// neutralisé (escapeLike) comme partout. Les index trigram portent les ILIKE.
package store

import (
	"context"
	"fmt"
)

// SearchFilter porte les paramètres de la recherche globale. Kinds vide = tous.
type SearchFilter struct {
	Game  string
	Q     string
	Kinds []string
	Limit int
}

// SearchResult est un résultat de recherche. Detail est un contexte court
// d'affichage (constructeur + classe pour une voiture, type pour un tracé…).
type SearchResult struct {
	Kind   string
	ID     string
	Game   string
	Name   string
	Detail *string
}

// searchKinds est l'ensemble des types cherchables (cf. enum SearchResultKind).
var searchKinds = []string{"car", "track", "event", "pr_stunt", "manufacturer", "dlc_pack"}

// Search exécute la recherche multi-ressources, groupée par type puis nom
// croissant, bornée par Limit. Le détail voiture concatène make/class/pi (les
// NULL se propagent : COALESCE retombe sur make seul).
func (s *Store) Search(ctx context.Context, f SearchFilter) ([]SearchResult, error) {
	enabled := map[string]bool{}
	if len(f.Kinds) == 0 {
		for _, k := range searchKinds {
			enabled[k] = true
		}
	}
	for _, k := range f.Kinds {
		enabled[k] = true
	}

	const q = `
SELECT kind, id, game, name, detail FROM (
  SELECT 'car' AS kind, id, game, name,
         make || COALESCE(' · ' || class || ' ' || pi::text, '') AS detail
  FROM cars
  WHERE $3 AND game = $1 AND (name ILIKE '%' || $2 || '%' OR model ILIKE '%' || $2 || '%')
  UNION ALL
  SELECT 'track', id, game, name, type FROM tracks
  WHERE $4 AND game = $1 AND name ILIKE '%' || $2 || '%'
  UNION ALL
  SELECT 'event', id, game, name, type FROM events
  WHERE $5 AND game = $1 AND name ILIKE '%' || $2 || '%'
  UNION ALL
  SELECT 'pr_stunt', id, game, name, type FROM pr_stunts
  WHERE $6 AND game = $1 AND name ILIKE '%' || $2 || '%'
  UNION ALL
  SELECT 'manufacturer', name, game, name, country FROM manufacturers
  WHERE $7 AND game = $1 AND name ILIKE '%' || $2 || '%'
  UNION ALL
  SELECT 'dlc_pack', id, game, name, kind FROM dlc_packs
  WHERE $8 AND game = $1 AND name ILIKE '%' || $2 || '%'
) u
ORDER BY kind, name, id
LIMIT $9`
	rows, err := s.DB.Query(ctx, q, f.Game, escapeLike(f.Q),
		enabled["car"], enabled["track"], enabled["event"], enabled["pr_stunt"],
		enabled["manufacturer"], enabled["dlc_pack"], f.Limit)
	if err != nil {
		return nil, fmt.Errorf("query search: %w", err)
	}
	defer rows.Close()

	out := make([]SearchResult, 0, f.Limit)
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.Kind, &r.ID, &r.Game, &r.Name, &r.Detail); err != nil {
			return nil, fmt.Errorf("scan search result: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate search results: %w", err)
	}
	return out, nil
}
