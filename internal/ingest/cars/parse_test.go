package cars

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Infoboxes figées (frozen) : aucun appel réseau en test, conformément à la
// doctrine (parsing testé sur fixtures). Une voiture de la liste (« No Infobox
// Car ») n'en a volontairement pas → doit être écartée.
const (
	ibAbarth = `{{CarInfobox
|manufacturer = Abarth
|model = 695 Biposto
|year = 2016
|type = p
|origin = italy
|layout = ff
|power = 186
}}`
	ibFerrari = `{{CarInfobox
|manufacturer = Ferrari
|model = F50
|layout = mr
|origin = ita
}}`
	ibHonda = `{{CarInfobox
|manufacturer = Honda
|model = Civic Type R
|layout = ff
|origin = japan
}}`
)

func loadList(t *testing.T) []ListRow {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "cars_list.wikitext"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return ParseCarList(string(raw), "CarListStatsFH6")
}

func TestParseCarList(t *testing.T) {
	rows := loadList(t)
	if len(rows) != 5 {
		t.Fatalf("rows = %d, want 5", len(rows))
	}
	got := rows[0]
	if got.Name != "Abarth 695 Biposto" || got.PI != "540" || got.Rarity != "c" ||
		got.Obtain != "htf" || got.Country != "ita" || got.Stats[0] != "4.9" {
		t.Fatalf("first row mal parsée: %+v", got)
	}
}

func TestParseInfobox(t *testing.T) {
	ib := ParseInfobox(ibAbarth)
	if ib.Layout != "ff" || ib.Model != "695 Biposto" || ib.Manufacturer != "Abarth" {
		t.Fatalf("infobox mal parsée: %+v", ib)
	}
	if (Infobox{}) != ParseInfobox("pas de template ici") {
		t.Fatalf("wikitext sans CarInfobox doit donner un Infobox vide")
	}
}

func TestNormalize(t *testing.T) {
	rows := loadList(t)
	infoboxes := map[string]Infobox{
		"Abarth 695 Biposto": ParseInfobox(ibAbarth),
		"Ferrari F50":        ParseInfobox(ibFerrari),
		"Honda Civic Type R": ParseInfobox(ibHonda),
		// « No Infobox Car » absente volontairement.
	}

	out, rep := Normalize(rows, infoboxes, "fh6")

	if rep.Imported != 3 || len(out) != 3 {
		t.Fatalf("imported = %d (out=%d), want 3", rep.Imported, len(out))
	}
	if rep.SkipReasons["no_pi"] != 1 {
		t.Errorf("no_pi = %d, want 1 (Mystery Concept, PI ???)", rep.SkipReasons["no_pi"])
	}
	if rep.SkipReasons["no_infobox"] != 1 {
		t.Errorf("no_infobox = %d, want 1 (No Infobox Car)", rep.SkipReasons["no_infobox"])
	}

	by := map[string]int{}
	for i := range out {
		by[out[i].ID] = i
	}

	// Abarth : PI 540 → class B, layout ff → FWD, make dérivé, valeur 250000.
	ab := out[by["fh6-abarth-695-biposto"]]
	if ab.Class != "B" || ab.Drivetrain != "FWD" || ab.Make != "Abarth" {
		t.Errorf("Abarth: class/dt/make = %s/%s/%s, want B/FWD/Abarth", ab.Class, ab.Drivetrain, ab.Make)
	}
	if ab.ValueCr == nil || *ab.ValueCr != 250000 {
		t.Errorf("Abarth value_cr = %v, want 250000", ab.ValueCr)
	}
	if ab.Rarity == nil || *ab.Rarity != "common" {
		t.Errorf("Abarth rarity = %v, want common", ab.Rarity)
	}
	if ab.ObtainMethod == nil || *ab.ObtainMethod != "Hard to Find" {
		t.Errorf("Abarth obtain = %v, want 'Hard to Find'", ab.ObtainMethod)
	}
	// Champs non sourcés → NULL (jamais inventés).
	if ab.BodyType != nil || ab.Category != nil || ab.ImageURL != nil {
		t.Errorf("Abarth: body_type/category/image_url doivent être NULL, got %v/%v/%v",
			ab.BodyType, ab.Category, ab.ImageURL)
	}
	// stats JSONB : 6 clés dont offroad.
	var stats map[string]float64
	if err := json.Unmarshal(ab.Stats, &stats); err != nil {
		t.Fatalf("Abarth stats invalides: %v", err)
	}
	if stats["speed"] != 4.9 || stats["offroad"] != 5.1 || len(stats) != 6 {
		t.Errorf("Abarth stats = %v", stats)
	}

	// Ferrari F50 : PI 912 → class R (bande FH6 901-998), mid-engine RWD.
	fe := out[by["fh6-ferrari-f50"]]
	if fe.Class != "R" || fe.Drivetrain != "RWD" {
		t.Errorf("Ferrari: class/dt = %s/%s, want R/RWD", fe.Class, fe.Drivetrain)
	}

	// Honda Forza Edition : id suffixé, nom suffixé, rareté forza_edition.
	idx, ok := by["fh6-honda-civic-type-r-forza-edition"]
	if !ok {
		t.Fatalf("id Forza Edition absent du catalogue")
	}
	hf := out[idx]
	if hf.Name != "Honda Civic Type R Forza Edition" || hf.Class != "A" {
		t.Errorf("Honda FE: name/class = %q/%s, want 'Honda Civic Type R Forza Edition'/A", hf.Name, hf.Class)
	}
	if hf.Rarity == nil || *hf.Rarity != "forza_edition" {
		t.Errorf("Honda FE rarity = %v, want forza_edition", hf.Rarity)
	}
}

func TestClassFromPI(t *testing.T) {
	cases := []struct {
		pi   int
		want string
	}{
		{100, "D"}, {400, "D"}, {401, "C"}, {500, "C"}, {501, "B"}, {600, "B"},
		{601, "A"}, {700, "A"}, {701, "S1"}, {800, "S1"}, {801, "S2"}, {900, "S2"},
		{901, "R"}, {998, "R"}, {999, "X"},
	}
	for _, c := range cases {
		if got := classFromPI("fh6", c.pi); got != c.want {
			t.Errorf("classFromPI(fh6, %d) = %q, want %q", c.pi, got, c.want)
		}
	}
	if classFromPI("fh5", 540) != "" {
		t.Errorf("jeu non supporté doit donner \"\"")
	}
}

func TestDrivetrainFromLayout(t *testing.T) {
	cases := map[string]string{
		"ff": "FWD", "fr": "RWD", "mr": "RWD", "rr": "RWD",
		"f4": "AWD", "fa": "AWD", "m4": "AWD", "r4": "AWD",
		"rwd": "RWD", "awd": "AWD", "4wd": "AWD",
		"": "", "x": "", "weird": "",
	}
	for in, want := range cases {
		if got := drivetrainFromLayout(in); got != want {
			t.Errorf("drivetrainFromLayout(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValueAndMakeHelpers(t *testing.T) {
	if v := valueCrPtr("3,000,000"); v == nil || *v != 3000000 {
		t.Errorf("valueCrPtr = %v, want 3000000", v)
	}
	if valueCrPtr("") != nil || valueCrPtr("?") != nil {
		t.Errorf("valeur vide/non numérique doit donner nil")
	}
	if m := deriveMake("Abarth 695 Biposto", "695 Biposto"); m != "Abarth" {
		t.Errorf("deriveMake = %q, want Abarth", m)
	}
	if m := deriveMake("Ferrari F50", ""); m != "Ferrari" {
		t.Errorf("deriveMake sans model = %q, want Ferrari", m)
	}
	if slug("Abarth 695 Biposto!!") != "abarth-695-biposto" {
		t.Errorf("slug = %q", slug("Abarth 695 Biposto!!"))
	}
}
