package playlist

import (
	"testing"
	"time"
)

// concordeEvent est l'identité forza.net attendue pour la fixture forum_topic.json
// (FH6 Series 01 Week 2, fenêtre 28 mai → 4 juin) : les deux sources concordent.
func concordeEvent() eventMeta {
	return eventMeta{
		Game:   "fh6",
		Series: 1,
		Week:   2,
		Start:  time.Date(2026, 5, 28, 14, 30, 0, 0, time.UTC),
		End:    time.Date(2026, 6, 4, 14, 30, 0, 0, time.UTC),
	}
}

func TestParseForumTitleFixture(t *testing.T) {
	fd, err := ParseForum(readFixture(t, "forum_topic.json"))
	if err != nil {
		t.Fatalf("ParseForum: %v", err)
	}
	fi := fd.Identity
	if !fi.OK {
		t.Fatalf("identité titre non parsée: title=%q", fd.Title)
	}
	if fi.Game != "fh6" || fi.Series != 1 || fi.Week != 2 {
		t.Errorf("identité = game=%q series=%d week=%d, want fh6/1/2", fi.Game, fi.Series, fi.Week)
	}
	if fi.StartMonth != time.May || fi.StartDay != 28 || fi.EndMonth != time.June || fi.EndDay != 4 {
		t.Errorf("fenêtre = %s %d → %s %d, want May 28 → June 4",
			fi.StartMonth, fi.StartDay, fi.EndMonth, fi.EndDay)
	}
}

func TestReconcileConcorde(t *testing.T) {
	fd, _ := ParseForum(readFixture(t, "forum_topic.json"))
	ev := concordeEvent()

	got, div := Reconcile(ev, fd)
	if len(div.Divergences) != 0 {
		t.Errorf("sources concordantes, want 0 divergence, got %+v", div.Divergences)
	}
	if got != ev {
		t.Errorf("ev modifié sans raison: got %+v, want %+v", got, ev)
	}
}

func TestReconcileMismatchForzaWins(t *testing.T) {
	fd, _ := ParseForum(readFixture(t, "forum_topic.json"))
	ev := concordeEvent()
	ev.Series = 2                                             // diverge du forum (1)
	ev.Start = time.Date(2026, 5, 21, 14, 30, 0, 0, time.UTC) // May 21 ≠ May 28

	got, div := Reconcile(ev, fd)

	if d, ok := findDivergence(div, "series"); !ok || d.Kind != "mismatch" || d.Forza != "2" || d.Forum != "1" {
		t.Errorf("divergence series = %+v (ok=%v), want mismatch forza=2 forum=1", d, ok)
	}
	if d, ok := findDivergence(div, "start"); !ok || d.Kind != "mismatch" || d.Forum != "May 28" {
		t.Errorf("divergence start = %+v (ok=%v), want mismatch forum=\"May 28\"", d, ok)
	}
	// forza.net présent fait foi : la valeur primaire n'est jamais écrasée.
	if got.Series != 2 {
		t.Errorf("series écrasée par le forum: got %d, want 2 (forza.net fait foi)", got.Series)
	}
}

func TestReconcileFallbackForumFillsGaps(t *testing.T) {
	fd, _ := ParseForum(readFixture(t, "forum_topic.json"))
	// forza.net a « changé de structure » : slug non parsé → identité à zéro.
	ev := eventMeta{}

	got, div := Reconcile(ev, fd)

	// Le forum prend le relais sur les champs d'identité sûrs.
	if got.Game != "fh6" || got.Series != 1 || got.Week != 2 {
		t.Errorf("fallback identité = game=%q series=%d week=%d, want fh6/1/2", got.Game, got.Series, got.Week)
	}
	for _, field := range []string{"game", "series", "week", "start", "end"} {
		if d, ok := findDivergence(div, field); !ok || d.Kind != "missing_primary" {
			t.Errorf("divergence %s = %+v (ok=%v), want missing_primary", field, d, ok)
		}
	}
	// Doctrine zéro gris : pas de date reconstruite (titre sans année).
	if !got.Start.IsZero() || !got.End.IsZero() {
		t.Errorf("dates reconstruites depuis le forum: start=%s end=%s, want zéro", got.Start, got.End)
	}
}

func TestReconcileUnparsedTitle(t *testing.T) {
	fd := forumData{Title: "Off-topic discussion", Identity: parseForumTitle("Off-topic discussion")}
	ev := concordeEvent()

	got, div := Reconcile(ev, fd)
	if len(div.Divergences) != 1 {
		t.Fatalf("titre illisible, want 1 divergence, got %+v", div.Divergences)
	}
	if d := div.Divergences[0]; d.Field != "title" || d.Kind != "unparsed" {
		t.Errorf("divergence = %+v, want field=title kind=unparsed", d)
	}
	if got != ev {
		t.Errorf("ev modifié alors que titre illisible: got %+v, want %+v", got, ev)
	}
}

func findDivergence(div DivergenceReport, field string) (Divergence, bool) {
	for _, d := range div.Divergences {
		if d.Field == field {
			return d, true
		}
	}
	return Divergence{}, false
}
