// Journal des changements de données : lecture paginée pour GET /v1/changes,
// écriture par les jobs d'ingestion (jamais par les handlers de lecture).
// Requêtes paramétrées only. resource est libre (car, dlc_pack, series…).
package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Change est la vue DB d'un changement de données. Summary absent → nil.
type Change struct {
	ID         string
	Game       string
	Resource   string
	ResourceID string
	Action     string // added | updated | removed
	Summary    *string
	OccurredAt time.Time
}

// ChangeFilter porte les filtres de GET /v1/changes. Pointeur nil = pas de filtre.
type ChangeFilter struct {
	Game     string
	Resource *string
	Action   *string
	Since    *time.Time
	Limit    int
	Offset   int
}

// ListChanges renvoie une page de changements (plus récents d'abord) + le total
// (COUNT séparé : total exact même hors borne). fromWhere est partagé COUNT/SELECT.
func (s *Store) ListChanges(ctx context.Context, f ChangeFilter) ([]Change, int64, error) {
	const fromWhere = `
FROM data_changes
WHERE game = $1
  AND ($2::text IS NULL OR resource = $2)
  AND ($3::text IS NULL OR action = $3)
  AND ($4::timestamptz IS NULL OR occurred_at > $4)`

	var total int64
	if err := s.DB.QueryRow(ctx, `SELECT count(*)`+fromWhere,
		f.Game, f.Resource, f.Action, f.Since).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count changes: %w", err)
	}

	const q = `
SELECT id::text, game, resource, resource_id, action, summary, occurred_at` + fromWhere + `
ORDER BY occurred_at DESC, id
LIMIT $5 OFFSET $6`
	rows, err := s.DB.Query(ctx, q, f.Game, f.Resource, f.Action, f.Since,
		f.Limit, f.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query changes: %w", err)
	}
	defer rows.Close()

	out := make([]Change, 0, f.Limit)
	for rows.Next() {
		var c Change
		if err := rows.Scan(&c.ID, &c.Game, &c.Resource, &c.ResourceID, &c.Action,
			&c.Summary, &c.OccurredAt); err != nil {
			return nil, 0, fmt.Errorf("scan change: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate changes: %w", err)
	}
	return out, total, nil
}

// RecordChanges journalise des changements de données (appelé par l'ingestion).
func (s *Store) RecordChanges(ctx context.Context, changes []Change) error {
	return insertChanges(ctx, s.DB, changes)
}

// changeWriter est le sous-ensemble commun pool/tx utilisé par insertChanges :
// permet de journaliser dans la transaction d'un upsert ou hors transaction.
type changeWriter interface {
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
}

// insertChanges insère les changements (id et occurred_at par défaut DB).
func insertChanges(ctx context.Context, db changeWriter, changes []Change) error {
	if len(changes) == 0 {
		return nil
	}
	const q = `
INSERT INTO data_changes (game, resource, resource_id, action, summary)
VALUES ($1,$2,$3,$4,$5)`
	batch := &pgx.Batch{}
	for _, c := range changes {
		batch.Queue(q, c.Game, c.Resource, c.ResourceID, c.Action, c.Summary)
	}
	br := db.SendBatch(ctx, batch)
	for range changes {
		if _, err := br.Exec(); err != nil {
			_ = br.Close()
			return fmt.Errorf("insert change: %w", err)
		}
	}
	if err := br.Close(); err != nil {
		return fmt.Errorf("close changes batch: %w", err)
	}
	return nil
}
