// Handler de l'endpoint d'identité de clé : GET /v1/me. Authentification REQUISE
// (security ApiKeyAuth au contrat) : renvoie le nom, les scopes, le quota et
// l'état courant du rate-limit de la clé présentée. Réponse propre à la clé et
// volatile → no-store. Pattern GitHub /rate_limit + Stripe /v1/me.
package handler

import (
	"context"
	"time"

	"github.com/zinackes/forza-open-api/internal/oas"
)

// meCacheControl : réponse propre à la clé (identité + quota), volatile à chaque
// requête (remaining décroît). Jamais mise en cache, edge ou navigateur.
const meCacheControl = "no-store"

// GetMe implémente GET /v1/me. La clé a déjà été validée par le SecurityHandler
// (AuthInfo en context) ; on relit la ligne api_keys pour le nom, les scopes et
// la date de création, et on réutilise l'état du quota déposé par le middleware
// de rate-limit. remaining/resetAt sont omis si le compteur (Redis) est
// indisponible (rien d'inventé).
func (h *Handler) GetMe(ctx context.Context) (oas.GetMeRes, error) {
	info, ok := AuthInfoFromContext(ctx)
	if !ok {
		// Endpoint authentifié mais clé non résolue : soit aucune clé (ogen renvoie
		// alors 401 avant d'arriver ici), soit auth tombée en fail-open anonyme
		// (DB indisponible au moment de la validation). Pas d'identité → 401.
		return nil, errUnauthorized("api key required")
	}

	k, err := h.store.LookupAPIKey(ctx, info.Hash)
	if err != nil {
		return nil, err
	}
	if k == nil || k.RevokedAt != nil {
		// Course rare : clé résolue via le cache court puis révoquée/supprimée.
		return nil, errUnauthorized("invalid or revoked API key")
	}

	// scopes est NOT NULL en base (DEFAULT {read}) ; on garantit un tableau non nil
	// pour sérialiser [] plutôt que null dans le cas dégénéré d'une clé sans scope.
	scopes := k.Scopes
	if scopes == nil {
		scopes = []string{}
	}

	me := oas.Me{
		Name:      k.Name,
		Scopes:    scopes,
		RateLimit: k.RateLimit,
		CreatedAt: k.CreatedAt.UTC(),
	}

	// État du quota : on réutilise le RateLimitResult calculé par le middleware
	// pour CETTE requête (donc cohérent avec les en-têtes X-RateLimit-*), sans
	// second appel Redis. Absent si le compteur n'a pas tourné (anonyme impossible
	// ici, ou Redis indisponible → fail-open) → champs omis.
	if res, ok := RateLimitResultFromContext(ctx); ok {
		me.Remaining = oas.NewOptInt(res.Remaining)
		me.ResetAt = oas.NewOptDateTime(time.Now().UTC().Add(res.ResetAfter))
	}

	return &oas.MeHeaders{
		CacheControl: oas.NewOptString(meCacheControl),
		Response:     me,
	}, nil
}
