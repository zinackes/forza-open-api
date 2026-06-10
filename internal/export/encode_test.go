// Tests des encodeurs (fonctions pures, fixtures figées, sans I/O) : c'est le cœur
// de confiance des archives. Vérifie le JSON tableau, le JSONL, et la projection
// CSV (NULL → vide, chaîne déquotée, JSONB imbriqué en cellule, échappement csv).
package export

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/zinackes/forza-open-api/internal/store"
)

func raw(s string) json.RawMessage { return json.RawMessage(s) }

func TestEncodeJSON(t *testing.T) {
	rows := []json.RawMessage{raw(`{"id":"a","n":1}`), raw(`{"id":"b","n":2}`)}
	got := string(encodeJSON(rows))
	want := `[{"id":"a","n":1},{"id":"b","n":2}]` + "\n"
	if got != want {
		t.Errorf("encodeJSON = %q, want %q", got, want)
	}
	// Tableau vide : un dataset sans ligne reste un JSON valide « [] ».
	if got := string(encodeJSON(nil)); got != "[]\n" {
		t.Errorf("encodeJSON(nil) = %q, want %q", got, "[]\n")
	}
}

func TestEncodeJSONL(t *testing.T) {
	rows := []json.RawMessage{raw(`{"id":"a"}`), raw(`{"id":"b"}`)}
	got := string(encodeJSONL(rows))
	want := "{\"id\":\"a\"}\n{\"id\":\"b\"}\n"
	if got != want {
		t.Errorf("encodeJSONL = %q, want %q", got, want)
	}
	if got := string(encodeJSONL(nil)); got != "" {
		t.Errorf("encodeJSONL(nil) = %q, want empty", got)
	}
}

func TestEncodeCSV(t *testing.T) {
	d := store.Dump{
		Columns: []string{"id", "name", "pi", "stats", "active"},
		Rows: []json.RawMessage{
			raw(`{"id":"car-1","name":"Quartz, Regalia","pi":850,"stats":{"power":300},"active":true}`),
			raw(`{"id":"car-2","name":null,"pi":null,"stats":null,"active":false}`),
		},
	}
	got, err := encodeCSV(d)
	if err != nil {
		t.Fatalf("encodeCSV: %v", err)
	}
	// name contient une virgule → quoté ; stats imbriqué → JSON compact en cellule
	// (quotes doublées) ; NULL → cellule vide ; booléens en texte brut.
	want := "id,name,pi,stats,active\n" +
		"car-1,\"Quartz, Regalia\",850,\"{\"\"power\"\":300}\",true\n" +
		"car-2,,,,false\n"
	if string(got) != want {
		t.Errorf("encodeCSV =\n%q\nwant\n%q", got, want)
	}
}

func TestEncodeCSVRejectsNested(t *testing.T) {
	// Columns nil = ressource imbriquée (playlist) → pas de CSV.
	if _, err := encodeCSV(store.Dump{Rows: []json.RawMessage{raw(`{"id":"s1"}`)}}); err == nil {
		t.Error("encodeCSV sans colonnes : erreur attendue")
	}
}

func TestCsvCell(t *testing.T) {
	cases := map[string]string{
		`null`:             "",
		`"hello"`:          "hello",
		`"with \"quote\""`: `with "quote"`,
		`42`:               "42",
		`1.5`:              "1.5",
		`true`:             "true",
		`{"k":1}`:          `{"k":1}`,
		`[1,2]`:            `[1,2]`,
	}
	for in, want := range cases {
		if got := csvCell(raw(in)); got != want {
			t.Errorf("csvCell(%s) = %q, want %q", in, got, want)
		}
	}
	// RawMessage vide (colonne absente de l'objet) → cellule vide.
	if got := csvCell(nil); got != "" {
		t.Errorf("csvCell(nil) = %q, want empty", got)
	}
}

func TestEncodeDispatch(t *testing.T) {
	d := store.Dump{Columns: []string{"id"}, Rows: []json.RawMessage{raw(`{"id":"x"}`)}}
	for _, f := range []string{FormatJSON, FormatJSONL, FormatCSV} {
		if _, err := encode(f, d); err != nil {
			t.Errorf("encode(%s): %v", f, err)
		}
	}
	if _, err := encode("xml", d); err == nil {
		t.Error("encode(xml) : format inconnu, erreur attendue")
	}
}

// TestEtagStability documente l'invariant ETag : hash de contenu déterministe →
// deux encodages identiques produisent le même etag (revalidation conditionnelle).
func TestEtagStability(t *testing.T) {
	d := store.Dump{Columns: []string{"id"}, Rows: []json.RawMessage{raw(`{"id":"x"}`)}}
	b1, _ := encode(FormatCSV, d)
	b2, _ := encode(FormatCSV, d)
	h1, h2 := sha256.Sum256(b1), sha256.Sum256(b2)
	if hex.EncodeToString(h1[:]) != hex.EncodeToString(h2[:]) {
		t.Error("etag non déterministe pour un même contenu")
	}
}
