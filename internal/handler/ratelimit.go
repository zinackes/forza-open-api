// Middleware de rate-limit par clé API, exécuté AVANT les handlers (il enveloppe
// le serveur ogen côté net/http). Il s'exécute hors du flux ogen à dessein : un
// middleware ogen n'a pas accès au http.ResponseWriter et ne pourrait donc pas
// poser les en-têtes X-RateLimit-* sur les réponses réussies. Ici on résout la
// clé nous-mêmes (même cache court que le SecurityHandler) puis on applique la
// fenêtre glissante Redis.
package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/zinackes/forza-open-api/internal/apikey"
	"github.com/zinackes/forza-open-api/internal/store"
)

// rateLimitKeyPrefix préfixe (versionné) les clés de quota dans Redis ; disjoint
// du cache de résolution des clés (apikey:v1:) pour éviter toute collision.
const rateLimitKeyPrefix = "ratelimit:v1:"

// rateLimitPolicyName nomme la politique unique exposée dans les en-têtes IETF
// RateLimit / RateLimit-Policy.
const rateLimitPolicyName = "default"

// RateLimiter applique un quota par clé API en fenêtre glissante (Redis).
type RateLimiter struct {
	// sec réutilise la résolution de clé du SecurityHandler (cache Redis court
	// + Postgres), pour partager exactement la même notion de « clé valide ».
	sec    SecurityHandler
	window time.Duration
}

// NewRateLimiter construit le middleware. window est la largeur de la fenêtre
// glissante (cf. config.RateLimitWindow).
func NewRateLimiter(sec SecurityHandler, window time.Duration) *RateLimiter {
	return &RateLimiter{sec: sec, window: window}
}

// Middleware enveloppe next. Comportement :
//   - sans X-API-Key → lecture publique anonyme, pas de quota par clé (la
//     protection du trafic anonyme relève du bord, Cloudflare) ;
//   - clé invalide/révoquée → on laisse passer pour qu'ogen renvoie 401 ;
//   - panne Redis ou résolution impossible → fail-open (on n'érige pas une
//     panne d'infra en 429 pour un client légitime) ;
//   - clé valide → fenêtre glissante : en-têtes posés, 429 typé au dépassement.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.Header.Get("X-API-Key")
		if raw == "" {
			next.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()
		hash := apikey.Hash(raw)

		limit, err := rl.sec.resolve(ctx, hash)
		if err != nil {
			// errInvalidAPIKey : clé inconnue/révoquée → ogen 401 en aval, rien à
			// limiter. Autre erreur : infra (DB) indisponible → fail-open + log.
			if !errors.Is(err, errInvalidAPIKey) {
				slog.WarnContext(ctx, "rate limit: key resolve failed, allowing", "err", err)
			}
			next.ServeHTTP(w, r)
			return
		}

		res, err := rl.sec.store.AllowRequest(ctx, rateLimitKeyPrefix+hash, limit, rl.window)
		if err != nil {
			slog.WarnContext(ctx, "rate limit: redis unavailable, allowing", "err", err)
			next.ServeHTTP(w, r)
			return
		}

		setRateLimitHeaders(w.Header(), res, rl.window)
		if !res.Allowed {
			writeRateLimited(w, res)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// setRateLimitHeaders pose les deux familles d'en-têtes, qui coexistent pendant
// la transition du standard (audit 2026-06-10) : les X-RateLimit-* historiques
// (delta en secondes) et les champs structurés IETF
// draft-ietf-httpapi-ratelimit-headers-11 (RateLimit / RateLimit-Policy).
func setRateLimitHeaders(h http.Header, res store.RateLimitResult, window time.Duration) {
	resetSecs := int64(math.Ceil(res.ResetAfter.Seconds()))
	windowSecs := int64(window.Seconds())

	h.Set("X-RateLimit-Limit", strconv.Itoa(res.Limit))
	h.Set("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))
	h.Set("X-RateLimit-Reset", strconv.FormatInt(resetSecs, 10))

	// Structured Fields (RFC 9651) : nom de politique entre guillemets via %q.
	h.Set("RateLimit-Policy", fmt.Sprintf("%q;q=%d;w=%d", rateLimitPolicyName, res.Limit, windowSecs))
	h.Set("RateLimit", fmt.Sprintf("%q;r=%d;t=%d", rateLimitPolicyName, res.Remaining, resetSecs))
}

// writeRateLimited rend un 429 RFC 9457 (application/problem+json) avec
// Retry-After (delta en secondes, >= 1). Les en-têtes de quota ont déjà été
// posés par setRateLimitHeaders.
func writeRateLimited(w http.ResponseWriter, res store.RateLimitResult) {
	retry := int64(math.Ceil(res.ResetAfter.Seconds()))
	if retry < 1 {
		retry = 1
	}
	w.Header().Set("Retry-After", strconv.FormatInt(retry, 10))
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(http.StatusTooManyRequests)
	_ = json.NewEncoder(w).Encode(problem{
		Type:   "about:blank",
		Title:  http.StatusText(http.StatusTooManyRequests),
		Status: http.StatusTooManyRequests,
		Detail: "rate limit exceeded",
	})
}
