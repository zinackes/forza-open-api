package manufacturers

import (
	"fmt"

	"github.com/zinackes/forza-open-api/internal/health"
)

// Invariants d'une passe « constructeurs » saine. Heuristiques pensées pour
// attraper une RUPTURE DE STRUCTURE de la source (catégorie renommée/vidée, infobox
// au format changé), pas pour valider chaque champ.

// CheckRun contrôle le résultat d'IngestManufacturers et renvoie les invariants
// violés. Aucune violation = run sain.
func CheckRun(res Result) []health.Violation {
	var vs []health.Violation
	add := func(sev health.Severity, rule, detail string) {
		vs = append(vs, health.Violation{Rule: rule, Detail: detail, Severity: sev})
	}

	// Catégorie absente : jeu non (encore) documenté → avertissement (pré-lancement
	// FH6, comme Stories/Tours : on n'invente rien).
	if !res.CategoryFound {
		add(health.SevWarning, "no_category",
			"sous-catégorie Manufacturers du jeu absente (jeu non documenté ?)")
		return vs
	}

	// Catégorie présente mais aucune page récupérée / normalisée : la structure de la
	// source a changé (catégorie vidée, titres illisibles) → anomalie.
	if res.Members > 0 && res.PagesFetched == 0 {
		add(health.SevAnomaly, "no_pages",
			"catégorie listée mais aucune page constructeur récupérée (structure changée ?)")
	}
	if res.PagesFetched > 0 && res.Manufacturers == 0 {
		add(health.SevAnomaly, "no_manufacturers",
			"pages récupérées mais aucun constructeur normalisé (titres vides ?)")
	}

	// Pages présentes mais aucune country résolue : l'{{InfoboxMFR}} ou son champ
	// origin a changé de format → anomalie (le filtre ?country resterait vide).
	if res.Manufacturers > 0 && res.CountryResolved == 0 {
		add(health.SevAnomaly, "no_country_resolved",
			"aucune country résolue depuis origin (InfoboxMFR/origin modifié ?)")
	}

	// Beaucoup de codes origin non mappés : la source a introduit de nouveaux codes
	// (table de normalisation à compléter) → avertissement.
	if unknown := totalCounts(res.UnknownOrigins); res.Manufacturers > 0 && unknown > res.Manufacturers/10 {
		add(health.SevWarning, "many_unknown_origins",
			fmt.Sprintf("codes origin non mappés nombreux (%d) : %s", unknown, SortedReport(res.UnknownOrigins)))
	}

	// Aucun constructeur rapproché d'un make du catalogue : noms divergents (le
	// car_count resterait nul partout) → avertissement (catalogue absent/format changé).
	if res.Manufacturers > 0 && res.MatchedToCars == 0 {
		add(health.SevWarning, "no_car_match",
			"aucun constructeur rapproché d'un make du catalogue (noms divergents ou catalogue vide)")
	}

	return vs
}

func totalCounts(m map[string]int) int {
	n := 0
	for _, v := range m {
		n += v
	}
	return n
}
