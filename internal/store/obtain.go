// Vue agrégée « comment obtenir cette voiture » pour GET /v1/cars/{id}/obtain :
// méthode du catalogue + packs DLC + Barn Find + Treasure Car + paliers Journal
// + perks Car Mastery (car_unlock). Lectures simples réutilisant les vues DB
// existantes ; rien d'inventé — chaque voie absente reste vide/nil.
package store

import (
	"context"
	"fmt"
)

// MasteryUnlock référence une perk Car Mastery (d'une autre voiture) qui
// débloque la voiture demandée.
type MasteryUnlock struct {
	PerkID     string
	OwnerCarID string
	PerkName   *string
}

// CarObtain agrège les voies d'obtention d'une voiture. Car porte la méthode du
// catalogue (obtain_method, value_cr). Slices vides (jamais nil) si aucune voie.
type CarObtain struct {
	Car            Car
	DlcPacks       []DlcPack
	BarnFind       *BarnFind
	TreasureCar    *TreasureCar
	JournalTiers   []JournalTier
	MasteryUnlocks []MasteryUnlock
}

// GetCarObtain renvoie la vue agrégée d'obtention. (nil, nil) si la voiture est
// inconnue — le handler en fait un 404.
func (s *Store) GetCarObtain(ctx context.Context, id string) (*CarObtain, error) {
	car, err := s.GetCar(ctx, id)
	if err != nil {
		return nil, err
	}
	if car == nil {
		return nil, nil
	}
	out := CarObtain{
		Car:            *car,
		DlcPacks:       make([]DlcPack, 0),
		JournalTiers:   make([]JournalTier, 0),
		MasteryUnlocks: make([]MasteryUnlock, 0),
	}

	const dlcQ = `
SELECT d.id, d.game, d.name, d.kind, d.released_at, d.description, d.source,
       d.last_verified
FROM dlc_packs d
JOIN car_dlc cd ON cd.dlc_id = d.id
WHERE cd.car_id = $1
ORDER BY d.released_at DESC NULLS LAST, d.name`
	rows, err := s.DB.Query(ctx, dlcQ, id)
	if err != nil {
		return nil, fmt.Errorf("query obtain dlc %s: %w", id, err)
	}
	for rows.Next() {
		var d DlcPack
		if err := rows.Scan(&d.ID, &d.Game, &d.Name, &d.Kind, &d.ReleasedAt,
			&d.Description, &d.Source, &d.LastVerified); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan obtain dlc: %w", err)
		}
		out.DlcPacks = append(out.DlcPacks, d)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate obtain dlc: %w", err)
	}

	// Un Barn Find / une Treasure Car au plus par voiture en pratique ; on prend
	// la première par id pour rester déterministe si la donnée en avait plusieurs.
	bf, _, err := s.ListBarnFinds(ctx, HiddenCarFilter{
		Game: car.Game, CarID: &id, Limit: 1,
	})
	if err != nil {
		return nil, err
	}
	if len(bf) > 0 {
		out.BarnFind = &bf[0]
	}

	tc, _, err := s.ListTreasureCars(ctx, HiddenCarFilter{
		Game: car.Game, CarID: &id, Limit: 1,
	})
	if err != nil {
		return nil, err
	}
	if len(tc) > 0 {
		out.TreasureCar = &tc[0]
	}

	const journalQ = `
SELECT id, game, track, level, color, name, points_required,
       reward_car_id, unlocks_description, source, last_verified
FROM journal_tiers
WHERE reward_car_id = $1
ORDER BY track, level`
	rows, err = s.DB.Query(ctx, journalQ, id)
	if err != nil {
		return nil, fmt.Errorf("query obtain journal %s: %w", id, err)
	}
	for rows.Next() {
		var j JournalTier
		if err := rows.Scan(&j.ID, &j.Game, &j.Track, &j.Level, &j.Color, &j.Name,
			&j.PointsRequired, &j.RewardCarID, &j.UnlocksDescription, &j.Source,
			&j.LastVerified); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan obtain journal: %w", err)
		}
		out.JournalTiers = append(out.JournalTiers, j)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate obtain journal: %w", err)
	}

	const masteryQ = `
SELECT id, car_id, name
FROM car_mastery_perks
WHERE unlocked_car_id = $1
ORDER BY id`
	rows, err = s.DB.Query(ctx, masteryQ, id)
	if err != nil {
		return nil, fmt.Errorf("query obtain mastery %s: %w", id, err)
	}
	for rows.Next() {
		var m MasteryUnlock
		if err := rows.Scan(&m.PerkID, &m.OwnerCarID, &m.PerkName); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan obtain mastery: %w", err)
		}
		out.MasteryUnlocks = append(out.MasteryUnlocks, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate obtain mastery: %w", err)
	}

	return &out, nil
}
