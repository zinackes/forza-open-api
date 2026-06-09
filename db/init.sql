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
    class         TEXT CHECK (class IN ('D','C','B','A','S1','S2','X')),
    pi            INT  CHECK (pi BETWEEN 100 AND 999),
    drivetrain    TEXT CHECK (drivetrain IN ('FWD','RWD','AWD')),
    stats         JSONB,
    body_type     TEXT,
    rarity        TEXT,
    value_cr      BIGINT,
    obtain_method TEXT,
    image_url     TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS cars_game_idx       ON cars (game);
CREATE INDEX IF NOT EXISTS cars_game_class_idx ON cars (game, class);
CREATE INDEX IF NOT EXISTS cars_game_pi_idx    ON cars (game, pi);
CREATE INDEX IF NOT EXISTS cars_game_make_idx  ON cars (game, make);

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

CREATE TABLE IF NOT EXISTS events (
    id                    TEXT PRIMARY KEY,
    game                  TEXT NOT NULL,
    name                  TEXT NOT NULL,
    type                  TEXT NOT NULL CHECK (type IN ('circuit','road','dirt','cross','street','touge_battle','horizon_rush','drag_meet','time_attack')),
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

-- Clés API (jamais la clé en clair : seul le hash sha256 est stocké) -----------
CREATE TABLE IF NOT EXISTS api_keys (
    key_hash   TEXT PRIMARY KEY,
    name       TEXT,
    rate_limit INT,
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
