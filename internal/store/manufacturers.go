// Constructeurs : lecture pour GET /v1/manufacturers ; écriture (upsert) par
// l'ingestion cars (cmd/seed), qui dérive make + code pays de la page liste du
// wiki. car_count est un agrégat (GROUP BY) sur cars joint par make. Requêtes
// paramétrées only.
package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
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

// UpsertManufacturers insère ou met à jour des constructeurs de façon
// idempotente (ON CONFLICT sur la clé naturelle game+name). Rejouable sans
// doublon. Appelé par l'ingestion (cmd/seed cars), jamais un handler. COALESCE :
// un re-run dont la source a perdu le code pays ne détruit pas la valeur déjà
// acquise (précision > exhaustivité) ; le garde IS DISTINCT FROM évite de
// toucher une ligne inchangée. pgx.Batch dans une transaction, comme UpsertCars.
func (s *Store) UpsertManufacturers(ctx context.Context, mans []Manufacturer) error {
	if len(mans) == 0 {
		return nil
	}
	const q = `
INSERT INTO manufacturers (game, name, country)
VALUES ($1, $2, $3)
ON CONFLICT (game, name) DO UPDATE SET
    country = COALESCE(EXCLUDED.country, manufacturers.country)
WHERE COALESCE(EXCLUDED.country, manufacturers.country)
      IS DISTINCT FROM manufacturers.country`
	batch := &pgx.Batch{}
	for _, m := range mans {
		batch.Queue(q, m.Game, m.Name, m.Country)
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin upsert manufacturers: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op après Commit

	br := tx.SendBatch(ctx, batch)
	for _, m := range mans {
		if _, err := br.Exec(); err != nil {
			_ = br.Close()
			return fmt.Errorf("upsert manufacturer %s/%s: %w", m.Game, m.Name, err)
		}
	}
	if err := br.Close(); err != nil {
		return fmt.Errorf("close upsert manufacturers batch: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit upsert manufacturers: %w", err)
	}
	return nil
}
