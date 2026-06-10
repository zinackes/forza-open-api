// Rate-limit par clé en fenêtre glissante (sliding-window log) dans Redis/Valkey.
//
// Pourquoi un sorted set plutôt qu'un simple INCR+EXPIRE (fenêtre fixe) : la
// fenêtre fixe autorise jusqu'à 2× la limite à cheval sur deux fenêtres (burst
// de bord). Le sliding-window log horodate chaque requête dans un ZSET, purge
// celles sorties de la fenêtre, puis compte : la limite est respectée sur toute
// fenêtre glissante. Le tout dans UN script Lua → atomique (pas de race entre
// le comptage et l'ajout sous forte concurrence sur une même clé).
package store

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// rateLimitScript implémente la fenêtre glissante de façon atomique.
//
//	KEYS[1] = clé de quota (ex. "ratelimit:v1:<hash>")
//	ARGV[1] = now en ms ; ARGV[2] = fenêtre en ms ; ARGV[3] = limite
//	ARGV[4] = membre unique pour CETTE requête (horodatage + aléa)
//
// Retourne {allowed (0/1), used (compte dans la fenêtre), resetMs (ms avant
// qu'un créneau se libère)}.
var rateLimitScript = redis.NewScript(`
local key    = KEYS[1]
local now    = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit  = tonumber(ARGV[3])
local member = ARGV[4]

-- Purge les entrées sorties de la fenêtre glissante.
redis.call('ZREMRANGEBYSCORE', key, 0, now - window)

local used    = redis.call('ZCARD', key)
local allowed = 0
if used < limit then
  redis.call('ZADD', key, now, member)
  used    = used + 1
  allowed = 1
end

-- TTL = fenêtre : une clé inactive disparaît d'elle-même (pas de fuite mémoire).
redis.call('PEXPIRE', key, window)

-- reset = délai avant que la plus ancienne entrée quitte la fenêtre (= un
-- créneau se libère). Fenêtre pleine vidée → la fenêtre entière.
local reset  = window
local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
if oldest[2] then
  reset = (tonumber(oldest[2]) + window) - now
  if reset < 0 then reset = 0 end
end

return {allowed, used, reset}
`)

// RateLimitResult porte la décision et l'état du quota, de quoi remplir les
// en-têtes X-RateLimit-* / RateLimit côté handler.
type RateLimitResult struct {
	Allowed    bool          // la requête courante est-elle dans le quota
	Limit      int           // quota maximal sur la fenêtre
	Remaining  int           // requêtes restantes (jamais négatif)
	ResetAfter time.Duration // délai avant qu'un créneau se libère
}

// AllowRequest applique la fenêtre glissante à key et renvoie la décision. La
// limite et la fenêtre sont fournies par l'appelant (issues de la clé API et de
// la config). Erreur si Redis n'est pas configuré ou indisponible : l'appelant
// décide alors de la politique (le middleware « fail-open » pour ne pas bloquer
// un client légitime sur une panne Redis).
func (s *Store) AllowRequest(ctx context.Context, key string, limit int, window time.Duration) (RateLimitResult, error) {
	if s.Redis == nil {
		return RateLimitResult{}, fmt.Errorf("rate limit: redis not configured")
	}

	now := time.Now()
	// Membre unique : deux requêtes dans la même ms (y compris depuis des
	// instances différentes) doivent compter pour deux entrées distinctes,
	// sinon ZADD écraserait et sous-compterait. Horodatage ns + aléa 64 bits.
	member := strconv.FormatInt(now.UnixNano(), 10) + "-" + strconv.FormatUint(rand.Uint64(), 10)

	raw, err := rateLimitScript.Run(ctx, s.Redis, []string{key},
		now.UnixMilli(), window.Milliseconds(), limit, member).Slice()
	if err != nil {
		return RateLimitResult{}, fmt.Errorf("rate limit script: %w", err)
	}
	if len(raw) != 3 {
		return RateLimitResult{}, fmt.Errorf("rate limit script: unexpected result %v", raw)
	}

	allowed, _ := raw[0].(int64)
	used, _ := raw[1].(int64)
	resetMs, _ := raw[2].(int64)

	remaining := limit - int(used)
	if remaining < 0 {
		remaining = 0
	}
	return RateLimitResult{
		Allowed:    allowed == 1,
		Limit:      limit,
		Remaining:  remaining,
		ResetAfter: time.Duration(resetMs) * time.Millisecond,
	}, nil
}
