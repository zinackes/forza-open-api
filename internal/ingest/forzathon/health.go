package forzathon

import (
	"fmt"
	"time"

	"github.com/zinackes/forza-open-api/internal/health"
)

// Invariants d'une passe Forzathon Shop saine. Heuristiques pensées pour attraper
// une RUPTURE DE STRUCTURE de la source (page wiki renommée/vidée, table au format
// changé), pas pour valider chaque champ : une rotation a toujours des objets, une
// fenêtre cohérente, et des coûts FP. Leur absence sur un parse « réussi » trahit
// un changement de structure → anomaly (alerte).

// CheckRun contrôle le résultat d'IngestCurrent et renvoie les invariants violés.
// now sert à détecter une fenêtre déjà périmée. Aucune violation = run sain.
func CheckRun(res Result, now time.Time) []health.Violation {
	var vs []health.Violation
	add := func(sev health.Severity, rule, detail string) {
		vs = append(vs, health.Violation{Rule: rule, Detail: detail, Severity: sev})
	}

	if res.WeekStart.IsZero() || res.WeekEnd.IsZero() {
		add(health.SevAnomaly, "window_zero",
			fmt.Sprintf("fenêtre manquante (start_zero=%t end_zero=%t)", res.WeekStart.IsZero(), res.WeekEnd.IsZero()))
	} else if !res.WeekEnd.After(res.WeekStart) {
		add(health.SevAnomaly, "window_inverted",
			fmt.Sprintf("fin %s <= début %s", res.WeekEnd.Format(time.RFC3339), res.WeekStart.Format(time.RFC3339)))
	} else if res.WeekEnd.Before(now) {
		add(health.SevAnomaly, "stale_rotation",
			fmt.Sprintf("rotation terminée le %s (rollover manqué ?)", res.WeekEnd.Format(time.RFC3339)))
	}
	if res.Items == 0 {
		add(health.SevAnomaly, "no_items", "0 objet (page wiki absente ou table non parsée)")
	}
	// Tous les coûts à NULL alors qu'il y a des objets : colonne coût probablement
	// déplacée/renommée → avertissement (le reste de la donnée reste exploitable).
	if res.Items > 0 && res.ItemsWithCost == 0 {
		add(health.SevWarning, "no_fp_cost", "aucun coût FP parsé (colonne coût modifiée ?)")
	}
	return vs
}
