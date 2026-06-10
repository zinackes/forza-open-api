import createOpenapiFetchClient, {
  type Client,
  type ClientOptions,
} from "openapi-fetch";
import type { components, paths } from "./schema.js";

/** Production Forza Open API server, taken from the OpenAPI contract. */
export const DEFAULT_BASE_URL = "https://api.forza-open-api.org";

export interface ForzaClientOptions extends Omit<ClientOptions, "headers"> {
  /**
   * Optional API key sent as the `X-API-Key` header. Auth is optional: without
   * a key the client reads public data; a key raises rate limits and unlocks
   * authenticated writes. Never required.
   */
  apiKey?: string;
  /** Extra default headers merged into every request. */
  headers?: Record<string, string>;
}

/** A fully typed client for every path in the Forza Open API contract. */
export type ForzaClient = Client<paths>;

/**
 * Create a typed client for the Forza Open API.
 *
 * @example
 * const client = createClient({ apiKey: process.env.FORZA_API_KEY });
 * const { data, error } = await client.GET("/v1/cars", {
 *   params: { query: { game: "fh6", page_size: 20 } },
 * });
 */
export function createClient(options: ForzaClientOptions = {}): ForzaClient {
  const { apiKey, baseUrl, headers, ...rest } = options;
  return createOpenapiFetchClient<paths>({
    ...rest,
    baseUrl: baseUrl ?? DEFAULT_BASE_URL,
    headers: {
      ...(apiKey ? { "X-API-Key": apiKey } : {}),
      ...headers,
    },
  });
}

export default createClient;

/** Convenience alias for the contract's `components.schemas` object types. */
export type Schemas = components["schemas"];

export type { components, operations, paths } from "./schema.js";
