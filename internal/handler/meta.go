// Handler des métadonnées de service : GET /v1/meta. Lecture seule, store
// paramétré. Énumère les jeux supportés (enum Game du contrat) avec leurs volumes
// et la fraîcheur des données (derniers timestamps d'ingestion), plus une version
// de jeu de données optionnelle. Cache court : la fraîcheur est l'objet de l'endpoint.
package handler

import (
	"context"
	"time"

	"github.com/zinackes/forza-open-api/internal/oas"
)

// metaCacheControl : cache court côté origin/edge. /v1/meta sert la fraîcheur des
// données ; un cache long la masquerait. stale-while-revalidate absorbe les pics
// sans servir une réponse trop ancienne.
const metaCacheControl = "public, max-age=60, stale-while-revalidate=300"

// GetMeta implémente GET /v1/meta. Énumère tous les jeux supportés (enum Game),
// même sans données (carCount 0, timestamps omis), pour amorcer un sélecteur de
// jeu et indiquer la fraîcheur du catalogue / de la playlist.
func (h *Handler) GetMeta(ctx context.Context) (oas.GetMetaRes, error) {
	supported := oas.Game("").AllValues()
	codes := make([]string, len(supported))
	for i, g := range supported {
		codes[i] = string(g)
	}

	rows, err := h.store.Meta(ctx, codes)
	if err != nil {
		return nil, err
	}

	games := make([]oas.MetaGame, 0, len(rows))
	for _, g := range rows {
		games = append(games, oas.MetaGame{
			Game:              oas.Game(g.Game),
			CarCount:          g.CarCount,
			CatalogUpdatedAt:  optTime(g.CatalogUpdatedAt),
			PlaylistUpdatedAt: optTime(g.PlaylistUpdatedAt),
		})
	}

	// dataVersion n'est exposée que si l'opérateur l'a tamponnée (env DATA_VERSION) :
	// vide → champ omis, jamais de version inventée.
	var dataVersion oas.OptString
	if h.dataVersion != "" {
		dataVersion = oas.NewOptString(h.dataVersion)
	}

	return &oas.MetaHeaders{
		CacheControl: oas.NewOptString(metaCacheControl),
		Response: oas.Meta{
			Games:       games,
			DataVersion: dataVersion,
			GeneratedAt: time.Now().UTC(),
		},
	}, nil
}
