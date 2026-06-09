Pour chaque type : la source PROPRE et la méthode. Off-limits en bas.

## Catalogue voitures

- Datasets communautaires (Kaggle Forza), listes car-ID sur GitHub, **wiki Fandom** (API MediaWiki).
- Normalisation → table `cars`. Réconciliation des ids via les **CarOrdinal** de la télémétrie quand dispo.
- FH6 ~550 voitures (Japon, kei cars). FH5 en backfill.

## Festival Playlist

- **forza.net/events** (backé Strapi) : tenter l'API JSON, sinon scrape HTML (URLs SxxWx).
- **Forums officiels** Forza : source secondaire (récompenses hebdo), validation croisée.
- **Wiki Fandom** (API MediaWiki) : backfill historique (S1 → courante).
- Rafraîchi chaque jeudi (reset Forza) par le scheduler.

## Perfs & leaderboards

- **Data Out** officiel (UDP, 60 pkt/s, layout Horizon FH4/5/6, 324 octets, offset +12). Companion desktop **read-only** → POST /v1/sessions.
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