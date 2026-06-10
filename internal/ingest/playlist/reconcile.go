package playlist

import (
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Validation croisée identité forza.net ↔ forum (doctrine docs/DATA-SOURCES.md :
// « Forums officiels Forza : source secondaire (récompenses hebdo), validation
// croisée »). forza.net (carte 3.2) reste la source PRIMAIRE de l'identité de la
// série ; le titre du topic Discourse porte la MÊME identité (jeu, série, semaine,
// fenêtre de dates) et sert ici de filet à deux usages :
//   - VALIDER : tout écart entre les deux sources est reporté — alerte précoce si
//     forza.net change de structure et dérive silencieusement ;
//   - COMPLÉTER : si forza.net ne fournit plus un champ d'identité (slug non parsé
//     → game/série/semaine à zéro), on retombe sur la valeur du forum.
//
// Les dates ne sont JAMAIS reconstruites depuis le forum : son titre ne porte ni
// année ni heure (« May 28 - June 4 »). On VALIDE le couple (mois, jour) contre
// forza.net mais on n'invente pas un time.Time partiel (doctrine : champ absent →
// zéro/NULL). forza.net présent fait toujours foi ; le forum ne l'écrase jamais.

// forumIdentity est l'identité de la série lue dans le TITRE du topic Discourse,
// p.ex. « FH6 Festival Playlist events and rewards May 28 - June 4 (Series 01
// Week 2) ». Les bornes sont en (mois, jour) seulement — pas d'année dans le titre.
type forumIdentity struct {
	OK         bool // titre exploitable (jeu + série + semaine reconnus)
	Game       string
	Series     int
	Week       int
	StartMonth time.Month
	StartDay   int
	EndMonth   time.Month
	EndDay     int
}

var (
	// reForumSeries capte le jeu, le numéro de série et la semaine du titre.
	reForumSeries = regexp.MustCompile(`(?i)^(FH\d)\b.*?\bSeries\s+(\d+)\s+Week\s+(\d+)`)
	// reForumDates capte la fenêtre « <Mois> <jour> - <Mois> <jour> » (année absente).
	reForumDates = regexp.MustCompile(`([A-Z][a-z]+)\s+(\d{1,2})\s*[-–—]\s*([A-Z][a-z]+)\s+(\d{1,2})`)
)

// parseForumTitle extrait l'identité (jeu, série, semaine, fenêtre mois/jour) du
// titre d'un topic forum. OK reste false si le titre ne suit pas le format attendu.
func parseForumTitle(title string) forumIdentity {
	m := reForumSeries.FindStringSubmatch(title)
	if m == nil {
		return forumIdentity{}
	}
	series, _ := strconv.Atoi(m[2])
	week, _ := strconv.Atoi(m[3])
	fi := forumIdentity{OK: true, Game: strings.ToLower(m[1]), Series: series, Week: week}

	// Bornes de dates : restreintes à l'avant de « (Series … » pour écarter toute
	// capture parasite.
	head := title
	if i := strings.IndexByte(head, '('); i >= 0 {
		head = head[:i]
	}
	if d := reForumDates.FindStringSubmatch(head); d != nil {
		sd, _ := strconv.Atoi(d[2])
		ed, _ := strconv.Atoi(d[4])
		fi.StartMonth, fi.StartDay = monthFromName(d[1]), sd
		fi.EndMonth, fi.EndDay = monthFromName(d[3]), ed
	}
	return fi
}

// monthFromName convertit un nom de mois anglais complet (« May ») en time.Month
// (0 si non reconnu).
func monthFromName(name string) time.Month {
	t, err := time.Parse("January", name)
	if err != nil {
		return 0
	}
	return t.Month()
}

// Divergence décrit un écart entre l'identité forza.net (primaire) et celle du
// forum (secondaire). Kind classe l'écart, Field le champ, Forza/Forum les valeurs
// lisibles (Forza vide = forza.net ne fournit pas le champ).
type Divergence struct {
	Field string
	Kind  string // mismatch | missing_primary | unparsed
	Forza string
	Forum string
}

// DivergenceReport agrège les écarts d'une passe de réconciliation.
type DivergenceReport struct {
	Divergences []Divergence
}

// Log émet une ligne slog.Warn par divergence (filet d'alerte sur changement de
// structure forza.net) ; silencieux si les sources concordent.
func (r DivergenceReport) Log(logger *slog.Logger) {
	for _, d := range r.Divergences {
		logger.Warn("playlist source divergence",
			"field", d.Field, "kind", d.Kind, "forza", d.Forza, "forum", d.Forum)
	}
}

// Reconcile croise l'identité forza.net (primaire) avec celle du titre forum
// (secondaire). Elle ne modifie ev que pour COMPLÉTER un champ d'identité absent
// côté forza.net (fallback game/série/semaine), jamais pour écraser une valeur
// présente. Tout écart est reporté. Pure (aucune I/O) → testée sur structs figées.
func Reconcile(ev eventMeta, fd forumData) (eventMeta, DivergenceReport) {
	fi := fd.Identity
	var rep DivergenceReport
	add := func(field, kind, forza, forum string) {
		rep.Divergences = append(rep.Divergences,
			Divergence{Field: field, Kind: kind, Forza: forza, Forum: forum})
	}

	if !fi.OK {
		// Titre forum illisible : plus de filet ni de cross-check pour cette passe.
		add("title", "unparsed", "", fd.Title)
		return ev, rep
	}

	switch {
	case ev.Game == "":
		add("game", "missing_primary", "", fi.Game)
		ev.Game = fi.Game
	case ev.Game != fi.Game:
		add("game", "mismatch", ev.Game, fi.Game)
	}

	switch {
	case ev.Series == 0:
		add("series", "missing_primary", "", strconv.Itoa(fi.Series))
		ev.Series = fi.Series
	case ev.Series != fi.Series:
		add("series", "mismatch", strconv.Itoa(ev.Series), strconv.Itoa(fi.Series))
	}

	switch {
	case ev.Week == 0:
		add("week", "missing_primary", "", strconv.Itoa(fi.Week))
		ev.Week = fi.Week
	case ev.Week != fi.Week:
		add("week", "mismatch", strconv.Itoa(ev.Week), strconv.Itoa(fi.Week))
	}

	reconcileDate("start", ev.Start, fi.StartMonth, fi.StartDay, add)
	reconcileDate("end", ev.End, fi.EndMonth, fi.EndDay, add)

	return ev, rep
}

// reconcileDate VALIDE une borne forza.net contre le couple (mois, jour) du titre
// forum. Aucune reconstruction : si forza.net n'a pas la date, on signale seulement
// (le titre n'a pas d'année → on n'invente pas de time.Time).
func reconcileDate(field string, primary time.Time, fm time.Month, fday int, add func(field, kind, forza, forum string)) {
	if fm == 0 || fday == 0 {
		return // le forum n'a pas de date exploitable pour cette borne
	}
	forum := fmt.Sprintf("%s %d", fm, fday)
	switch {
	case primary.IsZero():
		add(field, "missing_primary", "", forum)
	case primary.Month() != fm || primary.Day() != fday:
		add(field, "mismatch", fmt.Sprintf("%s %d", primary.Month(), primary.Day()), forum)
	}
}
