// Le contrat est aussi servi par l'API elle-même (GET /openapi.yaml) : les
// clients, outils (Postman, générateurs) et assistants IA le découvrent sans
// passer par le dépôt.
package api

import _ "embed"

//go:embed openapi.yaml
var OpenAPI []byte
