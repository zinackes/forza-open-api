package manufacturers

// Couche réseau de l'ingestion des constructeurs. Source PROPRE, lecture seule,
// user-agent identifiable, délais polis (doctrine zéro gris — jamais le jeu) :
// l'API MediaWiki du wiki Fandom Forza (forza.fandom.com), même canal stable que
// les autres ingestions (action=query, pas de scrape HTML).
//
// Deux requêtes alimentent une passe :
//   - list=categorymembers de « Category:Manufacturers (<TAG>) » : le roster du jeu
//     (TAG = FH6, FH5…). C'est la sous-catégorie PAR JEU du wiki (la catégorie plate
//     « Manufacturers » est cross-jeux : l'attribuer à un jeu serait inventer un lien) ;
//   - prop=revisions par lots multi-titres : le wikitext de chaque page constructeur,
//     dont on extrait l'{{InfoboxMFR}} (origin → country).

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

const (
	wikiEndpoint = "https://forza.fandom.com/api.php"
	// pageBatch borne le nombre de titres par requête revisions (limite MediaWiki
	// pour un client non-bot). politeDelay espace les lots (délais polis).
	pageBatch   = 50
	politeDelay = 400 * time.Millisecond
)

var httpc = &http.Client{Timeout: 25 * time.Second}

// gameWikiTag mappe un jeu vers le suffixe de sa sous-catégorie constructeurs du
// wiki Fandom (« Category:Manufacturers (FH6) », etc.).
var gameWikiTag = map[string]string{
	"fh6": "FH6",
	"fh5": "FH5",
}

// categoryTitle construit le titre de la sous-catégorie constructeurs d'un jeu.
// Renvoie une erreur si le jeu n'a pas de mapping wiki.
func categoryTitle(game string) (string, error) {
	tag, ok := gameWikiTag[strings.ToLower(game)]
	if !ok {
		return "", fmt.Errorf("jeu %q sans mapping wiki", game)
	}
	return "Category:Manufacturers (" + tag + ")", nil
}

// fetchCategoryMembers liste les titres des pages constructeurs du jeu (membres de
// la sous-catégorie). found=false si la catégorie est absente (jeu non documenté :
// soft-skip, pas une erreur dure). Suit la pagination MediaWiki (cmcontinue).
func fetchCategoryMembers(ctx context.Context, game string) (titles []string, found bool, err error) {
	cat, err := categoryTitle(game)
	if err != nil {
		return nil, false, err
	}
	cmcontinue := ""
	for {
		q := url.Values{
			"action": {"query"}, "list": {"categorymembers"},
			"cmtitle": {cat}, "cmlimit": {"500"}, "cmtype": {"page"},
			"format": {"json"}, "formatversion": {"2"},
		}
		if cmcontinue != "" {
			q.Set("cmcontinue", cmcontinue)
		}
		var resp struct {
			Query struct {
				CategoryMembers []struct {
					Title string `json:"title"`
				} `json:"categorymembers"`
			} `json:"query"`
			Continue struct {
				CmContinue string `json:"cmcontinue"`
			} `json:"continue"`
		}
		if err := getJSON(ctx, wikiEndpoint+"?"+q.Encode(), &resp); err != nil {
			return nil, false, err
		}
		for _, m := range resp.Query.CategoryMembers {
			titles = append(titles, m.Title)
		}
		if resp.Continue.CmContinue == "" {
			break
		}
		cmcontinue = resp.Continue.CmContinue
		if err := sleep(ctx, politeDelay); err != nil {
			return nil, false, err
		}
	}
	return titles, len(titles) > 0, nil
}

// fetchManufacturerPages récupère le wikitext des pages constructeurs par lots
// multi-titres. La clé de la map est le titre renvoyé par MediaWiki (= titre
// canonique du membre de catégorie). Une page manquante est simplement absente.
func fetchManufacturerPages(ctx context.Context, titles []string) (map[string]string, error) {
	out := make(map[string]string, len(titles))
	for start := 0; start < len(titles); start += pageBatch {
		end := start + pageBatch
		if end > len(titles) {
			end = len(titles)
		}
		if err := fetchPageBatch(ctx, titles[start:end], out); err != nil {
			return nil, err
		}
		if end < len(titles) {
			if err := sleep(ctx, politeDelay); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

// fetchPageBatch récupère un lot de pages et écrit chaque (titre → wikitext) dans dst.
func fetchPageBatch(ctx context.Context, titles []string, dst map[string]string) error {
	q := url.Values{
		"action": {"query"}, "prop": {"revisions"}, "rvprop": {"content"},
		"rvslots": {"main"}, "redirects": {"1"}, "format": {"json"},
		"formatversion": {"2"}, "titles": {strings.Join(titles, "|")},
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
	if err := getJSON(ctx, wikiEndpoint+"?"+q.Encode(), &resp); err != nil {
		return err
	}
	for _, p := range resp.Query.Pages {
		if p.Missing || len(p.Revisions) == 0 {
			continue
		}
		dst[p.Title] = p.Revisions[0].Slots.Main.Content
	}
	return nil
}

// getJSON exécute un GET MediaWiki (timeout, user-agent, taille bornée) et décode
// le corps dans v.
func getJSON(ctx context.Context, rawURL string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
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

// sleep attend d (ou rend ctx.Err() si annulé) — délai poli interruptible.
func sleep(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}
