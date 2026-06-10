// Handler du manifeste des archives téléchargeables : GET /v1/exports. Lecture
// seule, store paramétré. `game` est un filtre OPTIONNEL (manifeste cross-jeu,
// comme /v1/meta) : absent → toutes les archives. Ne renvoie que ce qui a été
// réellement généré (jamais d'URL inventée). Cache moyen : le manifeste ne bouge
// qu'à la régénération (~quotidienne) ; les fichiers pointés ont un cache long +
// ETag côté static/Cloudflare.
package handler

import (
	"context"

	"github.com/zinackes/forza-open-api/internal/oas"
)

// exportsCacheControl : cache moyen côté origin/edge pour le manifeste.
const exportsCacheControl = "public, max-age=300, stale-while-revalidate=600"

// ListExports implémente GET /v1/exports. Filtre par jeu si fourni, sinon liste
// toutes les archives.
func (h *Handler) ListExports(ctx context.Context, params oas.ListExportsParams) (oas.ListExportsRes, error) {
	var game *string
	if g, ok := params.Game.Get(); ok {
		s := string(g)
		game = &s
	}

	rows, err := h.store.ListExports(ctx, game)
	if err != nil {
		return nil, err
	}

	exports := make([]oas.Export, 0, len(rows))
	for _, e := range rows {
		exports = append(exports, oas.Export{
			Game:        oas.Game(e.Game),
			Resource:    oas.ExportResource(e.Resource),
			Format:      oas.ExportFormat(e.Format),
			URL:         e.URL,
			SizeBytes:   e.SizeBytes,
			Etag:        e.ETag,
			GeneratedAt: e.GeneratedAt,
		})
	}

	return &oas.ListExportsOKHeaders{
		CacheControl: oas.NewOptString(exportsCacheControl),
		Response:     exports,
	}, nil
}
