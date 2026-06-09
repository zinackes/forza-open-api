package cars

import "github.com/zinackes/forza-open-api/internal/store"

// --- Réconciliation catalogue ↔ télémétrie (CarOrdinal) -----------------------
//
// La télémétrie « Data Out » de Forza (UDP, lecture seule, hors-process →
// EAC-safe) expose un CarOrdinal : l'identifiant interne du jeu pour la voiture
// pilotée. Notre catalogue est, lui, clé par slug wiki (ex.
// "fh6-abarth-695-biposto"). Pour relier une session télémétrique à une voiture
// du catalogue il faut un mapping CarOrdinal → id catalogue.
//
// HOOK FUTUR : la structure de matching est posée ci-dessous ; ce qui manque est
// (1) la SOURCE PROPRE du mapping et (2) l'exposition de l'ordinal au contrat.

// CarOrdinalLink associe l'identifiant interne du jeu (CarOrdinal, télémétrie) à
// l'id catalogue. Game désambiguïse : un ordinal n'est pas garanti stable d'un
// jeu Forza à l'autre.
type CarOrdinalLink struct {
	Game       string
	CarOrdinal int
	CarID      string
}

// ReconcileReport résume une passe de réconciliation.
type ReconcileReport struct {
	Linked            int      // voitures catalogue reliées à un CarOrdinal
	UnlinkedCars      []string // voitures catalogue sans CarOrdinal connu
	UnmatchedOrdinals []int    // CarOrdinal sans voiture catalogue correspondante
}

// Reconcile apparie les CarOrdinal aux voitures du catalogue (clé : Game+CarID)
// et reporte les écarts des deux côtés. La structure de matching est complète ;
// le hook reste à finir.
//
// TODO(reconcile):
//  1. Sourcer `links` PROPREMENT (doctrine zéro gris) : dataset communautaire
//     CarOrdinal→nom, ou crowdsourcing via Data Out (le joueur conduit, on capte
//     CarOrdinal + nom affiché). JAMAIS de lecture mémoire / scraping du jeu.
//  2. Exposer carOrdinal au contrat (api/openapi.yaml → task generate → colonne
//     cars.car_ordinal) puis écrire l'ordinal sur la voiture appariée (cf. point
//     marqué ci-dessous).
//  3. Persister le mapping (table car_ordinals, PK game+car_ordinal) pour relier
//     les sessions télémétriques à l'ingestion.
func Reconcile(cars []store.Car, links []CarOrdinalLink) ReconcileReport {
	key := func(game, id string) string { return game + "\x00" + id }

	byCar := make(map[string]CarOrdinalLink, len(links))
	for _, l := range links {
		byCar[key(l.Game, l.CarID)] = l
	}

	var rep ReconcileReport
	matched := make(map[string]bool, len(links))
	for _, c := range cars {
		k := key(c.Game, c.ID)
		if _, ok := byCar[k]; ok {
			rep.Linked++
			matched[k] = true
			// TODO(reconcile): écrire l'ordinal une fois le contrat étendu —
			//   c.CarOrdinal = byCar[k].CarOrdinal
		} else {
			rep.UnlinkedCars = append(rep.UnlinkedCars, c.ID)
		}
	}
	for _, l := range links {
		if !matched[key(l.Game, l.CarID)] {
			rep.UnmatchedOrdinals = append(rep.UnmatchedOrdinals, l.CarOrdinal)
		}
	}
	return rep
}
