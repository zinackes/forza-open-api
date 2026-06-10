# Changelog du contrat

Journal des changements de l'API **contract-first** (`api/openapi.yaml`).
Format [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/), versionné en
[SemVer](https://semver.org/lang/fr/). La version vit dans `info.version`.

Rappel de la politique (→ `docs/API.md`) :

- **MAJOR** = breaking change → nouveau préfixe d'URL (`/v2`), livré en additif ;
  l'ancien major reste servi le temps de la fenêtre de dépréciation.
- **MINOR** = ajout rétro-compatible (endpoint, champ/param optionnel, valeur
  d'enum) — reste dans `/v1`.
- **PATCH** = doc, exemples, descriptions — aucun changement de comportement.

Sections par entrée : `Added` / `Changed` / `Deprecated` / `Removed` / `Fixed`.
Les entrées `Deprecated` / `Removed` rappellent les dates `Deprecation` / `Sunset`.

## [Unreleased]

## [1.0.0] — 2026-06-09

- Contrat initial publié : préfixe `/v1`, ressources `cars`, `manufacturers`,
  `playlist`, `forzathon-shop`, `tracks` et transverses (`search`, `changes`,
  `reference`, `meta`). Erreurs RFC 9457, pagination `page`/`page_size`, champ
  `game` sur les ressources multi-jeux.

[Unreleased]: https://github.com/zinackes/forza-open-api/compare/contract-v1.0.0...HEAD
[1.0.0]: https://github.com/zinackes/forza-open-api/releases/tag/contract-v1.0.0
