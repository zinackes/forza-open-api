# Developer guides

Onboarding for the **Forza Open API** — a free, community, contract-first REST
API for Forza Horizon data (FH6 first, FH5 backfill).

Start here:

1. **[Quickstart](./quickstart.md)** — your first request in under 5 minutes. No
   signup required.
2. **[Getting an API key](./api-keys.md)** — when and how to get a key (higher
   limits, writes). Reads work without one.
3. **[Examples](./examples.md)** — copy‑paste recipes: `curl`, the TypeScript
   SDK, Python — plus a Discord bot, a stream overlay, and a browser fetch.
4. **[Rate limits & caching](./rate-limits.md)** — quotas, the `429` contract,
   and how to be a good client (`Cache-Control`, `ETag`/`If-None-Match`).

## At a glance

| | |
|---|---|
| Base URL (production) | `https://api.forza-open-api.org` *(placeholder domain, to be confirmed)* |
| Base URL (local dev) | `http://localhost:8080` |
| Auth | Optional `X-API-Key` header. Reads are public. |
| `game` parameter | **Required** on multi‑game resources (`fh6` or `fh5`). |
| Pagination | `page` (1‑based) + `page_size` (default `50`, max `100`). |
| Errors | [RFC 9457](https://www.rfc-editor.org/rfc/rfc9457) `application/problem+json`. |
| Contract | `GET /openapi.yaml` (the source of truth). |
| For AI assistants | `GET /llms.txt`. |

The contract is the source of truth: `api/openapi.yaml`. Every endpoint, field,
and enum below comes from it.
