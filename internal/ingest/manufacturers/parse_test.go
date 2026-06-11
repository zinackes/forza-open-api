package manufacturers

import "testing"

// Fixtures figées (wikitext réel simplifié des pages Category:Manufacturers) —
// aucun appel réseau en test (doctrine testing).

// origin sous forme de NOM de pays.
const wtItaly = `{{game|fh5=y|fh6=y}}
{{InfoboxMFR
|image   = FH6 Abarth Logo.png
|founded = 1949
|origin  = italy
|related = [[FIAT]]
}}
'''Abarth''' is an Italian car manufacturer.`

// origin sous forme de CODE + suffixe drapeau (« usa f ») — on ne retient que le
// premier champ.
const wtUSAFlag = `{{game|fh6=y}}
{{InfoboxMFR
|image = Icon Make AMG Transport Dynamics FH5.png
|founded = Early 2300s
|background = black
|origin = usa f
}}
'''AMG Transport Dynamics''' is a fictional manufacturer.`

// origin composite « ukger » (MINI : marque britannique, propriété BMW) → primaire UK.
const wtUKGer = `{{InfoboxMFR
|origin = ukger
}}
'''MINI''' is a British marque.`

// origin code inconnu → country NULL + anomalie reportée.
const wtUnknown = `{{InfoboxMFR
|founded = 2000
|origin = atlantis
}}`

// InfoboxMFR sans champ origin → country NULL (pas une anomalie).
const wtNoOrigin = `{{InfoboxMFR
|founded = 1990
|related = [[Foo]]
}}`

// Pas d'InfoboxMFR du tout → country NULL.
const wtNoInfobox = `{{game|fh6=y}}
'''Mystery''' is a manufacturer without an infobox.`

func TestNormalize(t *testing.T) {
	pages := map[string]string{
		"Abarth":                 wtItaly,
		"AMG Transport Dynamics": wtUSAFlag,
		"MINI":                   wtUKGer,
		"Atlantis Motors":        wtUnknown,
		"NoOrigin Co":            wtNoOrigin,
		"Mystery":                wtNoInfobox,
	}
	mfrs, rep := Normalize("fh6", pages)

	if got, want := len(mfrs), 6; got != want {
		t.Fatalf("manufacturers = %d, want %d", got, want)
	}
	if rep.Rows != 6 {
		t.Errorf("Rows = %d, want 6", rep.Rows)
	}
	if rep.CountryResolved != 3 {
		t.Errorf("CountryResolved = %d, want 3 (Abarth, AMG, MINI)", rep.CountryResolved)
	}
	if rep.NoOrigin != 2 {
		t.Errorf("NoOrigin = %d, want 2 (NoOrigin Co, Mystery)", rep.NoOrigin)
	}
	if got := rep.UnknownOrigins["origin:atlantis"]; got != 1 {
		t.Errorf("UnknownOrigins[origin:atlantis] = %d, want 1", got)
	}

	// Sortie triée par nom (déterministe) ; vérifie name + country par entrée.
	want := map[string]*string{
		"Abarth":                 strptr("Italy"),
		"AMG Transport Dynamics": strptr("United States"),
		"MINI":                   strptr("United Kingdom"),
		"Atlantis Motors":        nil,
		"NoOrigin Co":            nil,
		"Mystery":                nil,
	}
	for _, m := range mfrs {
		if m.Game != "fh6" {
			t.Errorf("%s: game = %q, want fh6", m.Name, m.Game)
		}
		exp, ok := want[m.Name]
		if !ok {
			t.Errorf("unexpected manufacturer %q", m.Name)
			continue
		}
		switch {
		case exp == nil && m.Country != nil:
			t.Errorf("%s: country = %q, want NULL", m.Name, *m.Country)
		case exp != nil && m.Country == nil:
			t.Errorf("%s: country = NULL, want %q", m.Name, *exp)
		case exp != nil && m.Country != nil && *exp != *m.Country:
			t.Errorf("%s: country = %q, want %q", m.Name, *m.Country, *exp)
		}
	}
}

// Sortie déterministe : Normalize trie par titre.
func TestNormalizeSorted(t *testing.T) {
	pages := map[string]string{"Zonda": wtItaly, "Alpha": wtItaly, "Mid": wtItaly}
	mfrs, _ := Normalize("fh6", pages)
	want := []string{"Alpha", "Mid", "Zonda"}
	for i, m := range mfrs {
		if m.Name != want[i] {
			t.Fatalf("order[%d] = %q, want %q", i, m.Name, want[i])
		}
	}
}

func TestCountryFromOrigin(t *testing.T) {
	cases := []struct {
		in         string
		want       string // "" => nil
		recognized bool
	}{
		{"italy", "Italy", true},
		{"ita", "Italy", true},
		{"usa", "United States", true},
		{"us", "United States", true},
		{"usa f", "United States", true},  // suffixe drapeau ignoré
		{"UK F", "United Kingdom", true},  // casse/espaces
		{"ukger", "United Kingdom", true}, // composite → primaire
		{"sw", "Sweden", true},            // Koenigsegg
		{"uae", "United Arab Emirates", true},
		{"", "", true},          // absence : pas une anomalie
		{"atlantis", "", false}, // inconnu : country NULL + non reconnu
	}
	for _, c := range cases {
		// countryFromOrigin attend un token déjà lowercased par parseOrigin ; on
		// reproduit la normalisation d'entrée pour les cas en majuscules.
		country, _, rec := countryFromOrigin(lower(c.in))
		if rec != c.recognized {
			t.Errorf("%q: recognized = %v, want %v", c.in, rec, c.recognized)
		}
		switch {
		case c.want == "" && country != nil:
			t.Errorf("%q: country = %q, want nil", c.in, *country)
		case c.want != "" && country == nil:
			t.Errorf("%q: country = nil, want %q", c.in, c.want)
		case c.want != "" && country != nil && *country != c.want:
			t.Errorf("%q: country = %q, want %q", c.in, *country, c.want)
		}
	}
}

func TestParseOrigin(t *testing.T) {
	if got := parseOrigin(wtItaly); got != "italy" {
		t.Errorf("parseOrigin(italy) = %q, want italy", got)
	}
	if got := parseOrigin(wtUSAFlag); got != "usa f" {
		t.Errorf("parseOrigin(usa f) = %q, want %q", got, "usa f")
	}
	if got := parseOrigin(wtNoOrigin); got != "" {
		t.Errorf("parseOrigin(no origin) = %q, want empty", got)
	}
	if got := parseOrigin(wtNoInfobox); got != "" {
		t.Errorf("parseOrigin(no infobox) = %q, want empty", got)
	}
}

func strptr(s string) *string { return &s }

func lower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}
