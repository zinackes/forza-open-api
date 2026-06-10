package health

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Alert est le corps JSON POSTé sur le webhook d'alerte. Schéma générique :
// convient à un endpoint maison ou à un relais Slack/Discord qui sait le
// transformer. Aucune donnée sensible (pas de secret, pas de clé).
type Alert struct {
	Service    string      `json:"service"`
	Source     string      `json:"source"` // playlist | cars | tracks
	Game       string      `json:"game,omitempty"`
	Status     string      `json:"status"` // anomaly | failed
	Records    int         `json:"records"`
	Violations []Violation `json:"violations,omitempty"`
	Error      string      `json:"error,omitempty"`
	Time       time.Time   `json:"time"`
}

// Notifier poste les alertes sur un webhook HTTP. Timeout borné. Toute erreur est
// renvoyée à l'appelant (qui se contente de la logger) : une alerte ratée ne doit
// jamais casser le run d'ingestion.
type Notifier struct {
	URL    string
	Client *http.Client
}

// NewNotifier construit un Notifier, ou nil si url est vide (alerte HTTP
// désactivée → le Monitor se rabat sur le log structuré seul).
func NewNotifier(url string) *Notifier {
	if url == "" {
		return nil
	}
	return &Notifier{URL: url, Client: &http.Client{Timeout: 10 * time.Second}}
}

// Notify POST l'alerte en JSON. Erreur si la construction, l'envoi, ou un statut
// HTTP >= 300 échouent.
func (n *Notifier) Notify(ctx context.Context, a Alert) error {
	body, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("marshal alert: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build alert request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.Client.Do(req)
	if err != nil {
		return fmt.Errorf("post alert: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("alert webhook: status %d", resp.StatusCode)
	}
	return nil
}
