// Tests d'UpsertManufacturers sur Postgres jetable (testcontainers, db/init.sql).
// Vérifie l'idempotence (clé naturelle game+name), le COALESCE (un re-run sans
// code pays ne détruit pas la valeur acquise) et la mise à jour effective.
package store_test

import (
	"context"
	"testing"

	"github.com/zinackes/forza-open-api/internal/store"
)

func TestUpsertManufacturers(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	ita, jpn := "ita", "jpn"
	first := []store.Manufacturer{
		{Game: "fh6", Name: "Abarth", Country: &ita},
		{Game: "fh6", Name: "Honda", Country: nil}, // pas encore de code sourcé
	}
	if err := st.UpsertManufacturers(ctx, first); err != nil {
		t.Fatalf("upsert initial: %v", err)
	}

	list := func() map[string]*string {
		t.Helper()
		rows, err := st.ListManufacturers(ctx, store.ManufacturerFilter{Game: "fh6"})
		if err != nil {
			t.Fatalf("list manufacturers: %v", err)
		}
		out := make(map[string]*string, len(rows))
		for _, m := range rows {
			out[m.Name] = m.Country
		}
		return out
	}

	got := list()
	if len(got) != 2 {
		t.Fatalf("manufacturers = %d, want 2 (%v)", len(got), got)
	}
	if c := got["Abarth"]; c == nil || *c != "ita" {
		t.Errorf("Abarth: country = %v, want ita", c)
	}
	if c := got["Honda"]; c != nil {
		t.Errorf("Honda: country = %v, want nil", c)
	}

	// Re-run : Honda gagne son code, Abarth perd le sien dans la source →
	// COALESCE conserve ita (jamais de régression de donnée) ; pas de doublon.
	second := []store.Manufacturer{
		{Game: "fh6", Name: "Abarth", Country: nil},
		{Game: "fh6", Name: "Honda", Country: &jpn},
	}
	if err := st.UpsertManufacturers(ctx, second); err != nil {
		t.Fatalf("upsert re-run: %v", err)
	}

	got = list()
	if len(got) != 2 {
		t.Fatalf("après re-run: manufacturers = %d, want 2 (pas de doublon)", len(got))
	}
	if c := got["Abarth"]; c == nil || *c != "ita" {
		t.Errorf("Abarth après re-run: country = %v, want ita (COALESCE)", c)
	}
	if c := got["Honda"]; c == nil || *c != "jpn" {
		t.Errorf("Honda après re-run: country = %v, want jpn", c)
	}

	// Slice vide : no-op sans erreur.
	if err := st.UpsertManufacturers(ctx, nil); err != nil {
		t.Fatalf("upsert vide: %v", err)
	}
}
