// Journal de santé des passes d'ingestion (scrapers) : une ligne par run, écrite
// par les jobs (cmd/seed, cmd/scheduler) via internal/health, jamais par les
// handlers de lecture. Nourrit le dashboard ops `seed health` (fraîcheur / échecs
// par source). Requêtes paramétrées only ; les violations sont stockées telles
// quelles en JSONB (ce package ne dépend pas de internal/health).
package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// ScrapeRun est une passe d'ingestion à journaliser. Game vide → NULL (sources
// sans jeu). Violations = JSON brut ([]health.Violation marshalé par l'appelant) ;
// vide → NULL. Error vide → NULL (run sans échec dur).
type ScrapeRun struct {
	Source     string
	Game       string
	Status     string // ok | anomaly | failed
	Records    int
	Violations json.RawMessage
	Error      string
	DurationMS int
	StartedAt  time.Time
	FinishedAt time.Time
}

// ScrapeRunSummary est le dernier run d'une source/jeu (vue dashboard). Game/Error
// nil quand absents en base ; Violations brut (décodé par l'appelant pour l'affichage).
type ScrapeRunSummary struct {
	Source     string
	Game       *string
	Status     string
	Records    int
	Violations json.RawMessage
	Error      *string
	FinishedAt time.Time
}

// InsertScrapeRun journalise une passe d'ingestion (id et finished_at à défaut DB
// si non fournis ; ici finished_at est fourni par l'appelant pour la cohérence).
func (s *Store) InsertScrapeRun(ctx context.Context, r ScrapeRun) error {
	const q = `
INSERT INTO scrape_runs (source, game, status, records, violations, error, duration_ms, started_at, finished_at)
VALUES ($1, NULLIF($2,''), $3, $4, $5, NULLIF($6,''), $7, $8, $9)`
	var violations any
	if len(r.Violations) > 0 {
		violations = []byte(r.Violations)
	}
	if _, err := s.DB.Exec(ctx, q,
		r.Source, r.Game, r.Status, r.Records, violations, r.Error,
		r.DurationMS, r.StartedAt, r.FinishedAt,
	); err != nil {
		return fmt.Errorf("insert scrape_run (%s/%s): %w", r.Source, r.Game, err)
	}
	return nil
}

// LatestScrapeRuns renvoie le dernier run de chaque (source, jeu), trié pour un
// affichage stable. DISTINCT ON garde la ligne la plus récente par groupe.
func (s *Store) LatestScrapeRuns(ctx context.Context) ([]ScrapeRunSummary, error) {
	const q = `
SELECT DISTINCT ON (source, coalesce(game,''))
       source, game, status, coalesce(records,0), violations, error, finished_at
FROM scrape_runs
ORDER BY source, coalesce(game,''), finished_at DESC`
	rows, err := s.DB.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query scrape_runs: %w", err)
	}
	defer rows.Close()

	var out []ScrapeRunSummary
	for rows.Next() {
		var r ScrapeRunSummary
		if err := rows.Scan(&r.Source, &r.Game, &r.Status, &r.Records,
			&r.Violations, &r.Error, &r.FinishedAt); err != nil {
			return nil, fmt.Errorf("scan scrape_run: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate scrape_runs: %w", err)
	}
	return out, nil
}
