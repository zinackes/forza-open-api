---
name: new-endpoint
description: Ajouter un endpoint en contract-first (OpenAPI -> ogen -> handler -> test). À utiliser dès qu'on veut exposer une nouvelle route de l'API.
---
# Ajouter un endpoint (contract-first)
1. Contrat : édite api/openapi.yaml — path + opération (operationId, x-ogen-operation-group), schémas, paramètres (game requis si multi-jeux, pagination), réponses (200 + erreurs RFC 9457).
2. task generate (ogen) — vérifie que internal/oas compile.
3. Handler : implémente la méthode de oas.Handler dans internal/handler (pgx paramétré, mapping, erreurs typées).
4. Cache : ajoute Cache-Control/ETag selon la nature (statique vs volatile).
5. Test : table-test du handler (nominal + 404/400 + pagination).
6. curl + task test. Commit conventionnel. Ne code jamais l'endpoint avant le contrat.