Format léger : contexte → décision → conséquences.

## ADR-0001 — Contract-first avec ogen

**Contexte.** API publique, clients multiples, besoin de cohérence et de SDK.

**Décision.** `api/openapi.yaml` (OpenAPI **3.1**) = source de vérité. Serveur + modèles générés par **ogen** (3.1 natif, pas de reflect, Handler + SecurityHandler générés, `x-ogen-operation-group`). Clients TS via openapi-typescript/openapi-fetch.

**Conséquences.** oapi-codegen écarté (encore OAS 3.0). Un check CI échoue si le code généré dérive du spec. On ne code pas un endpoint hors contrat.

## ADR-0002 — net/http stdlib, pas de framework

**Décision.** net/http (routing Go 1.22+) sans Gin/Echo/Fiber. Le goulot est la DB/cache, pas le routeur.

**Conséquences.** Moins de dépendances, code idiomatique, ogen monte sur http.ServeMux.

## ADR-0003 — PostgreSQL 18 + Valkey

**Décision.** Postgres = source de vérité (pgx v5). Valkey/Redis = cache + rate-limit (sliding window Lua) + leaderboards (sorted sets).

**Conséquences.** Une base relationnelle ; Redis pour le chaud. Migrations golang-migrate.

## ADR-0004 — Hetzner ARM + Cloudflare Tunnel

**Décision.** Hetzner Cloud ARM (CAX) EU, Docker Compose/Coolify ; Cloudflare Tunnel (pas d'IP publique) + Cache Rules + SWR.

**Conséquences.** ~5€/mois, cache edge gratuit, aucun port public ouvert.

## ADR-0005 — Doctrine data « zéro gris »

**Décision.** Uniquement sources propres (cf. DATA-SOURCES). Jamais de scraping du jeu, mémoire, injection, botting (EAC). Captures read-only hors-process.

**Conséquences.** Pas de marché AH live ni d'endpoints Xbox privés.

## ADR-0006 — Leaderboards construits via télémétrie

**Décision.** On ne lit pas les Rivals du jeu ; on construit nos classements depuis le Data Out officiel (companion → sessions → perfs → sorted sets Redis).

**Conséquences.** Légit, EAC-safe, mais dépend du crowdsourcing (couverture partielle assumée).

## ADR-0007 — Champ `game` (game-agnostic)

**Décision.** `game` (fh6, fh5, …) dans chaque table et endpoint dès le départ.

**Conséquences.** Survit aux transitions ; FH6 d'abord, FH5 backfill, futurs titres faciles.

## ADR-0008 — Escape hatches analytics

**Décision.** Rester sur Postgres (TimescaleDB si besoin) ; ClickHouse seulement à l'échelle (centaines de M de lignes, comme deadlock-api).

**Conséquences.** Simplicité maintenant, chemin de sortie clair plus tard.