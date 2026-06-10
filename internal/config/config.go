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
	// CORSAllowedOrigins : origines autorisées pour la lecture publique (GET/HEAD).
	// Large par défaut ("*") car l'API en lecture est ouverte ; env
	// CORS_ALLOWED_ORIGINS (CSV). "*" → toute origine.
	CORSAllowedOrigins []string
	// CORSWriteOrigins : origines autorisées pour les writes (POST/PUT/PATCH/
	// DELETE). Restreint par défaut (vide → aucun write navigateur) ; env
	// CORS_WRITE_ORIGINS (CSV). À renseigner explicitement par déploiement.
	CORSWriteOrigins []string
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
		// Lecture publique ouverte par défaut ; writes fermés tant que des
		// origines ne sont pas explicitement autorisées.
		CORSAllowedOrigins: splitCSV(getenv("CORS_ALLOWED_ORIGINS", "*")),
		CORSWriteOrigins:   splitCSV(os.Getenv("CORS_WRITE_ORIGINS")),
	}
}

// splitCSV découpe une liste CSV d'environnement en éléments non vides trimés.
// Absente/vide → slice nil (aucune origine).
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
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
