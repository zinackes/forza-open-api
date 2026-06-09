// Command seed orchestre l'ingestion du catalogue depuis des sources PROPRES
// (datasets communautaires, exports wiki Fandom). Jamais de scraping du jeu,
// lecture mémoire, injection ni botting. Upserts idempotents (ON CONFLICT),
// rejouables sans doublon.
//
// Usage :
//
//	seed tracks <dataset.json | https://…>   ingère un dataset de tracés.
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/zinackes/forza-open-api/internal/config"
	"github.com/zinackes/forza-open-api/internal/ingest/tracks"
	"github.com/zinackes/forza-open-api/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if len(os.Args) < 3 || os.Args[1] != "tracks" {
		logger.Info("usage: seed tracks <dataset.json | URL> — ingestion idempotente d'un dataset de tracés")
		return
	}
	src := os.Args[2]

	ctx := context.Background()

	raw, err := tracks.FetchDataset(ctx, src)
	if err != nil {
		logger.Error("fetch dataset", "src", src, "err", err)
		os.Exit(1)
	}
	parsed, skipped, err := tracks.ParseTracks(raw, time.Now().UTC())
	if err != nil {
		logger.Error("parse dataset", "src", src, "err", err)
		os.Exit(1)
	}

	st, err := store.New(ctx, config.Load())
	if err != nil {
		logger.Error("store init", "err", err)
		os.Exit(1)
	}
	defer st.Close()

	if err := st.UpsertTracks(ctx, parsed); err != nil {
		logger.Error("upsert tracks", "err", err)
		os.Exit(1)
	}
	logger.Info("seed tracks ok", "upserted", len(parsed), "skipped", skipped, "src", src)
}
