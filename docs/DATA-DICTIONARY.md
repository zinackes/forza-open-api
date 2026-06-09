Toutes les tables portent `game` (fh6, fh5, …). snake_case en DB ; le contrat expose des vues camelCase.

## cars

id (PK, TEXT), game, name, make, model, year, class (D..X), pi (100-999), drivetrain (FWD/RWD/AWD), stats (JSONB : speed/handling/accel/launch/braking…), body_type, rarity, value_cr (BIGINT), obtain_method, image_url, created_at.

Index : (game), (game,class), (game,pi), (game,make).

## manufacturers

game, name, country. PK (game, name).

## series (Festival Playlist)

id (PK), game, series (INT), name, season, week, starts_at, ends_at, is_current (BOOL). Index partiel **unique** sur is_current par game.

## rewards

id, series_id (FK → series, ON DELETE CASCADE), at_percent, type, item.

## challenges

id, series_id (FK), scope, name, requirement, reward, expires_at.

## api_keys

key_hash (PK, sha256), name, rate_limit (INT), created_at, revoked_at. **Jamais la clé en clair.**

## tunes / liveries (crowdsourcés)

id, car_id (FK → cars), game, share_code, surface (road/dirt/cross — tunes), tune_type, pi, title, notes / image_url, upvotes (INT), author_id, created_at.

## sessions (télémétrie)

id, api_key, game, car_ordinal, raw (JSONB agrégé), created_at. Dérivées → **perfs** (0-100, vmax, freinage, lap) par voiture/tune ; **leaderboards** en Redis (sorted sets), PG = historique autoritaire.

## tracks (carte — Phase 8/9)

id (PK), game, name, type (circuit/road/dirt/cross/street/touge/horizon_rush), region, length_m (INT), surface_mix, start_lat/start_lng (NUMERIC), source, last_verified, created_at, updated_at. Index : (game,type), (game,region).

## pr_stunts (carte)

id (PK), game, type (speed_trap/speed_zone/drift_zone/danger_sign), name, region, lat/lng (NUMERIC), target_score (INT, nullable). Index : (game,type), (game,region).

## events (carte)

id (PK), game, name, type (circuit/road/dirt/cross/street/touge_battle/horizon_rush/drag_meet/time_attack), region, start_lat/lng, end_lat/lng (NUMERIC), route_geojson (JSONB, nullable), car_class_restriction, length_m (INT). Index : (game,type), (game,region).

Ingestion : datasets communautaires / wiki Fandom via `cmd/seed` (`internal/ingest/tracks`), upsert idempotent. Jamais le jeu. Coords/champs absents → NULL.

## Conventions

Types stricts, FK ON DELETE CASCADE pour rewards/challenges. Pas de donnée inventée (NULL si absent).