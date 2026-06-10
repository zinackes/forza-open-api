// Middleware de rate-limit par clé API, exécuté AVANT les handlers (il enveloppe
// le serveur ogen côté net/http). Il s'exécute hors du flux ogen à dessein : un
// middleware ogen n'a pas accès au http.ResponseWriter et ne pourrait donc pas
// poser les en-têtes X-RateLimit-* sur les réponses réussies. Ici on résout la
// clé nous-mêmes (même cache court que le SecurityHandler) puis on applique la
// fenêtre glissante Redis.
package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net"
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

// rateLimitResultCtxKey est la clé de context (type privé) sous laquelle le
// middleware dépose le RateLimitResult de la requête courante, à destination des
// handlers qui exposent l'état du quota (GET /v1/me). Posé uniquement quand le
// compteur a effectivement tourné (clé valide + Redis joignable).
type rateLimitResultCtxKey struct{}

// RateLimitResultFromContext renvoie l'état du quota calculé par le middleware
// pour la requête courante. ok=false si aucun quota n'a été appliqué (requête
// anonyme, ou Redis indisponible → fail-open) : l'appelant omet alors les champs.
func RateLimitResultFromContext(ctx context.Context) (store.RateLimitResult, bool) {
	res, ok := ctx.Value(rateLimitResultCtxKey{}).(store.RateLimitResult)
	return res, ok
}

// RateLimiter applique un quota par clé API en fenêtre glissante (Redis).
type RateLimiter struct {
	// sec réutilise la résolution de clé du SecurityHandler (cache Redis court
	// + Postgres), pour partager exactement la même notion de « clé valide ».
	sec    SecurityHandler
	window time.Duration
	// anonLimit borne le trafic SANS clé, par IP source, sur la même fenêtre.
	// Protection de l'origin tant que le bord (Cloudflare, Phase 5) n'est pas
	// devant. 0 = désactivé — à faire une fois derrière le bord : le trafic
	// tunnelé partage une même IP locale et le bord assure déjà cette protection.
	anonLimit int
}

// NewRateLimiter construit le middleware. window est la largeur de la fenêtre
// glissante (cf. config.RateLimitWindow) ; anonLimit le quota par IP du trafic
// anonyme (cf. config.AnonRateLimit, 0 = désactivé).
func NewRateLimiter(sec SecurityHandler, window time.Duration, anonLimit int) *RateLimiter {
	return &RateLimiter{sec: sec, window: window, anonLimit: anonLimit}
}

// Middleware enveloppe next. Comportement :
//   - sans X-API-Key → quota par IP source (anonLimit ; 0 = passthrough et la
//     protection du trafic anonyme relève du bord, Cloudflare) ;
//   - clé invalide/révoquée → on laisse passer pour qu'ogen renvoie 401 ;
//   - panne Redis ou résolution impossible → fail-open (on n'érige pas une
//     panne d'infra en 429 pour un client légitime) ;
//   - clé valide → fenêtre glissante : en-têtes posés, 429 typé au dépassement.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.Header.Get("X-API-Key")
		if raw == "" {
			rl.serveAnonymous(w, r, next)
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
			writeRateLimited(w, r, res)
			return
		}
		// On expose l'état du quota au handler aval (GET /v1/me) via le context :
		// même calcul que les en-têtes X-RateLimit-*, donc corps et en-têtes
		// restent cohérents, sans second appel Redis.
		next.ServeHTTP(w, r.WithContext(context.WithValue(ctx, rateLimitResultCtxKey{}, res)))
	})
}

// serveAnonymous applique le quota par IP au trafic sans clé. Même fenêtre
// glissante Redis que les clés ; le seau est dérivé de l'IP source HASHÉE (pas
// d'IP en clair dans Redis ; les entrées expirent avec la fenêtre). Fail-open
// sur toute erreur, comme le chemin par clé. Le RateLimitResult n'est PAS posé
// dans le context : /v1/me exige une clé et le quota anonyme n'y a pas de sens.
func (rl *RateLimiter) serveAnonymous(w http.ResponseWriter, r *http.Request, next http.Handler) {
	if rl.anonLimit <= 0 {
		next.ServeHTTP(w, r)
		return
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || ip == "" {
		// RemoteAddr inattendu (proxy local, test) : fail-open plutôt que de
		// mettre tout le trafic anonyme dans un seau commun erroné.
		next.ServeHTTP(w, r)
		return
	}

	ctx := r.Context()
	res, err := rl.sec.store.AllowRequest(ctx,
		rateLimitKeyPrefix+"anon:"+apikey.Hash(ip), rl.anonLimit, rl.window)
	if err != nil {
		slog.WarnContext(ctx, "rate limit: redis unavailable, allowing anonymous", "err", err)
		next.ServeHTTP(w, r)
		return
	}

	setRateLimitHeaders(w.Header(), res, rl.window)
	if !res.Allowed {
		writeRateLimited(w, r, res)
		return
	}
	next.ServeHTTP(w, r)
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
// posés par setRateLimitHeaders ; le corps est délégué à writeProblem, l'unique
// écrivain d'erreur, pour rester strictement au même format que 400/401/404.
func writeRateLimited(w http.ResponseWriter, r *http.Request, res store.RateLimitResult) {
	retry := int64(math.Ceil(res.ResetAfter.Seconds()))
	if retry < 1 {
		retry = 1
	}
	w.Header().Set("Retry-After", strconv.FormatInt(retry, 10))
	writeProblem(w, r, http.StatusTooManyRequests, "rate limit exceeded")
}
