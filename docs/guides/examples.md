# Examples

Copy‑paste recipes. Reads need no key — drop the `X-API-Key` header if you don't
have one. Base URL is `https://api.forza-open-api.org` (or `http://localhost:8080`
locally).

- [curl](#curl)
- [TypeScript (SDK)](#typescript-sdk)
- [Python](#python)
- [Recipe: Discord bot](#recipe-discord-bot)
- [Recipe: stream overlay](#recipe-stream-overlay)
- [Recipe: simple browser fetch](#recipe-simple-browser-fetch)

## curl

```bash
BASE="https://api.forza-open-api.org"

# List + filter + sort (paginated envelope: items/total/page/pageSize)
curl -s "$BASE/v1/cars?game=fh6&class=S1&drivetrain=AWD&sort=-pi&page_size=10"

# One car by id
curl -s "$BASE/v1/cars/fh6-mazda-rx7-1997"

# A random car matching filters (never cached)
curl -s "$BASE/v1/cars/random?game=fh6&pi_min=900"

# Compare 2–3 cars side by side (no `game` — the id is the global key)
curl -s "$BASE/v1/cars/compare?ids=fh6-mazda-rx7-1997,fh6-mazda-furai-2008"

# Current Festival Playlist
curl -s "$BASE/v1/playlist/current?game=fh6"

# This week's Forzathon Shop
curl -s "$BASE/v1/forzathon-shop?game=fh6"

# Service metadata: supported games, volumes, data freshness
curl -s "$BASE/v1/meta"

# Conditional GET — revalidate cheaply (see Rate limits & caching)
ETAG=$(curl -s -o /dev/null -D - "$BASE/v1/cars?game=fh6" | awk 'tolower($1)=="etag:"{print $2}' | tr -d '\r')
curl -s -o /dev/null -w '%{http_code}\n' "$BASE/v1/cars?game=fh6" -H "If-None-Match: $ETAG"
# => 304   (Not Modified, no body re-downloaded)

# With a key
curl -s "$BASE/v1/cars?game=fh6" -H "X-API-Key: $FORZA_API_KEY"
```

Errors are [RFC 9457](https://www.rfc-editor.org/rfc/rfc9457)
`application/problem+json`:

```json
{
  "type": "urn:forza-open-api:problem:validation",
  "title": "Bad Request",
  "status": 400,
  "detail": "game is required",
  "instance": "/v1/cars"
}
```

`type` is a stable URN per error class (`…:validation`, `…:unauthorized`,
`…:not-found`, `…:rate-limited`); `instance` is the request path. Switch on
`status` or `type`, not on the human‑readable `title`/`detail`.

## TypeScript (SDK)

```bash
npm install @forza-open-api/client
```

```ts
import createClient, { type Schemas } from "@forza-open-api/client";

// apiKey is optional; omit it for public reads. baseUrl defaults to production.
const client = createClient({ apiKey: process.env.FORZA_API_KEY });

const { data, error } = await client.GET("/v1/cars", {
  params: { query: { game: "fh6", class: "S1", sort: "-pi", page_size: 20 } },
});

if (error) {
  // error is the typed RFC 9457 problem payload
  throw new Error(`${error.status} ${error.title}: ${error.detail ?? ""}`);
}

const cars: Schemas["Car"][] = data.items;
console.log(cars.map((c) => `${c.name} (${c.pi})`));

// Path params are typed too
const { data: car } = await client.GET("/v1/cars/{id}", {
  params: { path: { id: "fh6-mazda-rx7-1997" } },
});
```

Everything is generated from the contract: query/path params, response shapes,
and enums are checked at compile time. `createClient()` with no argument reads
public data against `https://api.forza-open-api.org`.

## Python

A small session with retries and ETag‑based caching — a good base for any
script or bot.

```python
import os
import requests

BASE = "https://api.forza-open-api.org"

session = requests.Session()
session.headers["User-Agent"] = "my-forza-app/1.0"
if key := os.environ.get("FORZA_API_KEY"):
    session.headers["X-API-Key"] = key  # optional


def get_cars(game: str = "fh6", **filters):
    r = session.get(f"{BASE}/v1/cars", params={"game": game, **filters}, timeout=10)
    r.raise_for_status()
    return r.json()


# `class` is a reserved word in Python — pass reserved/hyphenated params via a dict.
page = get_cars(game="fh6", sort="-pi", page_size=10, **{"class": "S1"})
for car in page["items"]:
    print(car["name"], car["pi"], car["drivetrain"])
```

Conditional GET (revalidate without re‑downloading):

```python
_cache: dict[str, tuple[str, object]] = {}  # url -> (etag, json)

def get_cached(url: str, **params):
    headers = {}
    if url in _cache:
        headers["If-None-Match"] = _cache[url][0]
    r = session.get(url, params=params, headers=headers, timeout=10)
    if r.status_code == 304:
        return _cache[url][1]            # unchanged, reuse cached body
    r.raise_for_status()
    if etag := r.headers.get("ETag"):
        _cache[url] = (etag, r.json())
    return r.json()
```

## Recipe: Discord bot

A `/randomcar` slash command using [discord.py](https://discordpy.readthedocs.io/).
The bot runs on your server, so it can safely hold a key (none required, though).

```python
import os
import discord
from discord import app_commands
import requests

BASE = "https://api.forza-open-api.org"
http = requests.Session()
http.headers["User-Agent"] = "forza-discord-bot/1.0"

intents = discord.Intents.default()
client = discord.Client(intents=intents)
tree = app_commands.CommandTree(client)


@tree.command(name="randomcar", description="Get a random Forza Horizon car")
@app_commands.describe(game="fh6 or fh5", min_pi="minimum Performance Index")
async def randomcar(interaction: discord.Interaction, game: str = "fh6", min_pi: int = 0):
    params = {"game": game}
    if min_pi:
        params["pi_min"] = min_pi
    r = http.get(f"{BASE}/v1/cars/random", params=params, timeout=10)

    if r.status_code == 404:  # no car matches the filters
        await interaction.response.send_message("No car matches those filters.")
        return
    r.raise_for_status()
    c = r.json()

    embed = discord.Embed(title=c["name"], description=f"{c['make']} · {c.get('category', '')}")
    embed.add_field(name="PI", value=f"{c['pi']} ({c['class']})")
    embed.add_field(name="Drivetrain", value=c["drivetrain"])
    if c.get("imageUrl"):
        embed.set_thumbnail(url=c["imageUrl"])
    await interaction.response.send_message(embed=embed)


@client.event
async def on_ready():
    await tree.sync()


client.run(os.environ["DISCORD_TOKEN"])
```

`/v1/cars/random` is `no-store`, so every call re‑rolls — exactly what you want
for a "random car of the day" command.

## Recipe: stream overlay

A single HTML file you can add as an **OBS browser source**. It shows the current
FH6 Festival Playlist and refreshes itself. It uses **only public endpoints — no
key** (overlays run client‑side, where secrets would leak).

```html
<!doctype html>
<meta charset="utf-8" />
<style>
  body { margin: 0; font-family: system-ui, sans-serif; color: #fff;
         text-shadow: 0 2px 6px #000; background: transparent; }
  .card { padding: 16px 20px; background: rgba(0,0,0,.55); border-radius: 12px;
          display: inline-block; }
  h1 { margin: 0 0 4px; font-size: 22px; }
  li { font-size: 15px; opacity: .9; }
</style>
<div class="card" id="overlay">Loading playlist…</div>

<script type="module">
  const BASE = "https://api.forza-open-api.org";
  const el = document.getElementById("overlay");

  async function render() {
    try {
      // The browser's HTTP cache + ETag handle revalidation for free.
      const res = await fetch(`${BASE}/v1/playlist/current?game=fh6`);
      if (!res.ok) return;
      const s = await res.json();
      const challenges = (s.challenges ?? [])
        .map((c) => `<li>${c.name} — ${c.requirement}</li>`)
        .join("");
      el.innerHTML = `<h1>${s.name}</h1>
        <div>Series ${s.series}${s.week ? ` · Week ${s.week}` : ""}</div>
        <ul>${challenges}</ul>`;
    } catch { /* keep last render on transient errors */ }
  }

  render();
  // The playlist rotates ~weekly; polling every 10 min is plenty and stays
  // within edge cache (see Rate limits & caching).
  setInterval(render, 10 * 60 * 1000);
</script>
```

## Recipe: simple browser fetch

Plain `fetch`, no dependencies — drop into any page or app. Public endpoints,
no key.

```js
async function searchCars(query, game = "fh6") {
  const url = new URL("https://api.forza-open-api.org/v1/cars");
  url.search = new URLSearchParams({ game, q: query, page_size: "10" });

  const res = await fetch(url);
  if (!res.ok) {
    const problem = await res.json(); // RFC 9457
    throw new Error(`${problem.status} ${problem.title}`);
  }
  const { items, total } = await res.json();
  return { cars: items, total };
}

const { cars } = await searchCars("supra");
console.log(cars.map((c) => c.name));
```

> Calling the API from a browser relies on CORS being enabled on the service.
> Never put an API key in client‑side code — use the public endpoints, or proxy
> through your own backend.

## Next steps

- **[Rate limits & caching](./rate-limits.md)** — quotas, `429`, and `ETag`.
- Full reference: `GET /openapi.yaml`.
