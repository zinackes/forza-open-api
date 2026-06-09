// Constructeurs : lecture pour GET /v1/manufacturers. car_count est un agrégat
// (GROUP BY) sur cars joint par make. Requêtes paramétrées only.
package store

import (
	"context"
	"fmt"
)

// Manufacturer est la vue DB d'un constructeur (snake_case). Country absent → nil.
// CarCount est le nombre de voitures du jeu portant ce make (0 si aucune).
type Manufacturer struct {
	Game     string
	Name     string
	Country  *string
	CarCount int64
}

// ListManufacturers renvoie les constructeurs d'un jeu avec leur nombre de
// voitures, par ordre alphabétique. La table manufacturers est la source des noms
// et pays ; le compte vient d'un LEFT JOIN sur cars (par game + make) agrégé en
// GROUP BY, si bien qu'un constructeur sans voiture sourcée remonte avec count 0.
func (s *Store) ListManufacturers(ctx context.Context, game string) ([]Manufacturer, error) {
	const q = `
SELECT m.game, m.name, m.country, count(c.id) AS car_count
FROM manufacturers m
LEFT JOIN cars c ON c.game = m.game AND c.make = m.name
WHERE m.game = $1
GROUP BY m.game, m.name, m.country
ORDER BY m.name`
	rows, err := s.DB.Query(ctx, q, game)
	if err != nil {
		return nil, fmt.Errorf("query manufacturers: %w", err)
	}
	defer rows.Close()

	out := make([]Manufacturer, 0)
	for rows.Next() {
		var m Manufacturer
		if err := rows.Scan(&m.Game, &m.Name, &m.Country, &m.CarCount); err != nil {
			return nil, fmt.Errorf("scan manufacturer: %w", err)
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate manufacturers: %w", err)
	}
	return out, nil
}
