## Vue d'ensemble

API REST en lecture (+ quelques writes crowdsourcés) servant des données Forza Horizon, dérivées de sources propres, mises en cache à l'edge. Stateless.

## Composants

- **API Go** (net/http + serveur ogen) — sert le contrat OpenAPI 3.1.
- **PostgreSQL 18** — source de vérité (catalogue, playlist, clés, tunes/liveries, sessions). pgx v5.
- **Valkey/Redis** — cache, rate-limit (sliding window Lua), leaderboards (sorted sets).
- **Ingestion** (`internal/ingest` + `cmd/seed`) — catalogue (datasets/GitHub/wiki), playlist (forza.net/forums/wiki), planifiée (cron/pg-boss).
- **Companion télémétrie** (desktop, Go) — capte Data Out (UDP), agrège, upload (Phase 8).
- **Edge** — Cloudflare (Cache Rules + stale-while-revalidate) via Cloudflare Tunnel.
- **Clients** — SDK TS généré (openapi-typescript + openapi-fetch).

## Flux de données

Sources propres → ingestion (normalisation, upsert idempotent) → Postgres → API (handlers pgx) → cache edge → consommateurs.

Télémétrie : jeu → companion (UDP, lecture seule) → POST /v1/sessions → pg-boss → perfs → leaderboards (Redis sorted sets).

## Déploiement

Hetzner Cloud ARM (CAX), Docker Compose/Coolify, EU. Cloudflare Tunnel (pas d'IP publique). ~5€/mois.

## Principes structurants

Contract-first (ogen), API stateless, game-agnostic, cache edge agressif sur le quasi-statique. Escape hatches analytics : Postgres (TimescaleDB) → ClickHouse seulement à l'échelle. Choix détaillés → `docs/DECISIONS.md`.