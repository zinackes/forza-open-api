# forza-open-api-client (Python)

Typed Python SDK for the [Forza Open API](https://github.com/zinackes/forza-open-api),
**generated from the OpenAPI contract** (`api/openapi.yaml`) with
[openapi-generator](https://openapi-generator.tech/) (`python` generator —
[urllib3](https://urllib3.readthedocs.io/) + [pydantic v2](https://docs.pydantic.dev/)
models, ships a `py.typed` marker). The package version tracks the contract version.

## Install

```bash
pip install forza-open-api-client
```

## Usage

```python
from forza_open_api_client import ApiClient, Configuration, CarsApi, MetaApi, Game

# Auth is optional. Without a key you read public data; a key raises rate
# limits and unlocks authenticated writes (sent as the `X-API-Key` header).
config = Configuration(host="https://api.forza-open-api.org")
# config.api_key["ApiKeyAuth"] = "your-api-key"  # optional

with ApiClient(config) as client:
    meta = MetaApi(client).get_meta()          # -> Meta (typed)
    print(meta.generated_at, [g.game.value for g in meta.games])

    cars = CarsApi(client).list_cars(game=Game.FH6, page_size=20)  # -> CarList
    for car in cars.items:                      # car: Car (typed)
        print(car.id, car.make, car.model)
```

Every path, parameter and response in the contract is a typed method on a
tag-grouped API class (`CarsApi`, `MetaApi`, `PlaylistApi`, …). Models are
pydantic v2 — `model_dump()` / `model_validate()` work as usual. Errors raise
`forza_open_api_client.ApiException` (the RFC 9457 problem payload is on
`exc.body`).

A runnable, typed example lives in [`examples/list_cars.py`](./examples/list_cars.py):

```bash
FORZA_BASE_URL=http://localhost:8080 python examples/list_cars.py
```

## Versioning & releases

The contract is the source of truth. The package version is set from
`info.version` of `api/openapi.yaml` at generation time (`task client:py:generate`).
A push of a `client-py-v<version>` git tag triggers the publish workflow
(`.github/workflows/release-client-python.yml`), which verifies the tag matches
the contract version and publishes to PyPI.

> **Maintainers — one-time setup:** create the PyPI project and add a
> `PYPI_TOKEN` repository secret before the first release.
> Full checklist: [RELEASE.md](./RELEASE.md).

## Development

The whole package under `forza_open_api_client/` plus `pyproject.toml` /
`setup.py` is **generated and committed** — never edit it by hand. Change
`api/openapi.yaml`, then regenerate (needs Docker; the generator runs in a
pinned `openapitools/openapi-generator-cli` image):

```bash
task client:py:generate   # regenerate from ../../api/openapi.yaml
task client:py:check      # install in a venv, smoke-import, type-check the example
```

CI fails if the generated code drifts from the contract (`task client:py:generate`
+ clean working tree), exactly like the TypeScript SDK.
