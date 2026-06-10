package cars

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/zinackes/forza-open-api/internal/store"
)

// Étape qualité du seed : contrôle les invariants du contrat sur le catalogue
// DÉJÀ normalisé, sans bloquer l'ingestion. Deux sévérités :
//   - blocking : viole une contrainte DB (CHECK class/pi/drivetrain) ou un champ
//     requis (NOT NULL game/name/make). La ligne serait rejetée par Postgres ou
//     trahit une donnée corrompue → à écarter de l'upsert (cf. cmd/seed).
//   - warning : entrée douteuse mais insérable (class incohérente avec la bande
//     PI, make sans lettre). Seedée telle quelle, mais signalée pour investigation.
//
// Objectif opérationnel : 0 anomalie blocking sur la sortie de Normalize (cf.
// validate_test.go). Normalize garantit déjà ces invariants ; Validate les
// reverifie de façon indépendante pour rattraper toute régression future ou une
// source d'ingestion alternative.

// Severity classe une anomalie. Valeur lisible (logs/tests).
type Severity string

const (
	SevWarning  Severity = "warning"
	SevBlocking Severity = "blocking"
)

// Anomaly décrit une entrée du catalogue en défaut. CarID identifie la voiture,
// Rule le contrôle violé (stable, pour agrégation), Detail le contexte humain.
type Anomaly struct {
	CarID    string
	Severity Severity
	Rule     string
	Detail   string
}

// ValidationReport agrège les anomalies d'une passe de validation.
type ValidationReport struct {
	Checked   int
	Anomalies []Anomaly
}

// Count renvoie le nombre d'anomalies de la sévérité donnée.
func (r ValidationReport) Count(sev Severity) int {
	n := 0
	for _, a := range r.Anomalies {
		if a.Severity == sev {
			n++
		}
	}
	return n
}

// BlockingIDs renvoie l'ensemble des ids de voitures portant ≥1 anomalie
// blocking, pour les écarter de l'upsert.
func (r ValidationReport) BlockingIDs() map[string]struct{} {
	ids := make(map[string]struct{})
	for _, a := range r.Anomalies {
		if a.Severity == SevBlocking {
			ids[a.CarID] = struct{}{}
		}
	}
	return ids
}

// RuleCounts agrège les anomalies par "severity:rule" (log compact déterministe
// via SortedReport).
func (r ValidationReport) RuleCounts() map[string]int {
	m := make(map[string]int)
	for _, a := range r.Anomalies {
		m[string(a.Severity)+":"+a.Rule]++
	}
	return m
}

// validClasses / validDrivetrains : enums du contrat (= CHECK DB cars.class /
// cars.drivetrain). Tenir aligné avec api/openapi.yaml et db/init.sql.
var validClasses = map[string]bool{
	"D": true, "C": true, "B": true, "A": true,
	"S1": true, "S2": true, "X": true, "R": true,
}

var validDrivetrains = map[string]bool{"FWD": true, "RWD": true, "AWD": true}

// Validate contrôle le catalogue normalisé et renvoie le rapport d'anomalies
// sans rien modifier ni bloquer. L'appelant décide quoi seeder (cf. cmd/seed :
// warnings seedés, blocking écartés).
func Validate(cars []store.Car, game string) ValidationReport {
	rep := ValidationReport{Checked: len(cars)}
	add := func(id string, sev Severity, rule, detail string) {
		rep.Anomalies = append(rep.Anomalies, Anomaly{CarID: id, Severity: sev, Rule: rule, Detail: detail})
	}

	for _, c := range cars {
		// --- Contraintes dures (rejetées par la DB) -------------------------
		if c.PI < 100 || c.PI > 999 {
			add(c.ID, SevBlocking, "pi_range", fmt.Sprintf("pi=%d hors [100,999]", c.PI))
		}
		if !validClasses[c.Class] {
			add(c.ID, SevBlocking, "class_enum", fmt.Sprintf("class=%q hors enum", c.Class))
		}
		if !validDrivetrains[c.Drivetrain] {
			add(c.ID, SevBlocking, "drivetrain_enum", fmt.Sprintf("drivetrain=%q hors enum", c.Drivetrain))
		}
		if c.Game == "" {
			add(c.ID, SevBlocking, "game_empty", "game vide")
		}
		if c.Name == "" {
			add(c.ID, SevBlocking, "name_empty", "name vide")
		}
		if c.Make == "" {
			add(c.ID, SevBlocking, "make_empty", "make vide")
		} else if !strings.ContainsFunc(c.Make, unicode.IsLetter) {
			// make non vide mais sans aucune lettre (ex. dérivation ratée sur un
			// nom purement numérique) : insérable mais suspect.
			add(c.ID, SevWarning, "make_no_letter", fmt.Sprintf("make=%q sans lettre", c.Make))
		}

		// --- Cohérence douteuse (insérable) ---------------------------------
		// La class doit refléter la bande PI publiée du jeu ; un écart signale
		// une donnée wiki incohérente ou une régression de normalisation.
		if want := classFromPI(game, c.PI); want != "" && want != c.Class && validClasses[c.Class] {
			add(c.ID, SevWarning, "class_pi_mismatch",
				fmt.Sprintf("class=%s mais bande PI %d → %s", c.Class, c.PI, want))
		}
	}
	return rep
}
