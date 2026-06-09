// Catalogue voitures : lecture paginée filtrée pour GET /v1/cars. Requêtes
// paramétrées only. Mapping snake_case DB → vue camelCase du contrat côté handler.
package store

import (
	"context"
	"fmt"
	"time"
)

// CarFilter porte les filtres de GET /v1/cars. Pointeur nil = pas de filtre.
type CarFilter struct {
	Game       string
	Make       *string
	Class      *string
	PIMin      *int
	PIMax      *int
	Drivetrain *string
	Category   *string // division in-game, correspondance exacte
	Q          *string
	Dlc        *string // identifiant d'un dlc_packs : restreint aux voitures du pack
	Limit      int
	Offset     int
}

// Car est la vue DB d'une voiture (snake_case). class/pi/drivetrain sont requis
// par le contrat (NOT NULL côté catalogue) ; champs réellement optionnels → nil.
type Car struct {
	ID           string
	Game         string
	Name         string
	Make         string
	Model        *string
	Year         *int
	Class        string
	PI           int
	Drivetrain   string
	Stats        []byte // JSONB brut
	BodyType     *string
	Category     *string
	Rarity       *string
	ValueCr      *int64
	ObtainMethod *string
	ImageURL     *string
	CreatedAt    time.Time
}

// ListCars renvoie une page de voitures filtrées + le total (count fenêtré).
// Le filtre Dlc joint car_dlc (EXISTS) : seules les voitures du pack remontent.
func (s *Store) ListCars(ctx context.Context, f CarFilter) ([]Car, int64, error) {
	const q = `
SELECT c.id, c.game, c.name, c.make, c.model, c.year, c.class, c.pi,
       c.drivetrain, c.stats, c.body_type, c.category, c.rarity, c.value_cr,
       c.obtain_method, c.image_url, c.created_at, count(*) OVER() AS total
FROM cars c
WHERE c.game = $1
  AND ($2::text IS NULL OR c.make = $2)
  AND ($3::text IS NULL OR c.class = $3)
  AND ($4::int  IS NULL OR c.pi >= $4)
  AND ($5::int  IS NULL OR c.pi <= $5)
  AND ($6::text IS NULL OR c.drivetrain = $6)
  AND ($7::text IS NULL OR c.category = $7)
  AND ($8::text IS NULL OR c.name ILIKE '%' || $8 || '%' OR c.model ILIKE '%' || $8 || '%')
  AND ($9::text IS NULL OR EXISTS (
        SELECT 1 FROM car_dlc cd WHERE cd.car_id = c.id AND cd.dlc_id = $9))
ORDER BY c.name
LIMIT $10 OFFSET $11`
	rows, err := s.DB.Query(ctx, q, f.Game, f.Make, f.Class, f.PIMin, f.PIMax,
		f.Drivetrain, f.Category, f.Q, f.Dlc, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query cars: %w", err)
	}
	defer rows.Close()

	out := make([]Car, 0, f.Limit)
	var total int64
	for rows.Next() {
		var c Car
		if err := rows.Scan(&c.ID, &c.Game, &c.Name, &c.Make, &c.Model, &c.Year,
			&c.Class, &c.PI, &c.Drivetrain, &c.Stats, &c.BodyType, &c.Category, &c.Rarity,
			&c.ValueCr, &c.ObtainMethod, &c.ImageURL, &c.CreatedAt, &total); err != nil {
			return nil, 0, fmt.Errorf("scan car: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate cars: %w", err)
	}
	return out, total, nil
}
