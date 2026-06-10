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

## Erreurs

**RFC 9457** (application/problem+json) : type, title, status, detail, instance. Codes stables documentés.

## Cache

cars : Cache-Control long + ETag (304). playlist : TTL court + stale-while-revalidate. Cloudflare Cache Rules par endpoint.