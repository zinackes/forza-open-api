package export

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LocalFS écrit les archives sous Dir (un fichier par clé). Sert au dev et aux
// tests, et de backend par défaut quand R2 n'est pas configuré : le dossier est
// alors exposé en statique par l'API (derrière le cache Cloudflare). Les clés sont
// internes (jeu + ressource + extension) ; on refuse tout « .. » par prudence.
type LocalFS struct{ Dir string }

func (l LocalFS) Put(_ context.Context, key string, body []byte, _ string) error {
	if strings.Contains(key, "..") {
		return fmt.Errorf("clé d'archive invalide: %q", key)
	}
	path := filepath.Join(l.Dir, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
