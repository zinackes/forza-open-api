// Validation réelle de la clé X-API-Key (oas.SecurityHandler). La sécurité est
// OPTIONNELLE dans le contrat (security: [{}, {ApiKeyAuth: []}]) : ogen n'appelle
// HandleApiKeyAuth QUE si l'en-tête X-API-Key est présent. L'absence de clé suit
// le chemin « lecture publique » et n'entre jamais ici. Quand une clé est fournie,
// elle doit être valide : inconnue ou révoquée → 401.
package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zinackes/forza-open-api/internal/apikey"
	"github.com/zinackes/forza-open-api/internal/oas"
	"github.com/zinackes/forza-open-api/internal/store"
)

// apiKeyCacheTTL : durée de vie courte du cache Redis des résolutions de clé.
// Courte pour qu'une révocation ou un changement de quota se propagent vite (au
// pire après ce délai) ; suffisante pour absorber une rafale d'une même clé sans
// retaper Postgres, qui reste la source de vérité.
const apiKeyCacheTTL = 60 * time.Second

// apiKeyCachePrefix préfixe les entrées de cache (versionné : permet d'invalider
// en masse si le format de la valeur évolue).
const apiKeyCachePrefix = "apikey:v1:"

// apiKeyCacheDeny est la valeur sentinelle d'un cache négatif (hash inconnu ou
// clé révoquée) : évite de retaper Postgres pour une clé invalide répétée.
const apiKeyCacheDeny = "deny"

// errInvalidAPIKey : clé absente de la base ou révoquée. Renvoyée par
// HandleApiKeyAuth ; ogen l'emballe en *ogenerrors.SecurityError (Code 401),
// rendu en application/problem+json par ProblemErrorHandler.
var errInvalidAPIKey = errors.New("invalid or revoked API key")

// AuthInfo porte la clé API résolue, placée dans le context par le
// SecurityHandler à destination du middleware de rate-limit (carte 2.6). On n'y
// met que le hash (jamais la clé en clair) et le quota associé.
type AuthInfo struct {
	Hash      string
	RateLimit int
}

// authInfoCtxKey est la clé de context (type privé : aucune collision possible).
type authInfoCtxKey struct{}

// AuthInfoFromContext renvoie la clé résolue placée par le SecurityHandler.
// ok=false pour une requête anonyme (lecture publique sans clé valide).
func AuthInfoFromContext(ctx context.Context) (AuthInfo, bool) {
	info, ok := ctx.Value(authInfoCtxKey{}).(AuthInfo)
	return info, ok
}

// SecurityHandler implémente oas.SecurityHandler.
type SecurityHandler struct {
	store *store.Store
}

// NewSecurityHandler construit le SecurityHandler avec son accès aux données :
// Postgres pour le lookup, Redis pour le cache court. Redis peut être nil — le
// cache est alors simplement désactivé (Postgres reste la source de vérité).
func NewSecurityHandler(st *store.Store) SecurityHandler {
	return SecurityHandler{store: st}
}

// HandleApiKeyAuth valide la clé reçue dans X-API-Key : hash sha256 → cache Redis
// court → fallback Postgres. Clé inconnue ou révoquée → 401. Clé valide → AuthInfo
// (hash + quota) placée dans le context pour le middleware de rate-limit.
//
// Tolérance aux pannes : une clé présente mais invérifiable (DB indisponible) ne
// doit pas 401 un client légitime — la lecture étant publique, on poursuit en
// anonyme (sans AuthInfo) et on logue l'incident.
func (h SecurityHandler) HandleApiKeyAuth(ctx context.Context, _ oas.OperationName, t oas.ApiKeyAuth) (context.Context, error) {
	hash := apikey.Hash(t.APIKey)

	rateLimit, err := h.resolve(ctx, hash)
	switch {
	case err == nil:
		return context.WithValue(ctx, authInfoCtxKey{}, AuthInfo{Hash: hash, RateLimit: rateLimit}), nil
	case errors.Is(err, errInvalidAPIKey):
		return ctx, err
	default:
		slog.WarnContext(ctx, "api key lookup failed, serving as anonymous", "err", err)
		return ctx, nil
	}
}

// resolve renvoie le rate_limit d'une clé active (par son hash), errInvalidAPIKey
// si elle est inconnue/révoquée, ou une erreur d'infrastructure sinon. Tente le
// cache Redis (positif : rate_limit ; négatif : "deny"), puis Postgres.
func (h SecurityHandler) resolve(ctx context.Context, hash string) (int, error) {
	cacheKey := apiKeyCachePrefix + hash

	// 1. Cache Redis (court, optionnel). Miss ou panne Redis → fallback Postgres.
	if h.store.Redis != nil {
		switch v, err := h.store.Redis.Get(ctx, cacheKey).Result(); {
		case err == nil && v == apiKeyCacheDeny:
			return 0, errInvalidAPIKey
		case err == nil:
			if rl, convErr := strconv.Atoi(v); convErr == nil {
				return rl, nil
			}
			// Valeur corrompue : on ignore le cache et on réinterroge Postgres.
		case !errors.Is(err, redis.Nil):
			slog.DebugContext(ctx, "api key cache get failed", "err", err)
		}
	}

	// 2. Fallback Postgres (source de vérité).
	k, err := h.store.LookupAPIKey(ctx, hash)
	if err != nil {
		return 0, fmt.Errorf("lookup api key: %w", err)
	}
	if k == nil || k.RevokedAt != nil {
		h.cacheSet(ctx, cacheKey, apiKeyCacheDeny)
		return 0, errInvalidAPIKey
	}
	h.cacheSet(ctx, cacheKey, strconv.Itoa(k.RateLimit))
	return k.RateLimit, nil
}

// cacheSet écrit une résolution dans le cache (TTL court). Best-effort : une
// panne Redis ne casse pas l'auth. No-op si Redis n'est pas configuré.
func (h SecurityHandler) cacheSet(ctx context.Context, key, val string) {
	if h.store.Redis == nil {
		return
	}
	if err := h.store.Redis.Set(ctx, key, val, apiKeyCacheTTL).Err(); err != nil {
		slog.DebugContext(ctx, "api key cache set failed", "err", err)
	}
}
