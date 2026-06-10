package tracks

import (
	"fmt"

	"github.com/zinackes/forza-open-api/internal/health"
)

// CheckRun contrôle une passe d'ingestion de tracés. Source = dataset communautaire
// fourni par l'opérateur (pas de structure forza.net volatile) : le seul invariant
// de rupture utile est un dataset vide (0 tracé upserté) → anomaly. skipped sert
// au contexte (lignes ignorées). Aucune violation = run sain.
func CheckRun(upserted, skipped int) []health.Violation {
	if upserted == 0 {
		return []health.Violation{{
			Rule:     "zero_records",
			Detail:   fmt.Sprintf("0 tracé upserté (%d ignoré(s)) : dataset vide ou format inattendu", skipped),
			Severity: health.SevAnomaly,
		}}
	}
	return nil
}
