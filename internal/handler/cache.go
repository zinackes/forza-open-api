// Middleware de cache HTTP conditionnel (ETag + If-None-Match → 304), exécuté
// côté net/http en enveloppant le serveur ogen — comme le rate-limit, parce qu'un
// middleware ogen n'a pas accès au http.ResponseWriter et ne peut donc ni calculer
// l'ETag sur le corps final, ni court-circuiter en 304. Le Cache-Control reste
// posé par les handlers (au contrat) ; ici on ne fait que la validation
// conditionnelle, transparente pour les clients qui ne l'utilisent pas.
package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
)

// ConditionalGet ajoute un ETag fort aux réponses GET cacheables et répond 304
// quand l'If-None-Match du client correspond. dataVersion (env DATA_VERSION) est
// mêlé au hash : un bump au refresh du catalogue invalide tous les ETags, ce qui
// force une revalidation propre côté navigateur et bord (Cloudflare).
//
// Cacheable = GET + 200 + Cache-Control sans no-store. Les réponses no-store
// (ex. /v1/cars/random) ne reçoivent jamais d'ETag. Les non-GET et les statuts
// d'erreur passent inchangés.
func ConditionalGet(dataVersion string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cw := &cacheBufferWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(cw, r)

			header := w.Header()
			body := cw.buf.Bytes()

			if r.Method != http.MethodGet || cw.status != http.StatusOK ||
				strings.Contains(strings.ToLower(header.Get("Cache-Control")), "no-store") {
				w.WriteHeader(cw.status)
				_, _ = w.Write(body)
				return
			}

			etag := computeETag(dataVersion, body)
			header.Set("ETag", etag)

			if etagMatches(r.Header.Get("If-None-Match"), etag) {
				// 304 : ETag et Cache-Control sont déjà dans header, pas de corps
				// (RFC 9110 §15.4.5). Content-Length est omis faute d'écriture.
				w.WriteHeader(http.StatusNotModified)
				return
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		})
	}
}

// cacheBufferWriter capture statut et corps sans les transmettre : ConditionalGet
// a besoin du corps complet pour le hash et la décision 304. Header() reste le map
// réel — les en-têtes posés par le handler (Cache-Control) et en amont
// (X-RateLimit-*) sont donc préservés au flush.
type cacheBufferWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
	buf         bytes.Buffer
}

func (w *cacheBufferWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.status = code
		w.wroteHeader = true
	}
}

func (w *cacheBufferWriter) Write(b []byte) (int, error) {
	w.wroteHeader = true
	return w.buf.Write(b)
}

// computeETag dérive un ETag fort des octets de la réponse, préfixés par
// dataVersion. Strong validator : tout changement d'octet (ou de version) change
// l'ETag.
func computeETag(dataVersion string, body []byte) string {
	h := sha256.New()
	if dataVersion != "" {
		_, _ = h.Write([]byte(dataVersion))
		_, _ = h.Write([]byte{'\n'})
	}
	_, _ = h.Write(body)
	return `"` + hex.EncodeToString(h.Sum(nil)) + `"`
}

// etagMatches implémente la comparaison faible d'If-None-Match (RFC 9110 §13.1.2,
// §8.8.3.2) : "*" matche tout, sinon comparaison de la valeur opaque token par
// token, préfixe W/ ignoré de part et d'autre.
func etagMatches(ifNoneMatch, etag string) bool {
	ifNoneMatch = strings.TrimSpace(ifNoneMatch)
	if ifNoneMatch == "" {
		return false
	}
	if ifNoneMatch == "*" {
		return true
	}
	want := strings.TrimPrefix(etag, "W/")
	for _, tok := range strings.Split(ifNoneMatch, ",") {
		if strings.TrimPrefix(strings.TrimSpace(tok), "W/") == want {
			return true
		}
	}
	return false
}
