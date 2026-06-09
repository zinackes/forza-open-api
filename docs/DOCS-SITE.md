> Décision sur le **site de documentation**. Complète `docs/DECISIONS.md` et précise les cartes 4.1 / 4.3 / 6.1 du Kanban.
> 

# Site de doc

## Décision

Un **seul site Astro Starlight**, hébergé sur **Cloudflare Pages** (statique, gratuit, sous-domaine `docs.*`). Il contient : accueil (splash) + guides (getting started, exemples) + **référence API**. Projet séparé de l'API Go.

## Référence API

Générée depuis `api/openapi.yaml` via **`starlight-openapi`** (toujours synchro avec le contrat) — c'est ainsi qu'on réalise la carte 4.1, au lieu de Scalar/Redoc en standalone. L'API Go ne sert que le `/openapi.yaml` brut.

## Theming

Via les **design tokens de Starlight** (CSS custom properties : couleur d'accent, fonds, polices) + un peu de CSS custom. Identité à définir (sobre, vibe Forza/festival). Aucun framework UI requis pour ça.

## Tailwind — *optionnel*

Utile seulement pour des **pages/composants custom** (ex. landing enrichie, petit widget démo). Pas nécessaire pour la doc elle-même. S'intègre à Starlight si besoin — on confirme le setup Tailwind v4 au scaffold.

## Landing (carte 6.1)

= la page d'accueil (splash) du **même** site Starlight. Pas de site séparé.

## Contenu = cartes existantes

- **4.1** → référence API (starlight-openapi)
- **4.3** → guides & onboarding (pages Starlight)
- **6.1** → accueil/splash + démo live