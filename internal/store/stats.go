// Agrégats statistiques du catalogue pour GET /v1/stats. Lecture seule, requêtes
// paramétrées only, scopées au jeu demandé. Réutilise carFacetCounts (cf.
// reference.go) pour class/drivetrain/body_type/category ; ajoute les décennies,
// les volumes par constructeur, l'histogramme PI (paliers de 50) et les
// classements (top 10). Les classements speed/acceleration s'appuient sur les
// NOTES in-game (stats JSONB, échelle 0–10) : proxys, pas d'unités physiques —
// la doctrine zéro gris interdit d'inventer vitesse de pointe / 0–100. Aucun
// agrégat n'inclut de valeur NULL (rien d'inventé).
package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

const topN = 10

// TopCar est une entrée de classement : voiture et valeur de la métrique classée.
type TopCar struct {
	CarID string
	Name  string
	Value float64
}

// Stats agrège les statistiques du catalogue d'un jeu. Les facettes énumérées
// (Class, Drivetrain) portent tous leurs codes canoniques, count 0 inclus ; les
// facettes libres (BodyType, Category, YearDecade) seulement les valeurs
// présentes. Les cartes sont toujours non-nil (sérialisées {} si vides).
type Stats struct {
	Class        map[string]int64
	Drivetrain   map[string]int64
	BodyType     map[string]int64
	Category     map[string]int64
	YearDecade   map[string]int64
	Manufacturer []RefCount // make → count, plus fréquents d'abord (Code = make)
	PiHistogram  []RefCount // palier de 50 → count, ordre croissant (Code = "lo-hi")

	TopPI           []TopCar
	TopSpeed        []TopCar
	TopAcceleration []TopCar
}

// GetStats renvoie les agrégats du catalogue pour un jeu. Jeu inconnu →
// class/drivetrain présents (count 0), reste vide (pas d'erreur).
func (s *Store) GetStats(ctx context.Context, game string) (Stats, error) {
	carFacets, err := s.carFacetCounts(ctx, game)
	if err != nil {
		return Stats{}, err
	}

	yearDecade, err := s.yearDecadeCounts(ctx, game)
	if err != nil {
		return Stats{}, err
	}

	manufacturer, err := s.manufacturerCarCounts(ctx, game)
	if err != nil {
		return Stats{}, err
	}

	histogram, err := s.piHistogram(ctx, game)
	if err != nil {
		return Stats{}, err
	}

	topPI, err := s.topByColumn(ctx, game, "pi")
	if err != nil {
		return Stats{}, err
	}
	topSpeed, err := s.topByStat(ctx, game, "speed")
	if err != nil {
		return Stats{}, err
	}
	topAccel, err := s.topByStat(ctx, game, "acceleration")
	if err != nil {
		return Stats{}, err
	}

	return Stats{
		Class:           canonicalMap(canonicalClasses, carFacets["class"]),
		Drivetrain:      canonicalMap(canonicalDrivetrains, carFacets["drivetrain"]),
		BodyType:        nonNil(carFacets["body_type"]),
		Category:        nonNil(carFacets["category"]),
		YearDecade:      yearDecade,
		Manufacturer:    manufacturer,
		PiHistogram:     histogram,
		TopPI:           topPI,
		TopSpeed:        topSpeed,
		TopAcceleration: topAccel,
	}, nil
}

// yearDecadeCounts compte les voitures par décennie (clé = année de début, ex.
// "1990") pour un jeu. Voitures sans année (NULL) exclues ; décennies vides absentes.
func (s *Store) yearDecadeCounts(ctx context.Context, game string) (map[string]int64, error) {
	const q = `
SELECT (year / 10 * 10) AS decade, count(*) AS n
FROM cars
WHERE game = $1 AND year IS NOT NULL
GROUP BY decade`
	rows, err := s.DB.Query(ctx, q, game)
	if err != nil {
		return nil, fmt.Errorf("query year decades: %w", err)
	}
	defer rows.Close()

	out := map[string]int64{}
	for rows.Next() {
		var decade int
		var n int64
		if err := rows.Scan(&decade, &n); err != nil {
			return nil, fmt.Errorf("scan year decade: %w", err)
		}
		out[fmt.Sprintf("%d", decade)] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate year decades: %w", err)
	}
	return out, nil
}

// manufacturerCarCounts compte les voitures par constructeur (make, NOT NULL au
// schéma) pour un jeu, plus fréquents d'abord puis make croissant (ordre stable).
func (s *Store) manufacturerCarCounts(ctx context.Context, game string) ([]RefCount, error) {
	const q = `
SELECT make, count(*) AS n
FROM cars
WHERE game = $1
GROUP BY make
ORDER BY n DESC, make ASC`
	rows, err := s.DB.Query(ctx, q, game)
	if err != nil {
		return nil, fmt.Errorf("query manufacturer counts: %w", err)
	}
	defer rows.Close()

	out := make([]RefCount, 0)
	for rows.Next() {
		var r RefCount
		if err := rows.Scan(&r.Code, &r.Count); err != nil {
			return nil, fmt.Errorf("scan manufacturer count: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate manufacturer counts: %w", err)
	}
	return out, nil
}

// piHistogram compte les voitures par palier de PI de 50 (intervalle fermé-ouvert,
// ex. "100-150"), pour un jeu. Paliers vides omis ; PI NULL exclu. Ordre croissant.
func (s *Store) piHistogram(ctx context.Context, game string) ([]RefCount, error) {
	const q = `
SELECT (pi / 50 * 50) AS lo, count(*) AS n
FROM cars
WHERE game = $1 AND pi IS NOT NULL
GROUP BY lo
ORDER BY lo`
	rows, err := s.DB.Query(ctx, q, game)
	if err != nil {
		return nil, fmt.Errorf("query pi histogram: %w", err)
	}
	defer rows.Close()

	out := make([]RefCount, 0)
	for rows.Next() {
		var lo int
		var n int64
		if err := rows.Scan(&lo, &n); err != nil {
			return nil, fmt.Errorf("scan pi bucket: %w", err)
		}
		out = append(out, RefCount{Code: fmt.Sprintf("%d-%d", lo, lo+50), Count: n})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pi histogram: %w", err)
	}
	return out, nil
}

// topByColumn renvoie les topN voitures d'un jeu pour une colonne numérique
// (ex. pi), valeur décroissante puis id (départage stable). col est un littéral
// contrôlé par l'appelant (jamais une entrée utilisateur).
func (s *Store) topByColumn(ctx context.Context, game, col string) ([]TopCar, error) {
	q := fmt.Sprintf(`
SELECT id, name, %s::float8 AS v
FROM cars
WHERE game = $1 AND %s IS NOT NULL
ORDER BY v DESC, id
LIMIT %d`, col, col, topN)
	rows, err := s.DB.Query(ctx, q, game)
	if err != nil {
		return nil, fmt.Errorf("query top by column %q: %w", col, err)
	}
	defer rows.Close()
	return collectTop(rows)
}

// topByStat renvoie les topN voitures d'un jeu pour une note in-game (clé du
// JSONB stats, ex. speed). Seules les valeurs numériques comptent (jsonb_typeof) :
// une stat absente/non numérique est ignorée (jamais inventée).
func (s *Store) topByStat(ctx context.Context, game, key string) ([]TopCar, error) {
	const q = `
SELECT id, name, (stats->>$2)::float8 AS v
FROM cars
WHERE game = $1 AND jsonb_typeof(stats->$2) = 'number'
ORDER BY v DESC, id
LIMIT $3`
	rows, err := s.DB.Query(ctx, q, game, key, topN)
	if err != nil {
		return nil, fmt.Errorf("query top by stat %q: %w", key, err)
	}
	defer rows.Close()
	return collectTop(rows)
}

// collectTop matérialise un classement (id, name, value).
func collectTop(rows pgx.Rows) ([]TopCar, error) {
	out := make([]TopCar, 0, topN)
	for rows.Next() {
		var c TopCar
		if err := rows.Scan(&c.CarID, &c.Name, &c.Value); err != nil {
			return nil, fmt.Errorf("scan top car: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate top: %w", err)
	}
	return out, nil
}

// canonicalMap projette les compteurs sur l'ordre canonique d'une facette
// énumérée : chaque code valide est présent, count 0 s'il est absent des données.
// Carte toujours non-nil.
func canonicalMap(order []string, counts map[string]int64) map[string]int64 {
	out := make(map[string]int64, len(order))
	for _, code := range order {
		out[code] = counts[code]
	}
	return out
}

// nonNil garantit une carte non-nil (sérialisée {} plutôt que null).
func nonNil(m map[string]int64) map[string]int64 {
	if m == nil {
		return map[string]int64{}
	}
	return m
}
