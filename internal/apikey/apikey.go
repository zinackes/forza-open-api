// Package apikey génère et hashe les clés API. La clé en clair n'existe qu'à la
// création (montrée une seule fois) : seul son hash sha256 est persisté et
// comparé. Génération et hash sont centralisés ici car le middleware d'auth
// hashera la clé entrante avec la même Hash pour retrouver la ligne en DB.
package apikey

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
)

// keyBytes est l'entropie brute d'une clé (256 bits), encodée ensuite en base62.
const keyBytes = 32

// Generate renvoie une clé opaque : 32 octets aléatoires (crypto/rand) encodés
// en base62 (alphabet 0-9a-zA-Z, via big.Int, sans biais). C'est la SEULE forme
// en clair de la clé ; l'appelant la montre une fois puis ne persiste que Hash.
func Generate() (string, error) {
	raw := make([]byte, keyBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return new(big.Int).SetBytes(raw).Text(62), nil
}

// Hash renvoie le sha256 hex d'une clé en clair. Déterministe : sert à l'insert
// (création) comme au lookup (auth). Jamais l'inverse — la clé ne se déduit pas
// du hash.
func Hash(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}
