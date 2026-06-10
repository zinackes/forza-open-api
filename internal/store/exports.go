// Manifeste des archives téléchargeables (GET /v1/exports). Lecture seule pour le
// handler ; écriture (upsert) réservée au job de génération (internal/export via
// cmd/seed | cmd/scheduler). La table ne porte QUE le manifeste : les fichiers
// eux-mêmes vivent sur l'object store / l'edge. Requêtes paramétrées.
package store

import (
	"context"
	"fmt"
	"time"
)

// Export décrit une archive statique : un fichier (un jeu, une ressource, un
// format) servi depuis l'edge. ETag = hash de contenu calculé à la génération.
type Export struct {
	Game        string
	Resource    string
	Format      string
	URL         string
	SizeBytes   int64
	ETag        string
	GeneratedAt time.Time
}

// UpsertExport enregistre (ou rafraîchit) une entrée du manifeste. La clé
// naturelle (game, resource, format) rend la régénération idempotente : rejouer
// le job ne crée pas de doublon, il met à jour url/taille/etag/instant.
func (s *Store) UpsertExport(ctx context.Context, e Export) error {
	const q = `
INSERT INTO exports (game, resource, format, url, size_bytes, etag, generated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (game, resource, format) DO UPDATE SET
    url          = EXCLUDED.url,
    size_bytes   = EXCLUDED.size_bytes,
    etag         = EXCLUDED.etag,
    generated_at = EXCLUDED.generated_at`
	if _, err := s.DB.Exec(ctx, q, e.Game, e.Resource, e.Format, e.URL, e.SizeBytes, e.ETag, e.GeneratedAt); err != nil {
		return fmt.Errorf("upsert export %s/%s/%s: %w", e.Game, e.Resource, e.Format, err)
	}
	return nil
}

// ListExports renvoie le manifeste, filtré par jeu si game != nil (game == nil →
// toutes les archives, tous jeux). Ordre stable (game, resource, format).
func (s *Store) ListExports(ctx context.Context, game *string) ([]Export, error) {
	const q = `
SELECT game, resource, format, url, size_bytes, etag, generated_at
FROM exports
WHERE ($1::text IS NULL OR game = $1)
ORDER BY game, resource, format`
	rows, err := s.DB.Query(ctx, q, game)
	if err != nil {
		return nil, fmt.Errorf("query exports: %w", err)
	}
	defer rows.Close()

	out := []Export{}
	for rows.Next() {
		var e Export
		if err := rows.Scan(&e.Game, &e.Resource, &e.Format, &e.URL, &e.SizeBytes, &e.ETag, &e.GeneratedAt); err != nil {
			return nil, fmt.Errorf("scan export: %w", err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate exports: %w", err)
	}
	return out, nil
}
