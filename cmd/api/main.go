// Command api est le serveur HTTP du Forza Open API (net/http stdlib, aucun framework).
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zinackes/forza-open-api/internal/config"
	"github.com/zinackes/forza-open-api/internal/handler"
	"github.com/zinackes/forza-open-api/internal/oas"
	"github.com/zinackes/forza-open-api/internal/store"
)

func main() {
	cfg := config.Load()

	// Mode sonde : `api -healthcheck` interroge /healthz et sort 0 (sain) / 1.
	// Utilisé par la directive HEALTHCHECK Docker, l'image distroless n'ayant
	// ni shell ni curl pour une sonde externe.
	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
		os.Exit(runHealthcheck(cfg))
	}

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

	oasSrv, err := oas.NewServer(handler.New(st), handler.SecurityHandler{},
		oas.WithErrorHandler(handler.ProblemErrorHandler))
	if err != nil {
		logger.Error("oas server", "err", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz(st))
	mux.Handle("/", oasSrv) // routes du contrat (/v1/...) ; /healthz reste prioritaire

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("listening", "addr", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("listen", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown", "err", err)
	}
}

// runHealthcheck interroge /healthz en local et renvoie un code de sortie
// conforme à HEALTHCHECK Docker (0 = sain). Sonde sans dépendance externe,
// adaptée à l'image distroless.
func runHealthcheck(cfg config.Config) int {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1"+cfg.Addr+"/healthz", nil)
	if err != nil {
		return 1
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

// healthz pingue Postgres et Valkey/Redis ; 200 si tout est up, 503 sinon.
func healthz(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		body := map[string]string{"status": "ok", "postgres": "ok", "redis": "ok"}
		code := http.StatusOK
		if err := st.PingPostgres(ctx); err != nil {
			body["postgres"], body["status"], code = "down", "degraded", http.StatusServiceUnavailable
		}
		if err := st.PingRedis(ctx); err != nil {
			body["redis"], body["status"], code = "down", "degraded", http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(body)
	}
}
