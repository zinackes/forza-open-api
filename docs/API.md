## Contrat

OpenAPI **3.1**, `api/openapi.yaml` = source de vérité. Généré par ogen. Opérations groupées (`x-ogen-operation-group` : Cars, Manufacturers, Playlist, Tunes, Liveries, Leaderboards, Sessions).

## Versioning

Préfixe `/v1`. Semver du contrat. Breaking change → /v2 + dépréciation annoncée (headers Deprecation/Sunset). CHANGELOG du contrat tenu. Diff de spec en CI.

## Pagination

`page` (1-based) + `page_size` (défaut 50, max 100). Réponses *List : items + total + page + page_size.

## Filtres

`game` **obligatoire** sur les ressources multi-jeux. cars : make, class, pi_min/pi_max, drivetrain, category, q (recherche name/model), dlc, updated_since. tracks/events/pr-stunts : type, region, q. barn-finds/treasure-cars : region, car_id (lookup inverse). manufacturers : country, q. dlc-packs : kind. `updated_since` (RFC 3339, cars + tracks) = sync incrémentale.

## Découverte & transverse

- `GET /openapi.yaml` : le contrat servi par l'API. `GET /llms.txt` : description pour assistants IA.
- `GET /v1/search` : recherche multi-ressources (autocomplete), bornée par `limit` (pas de pagination).
- `GET /v1/changes` : journal des changements de données (alimenté par l'ingestion) — base du « what's new », des futurs RSS/webhooks.
- `GET /v1/cars/{id}/obtain` : agrégat des voies d'obtention (autoshow, DLC, barn find, treasure, journal, mastery).
- Détail par id sur toutes les ressources listées (tracks, events, pr-stunts, barn-finds, treasure-cars, dlc-packs).

## Auth

Header `X-API-Key` (SecurityHandler ogen). Clés opaques hashées. Lecture publique possible ; writes (tunes, sessions) authentifiés.

## Rate-limit

Sliding window par clé (Redis). Headers X-RateLimit-Limit/Remaining/Reset. 429 + Retry-After au dépassement.

## CORS

L'API est consommable **depuis le navigateur** (overlays, apps web via le SDK TS). Middleware transport enveloppant tout le mux, **avant** le rate-limit et le cache.

- **Lecture (GET/HEAD)** : large. `CORS_ALLOWED_ORIGINS` (CSV, défaut `*`). `*` → `Access-Control-Allow-Origin: *` ; sinon écho de l'`Origin` autorisée + `Vary: Origin`.
- **Writes (POST/PUT/PATCH/DELETE)** : **restreints**. `CORS_WRITE_ORIGINS` (CSV, **défaut vide = aucun write navigateur**) ; à renseigner explicitement par déploiement.
- **Preflight `OPTIONS`** : court-circuité en `204` (jamais routé vers ogen, aucun quota consommé). Classe d'origines choisie selon `Access-Control-Request-Method`. `Access-Control-Max-Age: 600`.
- **Requête autorisés** (preflight) : `X-API-Key`, `Content-Type`, `If-None-Match`.
- **Réponse exposés** au JS (`Access-Control-Expose-Headers`) : `X-RateLimit-Limit/Remaining/Reset`, `RateLimit`, `RateLimit-Policy`, `Retry-After`, `ETag`.
- Pas d'`Access-Control-Allow-Credentials` : l'auth passe par l'en-tête `X-API-Key`, pas par cookie → compatible avec `Allow-Origin: *`.

**En-têtes de sécurité** (toutes réponses, en coordination avec Cloudflare qui gère TLS/HSTS) : `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`, `Cross-Origin-Resource-Policy: cross-origin`. Pas de CSP (API JSON, aucun HTML rendu).

## Erreurs

**RFC 9457** (application/problem+json) : type, title, status, detail, instance. Codes stables documentés.

## Cache

cars : `Cache-Control` long + **ETag** ; revalidation conditionnelle via `If-None-Match` → **304**. playlist : TTL court + `stale-while-revalidate`. random : `no-store`. ETag/304 posés par un middleware transport (`If-None-Match` en requête, `ETag` en réponse) ; le `Cache-Control` est au contrat. Cloudflare Cache Rules + bump de version/purge au refresh → **`docs/CACHING.md`**.