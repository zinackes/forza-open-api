package cars

import "testing"

// TestManufacturers vérifie la dérivation make → code pays depuis la fixture
// liste : mêmes conditions de make que Normalize (ligne sans infobox écartée),
// code pays stocké tel que sourcé (minuscules), absence de code → nil, tri par
// nom. Fixture : Abarth/ita, Ferrari/ita, Honda/jpn ; « Mystery Concept » et
// « No Infobox Car » sans infobox → écartées.
func TestManufacturers(t *testing.T) {
	rows := loadList(t)
	infoboxes := map[string]Infobox{
		"Abarth 695 Biposto": ParseInfobox(ibAbarth),
		"Ferrari F50":        ParseInfobox(ibFerrari),
		"Honda Civic Type R": ParseInfobox(ibHonda),
	}

	mans := Manufacturers(rows, infoboxes, "fh6")

	if len(mans) != 3 {
		t.Fatalf("manufacturers = %d, want 3 (%+v)", len(mans), mans)
	}
	want := map[string]string{"Abarth": "ita", "Ferrari": "ita", "Honda": "jpn"}
	for i, m := range mans {
		if m.Game != "fh6" {
			t.Errorf("%s: game = %q, want fh6", m.Name, m.Game)
		}
		code, ok := want[m.Name]
		if !ok {
			t.Errorf("make inattendu: %q", m.Name)
			continue
		}
		if m.Country == nil || *m.Country != code {
			t.Errorf("%s: country = %v, want %q", m.Name, m.Country, code)
		}
		// Tri par nom (déterminisme de l'upsert).
		if i > 0 && mans[i-1].Name > m.Name {
			t.Errorf("résultat non trié: %q avant %q", mans[i-1].Name, m.Name)
		}
	}
}

// TestManufacturersConflictAndEmpty : un même make porté par plusieurs lignes —
// le code pays majoritaire gagne ; un make dont aucune ligne ne donne de code →
// Country nil (jamais de valeur inventée).
func TestManufacturersConflictAndEmpty(t *testing.T) {
	rows := []ListRow{
		{Name: "Dual Make A", Country: "ita"},
		{Name: "Dual Make B", Country: "ger"},
		{Name: "Dual Make C", Country: "ger"},
		{Name: "Bare Make X", Country: ""},
	}
	// Model = tout sauf le premier mot → make commun « Dual » / « Bare ».
	infoboxes := map[string]Infobox{
		"Dual Make A": {Model: "Make A"},
		"Dual Make B": {Model: "Make B"},
		"Dual Make C": {Model: "Make C"},
		"Bare Make X": {Model: "Make X"},
	}

	mans := Manufacturers(rows, infoboxes, "fh6")
	if len(mans) != 2 {
		t.Fatalf("manufacturers = %d, want 2 (%+v)", len(mans), mans)
	}

	bare, dual := mans[0], mans[1]
	if bare.Name != "Bare" || dual.Name != "Dual" {
		t.Fatalf("makes = %q, %q ; want Bare, Dual", bare.Name, dual.Name)
	}
	if bare.Country != nil {
		t.Errorf("Bare: country = %q, want nil (aucun code sourcé)", *bare.Country)
	}
	if dual.Country == nil || *dual.Country != "ger" {
		t.Errorf("Dual: country = %v, want ger (majorité 2/3)", dual.Country)
	}
}
