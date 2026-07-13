# ÉTAT CONSOLIDÉ — 2026-07-13 (source unique du reste-à-faire) — MAJ 13/07 soir

> FILE OPUS (MAJ 13/07 soir, après arbitrages utilisateur) :
> 1) fin du lot ops/qualité (en vol : fix URL login device-flow + alerte disque +
>    data-quality H5 + populate-assets) → 2) **D4 Explorer briefing cards (GO)** →
> 3) **D1 Momentum (GO)** → 4) auth device-flow lots A+D → 5) V10c → 6) fixture E2E →
> train de merge unique.
> RETIRÉ de la file (recadrage utilisateur) : tout chantier tiré du backlog Notion —
> carnet personnel, pas une file d'exécution. D2/D5 : reportés à la v7.1.

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
| Lot ops/qualité (items 0/0b SSO+UI, 1 leaderboard, 2 alerte disque, 3 data-quality H5 FAITS ; reste : populate-assets) | agent Opus sur `chore/lot-ops-qualite` | rapport final à la fin du lot |
| Observation `legacy_source_used` = 0 | prod (T0 = 2026-07-13) | D2 armable ≥ 2026-07-20 → chantier Phase 5 ADR 0023 (retrait fallbacks legacy) |
| Soak bruit prod B2.4 | re-mesure script §Mesure du plan triage | 2026-07-14 |
| Soak 30 j B7.4 (cible ERROR ≈ 0/j) + décision endpoint `/admin/monitoring/errors` | plan triage | ~2026-08-11 |

## 3. POUR GUILLAUME — actions rapides

- [ ] **Actions data-quality → SUR PROD** (`https://lvelup.info/admin/data`, session
      habituelle — ce sont des actions sur les données prod, le local n'est pas le bon
      endroit) : DRY-RUN d'abord — B4.1 registry-names · B4.2 aliases/orphelins ·
      B4.3 lying-bits · B4.4 mode translations · B5.5 migrate-media-paths.
- [ ] Admin en LOCAL — VÉRIFIÉ + RÉPARÉ le 13/07 (lot ops item 0 + retouche UI) :
      `.env.local` en `LEVELUP_AUTH_MODE=xbox` pris en compte (serveur redémarré) ;
      le login SSO Xbox — CASSÉ pour tout le monde (URL device-code 404 depuis
      l'introduction de SISU, cf. `PLAN_AUTH_DEVICE_FLOW_SISU_404` requalifié) — est
      corrigé ; et ton signalement UI est réglé : le lien affiché est maintenant
      court (`microsoft.com/link`) ET pointe la vraie page de SAISIE du code
      (`oauth20_remoteconnect.srf`, vérifié navigateur) — l'ancienne URL authorize
      interminable ne demandait jamais le code. **Toi : ouvre
      `http://localhost:5173/login`, va sur `microsoft.com/link` (lien affiché),
      connecte-toi avec ton compte Microsoft et saisis le code affiché par LevelUp —
      la session locale devient automatiquement ADMIN (ton xuid 2533274823110022 est
      dans `users.json`).**
      NOTE produit (retour utilisateur) : « un admin est un joueur » — modes exclusifs
      = UX discutable ; coexistence SSO+password notée au backlog (item 9, §5).
- [ ] **Webhook Discord en PROD pour l'alerte disque** (après merge+deploy du lot ops) :
      dans `/opt/levelup/app_settings.json`, passer `"discord_notifications_enabled": true`
      et renseigner `"discord_webhook_url": "https://discord.com/api/webhooks/…"`
      (ou env `LEVELUP_DISCORD_WEBHOOK_URL`), puis redémarrer le conteneur. Vérifié
      le 13/07 (lecture seule) : actuellement `false` + pas d'URL → sans ça, l'alerte
      disque reste visible UNIQUEMENT dans Admin > Monitoring (détections + badge).
- [x] ~~Fuite inter-titres~~ : confirmée corrigée par l'utilisateur le 13/07.
- [ ] Passe visuelle restante (prod à jour) : Explorer « matchs récents » profil H5 ·
      « En placement » · « Super Fiesta » · grille KPI sans trous · mention
      « calibration provisoire » disparue (H5) · match `bc918a5a` (courbe affichée) ·
      galerie médias H5.

## 4. POUR GUILLAUME — décisions attendues (chacune débloque un chantier pilotable)

| # | Décision | Détail |
|---|---|---|
| D1 | ~~GO exécution momentum~~ **GO REÇU le 13/07** | En file position 3 (après le lot ops et D4) |
| D2 | ~~Revue analytique~~ **REPORTÉ v7.1** (décision utilisateur 13/07) | Les DEC-1..9 seront tranchées à l'ouverture du chantier v7.1 |
| D3 | ~~Trancher l'option du lot B~~ **SANS OBJET le 13/07** (lot ops item 0) | La prémisse « endpoint MS retiré » était fausse : l'URL du code n'a jamais été la bonne (`/oauth20_connect/device` ; la vraie = `oauth20_connect.srf`). Fix livré + vérifié navigateur = Option 1 de fait, plus rien à trancher. Restent au plan : lots A (StepDeviceCode) et D (garde-rail + doc) sur simple feu vert |
| D4 | ~~GO exécution briefing cards~~ **GO REÇU le 13/07** (« plus tôt que ce que j'envisageais » — replanifiable d'un mot) | En file position 2, juste après le lot ops |
| D5 | ~~Prioriser les plans exclus~~ **REPORTÉ v7.1** (décision utilisateur 13/07) | Ascension UX · Diag apparence admin · Relations UX · weapon_attribution_v3 — dossier v7.1 |
| D7 | ~~Titre dans l'URL~~ **PRINCIPE APPROUVÉ le 13/07** | Plan en cours d'écriture (agent Opus, worktree). Sera soumis à relecture avant exécution. Routes `/t/{slug}/...`, la garde deep-link #59 reste en défense en profondeur |

## 5. BACKLOG PILOTABLE SANS DÉCISION (sur simple feu vert, par valeur estimée)

1. ~~Requalifier le cron leaderboard 404 saison~~ FAIT (lot ops item 1, commit
   `f4721be0f` — en attente de merge avec le reste du lot).
2. ~~Alerte disque VPS > 80 % → notification push~~ FAIT (lot ops item 2) : boucle
   serveur `RunDiskWatchLoop` (15 min, volume data = FS hôte via bind mount) →
   détection persistée + badge admin (WARN/ERROR stable) + notif Discord
   (transition + rappel 24 h + rétablissement, toggle `discord_notify_disk`).
   Seuils A5.3 étendus : 80 %/90 % d'occupation EN PLUS des absolus 2 Go/500 Mo.
   **Pour la notif push en PROD, config requise (§3)** ; sans webhook l'alerte
   reste visible (détections admin). Script hôte journald = redondance, retrait
   optionnel.
3. ~~Items Notion sans plan~~ **RETIRÉ (recadrage utilisateur 13/07)** : le backlog
   Notion est le carnet personnel de Guillaume — aucun chantier n'en sera tiré sans
   demande explicite de sa part.
4. Fixture E2E synthétique (réactiverait ~60 specs actuellement skippées en CI).
5. ~~Détecteur data-quality H5 en erreur en LOCAL~~ FAIT (lot ops item 3) : cause =
   la metadata H5 (schéma PROPRE, PMT-9) n'a NI `mode_name_tr` NI `playlists_catalog`
   (prouvé on-disk : 13 tables, `playlists` à la place) → Catalog Error → tout
   l'endpoint en 500. Fix title-agnostic : introspection de schéma → détecteurs
   `untranslated_modes`/`orphan_playlists` NON APPLICABLES pour le titre → 0 sans
   erreur (les détecteurs shared continuent de compter). Test de régression
   metadata H5-like.
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
