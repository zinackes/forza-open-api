package health

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/zinackes/forza-open-api/internal/store"
)

// serviceName identifie l'émetteur dans l'alerte (relais Slack/Discord, tri ops).
const serviceName = "scraper-health"

// RunStore persiste un run (implémenté par *store.Store). Interface pour tester le
// Monitor sans base et pour rendre la persistance optionnelle (nil = pas de DB).
type RunStore interface {
	InsertScrapeRun(ctx context.Context, r store.ScrapeRun) error
}

// Monitor observe le résultat de chaque passe d'ingestion : log structuré
// (toujours), persistance du run (si Store), alerte webhook sur anomalie ou échec
// (si Notifier). Tout est fail-safe : ni la persistance ni l'alerte ne doivent
// casser le run d'ingestion.
type Monitor struct {
	Store    RunStore  // nil → pas de persistance (log/alerte seuls)
	Notifier *Notifier // nil → pas d'alerte externe (log seul)
	Logger   *slog.Logger
}

// Observe logue, persiste et alerte le bilan d'un run (effets de bord seulement :
// l'appelant agrège lui-même rep.Err / rep.HasAnomaly()). startedAt sert la durée
// et le journal scrape_runs. Les échecs de persistance/alerte sont loggés, jamais
// propagés : une alerte ratée ne doit pas casser le run d'ingestion.
func (m *Monitor) Observe(ctx context.Context, rep Report, startedAt time.Time) {
	finishedAt := time.Now().UTC()
	durMS := finishedAt.Sub(startedAt).Milliseconds()
	status := rep.Status()

	m.log(rep, status, durMS)
	m.persist(ctx, rep, status, startedAt, finishedAt, durMS)
	if status != StatusOK {
		m.alert(ctx, rep, status)
	}
}

func (m *Monitor) log(rep Report, status string, durMS int64) {
	if m.Logger == nil {
		return
	}
	attrs := []any{
		"source", rep.Source, "game", rep.Game, "status", status,
		"records", rep.Records, "duration_ms", durMS,
	}
	switch status {
	case StatusFailed:
		m.Logger.Error("ingestion run failed", append(attrs, "err", rep.Err)...)
	case StatusAnomaly:
		m.Logger.Warn("ingestion run anomaly",
			append(attrs, "violations", violationRules(rep.Violations))...)
	default:
		if w := violationRules(warningsOnly(rep.Violations)); len(w) > 0 {
			attrs = append(attrs, "warnings", w)
		}
		m.Logger.Info("ingestion run ok", attrs...)
	}
}

func (m *Monitor) persist(ctx context.Context, rep Report, status string, started, finished time.Time, durMS int64) {
	if m.Store == nil {
		return
	}
	var violations json.RawMessage
	if len(rep.Violations) > 0 {
		if b, err := json.Marshal(rep.Violations); err == nil {
			violations = b
		}
	}
	errStr := ""
	if rep.Err != nil {
		errStr = rep.Err.Error()
	}
	run := store.ScrapeRun{
		Source: rep.Source, Game: rep.Game, Status: status, Records: rep.Records,
		Violations: violations, Error: errStr, DurationMS: int(durMS),
		StartedAt: started, FinishedAt: finished,
	}
	if err := m.Store.InsertScrapeRun(ctx, run); err != nil && m.Logger != nil {
		m.Logger.Error("scrape_run persist failed",
			"source", rep.Source, "game", rep.Game, "err", err)
	}
}

func (m *Monitor) alert(ctx context.Context, rep Report, status string) {
	if m.Notifier == nil {
		return
	}
	errStr := ""
	if rep.Err != nil {
		errStr = rep.Err.Error()
	}
	a := Alert{
		Service: serviceName, Source: rep.Source, Game: rep.Game, Status: status,
		Records: rep.Records, Violations: rep.Violations, Error: errStr,
		Time: time.Now().UTC(),
	}
	if err := m.Notifier.Notify(ctx, a); err != nil && m.Logger != nil {
		m.Logger.Error("alert webhook failed",
			"source", rep.Source, "game", rep.Game, "err", err)
	}
}

// violationRules réduit des violations à leurs règles (log compact, déterministe).
func violationRules(vs []Violation) []string {
	out := make([]string, 0, len(vs))
	for _, v := range vs {
		out = append(out, v.Rule)
	}
	return out
}

func warningsOnly(vs []Violation) []Violation {
	var out []Violation
	for _, v := range vs {
		if v.Severity == SevWarning {
			out = append(out, v)
		}
	}
	return out
}
