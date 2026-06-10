// Compile-only check (never executed) — type-checked by `npm run check`.
// Proves the generated contract types flow through the client: a valid call
// type-checks, the response narrows to the contract schema, and an invalid
// query param is rejected at compile time.
import { createClient, type Schemas } from "../src/index.js";

const client = createClient({ apiKey: "optional-api-key" });

export async function example(): Promise<void> {
  const { data, error } = await client.GET("/v1/cars", {
    params: { query: { game: "fh6", page_size: 20 } },
  });

  if (error) {
    // RFC 9457 problem payload, typed from the contract's Error schema.
    console.error(error.type);
    return;
  }

  // data is typed as CarList — items is Car[].
  const cars: Schemas["Car"][] = data.items;
  console.log(cars.length);

  // Path parameters are typed too.
  await client.GET("/v1/cars/{id}", { params: { path: { id: "audi-r8" } } });

  // @ts-expect-error — `game` is a required enum, an arbitrary string is rejected.
  await client.GET("/v1/cars", { params: { query: { game: "not-a-game" } } });

  // @ts-expect-error — `game` is required; omitting it must not type-check.
  await client.GET("/v1/cars", { params: { query: { page_size: 5 } } });
}
