// Catalogue voitures : lecture paginée filtrée pour GET /v1/cars. Requêtes
// paramétrées only. Mapping snake_case DB → vue camelCase du contrat côté handler.
package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// CarFilter porte les filtres de GET /v1/cars. Pointeur nil = pas de filtre.
type CarFilter struct {
	Game         string
	IDs          []string // restreint aux ids listés (vide/nil = pas de filtre) ; ids inconnus ignorés
	Make         *string
	Class        *string
	PIMin        *int
	PIMax        *int
	Drivetrain   *string
	Category     *string // division in-game, correspondance exacte
	Q            *string
	Dlc          *string // identifiant d'un dlc_packs : restreint aux voitures du pack
	Obtain       *string // valeur du contrat (autoshow, wheelspin, …) : match par token d'obtain_method
	UpdatedSince *time.Time
	Sort         string // valeur du contrat (pi/name/year/value, préfixe "-" = desc) ; vide = défaut pi
	Limit        int
	Offset       int
}

// carSortColumns whiteliste les clés de tri du contrat vers leur colonne SQL.
// La valeur client ne touche jamais le SQL : seul le mapping est interpolé.
var carSortColumns = map[string]string{
	"pi":    "c.pi",
	"name":  "c.name",
	"year":  "c.year",
	"value": "c.value_cr",
}

// carOrderBy traduit Sort en clause ORDER BY whitelistée. Clé inconnue ou vide →
// défaut pi croissant. NULLS LAST pour que year/value_cr absents sortent en
// dernier quel que soit le sens ; départage stable par name puis id.
func carOrderBy(sort string) string {
	dir := "ASC"
	key := strings.TrimPrefix(sort, "-")
	if key != sort {
		dir = "DESC"
	}
	col, ok := carSortColumns[key]
	if !ok {
		col, dir = "c.pi", "ASC"
	}
	return "\nORDER BY " + col + " " + dir + " NULLS LAST, c.name, c.id"
}

// carObtainTokens whiteliste les valeurs du paramètre obtain du contrat vers les
// tokens d'obtain_method qu'elles couvrent (texte multi-valeurs « Autoshow,
// Wheelspin » issu d'obtainNames côté ingestion). Tokens en minuscules : la
// comparaison SQL se fait sur lower(btrim(token)). La valeur client ne touche
// jamais le SQL : tout passe en paramètre text[].
var carObtainTokens = map[string][]string{
	"autoshow":     {"autoshow"},
	"wheelspin":    {"wheelspin"},
	"wristband":    {"wristband reward", "yellow wristband"},
	"barn_find":    {"barn find"},
	"treasure":     {"treasure car"},
	"car_mastery":  {"car mastery"},
	"journal":      {"collection journal"},
	"car_pass":     {"car pass"},
	"hard_to_find": {"hard to find"},
	"aftermarket":  {"aftermarket car"},
	"prologue":     {"complete the prologue"},
	"loyalty":      {"loyalty reward"},
	"preorder":     {"pre-order"},
	"promotional":  {"promotional"},
	"vip":          {"vip membership"},
	"welcome_pack": {"welcome pack"},
	"unobtainable": {"unobtainable"},
}

// obtainFilterTokens traduit la valeur du filtre en tokens à matcher ; nil = pas
// de filtre. Valeur hors whitelist (impossible via l'API — l'enum du contrat la
// rejette en 400) : on matche le token littéral, cohérent avec les codes bruts
// que l'ingestion conserve pour les valeurs inconnues.
func obtainFilterTokens(obtain *string) []string {
	if obtain == nil {
		return nil
	}
	if toks, ok := carObtainTokens[*obtain]; ok {
		return toks
	}
	return []string{strings.ToLower(strings.TrimSpace(*obtain))}
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
	UpdatedAt    time.Time
}

// ListCars renvoie une page de voitures filtrées + le total. Le total vient d'un
// COUNT(*) séparé (et non d'un count(*) OVER()) : il reste exact même quand la page
// demandée dépasse les données. Le filtre Dlc joint car_dlc (EXISTS) : seules les
// voitures du pack remontent. fromWhere est partagé par le COUNT et le SELECT pour
// éviter toute dérive des filtres (concat de constantes, pas de donnée runtime).
func (s *Store) ListCars(ctx context.Context, f CarFilter) ([]Car, int64, error) {
	const fromWhere = `
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
  AND ($10::timestamptz IS NULL OR c.updated_at > $10)
  AND ($11::text[] IS NULL OR c.id = ANY($11))
  AND ($12::text[] IS NULL OR EXISTS (
        SELECT 1 FROM unnest(string_to_array(c.obtain_method, ',')) tok
        WHERE lower(btrim(tok)) = ANY($12)))`

	// Slice vide = pas de filtre (et non « aucun résultat ») : on passe NULL.
	ids := f.IDs
	if len(ids) == 0 {
		ids = nil
	}
	obtainToks := obtainFilterTokens(f.Obtain)

	// q est une recherche de sous-chaîne LITTÉRALE : on neutralise les
	// métacaractères LIKE pour qu'un client ne puisse pas injecter ses propres
	// wildcards (patterns arbitrairement coûteux).
	qLit := f.Q
	if f.Q != nil {
		esc := escapeLike(*f.Q)
		qLit = &esc
	}

	var total int64
	if err := s.DB.QueryRow(ctx, `SELECT count(*)`+fromWhere,
		f.Game, f.Make, f.Class, f.PIMin, f.PIMax,
		f.Drivetrain, f.Category, qLit, f.Dlc, f.UpdatedSince, ids,
		obtainToks).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count cars: %w", err)
	}

	q := `
SELECT c.id, c.game, c.name, c.make, c.model, c.year, c.class, c.pi,
       c.drivetrain, c.stats, c.body_type, c.category, c.rarity, c.value_cr,
       c.obtain_method, c.image_url, c.created_at, c.updated_at` + fromWhere +
		carOrderBy(f.Sort) + `
LIMIT $13 OFFSET $14`
	rows, err := s.DB.Query(ctx, q, f.Game, f.Make, f.Class, f.PIMin, f.PIMax,
		f.Drivetrain, f.Category, qLit, f.Dlc, f.UpdatedSince, ids, obtainToks,
		f.Limit, f.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query cars: %w", err)
	}
	defer rows.Close()

	out := make([]Car, 0, f.Limit)
	for rows.Next() {
		var c Car
		if err := rows.Scan(&c.ID, &c.Game, &c.Name, &c.Make, &c.Model, &c.Year,
			&c.Class, &c.PI, &c.Drivetrain, &c.Stats, &c.BodyType, &c.Category, &c.Rarity,
			&c.ValueCr, &c.ObtainMethod, &c.ImageURL, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan car: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate cars: %w", err)
	}
	return out, total, nil
}

// GetCar renvoie une voiture par son identifiant stable. Renvoie (nil, nil) si
// aucune voiture ne porte cet id — le handler en fait un 404.
func (s *Store) GetCar(ctx context.Context, id string) (*Car, error) {
	const q = `
SELECT c.id, c.game, c.name, c.make, c.model, c.year, c.class, c.pi,
       c.drivetrain, c.stats, c.body_type, c.category, c.rarity, c.value_cr,
       c.obtain_method, c.image_url, c.created_at, c.updated_at
FROM cars c
WHERE c.id = $1`
	var c Car
	err := s.DB.QueryRow(ctx, q, id).Scan(&c.ID, &c.Game, &c.Name, &c.Make,
		&c.Model, &c.Year, &c.Class, &c.PI, &c.Drivetrain, &c.Stats, &c.BodyType,
		&c.Category, &c.Rarity, &c.ValueCr, &c.ObtainMethod, &c.ImageURL, &c.CreatedAt,
		&c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query car %s: %w", id, err)
	}
	return &c, nil
}

// CarsByIDs renvoie les voitures dont l'id figure dans ids (sans filtre game :
// l'id est la clé stable globale). Les ids inconnus n'ont simplement pas de
// ligne — le handler compare au nombre demandé et en fait un 404 (comparaison
// stricte). L'ordre du résultat n'est pas garanti : le handler le réaligne sur
// l'ordre de la requête.
func (s *Store) CarsByIDs(ctx context.Context, ids []string) ([]Car, error) {
	const q = `
SELECT c.id, c.game, c.name, c.make, c.model, c.year, c.class, c.pi,
       c.drivetrain, c.stats, c.body_type, c.category, c.rarity, c.value_cr,
       c.obtain_method, c.image_url, c.created_at, c.updated_at
FROM cars c
WHERE c.id = ANY($1)`
	rows, err := s.DB.Query(ctx, q, ids)
	if err != nil {
		return nil, fmt.Errorf("query cars by ids: %w", err)
	}
	defer rows.Close()

	out := make([]Car, 0, len(ids))
	for rows.Next() {
		var c Car
		if err := rows.Scan(&c.ID, &c.Game, &c.Name, &c.Make, &c.Model, &c.Year,
			&c.Class, &c.PI, &c.Drivetrain, &c.Stats, &c.BodyType, &c.Category, &c.Rarity,
			&c.ValueCr, &c.ObtainMethod, &c.ImageURL, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan car: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cars by ids: %w", err)
	}
	return out, nil
}

// UpsertCars insère ou met à jour des voitures de façon idempotente (ON CONFLICT
// sur l'id). Rejouable sans doublon. Appelé par l'ingestion (cmd/seed), jamais un
// handler. Stats est du JSONB brut (nil → NULL) ; les pointeurs nil → NULL.
// pgx.Batch dans une transaction : un seul aller-retour réseau (vs un par
// voiture) et tout-ou-rien — pas de catalogue partiel en cas d'échec.
//
// Le DO UPDATE est gardé par IS DISTINCT FROM : un re-run sans changement ne
// touche pas la ligne (updated_at stable, pas de WAL inutile). Chaque ligne
// réellement insérée/modifiée est journalisée dans data_changes (même
// transaction) pour GET /v1/changes — added si insert (xmax = 0), updated sinon.
func (s *Store) UpsertCars(ctx context.Context, cars []Car) error {
	const q = `
INSERT INTO cars (id, game, name, make, model, year, class, pi, drivetrain,
                  stats, body_type, category, rarity, value_cr, obtain_method,
                  image_url)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
ON CONFLICT (id) DO UPDATE SET
    game = EXCLUDED.game, name = EXCLUDED.name, make = EXCLUDED.make,
    model = EXCLUDED.model, year = EXCLUDED.year, class = EXCLUDED.class,
    pi = EXCLUDED.pi, drivetrain = EXCLUDED.drivetrain, stats = EXCLUDED.stats,
    body_type = EXCLUDED.body_type, category = EXCLUDED.category,
    rarity = EXCLUDED.rarity, value_cr = EXCLUDED.value_cr,
    obtain_method = EXCLUDED.obtain_method, image_url = EXCLUDED.image_url,
    updated_at = now()
WHERE (cars.game, cars.name, cars.make, cars.model, cars.year, cars.class,
       cars.pi, cars.drivetrain, cars.stats, cars.body_type, cars.category,
       cars.rarity, cars.value_cr, cars.obtain_method, cars.image_url)
      IS DISTINCT FROM
      (EXCLUDED.game, EXCLUDED.name, EXCLUDED.make, EXCLUDED.model,
       EXCLUDED.year, EXCLUDED.class, EXCLUDED.pi, EXCLUDED.drivetrain,
       EXCLUDED.stats, EXCLUDED.body_type, EXCLUDED.category, EXCLUDED.rarity,
       EXCLUDED.value_cr, EXCLUDED.obtain_method, EXCLUDED.image_url)
RETURNING (xmax = 0) AS inserted`
	batch := &pgx.Batch{}
	for _, c := range cars {
		var stats any // nil []byte → NULL ; sinon JSONB brut
		if len(c.Stats) > 0 {
			stats = c.Stats
		}
		batch.Queue(q, c.ID, c.Game, c.Name, c.Make, c.Model,
			c.Year, c.Class, c.PI, c.Drivetrain, stats, c.BodyType, c.Category,
			c.Rarity, c.ValueCr, c.ObtainMethod, c.ImageURL)
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin upsert cars: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op après Commit

	br := tx.SendBatch(ctx, batch)
	changes := make([]Change, 0)
	for _, c := range cars {
		rows, err := br.Query()
		if err != nil {
			_ = br.Close()
			return fmt.Errorf("upsert car %s: %w", c.ID, err)
		}
		// Zéro ligne = upsert no-op (donnée identique) → pas de changement.
		for rows.Next() {
			var inserted bool
			if err := rows.Scan(&inserted); err != nil {
				rows.Close()
				_ = br.Close()
				return fmt.Errorf("scan upsert car %s: %w", c.ID, err)
			}
			action := "updated"
			if inserted {
				action = "added"
			}
			name := c.Name
			changes = append(changes, Change{
				Game: c.Game, Resource: "car", ResourceID: c.ID,
				Action: action, Summary: &name,
			})
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			_ = br.Close()
			return fmt.Errorf("upsert car %s: %w", c.ID, err)
		}
	}
	if err := br.Close(); err != nil {
		return fmt.Errorf("close upsert batch: %w", err)
	}
	if err := insertChanges(ctx, tx, changes); err != nil {
		return fmt.Errorf("record car changes: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit upsert cars: %w", err)
	}
	return nil
}

// escapeLike neutralise les métacaractères LIKE/ILIKE (\, %, _) pour traiter
// l'entrée comme une sous-chaîne littérale (ESCAPE par défaut : backslash).
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// RandomCar tire une voiture au hasard parmi celles satisfaisant les filtres
// (mêmes filtres que ListCars, hors q/dlc/pagination). Renvoie (nil, nil) si
// aucune voiture ne correspond — le handler en fait un 404. ORDER BY random()
// suffit au volume du catalogue (quelques milliers de voitures par jeu).
func (s *Store) RandomCar(ctx context.Context, f CarFilter) (*Car, error) {
	const q = `
SELECT c.id, c.game, c.name, c.make, c.model, c.year, c.class, c.pi,
       c.drivetrain, c.stats, c.body_type, c.category, c.rarity, c.value_cr,
       c.obtain_method, c.image_url, c.created_at, c.updated_at
FROM cars c
WHERE c.game = $1
  AND ($2::text IS NULL OR c.make = $2)
  AND ($3::text IS NULL OR c.class = $3)
  AND ($4::int  IS NULL OR c.pi >= $4)
  AND ($5::int  IS NULL OR c.pi <= $5)
  AND ($6::text IS NULL OR c.drivetrain = $6)
  AND ($7::text IS NULL OR c.category = $7)
ORDER BY random()
LIMIT 1`
	var c Car
	err := s.DB.QueryRow(ctx, q, f.Game, f.Make, f.Class, f.PIMin, f.PIMax,
		f.Drivetrain, f.Category).Scan(&c.ID, &c.Game, &c.Name, &c.Make, &c.Model,
		&c.Year, &c.Class, &c.PI, &c.Drivetrain, &c.Stats, &c.BodyType, &c.Category,
		&c.Rarity, &c.ValueCr, &c.ObtainMethod, &c.ImageURL, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query random car: %w", err)
	}
	return &c, nil
}
