// Command scheduler rafraîchit périodiquement les données volatiles de l'API
// (Festival Playlist, Forzathon Shop) via un cron interne (robfig/cron). Process
// SÉPARÉ de l'API de lecture : l'ingestion reste isolée (cf. .claude/rules/data.md).
// Le reset hebdomadaire Forza tombe le jeudi 14:30 UTC → les jobs se déclenchent
// peu après (défaut PLAYLIST_CRON / FORZATHON_CRON « 0 15 * * 4 », fuseau forcé UTC).
//
// Usage :
//
//	scheduler          démarre le daemon cron (bloquant jusqu'à SIGINT/SIGTERM).
//	scheduler -once    exécute un rafraîchissement immédiat de toutes les sources
//	                   puis sort (run manuel, debug, one-shot CI). Code de sortie
//	                   != 0 si une source a échoué.
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/zinackes/forza-open-api/internal/config"
	"github.com/zinackes/forza-open-api/internal/health"
	"github.com/zinackes/forza-open-api/internal/ingest/forzathon"
	"github.com/zinackes/forza-open-api/internal/ingest/playlist"
	"github.com/zinackes/forza-open-api/internal/scheduler"
	"github.com/zinackes/forza-open-api/internal/store"
)

func main() {
	once := flag.Bool("once", false, "exécute un rafraîchissement immédiat puis sort")
	flag.Parse()

	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := store.New(ctx, cfg)
	if err != nil {
		logger.Error("store init", "err", err)
		os.Exit(1)
	}
	defer st.Close()

	monitor := &health.Monitor{
		Store:    st,
		Notifier: health.NewNotifier(cfg.AlertWebhookURL),
		Logger:   logger,
	}
	// Une instance de Runner par source. Le réseau d'une passe est borné par un
	// timeout (sources + délais polis) et hérite de l'annulation du daemon (SIGTERM
	// interrompt un fetch). Parse « réussi » → on contrôle les invariants (anomalie
	// silencieuse = rupture de structure source) avant de déclarer le run sain.
	playlistRunner := &scheduler.Runner{
		Source: "playlist", Games: cfg.PlaylistGames, Monitor: monitor, Logger: logger,
		Ingest: func(ctx context.Context, game string, now time.Time) (health.Report, error) {
			ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			defer cancel()
			res, err := playlist.IngestCurrent(ctx, st, game, "", now, logger)
			if err != nil {
				return health.Report{}, err
			}
			return health.Report{Records: 1, Violations: playlist.CheckRun(res, now)}, nil
		},
	}
	forzathonRunner := &scheduler.Runner{
		Source: "forzathon_shop", Games: cfg.ForzathonGames, Monitor: monitor, Logger: logger,
		Ingest: func(ctx context.Context, game string, now time.Time) (health.Report, error) {
			ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			defer cancel()
			res, err := forzathon.IngestCurrent(ctx, st, game, now, logger)
			if err != nil {
				return health.Report{}, err
			}
			return health.Report{Records: res.Items, Violations: forzathon.CheckRun(res, now)}, nil
		},
	}

	type job struct {
		spec   string
		runner *scheduler.Runner
	}
	jobs := []job{
		{cfg.PlaylistCron, playlistRunner},
		{cfg.ForzathonCron, forzathonRunner},
	}

	// -once : run synchrone immédiat de toutes les sources (run manuel / CI). Le
	// code de sortie reflète l'échec éventuel pour un usage scriptable.
	if *once {
		now := time.Now().UTC()
		var failed bool
		for _, j := range jobs {
			if err := j.runner.Run(ctx, now); err != nil {
				logger.Error("refresh (once)", "source", j.runner.Source, "err", err)
				failed = true
			}
		}
		if failed {
			os.Exit(1)
		}
		return
	}

	c := cron.New(cron.WithLocation(time.UTC), cron.WithLogger(cronLogger{logger}))
	for _, j := range jobs {
		if _, err := c.AddFunc(j.spec, func() {
			// Borne généreuse par tick ; les erreurs sont déjà loggées et alertées
			// dans Run, on ne les remonte pas plus haut.
			runCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
			defer cancel()
			_ = j.runner.Run(runCtx, time.Now().UTC())
		}); err != nil {
			logger.Error("cron schedule invalide", "source", j.runner.Source, "spec", j.spec, "err", err)
			os.Exit(1)
		}
	}

	logger.Info("scheduler up",
		"playlist_spec", cfg.PlaylistCron, "playlist_games", cfg.PlaylistGames,
		"forzathon_spec", cfg.ForzathonCron, "forzathon_games", cfg.ForzathonGames,
		"alert_webhook", cfg.AlertWebhookURL != "")
	c.Start()

	<-ctx.Done()
	logger.Info("scheduler shutting down")
	// Stop() empêche tout nouveau tick et rend un contexte clos une fois les jobs
	// en cours terminés → drain propre avant de rendre la main.
	<-c.Stop().Done()
}

// cronLogger route les logs internes de robfig/cron vers le slog structuré du
// service (pas de sortie brute hors discipline « slog only »).
type cronLogger struct{ l *slog.Logger }

func (c cronLogger) Info(msg string, kv ...any) { c.l.Info("cron: "+msg, kv...) }

func (c cronLogger) Error(err error, msg string, kv ...any) {
	c.l.Error("cron: "+msg, append([]any{"err", err}, kv...)...)
}
