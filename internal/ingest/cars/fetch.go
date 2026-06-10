// Package cars ingère le catalogue voitures depuis une source PROPRE : l'API
// MediaWiki du wiki Fandom Forza (forza.fandom.com). Jamais le jeu : pas de
// scraping in-game, lecture mémoire ni injection (doctrine zéro gris).
//
// Deux pages alimentent une voiture :
//   - « <game>/Cars » : une ligne {{CarListStats<jeu>|…}} par voiture (PI, stats,
//     valeur, rareté, méthode d'obtention, pays). Mêmes positions FH5/FH6 ;
//     codes d'obtention propres à chaque jeu (cf. obtainNames).
//   - la page voiture (CarInfobox) : layout → drivetrain, manufacturer/model,
//     type → body_type. Récupérée par lots (multi-titres) pour limiter le réseau.
//
// Le parsing (parse.go) est testé sur fixtures figées (testdata/), sans réseau.
// Upsert idempotent côté store. Aucune donnée inventée : champ absent → NULL ;
// voiture sans champ requis par le contrat (class/PI/drivetrain) → ignorée.
package cars

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// userAgent identifie l'ingestion auprès de la source (politesse exigée par la
// doctrine : user-agent identifiable, lecture seule).
const userAgent = "forza-open-api-ingestion/0.1 (+https://github.com/zinackes/forza-open-api)"

// apiEndpoint est l'API MediaWiki du wiki Fandom Forza.
const apiEndpoint = "https://forza.fandom.com/api.php"

// pageBatch borne le nombre de titres par requête multi-titres (limite MediaWiki
// pour un client non-bot). politeDelay espace les lots (délais polis).
const (
	pageBatch   = 50
	politeDelay = 300 * time.Millisecond
)

var httpc = &http.Client{Timeout: 30 * time.Second}

// FetchCarListWikitext récupère le wikitext de la page liste « <listPage> » (ex.
// « Forza Horizon 6/Cars ») via action=parse. Lecture seule, user-agent identifiable.
func FetchCarListWikitext(ctx context.Context, listPage string) (string, error) {
	q := url.Values{
		"action": {"parse"}, "page": {listPage}, "prop": {"wikitext"},
		"format": {"json"}, "formatversion": {"1"},
	}
	var out struct {
		Parse struct {
			Wikitext struct {
				Star string `json:"*"`
			} `json:"wikitext"`
		} `json:"parse"`
		Error *struct {
			Info string `json:"info"`
		} `json:"error"`
	}
	if err := getJSON(ctx, q, &out); err != nil {
		return "", err
	}
	if out.Error != nil {
		return "", fmt.Errorf("mediawiki parse %q: %s", listPage, out.Error.Info)
	}
	if out.Parse.Wikitext.Star == "" {
		return "", fmt.Errorf("mediawiki parse %q: empty wikitext", listPage)
	}
	return out.Parse.Wikitext.Star, nil
}

// FetchCarPages récupère le wikitext des pages voitures par lots multi-titres
// (prop=revisions, redirects=1). La clé de la map renvoyée est le titre DEMANDÉ
// (résolution interne des normalisations de casse et des redirections), ce qui
// permet au caller de retrouver l'infobox à partir du nom de la ligne liste.
// Une page manquante est simplement absente de la map (→ drivetrain NULL → ignorée).
func FetchCarPages(ctx context.Context, titles []string) (map[string]string, error) {
	out := make(map[string]string, len(titles))
	for start := 0; start < len(titles); start += pageBatch {
		end := start + pageBatch
		if end > len(titles) {
			end = len(titles)
		}
		batch := titles[start:end]
		if err := fetchPageBatch(ctx, batch, out); err != nil {
			return nil, err
		}
		if end < len(titles) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(politeDelay):
			}
		}
	}
	return out, nil
}

// mwBatch est la réponse query+revisions, incluant les tables de résolution de
// titres (normalisation de casse, redirections) que MediaWiki applique.
type mwBatch struct {
	Query struct {
		Normalized []struct{ From, To string } `json:"normalized"`
		Redirects  []struct{ From, To string } `json:"redirects"`
		Pages      map[string]struct {
			Title     string `json:"title"`
			Revisions []struct {
				Slots struct {
					Main struct {
						Star string `json:"*"`
					} `json:"main"`
				} `json:"slots"`
			} `json:"revisions"`
		} `json:"pages"`
	} `json:"query"`
}

// fetchPageBatch récupère un lot et écrit chaque (titre demandé → wikitext) dans dst.
func fetchPageBatch(ctx context.Context, titles []string, dst map[string]string) error {
	q := url.Values{
		"action": {"query"}, "prop": {"revisions"}, "rvprop": {"content"},
		"rvslots": {"main"}, "redirects": {"1"}, "format": {"json"},
		"formatversion": {"1"}, "titles": {strings.Join(titles, "|")},
	}
	var resp mwBatch
	if err := getJSON(ctx, q, &resp); err != nil {
		return err
	}

	// Index titre-final → wikitext.
	byTitle := make(map[string]string, len(resp.Query.Pages))
	for _, p := range resp.Query.Pages {
		if len(p.Revisions) == 0 {
			continue
		}
		byTitle[p.Title] = p.Revisions[0].Slots.Main.Star
	}
	// Tables de résolution : demandé → normalisé → cible de redirection.
	norm := make(map[string]string, len(resp.Query.Normalized))
	for _, n := range resp.Query.Normalized {
		norm[n.From] = n.To
	}
	redir := make(map[string]string, len(resp.Query.Redirects))
	for _, r := range resp.Query.Redirects {
		redir[r.From] = r.To
	}
	for _, want := range titles {
		final := want
		if v, ok := norm[final]; ok {
			final = v
		}
		// Suivre la chaîne de redirections (bornée pour éviter une boucle).
		for i := 0; i < 4; i++ {
			v, ok := redir[final]
			if !ok {
				break
			}
			final = v
		}
		if wt, ok := byTitle[final]; ok {
			dst[want] = wt
		}
	}
	return nil
}

// getJSON exécute un GET MediaWiki (timeout, user-agent, taille bornée) et décode
// le corps dans v.
func getJSON(ctx context.Context, q url.Values, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiEndpoint+"?"+q.Encode(), nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := httpc.Do(req)
	if err != nil {
		return fmt.Errorf("mediawiki request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("mediawiki request: status %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 24<<20))
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	if err := json.Unmarshal(raw, v); err != nil {
		return fmt.Errorf("decode mediawiki json: %w", err)
	}
	return nil
}
