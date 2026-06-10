// Package playlist ingère la série/saison courante de la Festival Playlist Forza
// Horizon depuis des sources PROPRES et officielles (doctrine zéro gris) :
//   - forza.net/events (Nuxt 3, payload JSON SSR) → identité de la série : nom,
//     saison, numéro de série, semaine, dates, et lien vers le post forum ;
//   - forums.forza.net (API JSON Discourse, /t/<id>.json) → le détail défis +
//     récompenses du 1er post (templé par saison).
//
// Jamais le jeu : pas de lecture mémoire, d'injection ni de scraping in-game.
// Le parsing est testé sur fixtures figées (testdata/), sans réseau. Upsert
// idempotent côté store. Aucune donnée inventée : champ absent → nil/NULL ;
// item non templé → ignoré (précision > exhaustivité).
package playlist

import (
	"encoding/json"
	"fmt"
	"html"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/zinackes/forza-open-api/internal/store"
)

// forumData est le détail extrait d'un topic forum (Discourse). Title/Identity
// portent l'identité de la série lue dans le TITRE (source secondaire croisée avec
// forza.net, cf. reconcile.go) ; Season/Saga/Rewards/Challenges viennent du 1er post.
type forumData struct {
	Title      string        // titre brut du topic
	Identity   forumIdentity // identité parsée du titre (jeu, série, semaine, dates)
	Season     string        // ex. « Autumn »
	Saga       string        // nom de la série, ex. « Welcome to Japan »
	Rewards    []parsedReward
	Challenges []parsedChallenge
}

type parsedReward struct {
	Type      string // series | season | event | car_pass | history
	Item      string
	AtPercent *int // ratio points/total pour les paliers, sinon nil
}

type parsedChallenge struct {
	Scope       string // weekly | daily | seasonal
	Name        string
	Requirement string
	Reward      string
}

// discourseTopic est le sous-ensemble utile de /t/<id>.json (1er post cooked).
type discourseTopic struct {
	Title      string `json:"title"`
	PostStream struct {
		Posts []struct {
			Cooked string `json:"cooked"`
		} `json:"posts"`
	} `json:"post_stream"`
}

// ParseForum extrait défis + récompenses du 1er post d'un topic Discourse forza.
func ParseForum(raw []byte) (forumData, error) {
	var t discourseTopic
	if err := json.Unmarshal(raw, &t); err != nil {
		return forumData{}, fmt.Errorf("decode discourse topic: %w", err)
	}
	if len(t.PostStream.Posts) == 0 {
		return forumData{}, fmt.Errorf("discourse topic: aucun post")
	}
	cooked := t.PostStream.Posts[0].Cooked

	fd := forumData{Title: t.Title, Identity: parseForumTitle(t.Title)}
	fd.Season, fd.Saga = parseHeading(cooked)
	fd.Rewards = parseRewards(cooked)
	fd.Challenges = parseChallenges(cooked)
	return fd, nil
}

var (
	reH1      = regexp.MustCompile(`(?is)<h1[^>]*>(.*?)</h1>`)
	reHeading = regexp.MustCompile(`-\s*([^|<]+?)\s*\|\s*(.+?)\s*$`)
)

// parseHeading lit la saison et le nom de saga depuis le H1 du post, du type
// « Festival Playlist events and rewards - Autumn | Welcome to Japan ».
func parseHeading(cooked string) (season, saga string) {
	m := reH1.FindStringSubmatch(cooked)
	if m == nil {
		return "", ""
	}
	h := stripTags(m[1])
	if mm := reHeading.FindStringSubmatch(h); mm != nil {
		return strings.TrimSpace(mm[1]), strings.TrimSpace(mm[2])
	}
	return "", ""
}

var (
	reSeriesTotal = regexp.MustCompile(`(?i)^Series:\s*([\d,]+)\s+(?:points total|total points)`)
	reReward      = regexp.MustCompile(`(?i)^(Series|Season) Reward:\s*(.+?)\s*\((\d+)\s*points\)\s*$`)
	reParen       = regexp.MustCompile(`\s*\([^)]*\)\s*$`)
	reCarYear     = regexp.MustCompile(`^(?:19|20)\d{2}\b`)
)

// parseRewards extrait les récompenses (paliers de points + events individuels +
// Car Pass + History) depuis les lignes texte du post. total suit le « Series: N
// points total » courant pour calculer at_percent des paliers.
func parseRewards(cooked string) []parsedReward {
	lines := htmlToLines(cooked)
	var out []parsedReward
	total := 0
	sub := "" // events | carpass | history

	for _, line := range lines {
		switch {
		case reSeriesTotal.MatchString(line):
			total = atoiComma(reSeriesTotal.FindStringSubmatch(line)[1])
			sub = ""
		case reReward.MatchString(line):
			m := reReward.FindStringSubmatch(line)
			typ := "series"
			if strings.EqualFold(m[1], "Season") {
				typ = "season"
			}
			pts, _ := strconv.Atoi(m[3])
			out = append(out, parsedReward{Type: typ, Item: stripParen(m[2]), AtPercent: pct(pts, total)})
		case strings.Contains(line, "completing individual events below"):
			sub = "events"
		case line == "Car Pass":
			sub = "carpass"
		case strings.Contains(line, "Playlist History Rewards"):
			sub = "history"
		case line == "Collection Journal rewards" || line == "Badges" || line == "Achievements":
			sub = ""
		case sub == "events" && strings.HasPrefix(line, "Car:"):
			if item := stripParen(strings.TrimSpace(strings.TrimPrefix(line, "Car:"))); item != "" {
				out = append(out, parsedReward{Type: "event", Item: item})
			}
		case sub == "events" && strings.HasPrefix(line, "LINK Phrase:"):
			out = append(out, parsedReward{Type: "event", Item: stripParen(strings.TrimSpace(line))})
		case sub == "carpass" && reCarYear.MatchString(line):
			out = append(out, parsedReward{Type: "car_pass", Item: stripParen(line)})
			sub = ""
		case sub == "history" && reCarYear.MatchString(line):
			out = append(out, parsedReward{Type: "history", Item: stripParen(line)})
			sub = ""
		}
	}
	return out
}

var (
	reWeekly  = regexp.MustCompile(`(?i)^Weekly Challenge:\s*(.+?)\s*\((\d+)\s*pts?\)`)
	reDaily   = regexp.MustCompile(`(?i)^Daily Challenges`)
	reCredits = regexp.MustCompile(`([\d,]+)\s*[Cc]redits`)
)

// parseChallenges extrait le défi hebdomadaire, les défis quotidiens (lignes) et
// les événements saisonniers (tables) du post.
func parseChallenges(cooked string) []parsedChallenge {
	var out []parsedChallenge
	lines := htmlToLines(cooked)

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		switch {
		case reWeekly.MatchString(line):
			m := reWeekly.FindStringSubmatch(line)
			var steps []string
			credits := ""
			j := i + 1
			for ; j < len(lines) && !reDaily.MatchString(lines[j]) && !isRewardHeader(lines[j]); j++ {
				if strings.Contains(lines[j], "must be completed in sequence") {
					if c := reCredits.FindStringSubmatch(lines[j]); c != nil {
						credits = "; " + c[1] + " credits"
					}
					continue
				}
				steps = append(steps, lines[j])
			}
			out = append(out, parsedChallenge{
				Scope: "weekly", Name: m[1],
				Requirement: strings.Join(steps, "; "),
				Reward:      m[2] + " pts" + credits,
			})
			i = j - 1
		case reDaily.MatchString(line):
			credits := ""
			j := i + 1
			if j < len(lines) && strings.Contains(lines[j], "Credits for each") {
				if c := reCredits.FindStringSubmatch(lines[j]); c != nil {
					credits = "; " + c[1] + " credits"
				}
				j++
			}
			var seq []string
			for ; j < len(lines) && !isSeasonalHeader(lines[j]) && !isRewardHeader(lines[j]); j++ {
				seq = append(seq, lines[j])
			}
			for k := 0; k+1 < len(seq); k += 2 {
				out = append(out, parsedChallenge{
					Scope: "daily", Name: seq[k], Requirement: seq[k+1],
					Reward: "1 point" + credits,
				})
			}
			i = j - 1
		}
	}

	out = append(out, parseSeasonalTables(cooked)...)
	return out
}

var (
	reTable = regexp.MustCompile(`(?is)<table[^>]*>(.*?)</table>`)
	reRow   = regexp.MustCompile(`(?is)<tr[^>]*>(.*?)</tr>`)
	reCell  = regexp.MustCompile(`(?is)<t[dh][^>]*>(.*?)</t[dh]>`)
)

// parseSeasonalTables transforme les tables « Seasonal Solo and Online Events » /
// « Series Events » en défis saisonniers. Colonnes : Pts, Event Type, Event Name,
// puis Car/Requirement (+ Route). Les lignes d'en-tête (1ʳᵉ cellule « Pts ») sont
// ignorées.
func parseSeasonalTables(cooked string) []parsedChallenge {
	var out []parsedChallenge
	for _, tbl := range reTable.FindAllStringSubmatch(cooked, -1) {
		for _, row := range reRow.FindAllStringSubmatch(tbl[1], -1) {
			var cells []string
			for _, c := range reCell.FindAllStringSubmatch(row[1], -1) {
				cells = append(cells, cleanCell(c[1]))
			}
			if len(cells) < 3 || strings.EqualFold(cells[0], "Pts") {
				continue
			}
			req := cells[1] // Event Type
			if rest := strings.Join(dropEmpty(cells[3:]), " — "); rest != "" {
				req += ": " + rest
			}
			out = append(out, parsedChallenge{
				Scope: "seasonal", Name: cells[2], Requirement: req,
				Reward: cells[0] + " pts",
			})
		}
	}
	return out
}

// Normalize fusionne l'identité (forza.net) et le détail (forum) en lignes store
// prêtes à l'upsert. isCurrent est décidé par l'appelant (la commande) : la série
// découverte comme courante l'est ; un re-seed explicite d'une semaine passée ne
// l'est pas. expires_at : fin de saison pour weekly/seasonal (expirent au
// changement de saison), nil pour daily (prolongés 7 j dans la saison suivante).
func Normalize(ev eventMeta, fd forumData, isCurrent bool) (store.Series, []store.Reward, []store.Challenge) {
	id := fmt.Sprintf("%s-s%02dw%d", ev.Game, ev.Series, ev.Week)

	ser := store.Series{
		ID:        id,
		Game:      ev.Game,
		Series:    ev.Series,
		Name:      strPtr(fd.Saga),
		Season:    strPtr(fd.Season),
		Week:      ev.Week,
		StartsAt:  ev.Start,
		EndsAt:    ev.End,
		IsCurrent: isCurrent,
	}

	rewards := make([]store.Reward, 0, len(fd.Rewards))
	for i, r := range fd.Rewards {
		rewards = append(rewards, store.Reward{
			ID: fmt.Sprintf("%s-rw%02d", id, i+1), SeriesID: id,
			AtPercent: r.AtPercent, Type: r.Type, Item: r.Item,
		})
	}

	challenges := make([]store.Challenge, 0, len(fd.Challenges))
	for i, c := range fd.Challenges {
		var expires *time.Time
		if c.Scope != "daily" && !ev.End.IsZero() {
			e := ev.End
			expires = &e
		}
		challenges = append(challenges, store.Challenge{
			ID: fmt.Sprintf("%s-ch%02d", id, i+1), SeriesID: id,
			Scope: c.Scope, Name: c.Name,
			Requirement: strPtr(c.Requirement), Reward: strPtr(c.Reward),
			ExpiresAt: expires,
		})
	}
	return ser, rewards, challenges
}

// --- helpers texte/HTML ------------------------------------------------------

var (
	reSvg      = regexp.MustCompile(`(?is)<svg.*?</svg>`)
	reMetaDiv  = regexp.MustCompile(`(?is)<div class="meta">.*?</div>`)
	reImg      = regexp.MustCompile(`(?is)<img[^>]*>`)
	reBlockEnd = regexp.MustCompile(`(?is)</(p|li|h1|h2|h3|h4|tr|blockquote)>|<br\s*/?>`)
	reTag      = regexp.MustCompile(`(?s)<[^>]+>`)
	reSpaces   = regexp.MustCompile(`[ \t]+`)
)

// htmlToLines aplatit le cooked HTML (hors bruit images) en lignes texte propres.
func htmlToLines(cooked string) []string {
	s := reSvg.ReplaceAllString(cooked, "")
	s = reMetaDiv.ReplaceAllString(s, "")
	s = reImg.ReplaceAllString(s, "")
	s = reBlockEnd.ReplaceAllString(s, "\n")
	s = stripTags(s)
	s = html.UnescapeString(s)

	var out []string
	for _, l := range strings.Split(s, "\n") {
		l = strings.TrimSpace(reSpaces.ReplaceAllString(l, " "))
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}

func stripTags(s string) string { return reTag.ReplaceAllString(s, "") }

// reWS normalise tout blanc (espaces, tabs, retours d'un <br> de cellule).
var reWS = regexp.MustCompile(`\s+`)

// cleanCell projette une cellule de table HTML en texte propre (tags retirés,
// entités décodées, blancs normalisés).
func cleanCell(s string) string {
	return strings.TrimSpace(reWS.ReplaceAllString(html.UnescapeString(stripTags(s)), " "))
}

// isRewardHeader/isSeasonalHeader bornent les blocs weekly/daily.
func isRewardHeader(l string) bool {
	switch l {
	case "Rewards", "Car Pass", "Playlist History Rewards", "Collection Journal rewards",
		"Badges", "Achievements", "Festival Playlist points system", "Seasonal Challenges and Events":
		return true
	}
	return reSeriesTotal.MatchString(l)
}

func isSeasonalHeader(l string) bool {
	return strings.Contains(l, "Seasonal Solo and Online Events") || l == "Series Events"
}

// pct renvoie le ratio entier points/total (0..100), nil si total inconnu.
func pct(points, total int) *int {
	if total <= 0 {
		return nil
	}
	v := int(math.Round(float64(points*100) / float64(total)))
	return &v
}

func stripParen(s string) string { return strings.TrimSpace(reParen.ReplaceAllString(s, "")) }

func atoiComma(s string) int {
	n, _ := strconv.Atoi(strings.ReplaceAll(s, ",", ""))
	return n
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func dropEmpty(ss []string) []string {
	out := ss[:0:0]
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}
