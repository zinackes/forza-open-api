Outil : **API REST ouverte, gratuite et communautaire** pour les données **Forza Horizon** (FH6 d'abord, FH5 en backfill). Ni scraper du jeu, ni outil propriétaire. **Contract-first.**

## Règles produit — non négociables

- **Zéro gris.** Uniquement des sources propres : web public officiel (forza.net), datasets communautaires, télémétrie officielle (Data Out), crowdsourcing. **Jamais** de scraping du jeu, de lecture mémoire, d'injection ou de botting — EAC = ban hardware.
- **Le contrat est la source de vérité.** `api/openapi.yaml` (OpenAPI 3.1) d'abord ; le serveur Go (ogen) et les clients TS en découlent. On ne code pas un endpoint avant de l'avoir mis au contrat.
- **Game-agnostic.** Champ `game` (fh6, fh5, …) partout, dès le schéma.
- **~0 €.** Self-host cheap (Hetzner ARM) + cache edge (Cloudflare). Pas de service managé coûteux.
- **EAC-safe.** Toute capture (télémétrie, CV) est en lecture seule, hors-process. Doute → on ne le fait pas.

Détails → `docs/PRINCIPLES.md`. Doctrine data → `docs/DATA-SOURCES.md`.

## Discipline de code (principes Karpathy, adaptés)

1. **Réfléchis avant de coder.** N'assume pas, expose les tradeoffs ; si c'est flou, arrête-toi et demande.
2. **Simplicité d'abord.** Le minimum de code qui résout le problème. Pas de spéculatif, pas d'abstraction pour un seul usage.
3. **Changements chirurgicaux.** Touche seulement le nécessaire. Ne reformate pas ce que tu n'as pas cassé.
4. **Exécution orientée objectif.** Critères de succès mesurables + vérification avant d'écrire, puis boucle jusqu'à vérifié.

## Repo (API Go unique)

- `cmd/api/` — serveur net/http. `cmd/seed/`, `cmd/keys/` — outils.
- `internal/oas/` — **généré par ogen** (ne pas éditer à la main).
- `internal/handler/` — implémente l'interface oas.Handler. `internal/store/` — pgx. `internal/ingest/` — scrapers.
- `api/openapi.yaml` — le contrat. `db/` — schéma/migrations. `docs/` — référence. `.claude/` — règles, agents, skills, config.

## Commandes (via Taskfile)

`task tidy` · `task generate` (ogen) · `task run` · `task build` · `task test` · `task up`/`down`. Réf : Taskfile.yml — ne pas dupliquer ici.

## Workflow

- **Contrat d'abord** : tout changement d'API passe par api/openapi.yaml puis `task generate`.
- **Lance les tests avant de dire « fini ».** Le code généré doit matcher le spec (CI spec-drift).
- **Conventional Commits.** Ne pousse jamais sur `main`. Plan d'abord pour 3+ étapes ; checkpoint .md avant un /clear.

## Outillage Claude Code

- Si `.codegraph/` existe : **préfère mcp__codegraph__* à Read/Grep/Glob**.
- Enforcement dur (gofmt, secrets, bash dangereux) via **hooks** (settings.json), pas ce fichier.

## Règles modulaires (chargées à la demande)

@.claude/rules/go.md

@.claude/rules/api-contract.md

@.claude/rules/data.md

@.claude/rules/git.md

@.claude/rules/testing.md

@.claude/rules/securite.md