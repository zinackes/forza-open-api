package cars

import (
	"testing"

	"github.com/zinackes/forza-open-api/internal/store"
)

// Voiture de référence valide : PI 650 → class A (bande FH6), AWD, make avec
// lettres. Doit produire 0 anomalie. Les cas mutent un seul champ à la fois.
func validCar() store.Car {
	return store.Car{
		ID: "fh6-ref", Game: "fh6", Name: "Ref Car", Make: "Honda",
		Class: "A", PI: 650, Drivetrain: "AWD",
	}
}

func TestValidate(t *testing.T) {
	mut := func(f func(*store.Car)) store.Car {
		c := validCar()
		f(&c)
		return c
	}

	cases := []struct {
		name  string
		car   store.Car
		rules []string // ensemble attendu de "severity:rule"
	}{
		{"valide", validCar(), nil},
		{"pi trop bas", mut(func(c *store.Car) { c.PI = 50; c.Class = "D" }), []string{"blocking:pi_range"}},
		{"pi trop haut", mut(func(c *store.Car) { c.PI = 1500; c.Class = "X" }), []string{"blocking:pi_range"}},
		{"class hors enum", mut(func(c *store.Car) { c.Class = "Z" }), []string{"blocking:class_enum"}},
		{"drivetrain hors enum", mut(func(c *store.Car) { c.Drivetrain = "6WD" }), []string{"blocking:drivetrain_enum"}},
		{"game vide", mut(func(c *store.Car) { c.Game = "" }), []string{"blocking:game_empty"}},
		{"name vide", mut(func(c *store.Car) { c.Name = "" }), []string{"blocking:name_empty"}},
		{"make vide", mut(func(c *store.Car) { c.Make = "" }), []string{"blocking:make_empty"}},
		{"make sans lettre", mut(func(c *store.Car) { c.Make = "695" }), []string{"warning:make_no_letter"}},
		{"class incohérente avec PI", mut(func(c *store.Car) { c.Class = "S2" }), []string{"warning:class_pi_mismatch"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rep := Validate([]store.Car{tc.car}, "fh6")
			if rep.Checked != 1 {
				t.Fatalf("Checked = %d, want 1", rep.Checked)
			}
			got := map[string]bool{}
			for _, a := range rep.Anomalies {
				if a.CarID != tc.car.ID {
					t.Errorf("anomalie sur id %q, want %q", a.CarID, tc.car.ID)
				}
				if a.Detail == "" {
					t.Errorf("rule %s sans Detail", a.Rule)
				}
				got[string(a.Severity)+":"+a.Rule] = true
			}
			if len(got) != len(tc.rules) {
				t.Fatalf("anomalies = %v, want %v", got, tc.rules)
			}
			for _, want := range tc.rules {
				if !got[want] {
					t.Errorf("anomalie %s manquante (got %v)", want, got)
				}
			}
		})
	}
}

// Helpers du rapport : Count, BlockingIDs, RuleCounts sur un lot mixte.
func TestValidationReportHelpers(t *testing.T) {
	cars := []store.Car{
		validCar(),
		{ID: "fh6-bad", Game: "fh6", Name: "Bad", Make: "X", Class: "Z", PI: 50, Drivetrain: "6WD"},
		{ID: "fh6-warn", Game: "fh6", Name: "Warn", Make: "695", Class: "A", PI: 650, Drivetrain: "AWD"},
	}
	rep := Validate(cars, "fh6")

	if got := rep.Count(SevBlocking); got != 3 { // pi_range + class_enum + drivetrain_enum
		t.Errorf("blocking count = %d, want 3", got)
	}
	if got := rep.Count(SevWarning); got != 1 { // make_no_letter
		t.Errorf("warning count = %d, want 1", got)
	}
	ids := rep.BlockingIDs()
	if _, ok := ids["fh6-bad"]; !ok || len(ids) != 1 {
		t.Errorf("BlockingIDs = %v, want {fh6-bad}", ids)
	}
	if c := rep.RuleCounts()["blocking:pi_range"]; c != 1 {
		t.Errorf("RuleCounts[blocking:pi_range] = %d, want 1", c)
	}
}

// Objectif : 0 anomalie bloquante sur le catalogue réellement normalisé depuis
// les fixtures. Normalize doit garantir l'invariant ; Validate le prouve.
func TestNoBlockingAnomalyOnCatalog(t *testing.T) {
	rows := loadList(t)
	infoboxes := map[string]Infobox{
		"Abarth 695 Biposto": ParseInfobox(ibAbarth),
		"Ferrari F50":        ParseInfobox(ibFerrari),
		"Honda Civic Type R": ParseInfobox(ibHonda),
	}
	out, _ := Normalize(rows, infoboxes, "fh6")
	if len(out) == 0 {
		t.Fatal("catalogue vide : fixture cassée")
	}

	rep := Validate(out, "fh6")
	if n := rep.Count(SevBlocking); n != 0 {
		t.Fatalf("%d anomalies bloquantes sur le catalogue, want 0: %+v", n, rep.Anomalies)
	}
}
