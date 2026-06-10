package forzathon

// Parsing du wikitext de la page Forzathon Shop (source primaire) et dérivation
// de la fenêtre de rotation. Le parser tourne sur fixtures figées (pas de réseau
// en test, cf. testing.md). Aucune donnée inventée : un champ absent reste nil.

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Reset hebdomadaire Forza, documenté sur le wiki (« each reset at 14:30 (UTC)
// every Thursday ») — même ancre que l'ingestion playlist. La rotation Forzathon
// Shop bascule à ce reset ; on en dérive la fenêtre plutôt que de l'inventer.
const (
	resetHour = 14
	resetMin  = 30
)

// weekWindow renvoie [start, end) de la rotation courante : le dernier reset hebdo
// (jeudi 14:30 UTC) <= now, et end = start + 7 jours. Déterministe (testable, pas
// de dépendance à la précision d'une date scrappée).
func weekWindow(now time.Time) (time.Time, time.Time) {
	n := now.UTC()
	reset := time.Date(n.Year(), n.Month(), n.Day(), resetHour, resetMin, 0, 0, time.UTC)
	offset := (int(reset.Weekday()) - int(time.Thursday) + 7) % 7
	start := reset.AddDate(0, 0, -offset)
	if start.After(n) { // jeudi mais avant 14:30 → la rotation courante a démarré la semaine d'avant
		start = start.AddDate(0, 0, -7)
	}
	return start, start.AddDate(0, 0, 7)
}

// rawItem est une ligne de la table Forzathon Shop, déjà classée et nettoyée.
type rawItem struct {
	Name        string
	Kind        string // enum contrat : car|horn|clothing|forza_link_phrase|other
	FpCost      *int
	Description *string
}

var reWikiLink = regexp.MustCompile(`\[\[(?:[^\]|]*\|)?([^\]]+)\]\]`)

// stripWiki réduit une cellule wikitext en texte brut : déréférence les liens
// [[Cible|Libellé]] → Libellé, retire le gras/italique, trim.
func stripWiki(s string) string {
	s = reWikiLink.ReplaceAllString(s, "$1")
	s = strings.ReplaceAll(s, "'''", "")
	s = strings.ReplaceAll(s, "''", "")
	s = strings.TrimSpace(s)
	return s
}

// parseShopTable extrait les objets de la première table wikitext de la page.
// Structure attendue (en-tête + lignes séparées par « |- ») :
//
//	{| class="wikitable"
//	! Item !! Category !! Cost
//	|-
//	| [[Mazda Furai]] || Car || 750
//	|}
//
// Tolère les variantes d'intitulés (item/name, category/type, cost/fp/price) ;
// défaut positionnel (0=nom, 1=catégorie, 2=coût) si l'en-tête est absent.
func parseShopTable(wikitext string) []rawItem {
	var header []string
	var rows [][]string
	var cur []string
	flush := func() {
		if len(cur) > 0 {
			rows = append(rows, cur)
			cur = nil
		}
	}
	for _, ln := range strings.Split(wikitext, "\n") {
		t := strings.TrimSpace(ln)
		switch {
		case strings.HasPrefix(t, "{|"), strings.HasPrefix(t, "|+"):
			// début de table / légende : ignorés
		case strings.HasPrefix(t, "!"):
			for _, c := range strings.Split(strings.TrimPrefix(t, "!"), "!!") {
				header = append(header, strings.ToLower(stripWiki(c)))
			}
		case strings.HasPrefix(t, "|-"), strings.HasPrefix(t, "|}"):
			flush()
		case strings.HasPrefix(t, "|"):
			for _, c := range strings.Split(strings.TrimPrefix(t, "|"), "||") {
				cur = append(cur, stripWiki(c))
			}
		}
	}
	flush()

	nameIdx, kindIdx, costIdx := columnIndices(header)

	out := make([]rawItem, 0, len(rows))
	for _, r := range rows {
		name := at(r, nameIdx)
		if name == "" {
			continue // ligne sans nom : on n'invente pas
		}
		out = append(out, rawItem{
			Name:        name,
			Kind:        classifyKind(at(r, kindIdx)),
			FpCost:      parseCost(at(r, costIdx)),
			Description: optText(at(r, descIndex(header))),
		})
	}
	return out
}

// columnIndices résout les colonnes nom/catégorie/coût depuis l'en-tête (défaut
// positionnel 0/1/2 si absent ou non reconnu).
func columnIndices(header []string) (name, kind, cost int) {
	name, kind, cost = 0, 1, 2
	for i, h := range header {
		switch {
		case strings.Contains(h, "item"), strings.Contains(h, "name"), strings.Contains(h, "reward"):
			name = i
		case strings.Contains(h, "categ"), strings.Contains(h, "type"), strings.Contains(h, "kind"):
			kind = i
		case strings.Contains(h, "cost"), strings.Contains(h, "fp"), strings.Contains(h, "price"), strings.Contains(h, "point"):
			cost = i
		}
	}
	return
}

// descIndex repère une colonne description (-1 si aucune).
func descIndex(header []string) int {
	for i, h := range header {
		if strings.Contains(h, "desc") || strings.Contains(h, "note") {
			return i
		}
	}
	return -1
}

func at(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return row[idx]
}

// reDigits capte le premier entier d'une cellule de coût (« 750 FP », « 1,500 »).
var reDigits = regexp.MustCompile(`\d[\d,]*`)

func parseCost(s string) *int {
	m := reDigits.FindString(s)
	if m == "" {
		return nil
	}
	n, err := strconv.Atoi(strings.ReplaceAll(m, ",", ""))
	if err != nil {
		return nil
	}
	return &n
}

func optText(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// classifyKind range une catégorie libre de la source dans l'enum du contrat.
func classifyKind(s string) string {
	l := strings.ToLower(s)
	switch {
	case strings.Contains(l, "car"), strings.Contains(l, "vehicle"):
		return "car"
	case strings.Contains(l, "horn"):
		return "horn"
	case strings.Contains(l, "cloth"), strings.Contains(l, "outfit"), strings.Contains(l, "apparel"), strings.Contains(l, "wear"), strings.Contains(l, "suit"):
		return "clothing"
	case strings.Contains(l, "link"), strings.Contains(l, "phrase"), strings.Contains(l, "emote"):
		return "forza_link_phrase"
	default:
		return "other"
	}
}
