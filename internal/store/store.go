// Package store encapsule l'accès aux dépendances de données (Postgres, Valkey).
//
// Phase 0 : connexions paresseuses (aucun appel réseau au démarrage) + Ping
// pour /healthz. Les requêtes métier arriveront avec les handlers.
package store

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/zinackes/forza-open-api/internal/config"
)

// redisLogger route les logs internes de go-redis vers slog (Debug) au lieu de
// stderr brut, pour rester conforme à la discipline « slog only ».
type redisLogger struct{}

func (redisLogger) Printf(ctx context.Context, format string, v ...any) {
	slog.DebugContext(ctx, fmt.Sprintf(format, v...))
}

// Store détient les pools partagés vers Postgres et Valkey/Redis.
type Store struct {
	DB    *pgxpool.Pool
	Redis *redis.Client
}

// New construit les pools sans ouvrir de connexion (pgxpool et go-redis sont
// paresseux) : le serveur démarre même si les dépendances sont down.
func New(ctx context.Context, cfg config.Config) (*Store, error) {
	redis.SetLogger(redisLogger{})

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("pgxpool: %w", err)
	}

	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("redis url: %w", err)
	}

	return &Store{DB: pool, Redis: redis.NewClient(opt)}, nil
}

// PingPostgres vérifie la disponibilité de Postgres.
func (s *Store) PingPostgres(ctx context.Context) error {
	if err := s.DB.Ping(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	return nil
}

// PingRedis vérifie la disponibilité de Valkey/Redis.
func (s *Store) PingRedis(ctx context.Context) error {
	if err := s.Redis.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}
	return nil
}

// Close libère les pools.
func (s *Store) Close() {
	if s.DB != nil {
		s.DB.Close()
	}
	if s.Redis != nil {
		_ = s.Redis.Close()
	}
}
