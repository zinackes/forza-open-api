// Package api porte la directive de génération du code serveur/clients à partir
// du contrat OpenAPI. Le contrat (openapi.yaml) est la source de vérité ; le
// code dans internal/oas en découle et ne doit jamais être édité à la main.
package api

//go:generate go tool ogen --target ../internal/oas --package oas --clean openapi.yaml
