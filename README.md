# Forza Open API

API REST **ouverte, gratuite et communautaire** pour les données **Forza Horizon**
(FH6 d'abord, FH5 en backfill). Ni scraper du jeu, ni outil propriétaire.
**Contract-first** : `api/openapi.yaml` est la source de vérité ; le serveur Go
(ogen) et les clients TS en découlent.

## Principes non négociables

- **Zéro gris** — uniquement des sources propres (web officiel, datasets
  communautaires, télémétrie officielle Data Out, crowdsourcing). Jamais de
  scraping du jeu, lecture mémoire, injection ni botting (EAC = ban hardware).
- **Le contrat est la vérité** — aucun endpoint n'est codé avant d'être au contrat.
- **Game-agnostic** — champ `game` (fh6, fh5, …) partout, dès le schéma.
- **~0 €** — self-host cheap (Hetzner ARM) + cache edge (Cloudflare).
- **EAC-safe** — toute capture est en lecture seule, hors-process.

Détails → `docs/PRINCIPLES.md`, doctrine data → `docs/DATA-SOURCES.md`.

## Stack

Go 1.26 (net/http stdlib, **aucun framework**) · ogen (OpenAPI 3.1) · PostgreSQL 18
(pgx v5) · Valkey/Redis · Docker Compose · Cloudflare (edge).

## Démarrage

```bash
task up          # Postgres 18 + Valkey
task generate    # (re)génère internal/oas depuis api/openapi.yaml
task run         # lance l'API sur :8080
curl localhost:8080/healthz
```

Autres tâches : `task tidy` · `task build` · `task test` · `task down`
(réf : `Taskfile.yml`).

## Structure

```
cmd/api/        serveur net/http (+ /healthz)
cmd/seed/       ingestion catalogue (stub)
cmd/keys/       gestion des clés API (stub)
internal/config/  config via env
internal/oas/     généré par ogen — NE PAS éditer à la main
internal/handler/ implémente oas.Handler (stubs 501 en Phase 0)
internal/store/   pools pgx + go-redis
api/openapi.yaml  LE contrat (source de vérité)
db/init.sql       schéma Phase 0
docs/             référence (architecture, principes, décisions, données)
clients/ts/       SDK TypeScript généré depuis le contrat (openapi-typescript + openapi-fetch)
```

## Contribuer

Contract-first : toute évolution d'API passe par `api/openapi.yaml` puis
`task generate` puis le handler. Conventional Commits. Voir `CLAUDE.md` et
`.claude/rules/` pour les règles détaillées.
