// Middlewares transport CORS + en-têtes de sécurité, exécutés côté net/http en
// enveloppant tout le mux (statiques inclus), AVANT le rate-limit et le cache.
// Le preflight OPTIONS est court-circuité ici : il ne doit pas atteindre le
// routeur ogen (qui ne connaît pas OPTIONS) ni consommer de quota.
//
// L'API est consommée depuis le navigateur (overlays, apps web via le SDK TS) :
// lecture publique large, writes restreints à des origines explicites.
package handler

import (
	"net/http"
	"strings"
)

// corsExposeHeaders : en-têtes de réponse lisibles par le JS navigateur. Sans
// Access-Control-Expose-Headers, fetch() ne voit que les en-têtes « safelisted » ;
// le client doit pouvoir lire le quota (X-RateLimit-*, RateLimit), le Retry-After
// d'un 429 et l'ETag pour la revalidation conditionnelle.
var corsExposeHeaders = strings.Join([]string{
	"X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset",
	"RateLimit", "RateLimit-Policy", "Retry-After", "ETag",
}, ", ")

// corsAllowHeaders : en-têtes de requête autorisés en preflight. X-API-Key porte
// l'auth, If-None-Match la revalidation conditionnelle, Content-Type les futurs
// writes JSON.
const corsAllowHeaders = "X-API-Key, Content-Type, If-None-Match"

const (
	corsReadMethods  = "GET, HEAD, OPTIONS"
	corsWriteMethods = "POST, PUT, PATCH, DELETE, OPTIONS"
	corsMaxAge       = "600" // 10 min : durée de cache navigateur du preflight.
)

// CORS applique la politique CORS : lecture large, writes restreints. Les deux
// jeux d'origines sont résolus selon la méthode (réelle ou demandée en preflight).
type CORS struct {
	read  originSet
	write originSet
}

// NewCORS construit le middleware. read = origines de lecture (GET/HEAD),
// write = origines des writes (POST/...). "*" dans une liste → toute origine.
func NewCORS(read, write []string) *CORS {
	return &CORS{read: newOriginSet(read), write: newOriginSet(write)}
}

// originSet représente un ensemble d'origines autorisées, avec le cas joker "*".
type originSet struct {
	any  bool            // "*" présent → toute origine autorisée
	list map[string]bool // origines exactes (scheme://host[:port])
}

func newOriginSet(origins []string) originSet {
	s := originSet{list: make(map[string]bool, len(origins))}
	for _, o := range origins {
		switch o = strings.TrimSpace(o); o {
		case "":
		case "*":
			s.any = true
		default:
			s.list[o] = true
		}
	}
	return s
}

// allow décide d'Access-Control-Allow-Origin pour origin. value est la valeur à
// poser ("*" ou l'origine en écho), vary indique si la réponse dépend de l'origine
// (→ Vary: Origin), ok signale une origine autorisée.
func (s originSet) allow(origin string) (value string, vary, ok bool) {
	if s.any {
		return "*", false, true
	}
	if origin != "" && s.list[origin] {
		return origin, true, true
	}
	return "", false, false
}

// Middleware enveloppe next. Sans en-tête Origin (curl, appel serveur) il est
// transparent. Sur une requête réelle il pose Allow-Origin/Expose-Headers selon
// la classe de méthode ; sur un preflight il répond 204 sans toucher l'aval.
func (c *CORS) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			c.preflight(w, r, origin)
			return
		}

		set := c.read
		if !isCORSSafeMethod(r.Method) {
			set = c.write
		}
		if value, vary, ok := set.allow(origin); ok {
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", value)
			if vary {
				h.Add("Vary", "Origin")
			}
			h.Set("Access-Control-Expose-Headers", corsExposeHeaders)
		}
		next.ServeHTTP(w, r)
	})
}

// preflight répond à OPTIONS + Access-Control-Request-Method. La classe d'origines
// dépend de la méthode demandée (read vs write). Origine non autorisée → 204 sans
// en-tête CORS : le navigateur bloque alors la requête réelle.
func (c *CORS) preflight(w http.ResponseWriter, r *http.Request, origin string) {
	h := w.Header()
	// La réponse varie selon ces trois en-têtes de requête (caches/CDN).
	h.Add("Vary", "Origin")
	h.Add("Vary", "Access-Control-Request-Method")
	h.Add("Vary", "Access-Control-Request-Headers")

	reqMethod := r.Header.Get("Access-Control-Request-Method")
	set, methods := c.read, corsReadMethods
	if !isCORSSafeMethod(reqMethod) {
		set, methods = c.write, corsWriteMethods
	}

	value, _, ok := set.allow(origin)
	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	h.Set("Access-Control-Allow-Origin", value)
	h.Set("Access-Control-Allow-Methods", methods)
	h.Set("Access-Control-Allow-Headers", corsAllowHeaders)
	h.Set("Access-Control-Max-Age", corsMaxAge)
	w.WriteHeader(http.StatusNoContent)
}

// isCORSSafeMethod : méthodes sans effet de bord, relevant de la politique de
// lecture. Tout le reste (POST/PUT/PATCH/DELETE) relève de la politique d'écriture.
func isCORSSafeMethod(m string) bool {
	return m == http.MethodGet || m == http.MethodHead || m == http.MethodOptions
}

// SecurityHeaders pose des en-têtes de sécurité de base sur toutes les réponses.
// Volontairement minimal et complémentaire au bord (Cloudflare gère TLS/HSTS) :
// pas de HSTS ni de CSP (API JSON, aucun HTML rendu). nosniff coupe le MIME-
// sniffing, DENY interdit l'embarquement en iframe, no-referrer évite les fuites
// d'URL, et CORP cross-origin assume la consommation cross-site de l'API publique.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Cross-Origin-Resource-Policy", "cross-origin")
		next.ServeHTTP(w, r)
	})
}
