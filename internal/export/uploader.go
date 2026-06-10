// Package export génère les archives téléchargeables du dataset (GET /v1/exports) :
// un fichier statique par jeu, ressource et format (JSON/CSV/JSONL), publié vers
// un object store / l'edge et enregistré au manifeste (table exports). Le job est
// rejouable (upsert idempotent) ; il tourne via cmd/seed (ponctuel) et
// cmd/scheduler (cron quotidien). Sérialisation déléguée à Postgres (store.Dump,
// to_jsonb) ; ce package orchestre encodage → upload → manifeste, plus le contrôle
// de santé (parité avec internal/ingest/*).
package export

import (
	"context"
	"fmt"
)

// Uploader publie le contenu d'une archive sous une clé (« fh6/cars.csv »). body
// est le fichier complet déjà sérialisé ; contentType son type MIME. Deux
// implémentations : LocalFS (dev/tests, servi en statique derrière Cloudflare) et
// R2 (prod, S3 API). L'appelant calcule l'URL publique à partir d'un baseURL + clé.
type Uploader interface {
	Put(ctx context.Context, key string, body []byte, contentType string) error
}

// UploaderFor choisit le backend : R2 si l'endpoint ET le bucket sont configurés
// (prod), sinon le filesystem local (dev / défaut, servi en statique derrière
// Cloudflare). Garde le câblage backend hors des commandes (cmd/seed, cmd/scheduler).
//
// Une config R2 PARTIELLE est une erreur au démarrage plutôt qu'un échec différé
// au premier upload du cron (audit 2026-06-10) : endpoint/bucket dépareillés, ou
// credentials manquants alors que R2 est visé.
func UploaderFor(r2 R2Config, localDir string) (Uploader, error) {
	if r2.Endpoint == "" && r2.Bucket == "" {
		return LocalFS{Dir: localDir}, nil
	}
	switch {
	case r2.Endpoint == "" || r2.Bucket == "":
		return nil, fmt.Errorf("config R2 partielle: R2_ENDPOINT et R2_BUCKET vont ensemble (endpoint=%q, bucket=%q)", r2.Endpoint, r2.Bucket)
	case r2.AccessKeyID == "" || r2.SecretAccessKey == "":
		return nil, fmt.Errorf("config R2 incomplète: R2_ACCESS_KEY_ID / R2_SECRET_ACCESS_KEY requis quand R2 est configuré")
	}
	return NewR2(r2), nil
}
