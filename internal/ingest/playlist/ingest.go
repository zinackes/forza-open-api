package playlist

// Orchestration d'une passe d'ingestion « courante » : enchaîne fetch (forza.net
// + forum) → réconciliation → normalisation → upsert. Partagée par la commande
// `cmd/seed playlist` (run ponctuel) et le scheduler hebdomadaire (cmd/scheduler).
// Idempotente et rejouable : l'upsert porte sur l'id stable de la série et bascule
// is_current côté store (index partiel series_one_current_per_game).

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/zinackes/forza-open-api/internal/store"
)

// Result résume une passe IngestCurrent (pour le log de l'appelant).
type Result struct {
	Series      store.Series
	Added       bool // série nouvelle (vs mise à jour)
	Rewards     int
	Challenges  int
	Divergences int // écarts forza.net ↔ forum reportés
}

// IngestCurrent exécute une passe complète d'ingestion de la série visée pour un
// jeu. override vide = série courante auto-découverte depuis forza.net/events ;
// « SxxWx » cible une semaine précise (re-seed d'historique). Le réseau est borné
// par le ctx fourni par l'appelant. Idempotent et rejouable sans doublon.
func IngestCurrent(ctx context.Context, st *store.Store, game, override string, now time.Time, logger *slog.Logger) (Result, error) {
	ev, fd, err := FetchCurrent(ctx, game, override, now)
	if err != nil {
		return Result{}, fmt.Errorf("fetch playlist (%s): %w", game, err)
	}

	// Validation croisée forza.net (primaire) ↔ titre forum (secondaire) : logue
	// toute divergence (filet si forza.net change de structure) et complète les
	// champs d'identité que forza.net ne fournirait plus.
	ev, div := Reconcile(ev, fd)
	if logger != nil {
		div.Log(logger)
	}

	// Auto-discovery → c'est la série courante (rollover géré par l'upsert).
	// Override d'une semaine précise → courante seulement si la fenêtre couvre now
	// (un re-seed d'historique ne vole pas le flag « courante »).
	isCurrent := override == "" ||
		(!ev.Start.IsZero() && !ev.End.IsZero() && !ev.Start.After(now) && !ev.End.Before(now))

	ser, rewards, challenges := Normalize(ev, fd, isCurrent)

	added, err := st.UpsertSeries(ctx, ser, rewards, challenges)
	if err != nil {
		return Result{}, fmt.Errorf("upsert series %s: %w", ser.ID, err)
	}

	return Result{
		Series:      ser,
		Added:       added,
		Rewards:     len(rewards),
		Challenges:  len(challenges),
		Divergences: len(div.Divergences),
	}, nil
}
