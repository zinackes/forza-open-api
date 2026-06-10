package playlist

// Backfill historique de la Festival Playlist depuis l'API MediaWiki du wiki
// Fandom Forza (forza.fandom.com/api.php, JSON) — source PROPRE, lecture seule,
// jamais le jeu (doctrine zéro gris). Pas de scrape HTML : on consomme l'API
// (action=query allpages pour découvrir les séries, prop=revisions par lots pour
// le wikitext). Le parsing (wiki_parse.go) tourne sur fixtures figées, sans réseau.
//
// Structure du wiki (identique FH5/FH6, préfixe template FH5/FH6) :
//   - « <jeu>/Series N » : infobox identité (saga, dates, totaux/saison) + paliers
//     série {{FH*FPEvent|seriescar|pts|{{FH*FPReward}}}} + 4 transclusions saison ;
//   - « <jeu>/Series N/<Season> Season » : infobox saison + paliers saison + défis
//     (weekly/daily) + events (seasonal/online/monthly), en templates FH*FPEvent.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

// wikiEndpoint est l'API MediaWiki du wiki Fandom Forza (mutualisé avec l'ingestion
// catalogue voitures, autre source du même wiki).
const wikiEndpoint = "https://forza.fandom.com/api.php"

// pageBatch borne le nombre de titres par requête multi-titres (limite MediaWiki
// pour un client non-bot).
const pageBatch = 50

// seriesPageset regroupe, pour une série, sa page d'identité et ses sous-pages
// saison effectivement présentes sur le wiki (Spring peut manquer si pas encore
// publiée). season ∈ {summer, autumn, winter, spring}.
type seriesPageset struct {
	series      int
	seriesTitle string
	seasonTitle map[string]string
}

// discoverSeries énumère les pages « <jeu>/Series N » et leurs sous-pages saison
// via list=allpages (un seul appel borné à 500, continuation gérée par sûreté),
// puis les classe par numéro de série (croissant).
func discoverSeries(ctx context.Context, g wikiGame) ([]seriesPageset, error) {
	titles, err := allPagesWithPrefix(ctx, g.titlePrefix+"/Series ")
	if err != nil {
		return nil, err
	}

	quoted := regexp.QuoteMeta(g.titlePrefix)
	reSeriesPage := regexp.MustCompile(`^` + quoted + `/Series (\d+)$`)
	reSeasonPage := regexp.MustCompile(`(?i)^` + quoted + `/Series (\d+)/(Summer|Autumn|Winter|Spring) Season$`)

	bySeries := map[int]*seriesPageset{}
	ensure := func(n int) *seriesPageset {
		ps, ok := bySeries[n]
		if !ok {
			ps = &seriesPageset{series: n, seasonTitle: map[string]string{}}
			bySeries[n] = ps
		}
		return ps
	}
	for _, t := range titles {
		if m := reSeriesPage.FindStringSubmatch(t); m != nil {
			ensure(atoi(m[1])).seriesTitle = t
		} else if m := reSeasonPage.FindStringSubmatch(t); m != nil {
			ensure(atoi(m[1])).seasonTitle[strings.ToLower(m[2])] = t
		}
	}

	out := make([]seriesPageset, 0, len(bySeries))
	for _, ps := range bySeries {
		if ps.seriesTitle == "" {
			continue // sous-pages sans page série parente : on ignore
		}
		out = append(out, *ps)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].series < out[j].series })
	return out, nil
}

// allPagesWithPrefix renvoie tous les titres (namespace 0) sous un préfixe,
// continuation suivie au besoin (délai poli entre pages).
func allPagesWithPrefix(ctx context.Context, prefix string) ([]string, error) {
	var out []string
	apcontinue := ""
	for {
		q := url.Values{
			"action": {"query"}, "list": {"allpages"}, "apnamespace": {"0"},
			"apprefix": {prefix}, "aplimit": {"500"},
			"format": {"json"}, "formatversion": {"2"},
		}
		if apcontinue != "" {
			q.Set("apcontinue", apcontinue)
		}
		var resp struct {
			Query struct {
				Allpages []struct {
					Title string `json:"title"`
				} `json:"allpages"`
			} `json:"query"`
			Continue struct {
				Apcontinue string `json:"apcontinue"`
			} `json:"continue"`
		}
		if err := wikiGetJSON(ctx, q, &resp); err != nil {
			return nil, err
		}
		for _, p := range resp.Query.Allpages {
			out = append(out, p.Title)
		}
		if resp.Continue.Apcontinue == "" {
			return out, nil
		}
		apcontinue = resp.Continue.Apcontinue
		if err := sleep(ctx, politeDelay); err != nil {
			return nil, err
		}
	}
}

// fetchWikitexts récupère le wikitext (slot main) de chaque titre par lots de
// pageBatch (prop=revisions). Une page absente est simplement omise de la map.
func fetchWikitexts(ctx context.Context, titles []string) (map[string]string, error) {
	out := make(map[string]string, len(titles))
	for start := 0; start < len(titles); start += pageBatch {
		end := min(start+pageBatch, len(titles))
		q := url.Values{
			"action": {"query"}, "prop": {"revisions"}, "rvprop": {"content"},
			"rvslots": {"main"}, "format": {"json"}, "formatversion": {"2"},
			"titles": {strings.Join(titles[start:end], "|")},
		}
		var resp struct {
			Query struct {
				Pages []struct {
					Title     string `json:"title"`
					Missing   bool   `json:"missing"`
					Revisions []struct {
						Slots struct {
							Main struct {
								Content string `json:"content"`
							} `json:"main"`
						} `json:"slots"`
					} `json:"revisions"`
				} `json:"pages"`
			} `json:"query"`
		}
		if err := wikiGetJSON(ctx, q, &resp); err != nil {
			return nil, err
		}
		for _, p := range resp.Query.Pages {
			if p.Missing || len(p.Revisions) == 0 {
				continue
			}
			out[p.Title] = p.Revisions[0].Slots.Main.Content
		}
		if end < len(titles) {
			if err := sleep(ctx, politeDelay); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

// wikiGetJSON exécute un GET MediaWiki (timeout, user-agent identifiable, taille
// bornée) et décode le corps JSON dans v. Réutilise httpc/userAgent (fetch.go).
func wikiGetJSON(ctx context.Context, q url.Values, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wikiEndpoint+"?"+q.Encode(), nil)
	if err != nil {
		return fmt.Errorf("build wiki request: %w", err)
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
		return fmt.Errorf("read wiki body: %w", err)
	}
	if err := json.Unmarshal(raw, v); err != nil {
		return fmt.Errorf("decode mediawiki json: %w", err)
	}
	return nil
}
