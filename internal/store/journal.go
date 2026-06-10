// Collection Journal FH6 : paliers de progression (wristbands Horizon Festival +
// stamps Discover Japan). Lecture pour GET /v1/journal (ensemble borné, ≤ 14 par
// jeu → pas de pagination). Requêtes paramétrées only. Mapping snake_case DB →
// vue camelCase du contrat côté handler. Champs non sourcés → nil (color est NULL
// pour les stamps).
package store

import (
	"context"
	"fmt"
	"time"
)

// JournalFilter porte les filtres de ListJournalTiers.
type JournalFilter struct {
	Game  string
	Track *string // nil = pas de filtre
}

// JournalTier est la vue DB d'un palier du Collection Journal. Champs absents → nil.
type JournalTier struct {
	ID                 string
	Game               string
	Track              string
	Level              int
	Color              *string
	Name               string
	PointsRequired     *int
	RewardCarID        *string
	UnlocksDescription *string
	Source             *string
	LastVerified       *time.Time
}

// ListJournalTiers renvoie les paliers du Collection Journal d'un jeu, filtrés
// éventuellement par piste, ordonnés par piste puis niveau croissant (sens de la
// progression). Jeu inconnu ou paliers non sourcés → vide.
func (s *Store) ListJournalTiers(ctx context.Context, f JournalFilter) ([]JournalTier, error) {
	const q = `
SELECT id, game, track, level, color, name, points_required,
       reward_car_id, unlocks_description, source, last_verified
FROM journal_tiers
WHERE game = $1
  AND ($2::text IS NULL OR track = $2)
ORDER BY track, level`
	rows, err := s.DB.Query(ctx, q, f.Game, f.Track)
	if err != nil {
		return nil, fmt.Errorf("query journal_tiers: %w", err)
	}
	defer rows.Close()

	out := make([]JournalTier, 0)
	for rows.Next() {
		var j JournalTier
		if err := rows.Scan(&j.ID, &j.Game, &j.Track, &j.Level, &j.Color, &j.Name,
			&j.PointsRequired, &j.RewardCarID, &j.UnlocksDescription, &j.Source,
			&j.LastVerified); err != nil {
			return nil, fmt.Errorf("scan journal_tier: %w", err)
		}
		out = append(out, j)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate journal_tiers: %w", err)
	}
	return out, nil
}
