# Cache HTTP & edge

L'API est en lecture seule et largement statique. La stratégie : un `Cache-Control`
posé à l'**origin** par endpoint, une **validation conditionnelle** (ETag /
If-None-Match → 304) pour rendre la revalidation gratuite, et un **cache edge**
Cloudflare (Phase 5) qui respecte ces directives. Objectif ~0 € : le bord absorbe
le trafic, l'origin ne sert que les `MISS` et les revalidations 304.

## Cache-Control par endpoint (origin)

| Endpoint | Cache-Control | Pourquoi |
|---|---|---|
| `GET /v1/cars` | `public, max-age=3600, s-maxage=86400, stale-while-revalidate=86400` | Catalogue quasi-statique (change à un run d'ingestion). Navigateur 1 h, bord 1 j, sert périmé 1 j en revalidant. |
| `GET /v1/cars/{id}` | idem | idem |
| `GET /v1/cars/compare` | idem | idem |
| `GET /v1/cars/random` | `no-store` | Tirage aléatoire : jamais mis en cache (sinon tous les clients voient la même voiture). |
| `GET /v1/meta` | `public, max-age=60, stale-while-revalidate=300` | Fraîcheur des données = objet de l'endpoint ; cache court. |
| `GET /v1/playlist/current` | `public, max-age=300, stale-while-revalidate=3600` | Festival Playlist : rafraîchie fréquemment côté jeu ; TTL court + SWR. Posé via le wrapper `*Headers` du contrat (comme cars/meta). |
| `GET /v1/playlist/series`, `…/series/{id}` | `public, max-age=300, stale-while-revalidate=3600` | idem. |
| `GET /v1/me` | `no-store` | Réponse propre à la clé appelante et volatile (quota courant) : jamais mise en cache. |

## Validation conditionnelle (ETag / 304)

Middleware `internal/handler/cache.go` (`ConditionalGet`), branché côté net/http
après le rate-limit (cf. `cmd/api/main.go`). Pour toute réponse **GET 200** non
`no-store` :

- pose un **ETag fort** = `"sha256(DATA_VERSION + "\n" + corps)"` (hex). Strong
  validator : tout changement d'octet — ou un bump de `DATA_VERSION` — change l'ETag ;
- si la requête porte un `If-None-Match` correspondant (`*`, liste CSV ou `W/` gérés,
  comparaison faible RFC 9110 §13.1.2) → **304 Not Modified**, sans corps, ETag et
  Cache-Control conservés.

Couvre cars (et tout futur GET cacheable). Transparent pour les clients qui n'envoient
pas d'If-None-Match. Non modélisé au contrat : c'est un en-tête transport, au même
titre que `X-RateLimit-*` (seul le `429` est au contrat, pas ses en-têtes).

**Côté client** : conserver l'`ETag` reçu et le renvoyer en `If-None-Match` au prochain
appel pour ne re-télécharger que si le catalogue a changé.

**CORS** (`internal/handler/cors.go`, politique → `docs/API.md`) : la couche CORS
autorise le header requête `If-None-Match` (`Access-Control-Allow-Headers`) et expose
`ETag` (`Access-Control-Expose-Headers`) pour que le SDK browser puisse faire du
conditional GET. Le cache HTTP natif du navigateur, lui, gère 304 sans exposition CORS.

## Cloudflare Cache Rules par endpoint (Phase 5)

À créer dans Dashboard → Caching → Cache Rules. Toutes respectent l'origin
(`Cache-Control`/`s-maxage`) plutôt que de figer un TTL en dur, pour garder une seule
source de vérité.

| Règle (match expression) | Eligible for cache | Edge TTL | Browser TTL | Serve stale |
|---|---|---|---|---|
| `starts_with(http.request.uri.path, "/v1/cars")` **et** `http.request.uri.path ne "/v1/cars/random"` | Oui | Respecter l'origin (`s-maxage` = 1 j) | Respecter l'origin | Activé (while revalidating) |
| `http.request.uri.path eq "/v1/cars/random"` | **Bypass cache** | — | — | — |
| `starts_with(http.request.uri.path, "/v1/playlist")` | Oui | Respecter l'origin (300 s) | Respecter l'origin | Activé |
| `http.request.uri.path eq "/v1/meta"` | Oui | Respecter l'origin (60 s) | Respecter l'origin | Activé |
| `starts_with(http.request.uri.path, "/v1/")` (catch-all lecture) | Oui (si origin cacheable) | Respecter l'origin | Respecter l'origin | Activé |

Notes :
- Le **`Vary`** doit inclure `Accept-Encoding` ; ne pas mettre `X-API-Key` en clé de
  cache (les lectures sont identiques quelle que soit la clé) — sinon le cache devient
  inutile. Configurer la Cache Key pour **ignorer** `X-API-Key` et l'en-tête
  `If-None-Match` (Cloudflare gère la revalidation via l'ETag d'origin).
- Cloudflare répond lui-même 304 sur HIT si l'ETag correspond.

## Bump de version / purge au refresh

Au refresh des données (run `cmd/seed` pour le catalogue, ingest playlist) :

1. **Bump `DATA_VERSION`** (env de l'API). Comme `DATA_VERSION` est mêlé au hash de
   l'ETag, tous les ETags changent → la prochaine revalidation client/bord renvoie un
   200 frais au lieu d'un 304 obsolète. `DATA_VERSION` est aussi exposé par `/v1/meta`.
2. **Purge Cloudflare** des chemins concernés :
   - Plan Free/Pro : **purge par préfixe d'URL** (`/v1/cars*`, `/v1/playlist*`) ou
     « Purge Everything » si besoin.
   - La purge par **Cache-Tag** (taguer les réponses `Cache-Tag: cars,fh6`) est
     réservée au plan **Enterprise** — non retenue pour le ~0 €.

Sans purge, le bord se met de toute façon à jour à l'expiration de l'`s-maxage` (ou
sert périmé via `stale-while-revalidate` en revalidant). La purge ne sert qu'à rendre
le refresh **immédiat**.
