## Mission

Donner à la communauté Forza l'API de données ouverte qu'elle n'a jamais eue. Plaisir + utilité, **pas de monétisation**.

## Non négociables

1. **Zéro gris.** Sources propres uniquement (cf. `docs/DATA-SOURCES.md`). Jamais de scraping du jeu, lecture mémoire, injection, botting.
2. **EAC-safe.** Captures en lecture seule, hors-process. Doute → non.
3. **Contract-first.** Le contrat OpenAPI précède le code.
4. **Game-agnostic.** `game` partout.
5. **~0€.** Cheap par conception.
6. **Respect des sources.** User-agent identifiable, délais polis, pas de hammering ; API officielles (Strapi, MediaWiki) > scrape HTML.

## Qualité des données

**Précision > exhaustivité.** On préfère un champ NULL à une valeur inventée. Réconciliation via les ordinals télémétrie quand dispo.

## Éthique

Projet communautaire, transparent, open source. On ne reconstruit jamais ce qui violerait les ToS ou l'anti-cheat (l'AH live, les endpoints privés Xbox). Les leaderboards sont **les nôtres** (télémétrie), pas ceux du jeu.