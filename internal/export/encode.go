package export

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"

	"github.com/zinackes/forza-open-api/internal/store"
)

// Formats de sérialisation supportés.
const (
	FormatJSON  = "json"
	FormatJSONL = "jsonl"
	FormatCSV   = "csv"
)

// encode sérialise un Dump dans le format demandé. Fonctions pures (pas d'I/O) :
// le cœur testable sur fixtures figées.
func encode(format string, d store.Dump) ([]byte, error) {
	switch format {
	case FormatJSON:
		return encodeJSON(d.Rows), nil
	case FormatJSONL:
		return encodeJSONL(d.Rows), nil
	case FormatCSV:
		return encodeCSV(d)
	default:
		return nil, fmt.Errorf("format inconnu %q", format)
	}
}

// encodeJSON produit un tableau JSON compact des objets-lignes (les RawMessage
// sont déjà du JSON valide produit par to_jsonb).
func encodeJSON(rows []json.RawMessage) []byte {
	var b bytes.Buffer
	b.WriteByte('[')
	for i, r := range rows {
		if i > 0 {
			b.WriteByte(',')
		}
		b.Write(r)
	}
	b.WriteString("]\n")
	return b.Bytes()
}

// encodeJSONL produit un objet JSON par ligne (newline-delimited) : pratique pour
// le streaming et les outils data (jq, pandas read_json lines=True).
func encodeJSONL(rows []json.RawMessage) []byte {
	var b bytes.Buffer
	for _, r := range rows {
		b.Write(r)
		b.WriteByte('\n')
	}
	return b.Bytes()
}

// encodeCSV projette les objets-lignes sur les colonnes ordonnées de la table.
// Refuse une ressource imbriquée (Columns nil). encoding/csv gère l'échappement.
func encodeCSV(d store.Dump) ([]byte, error) {
	if d.Columns == nil {
		return nil, fmt.Errorf("csv: ressource imbriquée (pas de colonnes tabulaires)")
	}
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	if err := w.Write(d.Columns); err != nil {
		return nil, fmt.Errorf("csv header: %w", err)
	}
	for _, raw := range d.Rows {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(raw, &obj); err != nil {
			return nil, fmt.Errorf("csv: ligne illisible: %w", err)
		}
		rec := make([]string, len(d.Columns))
		for i, col := range d.Columns {
			rec[i] = csvCell(obj[col])
		}
		if err := w.Write(rec); err != nil {
			return nil, fmt.Errorf("csv row: %w", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("csv flush: %w", err)
	}
	return b.Bytes(), nil
}

// csvCell rend une valeur JSON en cellule CSV : NULL → vide, chaîne → déquotée,
// nombre/booléen → texte brut, objet/tableau imbriqué (JSONB stats, geojson) →
// JSON compact tel quel.
func csvCell(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			return s
		}
	}
	return string(raw)
}
