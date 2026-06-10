// Command api est le serveur HTTP du Forza Open API (net/http stdlib, aucun framework).
package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zinackes/forza-open-api/api"
	"github.com/zinackes/forza-open-api/internal/config"
	"github.com/zinackes/forza-open-api/internal/handler"
	"github.com/zinackes/forza-open-api/internal/oas"
	"github.com/zinackes/forza-open-api/internal/store"
)

// llmsTxt décrit l'API pour les assistants IA (convention llms.txt) ; servi en
// statique comme /openapi.yaml — DX sans coût, aucune donnée dynamique.
//
//go:embed llms.txt
var llmsTxt []byte

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

	oasSrv, err := oas.NewServer(handler.New(st, cfg.DataVersion), handler.SecurityHandler{},
		oas.WithErrorHandler(handler.ProblemErrorHandler))
	if err != nil {
		logger.Error("oas server", "err", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz(st))
	mux.HandleFunc("GET /openapi.yaml", staticFile("application/yaml", api.OpenAPI))
	mux.HandleFunc("GET /llms.txt", staticFile("text/plain; charset=utf-8", llmsTxt))
	mux.Handle("/", oasSrv) // routes du contrat (/v1/...) ; les statiques restent prioritaires

	srv := &http.Server{
		Addr:    cfg.Addr,
		Handler: accessLog(mux),
		// Timeouts complets : sans eux une connexion lente (slowloris) retient
		// goroutine + FD indéfiniment. API GET-only → bornes courtes.
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
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

// staticFile sert un contenu embarqué immuable (contrat, llms.txt) avec un
// cache long : le contenu ne change qu'au déploiement.
func staticFile(contentType string, body []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "public, max-age=3600")
		_, _ = w.Write(body)
	}
}

// statusRecorder capture le code de statut écrit par le handler aval.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// accessLog logue chaque requête (méthode, path, statut, durée) en slog
// structuré. /healthz est exclu : sondé toutes les 5 s par Docker, il noierait
// les logs sans valeur.
func accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(rec, r)
		slog.InfoContext(r.Context(), "request",
			"method", r.Method, "path", r.URL.Path,
			"status", rec.status, "duration_ms", time.Since(start).Milliseconds())
	})
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
	defer func() { _ = resp.Body.Close() }()
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
