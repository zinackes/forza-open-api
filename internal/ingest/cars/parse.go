package cars

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/zinackes/forza-open-api/internal/store"
)

// ListRow est une ligne brute de la page liste (template {{CarListStats…}}).
// Positions (1-indexées) du template FH6 :
//
//	1 nom (= titre de la page voiture)   2 libellé variante (FE/WP, optionnel)
//	3 année   4 code rareté   5 valeur (cr)   6 code d'obtention
//	7..12 stats (speed,handling,acceleration,launch,braking,offroad)
//	13 PI   14 code pays   (15+ = paramètres nommés ignorés)
type ListRow struct {
	Name    string
	Variant string
	Year    string
	Rarity  string
	Value   string
	Obtain  string
	Stats   [6]string
	PI      string
	Country string
}

// Infobox porte les champs du {{CarInfobox}} d'une page voiture utiles au
// catalogue. Les autres champs (puissance, poids…) ne sont pas captés ici.
type Infobox struct {
	Manufacturer string
	Model        string
	Layout       string // ex. "ff", "mr", "f4" → drivetrain
	Type         string
	Origin       string
}

// Report résume une passe d'ingestion (rapport logué par cmd/seed). SkipReasons
// agrège les voitures écartées par motif ; UnknownCodes liste les codes
// d'obtention/layout non mappés (anomalies à investiguer, pas inventés).
type Report struct {
	Rows         int
	Imported     int            // lignes normalisées avec succès (avant dédup)
	Distinct     int            // voitures distinctes upsertées (= lignes en DB)
	Collisions   map[string]int // id → nb de lignes le partageant (>1) : doublons de listing
	SkipReasons  map[string]int
	Classes      map[string]int
	Drivetrains  map[string]int
	UnknownCodes map[string]int
}

func newReport() Report {
	return Report{
		Collisions:   map[string]int{},
		SkipReasons:  map[string]int{},
		Classes:      map[string]int{},
		Drivetrains:  map[string]int{},
		UnknownCodes: map[string]int{},
	}
}

// ParseCarList extrait les lignes du template `tmpl` (ex. "CarListStatsFH6")
// depuis le wikitext de la page liste. Les lignes trop courtes (template non
// conforme) sont ignorées silencieusement (comptées via Rows vs len).
func ParseCarList(wikitext, tmpl string) []ListRow {
	bodies := extractTemplateBodies(wikitext, tmpl)
	rows := make([]ListRow, 0, len(bodies))
	for _, body := range bodies {
		f := splitParams(body)
		if len(f) < 14 {
			continue
		}
		r := ListRow{
			Name:    strings.TrimSpace(f[0]),
			Variant: strings.TrimSpace(f[1]),
			Year:    strings.TrimSpace(f[2]),
			Rarity:  strings.TrimSpace(f[3]),
			Value:   strings.TrimSpace(f[4]),
			Obtain:  strings.TrimSpace(f[5]),
			PI:      strings.TrimSpace(f[12]),
			Country: strings.TrimSpace(f[13]),
		}
		for i := 0; i < 6; i++ {
			r.Stats[i] = strings.TrimSpace(f[6+i])
		}
		if r.Name != "" {
			rows = append(rows, r)
		}
	}
	return rows
}

// ParseInfobox extrait les champs {{CarInfobox|key = value}} d'une page voiture.
// Parsing ligne à ligne : suffit aux champs mono-ligne dont on a besoin.
func ParseInfobox(wikitext string) Infobox {
	body, ok := firstTemplateBody(wikitext, "CarInfobox")
	if !ok {
		return Infobox{}
	}
	var ib Infobox
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			continue
		}
		kv := strings.SplitN(line[1:], "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		switch key {
		case "manufacturer":
			ib.Manufacturer = val
		case "model":
			ib.Model = val
		case "layout":
			ib.Layout = val
		case "type":
			ib.Type = val
		case "origin":
			ib.Origin = val
		}
	}
	return ib
}

// Normalize projette les lignes liste + infoboxes sur des store.Car prêts à
// l'upsert, pour le jeu `game`. Une voiture n'est retenue que si elle a les
// champs requis par le contrat (PI → class, et drivetrain via l'infobox) ;
// sinon elle est écartée et comptée dans le rapport. Aucun champ n'est inventé :
// absent → NULL (pointeur nil).
func Normalize(rows []ListRow, infoboxes map[string]Infobox, game string) ([]store.Car, Report) {
	rep := newReport()
	rep.Rows = len(rows)
	out := make([]store.Car, 0, len(rows))
	seen := make(map[string]int, len(rows)) // id → index dans out

	for _, r := range rows {
		pi, err := strconv.Atoi(r.PI)
		if err != nil || pi < 100 || pi > 999 {
			rep.SkipReasons["no_pi"]++
			continue
		}
		class := classFromPI(game, pi)
		if class == "" {
			rep.SkipReasons["no_class"]++
			continue
		}
		ib, ok := infoboxes[r.Name]
		if !ok {
			rep.SkipReasons["no_infobox"]++
			continue
		}
		drivetrain := drivetrainFromLayout(ib.Layout)
		if drivetrain == "" {
			if ib.Layout != "" {
				rep.UnknownCodes["layout:"+ib.Layout]++
			}
			rep.SkipReasons["no_drivetrain"]++
			continue
		}

		isFE := strings.EqualFold(r.Rarity, "fe")
		name := r.Name
		id := game + "-" + slug(r.Name)
		// FH6 liste les FE sous le nom de la voiture de base → suffixer pour
		// les distinguer. FH5 leur dédie une page dont le titre contient déjà
		// « Forza Edition » → ne pas doubler le suffixe.
		if isFE && !strings.Contains(name, "Forza Edition") {
			name += " Forza Edition"
			id += "-forza-edition"
		}

		model := nilIfEmpty(strings.TrimSpace(ib.Model))
		carMake := deriveMake(r.Name, ib.Model)

		c := store.Car{
			ID:           id,
			Game:         game,
			Name:         name,
			Make:         carMake,
			Model:        model,
			Year:         atoiPtr(r.Year),
			Class:        class,
			PI:           pi,
			Drivetrain:   drivetrain,
			Stats:        statsJSON(r.Stats),
			Rarity:       rarityFromCode(r.Rarity),
			ValueCr:      valueCrPtr(r.Value),
			ObtainMethod: obtainMethod(game, r.Obtain, rep),
			// body_type / category / image_url : non sourcés proprement → NULL.
		}
		rep.Imported++
		// Dédup par id (= page wiki) : une même voiture est parfois listée deux
		// fois — version stock (Autoshow) et copie pré-tunée du Welcome Pack, au
		// PI plus élevé. Le catalogue garde la voiture STOCK : on conserve donc
		// le PI le plus bas. Déterministe (indépendant de l'ordre des lignes).
		if idx, ok := seen[c.ID]; ok {
			rep.Collisions[c.ID]++
			if c.PI < out[idx].PI {
				out[idx] = c
			}
			continue
		}
		seen[c.ID] = len(out)
		out = append(out, c)
	}

	rep.Distinct = len(out)
	for i := range out {
		rep.Classes[out[i].Class]++
		rep.Drivetrains[out[i].Drivetrain]++
	}
	return out, rep
}

// CarTitles renvoie les titres de pages voitures à récupérer (déduplique sur le
// nom = titre de la page). Sert à FetchCarPages.
func CarTitles(rows []ListRow) []string {
	seen := make(map[string]bool, len(rows))
	titles := make([]string, 0, len(rows))
	for _, r := range rows {
		if r.Name == "" || seen[r.Name] {
			continue
		}
		seen[r.Name] = true
		titles = append(titles, r.Name)
	}
	return titles
}

// --- Tables de normalisation (sourcées du wiki, pas inventées) ----------------

// classFromPI applique les bandes PI→class publiées du jeu (page wiki
// « Performance Index »). FH6 : D 100-400, C 401-500, B 501-600, A 601-700,
// S1 701-800, S2 801-900, R 901-998, X 999. FH5 (série Horizon avant FH6,
// pas de classe R) : D 100-500, C 501-600, B 601-700, A 701-800, S1 801-900,
// S2 901-998, X 999. Jeu inconnu → "" (voiture écartée).
func classFromPI(game string, pi int) string {
	switch game {
	case "fh6":
		switch {
		case pi <= 400:
			return "D"
		case pi <= 500:
			return "C"
		case pi <= 600:
			return "B"
		case pi <= 700:
			return "A"
		case pi <= 800:
			return "S1"
		case pi <= 900:
			return "S2"
		case pi <= 998:
			return "R"
		default:
			return "X"
		}
	case "fh5":
		switch {
		case pi <= 500:
			return "D"
		case pi <= 600:
			return "C"
		case pi <= 700:
			return "B"
		case pi <= 800:
			return "A"
		case pi <= 900:
			return "S1"
		case pi <= 998:
			return "S2"
		default:
			return "X"
		}
	}
	return ""
}

// drivetrainFromLayout dérive le drivetrain du code layout de l'infobox : le 2e
// caractère code les roues motrices (f=avant, r=arrière, 4/a=intégrale).
func drivetrainFromLayout(layout string) string {
	layout = strings.ToLower(strings.TrimSpace(layout))
	switch layout {
	case "fwd", "rwd", "awd":
		return strings.ToUpper(layout)
	case "4wd":
		return "AWD"
	}
	if len(layout) < 2 {
		return ""
	}
	switch layout[1] {
	case 'f':
		return "FWD"
	case 'r':
		return "RWD"
	case '4', 'a':
		return "AWD"
	}
	return ""
}

var rarityNames = map[string]string{
	"c": "common", "r": "rare", "e": "epic", "l": "legendary",
	"fe": "forza_edition", "barn": "barn_find", "treasure": "treasure",
	// Codes FH5 (badge du template CarListStatsFH5) : mot affiche le badge
	// FORZA EDITION, ae ANNIVERSARY, de DONUT EDITION.
	"mot": "forza_edition", "ae": "anniversary", "de": "donut_edition",
}

func rarityFromCode(code string) *string {
	if name, ok := rarityNames[strings.ToLower(strings.TrimSpace(code))]; ok {
		return &name
	}
	return nil
}

// obtainNames traduit les codes d'obtention (switch du template wiki de chaque
// jeu) en texte. Une table PAR JEU : les mêmes codes ont des sens différents
// d'un template à l'autre (ex. ita, pass, wp, as entre FH5 et FH6).
var obtainNames = map[string]map[string]string{
	"fh6": obtainNamesFH6,
	"fh5": obtainNamesFH5,
}

// obtainNamesFH6 : switch du template {{CarListStatsFH6}}.
var obtainNamesFH6 = map[string]string{
	"auto": "Autoshow", "w": "Wristband reward", "wh": "Wheelspin",
	"whw": "Wheelspin, Wristband reward", "barn": "Barn Find",
	"treasure": "Treasure Car", "cm": "Car Mastery", "htf": "Hard to Find",
	"awh": "Autoshow, Wheelspin", "as": "Autoshow, Complete the Prologue",
	"awhs": "Autoshow, Wheelspin, Complete the Prologue",
	"ay":   "Autoshow, Yellow Wristband",
	"awhy": "Autoshow, Wheelspin, Yellow Wristband",
	"awhl": "Autoshow, Wheelspin, Loyalty Reward", "af": "Aftermarket Car",
	"aaf": "Autoshow, Aftermarket Car", "awhaf": "Autoshow, Wheelspin, Aftermarket Car",
	"j": "Collection Journal", "aj": "Autoshow, Collection Journal",
	"whj": "Wheelspin, Collection Journal", "awhj": "Autoshow, Wheelspin, Collection Journal",
	"aafj":   "Autoshow, Aftermarket Car, Collection Journal",
	"awhafj": "Autoshow, Wheelspin, Aftermarket Car, Collection Journal",
	"pass":   "Car Pass", "wtac": "Time Attack Car Pack", "wp": "Welcome Pack",
	"vip": "VIP Membership", "un": "Unobtainable", "preorder": "Pre-order",
	"apromo": "Autoshow, Promotional", "ita": "Italian Passion Car Pack",
}

// obtainNamesFH5 : switch du template {{CarListStatsFH5}}. Les détails portés
// par des paramètres nommés du wiki (nom d'accolade, de story, de mission…)
// ne sont pas captés par ParseCarList → texte générique du code seul.
var obtainNamesFH5 = map[string]string{
	"auto": "Autoshow", "as": "Autoshow, Showcase reward", "wh": "Wheelspin",
	"mb": "Midnight Battle", "whmb": "Wheelspin, Midnight Battle",
	"ac": "Accolade reward", "barn": "Barn Find", "cm": "Car Mastery",
	"cc": "Car Collector", "htf": "Hard to Find", "awh": "Autoshow, Wheelspin",
	"aa": "Autoshow, Accolade reward", "wha": "Wheelspin, Accolade reward",
	"awha": "Autoshow, Wheelspin, Accolade reward", "hs": "Horizon Story",
	"hwa":  "HW Academy, Autoshow (DLC: Hot Wheels)",
	"hww":  "HW Academy (DLC: Hot Wheels), Wheelspin",
	"ra":   "Expedition, Autoshow (DLC: Rally Adventure)",
	"rah":  "Horizon Raptors, Autoshow (DLC: Rally Adventure)",
	"raa":  "Apex Predators, Autoshow (DLC: Rally Adventure)",
	"rag":  "Grit Reapers, Autoshow (DLC: Rally Adventure)",
	"pass": "Autoshow (DLC: Car Pass)", "fd": "Autoshow (DLC: Formula Drift Pack)",
	"hr":    "Autoshow (DLC: Horizon Racing Car Pack)",
	"ita":   "Autoshow (DLC: Italian Exotics Car Pack)",
	"spd":   "Autoshow (DLC: Super Speed Car Pack)",
	"ama":   "Autoshow (DLC: American Automotive Car Pack)",
	"fx":    "Autoshow (DLC: Fast X Car Pack)",
	"cls":   "Autoshow (DLC: Chinese Lucky Stars Car Pack)",
	"euro":  "Autoshow (DLC: European Automotive Car Pack)",
	"acc":   "Autoshow (DLC: Acceleration Car Pack)",
	"apex":  "Autoshow (DLC: Apex Allstars Car Pack)",
	"uni":   "Autoshow (DLC: Universal Icons Car Pack)",
	"jdm":   "Autoshow (DLC: JDM Jewels Car Pack)",
	"nrr":   "Autoshow (DLC: Nissan Retro Rides Car Pack)",
	"wp":    "Autoshow (DLC: Welcome Pack)",
	"vip":   "Gifted (DLC: VIP Membership)",
	"ctrl":  "Gifted (Forza Horizon 5 Limited Edition Xbox Controller)",
	"promo": "Gifted (Promotional)", "ap": "Autoshow, Gifted (Promotional)",
	"un": "Unobtainable",
}

// obtainMethod mappe le code en texte pour le jeu donné ; code inconnu → on
// conserve le code brut (donnée sourcée, pas inventée) et on l'enregistre
// comme anomalie.
func obtainMethod(game, code string, rep Report) *string {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil
	}
	if name, ok := obtainNames[game][strings.ToLower(code)]; ok {
		return &name
	}
	rep.UnknownCodes["obtain:"+code]++
	return &code
}

// statsJSON assemble le JSONB des stats (clés speed/handling/acceleration/launch/
// braking/offroad). Une stat absente/non numérique est OMISE (jamais inventée).
// Aucune stat exploitable → nil (NULL).
func statsJSON(s [6]string) []byte {
	keys := [6]string{"speed", "handling", "acceleration", "launch", "braking", "offroad"}
	m := make(map[string]float64, 6)
	for i, raw := range s {
		if v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64); err == nil {
			m[keys[i]] = v
		}
	}
	if len(m) == 0 {
		return nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	return b
}

// --- Petits helpers -----------------------------------------------------------

// deriveMake retire le suffixe model du nom complet pour isoler le constructeur
// (ex. "Abarth 695 Biposto" − "695 Biposto" = "Abarth"). À défaut, premier mot.
func deriveMake(name, model string) string {
	name = strings.TrimSpace(name)
	model = strings.TrimSpace(model)
	if model != "" {
		ln, lm := strings.ToLower(name), strings.ToLower(model)
		if i := strings.LastIndex(ln, lm); i > 0 {
			return strings.TrimSpace(name[:i])
		}
	}
	if i := strings.IndexByte(name, ' '); i > 0 {
		return name[:i]
	}
	return name
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func atoiPtr(s string) *int {
	if v, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
		return &v
	}
	return nil
}

// valueCrPtr parse une valeur en crédits ("250,000" → 250000). Vide/non numérique → nil.
func valueCrPtr(s string) *int64 {
	s = strings.ReplaceAll(strings.TrimSpace(s), ",", "")
	if s == "" {
		return nil
	}
	if v, err := strconv.ParseInt(s, 10, 64); err == nil {
		return &v
	}
	return nil
}

// slug produit un identifiant stable ASCII : minuscules, alphanum, '-' ailleurs.
func slug(s string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// --- Extraction de templates wiki (braces équilibrées) ------------------------

// extractTemplateBodies renvoie le corps (après "{{name|") de chaque occurrence
// du template `name`, en respectant l'imbrication {{ }}.
func extractTemplateBodies(wikitext, name string) []string {
	open := "{{" + name
	var out []string
	for i := 0; i+len(open) <= len(wikitext); {
		if wikitext[i:i+len(open)] != open {
			i++
			continue
		}
		// Le caractère suivant doit être un séparateur (| ou } ou espace),
		// sinon c'est un préfixe d'un autre template (ex. CarListStatsFH6X).
		next := wikitext[i+len(open)]
		if next != '|' && next != '}' && next != ' ' && next != '\n' {
			i++
			continue
		}
		body, end := readBalanced(wikitext, i+2) // après "{{"
		if end < 0 {
			break
		}
		// body = "name|p1|p2…" → retirer le nom de tête.
		if idx := strings.IndexByte(body, '|'); idx >= 0 {
			out = append(out, body[idx+1:])
		}
		i = end
	}
	return out
}

// firstTemplateBody renvoie le corps du premier template `name` (nom inclus).
func firstTemplateBody(wikitext, name string) (string, bool) {
	open := "{{" + name
	i := strings.Index(wikitext, open)
	if i < 0 {
		return "", false
	}
	body, end := readBalanced(wikitext, i+2)
	if end < 0 {
		return "", false
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

// splitParams découpe un corps de template sur les '|' de premier niveau, en
// ignorant ceux situés dans des liens [[a|b]], des templates imbriqués {{…}}
// ou des blocs <nowiki>…</nowiki> (pipe littéral, ex. liste FH5).
func splitParams(body string) []string {
	var parts []string
	var cur strings.Builder
	link, tmpl := 0, 0
	nowiki := false
	for i := 0; i < len(body); i++ {
		c := body[i]
		switch {
		case strings.HasPrefix(body[i:], "<nowiki>"):
			nowiki = true
			cur.WriteString("<nowiki>")
			i += len("<nowiki>") - 1
		case strings.HasPrefix(body[i:], "</nowiki>"):
			nowiki = false
			cur.WriteString("</nowiki>")
			i += len("</nowiki>") - 1
		case nowiki:
			cur.WriteByte(c)
		case c == '[' && i+1 < len(body) && body[i+1] == '[':
			link++
			cur.WriteString("[[")
			i++
		case c == ']' && i+1 < len(body) && body[i+1] == ']':
			if link > 0 {
				link--
			}
			cur.WriteString("]]")
			i++
		case c == '{' && i+1 < len(body) && body[i+1] == '{':
			tmpl++
			cur.WriteString("{{")
			i++
		case c == '}' && i+1 < len(body) && body[i+1] == '}':
			if tmpl > 0 {
				tmpl--
			}
			cur.WriteString("}}")
			i++
		case c == '|' && link == 0 && tmpl == 0:
			parts = append(parts, cur.String())
			cur.Reset()
		default:
			cur.WriteByte(c)
		}
	}
	parts = append(parts, cur.String())
	return parts
}

// SortedReport rend les distributions de façon déterministe (pour log/test).
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
