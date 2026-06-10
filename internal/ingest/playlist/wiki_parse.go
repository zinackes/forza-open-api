package playlist

// Parsing du wikitext Festival Playlist (pages « Series N » + sous-pages saison).
// Tout est en templates MediaWiki imbriqués :
//   {{FH6FPEvent|d1|Big in Japan|<requirement>|{{FH6FPReward|cr| |5,000|c}}|6=Icon}}
// → un tokenizer brace-aware (findTemplates/splitParams) extrait les templates et
// leurs paramètres en respectant l'imbrication {{…}} et les liens [[…]]. Aucun
// réseau ici : testé sur fixtures figées. Aucune donnée inventée : un champ absent
// reste vide/NULL côté store.

import (
	"regexp"
	"strconv"
	"strings"
)

// seriesInfo est l'identité d'une série lue sur sa page « Series N » (infobox +
// paliers série). total = points disponibles sur la série (pour at_percent).
type seriesInfo struct {
	name    string
	series  int
	start   string
	end     string
	total   int
	rewards []parsedReward // paliers série (Type = "series")
}

// seasonDetail est le détail d'une saison (sous-page « <Season> Season ») : paliers
// saison + défis (weekly agrégé, daily, events seasonal/online/monthly).
type seasonDetail struct {
	season     string
	start      string
	end        string
	total      int
	rewards    []parsedReward
	challenges []parsedChallenge
}

var (
	reWeeklyCode  = regexp.MustCompile(`^(ft)?w\d+$`)
	reDailyCode   = regexp.MustCompile(`^(ft)?d\d+$`)
	reWeeklyTitle = regexp.MustCompile(`(?i)weekly challenge:\s*"?([^"]+?)"?\s*$`)
	reWikiLink    = regexp.MustCompile(`\[\[[^\]]*\]\]`)
	reWSpace      = regexp.MustCompile(`\s+`)
)

// infobox renvoie les paramètres nommés de l'infobox Festival Playlist d'une page
// (nil si absente).
func (g wikiGame) infobox(wt string) map[string]string {
	bodies := findTemplates(wt, "Infobox"+g.tmplPrefix+"FestivalPlaylist")
	if len(bodies) == 0 {
		return nil
	}
	_, named := templateFields(bodies[0])
	return named
}

// parseSeries lit l'identité d'une série + ses paliers (seriescar/seriesgift).
func (g wikiGame) parseSeries(wt string) seriesInfo {
	si := seriesInfo{}
	if ib := g.infobox(wt); ib != nil {
		si.name = ib["name"]
		si.series = atoi(ib["series"])
		si.start = ib["start"]
		si.end = ib["end"]
		si.total = atoi(ib["total"])
		if si.total == 0 {
			si.total = atoi(ib["summer"]) + atoi(ib["autumn"]) + atoi(ib["winter"]) + atoi(ib["spring"])
		}
		if si.total == 0 {
			si.total = atoi(ib["complete"])
		}
	}
	for _, body := range g.events(wt) {
		pos, _ := templateFields(body)
		if !strings.HasPrefix(eventCode(pos), "series") {
			continue
		}
		if item := g.renderReward(joinFrom(pos, 2)); item != "" {
			si.rewards = append(si.rewards, parsedReward{
				Type: "series", Item: item, AtPercent: pct(atoi(at(pos, 1)), si.total),
			})
		}
	}
	return si
}

// parseSeason lit le détail d'une sous-page saison.
func (g wikiGame) parseSeason(wt string) seasonDetail {
	sd := seasonDetail{}
	if ib := g.infobox(wt); ib != nil {
		sd.season = strings.ToLower(ib["season"])
		sd.start = ib["start"]
		sd.end = ib["end"]
		sd.total = atoi(ib["total"])
	}

	section, sub := "preamble", ""
	var weeklyName, weeklyReward string
	var weeklyReqs []string
	flushWeekly := func() {
		if len(weeklyReqs) > 0 {
			sd.challenges = append(sd.challenges, parsedChallenge{
				Scope: "weekly", Name: weeklyName,
				Requirement: strings.Join(weeklyReqs, "; "), Reward: weeklyReward,
			})
		}
		weeklyName, weeklyReward, weeklyReqs = "", "", nil
	}

	for _, line := range strings.Split(wt, "\n") {
		if title, lvl := header(strings.TrimSpace(line)); lvl > 0 {
			flushWeekly()
			if lvl == 2 {
				section, sub = classifySection(title), ""
			} else {
				sub = ""
				switch {
				case isWeeklyHeader(title):
					sub, weeklyName = "weekly", weeklyTitle(title)
				case isDailyHeader(title):
					sub = "daily"
				case strings.Contains(strings.ToLower(title), "additional"):
					sub = "additional"
				}
			}
			continue
		}
		for _, body := range g.events(line) {
			g.handleEvent(body, section, sub, &sd, &weeklyReqs, &weeklyReward)
		}
	}
	flushWeekly()
	return sd
}

// handleEvent classe un {{FH*FPEvent}} : palier (series/season → reward), chapitre
// weekly (agrégé), daily, ou event d'une section (seasonal/online/monthly).
func (g wikiGame) handleEvent(body, section, sub string, sd *seasonDetail, weeklyReqs *[]string, weeklyReward *string) {
	pos, _ := templateFields(body)
	code := eventCode(pos)
	switch {
	case strings.HasPrefix(code, "season"):
		if item := g.renderReward(joinFrom(pos, 2)); item != "" {
			sd.rewards = append(sd.rewards, parsedReward{
				Type: "season", Item: item, AtPercent: pct(atoi(at(pos, 1)), sd.total),
			})
		}
	case strings.HasPrefix(code, "series"):
		// Palier série (capturé depuis la page série) : ignoré ici pour ne pas
		// le compter deux fois.
	case reWeeklyCode.MatchString(code):
		if req := renderWiki(at(pos, 2)); req != "" {
			*weeklyReqs = append(*weeklyReqs, req)
		}
		if rw := g.rewardFrom(pos, 2); rw != "" {
			*weeklyReward = rw
		}
	case reDailyCode.MatchString(code):
		g.addEventChallenge(sd, "daily", pos)
	case code == "test":
		// « Test Drive » : voiture à essayer, pas un défi à points → ignoré.
	default:
		if scope := sectionScope(section, sub); scope != "" {
			g.addEventChallenge(sd, scope, pos)
		}
	}
}

// addEventChallenge ajoute un défi (daily/seasonal/online/monthly) depuis un
// {{FH*FPEvent|code|name|requirement|reward}}. Un event sans nom ni requirement
// (séparateur de section type « arcade ») est ignoré.
func (g wikiGame) addEventChallenge(sd *seasonDetail, scope string, pos []string) {
	name := renderWiki(at(pos, 1))
	req := renderWiki(at(pos, 2))
	if name == "" && req == "" {
		return
	}
	sd.challenges = append(sd.challenges, parsedChallenge{
		Scope: scope, Name: name, Requirement: req, Reward: g.rewardFrom(pos, 2),
	})
}

// events renvoie le corps de chaque {{FH*FPEvent|…}} présent dans s (un par ligne
// dans le wikitext, mais robuste à plusieurs).
func (g wikiGame) events(s string) []string {
	if !strings.Contains(s, g.tmplPrefix+"FPEvent") {
		return nil
	}
	return findTemplates(s, g.tmplPrefix+"FPEvent")
}

// rewardFrom rend la première récompense {{FH*FPReward}} trouvée dans pos[from:].
func (g wikiGame) rewardFrom(pos []string, from int) string {
	for i := from; i < len(pos); i++ {
		if strings.Contains(pos[i], g.tmplPrefix+"FPReward") {
			return g.renderReward(pos[i])
		}
	}
	return ""
}

// renderReward projette un {{FH*FPReward|type|…}} en libellé lisible. Aucun nom
// inventé : on rend ce que porte le template (nom de voiture, crédits, phrase LINK,
// wheelspin…).
func (g wikiGame) renderReward(s string) string {
	bodies := findTemplates(s, g.tmplPrefix+"FPReward")
	if len(bodies) == 0 {
		return ""
	}
	pos, _ := templateFields(bodies[0])
	switch strings.ToLower(strings.TrimSpace(at(pos, 0))) {
	case "wheelspin":
		return "Wheelspin"
	case "super", "superwheelspin":
		return "Super Wheelspin"
	case "cr", "credits":
		if v := renderWiki(at(pos, 2)); v != "" {
			return v + " Credits"
		}
		return "Credits"
	case "link", "forzalink":
		if v := renderWiki(at(pos, 2)); v != "" {
			return "Forza LINK: " + v
		}
		return "Forza LINK Phrase"
	case "pass":
		if v := renderWiki(at(pos, 1)); v != "" {
			return v
		}
		return "Backstage Pass"
	default:
		// Voiture/objet : nom complet en param 1 (underscores), sinon libellé du lien.
		if v := cleanName(at(pos, 1)); v != "" {
			return v
		}
		return renderWiki(at(pos, 2))
	}
}

// --- tokenizer wikitext ------------------------------------------------------

// findTemplates renvoie le contenu (nom + paramètres, sans les accolades) de
// chaque {{name|…}} de premier niveau dans s, en respectant l'imbrication {{…}}.
func findTemplates(s, name string) []string {
	lname := strings.ToLower(name)
	var out []string
	for i := 0; i+1 < len(s); i++ {
		if s[i] != '{' || s[i+1] != '{' {
			continue
		}
		// Nom du template : jusqu'au premier | ou }.
		k := i + 2
		for k < len(s) && s[k] != '|' && s[k] != '}' {
			k++
		}
		if strings.ToLower(strings.TrimSpace(s[i+2:k])) != lname {
			continue
		}
		// Fin = }} appariant, profondeur suivie.
		depth, p := 1, i+2
		for p+1 < len(s) && depth > 0 {
			switch {
			case s[p] == '{' && s[p+1] == '{':
				depth++
				p += 2
			case s[p] == '}' && s[p+1] == '}':
				depth--
				p += 2
			default:
				p++
			}
		}
		out = append(out, s[i+2:p-2])
		i = p - 1
	}
	return out
}

// templateFields découpe « name|a|b|key=v » en paramètres positionnels (hors nom)
// et nommés, en respectant l'imbrication {{…}} et [[…]].
func templateFields(raw string) (pos []string, named map[string]string) {
	named = map[string]string{}
	for i, f := range splitParams(raw) {
		if i == 0 {
			continue // nom du template
		}
		if k, v, ok := splitNamedParam(f); ok {
			named[strings.ToLower(strings.TrimSpace(k))] = strings.TrimSpace(v)
		} else {
			pos = append(pos, f)
		}
	}
	return pos, named
}

// splitParams découpe sur les | de premier niveau (hors {{…}} et [[…]]).
func splitParams(s string) []string {
	var parts []string
	curl, brak, last := 0, 0, 0
	for i := 0; i < len(s); i++ {
		if i+1 < len(s) && s[i] == '{' && s[i+1] == '{' {
			curl++
			i++
		} else if i+1 < len(s) && s[i] == '}' && s[i+1] == '}' {
			if curl > 0 {
				curl--
			}
			i++
		} else if i+1 < len(s) && s[i] == '[' && s[i+1] == '[' {
			brak++
			i++
		} else if i+1 < len(s) && s[i] == ']' && s[i+1] == ']' {
			if brak > 0 {
				brak--
			}
			i++
		} else if s[i] == '|' && curl == 0 && brak == 0 {
			parts = append(parts, s[last:i])
			last = i + 1
		}
	}
	return append(parts, s[last:])
}

// splitNamedParam repère un paramètre nommé « key = value » : premier = de premier
// niveau (hors {{…}}/[[…]]). Renvoie ok=false pour un paramètre positionnel.
func splitNamedParam(f string) (key, val string, ok bool) {
	curl, brak := 0, 0
	for i := 0; i < len(f); i++ {
		if i+1 < len(f) && f[i] == '{' && f[i+1] == '{' {
			curl++
			i++
		} else if i+1 < len(f) && f[i] == '}' && f[i+1] == '}' {
			if curl > 0 {
				curl--
			}
			i++
		} else if i+1 < len(f) && f[i] == '[' && f[i+1] == '[' {
			brak++
			i++
		} else if i+1 < len(f) && f[i] == ']' && f[i+1] == ']' {
			if brak > 0 {
				brak--
			}
			i++
		} else if f[i] == '=' && curl == 0 && brak == 0 {
			return f[:i], f[i+1:], true
		}
	}
	return "", "", false
}

// renderWiki aplatit un fragment wikitext en texte lisible : templates de mise en
// forme résolus (innermost-first), liens [[…]] → libellé, gras/italique retirés.
func renderWiki(s string) string {
	for {
		start := strings.LastIndex(s, "{{")
		if start < 0 {
			break
		}
		rel := strings.Index(s[start:], "}}")
		if rel < 0 {
			break
		}
		end := start + rel
		s = s[:start] + renderTemplateArgs(s[start+2:end]) + s[end+2:]
	}
	s = reWikiLink.ReplaceAllStringFunc(s, linkDisplay)
	s = strings.ReplaceAll(s, "'''", "")
	s = strings.ReplaceAll(s, "''", "")
	return strings.TrimSpace(reWSpace.ReplaceAllString(s, " "))
}

// renderTemplateArgs rend un template de mise en forme (sans accolades) en texte :
// {{speed|85.0}}→« 85.0 mph », {{distance|800.5|ft}}→« 800.5 ft », sinon les
// paramètres positionnels joints (classe en majuscule).
func renderTemplateArgs(inner string) string {
	parts := splitParams(inner)
	if len(parts) == 0 {
		return ""
	}
	name := strings.ToLower(strings.TrimSpace(parts[0]))
	var vals []string
	for _, a := range parts[1:] {
		if _, _, ok := splitNamedParam(a); ok {
			continue
		}
		if a = strings.TrimSpace(a); a != "" {
			vals = append(vals, a)
		}
	}
	switch name {
	case "speed":
		if len(vals) > 0 {
			return vals[0] + " mph"
		}
		return ""
	case "distance":
		return strings.Join(vals, " ")
	}
	if len(vals) > 0 && len(vals[0]) <= 2 {
		vals[0] = strings.ToUpper(vals[0]) // jeton de classe (b, s1…)
	}
	return strings.Join(vals, " ")
}

// linkDisplay rend un lien wiki [[cible|libellé]]→libellé, [[a/b/c]]→« c ».
func linkDisplay(link string) string {
	inner := strings.TrimSuffix(strings.TrimPrefix(link, "[["), "]]")
	if i := strings.Index(inner, "|"); i >= 0 {
		return strings.TrimSpace(inner[i+1:])
	}
	inner = strings.TrimPrefix(inner, ":")
	if i := strings.LastIndex(inner, "/"); i >= 0 {
		inner = inner[i+1:]
	}
	return strings.TrimSpace(inner)
}

// --- helpers sections / headers ----------------------------------------------

func header(t string) (string, int) {
	if len(t) < 4 || !strings.HasPrefix(t, "==") || !strings.HasSuffix(t, "==") {
		return "", 0
	}
	lvl := 0
	for lvl < len(t)/2 && t[lvl] == '=' && t[len(t)-1-lvl] == '=' {
		lvl++
	}
	return strings.TrimSpace(t[lvl : len(t)-lvl]), lvl
}

// classifySection mappe un titre de section de niveau 2 vers une catégorie d'event.
func classifySection(h string) string {
	l := strings.ToLower(h)
	switch {
	case strings.Contains(l, "shop"):
		return "skip"
	case strings.Contains(l, "seasonal event"):
		return "seasonal"
	case l == "online" || strings.Contains(l, "online "):
		return "online"
	case strings.Contains(l, "monthly"):
		return "monthly"
	case strings.Contains(l, "challenge") || strings.Contains(l, "forzathon"):
		return "challenges" // conteneur des sous-sections weekly/daily
	}
	return "skip"
}

// sectionScope donne le scope d'un event de section (« » = à ignorer).
func sectionScope(section, sub string) string {
	if sub == "additional" {
		return "seasonal"
	}
	switch section {
	case "seasonal", "online", "monthly":
		return section
	}
	return ""
}

func isWeeklyHeader(h string) bool { return strings.Contains(strings.ToLower(h), "weekly challenge") }
func isDailyHeader(h string) bool  { return strings.Contains(strings.ToLower(h), "daily challenge") }

func weeklyTitle(h string) string {
	if m := reWeeklyTitle.FindStringSubmatch(h); m != nil {
		return strings.TrimSpace(m[1])
	}
	return "Weekly Challenge"
}

// --- petits helpers ----------------------------------------------------------

func eventCode(pos []string) string { return strings.ToLower(strings.TrimSpace(at(pos, 0))) }

func at(pos []string, i int) string {
	if i < 0 || i >= len(pos) {
		return ""
	}
	return pos[i]
}

func joinFrom(pos []string, from int) string {
	if from >= len(pos) {
		return ""
	}
	return strings.Join(pos[from:], "|")
}

// cleanName normalise un identifiant de voiture/objet (underscores → espaces).
func cleanName(s string) string {
	return strings.TrimSpace(reWSpace.ReplaceAllString(strings.ReplaceAll(s, "_", " "), " "))
}

func atoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}
