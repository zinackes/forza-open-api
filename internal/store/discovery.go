// Contenus Discovery FH6 : Stories (missions narratives) & Tours (visites
// guidées). Lectures paginées pour les handlers ; upserts idempotents pour
// l'ingestion. Requêtes paramétrées only. Champs non sourcés → nil.
package store

import (
	"context"
	"fmt"
	"time"
)

// DiscoveryFilter porte les filtres partagés par stories / tours.
type DiscoveryFilter struct {
	Game   string
	Region *string // nil = pas de filtre
	Limit  int
	Offset int
}

// Story est la vue DB d'une Story. Champs absents → pointeur nil.
type Story struct {
	ID                string
	Game              string
	Name              string
	Region            *string
	Description       *string
	ChaptersCount     *int
	RewardDescription *string
	Source            *string
	LastVerified      *time.Time
}

// Tour est la vue DB d'un Tour. Champs absents → pointeur nil.
type Tour struct {
	ID           string
	Game         string
	Name         string
	Region       *string
	Description  *string
	Source       *string
	LastVerified *time.Time
}

// ListStories renvoie une page de Stories filtrées + le total (COUNT séparé :
// total exact même hors borne). fromWhere est partagé COUNT/SELECT.
func (s *Store) ListStories(ctx context.Context, f DiscoveryFilter) ([]Story, int64, error) {
	const fromWhere = `
FROM stories
WHERE game = $1
  AND ($2::text IS NULL OR region = $2)`

	var total int64
	if err := s.DB.QueryRow(ctx, `SELECT count(*)`+fromWhere,
		f.Game, f.Region).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count stories: %w", err)
	}

	const q = `
SELECT id, game, name, region, description, chapters_count, reward_description,
       source, last_verified` + fromWhere + `
ORDER BY name, id
LIMIT $3 OFFSET $4`
	rows, err := s.DB.Query(ctx, q, f.Game, f.Region, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query stories: %w", err)
	}
	defer rows.Close()

	out := make([]Story, 0, f.Limit)
	for rows.Next() {
		var st Story
		if err := rows.Scan(&st.ID, &st.Game, &st.Name, &st.Region, &st.Description,
			&st.ChaptersCount, &st.RewardDescription, &st.Source,
			&st.LastVerified); err != nil {
			return nil, 0, fmt.Errorf("scan story: %w", err)
		}
		out = append(out, st)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate stories: %w", err)
	}
	return out, total, nil
}

// ListTours renvoie une page de Tours filtrés + le total (COUNT séparé : total
// exact même hors borne). fromWhere est partagé COUNT/SELECT.
func (s *Store) ListTours(ctx context.Context, f DiscoveryFilter) ([]Tour, int64, error) {
	const fromWhere = `
FROM tours
WHERE game = $1
  AND ($2::text IS NULL OR region = $2)`

	var total int64
	if err := s.DB.QueryRow(ctx, `SELECT count(*)`+fromWhere,
		f.Game, f.Region).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count tours: %w", err)
	}

	const q = `
SELECT id, game, name, region, description, source, last_verified` + fromWhere + `
ORDER BY name, id
LIMIT $3 OFFSET $4`
	rows, err := s.DB.Query(ctx, q, f.Game, f.Region, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query tours: %w", err)
	}
	defer rows.Close()

	out := make([]Tour, 0, f.Limit)
	for rows.Next() {
		var t Tour
		if err := rows.Scan(&t.ID, &t.Game, &t.Name, &t.Region, &t.Description,
			&t.Source, &t.LastVerified); err != nil {
			return nil, 0, fmt.Errorf("scan tour: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate tours: %w", err)
	}
	return out, total, nil
}

// UpsertStories insère ou met à jour des Stories de façon idempotente
// (ON CONFLICT sur l'id). Appelé par l'ingestion, jamais un handler.
func (s *Store) UpsertStories(ctx context.Context, stories []Story) error {
	const q = `
INSERT INTO stories (id, game, name, region, description, chapters_count,
                     reward_description, source, last_verified)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
ON CONFLICT (id) DO UPDATE SET
    game = EXCLUDED.game, name = EXCLUDED.name, region = EXCLUDED.region,
    description = EXCLUDED.description, chapters_count = EXCLUDED.chapters_count,
    reward_description = EXCLUDED.reward_description, source = EXCLUDED.source,
    last_verified = EXCLUDED.last_verified`
	for _, st := range stories {
		if _, err := s.DB.Exec(ctx, q, st.ID, st.Game, st.Name, st.Region,
			st.Description, st.ChaptersCount, st.RewardDescription, st.Source,
			st.LastVerified); err != nil {
			return fmt.Errorf("upsert story %s: %w", st.ID, err)
		}
	}
	return nil
}

// UpsertTours insère ou met à jour des Tours de façon idempotente (ON CONFLICT
// sur l'id). Appelé par l'ingestion, jamais un handler.
func (s *Store) UpsertTours(ctx context.Context, tours []Tour) error {
	const q = `
INSERT INTO tours (id, game, name, region, description, source, last_verified)
VALUES ($1,$2,$3,$4,$5,$6,$7)
ON CONFLICT (id) DO UPDATE SET
    game = EXCLUDED.game, name = EXCLUDED.name, region = EXCLUDED.region,
    description = EXCLUDED.description, source = EXCLUDED.source,
    last_verified = EXCLUDED.last_verified`
	for _, t := range tours {
		if _, err := s.DB.Exec(ctx, q, t.ID, t.Game, t.Name, t.Region,
			t.Description, t.Source, t.LastVerified); err != nil {
			return fmt.Errorf("upsert tour %s: %w", t.ID, err)
		}
	}
	return nil
}
