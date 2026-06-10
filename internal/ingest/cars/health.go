package cars

import (
	"fmt"

	"github.com/zinackes/forza-open-api/internal/health"
)

// Invariants d'une passe catalogue saine. Heuristiques de RUPTURE DE STRUCTURE
// (template {{CarListStats…}} du wiki modifié, page liste vide) : un catalogue
// Forza Horizon compte des centaines de voitures et la quasi-totalité des lignes
// se normalise. Un effondrement du count ou un fort taux d'écarts trahit un
// changement de structure → anomaly. Les bornes sont des planchers/plafonds
// volontairement larges (éviter les faux positifs), pas des comptes exacts.

// Fourchettes attendues par jeu (voitures distinctes upsertées). Hors map → bornes
// par défaut. À ajuster si un jeu sort un catalogue notablement plus grand/petit.
var (
	minCarsByGame = map[string]int{"fh6": 150, "fh5": 400}
	maxCarsByGame = map[string]int{"fh6": 1500, "fh5": 1500}
)

const (
	defaultMinCars = 100
	defaultMaxCars = 2000
	// Au-delà de ce ratio de lignes écartées, le parsing a probablement cassé.
	maxSkipRatio = 0.5
	// Nombre de codes (obtention/layout) non mappés au-delà duquel on s'inquiète.
	maxUnknownCodes = 10
)

// CheckRun contrôle le rapport d'ingestion (rep) et le rapport de validation
// (vrep) d'une passe catalogue et renvoie les invariants violés. Aucune violation
// = run sain.
func CheckRun(rep Report, vrep ValidationReport, game string) []health.Violation {
	var vs []health.Violation
	add := func(sev health.Severity, rule, detail string) {
		vs = append(vs, health.Violation{Rule: rule, Detail: detail, Severity: sev})
	}

	if rep.Rows == 0 {
		// Page liste totalement vide : le sélecteur de template ne matche plus rien.
		add(health.SevAnomaly, "zero_rows", "0 ligne parsée (page liste vide ou template renommé)")
		return vs // les autres checks n'ont plus de sens sur un parse vide
	}

	min := defaultMinCars
	if m, ok := minCarsByGame[game]; ok {
		min = m
	}
	max := defaultMaxCars
	if m, ok := maxCarsByGame[game]; ok {
		max = m
	}
	if rep.Distinct < min {
		add(health.SevAnomaly, "count_floor",
			fmt.Sprintf("%d voitures < plancher %d (%s)", rep.Distinct, min, game))
	}
	if rep.Distinct > max {
		add(health.SevWarning, "count_ceiling",
			fmt.Sprintf("%d voitures > plafond %d (%s)", rep.Distinct, max, game))
	}

	skipped := rep.Rows - rep.Imported
	if ratio := float64(skipped) / float64(rep.Rows); ratio > maxSkipRatio {
		add(health.SevAnomaly, "skip_ratio",
			fmt.Sprintf("%d/%d lignes écartées (%.0f%% > %.0f%%)", skipped, rep.Rows, ratio*100, maxSkipRatio*100))
	}

	// Anomalies bloquantes (CHECK/NOT NULL violés) déjà détectées par Validate :
	// 0 attendu sur une sortie saine de Normalize.
	if n := vrep.Count(SevBlocking); n > 0 {
		add(health.SevAnomaly, "blocking", fmt.Sprintf("%d entrée(s) violant une contrainte DB", n))
	}
	if n := len(rep.UnknownCodes); n > maxUnknownCodes {
		add(health.SevWarning, "unknown_codes", fmt.Sprintf("%d code(s) non mappé(s) (> %d)", n, maxUnknownCodes))
	}
	return vs
}
