package export

import (
	"time"

	"github.com/zinackes/forza-open-api/internal/health"
)

// CheckRun contrôle une passe de génération. Invariant de rupture : un dataset
// entièrement vide (0 ligne sur TOUTES les ressources) trahit un jeu non seedé, un
// mauvais code jeu ou une base injoignable — on alerte plutôt que de publier des
// archives vides en silence. Un trou sur une seule ressource reflète l'état réel
// de la base (pas un défaut d'export) → non signalé ici. La signature porte now
// pour la parité avec les autres CheckRun (internal/ingest/*).
func CheckRun(res Result, _ time.Time) []health.Violation {
	total := 0
	for _, n := range res.Rows {
		total += n
	}
	if total == 0 {
		return []health.Violation{{
			Rule:     "empty_dataset",
			Detail:   "0 ligne sur toutes les ressources (jeu non seedé, mauvais code jeu ou base vide ?)",
			Severity: health.SevAnomaly,
		}}
	}
	return nil
}
