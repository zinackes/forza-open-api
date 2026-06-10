# Quickstart (5 minutes)

Goal: go from zero to a working API call in five minutes. **No signup, no key,
no SDK required to start** — reads are public.

## 0. The one thing to know

- Base URL: `https://api.forza-open-api.org` *(placeholder domain — use
  `http://localhost:8080` if you run the API locally).*
- The `game` query parameter is **required** on multi‑game resources. Use `fh6`
  (Forza Horizon 6) or `fh5` (Forza Horizon 5).

## 1. Your first request (no key)

List S1‑class cars in FH6:

```bash
curl -s "https://api.forza-open-api.org/v1/cars?game=fh6&class=S1&page_size=5"
```

You get a paginated envelope:

```json
{
  "items": [
    {
      "id": "fh6-mazda-rx7-1997",
      "game": "fh6",
      "name": "1997 Mazda RX-7",
      "make": "Mazda",
      "model": "RX-7",
      "year": 1997,
      "class": "A",
      "pi": 800,
      "drivetrain": "RWD",
      "bodyType": "Coupe",
      "category": "Retro Sports Cars",
      "valueCr": 45000,
      "obtainMethod": "autoshow"
    }
  ],
  "total": 137,
  "page": 1,
  "pageSize": 5
}
```

Every list response has the same shape: `items`, `total`, `page`, `pageSize`.

## 2. Filter and sort

Common `/v1/cars` filters (all optional except `game`):

| Param | Example | Meaning |
|---|---|---|
| `make` | `make=Ford` | Manufacturer |
| `class` | `class=S1` | PI class (`D,C,B,A,S1,S2,X,R`) |
| `pi_min` / `pi_max` | `pi_min=800` | Performance Index range (100–999) |
| `drivetrain` | `drivetrain=RWD` | `FWD,RWD,AWD` |
| `q` | `q=supra` | Full‑text on name/model |
| `sort` | `sort=-pi` | `pi,name,year,value` (`-` = descending) |
| `page` / `page_size` | `page=2&page_size=50` | Pagination (max 100) |

```bash
# The 10 highest-PI rear-wheel-drive Toyotas in FH6
curl -s "https://api.forza-open-api.org/v1/cars?game=fh6&make=Toyota&drivetrain=RWD&sort=-pi&page_size=10"
```

Fetch one car by id, or a random one:

```bash
curl -s "https://api.forza-open-api.org/v1/cars/fh6-mazda-rx7-1997"
curl -s "https://api.forza-open-api.org/v1/cars/random?game=fh6&class=S1"
```

## 3. Pick your language

<details open>
<summary><b>TypeScript (SDK)</b></summary>

```bash
npm install @forza-open-api/client
```

```ts
import createClient from "@forza-open-api/client";

const client = createClient(); // no key needed for reads

const { data, error } = await client.GET("/v1/cars", {
  params: { query: { game: "fh6", class: "S1", page_size: 5 } },
});

if (error) throw new Error(error.title); // RFC 9457 problem
console.log(data.items.map((c) => c.name));
```

The client is fully typed from the contract — `data` narrows to `CarList`,
and an invalid `game` is a compile‑time error.
</details>

<details>
<summary><b>Python (requests)</b></summary>

```python
import requests

r = requests.get(
    "https://api.forza-open-api.org/v1/cars",
    params={"game": "fh6", "class": "S1", "page_size": 5},
    timeout=10,
)
r.raise_for_status()
for car in r.json()["items"]:
    print(car["name"], car["pi"])
```
</details>

<details>
<summary><b>Browser (fetch)</b></summary>

```js
const res = await fetch(
  "https://api.forza-open-api.org/v1/cars?game=fh6&class=S1&page_size=5"
);
const { items } = await res.json();
console.log(items.map((c) => c.name));
```
</details>

## 4. (Optional) Get a key

You only need a key for **higher rate limits** or **authenticated writes**.
Send it as the `X-API-Key` header:

```bash
curl -s "https://api.forza-open-api.org/v1/cars?game=fh6" \
  -H "X-API-Key: $FORZA_API_KEY"
```

See **[Getting an API key](./api-keys.md)**.

## 5. Check what's available

```bash
curl -s "https://api.forza-open-api.org/v1/meta"        # supported games, volumes, data freshness
curl -s "https://api.forza-open-api.org/v1/reference?game=fh6"  # enum values + counts to build filters
curl -s "https://api.forza-open-api.org/v1/search?game=fh6&q=supra"  # multi-resource autocomplete
```

## Next steps

- **[Examples](./examples.md)** — Discord bot, stream overlay, copy‑paste recipes.
- **[Rate limits & caching](./rate-limits.md)** — quotas + `ETag` revalidation.
- Full reference: `GET /openapi.yaml`.
