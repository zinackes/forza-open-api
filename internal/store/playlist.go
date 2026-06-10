// Festival Playlist : séries saisonnières FH (series + rewards + challenges).
// Écriture par l'ingestion (internal/ingest/playlist), jamais par un handler de
// lecture. Requêtes paramétrées only. Upsert idempotent et rejouable :
//   - la série est upsertée sur sa clé stable (ON CONFLICT id) ;
//   - une seule série courante par jeu (index partiel series_one_current_per_game)
//     → on éteint is_current des autres séries du jeu avant de poser la nouvelle ;
//   - rewards/challenges sont remplacés en bloc (DELETE puis INSERT) pour refléter
//     fidèlement la source sans accumuler d'anciens paliers (FK ON DELETE CASCADE).
//
// Le tout dans une transaction + une entrée data_changes (resource « series »,
// lue par Meta pour la fraîcheur de la playlist).
package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Series est la vue DB d'une série saisonnière de la Festival Playlist. Name et
// Season sont nil si la source ne les fournit pas (jamais inventés).
type Series struct {
	ID        string
	Game      string
	Series    int
	Name      *string // nom de la saga, ex. « Welcome to Japan »
	Season    *string // ex. « Autumn »
	Week      int
	StartsAt  time.Time
	EndsAt    time.Time
	IsCurrent bool
}

// Reward est une récompense d'une série. AtPercent (0..100) n'est rempli que pour
// les paliers de points (ratio points/total) ; nil pour les récompenses d'events
// individuels. Type est libre (series, season, event, car_pass, history).
type Reward struct {
	ID        string
	SeriesID  string
	AtPercent *int
	Type      string
	Item      string
}

// Challenge est un défi d'une série. Scope est libre (weekly, daily, seasonal).
// Requirement/Reward/ExpiresAt sont nil si absents de la source.
type Challenge struct {
	ID          string
	SeriesID    string
	Scope       string
	Name        string
	Requirement *string
	Reward      *string
	ExpiresAt   *time.Time
}

// UpsertSeries enregistre une série et ses rewards/challenges de façon idempotente
// et transactionnelle. Renvoie true si la série était nouvelle (sinon mise à jour).
// Journalise un data_changes resource « series ». Appelé par l'ingestion only.
func (s *Store) UpsertSeries(ctx context.Context, ser Series, rewards []Reward, challenges []Challenge) (added bool, err error) {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin series tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Nouvelle série ? (pour journaliser added vs updated).
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT true FROM series WHERE id = $1`, ser.ID).Scan(&exists); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return false, fmt.Errorf("probe series %s: %w", ser.ID, err)
		}
		exists = false
	}
	added = !exists

	// Une seule série courante par jeu : éteindre les autres avant de poser
	// celle-ci (sinon l'index partiel series_one_current_per_game refuserait).
	if ser.IsCurrent {
		if _, err = tx.Exec(ctx,
			`UPDATE series SET is_current = false WHERE game = $1 AND is_current AND id <> $2`,
			ser.Game, ser.ID); err != nil {
			return false, fmt.Errorf("clear current series (%s): %w", ser.Game, err)
		}
	}

	const upsert = `
INSERT INTO series (id, game, series, name, season, week, starts_at, ends_at, is_current)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
ON CONFLICT (id) DO UPDATE SET
    game = EXCLUDED.game, series = EXCLUDED.series, name = EXCLUDED.name,
    season = EXCLUDED.season, week = EXCLUDED.week, starts_at = EXCLUDED.starts_at,
    ends_at = EXCLUDED.ends_at, is_current = EXCLUDED.is_current`
	if _, err = tx.Exec(ctx, upsert, ser.ID, ser.Game, ser.Series, ser.Name,
		ser.Season, ser.Week, ser.StartsAt, ser.EndsAt, ser.IsCurrent); err != nil {
		return false, fmt.Errorf("upsert series %s: %w", ser.ID, err)
	}

	// Remplacement en bloc des enfants : rejouable sans doublon, et un palier
	// retiré côté source disparaît bien de la DB.
	if _, err = tx.Exec(ctx, `DELETE FROM rewards WHERE series_id = $1`, ser.ID); err != nil {
		return false, fmt.Errorf("clear rewards %s: %w", ser.ID, err)
	}
	if _, err = tx.Exec(ctx, `DELETE FROM challenges WHERE series_id = $1`, ser.ID); err != nil {
		return false, fmt.Errorf("clear challenges %s: %w", ser.ID, err)
	}

	batch := &pgx.Batch{}
	for _, r := range rewards {
		batch.Queue(
			`INSERT INTO rewards (id, series_id, at_percent, type, item) VALUES ($1,$2,$3,$4,$5)`,
			r.ID, ser.ID, r.AtPercent, r.Type, r.Item)
	}
	for _, c := range challenges {
		batch.Queue(
			`INSERT INTO challenges (id, series_id, scope, name, requirement, reward, expires_at)
             VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			c.ID, ser.ID, c.Scope, c.Name, c.Requirement, c.Reward, c.ExpiresAt)
	}
	if batch.Len() > 0 {
		br := tx.SendBatch(ctx, batch)
		for i := 0; i < batch.Len(); i++ {
			if _, err = br.Exec(); err != nil {
				_ = br.Close()
				return false, fmt.Errorf("insert series children %s: %w", ser.ID, err)
			}
		}
		if err = br.Close(); err != nil {
			return false, fmt.Errorf("close series children batch %s: %w", ser.ID, err)
		}
	}

	action := "updated"
	if added {
		action = "added"
	}
	summary := fmt.Sprintf("Festival Playlist %s S%02dW%d", ser.Game, ser.Series, ser.Week)
	if err = insertChanges(ctx, tx, []Change{{
		Game: ser.Game, Resource: "series", ResourceID: ser.ID,
		Action: action, Summary: &summary,
	}}); err != nil {
		return false, fmt.Errorf("record series change %s: %w", ser.ID, err)
	}

	if err = tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit series tx %s: %w", ser.ID, err)
	}
	return added, nil
}

// seriesCols est la liste de colonnes partagée par les lectures d'une série.
const seriesCols = `id, game, series, name, season, week, starts_at, ends_at, is_current`

// scanSeries projette une ligne (ordre seriesCols) en Series.
func scanSeries(row pgx.Row, ser *Series) error {
	return row.Scan(&ser.ID, &ser.Game, &ser.Series, &ser.Name, &ser.Season,
		&ser.Week, &ser.StartsAt, &ser.EndsAt, &ser.IsCurrent)
}

// ListSeries renvoie les séries connues d'un jeu, les plus récentes d'abord
// (série puis semaine décroissantes). Sans enfants : la liste reste légère ; le
// détail (rewards/challenges) est servi par SeriesDetail/CurrentSeries.
func (s *Store) ListSeries(ctx context.Context, game string) ([]Series, error) {
	rows, err := s.DB.Query(ctx,
		`SELECT `+seriesCols+` FROM series WHERE game = $1
         ORDER BY series DESC NULLS LAST, week DESC NULLS LAST`, game)
	if err != nil {
		return nil, fmt.Errorf("query series (%s): %w", game, err)
	}
	defer rows.Close()

	var out []Series
	for rows.Next() {
		var ser Series
		if err := scanSeries(rows, &ser); err != nil {
			return nil, fmt.Errorf("scan series: %w", err)
		}
		out = append(out, ser)
	}
	return out, rows.Err()
}

// SeriesDetail renvoie une série et ses rewards/challenges. (nil, nil, nil, nil)
// si l'identifiant est inconnu (le handler en fait un 404).
func (s *Store) SeriesDetail(ctx context.Context, id string) (*Series, []Reward, []Challenge, error) {
	var ser Series
	if err := scanSeries(s.DB.QueryRow(ctx, `SELECT `+seriesCols+` FROM series WHERE id = $1`, id), &ser); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil, nil
		}
		return nil, nil, nil, fmt.Errorf("query series %s: %w", id, err)
	}
	rewards, challenges, err := s.seriesChildren(ctx, id)
	if err != nil {
		return nil, nil, nil, err
	}
	return &ser, rewards, challenges, nil
}

// CurrentSeries renvoie la série courante d'un jeu (is_current) et ses enfants,
// ou (nil, …) si aucune n'est marquée courante.
func (s *Store) CurrentSeries(ctx context.Context, game string) (*Series, []Reward, []Challenge, error) {
	var ser Series
	if err := scanSeries(s.DB.QueryRow(ctx, `SELECT `+seriesCols+` FROM series WHERE game = $1 AND is_current`, game), &ser); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil, nil
		}
		return nil, nil, nil, fmt.Errorf("query current series (%s): %w", game, err)
	}
	rewards, challenges, err := s.seriesChildren(ctx, ser.ID)
	if err != nil {
		return nil, nil, nil, err
	}
	return &ser, rewards, challenges, nil
}

// seriesChildren charge les rewards et challenges d'une série, dans l'ordre de
// leur identifiant (= ordre d'ingestion, ids zero-paddés rw%02d/ch%02d).
func (s *Store) seriesChildren(ctx context.Context, id string) ([]Reward, []Challenge, error) {
	rRows, err := s.DB.Query(ctx,
		`SELECT id, series_id, at_percent, type, item FROM rewards WHERE series_id = $1 ORDER BY id`, id)
	if err != nil {
		return nil, nil, fmt.Errorf("query rewards %s: %w", id, err)
	}
	var rewards []Reward
	for rRows.Next() {
		var r Reward
		if err := rRows.Scan(&r.ID, &r.SeriesID, &r.AtPercent, &r.Type, &r.Item); err != nil {
			rRows.Close()
			return nil, nil, fmt.Errorf("scan reward: %w", err)
		}
		rewards = append(rewards, r)
	}
	rRows.Close()
	if err := rRows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate rewards %s: %w", id, err)
	}

	cRows, err := s.DB.Query(ctx,
		`SELECT id, series_id, scope, name, requirement, reward, expires_at FROM challenges WHERE series_id = $1 ORDER BY id`, id)
	if err != nil {
		return nil, nil, fmt.Errorf("query challenges %s: %w", id, err)
	}
	defer cRows.Close()
	var challenges []Challenge
	for cRows.Next() {
		var c Challenge
		if err := cRows.Scan(&c.ID, &c.SeriesID, &c.Scope, &c.Name, &c.Requirement, &c.Reward, &c.ExpiresAt); err != nil {
			return nil, nil, fmt.Errorf("scan challenge: %w", err)
		}
		challenges = append(challenges, c)
	}
	return rewards, challenges, cRows.Err()
}
