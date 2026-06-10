# @forza-open-api/client

Typed TypeScript SDK for the [Forza Open API](https://github.com/zinackes/forza-open-api),
**generated from the OpenAPI contract** (`api/openapi.yaml`). Types come from
[`openapi-typescript`](https://github.com/openapi-ts/openapi-typescript); the
runtime client is [`openapi-fetch`](https://openapi-ts.dev/openapi-fetch/) — a
~6 kB typed wrapper over `fetch`. The package version tracks the contract version.

## Install

```bash
npm i @forza-open-api/client
```

## Usage

```ts
import { createClient } from "@forza-open-api/client";

// Auth is optional. Without a key you read public data; a key raises rate
// limits and unlocks authenticated writes.
const client = createClient({ apiKey: process.env.FORZA_API_KEY });

const { data, error } = await client.GET("/v1/cars", {
  params: { query: { game: "fh6", page_size: 20 } },
});

if (error) {
  // RFC 9457 problem payload, fully typed.
  console.error(error.type, error.title);
} else {
  for (const car of data.items) {
    console.log(car.id);
  }
}
```

`createClient` returns a standard `openapi-fetch` client, so every path,
parameter, request body and response in the contract is type-checked. Override
`baseUrl` to target a self-hosted instance, and pass any `fetch` option
(`fetch`, `headers`, middleware via `client.use(...)`, …).

Schema object types are re-exported for convenience:

```ts
import type { Schemas, paths, components } from "@forza-open-api/client";

type Car = Schemas["Car"];
```

## Versioning & releases

The contract is the source of truth. `npm run sync-version` copies
`info.version` from `api/openapi.yaml` into `package.json`. A push of a
`client-v<version>` git tag triggers the publish workflow
(`.github/workflows/release-client.yml`), which verifies the tag matches the
contract version and publishes to npm.

> **Maintainers — one-time setup:** create the npm org `forza-open-api` and add
> an `NPM_TOKEN` repository secret (automation token with publish rights) before
> the first release.

## Development

```bash
npm install
npm run generate   # regenerate src/schema.d.ts from ../../api/openapi.yaml
npm run build      # emit dist/ (ESM + .d.ts)
npm run check      # type-check src + the typed-call compile test
```

`src/schema.d.ts` is generated and committed; CI fails if it drifts from the
contract (`task client:generate` + clean working tree). Never edit it by hand —
change `api/openapi.yaml` and regenerate.
