## Contrat

OpenAPI **3.1**, `api/openapi.yaml` = source de vérité. Généré par ogen. Opérations groupées (`x-ogen-operation-group` : Cars, Manufacturers, Playlist, Tunes, Liveries, Leaderboards, Sessions).

## Versioning

Préfixe `/v1`. Semver du contrat. Breaking change → /v2 + dépréciation annoncée (headers Deprecation/Sunset). CHANGELOG du contrat tenu. Diff de spec en CI.

## Pagination

`page` (1-based) + `page_size` (défaut 50, max 100). Réponses *List : items + total + page + page_size.

## Filtres

`game` **obligatoire** sur les ressources multi-jeux. cars : make, class, pi_min/pi_max, drivetrain, q (recherche name/model).

## Auth

Header `X-API-Key` (SecurityHandler ogen). Clés opaques hashées. Lecture publique possible ; writes (tunes, sessions) authentifiés.

## Rate-limit

Sliding window par clé (Redis). Headers X-RateLimit-Limit/Remaining/Reset. 429 + Retry-After au dépassement.

## Erreurs

**RFC 9457** (application/problem+json) : type, title, status, detail, instance. Codes stables documentés.

## Cache

cars : Cache-Control long + ETag (304). playlist : TTL court + stale-while-revalidate. Cloudflare Cache Rules par endpoint.