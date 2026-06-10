# Getting an API key

**You probably don't need one.** All read endpoints are public — you can list
cars, read the playlist, search, and build a whole app with no key at all.

Get a key only when you want:

- **Higher rate limits** — anonymous traffic is protected at the edge; a key
  gives you a dedicated, generous per‑key quota (default **1000 requests per
  60 s** window, raisable per key).
- **Authenticated writes** — community contributions like tunes and sessions
  require a key (moderated, anti‑spam).

## How keys work

- A key is an **opaque random string**. Send it as the `X-API-Key` request
  header.
- The server stores only its **SHA‑256 hash** — the plaintext key is never
  persisted and never logged. It is shown **once**, at creation, and is
  irrecoverable afterwards.
- Keys are **revocable**. A revoked or unknown key returns `401`.
- Auth is **optional by contract**: a missing key means "public read", not an
  error. Only writes and quota‑raising require one.

## Requesting a key (hosted API)

The hosted service has no self‑serve signup yet. To request a key for
`api.forza-open-api.org`, open an issue on the project repository describing your
use case (bot, overlay, app) and expected volume. Treat the key you receive as a
secret (see below).

## Issuing a key (self‑hosted)

If you run your own instance, the `keys` CLI manages keys against your database
and Redis. It prints the plaintext key **once** on stdout:

```bash
# create a key named "discord-bot" with the default quota
go run ./cmd/keys create discord-bot
# => k_live_8f3c...   <-- copy this now, it is never shown again

# create with a custom per-window quota (requests per RATE_LIMIT_WINDOW)
go run ./cmd/keys create heavy-importer 5000

# list keys (hashes only — never the plaintext)
go run ./cmd/keys list

# revoke by hash (from `list`)
go run ./cmd/keys revoke <key_hash>
```

The quota is the number of requests allowed per sliding window
(`RATE_LIMIT_WINDOW`, default 60 s). See **[Rate limits](./rate-limits.md)**.

## Using your key

Send it on every request as `X-API-Key`:

```bash
export FORZA_API_KEY="k_live_..."        # keep it out of your shell history
curl -s "https://api.forza-open-api.org/v1/cars?game=fh6" \
  -H "X-API-Key: $FORZA_API_KEY"
```

```ts
import createClient from "@forza-open-api/client";
const client = createClient({ apiKey: process.env.FORZA_API_KEY });
```

```python
import os, requests
session = requests.Session()
session.headers["X-API-Key"] = os.environ["FORZA_API_KEY"]
```

## Keeping your key safe

- **Server‑side only.** Never ship a key in browser JavaScript, a public repo,
  or a stream overlay loaded in OBS — anyone can read it. For those, call the
  public (keyless) endpoints, or proxy through your own backend that holds the
  key.
- **Use environment variables / a secret store**, not source code.
- **Rotate** by creating a new key and revoking the old one — keys are cheap.
- If a key leaks, **revoke it immediately**.

## Next steps

- **[Examples](./examples.md)** — including a key‑less browser overlay and a
  server‑side Discord bot.
- **[Rate limits & caching](./rate-limits.md)** — what your quota buys and how
  to read the headers.
