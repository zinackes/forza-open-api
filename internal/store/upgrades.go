// Catalogue des pièces d'upgrade & upgrades par voiture : lectures paginées pour
// GET /v1/upgrade-parts et GET /v1/cars/{id}/upgrades. Requêtes paramétrées only.
// Mapping snake_case DB → vue camelCase du contrat côté handler. Deltas NULL → nil.
package store

import (
	"context"
	"fmt"
	"time"
)

// UpgradePart est la vue DB d'une pièce d'upgrade (snake_case). Deltas absents → nil.
type UpgradePart struct {
	ID            string
	Game          string
	Category      string
	Name          string
	Level         *int
	PIDelta       *int
	WeightDeltaKg *int
	PowerDeltaHp  *int
	TorqueDeltaNm *int
	Source        *string
	LastVerified  *time.Time
}

// CarUpgrade = une pièce montable sur une voiture + ses contraintes d'installation
// (prérequis, groupe exclusif). La pièce complète est renvoyée au client.
type CarUpgrade struct {
	Part           UpgradePart
	RequiresPartID *string
	ExclusiveGroup *string
}

// UpgradePartFilter porte les filtres de GET /v1/upgrade-parts. Pointeur nil = pas de filtre.
type UpgradePartFilter struct {
	Game     string
	Category *string
	Limit    int
	Offset   int
}

// CarUpgradeFilter porte les filtres de GET /v1/cars/{id}/upgrades. Le jeu est
// déterminé par la voiture (pas de filtre game).
type CarUpgradeFilter struct {
	CarID    string
	Category *string
	Limit    int
	Offset   int
}

// ListUpgradeParts renvoie une page du catalogue d'un jeu + le total (COUNT séparé :
// total exact même hors borne). fromWhere est partagé COUNT/SELECT.
func (s *Store) ListUpgradeParts(ctx context.Context, f UpgradePartFilter) ([]UpgradePart, int64, error) {
	const fromWhere = `
FROM upgrade_parts
WHERE game = $1
  AND ($2::text IS NULL OR category = $2)`

	var total int64
	if err := s.DB.QueryRow(ctx, `SELECT count(*)`+fromWhere,
		f.Game, f.Category).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count upgrade_parts: %w", err)
	}

	const q = `
SELECT id, game, category, name, level, pi_delta, weight_delta_kg,
       power_delta_hp, torque_delta_nm, source, last_verified` + fromWhere + `
ORDER BY category, level NULLS LAST, name
LIMIT $3 OFFSET $4`
	rows, err := s.DB.Query(ctx, q, f.Game, f.Category, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query upgrade_parts: %w", err)
	}
	defer rows.Close()

	out := make([]UpgradePart, 0, f.Limit)
	for rows.Next() {
		var p UpgradePart
		if err := rows.Scan(&p.ID, &p.Game, &p.Category, &p.Name, &p.Level,
			&p.PIDelta, &p.WeightDeltaKg, &p.PowerDeltaHp, &p.TorqueDeltaNm,
			&p.Source, &p.LastVerified); err != nil {
			return nil, 0, fmt.Errorf("scan upgrade_part: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate upgrade_parts: %w", err)
	}
	return out, total, nil
}

// ListCarUpgrades renvoie les upgrades disponibles pour une voiture + le total
// (COUNT séparé : total exact même hors borne). Joint car_upgrades → upgrade_parts.
// Voiture inconnue ou sans upgrade → page vide. fromWhere est partagé COUNT/SELECT.
func (s *Store) ListCarUpgrades(ctx context.Context, f CarUpgradeFilter) ([]CarUpgrade, int64, error) {
	const fromWhere = `
FROM car_upgrades cu
JOIN upgrade_parts p ON p.id = cu.part_id
WHERE cu.car_id = $1
  AND ($2::text IS NULL OR p.category = $2)`

	var total int64
	if err := s.DB.QueryRow(ctx, `SELECT count(*)`+fromWhere,
		f.CarID, f.Category).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count car_upgrades: %w", err)
	}

	const q = `
SELECT p.id, p.game, p.category, p.name, p.level, p.pi_delta, p.weight_delta_kg,
       p.power_delta_hp, p.torque_delta_nm, p.source, p.last_verified,
       cu.requires_part_id, cu.exclusive_group` + fromWhere + `
ORDER BY p.category, p.level NULLS LAST, p.name
LIMIT $3 OFFSET $4`
	rows, err := s.DB.Query(ctx, q, f.CarID, f.Category, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query car_upgrades: %w", err)
	}
	defer rows.Close()

	out := make([]CarUpgrade, 0, f.Limit)
	for rows.Next() {
		var u CarUpgrade
		if err := rows.Scan(&u.Part.ID, &u.Part.Game, &u.Part.Category, &u.Part.Name,
			&u.Part.Level, &u.Part.PIDelta, &u.Part.WeightDeltaKg, &u.Part.PowerDeltaHp,
			&u.Part.TorqueDeltaNm, &u.Part.Source, &u.Part.LastVerified,
			&u.RequiresPartID, &u.ExclusiveGroup); err != nil {
			return nil, 0, fmt.Errorf("scan car_upgrade: %w", err)
		}
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate car_upgrades: %w", err)
	}
	return out, total, nil
}
