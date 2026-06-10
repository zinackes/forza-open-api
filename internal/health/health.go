// Package health surveille la santé des passes d'ingestion (scrapers). Au-delà de
// l'échec dur (fetch/upsert en erreur), il détecte les ANOMALIES SILENCIEUSES :
// un parse qui « réussit » mais produit des données vides ou tronquées — la
// signature d'un changement de structure côté source (payload Nuxt de forza.net,
// template wiki). Chaque source produit un Report (violations d'invariants) qu'un
// Monitor logue, persiste et alerte (webhook) en cas d'anomalie ou d'échec.
//
// Découplage : ce package porte le vocabulaire (Violation/Severity/Report) et
// l'orchestration (Monitor/Notifier) ; la logique métier « qu'est-ce qu'un run
// sain » vit dans chaque package d'ingestion (cars/playlist/tracks) qui n'importe
// d'ici que les types. Aucune dépendance vers le cron : utilisable par cmd/seed
// (run ponctuel) comme par cmd/scheduler (tick).
package health

// Severity classe une violation d'invariant.
//   - SevWarning : suspect mais tolérable (n'invalide pas le run).
//   - SevAnomaly : signe une rupture probable de structure → alerte.
type Severity string

const (
	SevWarning Severity = "warning"
	SevAnomaly Severity = "anomaly"
)

// Violation décrit un invariant non respecté. Rule est stable (agrégation/tests),
// Detail porte le contexte humain.
type Violation struct {
	Rule     string   `json:"rule"`
	Detail   string   `json:"detail"`
	Severity Severity `json:"severity"`
}

// Statut d'un run, persisté et porté par l'alerte.
const (
	StatusOK      = "ok"
	StatusAnomaly = "anomaly"
	StatusFailed  = "failed"
)

// Report est le bilan d'une passe d'ingestion d'une source pour un jeu. Err porte
// un échec dur (fetch/upsert) ; Violations les anomalies/avertissements détectés
// sur des données pourtant ingérées. Records = volume ingéré (séries, voitures…).
type Report struct {
	Source     string
	Game       string
	Records    int
	Violations []Violation
	Err        error
}

// HasAnomaly indique qu'au moins une violation est de sévérité anomalie.
func (r Report) HasAnomaly() bool {
	for _, v := range r.Violations {
		if v.Severity == SevAnomaly {
			return true
		}
	}
	return false
}

// Status résume le run : failed (échec dur) prime, puis anomaly (≥1 anomalie), sinon ok.
func (r Report) Status() string {
	switch {
	case r.Err != nil:
		return StatusFailed
	case r.HasAnomaly():
		return StatusAnomaly
	default:
		return StatusOK
	}
}
