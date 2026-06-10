# Rate limits & caching

The API is read‑mostly and largely static, so being a good client is easy: send
your key, honour the caching headers, and don't poll faster than the data
changes.

## Quotas

| | Anonymous (no key) | With an API key |
|---|---|---|
| Per‑key quota | None — traffic is shaped at the edge | **1000 requests / 60 s** by default (raisable per key) |
| Identified by | Source IP, at the edge (Cloudflare) | The key's hash |
| Guaranteed throughput | No | Yes, up to your quota |

The quota is a **sliding window** (Redis). "1000 / 60 s" means at any instant,
the last 60 seconds may contain at most 1000 of your requests. The window width
is the server's `RATE_LIMIT_WINDOW` (default 60 s); the limit is per‑key.

If the rate‑limit backend is unavailable, the API **fails open** — a legitimate
client is never turned into a `429` by an infrastructure hiccup.

## Reading the headers

Keyed responses carry both the historical `X-RateLimit-*` headers and the
structured [IETF `RateLimit`](https://datatracker.ietf.org/doc/draft-ietf-httpapi-ratelimit-headers/)
fields:

```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 998
X-RateLimit-Reset: 57            ; seconds until the window frees up
RateLimit-Policy: "default";q=1000;w=60
RateLimit: "default";r=998;t=57  ; r = remaining, t = seconds to reset
```

Watch `X-RateLimit-Remaining` and back off before it hits zero.

## When you exceed it: `429`

You get `429 Too Many Requests` with a `Retry-After` (seconds) and an
[RFC 9457](https://www.rfc-editor.org/rfc/rfc9457) body:

```http
HTTP/1.1 429 Too Many Requests
Retry-After: 12
Content-Type: application/problem+json
```
```json
{ "type": "urn:forza-open-api:problem:rate-limited", "title": "Too Many Requests",
  "status": 429, "detail": "rate limit exceeded", "instance": "/v1/cars" }
```

**Respect `Retry-After`.** A minimal backoff:

```python
r = session.get(url, params=params, timeout=10)
if r.status_code == 429:
    time.sleep(int(r.headers.get("Retry-After", "1")))
    r = session.get(url, params=params, timeout=10)
r.raise_for_status()
```

## Caching: do less, go faster

The single biggest thing you can do for both your quota and your latency is to
**not re‑fetch unchanged data**.

### 1. Honour `Cache-Control`

Each endpoint declares how cacheable it is:

| Endpoint | `Cache-Control` | Meaning |
|---|---|---|
| `GET /v1/cars`, `/v1/cars/{id}`, `/v1/cars/compare` | `public, max-age=3600, s-maxage=86400, stale-while-revalidate=86400` | Catalogue is near‑static — cache for an hour locally, a day at the edge. |
| `GET /v1/playlist/*` | `public, max-age=300, stale-while-revalidate=3600` | Rotates ~weekly; short TTL is plenty. |
| `GET /v1/meta` | `public, max-age=60, stale-while-revalidate=300` | Freshness indicator; short cache. |
| `GET /v1/cars/random` | `no-store` | Must re‑roll every call — never cache it. |

A browser, a CDN, or any HTTP cache (e.g. `requests-cache`) will respect these
automatically. **The catalogue changes only when an ingestion run publishes new
data**, so an hour of local caching is safe.

### 2. Revalidate with `ETag` / `If-None-Match`

Cacheable `GET`s return a strong `ETag`. Store it, and send it back as
`If-None-Match` on the next request:

```bash
curl -si "$BASE/v1/cars?game=fh6" -H 'If-None-Match: "a1b2c3..."' | head -1
# => HTTP/1.1 304 Not Modified   (no body)
```

On `304` you reuse your cached copy — no payload to download or parse. The
`ETag` changes whenever the data (or the server's `DATA_VERSION`) changes, so a
`200` means there's genuinely something new. The browser's native cache does
this for you; in scripts, keep a small `{url: (etag, body)}` map (see the
[Python example](./examples.md#python)).

## Polling cadence

Match your poll interval to how often the data actually moves:

| Data | Changes | Suggested poll |
|---|---|---|
| Car catalogue | On ingestion runs (infrequent) | Hourly at most — or only on `ETag` change |
| Festival Playlist | ~Weekly (reset Thu 14:30 UTC) | Every 10–15 min is generous |
| Forzathon Shop | Weekly (Thu 14:30 UTC rotation) | Hourly |
| `/v1/meta` | Tracks freshness | A minute |

## Best‑practice checklist

- ✅ Send an identifiable **`User-Agent`**.
- ✅ Honour **`Cache-Control`**; store and replay **`ETag`** via `If-None-Match`.
- ✅ Respect **`Retry-After`** on `429`; back off, don't hammer.
- ✅ **Batch** with `ids=` (up to 100) instead of N single‑car requests.
- ✅ Sync incrementally with **`updated_since`** (RFC 3339) on `/v1/cars` and
  `/v1/tracks` — fetch only what changed.
- ✅ Use a **key** if you need guaranteed throughput; keep it server‑side.
- ❌ Don't poll faster than the data changes. ❌ Don't cache `/v1/cars/random`.

## Next steps

- **[Getting an API key](./api-keys.md)** for a guaranteed quota.
- **[Examples](./examples.md)** with caching wired in.
- Background on the edge/cache design: [`docs/CACHING.md`](../CACHING.md).
