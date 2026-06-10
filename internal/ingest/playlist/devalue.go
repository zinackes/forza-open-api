package playlist

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Le payload SSR de forza.net (Nuxt 3) est encodé en « devalue » : un tableau
// JSON plat où, dans tout objet/tableau, un entier est un INDICE de référence
// vers une autre case du tableau (les scalaires sont eux aussi des cases). On
// résout les seules sous-arbres dont on a besoin (les objets « event »).

// resolver déréférence un payload devalue. seen casse les cycles (refs Vue).
type resolver struct {
	arr  []any
	seen map[int]struct{}
}

func newResolver(arr []any) *resolver {
	return &resolver{arr: arr, seen: map[int]struct{}{}}
}

// resolve déréférence la case idx en valeur concrète (string/float64/bool/nil,
// map ou slice). Hors borne ou sentinelle négative (undefined/hole/NaN…) → nil.
func (r *resolver) resolve(idx int) any {
	if idx < 0 || idx >= len(r.arr) {
		return nil
	}
	if _, ok := r.seen[idx]; ok {
		return nil
	}
	r.seen[idx] = struct{}{}
	defer delete(r.seen, idx)

	switch v := r.arr[idx].(type) {
	case []any:
		// Marqueur typé (["Reactive",N], ["Ref",N], …) : 1er élément string.
		if len(v) > 0 {
			if tag, ok := v[0].(string); ok {
				switch tag {
				case "Reactive", "ShallowReactive", "Ref", "ShallowRef", "EmptyRef", "Date":
					if len(v) > 1 {
						return r.resolve(toInt(v[1]))
					}
					return nil
				default:
					return nil // Set/Map/RegExp… non nécessaires ici
				}
			}
		}
		out := make([]any, len(v)) // tableau de données : éléments = indices
		for i, e := range v {
			out[i] = r.deref(e)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, val := range v {
			out[k] = r.deref(val)
		}
		return out
	default:
		return v // scalaire déjà concret
	}
}

// deref résout une valeur de propriété : un nombre est un indice, le reste est
// déjà concret (cas rares de scalaires inline).
func (r *resolver) deref(val any) any {
	if f, ok := val.(float64); ok {
		return r.resolve(int(f))
	}
	return val
}

func toInt(v any) int {
	if f, ok := v.(float64); ok {
		return int(f)
	}
	return -1
}

// eventMeta est l'identité d'un event de la Festival Playlist extraite du payload
// forza.net. Game/Series/Week sont dérivés du slug (le heading peut être périmé,
// ex. libellé « FH5 » sur un slug FH6).
type eventMeta struct {
	Slug     string
	Heading  string
	Excerpt  string
	Game     string
	Series   int
	Week     int
	Start    time.Time
	End      time.Time
	Category string
	ForumURL string
}

// slugRe capte le jeu, le numéro de série et la semaine d'un slug d'event
// « FH6-Festival-Playlist-S01W2 » (insensible à la casse).
var slugRe = regexp.MustCompile(`(?i)^(FH\d)-Festival-Playlist-S(\d+)W(\d+)$`)

// extractEvents déréférence tous les objets « event » Festival Playlist présents
// dans un payload forza.net (page event ou index /events). Un payload peut en
// contenir plusieurs (event courant + « more events ») : le caller choisit.
func extractEvents(raw []byte) ([]eventMeta, error) {
	var arr []any
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, fmt.Errorf("decode nuxt payload: %w", err)
	}

	out := make([]eventMeta, 0, 4)
	seenSlug := map[string]bool{}
	for idx, entry := range arr {
		m, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		// Candidat event : un objet Strapi qui porte slug + start + end.
		if _, ok := m["slug"]; !ok {
			continue
		}
		if _, ok := m["start"]; !ok {
			continue
		}
		if _, ok := m["end"]; !ok {
			continue
		}
		resolved, ok := newResolver(arr).resolve(idx).(map[string]any)
		if !ok {
			continue
		}
		ev, ok := toEventMeta(resolved)
		if !ok || seenSlug[ev.Slug] {
			continue
		}
		seenSlug[ev.Slug] = true
		out = append(out, ev)
	}
	return out, nil
}

// toEventMeta projette un objet event résolu en eventMeta. Renvoie false si le
// slug n'est pas un slug Festival Playlist exploitable (game/série/semaine).
func toEventMeta(m map[string]any) (eventMeta, bool) {
	slug := str(m["slug"])
	mm := slugRe.FindStringSubmatch(slug)
	if mm == nil {
		return eventMeta{}, false
	}
	series, _ := strconv.Atoi(mm[2])
	week, _ := strconv.Atoi(mm[3])

	ev := eventMeta{
		Slug:     slug,
		Heading:  str(m["heading"]),
		Excerpt:  str(m["excerpt"]),
		Game:     strings.ToLower(mm[1]),
		Series:   series,
		Week:     week,
		Start:    parseTime(str(m["start"])),
		End:      parseTime(str(m["end"])),
		Category: nestedStr(m["category"], "name"),
		ForumURL: nestedStr(m["externalLink"], "url"),
	}
	return ev, true
}

func str(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// nestedStr lit m[key] sur une valeur supposée map (sinon "").
func nestedStr(v any, key string) string {
	if m, ok := v.(map[string]any); ok {
		return str(m[key])
	}
	return ""
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}
