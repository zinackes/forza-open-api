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

// ManufacturerFilter porte les filtres de GET /v1/manufacturers.
type ManufacturerFilter struct {
	Game    string
	Country *string // correspondance exacte
	Q       *string // sous-chaîne sur le nom (escapeLike appliqué)
}

// ListManufacturers renvoie les constructeurs d'un jeu avec leur nombre de
// voitures, par ordre alphabétique. La table manufacturers est la source des noms
// et pays ; le compte vient d'un LEFT JOIN sur cars (par game + make) agrégé en
// GROUP BY, si bien qu'un constructeur sans voiture sourcée remonte avec count 0.
func (s *Store) ListManufacturers(ctx context.Context, f ManufacturerFilter) ([]Manufacturer, error) {
	qLit := f.Q
	if f.Q != nil {
		esc := escapeLike(*f.Q)
		qLit = &esc
	}

	const q = `
SELECT m.game, m.name, m.country, count(c.id) AS car_count
FROM manufacturers m
LEFT JOIN cars c ON c.game = m.game AND c.make = m.name
WHERE m.game = $1
  AND ($2::text IS NULL OR m.country = $2)
  AND ($3::text IS NULL OR m.name ILIKE '%' || $3 || '%')
GROUP BY m.game, m.name, m.country
ORDER BY m.name`
	rows, err := s.DB.Query(ctx, q, f.Game, f.Country, qLit)
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

// UpsertManufacturers enregistre des constructeurs de façon idempotente : INSERT …
// ON CONFLICT (game, name) DO UPDATE country. Donnée de RÉFÉRENCE (comme les tracks)
// → pas de journalisation data_changes. country NULL est conservé (champ non sourcé).
// Rejouable sans doublon ; n'écrit jamais car_count (agrégat calculé en lecture).
func (s *Store) UpsertManufacturers(ctx context.Context, mfrs []Manufacturer) error {
	const q = `
INSERT INTO manufacturers (game, name, country)
VALUES ($1, $2, $3)
ON CONFLICT (game, name) DO UPDATE SET country = EXCLUDED.country`
	for _, m := range mfrs {
		if _, err := s.DB.Exec(ctx, q, m.Game, m.Name, m.Country); err != nil {
			return fmt.Errorf("upsert manufacturer %s/%s: %w", m.Game, m.Name, err)
		}
	}
	return nil
}

// DistinctCarMakes renvoie les makes distincts présents dans le catalogue d'un jeu.
// Sert au contrôle de santé de l'ingestion manufacturers (rapprochement nom de
// constructeur ↔ make des voitures, qui alimente car_count en lecture). Lecture seule.
func (s *Store) DistinctCarMakes(ctx context.Context, game string) ([]string, error) {
	const q = `SELECT DISTINCT make FROM cars WHERE game = $1 AND make <> ''`
	rows, err := s.DB.Query(ctx, q, game)
	if err != nil {
		return nil, fmt.Errorf("query car makes (%s): %w", game, err)
	}
	defer rows.Close()

	out := make([]string, 0)
	for rows.Next() {
		var m string
		if err := rows.Scan(&m); err != nil {
			return nil, fmt.Errorf("scan car make: %w", err)
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate car makes: %w", err)
	}
	return out, nil
}
