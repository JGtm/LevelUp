# ÉTAT CONSOLIDÉ — 2026-07-13 (source unique du reste-à-faire)

> Ce document REMPLACE `CHECKLIST_POST_CAMPAGNE_2026-07-12.md` (partiellement périmée)
> comme vue consolidée. Mise à jour : à chaque clôture de chantier, ce fichier est le
> premier à corriger. Quand une section se vide, la supprimer ; quand tout est vide,
> archiver en V7. Miroir humain : page Notion « Backlog LevelUp ».

## 1. FAIT ET EN PROD (rappel de contexte, rien à faire)

- Campagne des 9 plans + outillage CI (PR #54, 2026-07-12) et train de suivi
  (PR #55, 2026-07-13) : monitoring (triage + refonte admin 6 onglets), notifications
  (DP1-DP15), engagement lobby + agnostic H5 (supported depuis PR #56), résidus
  Match View H5 + lot V prod, hotfix + rattrapage LUSR (2 551 matchs), squash baseline
  player v1, auth store-first, retouches revue utilisateur (7 items), lot rapide
  (dockerignore, spartan auth_only, ADRs 0030/0031/0027 acceptés, playlists H5),
  parité H5 (Q19c, known-set, achievements).
- Incident disque VPS 2026-07-13 (deploy #56) : récupéré + cause racine fixée
  (`docker builder prune` dans deploy.sh, PR #57) — cache borné à ~5 Go, vérifié.

## 2. EN COURS (machine — rien à décider)

| Quoi | Où | Échéance |
|---|---|---|
| BUG fuite inter-titres home (deep-link `f=` post-switch) + bouton déconnexion disparu | agent Opus sur `fix/title-switch-deeplink-leak`, repro navigateur | rapport imminent |
| Observation `legacy_source_used` = 0 | prod (T0 = 2026-07-13) | D2 armable ≥ 2026-07-20 → chantier Phase 5 ADR 0023 (retrait fallbacks legacy) |
| Soak bruit prod B2.4 | re-mesure script §Mesure du plan triage | 2026-07-14 |
| Soak 30 j B7.4 (cible ERROR ≈ 0/j) + décision endpoint `/admin/monitoring/errors` | plan triage | ~2026-08-11 |

## 3. POUR GUILLAUME — actions rapides

- [ ] Session admin locale : se déconnecter/reconnecter après l'ajout du xuid dans
      `data/auth/users.json` (fait) ; si besoin vider `data/sessions/` + relancer.
      (Bloqué tant que le bouton logout manque — fix en cours, sinon vider les cookies.)
- [ ] Passe visuelle (prod à jour) : Explorer « matchs récents » profil H5 ·
      `/admin/data` (après session admin) · « En placement » · « Super Fiesta » ·
      grille KPI sans trous · mention « calibration provisoire » disparue (H5) ·
      match `bc918a5a` (courbe affichée) · galerie médias H5.
- [ ] Actions data-quality (admin > Données, DRY-RUN d'abord) : B4.1 registry-names ·
      B4.2 aliases/orphelins · B4.3 lying-bits · B4.4 mode translations ·
      B5.5 migrate-media-paths. (Solde le plan triage côté actions.)

## 4. POUR GUILLAUME — décisions attendues (chacune débloque un chantier pilotable)

| # | Décision | Détail |
|---|---|---|
| D1 | GO exécution `PLAN_MATCHVIEW_MOMENTUM_2026-07` | DEC-1..7 déjà tranchées dans le plan (graphe momentum match view / escouade / solo) |
| D2 | Trancher DEC-1..9 de `PLAN_REVUE_ANALYTIQUE_TIMESERIES_SQUAD_2026-07` | Couvre les items Notion : graphes Synthesis/Timeseries, ordre chronologique Escouade, radar synergie |
| D3 | `PLAN_AUTH_DEVICE_FLOW_SISU_404_2026-07` : trancher l'option du lot B | Onboarding device-flow cassé (endpoint MS retiré). Reco exécutant : Option 3 (fallback auto SISU→MSAL) + ticket recherche Option 1. Lots A (fix spinner) et D (garde-rail) exécutables dès le GO |
| D4 | GO exécution `PLAN_EXPLORER_BRIEFING_CARDS_2026-07` | Plan prêt pour agent |
| D5 | Prioriser les plans exclus de la campagne | `PLAN_ASCENSION_UX` (item Notion « Finir la page Ascension ») · `PLAN_DIAG_APPARENCE_ADMIN` (probable réponse à « nameplate ne se met pas à jour ») · `PLAN_RELATIONS_UX` (item Notion « joueurs croisés multi-jeux ») · `PLAN_WEAPON_ATTRIBUTION_V3` (à requalifier : approche supersédée par same-clock) |

## 5. BACKLOG PILOTABLE SANS DÉCISION (sur simple feu vert, par valeur estimée)

1. Requalifier le cron leaderboard 404 saison (2 ERROR/j résiduels — reste C3 du
   chantier leaderboard, réf HANDOFF_LEADERBOARD_CATALOGUE).
2. Alerte disque VPS > 80 % → notification push (aujourd'hui : journald que personne
   ne lit ; l'incident du 13/07 l'a prouvé).
3. Items Notion sans plan : notif Discord · badges « Historique des rencontres » ·
   « Enregistrer cette compo » 404 · couleurs d'équipes H5 · image unranked H5 ·
   likes Médias · i18n EN restante (menu L1, rangs, battlepass/défis, glossaire
   markdown) · heatmap accessibilité/motifs · pills « Rôle » · tooltip durée de vie
   XmYYs · axe nombre de parties · intensité contributions ordre inversé.
4. Fixture E2E synthétique (réactiverait ~60 specs actuellement skippées en CI).
5. Détecteur data-quality H5 en erreur en LOCAL (schéma shared H5 absent ?).
6. Hérités pré-campagne : V10c (budgets sous charge → statuer J4/J6) ·
   `populate-assets` absent de l'image prod · backup restic off-site.
7. Dette lint gelée (~479 issues) — optionnel.
8. Gros morceaux Notion v7.1 (Replay 2D, NAScode, télémétrie coaching, spartan
   abilities, score/objectifs match view, flag prolongations, armes d'épaule) —
   nécessitent des plans dédiés.

## 6. RANGEMENT .ai (état après ce commit)

- Racine = uniquement : plans EN ATTENTE de décision/exécution (D1-D5 ci-dessus),
  `PLAN_MONITORING_TRIAGE` (tracker des actions B4/B5.5 + soaks), ce fichier,
  la checklist du 12/07 (supersédée, pointe ici), et les documents de référence
  permanents (BACKLOG.md, CHARTS_AND_TABLES.md, ENRICHMENTS_CATALOG.md,
  project_map, thought_log).
- Archivés en V7 par ce commit : `PLAN_MIGRATION_SQUASH_BASELINE` (M6 exécuté via
  PR #55) et `ENGAGEMENT_CALIBRATION_H5_2026-07-11` (rapport diagnostic du chantier
  F7, clos).
