package export

import (
	"bytes"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// R2Config porte les paramètres de connexion à un bucket Cloudflare R2 (API S3).
type R2Config struct {
	Endpoint        string // https://<accountid>.r2.cloudflarestorage.com
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
}

// R2 publie les archives sur Cloudflare R2 via l'API S3. Backend de prod (lecture
// publique servie par un domaine R2 + cache Cloudflare). SEUL ce fichier dépend de
// l'aws-sdk : le reste du package — et tous les tests, qui utilisent LocalFS ou un
// faux — n'en a pas besoin, gardant la CI sans réseau.
type R2 struct {
	client *s3.Client
	bucket string
}

// NewR2 construit un client R2 à partir de credentials statiques. region = "auto"
// (R2 ignore la région S3) ; path-style pour la compat avec un endpoint custom.
func NewR2(cfg R2Config) *R2 {
	client := s3.New(s3.Options{
		Region:       "auto",
		BaseEndpoint: aws.String(cfg.Endpoint),
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		UsePathStyle: true,
	})
	return &R2{client: client, bucket: cfg.Bucket}
}

func (r *R2) Put(ctx context.Context, key string, body []byte, contentType string) error {
	if _, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(body),
		ContentType: aws.String(contentType),
	}); err != nil {
		return fmt.Errorf("r2 put %s: %w", key, err)
	}
	return nil
}
