// Test de la génération de bout en bout SANS DB ni réseau : faux Store (dump
// canné) + Uploader en mémoire. Vérifie qu'on produit bien un fichier par ressource
// × format du registry, que le manifeste reflète chaque upload (URL, taille, etag
// sha256, content-type), que les ressources imbriquées (playlist) n'ont pas de CSV,
// et que CheckRun signale un dataset vide.
package export

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/zinackes/forza-open-api/internal/health"
	"github.com/zinackes/forza-open-api/internal/store"
)

// fakeStore rend un dump canné par ressource (plat sauf playlist) et capture les
// entrées de manifeste écrites.
type fakeStore struct {
	manifest []store.Export
}

func (f *fakeStore) DumpResource(_ context.Context, resource, game string) (store.Dump, error) {
	if resource == "playlist" {
		return store.Dump{Rows: []json.RawMessage{
			json.RawMessage(`{"id":"s1","rewards":[],"challenges":[]}`),
		}}, nil
	}
	return store.Dump{
		Columns: []string{"id", "game"},
		Rows:    []json.RawMessage{json.RawMessage(`{"id":"x","game":"` + game + `"}`)},
	}, nil
}

func (f *fakeStore) UpsertExport(_ context.Context, e store.Export) error {
	f.manifest = append(f.manifest, e)
	return nil
}

// memUploader capture les objets publiés.
type memUploader struct {
	objects map[string][]byte
	types   map[string]string
}

func newMemUploader() *memUploader {
	return &memUploader{objects: map[string][]byte{}, types: map[string]string{}}
}

func (u *memUploader) Put(_ context.Context, key string, body []byte, contentType string) error {
	u.objects[key] = body
	u.types[key] = contentType
	return nil
}

func expectedArtifacts() int {
	n := 0
	for _, s := range registry {
		n += len(s.formats)
	}
	return n
}

func TestGenerate(t *testing.T) {
	st := &fakeStore{}
	up := newMemUploader()
	now := time.Date(2026, 6, 10, 5, 0, 0, 0, time.UTC)

	res, err := Generate(context.Background(), st, up, "https://data.example/", "fh6", now)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	want := expectedArtifacts()
	if res.Artifacts != want {
		t.Errorf("Artifacts = %d, want %d", res.Artifacts, want)
	}
	if len(up.objects) != want || len(st.manifest) != want {
		t.Fatalf("objets=%d manifeste=%d, want %d chacun", len(up.objects), len(st.manifest), want)
	}

	// Chaque entrée de manifeste correspond exactement à l'objet publié.
	for _, e := range st.manifest {
		key := e.Game + "/" + e.Resource + "." + e.Format
		body, ok := up.objects[key]
		if !ok {
			t.Errorf("manifeste %s sans objet uploadé", key)
			continue
		}
		if e.URL != "https://data.example/"+key {
			t.Errorf("%s URL = %q", key, e.URL)
		}
		if e.SizeBytes != int64(len(body)) {
			t.Errorf("%s size = %d, want %d", key, e.SizeBytes, len(body))
		}
		sum := sha256.Sum256(body)
		if e.ETag != hex.EncodeToString(sum[:]) {
			t.Errorf("%s etag ne matche pas le sha256 du contenu", key)
		}
		if !e.GeneratedAt.Equal(now) {
			t.Errorf("%s generatedAt = %v, want %v", key, e.GeneratedAt, now)
		}
		if e.Game != "fh6" {
			t.Errorf("%s game = %q, want fh6", key, e.Game)
		}
	}

	// playlist (imbriquée) : json + jsonl, JAMAIS csv. cars (plate) : les trois.
	assertFormats(t, up, "fh6/playlist", []string{"json", "jsonl"}, []string{"csv"})
	assertFormats(t, up, "fh6/cars", []string{"json", "jsonl", "csv"}, nil)

	// Content-types corrects.
	if ct := up.types["fh6/cars.csv"]; ct != "text/csv; charset=utf-8" {
		t.Errorf("content-type csv = %q", ct)
	}
	if ct := up.types["fh6/cars.jsonl"]; ct != "application/x-ndjson" {
		t.Errorf("content-type jsonl = %q", ct)
	}
	if ct := up.types["fh6/cars.json"]; ct != "application/json" {
		t.Errorf("content-type json = %q", ct)
	}
}

func assertFormats(t *testing.T, up *memUploader, prefix string, present, absent []string) {
	t.Helper()
	for _, f := range present {
		if _, ok := up.objects[prefix+"."+f]; !ok {
			t.Errorf("%s.%s attendu mais absent", prefix, f)
		}
	}
	for _, f := range absent {
		if _, ok := up.objects[prefix+"."+f]; ok {
			t.Errorf("%s.%s ne devrait pas exister", prefix, f)
		}
	}
}

func TestCheckRunEmptyDataset(t *testing.T) {
	// Toutes ressources à 0 ligne → anomalie (jeu non seedé / base vide).
	empty := Result{Rows: map[string]int{"cars": 0, "playlist": 0}}
	vs := CheckRun(empty, time.Now())
	if len(vs) != 1 || vs[0].Severity != health.SevAnomaly || vs[0].Rule != "empty_dataset" {
		t.Fatalf("CheckRun(vide) = %+v, want une anomalie empty_dataset", vs)
	}
	// Au moins une ligne quelque part → run sain.
	if vs := CheckRun(Result{Rows: map[string]int{"cars": 5}}, time.Now()); len(vs) != 0 {
		t.Errorf("CheckRun(non vide) = %+v, want aucune violation", vs)
	}
}
