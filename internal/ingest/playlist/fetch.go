package playlist

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// userAgent identifie l'ingestion auprès des sources (politesse exigée par la
// doctrine : user-agent identifiable, lecture seule).
const userAgent = "forza-open-api-ingestion/0.1 (+https://github.com/zinackes/forza-open-api)"

const (
	forzaBase = "https://forza.net"
	forumBase = "https://forums.forza.net"
	// politeDelay espace les requêtes entre les deux sources (pas de hammering).
	politeDelay = 500 * time.Millisecond
)

var httpc = &http.Client{Timeout: 25 * time.Second}

// FetchEventPayload récupère le payload SSR Nuxt d'une page event forza.net
// (/events/<slug>/_payload.json). Lecture seule, user-agent identifiable.
func FetchEventPayload(ctx context.Context, slug string) ([]byte, error) {
	return getBytes(ctx, forzaBase+"/events/"+slug+"/_payload.json")
}

// FetchEventsIndex récupère le payload de l'index /events (liste des events
// courants), pour découvrir la série courante sans connaître son slug.
func FetchEventsIndex(ctx context.Context) ([]byte, error) {
	return getBytes(ctx, forzaBase+"/events/_payload.json")
}

// FetchForumTopic récupère un topic forum via l'API JSON Discourse (/t/<id>.json).
func FetchForumTopic(ctx context.Context, topicID string) ([]byte, error) {
	return getBytes(ctx, forumBase+"/t/"+topicID+".json")
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
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, fmt.Errorf("read body %s: %w", url, err)
	}
	return raw, nil
}

// reTopicID capte l'identifiant numérique de topic en fin d'URL forum Discourse.
var reTopicID = regexp.MustCompile(`/(\d+)/?$`)

// FetchCurrent récupère la série visée pour game (eventMeta + détail forum), prête
// pour Normalize. override (« S01W2 », vide = auto) cible une semaine précise ;
// sinon la série courante est découverte depuis l'index /events. Deux sources →
// un délai poli entre les deux requêtes réseau.
func FetchCurrent(ctx context.Context, game, override string, now time.Time) (eventMeta, forumData, error) {
	var ev eventMeta
	if override != "" {
		slug := fmt.Sprintf("%s-Festival-Playlist-%s", strings.ToUpper(game), strings.ToUpper(override))
		raw, err := FetchEventPayload(ctx, slug)
		if err != nil {
			return eventMeta{}, forumData{}, err
		}
		events, err := extractEvents(raw)
		if err != nil {
			return eventMeta{}, forumData{}, err
		}
		ev, err = pickSlug(events, slug)
		if err != nil {
			return eventMeta{}, forumData{}, err
		}
	} else {
		raw, err := FetchEventsIndex(ctx)
		if err != nil {
			return eventMeta{}, forumData{}, err
		}
		events, err := extractEvents(raw)
		if err != nil {
			return eventMeta{}, forumData{}, err
		}
		ev, err = pickCurrent(events, game, now)
		if err != nil {
			return eventMeta{}, forumData{}, err
		}
	}

	if ev.ForumURL == "" {
		return eventMeta{}, forumData{}, fmt.Errorf("event %s: pas de lien forum", ev.Slug)
	}
	m := reTopicID.FindStringSubmatch(strings.TrimSuffix(ev.ForumURL, ".json"))
	if m == nil {
		return eventMeta{}, forumData{}, fmt.Errorf("event %s: id topic introuvable dans %q", ev.Slug, ev.ForumURL)
	}

	if err := sleep(ctx, politeDelay); err != nil {
		return eventMeta{}, forumData{}, err
	}

	rawForum, err := FetchForumTopic(ctx, m[1])
	if err != nil {
		return eventMeta{}, forumData{}, err
	}
	fd, err := ParseForum(rawForum)
	if err != nil {
		return eventMeta{}, forumData{}, err
	}
	return ev, fd, nil
}

// pickSlug retourne l'event au slug demandé (résolution insensible à la casse).
func pickSlug(events []eventMeta, slug string) (eventMeta, error) {
	for _, e := range events {
		if strings.EqualFold(e.Slug, slug) {
			return e, nil
		}
	}
	return eventMeta{}, fmt.Errorf("slug %q absent du payload", slug)
}

// pickCurrent choisit la série courante d'un jeu : la dernière démarrée à ce jour
// (start ≤ now). À défaut (aucune démarrée), la plus proche à venir.
func pickCurrent(events []eventMeta, game string, now time.Time) (eventMeta, error) {
	var best, upcoming *eventMeta
	for i := range events {
		e := events[i]
		if e.Game != game || e.Start.IsZero() {
			continue
		}
		if e.Start.After(now) {
			if upcoming == nil || e.Start.Before(upcoming.Start) {
				upcoming = &events[i]
			}
			continue
		}
		if best == nil || e.Start.After(best.Start) {
			best = &events[i]
		}
	}
	switch {
	case best != nil:
		return *best, nil
	case upcoming != nil:
		return *upcoming, nil
	default:
		return eventMeta{}, fmt.Errorf("aucune série Festival Playlist trouvée pour %q", game)
	}
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
