// Command seed orchestre l'ingestion du catalogue depuis des sources propres
// (datasets/GitHub/wiki). Stub Phase 0 : aucune source branchée pour l'instant.
//
// Doctrine zéro gris : jamais de scraping du jeu, lecture mémoire, injection ni
// botting. Upserts idempotents (ON CONFLICT), rejouables sans doublon.
package main

import (
	"log/slog"
	"os"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("seed: aucune source branchée (stub phase 0)")
}
