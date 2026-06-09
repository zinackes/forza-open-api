// Package tracks ingère un catalogue de tracés depuis une source PROPRE : dataset
// communautaire (JSON GitHub) ou export d'un wiki Fandom. Jamais le jeu : pas de
// scraping in-game, lecture mémoire ni injection (doctrine zéro gris).
//
// Le parsing est testé sur fixtures figées (testdata/), sans appel réseau.
// Upsert idempotent côté store. Aucune donnée inventée : un champ absent reste
// NULL (pointeur nil), une ligne incomplète/de type inconnu est ignorée.
package tracks

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/zinackes/forza-open-api/internal/store"
)

// dataset est l'enveloppe d'un dataset de tracés. game/source décrivent la
// provenance (obligatoire pour rester game-agnostic et traçable).
type dataset struct {
	Game   string         `json:"game"`
	Source string         `json:"source"`
	Tracks []datasetTrack `json:"tracks"`
}

// datasetTrack est une ligne brute du dataset (snake_case, pointeurs = nullable).
type datasetTrack struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Region     *string  `json:"region"`
	LengthM    *int     `json:"length_m"`
	SurfaceMix *string  `json:"surface_mix"`
	StartLat   *float64 `json:"start_lat"`
	StartLng   *float64 `json:"start_lng"`
}

// validTypes : types de tracé acceptés (miroir du CHECK SQL et de l'enum OpenAPI).
var validTypes = map[string]bool{
	"circuit": true, "road": true, "dirt": true, "cross": true,
	"street": true, "touge": true, "horizon_rush": true,
}

// ParseTracks transforme un dataset JSON en lignes store.Track prêtes à l'upsert.
// game et source viennent de l'enveloppe ; verifiedAt date la vérification depuis
// la source (→ last_verified). Renvoie aussi le nombre de lignes ignorées
// (id/name manquant ou type inconnu) — précision > exhaustivité.
func ParseTracks(raw []byte, verifiedAt time.Time) ([]store.Track, int, error) {
	var ds dataset
	if err := json.Unmarshal(raw, &ds); err != nil {
		return nil, 0, fmt.Errorf("decode track dataset: %w", err)
	}
	if ds.Game == "" {
		return nil, 0, fmt.Errorf(`track dataset: champ "game" requis`)
	}

	var source *string
	if ds.Source != "" {
		s := ds.Source
		source = &s
	}

	out := make([]store.Track, 0, len(ds.Tracks))
	skipped := 0
	for _, t := range ds.Tracks {
		if t.ID == "" || t.Name == "" || !validTypes[t.Type] {
			skipped++
			continue
		}
		lv := verifiedAt
		out = append(out, store.Track{
			ID:           t.ID,
			Game:         ds.Game,
			Name:         t.Name,
			Type:         t.Type,
			Region:       t.Region,
			LengthM:      t.LengthM,
			SurfaceMix:   t.SurfaceMix,
			StartLat:     t.StartLat,
			StartLng:     t.StartLng,
			Source:       source,
			LastVerified: &lv,
		})
	}
	return out, skipped, nil
}
