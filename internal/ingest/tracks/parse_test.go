package tracks

import (
	"os"
	"testing"
	"time"
)

// TestParseTracks vérifie le parsing d'un dataset figé, sans appel réseau. Le
// fixture porte 2 entrées valides + 1 de type inconnu qui doit être ignorée.
// Données illustratives : ce n'est pas un backfill réel.
func TestParseTracks(t *testing.T) {
	raw, err := os.ReadFile("testdata/tracks.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	verifiedAt := time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC)

	parsed, skipped, err := ParseTracks(raw, verifiedAt)
	if err != nil {
		t.Fatalf("ParseTracks: %v", err)
	}
	if len(parsed) != 2 {
		t.Fatalf("got %d tracks, want 2", len(parsed))
	}
	if skipped != 1 {
		t.Errorf("skipped = %d, want 1 (type inconnu ignoré)", skipped)
	}

	got := parsed[0]
	if got.Game != "fh6" {
		t.Errorf("game = %q, want fh6", got.Game)
	}
	if got.Type != "circuit" {
		t.Errorf("type = %q, want circuit", got.Type)
	}
	if got.Source == nil || *got.Source == "" {
		t.Error("source non propagée depuis l'enveloppe")
	}
	if got.LastVerified == nil || !got.LastVerified.Equal(verifiedAt) {
		t.Errorf("lastVerified = %v, want %v", got.LastVerified, verifiedAt)
	}
	if got.StartLat == nil || *got.StartLat != 0.10 {
		t.Errorf("startLat = %v, want 0.10", got.StartLat)
	}

	// Champ absent → NULL (jamais inventé) : la 2e entrée n'a ni surface ni coords.
	if parsed[1].SurfaceMix != nil || parsed[1].StartLat != nil {
		t.Error("champ absent doit rester nil (pas de donnée inventée)")
	}
}

// TestParseTracksRejectsMissingGame : l'enveloppe doit déclarer game (game-agnostic).
func TestParseTracksRejectsMissingGame(t *testing.T) {
	if _, _, err := ParseTracks([]byte(`{"tracks":[]}`), time.Now()); err == nil {
		t.Fatal("attendu une erreur quand le champ game est absent")
	}
}

// TestParseTracksInvalidJSON : un JSON malformé remonte une erreur, pas un panic.
func TestParseTracksInvalidJSON(t *testing.T) {
	if _, _, err := ParseTracks([]byte(`{not json`), time.Now()); err == nil {
		t.Fatal("attendu une erreur de décodage")
	}
}
