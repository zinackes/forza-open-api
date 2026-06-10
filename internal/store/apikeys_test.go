// Tests d'intégration des clés API : Postgres jetable (newTestStore, cf.
// integration_test.go). Vérifie le cycle create → lookup par hash, l'invariant
// « jamais de clé en clair en DB », le DEFAULT de rate_limit et la révocation.
package store_test

import (
	"context"
	"slices"
	"testing"

	"github.com/zinackes/forza-open-api/internal/apikey"
)

func TestAPIKeyCreateAndLookup(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	ctx := context.Background()

	plain, err := apikey.Generate()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	hash := apikey.Hash(plain)

	// Invariant sécurité : on ne persiste que le hash, jamais la clé en clair.
	if hash == plain {
		t.Fatal("hash == clé en clair : le hashing ne fait rien")
	}
	if len(hash) != 64 {
		t.Fatalf("hash sha256 hex attendu (64 chars), reçu %d", len(hash))
	}

	if err := st.CreateAPIKey(ctx, hash, "overlay-prod", 5000, []string{"read", "submit-ugc"}); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := st.LookupAPIKey(ctx, hash)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if got == nil {
		t.Fatal("lookup par hash: clé introuvable après création")
	}
	if got.Hash != hash {
		t.Errorf("hash = %q, want %q", got.Hash, hash)
	}
	if got.Name != "overlay-prod" {
		t.Errorf("name = %q, want %q", got.Name, "overlay-prod")
	}
	if got.RateLimit != 5000 {
		t.Errorf("rate_limit = %d, want 5000", got.RateLimit)
	}
	if want := []string{"read", "submit-ugc"}; !slices.Equal(got.Scopes, want) {
		t.Errorf("scopes = %v, want %v", got.Scopes, want)
	}
	if got.RevokedAt != nil {
		t.Errorf("revoked_at = %v, want nil (clé fraîche)", got.RevokedAt)
	}

	// Lookup par la clé en clair (non hashée) ne doit RIEN trouver : le PK est le
	// hash, la clé en clair n'existe nulle part en DB.
	none, err := st.LookupAPIKey(ctx, plain)
	if err != nil {
		t.Fatalf("lookup clé en clair: %v", err)
	}
	if none != nil {
		t.Error("la clé en clair a matché une ligne : elle ne devrait jamais être en DB")
	}
}

func TestAPIKeyRateLimitDefault(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	ctx := context.Background()

	plain, err := apikey.Generate()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	hash := apikey.Hash(plain)

	// rateLimit <= 0 et scopes nil → colonnes omises : DEFAULT du schéma (rate_limit
	// 1000, scopes {read}) s'applique.
	if err := st.CreateAPIKey(ctx, hash, "default-rl", 0, nil); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := st.LookupAPIKey(ctx, hash)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if got == nil || got.RateLimit != 1000 {
		t.Fatalf("rate_limit par défaut attendu 1000, reçu %+v", got)
	}
	if want := []string{"read"}; !slices.Equal(got.Scopes, want) {
		t.Errorf("scopes par défaut = %v, want %v", got.Scopes, want)
	}
}

func TestAPIKeyRevoke(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	ctx := context.Background()

	plain, err := apikey.Generate()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	hash := apikey.Hash(plain)
	if err := st.CreateAPIKey(ctx, hash, "to-revoke", 0, nil); err != nil {
		t.Fatalf("create: %v", err)
	}

	found, err := st.RevokeAPIKey(ctx, hash)
	if err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if !found {
		t.Fatal("revoke d'une clé existante: found=false")
	}

	got, err := st.LookupAPIKey(ctx, hash)
	if err != nil {
		t.Fatalf("lookup après revoke: %v", err)
	}
	if got == nil || got.RevokedAt == nil {
		t.Fatalf("revoked_at devrait être renseigné après révocation, reçu %+v", got)
	}

	// Révoquer un hash inconnu : found=false, pas d'erreur.
	missing, err := st.RevokeAPIKey(ctx, apikey.Hash("inexistante"))
	if err != nil {
		t.Fatalf("revoke hash inconnu: %v", err)
	}
	if missing {
		t.Error("revoke d'un hash inconnu devrait renvoyer found=false")
	}
}
