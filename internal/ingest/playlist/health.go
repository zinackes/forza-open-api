package playlist

import (
	"fmt"
	"time"

	"github.com/zinackes/forza-open-api/internal/health"
)

// Invariants d'une passe playlist saine. Heuristiques pensées pour attraper une
// RUPTURE DE STRUCTURE (payload Nuxt forza.net ou template forum modifié), pas
// pour valider la donnée au champ près : une série courante a toujours un nom,
// des dates, des récompenses et des défis. Leur absence sur un parse « réussi »
// trahit un changement de structure → anomaly.

// CheckRun contrôle le résultat d'IngestCurrent et renvoie les invariants violés.
// now sert à détecter une série « courante » dont la fenêtre est déjà périmée
// (rollover hebdomadaire raté). Aucune violation = run sain.
func CheckRun(res Result, now time.Time) []health.Violation {
	var vs []health.Violation
	add := func(sev health.Severity, rule, detail string) {
		vs = append(vs, health.Violation{Rule: rule, Detail: detail, Severity: sev})
	}
	ser := res.Series

	if ser.ID == "" {
		add(health.SevAnomaly, "series_id_empty", "id de série vide (slug non parsé)")
	}
	if ser.Name == nil || *ser.Name == "" {
		add(health.SevAnomaly, "name_empty", "nom de série vide (heading forum non parsé)")
	}
	if ser.StartsAt.IsZero() || ser.EndsAt.IsZero() {
		add(health.SevAnomaly, "dates_zero",
			fmt.Sprintf("dates manquantes (start_zero=%t end_zero=%t)", ser.StartsAt.IsZero(), ser.EndsAt.IsZero()))
	} else if ser.EndsAt.Before(ser.StartsAt) {
		add(health.SevAnomaly, "dates_inverted",
			fmt.Sprintf("fin %s avant début %s", ser.EndsAt.Format(time.RFC3339), ser.StartsAt.Format(time.RFC3339)))
	}
	if ser.Week < 1 {
		add(health.SevAnomaly, "week_range", fmt.Sprintf("week=%d (< 1)", ser.Week))
	}
	if res.Rewards == 0 {
		add(health.SevAnomaly, "no_rewards", "0 récompense (détail forum non parsé)")
	}
	if res.Challenges == 0 {
		add(health.SevAnomaly, "no_challenges", "0 défi (détail forum non parsé)")
	}
	// Série marquée courante mais déjà terminée : le rafraîchissement hebdo n'a pas
	// basculé sur la nouvelle semaine → on sert une playlist périmée.
	if ser.IsCurrent && !ser.EndsAt.IsZero() && ser.EndsAt.Before(now) {
		add(health.SevAnomaly, "stale_current",
			fmt.Sprintf("série courante terminée le %s (rollover manqué ?)", ser.EndsAt.Format(time.RFC3339)))
	}
	// Divergences forza.net ↔ forum : filet de cross-check, pas forcément une
	// rupture → avertissement (déjà loggé en détail par Reconcile).
	if res.Divergences > 0 {
		add(health.SevWarning, "divergences", fmt.Sprintf("%d divergence(s) forza.net/forum", res.Divergences))
	}
	return vs
}
