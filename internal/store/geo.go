// Données « carte » : tracés, PR stunts, événements. Lectures paginées pour les
// handlers ; upserts idempotents pour l'ingestion (internal/ingest). Requêtes
// paramétrées only. lat/lng castés en float8 au SELECT : scan direct en float64.
package store

import (
	"context"
	"fmt"
	"time"
)

// GeoFilter porte les filtres partagés par tracks / pr_stunts / events.
type GeoFilter struct {
	Game   string
	Type   *string // nil = pas de filtre
	Region *string // nil = pas de filtre
	Limit  int
	Offset int
}

// Track est la vue DB d'un tracé (snake_case). Champs absents → pointeur nil.
type Track struct {
	ID           string
	Game         string
	Name         string
	Type         string
	Region       *string
	LengthM      *int
	SurfaceMix   *string
	StartLat     *float64
	StartLng     *float64
	Source       *string
	LastVerified *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// PRStunt est la vue DB d'un PR stunt.
type PRStunt struct {
	ID          string
	Game        string
	Type        string
	Name        string
	Region      *string
	Lat         *float64
	Lng         *float64
	TargetScore *int
}

// Event est la vue DB d'un événement. RouteGeojson est le JSONB brut (ou nil).
type Event struct {
	ID                  string
	Game                string
	Name                string
	Type                string
	Region              *string
	StartLat            *float64
	StartLng            *float64
	EndLat              *float64
	EndLng              *float64
	RouteGeojson        []byte
	CarClassRestriction *string
	LengthM             *int
}

// ListTracks renvoie une page de tracés filtrés + le total (COUNT séparé : total
// exact même quand la page dépasse les données). fromWhere est partagé COUNT/SELECT.
func (s *Store) ListTracks(ctx context.Context, f GeoFilter) ([]Track, int64, error) {
	const fromWhere = `
FROM tracks
WHERE game = $1
  AND ($2::text IS NULL OR type = $2)
  AND ($3::text IS NULL OR region = $3)`

	var total int64
	if err := s.DB.QueryRow(ctx, `SELECT count(*)`+fromWhere,
		f.Game, f.Type, f.Region).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count tracks: %w", err)
	}

	const q = `
SELECT id, game, name, type, region, length_m, surface_mix,
       start_lat::float8, start_lng::float8, source, last_verified,
       created_at, updated_at` + fromWhere + `
ORDER BY name
LIMIT $4 OFFSET $5`
	rows, err := s.DB.Query(ctx, q, f.Game, f.Type, f.Region, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query tracks: %w", err)
	}
	defer rows.Close()

	out := make([]Track, 0, f.Limit)
	for rows.Next() {
		var t Track
		if err := rows.Scan(&t.ID, &t.Game, &t.Name, &t.Type, &t.Region,
			&t.LengthM, &t.SurfaceMix, &t.StartLat, &t.StartLng, &t.Source,
			&t.LastVerified, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan track: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate tracks: %w", err)
	}
	return out, total, nil
}

// ListPRStunts renvoie une page de PR stunts filtrés + le total (COUNT séparé :
// total exact même hors borne). fromWhere est partagé COUNT/SELECT.
func (s *Store) ListPRStunts(ctx context.Context, f GeoFilter) ([]PRStunt, int64, error) {
	const fromWhere = `
FROM pr_stunts
WHERE game = $1
  AND ($2::text IS NULL OR type = $2)
  AND ($3::text IS NULL OR region = $3)`

	var total int64
	if err := s.DB.QueryRow(ctx, `SELECT count(*)`+fromWhere,
		f.Game, f.Type, f.Region).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count pr_stunts: %w", err)
	}

	const q = `
SELECT id, game, type, name, region, lat::float8, lng::float8, target_score` + fromWhere + `
ORDER BY name
LIMIT $4 OFFSET $5`
	rows, err := s.DB.Query(ctx, q, f.Game, f.Type, f.Region, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query pr_stunts: %w", err)
	}
	defer rows.Close()

	out := make([]PRStunt, 0, f.Limit)
	for rows.Next() {
		var p PRStunt
		if err := rows.Scan(&p.ID, &p.Game, &p.Type, &p.Name, &p.Region,
			&p.Lat, &p.Lng, &p.TargetScore); err != nil {
			return nil, 0, fmt.Errorf("scan pr_stunt: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate pr_stunts: %w", err)
	}
	return out, total, nil
}

// ListEvents renvoie une page d'événements filtrés + le total (COUNT séparé :
// total exact même hors borne). fromWhere est partagé COUNT/SELECT.
func (s *Store) ListEvents(ctx context.Context, f GeoFilter) ([]Event, int64, error) {
	const fromWhere = `
FROM events
WHERE game = $1
  AND ($2::text IS NULL OR type = $2)
  AND ($3::text IS NULL OR region = $3)`

	var total int64
	if err := s.DB.QueryRow(ctx, `SELECT count(*)`+fromWhere,
		f.Game, f.Type, f.Region).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count events: %w", err)
	}

	const q = `
SELECT id, game, name, type, region,
       start_lat::float8, start_lng::float8, end_lat::float8, end_lng::float8,
       route_geojson, car_class_restriction, length_m` + fromWhere + `
ORDER BY name
LIMIT $4 OFFSET $5`
	rows, err := s.DB.Query(ctx, q, f.Game, f.Type, f.Region, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query events: %w", err)
	}
	defer rows.Close()

	out := make([]Event, 0, f.Limit)
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.Game, &e.Name, &e.Type, &e.Region,
			&e.StartLat, &e.StartLng, &e.EndLat, &e.EndLng,
			&e.RouteGeojson, &e.CarClassRestriction, &e.LengthM); err != nil {
			return nil, 0, fmt.Errorf("scan event: %w", err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate events: %w", err)
	}
	return out, total, nil
}

// UpsertTracks insère ou met à jour des tracés de façon idempotente (ON CONFLICT
// sur l'id). Rejouable sans doublon. Appelé par l'ingestion, jamais un handler.
func (s *Store) UpsertTracks(ctx context.Context, tracks []Track) error {
	const q = `
INSERT INTO tracks (id, game, name, type, region, length_m, surface_mix,
                    start_lat, start_lng, source, last_verified, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11, now())
ON CONFLICT (id) DO UPDATE SET
    game = EXCLUDED.game, name = EXCLUDED.name, type = EXCLUDED.type,
    region = EXCLUDED.region, length_m = EXCLUDED.length_m,
    surface_mix = EXCLUDED.surface_mix, start_lat = EXCLUDED.start_lat,
    start_lng = EXCLUDED.start_lng, source = EXCLUDED.source,
    last_verified = EXCLUDED.last_verified, updated_at = now()`
	for _, t := range tracks {
		if _, err := s.DB.Exec(ctx, q, t.ID, t.Game, t.Name, t.Type, t.Region,
			t.LengthM, t.SurfaceMix, t.StartLat, t.StartLng, t.Source,
			t.LastVerified); err != nil {
			return fmt.Errorf("upsert track %s: %w", t.ID, err)
		}
	}
	return nil
}
