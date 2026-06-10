// Package scheduler orchestre le rafraîchissement périodique des données
// volatiles de l'API (Festival Playlist, Forzathon Shop) déclenché par
// cmd/scheduler (cron interne). La logique d'ingestion vit dans internal/ingest ;
// le contrôle de santé et l'alerte dans internal/health. Ce package se borne à
// l'orchestration : itération des jeux, observation de chaque run
// (log/persistance/alerte via le Monitor), agrégation des erreurs. Aucune I/O
// réseau ni dépendance store directe → testable par injection (IngestFunc) sans
// toucher aux sources externes. Une instance de Runner par source (cf. Source).
package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/zinackes/forza-open-api/internal/health"
)

// IngestFunc exécute une passe d'ingestion pour un jeu et renvoie son bilan santé
// (health.Report) en plus de l'erreur dure éventuelle. Injectée par cmd/scheduler
// (p. ex. playlist.IngestCurrent + playlist.CheckRun, ou forzathon.IngestCurrent +
// forzathon.CheckRun, fermés sur le store) et remplacée par un faux en test (pas
// de réseau, cf. testing.md).
type IngestFunc func(ctx context.Context, game string, now time.Time) (health.Report, error)

// Runner rafraîchit une source (Source : « playlist », « forzathon_shop »…) pour
// les jeux configurés. Une instance par source ; réutilisé tel quel pour un run
// manuel (cmd/scheduler -once) comme pour un tick cron.
type Runner struct {
	Source  string // étiquette de la source, portée par chaque Report (scrape_runs, alerte)
	Games   []string
	Ingest  IngestFunc
	Monitor *health.Monitor // log + persistance + alerte ; nil = orchestration seule
	Logger  *slog.Logger
}

// Run rafraîchit chaque jeu configuré pour la source du Runner. Le bilan de chaque
// jeu est observé (loggé, persisté, alerté sur anomalie ou échec via le Monitor).
// Un échec sur un jeu N'AVORTE PAS les autres ; les échecs durs ET les anomalies
// de structure sont agrégés dans l'erreur renvoyée (le tick cron l'ignore, `-once`
// en fait son code de sortie pour la CI). now est injecté pour des décisions
// déterministes (rotation courante, fraîcheur) et des tests reproductibles.
func (r *Runner) Run(ctx context.Context, now time.Time) error {
	var errs []error
	for _, game := range r.Games {
		start := time.Now()
		rep, err := r.Ingest(ctx, game, now)
		rep.Source, rep.Game = r.Source, game
		if err != nil {
			rep.Err = err
		}
		if r.Monitor != nil {
			r.Monitor.Observe(ctx, rep, start)
		}
		switch {
		case err != nil:
			errs = append(errs, fmt.Errorf("%s: %w", game, err))
		case rep.HasAnomaly():
			errs = append(errs, fmt.Errorf("%s: anomalie de structure détectée", game))
		}
	}
	return errors.Join(errs...)
}
