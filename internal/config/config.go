// Package config charge la configuration du service depuis l'environnement.
package config

import (
	"log/slog"
	"os"
	"strconv"
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
	}
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
