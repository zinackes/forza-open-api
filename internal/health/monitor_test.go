package health_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/zinackes/forza-open-api/internal/health"
	"github.com/zinackes/forza-open-api/internal/store"
)

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// fakeStore implémente health.RunStore en mémoire (pas de Docker requis).
type fakeStore struct {
	mu   sync.Mutex
	runs []store.ScrapeRun
}

func (f *fakeStore) InsertScrapeRun(_ context.Context, r store.ScrapeRun) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.runs = append(f.runs, r)
	return nil
}

func (f *fakeStore) last() store.ScrapeRun {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.runs) == 0 {
		return store.ScrapeRun{}
	}
	return f.runs[len(f.runs)-1]
}

// webhook capture les alertes POSTées (concurrence-safe).
type webhook struct {
	mu   sync.Mutex
	hits []health.Alert
	srv  *httptest.Server
}

func newWebhook() *webhook {
	w := &webhook{}
	w.srv = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		var a health.Alert
		_ = json.NewDecoder(r.Body).Decode(&a)
		w.mu.Lock()
		w.hits = append(w.hits, a)
		w.mu.Unlock()
		rw.WriteHeader(http.StatusNoContent)
	}))
	return w
}

func (w *webhook) all() []health.Alert {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]health.Alert(nil), w.hits...)
}

// TestObserveAnomalyAlertsAndPersists : un Report avec violation anomalie déclenche
// l'alerte (status=anomaly, violations transmises) ET persiste le run.
func TestObserveAnomalyAlertsAndPersists(t *testing.T) {
	wh := newWebhook()
	defer wh.srv.Close()
	fs := &fakeStore{}
	m := &health.Monitor{Store: fs, Notifier: health.NewNotifier(wh.srv.URL), Logger: discardLogger()}

	rep := health.Report{
		Source: "playlist", Game: "fh6", Records: 1,
		Violations: []health.Violation{{Rule: "no_rewards", Detail: "0 récompense", Severity: health.SevAnomaly}},
	}
	m.Observe(context.Background(), rep, time.Now())

	alerts := wh.all()
	if len(alerts) != 1 {
		t.Fatalf("alertes = %d, want 1", len(alerts))
	}
	a := alerts[0]
	if a.Status != health.StatusAnomaly || a.Source != "playlist" || a.Game != "fh6" {
		t.Errorf("alerte = %+v, want status=anomaly source=playlist game=fh6", a)
	}
	if len(a.Violations) != 1 || a.Violations[0].Rule != "no_rewards" {
		t.Errorf("violations alerte = %+v, want [no_rewards]", a.Violations)
	}

	run := fs.last()
	if run.Status != health.StatusAnomaly || run.Source != "playlist" || run.Game != "fh6" {
		t.Errorf("run persisté = %+v, want status=anomaly source=playlist game=fh6", run)
	}
	if len(run.Violations) == 0 {
		t.Error("run.Violations vide, want JSON des violations")
	}
}

// TestObserveOKSilent : un Report sain ne déclenche aucune alerte mais persiste un run ok.
func TestObserveOKSilent(t *testing.T) {
	wh := newWebhook()
	defer wh.srv.Close()
	fs := &fakeStore{}
	m := &health.Monitor{Store: fs, Notifier: health.NewNotifier(wh.srv.URL), Logger: discardLogger()}

	rep := health.Report{Source: "cars", Game: "fh6", Records: 700}
	m.Observe(context.Background(), rep, time.Now())
	if got := wh.all(); len(got) != 0 {
		t.Errorf("alertes = %d, want 0 (run sain)", len(got))
	}
	if run := fs.last(); run.Status != health.StatusOK || run.Records != 700 {
		t.Errorf("run persisté = %+v, want status=ok records=700", run)
	}
}

// TestObserveFailedAlerts : un échec dur (Err) déclenche une alerte status=failed,
// persiste le run et renvoie l'erreur d'origine pour agrégation par l'appelant.
func TestObserveFailedAlerts(t *testing.T) {
	wh := newWebhook()
	defer wh.srv.Close()
	fs := &fakeStore{}
	m := &health.Monitor{Store: fs, Notifier: health.NewNotifier(wh.srv.URL), Logger: discardLogger()}

	cause := errors.New("fetch car list: 500")
	rep := health.Report{Source: "cars", Game: "fh6", Err: cause}
	m.Observe(context.Background(), rep, time.Now())
	alerts := wh.all()
	if len(alerts) != 1 || alerts[0].Status != health.StatusFailed || alerts[0].Error == "" {
		t.Fatalf("alertes = %+v, want 1 alerte status=failed avec error", alerts)
	}
	if run := fs.last(); run.Status != health.StatusFailed || run.Error == "" {
		t.Errorf("run persisté = %+v, want status=failed avec error", run)
	}
}

// TestObserveNilStoreAndNotifier : Store et Notifier nil sont tolérés (log seul, pas de panic).
func TestObserveNilStoreAndNotifier(t *testing.T) {
	m := &health.Monitor{Logger: discardLogger()}
	rep := health.Report{Source: "tracks", Records: 0,
		Violations: []health.Violation{{Rule: "zero_records", Severity: health.SevAnomaly}}}
	m.Observe(context.Background(), rep, time.Now()) // ne doit pas paniquer
}

// TestNotifier couvre le notifier seul : nil si URL vide, erreur sur statut non-2xx.
func TestNotifier(t *testing.T) {
	if health.NewNotifier("") != nil {
		t.Error("NewNotifier(\"\") != nil, want nil (alerte désactivée)")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	if err := health.NewNotifier(srv.URL).Notify(context.Background(), health.Alert{Source: "x"}); err == nil {
		t.Error("Notify sur 500 = nil, want erreur")
	}
}
