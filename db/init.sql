-- Forza Open API — schéma Phase 0 (bootstrap).
-- Idempotent (IF NOT EXISTS) : rejouable sans casser une base existante.
-- Migrations ultérieures via golang-migrate. Toutes les tables portent `game`.
-- snake_case en DB ; le contrat OpenAPI expose des vues camelCase.

-- Catalogue voitures ----------------------------------------------------------
CREATE TABLE IF NOT EXISTS cars (
    id            TEXT PRIMARY KEY,
    game          TEXT NOT NULL,
    name          TEXT NOT NULL,
    make          TEXT NOT NULL,
    model         TEXT,
    year          INT,
    class         TEXT NOT NULL CHECK (class IN ('D','C','B','A','S1','S2','X','R')),
    pi            INT  NOT NULL CHECK (pi BETWEEN 100 AND 999),
    drivetrain    TEXT NOT NULL CHECK (drivetrain IN ('FWD','RWD','AWD')),
    stats         JSONB,
    body_type     TEXT,
    category      TEXT,
    rarity        TEXT,
    value_cr      BIGINT,
    obtain_method TEXT,
    image_url     TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- Bases créées avant l'ajout de updated_at : CREATE TABLE IF NOT EXISTS ne
-- complète pas les colonnes → ALTER idempotent pour les upgrades en place.
ALTER TABLE cars ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
-- Le contrat exige class/pi/drivetrain (la vue Car les scanne en non-pointeurs :
-- une ligne NULL ferait des 500 sur /v1/cars) et l'ingestion écarte déjà les
-- voitures incomplètes (cars.Validate). SET NOT NULL est rejouable (no-op si
-- déjà posé) ; il échouerait sur des lignes NULL héritées — impossibles via
-- l'ingestion, à nettoyer à la main le cas échéant.
ALTER TABLE cars ALTER COLUMN class      SET NOT NULL;
ALTER TABLE cars ALTER COLUMN pi         SET NOT NULL;
ALTER TABLE cars ALTER COLUMN drivetrain SET NOT NULL;
CREATE INDEX IF NOT EXISTS cars_game_idx          ON cars (game);
CREATE INDEX IF NOT EXISTS cars_game_class_idx    ON cars (game, class);
CREATE INDEX IF NOT EXISTS cars_game_pi_idx       ON cars (game, pi);
CREATE INDEX IF NOT EXISTS cars_game_make_idx     ON cars (game, make);
-- Facette categories de GET /v1/reference (GROUP BY category scopé au jeu).
CREATE INDEX IF NOT EXISTS cars_game_category_idx ON cars (game, category);
-- Recherche q (ILIKE '%…%') : un pattern à wildcard de tête ne peut pas user
-- d'un btree → index trigram GIN sur name/model pour éviter le seq scan.
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX IF NOT EXISTS cars_name_trgm_idx  ON cars USING gin (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS cars_model_trgm_idx ON cars USING gin (model gin_trgm_ops);

-- Constructeurs ---------------------------------------------------------------
CREATE TABLE IF NOT EXISTS manufacturers (
    game    TEXT NOT NULL,
    name    TEXT NOT NULL,
    country TEXT,
    PRIMARY KEY (game, name)
);

-- Festival Playlist : séries saisonnières -------------------------------------
CREATE TABLE IF NOT EXISTS series (
    id         TEXT PRIMARY KEY,
    game       TEXT NOT NULL,
    series     INT,
    name       TEXT,
    season     TEXT,
    week       INT,
    starts_at  TIMESTAMPTZ,
    ends_at    TIMESTAMPTZ,
    is_current BOOLEAN NOT NULL DEFAULT false
);
-- Une seule série courante par jeu.
CREATE UNIQUE INDEX IF NOT EXISTS series_one_current_per_game
    ON series (game) WHERE is_current;

CREATE TABLE IF NOT EXISTS rewards (
    id         TEXT PRIMARY KEY,
    series_id  TEXT NOT NULL REFERENCES series (id) ON DELETE CASCADE,
    at_percent INT CHECK (at_percent BETWEEN 0 AND 100),
    type       TEXT,
    item       TEXT
);
CREATE INDEX IF NOT EXISTS rewards_series_idx ON rewards (series_id);

CREATE TABLE IF NOT EXISTS challenges (
    id          TEXT PRIMARY KEY,
    series_id   TEXT NOT NULL REFERENCES series (id) ON DELETE CASCADE,
    scope       TEXT,
    name        TEXT,
    requirement TEXT,
    reward      TEXT,
    expires_at  TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS challenges_series_idx ON challenges (series_id);

-- Forzathon Shop : rotation hebdomadaire d'objets contre Forza Points ---------
-- FH6 confirme la mécanique : chaque jeudi (reset 14:30 UTC) une nouvelle vague
-- d'objets (voitures, klaxons, vêtements, phrases Forza LINK…) achetables contre
-- des Forza Points (qui se reportent d'une semaine à l'autre). Sources propres
-- (forza.net + forums + wiki Fandom) ; jamais le jeu. car_id réf. la voiture du
-- catalogue pour les objets kind=car identifiés (NULL sinon, ou si non rapproché).
-- Champs non sourcés → NULL (jamais inventés). week_end NULL si la source ne donne
-- pas la borne de fin. Clé naturelle (game, week_start, name) → upsert idempotent.
CREATE TABLE IF NOT EXISTS forzathon_shop_items (
    id            TEXT PRIMARY KEY,
    game          TEXT NOT NULL,
    week_start    TIMESTAMPTZ NOT NULL,
    week_end      TIMESTAMPTZ,
    kind          TEXT NOT NULL CHECK (kind IN ('car','horn','clothing','forza_link_phrase','other')),
    car_id        TEXT REFERENCES cars (id) ON DELETE SET NULL,
    name          TEXT NOT NULL,
    fp_cost       INT,
    description   TEXT,
    image_url     TEXT,
    source        TEXT,
    last_verified TIMESTAMPTZ
);
-- Un seul objet par (jeu, semaine, nom) : garde-fou d'intégrité + cible de
-- l'upsert idempotent (ON CONFLICT) — rejouable sans doublon.
CREATE UNIQUE INDEX IF NOT EXISTS forzathon_shop_items_key
    ON forzathon_shop_items (game, week_start, name);
-- Rotation courante + historique paginé « plus récentes d'abord » par jeu.
CREATE INDEX IF NOT EXISTS forzathon_shop_items_game_week_idx
    ON forzathon_shop_items (game, week_start DESC);
CREATE INDEX IF NOT EXISTS forzathon_shop_items_car_idx ON forzathon_shop_items (car_id);

-- Carte : tracés, PR stunts, événements --------------------------------------
-- Fonde les leaderboards (Phase 8) et la carte (Phase 9). Sources propres
-- (wiki Fandom, datasets communautaires) ; jamais le jeu. Coords NULL si absentes.
CREATE TABLE IF NOT EXISTS tracks (
    id            TEXT PRIMARY KEY,
    game          TEXT NOT NULL,
    name          TEXT NOT NULL,
    type          TEXT NOT NULL CHECK (type IN ('circuit','road','dirt','cross','street','touge','horizon_rush')),
    region        TEXT,
    length_m      INT,
    surface_mix   TEXT,
    start_lat     NUMERIC,
    start_lng     NUMERIC,
    source        TEXT,
    last_verified TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS tracks_game_type_idx   ON tracks (game, type);
CREATE INDEX IF NOT EXISTS tracks_game_region_idx ON tracks (game, region);
-- q (ILIKE '%…%') et /v1/search : trigram, même raison que cars_name_trgm_idx.
CREATE INDEX IF NOT EXISTS tracks_name_trgm_idx   ON tracks USING gin (name gin_trgm_ops);

CREATE TABLE IF NOT EXISTS pr_stunts (
    id           TEXT PRIMARY KEY,
    game         TEXT NOT NULL,
    type         TEXT NOT NULL CHECK (type IN ('speed_trap','speed_zone','drift_zone','danger_sign')),
    name         TEXT NOT NULL,
    region       TEXT,
    lat          NUMERIC,
    lng          NUMERIC,
    target_score INT
);
CREATE INDEX IF NOT EXISTS pr_stunts_game_type_idx   ON pr_stunts (game, type);
CREATE INDEX IF NOT EXISTS pr_stunts_game_region_idx ON pr_stunts (game, region);
CREATE INDEX IF NOT EXISTS pr_stunts_name_trgm_idx   ON pr_stunts USING gin (name gin_trgm_ops);

CREATE TABLE IF NOT EXISTS events (
    id                    TEXT PRIMARY KEY,
    game                  TEXT NOT NULL,
    name                  TEXT NOT NULL,
    type                  TEXT NOT NULL CHECK (type IN ('circuit','road','dirt','cross','street','touge_battle','horizon_rush','drag_meet','time_attack','showcase')),
    region                TEXT,
    start_lat             NUMERIC,
    start_lng             NUMERIC,
    end_lat               NUMERIC,
    end_lng               NUMERIC,
    route_geojson         JSONB,
    car_class_restriction TEXT,
    length_m              INT
);
CREATE INDEX IF NOT EXISTS events_game_type_idx   ON events (game, type);
CREATE INDEX IF NOT EXISTS events_game_region_idx ON events (game, region);
CREATE INDEX IF NOT EXISTS events_name_trgm_idx   ON events USING gin (name gin_trgm_ops);

-- DLC / extensions : Car Pass, expansions, standalone ------------------------
-- Référence des packs (sources propres : annonces forza.net + wiki Fandom).
-- released_at NULL = pack annoncé mais pas encore sorti (expansions planifiées).
-- Lien N-N vers cars via car_dlc ; permet de lister les voitures d'un pack.
CREATE TABLE IF NOT EXISTS dlc_packs (
    id            TEXT PRIMARY KEY,
    game          TEXT NOT NULL,
    name          TEXT NOT NULL,
    kind          TEXT NOT NULL CHECK (kind IN ('car_pass','expansion','standalone')),
    released_at   TIMESTAMPTZ,
    description   TEXT,
    source        TEXT,
    last_verified TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS dlc_packs_game_idx ON dlc_packs (game);

CREATE TABLE IF NOT EXISTS car_dlc (
    car_id TEXT NOT NULL REFERENCES cars (id)      ON DELETE CASCADE,
    dlc_id TEXT NOT NULL REFERENCES dlc_packs (id) ON DELETE CASCADE,
    PRIMARY KEY (car_id, dlc_id)
);
-- Le PK couvre les lookups par car_id (préfixe) ; index dédié pour filtrer par
-- pack (« voitures du DLC X »).
CREATE INDEX IF NOT EXISTS car_dlc_dlc_idx ON car_dlc (dlc_id);

-- Voitures cachées FH6 : Barn Finds & Treasure Cars (2 mécaniques distinctes) --
-- Sources propres (wiki Fandom + guides commu : GamesRadar, MitchCactus…).
-- Coords NULL si non sourcées. Tout porte game ; car_id réf. la voiture obtenue.
-- Barn Finds : épaves cachées dans une zone de recherche, déblocage progressif
-- via les stamps Discover Japan (prerequisite_stamp_level 1-7 : Visitor →
-- Master Explorer), puis restauration (restoration_time_h).
CREATE TABLE IF NOT EXISTS barn_finds (
    id                       TEXT PRIMARY KEY,
    car_id                   TEXT NOT NULL REFERENCES cars (id) ON DELETE CASCADE,
    game                     TEXT NOT NULL,
    region                   TEXT,
    search_zone_center_lat   NUMERIC,
    search_zone_center_lng   NUMERIC,
    search_zone_radius_m     INT,
    prerequisite_stamp_level INT CHECK (prerequisite_stamp_level BETWEEN 1 AND 7),
    restoration_time_h       INT,
    source                   TEXT,
    last_verified            TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS barn_finds_game_idx        ON barn_finds (game);
CREATE INDEX IF NOT EXISTS barn_finds_game_region_idx ON barn_finds (game, region);
CREATE INDEX IF NOT EXISTS barn_finds_car_idx         ON barn_finds (car_id);

-- Treasure Cars : voitures liées aux postcards (indice texte), conduisibles
-- immédiatement après la cutscene de lavage. Pas de stamp ni de restauration.
CREATE TABLE IF NOT EXISTS treasure_cars (
    id                 TEXT PRIMARY KEY,
    car_id             TEXT NOT NULL REFERENCES cars (id) ON DELETE CASCADE,
    game               TEXT NOT NULL,
    region             TEXT,
    postcard_clue_text TEXT,
    location_lat       NUMERIC,
    location_lng       NUMERIC,
    source             TEXT,
    last_verified      TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS treasure_cars_game_idx        ON treasure_cars (game);
CREATE INDEX IF NOT EXISTS treasure_cars_game_region_idx ON treasure_cars (game, region);
CREATE INDEX IF NOT EXISTS treasure_cars_car_idx         ON treasure_cars (car_id);

-- Catalogue des pièces d'upgrade & upgrades par voiture -----------------------
-- Sources propres (dataset forzagarage.com, wiki Fandom). Sourcing progressif :
-- voitures populaires d'abord, extension par séries ensuite. Deltas NULL si non
-- sourcés (précision > exhaustivité). Tout porte game.
-- upgrade_parts : pièce générique (delta PI/poids/puissance/couple par palier).
CREATE TABLE IF NOT EXISTS upgrade_parts (
    id              TEXT PRIMARY KEY,
    game            TEXT NOT NULL,
    category        TEXT NOT NULL CHECK (category IN ('engine','drivetrain','aspiration','tires','weight','aero','brakes','transmission','intake','exhaust','cooling','fuel_system')),
    name            TEXT NOT NULL,
    level           INT CHECK (level >= 1),
    pi_delta        INT,
    weight_delta_kg INT,
    power_delta_hp  INT,
    torque_delta_nm INT,
    source          TEXT,
    last_verified   TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS upgrade_parts_game_idx          ON upgrade_parts (game);
CREATE INDEX IF NOT EXISTS upgrade_parts_game_category_idx ON upgrade_parts (game, category);

-- car_upgrades : N-N voiture ↔ pièce + contraintes d'installation.
-- requires_part_id : pièce prérequise (chaîne d'upgrade). exclusive_group : une
-- seule pièce du groupe montable à la fois (ex. compounds de pneus).
CREATE TABLE IF NOT EXISTS car_upgrades (
    car_id           TEXT NOT NULL REFERENCES cars (id)          ON DELETE CASCADE,
    part_id          TEXT NOT NULL REFERENCES upgrade_parts (id) ON DELETE CASCADE,
    requires_part_id TEXT REFERENCES upgrade_parts (id)          ON DELETE SET NULL,
    exclusive_group  TEXT,
    PRIMARY KEY (car_id, part_id)
);
-- Le PK couvre les lookups par car_id (préfixe) ; index dédié pour retrouver les
-- voitures qui montent une pièce donnée.
CREATE INDEX IF NOT EXISTS car_upgrades_part_idx ON car_upgrades (part_id);

-- Car Mastery FH6 : grille de perks 4×4 par voiture ---------------------------
-- Sources propres (dataset forzagarage.com qui indexe déjà les ~622 arbres, wiki
-- Fandom). Sourcing progressif. Une perk = une case (row, col) de la grille,
-- débloquée contre des Skill Points (sp_cost). effect_type est LIBRE (valeurs
-- courantes : credits, xp_boost, wheelspin, super_wheelspin, skill_score,
-- car_unlock, …). prereq_perk_id chaîne les perks ; unlocked_car_id pointe la
-- voiture débloquée par une perk car_unlock (hidden cars FH6 : Corvette Stingray
-- 427 via Stingray Coupe, Honda Civic RS via Civic Type R, Ferrari F50 GT via
-- F50, Ford Supervan 4 via Supervan 3, …). Pas de game : dérivé via car_id.
CREATE TABLE IF NOT EXISTS car_mastery_perks (
    id                 TEXT PRIMARY KEY,
    car_id             TEXT NOT NULL REFERENCES cars (id) ON DELETE CASCADE,
    row                INT  CHECK (row BETWEEN 1 AND 4),
    col                INT  CHECK (col BETWEEN 1 AND 4),
    name               TEXT,
    sp_cost            INT,
    effect_description TEXT,
    effect_type        TEXT,
    effect_value       INT,
    prereq_perk_id     TEXT REFERENCES car_mastery_perks (id) ON DELETE SET NULL,
    unlocked_car_id    TEXT REFERENCES cars (id)              ON DELETE SET NULL,
    source             TEXT,
    last_verified      TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS car_mastery_perks_car_idx          ON car_mastery_perks (car_id);
CREATE INDEX IF NOT EXISTS car_mastery_perks_unlocked_car_idx ON car_mastery_perks (unlocked_car_id);

-- Collection Journal FH6 : paliers de progression (remplace les Accolades FH5) -
-- 2 pistes (track) : horizon_festival = 7 wristbands (Yellow → Gold ; Gold
-- débloque Legend Island + The Goliath), discover_japan = 7 stamps (Visitor →
-- Master Explorer ; poussent les Barn Finds). 17 voitures ne sont débloquables
-- que via reward_car_id. Sources propres (wiki Fandom + forza.net). color ne
-- s'applique qu'aux wristbands (NULL pour les stamps). Champs non sourcés → NULL.
CREATE TABLE IF NOT EXISTS journal_tiers (
    id                  TEXT PRIMARY KEY,
    game                TEXT NOT NULL,
    track               TEXT NOT NULL CHECK (track IN ('horizon_festival','discover_japan')),
    level               INT  NOT NULL CHECK (level BETWEEN 1 AND 7),
    color               TEXT CHECK (color IN ('yellow','green','blue','pink','orange','purple','gold')),
    name                TEXT NOT NULL,
    points_required     INT,
    reward_car_id       TEXT REFERENCES cars (id) ON DELETE SET NULL,
    unlocks_description TEXT,
    source              TEXT,
    last_verified       TIMESTAMPTZ
);
-- Un seul palier par (jeu, piste, niveau) : garde-fou d'intégrité + upsert
-- idempotent sur la clé naturelle. Couvre aussi le filtre game (+ track) en préfixe.
CREATE UNIQUE INDEX IF NOT EXISTS journal_tiers_game_track_level_key
    ON journal_tiers (game, track, level);
-- Lookup inverse « quel palier débloque la voiture X ».
CREATE INDEX IF NOT EXISTS journal_tiers_reward_car_idx ON journal_tiers (reward_car_id);

-- Stories & Tours FH6 : contenus Discovery ------------------------------------
-- Stories = missions narratives (Discover Japan), Tours = visites guidées.
-- Les deux rapportent des stamps au Collection Journal. Sources propres (wiki
-- Fandom). Champs non sourcés → NULL. Tout porte game.
CREATE TABLE IF NOT EXISTS stories (
    id                 TEXT PRIMARY KEY,
    game               TEXT NOT NULL,
    name               TEXT NOT NULL,
    region             TEXT,
    description        TEXT,
    chapters_count     INT CHECK (chapters_count >= 1),
    reward_description TEXT,
    source             TEXT,
    last_verified      TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS stories_game_idx        ON stories (game);
CREATE INDEX IF NOT EXISTS stories_game_region_idx ON stories (game, region);

CREATE TABLE IF NOT EXISTS tours (
    id            TEXT PRIMARY KEY,
    game          TEXT NOT NULL,
    name          TEXT NOT NULL,
    region        TEXT,
    description   TEXT,
    source        TEXT,
    last_verified TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS tours_game_idx        ON tours (game);
CREATE INDEX IF NOT EXISTS tours_game_region_idx ON tours (game, region);

-- Journal des changements de données (GET /v1/changes) -------------------------
-- Alimenté par les jobs d'ingestion (jamais par les handlers de lecture).
-- resource est LIBRE (car, dlc_pack, series…) : pas de CHECK pour ne pas figer
-- l'ensemble au schéma. Nourrit le « what's new » des clients + futurs RSS/webhooks.
CREATE TABLE IF NOT EXISTS data_changes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    game        TEXT NOT NULL,
    resource    TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    action      TEXT NOT NULL CHECK (action IN ('added','updated','removed')),
    summary     TEXT,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- Liste paginée « plus récents d'abord » filtrée par jeu (+ resource/action en
-- résiduel) : l'index (game, occurred_at DESC) porte le tri.
CREATE INDEX IF NOT EXISTS data_changes_game_occurred_idx ON data_changes (game, occurred_at DESC);

-- Santé des scrapers : une ligne par passe d'ingestion ------------------------
-- Écrite par les jobs (cmd/seed, cmd/scheduler) via internal/health ; jamais par
-- les handlers de lecture. status='anomaly' = parse « réussi » mais données
-- suspectes (rupture de structure source probable) ; 'failed' = échec dur.
-- Nourrit le dashboard ops `seed health` (fraîcheur / échecs par source).
CREATE TABLE IF NOT EXISTS scrape_runs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source      TEXT NOT NULL,                       -- playlist | cars | tracks
    game        TEXT,
    status      TEXT NOT NULL CHECK (status IN ('ok','anomaly','failed')),
    records     INT,                                 -- volume ingéré (séries, voitures, tracés)
    violations  JSONB,                               -- [{rule,detail,severity}] (NULL si ok)
    error       TEXT,                                -- message d'échec dur (NULL sinon)
    duration_ms INT,
    started_at  TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- Dernier run par source/jeu (dashboard) : DISTINCT ON (source, game) ORDER BY finished_at DESC.
CREATE INDEX IF NOT EXISTS scrape_runs_source_game_idx ON scrape_runs (source, game, finished_at DESC);

-- Clés API (jamais la clé en clair : seul le hash sha256 est stocké) -----------
CREATE TABLE IF NOT EXISTS api_keys (
    key_hash   TEXT PRIMARY KEY,
    name       TEXT,
    rate_limit INT NOT NULL DEFAULT 1000,
    -- Scopes accordés à la clé (ex. read, submit-ugc). Toute clé peut lire ;
    -- les scopes d'écriture (submit-ugc…) seront exigés par la feature writes.
    scopes     TEXT[] NOT NULL DEFAULT '{read}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at TIMESTAMPTZ
);

-- Crowdsourcing : tunes & liveries --------------------------------------------
CREATE TABLE IF NOT EXISTS tunes (
    id         TEXT PRIMARY KEY,
    car_id     TEXT REFERENCES cars (id),
    game       TEXT NOT NULL,
    share_code TEXT,
    surface    TEXT CHECK (surface IN ('road','dirt','cross')),
    tune_type  TEXT,
    pi         INT CHECK (pi BETWEEN 100 AND 999),
    title      TEXT,
    notes      TEXT,
    upvotes    INT NOT NULL DEFAULT 0,
    author_id  TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS tunes_car_idx  ON tunes (car_id);
CREATE INDEX IF NOT EXISTS tunes_game_idx ON tunes (game);

CREATE TABLE IF NOT EXISTS liveries (
    id         TEXT PRIMARY KEY,
    car_id     TEXT REFERENCES cars (id),
    game       TEXT NOT NULL,
    share_code TEXT,
    title      TEXT,
    image_url  TEXT,
    upvotes    INT NOT NULL DEFAULT 0,
    author_id  TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS liveries_car_idx  ON liveries (car_id);
CREATE INDEX IF NOT EXISTS liveries_game_idx ON liveries (game);

-- Télémétrie (Data Out, read-only côté jeu) -----------------------------------
CREATE TABLE IF NOT EXISTS sessions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    api_key     TEXT REFERENCES api_keys (key_hash) ON DELETE SET NULL,
    game        TEXT NOT NULL,
    car_ordinal INT,
    raw         JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS sessions_game_idx ON sessions (game);

-- Manifeste des archives téléchargeables (GET /v1/exports) --------------------
-- Écrite par le job de génération (internal/export via cmd/seed | cmd/scheduler) ;
-- jamais par les handlers de lecture. Une ligne = un fichier statique (un jeu, une
-- ressource, un format) servi depuis l'edge. Les fichiers eux-mêmes vivent sur
-- l'object store / l'edge ; cette table ne sert QUE le manifeste (url/taille/etag).
-- etag = hash de contenu calculé à la génération. Clé naturelle (game, resource,
-- format) → upsert idempotent (régénération rejouable sans doublon).
CREATE TABLE IF NOT EXISTS exports (
    game         TEXT        NOT NULL,
    resource     TEXT        NOT NULL,
    format       TEXT        NOT NULL CHECK (format IN ('json','csv','jsonl')),
    url          TEXT        NOT NULL,
    size_bytes   BIGINT      NOT NULL,
    etag         TEXT        NOT NULL,
    generated_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (game, resource, format)
);
-- Listing du manifeste filtré par jeu (GET /v1/exports?game=…).
CREATE INDEX IF NOT EXISTS exports_game_idx ON exports (game);
