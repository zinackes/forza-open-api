package playlist

import (
	"os"
	"testing"
	"time"
)

func mustRead(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(b)
}

// --- tokenizer ---------------------------------------------------------------

func TestSplitParamsRespectsNesting(t *testing.T) {
	got := splitParams(`FH6FPEvent|d1|Big in Japan|Win a [[Cross Country|race]]|{{FH6FPReward|cr| |5,000|c}}|6 = Icon`)
	want := []string{"FH6FPEvent", "d1", "Big in Japan", "Win a [[Cross Country|race]]", "{{FH6FPReward|cr| |5,000|c}}", "6 = Icon"}
	if len(got) != len(want) {
		t.Fatalf("got %d fields %q, want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("field %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestFindTemplatesNested(t *testing.T) {
	s := `x {{FH6FPEvent|seriescar|60|{{FH6FPReward|ex|Mazda_Furai|[[Mazda Furai]]|l}}}} y`
	ev := findTemplates(s, "FH6FPEvent")
	if len(ev) != 1 {
		t.Fatalf("FPEvent: got %d, want 1 (%q)", len(ev), ev)
	}
	rw := findTemplates(ev[0], "FH6FPReward")
	if len(rw) != 1 || rw[0] != "FH6FPReward|ex|Mazda_Furai|[[Mazda Furai]]|l" {
		t.Fatalf("nested FPReward = %q", rw)
	}
}

func TestRenderRewardForms(t *testing.T) {
	g := wikiGames["fh6"]
	cases := map[string]string{
		`{{FH6FPReward|ex|Mazda_Furai|[[Mazda Furai]]|l}}`: "Mazda Furai",
		`{{FH6FPReward|cr| |25,000|c}}`:                    "25,000 Credits",
		`{{FH6FPReward|link| |Oooh... shiny!|c}}`:          "Forza LINK: Oooh... shiny!",
		`{{FH6FPReward|wheelspin}}`:                        "Wheelspin",
		`{{FH6FPReward|super}}`:                            "Super Wheelspin",
	}
	for in, want := range cases {
		if got := g.renderReward(in); got != want {
			t.Errorf("renderReward(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRenderWikiStripsMarkup(t *testing.T) {
	cases := map[string]string{
		`'''Win''' a [[Forza Horizon 6/Events/Road Races|Road Race]]`: "Win a Road Race",
		`Earn a total of {{speed|300}} across Speed Zones`:            "Earn a total of 300 mph across Speed Zones",
	}
	for in, want := range cases {
		if got := renderWiki(in); got != want {
			t.Errorf("renderWiki(%q) = %q, want %q", in, got, want)
		}
	}
}

// --- parse séries / saisons --------------------------------------------------

func TestParseSeriesFH6(t *testing.T) {
	si := wikiGames["fh6"].parseSeries(mustRead(t, "wiki_fh6_series1.txt"))
	if si.name != "Welcome to Japan" || si.series != 1 || si.total != 180 {
		t.Fatalf("identity: name=%q series=%d total=%d", si.name, si.series, si.total)
	}
	if len(si.rewards) != 2 {
		t.Fatalf("series rewards = %d, want 2", len(si.rewards))
	}
	assertReward(t, si.rewards, "series", "Mazda Furai", 33)
	assertReward(t, si.rewards, "series", "Nissan 370Z", 67)
}

func TestParseSeasonFH6(t *testing.T) {
	sd := wikiGames["fh6"].parseSeason(mustRead(t, "wiki_fh6_s1_autumn.txt"))
	if sd.season != "autumn" || sd.start != "May 28, 2026" || sd.end != "June 4, 2026" || sd.total != 45 {
		t.Fatalf("season infobox: %+v", sd)
	}
	if len(sd.rewards) != 2 {
		t.Fatalf("season rewards = %d, want 2", len(sd.rewards))
	}
	assertReward(t, sd.rewards, "season", "Nissan Skyline GT-R V-Spec 1997", 33)

	if len(sd.challenges) != 19 {
		t.Fatalf("challenges = %d, want 19", len(sd.challenges))
	}
	w := sd.challenges[0]
	if w.Scope != "weekly" || w.Name != "King of the Road" || w.Reward != "25,000 Credits" {
		t.Fatalf("weekly = %+v", w)
	}
	for scope, want := range map[string]int{"daily": 7, "seasonal": 9, "online": 1, "monthly": 1} {
		if n := len(byScope(sd.challenges, scope)); n != want {
			t.Errorf("%s count = %d, want %d", scope, n, want)
		}
	}
}

func TestParseSeriesSeasonFH5(t *testing.T) {
	g := wikiGames["fh5"]
	si := g.parseSeries(mustRead(t, "wiki_fh5_series45.txt"))
	if si.name != "Horizon Wilds Takeover" || si.series != 45 || si.total != 285 {
		t.Fatalf("identity: name=%q series=%d total=%d", si.name, si.series, si.total)
	}
	if len(si.rewards) != 2 || *si.rewards[0].AtPercent != 28 || *si.rewards[1].AtPercent != 56 {
		t.Fatalf("series rewards = %+v", si.rewards)
	}

	sd := g.parseSeason(mustRead(t, "wiki_fh5_s45_summer.txt"))
	if sd.season != "summer" || sd.total != 70 {
		t.Fatalf("season infobox: season=%q total=%d", sd.season, sd.total)
	}
	if len(sd.challenges) == 0 || sd.challenges[0].Scope != "weekly" || sd.challenges[0].Name != "Ready to Hoon" {
		t.Fatalf("weekly = %+v", sd.challenges[0])
	}
	// Un séparateur {{FH5FPEvent|arcade}} ne doit pas produire de défi vide.
	for _, c := range sd.challenges {
		if c.Name == "" && c.Requirement == "" {
			t.Fatalf("empty challenge leaked: %+v", c)
		}
	}
}

// --- mapping vers le store ---------------------------------------------------

func TestBuildSeasonFH6(t *testing.T) {
	g := wikiGames["fh6"]
	si := g.parseSeries(mustRead(t, "wiki_fh6_series1.txt"))
	sd := g.parseSeason(mustRead(t, "wiki_fh6_s1_autumn.txt"))

	inWindow := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	u, ok := g.buildSeason(si, "autumn", sd, inWindow)
	if !ok {
		t.Fatal("buildSeason ok=false")
	}
	if u.Series.ID != "fh6-s01w2" || u.Series.Week != 2 || u.Series.Game != "fh6" {
		t.Fatalf("series identity: %+v", u.Series)
	}
	if u.Series.Season == nil || *u.Series.Season != "Autumn" {
		t.Fatalf("season = %v", u.Series.Season)
	}
	if !u.Series.IsCurrent {
		t.Error("expected is_current=true when now is in the season window")
	}
	want := time.Date(2026, 5, 28, 14, 30, 0, 0, time.UTC)
	if !u.Series.StartsAt.Equal(want) {
		t.Errorf("StartsAt = %s, want %s (Thursday 14:30 UTC reset)", u.Series.StartsAt, want)
	}
	// 2 paliers série répétés + 2 paliers saison = 4 ; ids déterministes.
	if len(u.Rewards) != 4 || u.Rewards[0].ID != "fh6-s01w2-rw01" || u.Rewards[3].ID != "fh6-s01w2-rw04" {
		t.Fatalf("rewards = %d %+v", len(u.Rewards), u.Rewards)
	}
	// expires_at : nil pour daily, fin de saison pour le reste.
	for _, c := range u.Challenges {
		if c.Scope == "daily" && c.ExpiresAt != nil {
			t.Errorf("daily %s should not expire", c.ID)
		}
		if c.Scope != "daily" && c.ExpiresAt == nil {
			t.Errorf("%s (%s) should expire at season end", c.ID, c.Scope)
		}
	}

	// Hors fenêtre → pas courante.
	after := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	if u2, _ := g.buildSeason(si, "autumn", sd, after); u2.Series.IsCurrent {
		t.Error("expected is_current=false when now is after the window")
	}
}

func TestBuildBackfillSkipsMissingSeasons(t *testing.T) {
	g := wikiGames["fh6"]
	pages := []seriesPageset{{
		series:      1,
		seriesTitle: "S1",
		seasonTitle: map[string]string{"summer": "S1summer", "autumn": "S1autumn"},
	}}
	wikitext := map[string]string{
		"S1":       mustRead(t, "wiki_fh6_series1.txt"),
		"S1autumn": mustRead(t, "wiki_fh6_s1_autumn.txt"),
		// S1summer absent du wikitext (page non encore publiée) → ignorée.
	}
	out := g.buildBackfill(pages, wikitext, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	if len(out) != 1 || out[0].Series.ID != "fh6-s01w2" {
		t.Fatalf("backfill = %d upserts %+v", len(out), out)
	}
}
