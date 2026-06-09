// Voitures cachées FH6 : Barn Finds & Treasure Cars (2 mécaniques distinctes).
// Lectures paginées pour les handlers. Requêtes paramétrées only ; lat/lng
// castés en float8 au SELECT pour un scan direct en float64. Coords NULL → nil.
package store

import (
	"context"
	"fmt"
	"time"
)

// HiddenCarFilter porte les filtres partagés par barn_finds / treasure_cars.
type HiddenCarFilter struct {
	Game   string
	Region *string // nil = pas de filtre
	Limit  int
	Offset int
}

// BarnFind est la vue DB d'un Barn Find. Champs absents → pointeur nil.
type BarnFind struct {
	ID                     string
	CarID                  string
	Game                   string
	Region                 *string
	SearchZoneCenterLat    *float64
	SearchZoneCenterLng    *float64
	SearchZoneRadiusM      *int
	PrerequisiteStampLevel *int
	RestorationTimeH       *int
	Source                 *string
	LastVerified           *time.Time
}

// TreasureCar est la vue DB d'une Treasure Car. Champs absents → pointeur nil.
type TreasureCar struct {
	ID               string
	CarID            string
	Game             string
	Region           *string
	PostcardClueText *string
	LocationLat      *float64
	LocationLng      *float64
	Source           *string
	LastVerified     *time.Time
}

// ListBarnFinds renvoie une page de Barn Finds filtrés + le total (count fenêtré).
// Ordre par niveau de stamp requis (progression Discover Japan), puis id.
func (s *Store) ListBarnFinds(ctx context.Context, f HiddenCarFilter) ([]BarnFind, int64, error) {
	const q = `
SELECT id, car_id, game, region,
       search_zone_center_lat::float8, search_zone_center_lng::float8,
       search_zone_radius_m, prerequisite_stamp_level, restoration_time_h,
       source, last_verified, count(*) OVER() AS total
FROM barn_finds
WHERE game = $1
  AND ($2::text IS NULL OR region = $2)
ORDER BY prerequisite_stamp_level NULLS LAST, id
LIMIT $3 OFFSET $4`
	rows, err := s.DB.Query(ctx, q, f.Game, f.Region, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query barn_finds: %w", err)
	}
	defer rows.Close()

	out := make([]BarnFind, 0, f.Limit)
	var total int64
	for rows.Next() {
		var b BarnFind
		if err := rows.Scan(&b.ID, &b.CarID, &b.Game, &b.Region,
			&b.SearchZoneCenterLat, &b.SearchZoneCenterLng, &b.SearchZoneRadiusM,
			&b.PrerequisiteStampLevel, &b.RestorationTimeH, &b.Source,
			&b.LastVerified, &total); err != nil {
			return nil, 0, fmt.Errorf("scan barn_find: %w", err)
		}
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate barn_finds: %w", err)
	}
	return out, total, nil
}

// ListTreasureCars renvoie une page de Treasure Cars filtrées + le total.
func (s *Store) ListTreasureCars(ctx context.Context, f HiddenCarFilter) ([]TreasureCar, int64, error) {
	const q = `
SELECT id, car_id, game, region, postcard_clue_text,
       location_lat::float8, location_lng::float8,
       source, last_verified, count(*) OVER() AS total
FROM treasure_cars
WHERE game = $1
  AND ($2::text IS NULL OR region = $2)
ORDER BY region NULLS LAST, id
LIMIT $3 OFFSET $4`
	rows, err := s.DB.Query(ctx, q, f.Game, f.Region, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query treasure_cars: %w", err)
	}
	defer rows.Close()

	out := make([]TreasureCar, 0, f.Limit)
	var total int64
	for rows.Next() {
		var t TreasureCar
		if err := rows.Scan(&t.ID, &t.CarID, &t.Game, &t.Region,
			&t.PostcardClueText, &t.LocationLat, &t.LocationLng,
			&t.Source, &t.LastVerified, &total); err != nil {
			return nil, 0, fmt.Errorf("scan treasure_car: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate treasure_cars: %w", err)
	}
	return out, total, nil
}
