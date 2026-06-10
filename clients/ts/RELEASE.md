# Releasing `@forza-open-api/client`

## ⚠️ TODO — one-time setup before the first publish

The publish workflow (`.github/workflows/release-client.yml`) will fail until
these are done. None of them live in the repo — they are manual actions on
npm / GitHub.

- [ ] **npm account + login** — create an account on npmjs.com, then `npm login`
      locally (`npm whoami` to verify).
- [ ] **Create the npm org `forza-open-api`** — npmjs.com → avatar →
      *Add Organization* (https://www.npmjs.com/org/create). Name must be exactly
      `forza-open-api` (it is the scope of `@forza-open-api/client`). Free plan is
      enough for public packages.
      > If the name is taken, change the package scope in `package.json`
      > (e.g. `@zinackes/forza-open-api`) and update this file + `README.md`.
- [ ] **Generate an automation token** — npmjs.com → avatar → *Access Tokens* →
      *Generate New Token* → *Classic Token* → type **Automation** (bypasses 2FA
      OTP in CI). Copy it once.
- [ ] **Add the GitHub secret `NPM_TOKEN`** — paste the token:
      ```
      gh secret set NPM_TOKEN --repo zinackes/forza-open-api
      ```
      (or repo → Settings → Secrets and variables → Actions → New repository secret).

## Cutting a release

The package version is **driven by the contract** (`api/openapi.yaml` →
`info.version`). `npm run sync-version` copies it into `package.json`.

1. Make sure the SDK is merged to `main` and the contract carries the version you
   want to ship.
2. Tag a commit on `main` that contains `clients/ts/`, using the `client-v` prefix:
   ```
   git checkout main && git pull
   git tag client-v1.0.0      # must match info.version in api/openapi.yaml
   git push origin client-v1.0.0
   ```
3. The `release-client` workflow runs: it verifies the tag matches the contract
   version, regenerates the types, builds, and runs `npm publish --access public`
   with `NPM_TOKEN`. Watch it under the repo's **Actions** tab.

The `client` CI job already guards against drift on every PR, so the published
types always match the contract at the tagged commit.
