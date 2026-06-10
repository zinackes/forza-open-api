// Forzathon Shop : rotation hebdomadaire d'objets achetables contre des Forza
// Points. Écriture par l'ingestion (internal/ingest/forzathon), jamais par un
// handler de lecture. Requêtes paramétrées only. Upsert idempotent et rejouable :
// chaque objet est upserté sur sa clé naturelle (game, week_start, name) — un
// re-run d'une même rotation ne crée pas de doublon, et un objet retiré de la
// source côté semaine déjà ingérée n'est PAS supprimé (l'historique est conservé).
// L'upsert journalise une entrée data_changes par rotation (resource
// « forzathon_shop »), lue par Meta/Changes pour la fraîcheur.
package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ForzathonShopItem est la vue DB d'un objet d'une rotation du Forzathon Shop.
// Les champs pointeur sont nil si la source ne les fournit pas (jamais inventés).
// CarID rapproche un objet kind=car d'une voiture du catalogue (nil sinon).
type ForzathonShopItem struct {
	ID           string
	Game         string
	WeekStart    time.Time
	WeekEnd      *time.Time
	Kind         string // car | horn | clothing | forza_link_phrase | other
	CarID        *string
	Name         string
	FpCost       *int
	Description  *string
	ImageURL     *string
	Source       *string
	LastVerified *time.Time
}

// forzathonCols est la liste de colonnes partagée par les lectures du shop.
const forzathonCols = `id, game, week_start, week_end, kind, car_id, name, fp_cost, description, image_url, source, last_verified`

// scanForzathon projette une ligne (ordre forzathonCols) en ForzathonShopItem.
func scanForzathon(row pgx.Row, it *ForzathonShopItem) error {
	return row.Scan(&it.ID, &it.Game, &it.WeekStart, &it.WeekEnd, &it.Kind, &it.CarID,
		&it.Name, &it.FpCost, &it.Description, &it.ImageURL, &it.Source, &it.LastVerified)
}

// UpsertForzathonRotation enregistre les objets d'une rotation de façon idempotente
// et transactionnelle (ON CONFLICT sur la clé naturelle game/week_start/name).
// Journalise une entrée data_changes par semaine distincte présente dans items
// (added si la semaine était inconnue, updated sinon). Appelé par l'ingestion only.
func (s *Store) UpsertForzathonRotation(ctx context.Context, items []ForzathonShopItem) error {
	if len(items) == 0 {
		return nil
	}
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin forzathon tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Semaines distinctes touchées + leur état AVANT upsert : une semaine sans ligne
	// préexistante est « added », sinon « updated » (sondé avant le batch, sinon les
	// lignes fraîchement upsertées masqueraient un re-run).
	type wk struct {
		game  string
		start time.Time
	}
	counts := map[wk]int{}
	for _, it := range items {
		counts[wk{it.Game, it.WeekStart}]++
	}
	priorEmpty := make(map[wk]bool, len(counts))
	for w := range counts {
		var prior int
		if err = tx.QueryRow(ctx,
			`SELECT count(*) FROM forzathon_shop_items WHERE game = $1 AND week_start = $2`,
			w.game, w.start).Scan(&prior); err != nil {
			return fmt.Errorf("probe forzathon week (%s %s): %w", w.game, w.start.Format(time.RFC3339), err)
		}
		priorEmpty[w] = prior == 0
	}

	const upsert = `
INSERT INTO forzathon_shop_items
    (id, game, week_start, week_end, kind, car_id, name, fp_cost, description, image_url, source, last_verified)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
ON CONFLICT (game, week_start, name) DO UPDATE SET
    week_end = EXCLUDED.week_end, kind = EXCLUDED.kind, car_id = EXCLUDED.car_id,
    fp_cost = EXCLUDED.fp_cost, description = EXCLUDED.description,
    image_url = EXCLUDED.image_url, source = EXCLUDED.source,
    last_verified = EXCLUDED.last_verified`

	batch := &pgx.Batch{}
	for _, it := range items {
		batch.Queue(upsert, it.ID, it.Game, it.WeekStart, it.WeekEnd, it.Kind, it.CarID,
			it.Name, it.FpCost, it.Description, it.ImageURL, it.Source, it.LastVerified)
	}
	br := tx.SendBatch(ctx, batch)
	for range items {
		if _, err = br.Exec(); err != nil {
			_ = br.Close()
			return fmt.Errorf("upsert forzathon item: %w", err)
		}
	}
	if err = br.Close(); err != nil {
		return fmt.Errorf("close forzathon batch: %w", err)
	}

	changes := make([]Change, 0, len(counts))
	for w, n := range counts {
		action := "updated"
		if priorEmpty[w] {
			action = "added"
		}
		summary := fmt.Sprintf("Forzathon Shop %s — semaine %s (%d objets)", w.game, w.start.Format("2006-01-02"), n)
		changes = append(changes, Change{
			Game: w.game, Resource: "forzathon_shop",
			ResourceID: fmt.Sprintf("%s-%s", w.game, w.start.Format("20060102")),
			Action:     action, Summary: &summary,
		})
	}
	if err = insertChanges(ctx, tx, changes); err != nil {
		return fmt.Errorf("record forzathon changes: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit forzathon tx: %w", err)
	}
	return nil
}

// CurrentForzathonShop renvoie les objets de la rotation courante d'un jeu : ceux
// de la semaine la plus récente connue (max week_start). Vide si aucune rotation
// n'a été ingérée pour le jeu.
func (s *Store) CurrentForzathonShop(ctx context.Context, game string) ([]ForzathonShopItem, error) {
	const q = `
SELECT ` + forzathonCols + ` FROM forzathon_shop_items
WHERE game = $1 AND week_start = (SELECT max(week_start) FROM forzathon_shop_items WHERE game = $1)
ORDER BY fp_cost NULLS LAST, name`
	rows, err := s.DB.Query(ctx, q, game)
	if err != nil {
		return nil, fmt.Errorf("query current forzathon (%s): %w", game, err)
	}
	defer rows.Close()

	var out []ForzathonShopItem
	for rows.Next() {
		var it ForzathonShopItem
		if err := scanForzathon(rows, &it); err != nil {
			return nil, fmt.Errorf("scan forzathon item: %w", err)
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// ForzathonShopHistory renvoie une page d'objets de toutes les rotations connues
// d'un jeu (plus récentes d'abord) + le total (COUNT séparé : exact même hors
// borne de pagination). Sert GET /v1/forzathon-shop/history.
func (s *Store) ForzathonShopHistory(ctx context.Context, game string, limit, offset int) ([]ForzathonShopItem, int64, error) {
	var total int64
	if err := s.DB.QueryRow(ctx,
		`SELECT count(*) FROM forzathon_shop_items WHERE game = $1`, game).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count forzathon (%s): %w", game, err)
	}

	const q = `
SELECT ` + forzathonCols + ` FROM forzathon_shop_items
WHERE game = $1
ORDER BY week_start DESC, fp_cost NULLS LAST, name
LIMIT $2 OFFSET $3`
	rows, err := s.DB.Query(ctx, q, game, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query forzathon history (%s): %w", game, err)
	}
	defer rows.Close()

	out := make([]ForzathonShopItem, 0, limit)
	for rows.Next() {
		var it ForzathonShopItem
		if err := scanForzathon(rows, &it); err != nil {
			return nil, 0, fmt.Errorf("scan forzathon item: %w", err)
		}
		out = append(out, it)
	}
	return out, total, rows.Err()
}
