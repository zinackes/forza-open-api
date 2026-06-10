// Dérivation de la table manufacturers depuis la page liste du wiki : chaque
// ligne {{CarListStats…}} porte un code pays (champ 14) et le make se dérive
// comme dans Normalize (nom complet − model de l'infobox). Aucune donnée
// inventée : le code pays est stocké TEL QUE sourcé (codes wiki minuscules :
// ita, jpn, ger, usa…) ; absent → NULL.
package cars

import (
	"sort"
	"strings"

	"github.com/zinackes/forza-open-api/internal/store"
)

// Manufacturers dérive les constructeurs (make → code pays) des lignes liste.
// Mêmes conditions de make que Normalize (ligne sans infobox écartée, pour que
// les noms collent exactement aux make du catalogue). Quand un même make porte
// plusieurs codes pays (rare : incohérence wiki), le plus fréquent gagne ;
// l'absence de code → Country nil. Résultat trié par nom (déterministe).
func Manufacturers(rows []ListRow, infoboxes map[string]Infobox, game string) []store.Manufacturer {
	// make → code pays → occurrences.
	counts := make(map[string]map[string]int)
	for _, r := range rows {
		ib, ok := infoboxes[r.Name]
		if !ok {
			continue
		}
		make := deriveMake(r.Name, ib.Model)
		if make == "" {
			continue
		}
		if counts[make] == nil {
			counts[make] = map[string]int{}
		}
		if code := strings.ToLower(strings.TrimSpace(r.Country)); code != "" {
			counts[make][code]++
		}
	}

	out := make([]store.Manufacturer, 0, len(counts))
	for make, codes := range counts {
		var country *string
		best := 0
		for code, n := range codes {
			// Majorité ; à égalité, l'ordre lexical départage (déterminisme).
			if n > best || (n == best && country != nil && code < *country) {
				c := code
				country, best = &c, n
			}
		}
		out = append(out, store.Manufacturer{Game: game, Name: make, Country: country})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
