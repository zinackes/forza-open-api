package forzathon

// Couche réseau de l'ingestion Forzathon Shop. Sources PROPRES, lecture seule,
// user-agent identifiable, délais polis (doctrine zéro gris — jamais le jeu) :
//   - wiki Fandom (API MediaWiki) : SOURCE PRIMAIRE structurée des objets de la
//     rotation (page « <jeu> Forzathon Shop », table wikitext) — même canal stable
//     que le backfill playlist (action=query prop=revisions, pas de scrape HTML) ;
//   - forza.net + forums : CORROBORATION best-effort de la fraîcheur (la rotation
//     est-elle bien annoncée cette semaine), non bloquante.
//
// NB : le titre de page wiki et les chemins forza.net/forum sont les structures
// supposées des sources ; un écart de structure (page absente, table vide) est
// rattrapé par CheckRun (anomalie → alerte), conforme à la méthode du repo
// (parser figé sur fixtures + validation en prod par le monitoring santé).

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

// userAgent identifie l'ingestion auprès des sources (politesse exigée par la
// doctrine : user-agent identifiable, lecture seule).
const userAgent = "forza-open-api-ingestion/0.1 (+https://github.com/zinackes/forza-open-api)"

const (
	wikiEndpoint = "https://forza.fandom.com/api.php"
	forzaBase    = "https://forza.net"
	forumBase    = "https://forums.forza.net"
	// politeDelay espace les requêtes entre sources (pas de hammering).
	politeDelay = 500 * time.Millisecond
)

var httpc = &http.Client{Timeout: 25 * time.Second}

// gameWiki mappe un jeu vers le préfixe de page du wiki Fandom (cf. ingestion
// playlist : « Forza Horizon 6 »). La page Forzathon Shop est « <préfixe>
// Forzathon Shop ».
var gameWiki = map[string]string{
	"fh6": "Forza Horizon 6",
	"fh5": "Forza Horizon 5",
}

// FetchWikiShop récupère le wikitext de la page Forzathon Shop du jeu via l'API
// MediaWiki (slot main). Renvoie une erreur si le jeu est inconnu ou la page
// absente (parse → CheckRun signalera l'anomalie).
func FetchWikiShop(ctx context.Context, game string) (string, error) {
	prefix, ok := gameWiki[strings.ToLower(game)]
	if !ok {
		return "", fmt.Errorf("jeu %q sans mapping wiki", game)
	}
	title := prefix + " Forzathon Shop"
	q := url.Values{
		"action": {"query"}, "prop": {"revisions"}, "rvprop": {"content"},
		"rvslots": {"main"}, "format": {"json"}, "formatversion": {"2"},
		"titles": {title},
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
		return "", err
	}
	for _, p := range resp.Query.Pages {
		if p.Missing || len(p.Revisions) == 0 {
			continue
		}
		return p.Revisions[0].Slots.Main.Content, nil
	}
	return "", fmt.Errorf("page wiki %q absente ou sans révision", title)
}

// corroborate consulte forza.net + forums (best-effort, non bloquant) pour
// confirmer qu'une rotation Forzathon Shop est bien référencée. Renvoie la liste
// des sources qui l'ont confirmée (« forza.net », « forums.forza.net »). Toute
// erreur réseau est loggée et avalée : le wiki reste la source autoritaire.
func corroborate(ctx context.Context, logf func(msg string, kv ...any)) []string {
	var ok []string
	check := func(name, url string) {
		raw, err := getBytes(ctx, url)
		if err != nil {
			if logf != nil {
				logf("corroboration forzathon indisponible", "source", name, "err", err)
			}
			return
		}
		if strings.Contains(strings.ToLower(string(raw)), "forzathon shop") {
			ok = append(ok, name)
		}
	}
	check("forza.net", forzaBase+"/news/_payload.json")
	_ = sleep(ctx, politeDelay)
	check("forums.forza.net", forumBase+"/c/discussion/forzathon/.json")
	return ok
}

// getJSON exécute un GET (timeout, user-agent, taille bornée) et décode le JSON.
func getJSON(ctx context.Context, url string, v any) error {
	raw, err := getBytes(ctx, url)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, v); err != nil {
		return fmt.Errorf("decode json %s: %w", url, err)
	}
	return nil
}

// getBytes exécute un GET (timeout, user-agent, taille bornée) et renvoie le corps.
func getBytes(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := httpc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: status %d", url, resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 24<<20))
	if err != nil {
		return nil, fmt.Errorf("read body %s: %w", url, err)
	}
	return raw, nil
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
