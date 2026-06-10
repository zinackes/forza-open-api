"""Typed usage example for the Forza Open API Python client.

Also doubles as the end-to-end verification script: it makes real, typed calls
against a running API and exits non-zero if anything fails.

    # against a local instance (task up)
    FORZA_BASE_URL=http://localhost:8080 python examples/list_cars.py

    # against production, with an optional API key for higher rate limits
    FORZA_API_KEY=... python examples/list_cars.py

Auth is optional: without a key you read public data; a key raises rate limits
and unlocks authenticated writes. It is never required.
"""

from __future__ import annotations

import os

from forza_open_api_client import (
    ApiClient,
    ApiException,
    CarList,
    CarsApi,
    Configuration,
    Game,
    Meta,
    MetaApi,
)

DEFAULT_BASE_URL = "https://api.forza-open-api.org"


def build_config() -> Configuration:
    config = Configuration(host=os.environ.get("FORZA_BASE_URL", DEFAULT_BASE_URL))
    api_key = os.environ.get("FORZA_API_KEY")
    if api_key:
        # Sent as the `X-API-Key` header (security scheme `ApiKeyAuth`).
        config.api_key["ApiKeyAuth"] = api_key
    return config


def main() -> int:
    with ApiClient(build_config()) as client:
        # Service metadata — no auth, no data required. Returns a typed `Meta`.
        meta: Meta = MetaApi(client).get_meta()
        print(f"generatedAt={meta.generated_at.isoformat()}")
        for g in meta.games:
            print(f"  {g.game.value}: {g.car_count} cars")

        # Catalogue listing — `game` is a required, typed enum (Game.FH6 / FH5).
        cars: CarList = CarsApi(client).list_cars(game=Game.FH6, page_size=20)
        print(f"cars page 1/{cars.total} total (page_size={cars.page_size})")
        for car in cars.items:
            # `car` is a typed `Car`; attribute access is checked by mypy.
            print(f"  {car.id}: {car.make} {car.model}")

    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except ApiException as exc:
        # RFC 9457 problem payload is available on exc.body / exc.data.
        print(f"API error {exc.status}: {exc.reason}")
        raise SystemExit(1)
