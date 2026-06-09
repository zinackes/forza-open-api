// Référence DLC / extensions : lecture pour GET /v1/dlc-packs. Sources propres
// (annonces forza.net, wiki Fandom). Requêtes paramétrées only.
package store

import (
	"context"
	"fmt"
	"time"
)

// DlcPack est la vue DB d'un pack DLC (snake_case). Champs absents → pointeur nil.
type DlcPack struct {
	ID           string
	Game         string
	Name         string
	Kind         string
	ReleasedAt   *time.Time
	Description  *string
	Source       *string
	LastVerified *time.Time
}

// ListDlcPacks renvoie les packs DLC d'un jeu, les plus récents d'abord. Les
// packs annoncés mais pas encore sortis (released_at NULL) viennent en dernier.
func (s *Store) ListDlcPacks(ctx context.Context, game string) ([]DlcPack, error) {
	const q = `
SELECT id, game, name, kind, released_at, description, source, last_verified
FROM dlc_packs
WHERE game = $1
ORDER BY released_at DESC NULLS LAST, name`
	rows, err := s.DB.Query(ctx, q, game)
	if err != nil {
		return nil, fmt.Errorf("query dlc_packs: %w", err)
	}
	defer rows.Close()

	out := make([]DlcPack, 0)
	for rows.Next() {
		var d DlcPack
		if err := rows.Scan(&d.ID, &d.Game, &d.Name, &d.Kind, &d.ReleasedAt,
			&d.Description, &d.Source, &d.LastVerified); err != nil {
			return nil, fmt.Errorf("scan dlc_pack: %w", err)
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dlc_packs: %w", err)
	}
	return out, nil
}
