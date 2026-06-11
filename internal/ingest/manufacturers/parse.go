package manufacturers

import (
	"fmt"
	"sort"
	"strings"

	"github.com/zinackes/forza-open-api/internal/store"
)

// Report résume une passe de normalisation (logué par l'appelant + CheckRun).
// UnknownOrigins liste les codes origin non mappés (anomalies à investiguer, pas
// inventés : country reste NULL pour ces entrées).
type Report struct {
	Rows            int            // pages constructeurs traitées
	CountryResolved int            // entrées dont l'origin a été mappée en country
	NoOrigin        int            // entrées sans champ origin (country NULL, pas une anomalie)
	UnknownOrigins  map[string]int // token origin non mappé → count (country NULL + anomalie)
}

func newReport() Report {
	return Report{UnknownOrigins: map[string]int{}}
}

// Normalize projette les pages constructeurs (titre → wikitext) sur des
// store.Manufacturer prêts à l'upsert, pour le jeu `game`. name = titre de page ;
// country = champ {{InfoboxMFR|origin}} mappé via countryNames (NULL si absent ou
// non reconnu — jamais inventé). Déduplique par nom (déterministe).
func Normalize(game string, pages map[string]string) ([]store.Manufacturer, Report) {
	rep := newReport()
	out := make([]store.Manufacturer, 0, len(pages))
	seen := make(map[string]bool, len(pages))

	// Ordre stable (les maps Go itèrent en ordre aléatoire) : tri par titre.
	titles := make([]string, 0, len(pages))
	for t := range pages {
		titles = append(titles, t)
	}
	sort.Strings(titles)

	for _, name := range titles {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		rep.Rows++

		origin := parseOrigin(pages[name])
		country, token, recognized := countryFromOrigin(origin)
		switch {
		case origin == "":
			rep.NoOrigin++
		case !recognized:
			rep.UnknownOrigins["origin:"+token]++
		default:
			rep.CountryResolved++
		}

		out = append(out, store.Manufacturer{Game: game, Name: name, Country: country})
	}
	return out, rep
}

// parseOrigin extrait le champ |origin = … de l'{{InfoboxMFR}} d'une page
// constructeur (lowercased, trimmé). "" si pas d'infobox ou pas de champ origin.
func parseOrigin(wikitext string) string {
	body, ok := firstTemplateBody(wikitext, "InfoboxMFR")
	if !ok {
		return ""
	}
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			continue
		}
		kv := strings.SplitN(line[1:], "=", 2)
		if len(kv) != 2 {
			continue
		}
		if strings.TrimSpace(kv[0]) == "origin" {
			return strings.ToLower(strings.TrimSpace(kv[1]))
		}
	}
	return ""
}

// countryFromOrigin mappe un token origin du wiki en nom de pays canonique.
// L'origin peut porter un suffixe drapeau (« usa f », « uk f ») : on ne retient que
// le PREMIER champ. origin vide → (nil, "", true) (absence, pas une anomalie).
// Token non reconnu → (nil, token, false) (country NULL + anomalie reportée).
func countryFromOrigin(origin string) (country *string, token string, recognized bool) {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return nil, "", true
	}
	token = strings.Fields(origin)[0]
	if name, ok := countryNames[token]; ok {
		return &name, token, true
	}
	return nil, token, false
}

// countryNames traduit les codes/noms origin du wiki Forza (champ {{InfoboxMFR|origin}})
// en nom de pays canonique. Construit à partir du vocabulaire OBSERVÉ sur l'ensemble
// des pages Category:Manufacturers (codes ISO-ish + noms anglais que les éditeurs
// mélangent). Aucune valeur inventée : un token absent de la table → country NULL.
// « ukger » = MINI (marque britannique, propriété BMW) → primaire United Kingdom.
var countryNames = map[string]string{
	"usa":         "United States",
	"us":          "United States",
	"uk":          "United Kingdom",
	"ukger":       "United Kingdom",
	"ger":         "Germany",
	"germany":     "Germany",
	"ita":         "Italy",
	"italy":       "Italy",
	"fra":         "France",
	"france":      "France",
	"jpn":         "Japan",
	"japan":       "Japan",
	"chn":         "China",
	"kor":         "South Korea",
	"spain":       "Spain",
	"can":         "Canada",
	"netherlands": "Netherlands",
	"sw":          "Sweden",
	"sweden":      "Sweden",
	"austria":     "Austria",
	"cro":         "Croatia",
	"denmark":     "Denmark",
	"ind":         "India",
	"mexico":      "Mexico",
	"australia":   "Australia",
	"uae":         "United Arab Emirates",
}

// --- Extraction de template wiki (braces équilibrées) -------------------------

// firstTemplateBody renvoie le corps du premier template `name` (sans le nom de
// tête), en respectant l'imbrication {{ }}. ok=false si absent ou non clos.
func firstTemplateBody(wikitext, name string) (string, bool) {
	open := "{{" + name
	i := strings.Index(wikitext, open)
	if i < 0 {
		return "", false
	}
	body, end := readBalanced(wikitext, i+2) // après "{{"
	if end < 0 {
		return "", false
	}
	if idx := strings.IndexByte(body, '|'); idx >= 0 {
		return body[idx:], true // garde les "|clé = valeur"
	}
	return body, true
}

// readBalanced lit à partir de start (juste après "{{") jusqu'au "}}" appariant,
// en suivant l'imbrication. Renvoie le contenu intérieur et l'index après "}}"
// (ou -1 si non clos).
func readBalanced(s string, start int) (string, int) {
	depth := 1
	i := start
	for i+1 < len(s) {
		switch {
		case s[i] == '{' && s[i+1] == '{':
			depth++
			i += 2
		case s[i] == '}' && s[i+1] == '}':
			depth--
			if depth == 0 {
				return s[start:i], i + 2
			}
			i += 2
		default:
			i++
		}
	}
	return "", -1
}

// SortedReport rend une distribution de façon déterministe (log/test).
func SortedReport(m map[string]int) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteString(" ")
		}
		fmt.Fprintf(&b, "%s=%d", k, m[k])
	}
	return b.String()
}
