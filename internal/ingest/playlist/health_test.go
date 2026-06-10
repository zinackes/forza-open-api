package playlist

import (
	"testing"
	"time"

	"github.com/zinackes/forza-open-api/internal/health"
)

// TestCheckRunHealthySeries : une passe complète (fixtures forza.net + forum
// figées) ne viole aucun invariant → run sain.
func TestCheckRunHealthySeries(t *testing.T) {
	events, err := extractEvents(readFixture(t, "event_payload.json"))
	if err != nil {
		t.Fatalf("extractEvents: %v", err)
	}
	ev, err := pickSlug(events, "FH6-Festival-Playlist-S01W2")
	if err != nil {
		t.Fatalf("pickSlug: %v", err)
	}
	fd, err := ParseForum(readFixture(t, "forum_topic.json"))
	if err != nil {
		t.Fatalf("ParseForum: %v", err)
	}
	ser, rewards, challenges := Normalize(ev, fd, true)
	res := Result{Series: ser, Rewards: len(rewards), Challenges: len(challenges)}

	now := time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC) // dans la fenêtre de la série
	if vs := CheckRun(res, now); len(vs) != 0 {
		t.Errorf("CheckRun (série saine) = %+v, want aucune violation", vs)
	}
}

// TestCheckRunFlagsBrokenStructure (cœur de la demande) : un payload forza.net
// dont la structure a changé (dates passées en epoch millis → non parsables →
// dates à zéro, et lien forum déplacé → aucun détail récupéré) parse SANS erreur
// mais produit une série dégradée. CheckRun doit lever des anomalies → alerte.
func TestCheckRunFlagsBrokenStructure(t *testing.T) {
	events, err := extractEvents(readFixture(t, "event_payload_broken.json"))
	if err != nil {
		t.Fatalf("extractEvents (payload cassé) = %v, want parse sans erreur (anomalie silencieuse)", err)
	}
	ev, err := pickSlug(events, "FH6-Festival-Playlist-S01W2")
	if err != nil {
		t.Fatalf("pickSlug: %v", err)
	}
	// Lien forum introuvable/déplacé → détail vide (récompenses + défis absents).
	ser, rewards, challenges := Normalize(ev, forumData{}, true)
	res := Result{Series: ser, Rewards: len(rewards), Challenges: len(challenges)}

	now := time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC)
	vs := CheckRun(res, now)

	anomalies := map[string]bool{}
	for _, v := range vs {
		if v.Severity == health.SevAnomaly {
			anomalies[v.Rule] = true
		}
	}
	if len(anomalies) == 0 {
		t.Fatalf("CheckRun (structure cassée) = %+v, want ≥1 anomalie", vs)
	}
	// Les invariants clés de rupture doivent être attrapés.
	for _, want := range []string{"dates_zero", "name_empty", "no_rewards", "no_challenges"} {
		if !anomalies[want] {
			t.Errorf("anomalie %q absente, got %v", want, keys(anomalies))
		}
	}
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
