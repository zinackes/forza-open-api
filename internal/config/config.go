// Package config charge la configuration du service depuis l'environnement.
package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config regroupe les paramètres runtime du serveur API.
type Config struct {
	Addr        string     // adresse d'écoute HTTP (ex. ":8080")
	DatabaseURL string     // DSN Postgres (pgx)
	RedisURL    string     // URL Valkey/Redis
	LogLevel    slog.Level // niveau de log slog
	DataVersion string     // version du jeu de données publiée, exposée par /v1/meta (vide = non tamponnée)
	// RateLimitWindow : largeur de la fenêtre glissante du rate-limit par clé.
	// Le quota (api_keys.rate_limit) s'entend « requêtes par fenêtre ». Env
	// RATE_LIMIT_WINDOW en secondes (défaut 60s).
	RateLimitWindow time.Duration
	// Scheduler (cmd/scheduler) — rafraîchissement périodique des données
	// volatiles. Les crons sont des specs 5 champs (fuseau UTC), défaut « 0 15 * *
	// 4 » = jeudi 15:00 UTC, peu après le reset hebdo Forza (14:30 UTC). Les *Games
	// sont les jeux rafraîchis (CSV, défaut « fh6 »).
	//   - PlaylistCron / PlaylistGames : Festival Playlist.
	//   - ForzathonCron / ForzathonGames : Forzathon Shop (rotation hebdo).
	// AlertWebhookURL : webhook d'alerte sur échec (vide = log structuré seul).
	PlaylistCron    string
	PlaylistGames   []string
	ForzathonCron   string
	ForzathonGames  []string
	AlertWebhookURL string
}

// Load lit la config depuis l'environnement avec des défauts orientés dev local.
func Load() Config {
	return Config{
		Addr:            getenv("API_ADDR", ":8080"),
		DatabaseURL:     getenv("DATABASE_URL", "postgres://forza:forza@localhost:5432/forza?sslmode=disable"),
		RedisURL:        getenv("REDIS_URL", "redis://localhost:6379/0"),
		LogLevel:        parseLevel(getenv("LOG_LEVEL", "info")),
		DataVersion:     os.Getenv("DATA_VERSION"),
		RateLimitWindow: time.Duration(getenvInt("RATE_LIMIT_WINDOW", 60)) * time.Second,
		PlaylistCron:    getenv("PLAYLIST_CRON", "0 15 * * 4"),
		PlaylistGames:   splitCSV(getenv("PLAYLIST_GAMES", "fh6")),
		ForzathonCron:   getenv("FORZATHON_CRON", "0 15 * * 4"),
		ForzathonGames:  splitCSV(getenv("FORZATHON_GAMES", "fh6")),
		AlertWebhookURL: os.Getenv("ALERT_WEBHOOK_URL"),
	}
}

// splitCSV découpe une liste CSV en éléments propres (trim, minuscules, vides
// écartés). Sert aux jeux du scheduler (« fh6,fh5 »).
func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.ToLower(strings.TrimSpace(p)); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// getenvInt lit un entier strictement positif depuis l'environnement ; toute
// valeur absente, non numérique ou <= 0 retombe sur def.
func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
