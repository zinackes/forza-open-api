// Métadonnées de service : volumes et fraîcheur des données pour GET /v1/meta.
// Lecture seule, requêtes paramétrées. Les timestamps de fraîcheur viennent du
// journal d'ingestion (data_changes) — seule source de « temps d'ingestion »
// uniforme entre ressources ; nil si la ressource n'a jamais été ingérée pour le
// jeu (jamais de donnée inventée).
package store

import (
	"context"
	"fmt"
	"time"
)

// MetaGame porte, pour un jeu, le nombre de voitures au catalogue et les derniers
// timestamps d'ingestion du catalogue (resource car) et de la playlist (series).
// Les pointeurs sont nil quand la ressource n'a jamais été ingérée pour le jeu.
type MetaGame struct {
	Game              string
	CarCount          int64
	CatalogUpdatedAt  *time.Time
	PlaylistUpdatedAt *time.Time
}

// Meta renvoie un MetaGame par jeu fourni (ordre préservé), même à zéro voiture
// et sans timestamp (jeu supporté mais non encore ingéré). carCount vient de la
// table cars ; les timestamps de data_changes (MAX(occurred_at) par resource).
func (s *Store) Meta(ctx context.Context, games []string) ([]MetaGame, error) {
	counts, err := s.carCountsByGame(ctx, games)
	if err != nil {
		return nil, err
	}
	ingest, err := s.ingestionTimestamps(ctx, games)
	if err != nil {
		return nil, err
	}

	out := make([]MetaGame, 0, len(games))
	for _, g := range games {
		mg := MetaGame{Game: g, CarCount: counts[g]}
		if byRes := ingest[g]; byRes != nil {
			if ts, ok := byRes["car"]; ok {
				t := ts
				mg.CatalogUpdatedAt = &t
			}
			if ts, ok := byRes["series"]; ok {
				t := ts
				mg.PlaylistUpdatedAt = &t
			}
		}
		out = append(out, mg)
	}
	return out, nil
}

// carCountsByGame compte les voitures par jeu (game → count), bornées aux jeux
// demandés. Un jeu sans voiture est absent de la map (count 0 côté appelant).
func (s *Store) carCountsByGame(ctx context.Context, games []string) (map[string]int64, error) {
	const q = `SELECT game, count(*) FROM cars WHERE game = ANY($1) GROUP BY game`
	rows, err := s.DB.Query(ctx, q, games)
	if err != nil {
		return nil, fmt.Errorf("query car counts: %w", err)
	}
	defer rows.Close()

	out := map[string]int64{}
	for rows.Next() {
		var game string
		var n int64
		if err := rows.Scan(&game, &n); err != nil {
			return nil, fmt.Errorf("scan car count: %w", err)
		}
		out[game] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate car counts: %w", err)
	}
	return out, nil
}

// ingestionTimestamps renvoie, par jeu puis par ressource (car, series), le
// dernier instant d'ingestion (MAX(occurred_at) du journal data_changes). Source
// unique et uniforme de « temps d'ingestion » : une ressource jamais ingérée
// pour un jeu est absente de la map (→ timestamp omis, jamais inventé).
func (s *Store) ingestionTimestamps(ctx context.Context, games []string) (map[string]map[string]time.Time, error) {
	const q = `
SELECT game, resource, max(occurred_at)
FROM data_changes
WHERE game = ANY($1) AND resource IN ('car','series')
GROUP BY game, resource`
	rows, err := s.DB.Query(ctx, q, games)
	if err != nil {
		return nil, fmt.Errorf("query ingestion timestamps: %w", err)
	}
	defer rows.Close()

	out := map[string]map[string]time.Time{}
	for rows.Next() {
		var game, resource string
		var ts time.Time
		if err := rows.Scan(&game, &resource, &ts); err != nil {
			return nil, fmt.Errorf("scan ingestion timestamp: %w", err)
		}
		if out[game] == nil {
			out[game] = map[string]time.Time{}
		}
		out[game][resource] = ts
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ingestion timestamps: %w", err)
	}
	return out, nil
}
