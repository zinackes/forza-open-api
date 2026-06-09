// Command keys gère les clés API (génération, hash sha256, révocation).
// Stub Phase 0 : la clé n'est jamais stockée ni loggée en clair (seul le hash
// est persisté). Implémentation à venir avec la table api_keys.
package main

import (
	"log/slog"
	"os"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("keys: gestion des clés API (stub phase 0)")
}
