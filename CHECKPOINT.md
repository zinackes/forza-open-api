# Checkpoint — 2026-06-10 · feat/api-expansion

## Contexte
Revue produit (endpoints/features manquants, dédupliquée contre le Notion) puis
implémentation complète sur la branche `feat/api-expansion` (basée sur
`feat/cars-seeder`, dont le WIP audit a été commité en 5 commits propres).

## Fait (commits sur feat/api-expansion)
- `4b72f18` contrat + ogen + db : gets par id (tracks/events/pr-stunts/
  barn-finds/treasure-cars/dlc-packs), q/car_id/kind/country/updated_since,
  facettes regions+types dans /v1/reference, /v1/cars/{id}/obtain, /v1/search,
  /v1/changes, /v1/tracks/random, EventType+=showcase, /v1/stories, /v1/tours,
  examples OpenAPI. Tables stories/tours/data_changes, index trigram, cars.updated_at.
- `463c298` store + handlers : tout implémenté ; UpsertCars journalise
  data_changes (même tx, no-op si identique via IS DISTINCT FROM).
- `53a2e1a` tests : intégration store (changes, search, obtain, car_id,
  updated_since, facettes geo, stories/tours) + 16 goldens handler.
- (en cours de commit) llms.txt + GET /openapi.yaml servis par l'API, docs.

## Décisions
- Historisation PI/stats (`as_of`) **non faite** : exploratoire phase 9 ;
  /v1/changes pose la fondation.
- Serveur MCP **non fait** (déployable séparé) ; llms.txt + /openapi.yaml
  livrés comme socle découvrabilité IA.
- Goldens : tous les champs timestamp normalisés `<ts>` (pgx scanne les
  timestamptz en time.Local → offset machine-dépendant).

## Reste à faire
- Notion : consigner les items implémentés dans « Backlog validé » (l'écriture
  a été refusée par les permissions ; note prête ci-dessous). Carte 9.4
  vérifiée : mascots/landmarks/murals déjà dans son scope — rien à changer.

### Note à coller dans « Backlog validé — à promouvoir en cartes »

> **🚢 Implémenté le 2026-06-10 — expansion du contrat (branche `feat/api-expansion`)**
> - GET par id : tracks, events, pr-stunts, barn-finds, treasure-cars, dlc-packs (404 RFC 9457)
> - Filtres : q (tracks/events/pr-stunts/manufacturers), car_id (barn-finds/treasure-cars), kind (dlc-packs), country (manufacturers), updated_since (cars + tracks)
> - /v1/reference enrichi : regions + trackTypes + eventTypes + prStuntTypes
> - /v1/cars/{id}/obtain (autoshow, DLC, barn find, treasure, journal, mastery)
> - /v1/search (autocomplete multi-ressources) · /v1/changes (journal de données, alimenté par le seed cars) · /v1/tracks/random
> - showcase dans EventType · /v1/stories + /v1/tours (seeders à venir)
> - /openapi.yaml + /llms.txt servis par l'API · examples OpenAPI
> Non fait (volontaire) : historisation PI (as_of) → phase 9 ; serveur MCP → carte à arbitrer.
- Seeders stories/tours (wiki Fandom) — endpoints répondent vide en attendant.
- Prochaine itération demandée par Mathys : re-chercher de la valeur ajoutée
  (fonctionnalités, endpoints, paramètres) une fois tout ceci mergé.
