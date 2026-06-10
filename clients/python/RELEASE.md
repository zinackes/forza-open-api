# Releasing `forza-open-api-client` (PyPI)

## ⚠️ TODO — one-time setup before the first publish

The publish workflow (`.github/workflows/release-client-python.yml`) will fail
until these are done. None of them live in the repo — they are manual actions on
PyPI / GitHub.

- [ ] **PyPI account** — create an account on https://pypi.org and verify the email.
- [ ] **Reserve the project name** — the dist name is `forza-open-api-client`
      (normalized from `forza_open_api_client`). The name is claimed by the first
      upload; the workflow does that, but make sure it isn't already taken by
      someone else. If it is, change `packageName` in
      `clients/python/openapi-generator-config.yaml`, regenerate, and update this
      file + `README.md`.
- [ ] **Generate an API token** — pypi.org → Account settings → API tokens →
      *Add API token*. Scope it to the whole account for the first upload, then
      re-scope it to the project afterwards. Copy it once (`pypi-...`).
- [ ] **Add the GitHub secret `PYPI_TOKEN`** — paste the token:
      ```
      gh secret set PYPI_TOKEN --repo zinackes/forza-open-api
      ```
      (or repo → Settings → Secrets and variables → Actions → New repository secret).

> Prefer [Trusted Publishing](https://docs.pypi.org/trusted-publishers/) (OIDC,
> no long-lived token)? Configure the publisher on PyPI for this repo + workflow
> and swap the `twine upload` step for `pypa/gh-action-pypi-publish`. The token
> path above is the zero-config default.

## Cutting a release

The package version is **driven by the contract** (`api/openapi.yaml` →
`info.version`); `task client:py:generate` stamps it into `pyproject.toml`.

1. Make sure the SDK is merged to `main` and the contract carries the version you
   want to ship (the generated `pyproject.toml` version must match `info.version`).
2. Tag a commit on `main` that contains `clients/python/`, using the
   `client-py-v` prefix (distinct from the TS `client-v` prefix):
   ```
   git checkout main && git pull
   git tag client-py-v1.0.0      # must match info.version in api/openapi.yaml
   git push origin client-py-v1.0.0
   ```
3. The `release-client-python` workflow runs: it verifies the tag matches the
   contract version, regenerates the client, builds the sdist + wheel and runs
   `twine upload` with `PYPI_TOKEN`. Watch it under the repo's **Actions** tab.

The `client-python` CI job already guards against drift on every PR, so the
published code always matches the contract at the tagged commit.
