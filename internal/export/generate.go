package export

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/zinackes/forza-open-api/internal/store"
)

// Store est le sous-ensemble de *store.Store dont la génération a besoin (lecture
// brute + écriture du manifeste). Interface pour injecter un faux en test (pas de DB).
type Store interface {
	DumpResource(ctx context.Context, resource, game string) (store.Dump, error)
	UpsertExport(ctx context.Context, e store.Export) error
}

// Result résume une passe de génération (log + contrôle de santé).
type Result struct {
	Artifacts int            // fichiers générés (ressource × format)
	Rows      map[string]int // ressource → nombre d'enregistrements
}

// Generate régénère toutes les archives d'un jeu : pour chaque ressource du
// registry et chaque format supporté, sérialise le dump, le publie via up et
// enregistre l'entrée au manifeste (upsert idempotent). baseURL préfixe l'URL
// publique des fichiers ; now horodate les entrées. Rejouable sans doublon. Toute
// erreur (dump / encodage / upload / manifeste) interrompt la passe.
func Generate(ctx context.Context, st Store, up Uploader, baseURL, game string, now time.Time) (Result, error) {
	base := strings.TrimRight(baseURL, "/")
	res := Result{Rows: make(map[string]int, len(registry))}

	for _, spec := range registry {
		dump, err := st.DumpResource(ctx, spec.resource, game)
		if err != nil {
			return Result{}, fmt.Errorf("dump %s/%s: %w", game, spec.resource, err)
		}
		res.Rows[spec.resource] = len(dump.Rows)

		for _, format := range spec.formats {
			body, err := encode(format, dump)
			if err != nil {
				return Result{}, fmt.Errorf("encode %s/%s.%s: %w", game, spec.resource, format, err)
			}
			key := game + "/" + spec.resource + "." + ext(format)
			if err := up.Put(ctx, key, body, contentType(format)); err != nil {
				return Result{}, fmt.Errorf("upload %s: %w", key, err)
			}
			sum := sha256.Sum256(body)
			if err := st.UpsertExport(ctx, store.Export{
				Game:        game,
				Resource:    spec.resource,
				Format:      format,
				URL:         base + "/" + key,
				SizeBytes:   int64(len(body)),
				ETag:        hex.EncodeToString(sum[:]),
				GeneratedAt: now,
			}); err != nil {
				return Result{}, fmt.Errorf("manifeste %s: %w", key, err)
			}
			res.Artifacts++
		}
	}
	return res, nil
}
