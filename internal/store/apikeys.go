// Accès à la table api_keys. On ne stocke et ne manipule QUE le hash sha256 :
// la clé en clair n'entre jamais ici (cf. internal/apikey). Requêtes paramétrées.
package store

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// APIKey est une ligne de api_keys. Hash est le sha256 hex (PK) ; jamais la clé
// en clair. RevokedAt est nil tant que la clé est active.
type APIKey struct {
	Hash      string
	Name      string
	RateLimit int
	Scopes    []string
	CreatedAt time.Time
	RevokedAt *time.Time
}

// CreateAPIKey insère une clé par son hash. rateLimit <= 0 et scopes vide omettent
// la colonne correspondante pour laisser le DEFAULT du schéma faire foi (rate_limit
// 1000, scopes {read}). Échoue si le hash existe (PK). Colonnes statiques + valeurs
// paramétrées : aucune concaténation de donnée dans le SQL.
func (s *Store) CreateAPIKey(ctx context.Context, hash, name string, rateLimit int, scopes []string) error {
	cols := []string{"key_hash", "name"}
	ph := []string{"$1", "$2"}
	args := []any{hash, name}
	if rateLimit > 0 {
		args = append(args, rateLimit)
		cols = append(cols, "rate_limit")
		ph = append(ph, "$"+strconv.Itoa(len(args)))
	}
	if len(scopes) > 0 {
		args = append(args, scopes)
		cols = append(cols, "scopes")
		ph = append(ph, "$"+strconv.Itoa(len(args)))
	}
	q := fmt.Sprintf("INSERT INTO api_keys (%s) VALUES (%s)",
		strings.Join(cols, ", "), strings.Join(ph, ", "))
	if _, err := s.DB.Exec(ctx, q, args...); err != nil {
		return fmt.Errorf("insert api key: %w", err)
	}
	return nil
}

// LookupAPIKey renvoie la clé par son hash (révoquée ou non : RevokedAt renseigne
// le statut, au caller de décider). Renvoie (nil, nil) si le hash est inconnu.
func (s *Store) LookupAPIKey(ctx context.Context, hash string) (*APIKey, error) {
	const q = `
SELECT key_hash, name, rate_limit, scopes, created_at, revoked_at
FROM api_keys
WHERE key_hash = $1`
	var k APIKey
	err := s.DB.QueryRow(ctx, q, hash).Scan(&k.Hash, &k.Name, &k.RateLimit, &k.Scopes, &k.CreatedAt, &k.RevokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query api key: %w", err)
	}
	return &k, nil
}

// RevokeAPIKey marque une clé révoquée (idempotent : ne réécrit pas revoked_at si
// déjà révoquée). Renvoie false si le hash est inconnu.
func (s *Store) RevokeAPIKey(ctx context.Context, hash string) (bool, error) {
	const q = `UPDATE api_keys SET revoked_at = now() WHERE key_hash = $1 AND revoked_at IS NULL`
	tag, err := s.DB.Exec(ctx, q, hash)
	if err != nil {
		return false, fmt.Errorf("revoke api key: %w", err)
	}
	if tag.RowsAffected() > 0 {
		return true, nil
	}
	// 0 ligne : soit le hash n'existe pas, soit elle était déjà révoquée. On
	// distingue pour ne pas mentir à l'admin (« introuvable » vs « déjà révoquée »).
	k, err := s.LookupAPIKey(ctx, hash)
	if err != nil {
		return false, err
	}
	return k != nil, nil
}

// ListAPIKeys renvoie toutes les clés (actives et révoquées), plus récentes
// d'abord. Ne renvoie jamais de clé en clair (impossible : non stockée).
func (s *Store) ListAPIKeys(ctx context.Context) ([]APIKey, error) {
	const q = `
SELECT key_hash, name, rate_limit, scopes, created_at, revoked_at
FROM api_keys
ORDER BY created_at DESC`
	rows, err := s.DB.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query api keys: %w", err)
	}
	defer rows.Close()

	var out []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.Hash, &k.Name, &k.RateLimit, &k.Scopes, &k.CreatedAt, &k.RevokedAt); err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		out = append(out, k)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate api keys: %w", err)
	}
	return out, nil
}
