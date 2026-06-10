package playlist

import (
	"os"
	"strings"
	"testing"
	"time"
)

// Fixtures figées capturées sur forza.net / forums.forza.net (FH6 Series 01
// Week 2, « Autumn / Welcome to Japan »). Aucun appel réseau en test.
func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return raw
}

func TestExtractEvents(t *testing.T) {
	events, err := extractEvents(readFixture(t, "event_payload.json"))
	if err != nil {
		t.Fatalf("extractEvents: %v", err)
	}
	ev, err := pickSlug(events, "FH6-Festival-Playlist-S01W2")
	if err != nil {
		t.Fatalf("pickSlug: %v", err)
	}

	if ev.Game != "fh6" || ev.Series != 1 || ev.Week != 2 {
		t.Errorf("identité = game=%q series=%d week=%d, want fh6/1/2", ev.Game, ev.Series, ev.Week)
	}
	wantStart := time.Date(2026, 5, 28, 14, 30, 0, 0, time.UTC)
	wantEnd := time.Date(2026, 6, 4, 14, 30, 0, 0, time.UTC)
	if !ev.Start.Equal(wantStart) || !ev.End.Equal(wantEnd) {
		t.Errorf("fenêtre = [%s,%s], want [%s,%s]", ev.Start, ev.End, wantStart, wantEnd)
	}
	if ev.Category != "Horizon6" {
		t.Errorf("category = %q, want Horizon6", ev.Category)
	}
	if want := "831070"; !contains(ev.ForumURL, want) {
		t.Errorf("forumURL = %q, want contient %q", ev.ForumURL, want)
	}
}

func TestPickCurrentFromIndex(t *testing.T) {
	events, err := extractEvents(readFixture(t, "events_index_payload.json"))
	if err != nil {
		t.Fatalf("extractEvents: %v", err)
	}
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC) // dans la fenêtre S01W2
	ev, err := pickCurrent(events, "fh6", now)
	if err != nil {
		t.Fatalf("pickCurrent: %v", err)
	}
	if ev.Slug != "FH6-Festival-Playlist-S01W2" {
		t.Errorf("courante = %q, want FH6-Festival-Playlist-S01W2", ev.Slug)
	}
}

func TestParseForumRewards(t *testing.T) {
	fd, err := ParseForum(readFixture(t, "forum_topic.json"))
	if err != nil {
		t.Fatalf("ParseForum: %v", err)
	}
	if fd.Season != "Autumn" || fd.Saga != "Welcome to Japan" {
		t.Errorf("heading = season=%q saga=%q, want Autumn / Welcome to Japan", fd.Season, fd.Saga)
	}

	// Paliers de points : at_percent = ratio points/total (180 série, 45 saison).
	assertReward(t, fd.Rewards, "series", "2008 Mazda Furai", 33)        // 60/180
	assertReward(t, fd.Rewards, "series", "2010 Nissan 370Z", 67)        // 120/180
	assertReward(t, fd.Rewards, "season", "1997 Nissan GT-R V-Spec", 33) // 15/45
	assertReward(t, fd.Rewards, "season", "1991 Honda CR-X SiR", 67)     // 30/45

	// Récompenses sans seuil : at_percent nil.
	assertRewardNoPercent(t, fd.Rewards, "event", "1993 Schuppan 962CR")
	assertRewardNoPercent(t, fd.Rewards, "car_pass", "2024 Koenigsegg Gemera")
	assertRewardNoPercent(t, fd.Rewards, "history", "1972 Mazda Cosmo 110S Series II")
}

func TestParseForumChallenges(t *testing.T) {
	fd, err := ParseForum(readFixture(t, "forum_topic.json"))
	if err != nil {
		t.Fatalf("ParseForum: %v", err)
	}

	weekly := byScope(fd.Challenges, "weekly")
	if len(weekly) != 1 {
		t.Fatalf("weekly = %d défis, want 1", len(weekly))
	}
	if weekly[0].Name != "King of the Road" {
		t.Errorf("weekly name = %q, want King of the Road", weekly[0].Name)
	}
	if !contains(weekly[0].Requirement, "Own and drive the 1989 Nissan Silvia") {
		t.Errorf("weekly requirement = %q, want étapes en séquence", weekly[0].Requirement)
	}
	if !contains(weekly[0].Reward, "25,000 credits") {
		t.Errorf("weekly reward = %q, want crédits", weekly[0].Reward)
	}

	if daily := byScope(fd.Challenges, "daily"); len(daily) != 7 {
		t.Errorf("daily = %d défis, want 7", len(daily))
	}

	seasonal := byScope(fd.Challenges, "seasonal")
	if len(seasonal) < 8 {
		t.Errorf("seasonal = %d défis, want ≥ 8", len(seasonal))
	}
	if !hasChallengeNamed(seasonal, "Trail Mix") || !hasChallengeNamed(seasonal, "#TokyoTower") {
		t.Errorf("seasonal sans Trail Mix / #TokyoTower : %v", names(seasonal))
	}
}

func TestNormalize(t *testing.T) {
	events, _ := extractEvents(readFixture(t, "event_payload.json"))
	ev, _ := pickSlug(events, "FH6-Festival-Playlist-S01W2")
	fd, _ := ParseForum(readFixture(t, "forum_topic.json"))

	// Découverte comme courante → is_current vrai.
	ser, rewards, challenges := Normalize(ev, fd, true)

	if ser.ID != "fh6-s01w2" {
		t.Errorf("id = %q, want fh6-s01w2", ser.ID)
	}
	if ser.Season == nil || *ser.Season != "Autumn" {
		t.Errorf("season = %v, want Autumn", ser.Season)
	}
	if ser.Name == nil || *ser.Name != "Welcome to Japan" {
		t.Errorf("name = %v, want Welcome to Japan", ser.Name)
	}
	if !ser.IsCurrent {
		t.Error("is_current = false, want true (now dans la fenêtre)")
	}
	if len(rewards) == 0 || len(challenges) == 0 {
		t.Fatalf("rewards=%d challenges=%d, want non vides", len(rewards), len(challenges))
	}
	// IDs stables et préfixés.
	if rewards[0].ID != "fh6-s01w2-rw01" || challenges[0].ID != "fh6-s01w2-ch01" {
		t.Errorf("ids enfants = %q / %q", rewards[0].ID, challenges[0].ID)
	}
	// Les défis daily n'expirent pas (prolongés) ; les autres expirent en fin de saison.
	for _, c := range challenges {
		if c.Scope == "daily" && c.ExpiresAt != nil {
			t.Errorf("daily %q a un expires_at non nul", c.Name)
		}
		if c.Scope == "weekly" && (c.ExpiresAt == nil || !c.ExpiresAt.Equal(ev.End)) {
			t.Errorf("weekly %q expires_at = %v, want %s", c.Name, c.ExpiresAt, ev.End)
		}
	}

	// Re-seed explicite non courant → is_current faux.
	serPast, _, _ := Normalize(ev, fd, false)
	if serPast.IsCurrent {
		t.Error("is_current = true alors que non courant, want false")
	}
}

// --- helpers d'assertion -----------------------------------------------------

func assertReward(t *testing.T, rs []parsedReward, typ, item string, pct int) {
	t.Helper()
	for _, r := range rs {
		if r.Type == typ && r.Item == item {
			if r.AtPercent == nil || *r.AtPercent != pct {
				t.Errorf("reward %s/%q at_percent = %v, want %d", typ, item, r.AtPercent, pct)
			}
			return
		}
	}
	t.Errorf("reward %s/%q absente", typ, item)
}

func assertRewardNoPercent(t *testing.T, rs []parsedReward, typ, item string) {
	t.Helper()
	for _, r := range rs {
		if r.Type == typ && r.Item == item {
			if r.AtPercent != nil {
				t.Errorf("reward %s/%q at_percent = %v, want nil", typ, item, *r.AtPercent)
			}
			return
		}
	}
	t.Errorf("reward %s/%q absente", typ, item)
}

func byScope(cs []parsedChallenge, scope string) []parsedChallenge {
	var out []parsedChallenge
	for _, c := range cs {
		if c.Scope == scope {
			out = append(out, c)
		}
	}
	return out
}

func hasChallengeNamed(cs []parsedChallenge, name string) bool {
	for _, c := range cs {
		if c.Name == name {
			return true
		}
	}
	return false
}

func names(cs []parsedChallenge) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.Name
	}
	return out
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }
