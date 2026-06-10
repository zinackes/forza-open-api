// Test de la fenêtre glissante AllowRequest sur un Redis jetable (testcontainers,
// image redis via l'API générique — aucun module testcontainers supplémentaire).
//
// Nécessite Docker (cf. .claude/rules/testing.md). En CI, Docker est présent.
package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/zinackes/forza-open-api/internal/store"
)

// newRedisStore lève un Redis jetable et renvoie un Store branché dessus
// (Postgres nil : AllowRequest ne touche qu'à Redis).
func newRedisStore(t *testing.T) *store.Store {
	t.Helper()
	ctx := context.Background()

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:7-alpine",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForListeningPort("6379/tcp").WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("démarrage Redis (Docker requis): %v", err)
	}
	t.Cleanup(func() { _ = c.Terminate(context.Background()) })

	host, err := c.Host(ctx)
	if err != nil {
		t.Fatalf("host: %v", err)
	}
	port, err := c.MappedPort(ctx, "6379/tcp")
	if err != nil {
		t.Fatalf("port: %v", err)
	}

	client := redis.NewClient(&redis.Options{Addr: host + ":" + port.Port()})
	t.Cleanup(func() { _ = client.Close() })
	return &store.Store{Redis: client}
}

// TestAllowRequestSlidingWindow : limit requêtes passent (Remaining décroît),
// puis la suivante est refusée (Remaining 0, ResetAfter > 0). Boucle de
// requêtes = le « test du dépassement » de la carte 2.3, au niveau Redis.
func TestAllowRequestSlidingWindow(t *testing.T) {
	st := newRedisStore(t)
	ctx := context.Background()

	const limit = 5
	const window = time.Minute
	key := "ratelimit:test:loop"

	for i := 1; i <= limit; i++ {
		res, err := st.AllowRequest(ctx, key, limit, window)
		if err != nil {
			t.Fatalf("req %d: %v", i, err)
		}
		if !res.Allowed {
			t.Fatalf("req %d refusée, attendu autorisée", i)
		}
		if res.Remaining != limit-i {
			t.Errorf("req %d: Remaining = %d, attendu %d", i, res.Remaining, limit-i)
		}
		if res.Limit != limit {
			t.Errorf("req %d: Limit = %d, attendu %d", i, res.Limit, limit)
		}
	}

	res, err := st.AllowRequest(ctx, key, limit, window)
	if err != nil {
		t.Fatalf("dépassement: %v", err)
	}
	if res.Allowed {
		t.Fatal("requête au-delà de la limite autorisée, attendu refusée")
	}
	if res.Remaining != 0 {
		t.Errorf("dépassement: Remaining = %d, attendu 0", res.Remaining)
	}
	if res.ResetAfter <= 0 {
		t.Errorf("dépassement: ResetAfter = %v, attendu > 0", res.ResetAfter)
	}
}

// TestAllowRequestIsolatesKeys : deux clés ont des compteurs indépendants (le
// dépassement de l'une n'affecte pas l'autre).
func TestAllowRequestIsolatesKeys(t *testing.T) {
	st := newRedisStore(t)
	ctx := context.Background()

	const limit = 2
	const window = time.Minute

	// Épuise k1.
	for i := 0; i < limit; i++ {
		if _, err := st.AllowRequest(ctx, "ratelimit:test:k1", limit, window); err != nil {
			t.Fatalf("k1 req %d: %v", i, err)
		}
	}
	if res, _ := st.AllowRequest(ctx, "ratelimit:test:k1", limit, window); res.Allowed {
		t.Fatal("k1 au-delà de la limite devrait être refusée")
	}

	// k2 intacte.
	res, err := st.AllowRequest(ctx, "ratelimit:test:k2", limit, window)
	if err != nil {
		t.Fatalf("k2: %v", err)
	}
	if !res.Allowed {
		t.Fatal("k2 (compteur indépendant) devrait être autorisée")
	}
	if res.Remaining != limit-1 {
		t.Errorf("k2: Remaining = %d, attendu %d", res.Remaining, limit-1)
	}
}
