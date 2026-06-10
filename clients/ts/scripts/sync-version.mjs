// Keep the SDK version in lockstep with the OpenAPI contract (info.version).
// The contract is the source of truth; this rewrites clients/ts/package.json's
// version to match. Run by `npm run sync-version` and in prepublishOnly.
import { readFileSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { parse } from "yaml";

const scriptDir = dirname(fileURLToPath(import.meta.url));
const contractPath = resolve(scriptDir, "../../../api/openapi.yaml");
const pkgPath = resolve(scriptDir, "../package.json");

const contract = parse(readFileSync(contractPath, "utf8"));
const contractVersion = contract?.info?.version;
if (typeof contractVersion !== "string" || contractVersion.length === 0) {
  console.error(`error: could not read info.version from ${contractPath}`);
  process.exit(1);
}

const pkg = JSON.parse(readFileSync(pkgPath, "utf8"));
if (pkg.version === contractVersion) {
  console.log(`package version already matches contract: ${contractVersion}`);
  process.exit(0);
}

const previous = pkg.version;
pkg.version = contractVersion;
writeFileSync(pkgPath, `${JSON.stringify(pkg, null, 2)}\n`);
console.log(`synced package version ${previous} -> ${contractVersion}`);
