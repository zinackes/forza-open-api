// Command keys gère les clés API : génération, révocation, listing.
//
// La clé en clair n'est JAMAIS stockée ni loggée : seul son hash sha256 est
// persisté. À la création, la clé en clair est affichée UNE seule fois sur
// stdout — l'admin la copie immédiatement, elle est ensuite irrécupérable.
//
// Usage :
//
//	keys create <name> [rate_limit]   génère une clé, affiche la clé en clair 1x.
//	keys list                         liste les clés (hash, jamais la clé).
//	keys revoke <key_hash>            révoque une clé par son hash (cf. list).
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/zinackes/forza-open-api/internal/apikey"
	"github.com/zinackes/forza-open-api/internal/config"
	"github.com/zinackes/forza-open-api/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch os.Args[1] {
	case "create":
		createKey(ctx, logger)
	case "list":
		listKeys(ctx, logger)
	case "revoke":
		revokeKey(ctx, logger)
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: keys create <name> [rate_limit] | keys list | keys revoke <key_hash>")
}

func createKey(ctx context.Context, logger *slog.Logger) {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: keys create <name> [rate_limit]")
		os.Exit(2)
	}
	name := os.Args[2]

	rateLimit := 0 // 0 → DEFAULT du schéma (1000)
	if len(os.Args) >= 4 {
		n, err := strconv.Atoi(os.Args[3])
		if err != nil || n <= 0 {
			fmt.Fprintf(os.Stderr, "rate_limit invalide: %q (entier > 0 attendu)\n", os.Args[3])
			os.Exit(2)
		}
		rateLimit = n
	}

	plain, err := apikey.Generate()
	if err != nil {
		logger.Error("génération de clé", "err", err)
		os.Exit(1)
	}
	hash := apikey.Hash(plain)

	st := mustStore(ctx, logger)
	defer st.Close()

	if err := st.CreateAPIKey(ctx, hash, name, rateLimit); err != nil {
		logger.Error("création de clé", "name", name, "err", err)
		os.Exit(1)
	}

	// La clé en clair sort UNE seule fois, sur stdout, hors logs. Irrécupérable
	// ensuite (seul le hash est en DB).
	fmt.Println(plain)
	logger.Info("clé créée", "name", name, "key_hash", hash) // hash, jamais la clé en clair
}

func listKeys(ctx context.Context, logger *slog.Logger) {
	st := mustStore(ctx, logger)
	defer st.Close()

	keys, err := st.ListAPIKeys(ctx)
	if err != nil {
		logger.Error("listing des clés", "err", err)
		os.Exit(1)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "KEY_HASH\tNAME\tRATE_LIMIT\tCREATED_AT\tSTATUS")
	for _, k := range keys {
		status := "active"
		if k.RevokedAt != nil {
			status = "revoked " + k.RevokedAt.UTC().Format(time.RFC3339)
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
			k.Hash, k.Name, k.RateLimit, k.CreatedAt.UTC().Format(time.RFC3339), status)
	}
	_ = w.Flush()
}

func revokeKey(ctx context.Context, logger *slog.Logger) {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: keys revoke <key_hash>")
		os.Exit(2)
	}
	hash := os.Args[2]

	st := mustStore(ctx, logger)
	defer st.Close()

	found, err := st.RevokeAPIKey(ctx, hash)
	if err != nil {
		logger.Error("révocation de clé", "key_hash", hash, "err", err)
		os.Exit(1)
	}
	if !found {
		logger.Error("clé introuvable", "key_hash", hash)
		os.Exit(1)
	}
	logger.Info("clé révoquée", "key_hash", hash)
}

func mustStore(ctx context.Context, logger *slog.Logger) *store.Store {
	st, err := store.New(ctx, config.Load())
	if err != nil {
		logger.Error("store init", "err", err)
		os.Exit(1)
	}
	return st
}
