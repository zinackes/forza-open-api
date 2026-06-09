// Car Mastery FH6 : grille de perks 4×4 par voiture. Lecture pour
// GET /v1/cars/{id}/mastery (grille bornée → pas de pagination). Requêtes
// paramétrées only. Mapping snake_case DB → vue camelCase du contrat côté
// handler. Champs non sourcés → nil. effect_type libre (pas d'enum).
package store

import (
	"context"
	"fmt"
	"time"
)

// CarMasteryPerk est la vue DB d'une perk de l'arbre Car Mastery. Champs absents → nil.
type CarMasteryPerk struct {
	ID                string
	CarID             string
	Row               *int
	Col               *int
	Name              *string
	SPCost            *int
	EffectDescription *string
	EffectType        *string
	EffectValue       *int
	PrereqPerkID      *string
	UnlockedCarID     *string
	Source            *string
	LastVerified      *time.Time
}

// ListCarMasteryPerks renvoie toutes les perks de l'arbre d'une voiture, dans
// l'ordre de la grille (row, col). Voiture inconnue ou arbre non sourcé → vide.
func (s *Store) ListCarMasteryPerks(ctx context.Context, carID string) ([]CarMasteryPerk, error) {
	const q = `
SELECT id, car_id, row, col, name, sp_cost, effect_description, effect_type,
       effect_value, prereq_perk_id, unlocked_car_id, source, last_verified
FROM car_mastery_perks
WHERE car_id = $1
ORDER BY row NULLS LAST, col NULLS LAST, id`
	rows, err := s.DB.Query(ctx, q, carID)
	if err != nil {
		return nil, fmt.Errorf("query car_mastery_perks: %w", err)
	}
	defer rows.Close()

	out := make([]CarMasteryPerk, 0)
	for rows.Next() {
		var p CarMasteryPerk
		if err := rows.Scan(&p.ID, &p.CarID, &p.Row, &p.Col, &p.Name, &p.SPCost,
			&p.EffectDescription, &p.EffectType, &p.EffectValue, &p.PrereqPerkID,
			&p.UnlockedCarID, &p.Source, &p.LastVerified); err != nil {
			return nil, fmt.Errorf("scan car_mastery_perk: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate car_mastery_perks: %w", err)
	}
	return out, nil
}
