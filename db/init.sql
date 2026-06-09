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
