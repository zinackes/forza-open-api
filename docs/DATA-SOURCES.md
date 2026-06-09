Pour chaque type : la source PROPRE et la méthode. Off-limits en bas.

## Catalogue voitures

- **Wiki Fandom** (API MediaWiki) — source primaire implémentée (`cmd/seed cars`) : page liste `{{CarListStatsFH6}}` + infobox par voiture. Datasets communautaires (Kaggle, listes GitHub) en complément/cross-check éventuel.
- Cross-check officiel : la liste publique **forza.net/fh6cars** (counts, nouveautés Car Pass).
- Normalisation → table `cars`. Réconciliation des ids via les **CarOrdinal** de la télémétrie quand dispo (source propre à trouver : crowdsourcing Data Out ; les dumps « car ID list » de mods extraits des fichiers du jeu sont OFF-LIMITS).
- FH6 ~620 voitures / ~87 constructeurs (juin 2026, Japon, kei cars). FH5 en backfill.

## Festival Playlist

- **forza.net/events** (backé Strapi) : tenter l'API JSON, sinon scrape HTML (URLs SxxWx).
- **Forums officiels** Forza : source secondaire (récompenses hebdo), validation croisée.
- **Wiki Fandom** (API MediaWiki) : backfill historique (S1 → courante).
- Rafraîchi chaque jeudi (reset Forza) par le scheduler.

## Perfs & leaderboards

- **Data Out** officiel (UDP one-way, fréquence = framerate). ⚠️ Le format FH6 est **fixe et distinct de FH4/FH5** : 3 champs supplémentaires (`CarGroup`, `SmashableVelDiff`, `SmashableMass`) insérés après `NumCylinders`, avant `PositionX` — pas de sélection de format in-game. Réf. : doc officielle « Forza Horizon 6 Data Out Documentation » (support.forza.net). Companion desktop **read-only** → POST /v1/sessions.
- On construit **nos** leaderboards. On ne lit pas ceux du jeu.

## Tunes & liveries

- **Crowdsourcing** : share codes soumis par la communauté (auth légère + modération).

## Exploratoire (Phase 9, opt-in)

- CV **read-only** (OCR écran) pour AH-lite / Rivals reconstruits. Prix **crowdsourcés** (déclaratif). Jamais le marché live.

## OFF-LIMITS (ban-bait, jamais)

- Scraping / lecture mémoire du jeu, injection, overlays intrusifs, bots de l'Auction House.
- Endpoints privés Xbox / relying-party tokens.
- Tout ce qui déclenche Easy Anti-Cheat (kernel, HWID bans).

Règle d'or : user-agent identifiable, délais polis, API officielles > scrape, doute → on s'abstient.