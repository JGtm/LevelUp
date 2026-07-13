# ÉTAT CONSOLIDÉ — 2026-07-13 (source unique du reste-à-faire) — MAJ 13/07 soir

> PAUSE LEVÉE le 13/07 soir (« ok je te laisse continuer ») — file Opus séquentielle :
> 1) vérif admin local SSO (navigateur) → 2) fin du lot ops/qualité (items 2-4) →
> 3) lots petits items Notion / i18n / auth A+D / V10c / fixture E2E.
> En parallèle (worktree, docs seulement) : écriture du plan D7 (titre dans l'URL).

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
- **Fuite inter-titres via deep-link `?f=` : CORRIGÉE, mergée (PR #59) et déployée**
  — titre estampillé dans `?f=` + réconciliation au bootstrap (reset si titre
  différent), rétro-compat des liens existants, 12 tests. **Confirmé par l'utilisateur
  le 13/07 : plus de mélange entre les jeux.** Bouton « Se déconnecter » : PAS un bug
  (local en `auth_mode: none` = aucune session, donc ni logout ni admin — par design).
- Consolidation + rangement .ai mergés (PR #58).

## 2. EN COURS (machine — rien à décider)

| Quoi | Où | Échéance |
|---|---|---|
| Lot ops/qualité (leaderboard 404, alerte disque→notif, data-quality H5 local, populate-assets) | agent Opus sur `chore/lot-ops-qualite` — **en clôture propre (pause)** : finit l'item en cours puis stop | rapport partiel imminent |
| Observation `legacy_source_used` = 0 | prod (T0 = 2026-07-13) | D2 armable ≥ 2026-07-20 → chantier Phase 5 ADR 0023 (retrait fallbacks legacy) |
| Soak bruit prod B2.4 | re-mesure script §Mesure du plan triage | 2026-07-14 |
| Soak 30 j B7.4 (cible ERROR ≈ 0/j) + décision endpoint `/admin/monitoring/errors` | plan triage | ~2026-08-11 |

## 3. POUR GUILLAUME — actions rapides

- [ ] **Actions data-quality → SUR PROD** (`https://lvelup.info/admin/data`, session
      habituelle — ce sont des actions sur les données prod, le local n'est pas le bon
      endroit) : DRY-RUN d'abord — B4.1 registry-names · B4.2 aliases/orphelins ·
      B4.3 lying-bits · B4.4 mode translations · B5.5 migrate-media-paths.
- [ ] Admin en LOCAL — CORRIGÉ par le superviseur le 13/07 : `.env.local` passé à
      `LEVELUP_AUTH_MODE=xbox` (l'essai `password` désactivait le SSO — les modes sont
      exclusifs, erreur de proposition). En mode `xbox` : connexion SSO habituelle →
      xuid résolu → session ADMIN (le xuid est dans `users.json`). Vérification
      navigateur par agent en cours ; toi : juste te reconnecter quand je confirme.
      NOTE produit (retour utilisateur) : « un admin est un joueur » — modes exclusifs
      = UX discutable ; coexistence SSO+password notée au backlog (item 9, §5).
- [x] ~~Fuite inter-titres~~ : confirmée corrigée par l'utilisateur le 13/07.
- [ ] Passe visuelle restante (prod à jour) : Explorer « matchs récents » profil H5 ·
      « En placement » · « Super Fiesta » · grille KPI sans trous · mention
      « calibration provisoire » disparue (H5) · match `bc918a5a` (courbe affichée) ·
      galerie médias H5.

## 4. POUR GUILLAUME — décisions attendues (chacune débloque un chantier pilotable)

| # | Décision | Détail |
|---|---|---|
| D1 | GO exécution `PLAN_MATCHVIEW_MOMENTUM_2026-07` | DEC-1..7 déjà tranchées dans le plan (graphe momentum match view / escouade / solo) |
| D2 | Trancher DEC-1..9 de `PLAN_REVUE_ANALYTIQUE_TIMESERIES_SQUAD_2026-07` | Couvre les items Notion : graphes Synthesis/Timeseries, ordre chronologique Escouade, radar synergie |
| D3 | `PLAN_AUTH_DEVICE_FLOW_SISU_404_2026-07` : trancher l'option du lot B | Onboarding device-flow cassé (endpoint MS retiré). Reco exécutant : Option 3 (fallback auto SISU→MSAL) + ticket recherche Option 1. Lots A (fix spinner) et D (garde-rail) exécutables dès le GO |
| D4 | GO exécution `PLAN_EXPLORER_BRIEFING_CARDS_2026-07` | Plan prêt pour agent |
| D5 | Prioriser les plans exclus de la campagne | `PLAN_ASCENSION_UX` (item Notion « Finir la page Ascension ») · `PLAN_DIAG_APPARENCE_ADMIN` (probable réponse à « nameplate ne se met pas à jour ») · `PLAN_RELATIONS_UX` (item Notion « joueurs croisés multi-jeux ») · `PLAN_WEAPON_ATTRIBUTION_V3` (à requalifier : approche supersédée par same-clock) |
| D7 | ~~Titre dans l'URL~~ **PRINCIPE APPROUVÉ le 13/07** | Plan en cours d'écriture (agent Opus, worktree). Sera soumis à relecture avant exécution. Routes `/t/{slug}/...`, la garde deep-link #59 reste en défense en profondeur |

## 5. BACKLOG PILOTABLE SANS DÉCISION (sur simple feu vert, par valeur estimée)

1. ~~Requalifier le cron leaderboard 404 saison~~ FAIT (lot ops item 1, commit
   `f4721be0f` — en attente de merge avec le reste du lot).
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
   `populate-assets` absent de l'image prod.
   Backup restic off-site : **DÉCISION 2026-07-13 — inaction actée par l'utilisateur**
   (pas d'actualité, à revoir beaucoup plus tard ; aucune tâche ouverte).
7. Dette lint gelée (~479 issues) — optionnel.
8. Gros morceaux Notion v7.1 (Replay 2D, NAScode, télémétrie coaching, spartan
   abilities, score/objectifs match view, flag prolongations, armes d'épaule) —
   nécessitent des plans dédiés.
9. Coexistence des modes d'auth (SSO Xbox ET password sur la même instance, au lieu
   de modes exclusifs) — retour UX utilisateur du 13/07 (« un admin est un joueur »).

## 6. RANGEMENT .ai (état après ce commit)

- Racine = uniquement : plans EN ATTENTE de décision/exécution (D1-D5 ci-dessus),
  `PLAN_MONITORING_TRIAGE` (tracker des actions B4/B5.5 + soaks), ce fichier,
  la checklist du 12/07 (supersédée, pointe ici), et les documents de référence
  permanents (BACKLOG.md, CHARTS_AND_TABLES.md, ENRICHMENTS_CATALOG.md,
  project_map, thought_log).
- Archivés en V7 par ce commit : `PLAN_MIGRATION_SQUASH_BASELINE` (M6 exécuté via
  PR #55) et `ENGAGEMENT_CALIBRATION_H5_2026-07-11` (rapport diagnostic du chantier
  F7, clos).
