---
name: ingestion-source
description: Ajouter une source d'ingestion propre (catalogue ou playlist) conforme à la doctrine zéro gris. À utiliser pour brancher une nouvelle source de données.
---
# Ajouter une source d'ingestion
1. Vérifie la doctrine (docs/DATA-SOURCES.md) : source propre ? API officielle (Strapi, MediaWiki) avant tout scrape HTML ? EAC-safe ?
2. Implémente dans internal/ingest/<source> : fetch (user-agent identifiable, délais polis, retries), parse (structure connue), normalise vers le schéma (game !).
3. Upsert idempotent (ON CONFLICT), rejouable sans doublon. Aucune donnée inventée (NULL si absent).
4. Logue un rapport (importés/ignorés/anomalies). Branche au scheduler si récurrent.
5. Test le parsing sur des fixtures figées (pas de réseau en test). Commit conventionnel.