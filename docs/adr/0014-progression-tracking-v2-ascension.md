# ADR 0014 — Progression Tracking V2 (Ascension)

**Date** : 2026-05-18 ; mis à jour 2026-05-22 (Phase 4 plan stabilisation — wiring fix)
**Status** : Accepted
**Branch** : `feat/progression-tracking-ascension` (commits 1-10), `feat/ascension-pipeline-v2-wiring` (Phase 4 commits)

## 2026-05-22 — Wiring corrigé (Phase 4 plan stabilisation)

Cette ADR a longtemps documenté la conception V2 sans signaler que **le câblage applicatif était incomplet** : le hook post-sync (delta notifications + pipeline progression) était attaché uniquement au `SyncHandler` HTTP. L'auto-sync scheduler (15 min, source de 100% des syncs en condition réelle) court-circuitait le handler en appelant `SyncEngine.RunDelta` directement.

Conséquence observée : tables `streak`, `record_history`, `player_records`, `milestone_earned` **restaient vides indéfiniment**. Page Ascension affichait "Aucun milestone configuré" pour tous les joueurs.

Cf. [.ai/archive/stabilisation-2026-05-22/AUDIT_ASCENSION_PIPELINE_DISCONNECTED_2026-05-21.md](../../.ai/archive/stabilisation-2026-05-22/AUDIT_ASCENSION_PIPELINE_DISCONNECTED_2026-05-21.md) pour le diagnostic complet.

**Fix appliqué (branche `feat/ascension-pipeline-v2-wiring`)** :
- Interface `port.PostSyncRunner` (before/finalizer) — `internal/port/post_sync_runner.go`
- `SyncEngine.WithPostSyncRunner(runner, slug)` — injection process-level
- Invocation dans `engine.run()` : `BeforeSync` au début, finalizer sur succès uniquement
- Adapter `api.NewPostSyncRunner(reg)` wrap `buildPostSyncDeltaHook(reg)` comme implémentation
- Injection dans `AutoSyncScheduler.WithPostSyncRunner` câblée dans `cmd/server/main.go`
- Migration backfill `seed_milestone_catalog_v1` qui charge le TOML au boot (idempotente, multi-titres-ready) — sinon `milestone_catalog` restait vide même si le pipeline tournait
- Endpoint diag `GET /api/v1/_diag/progression/{player_slug}` — counts des 5 tables V2 pour validation post-déploiement
- Tests garde anti-régression (`TestSyncEngine_WithPostSyncRunner_*` + `TestAutoSyncScheduler_DefaultRunnerFactory_InjectsRunner`)


## Context

After delivering V1 (PlayerProfile basics + LUSR + narrative engine), the
product gap was : the profile gives a **photo**, not a **mouvement**. Users
came once, saw their stats, and didn't have a reason to come back daily.

Three options were evaluated for V2 (cf. `.ai/PLAN_*_ASCENSION.md`) :

1. **Saisons Ascension** — fixed 90-day windows with anti-régression scoring.
   Rejected (cf. `PLAN_SEASONS_ASCENSION.md` DEPRECATED) : redundant with Halo's
   native seasons + Battle Pass, LUSR already mathematically anti-regresses,
   pénalité régression contre-productive psychologiquement.

2. **Saisons légères** (variant) — same idea but lighter scoring.
   Same drawbacks at lower magnitude. Rejected.

3. **PROGRESSION_TRACKING (retenu)** — 3 complementary layers focused on
   repetition without artificial deadlines :
   - **Streaks** : Duolingo-style series, peur de casser la chaîne
   - **Records & Milestones** : célébrer la progression, paliers cumulatifs
   - **Coach proactif** : notifications sur opportunités, jamais sur faiblesses

Plan reference : `.ai/V7/PLAN_PROGRESSION_TRACKING_ASCENSION.md`.

## Decision

Implement V2 in 10 commits over ~10 days, structured as :

### Backend (Go) — commits 1-7

1. **Types + DuckDB migrations** — `streak`, `record_history`, `milestone_earned`
   (stats.duckdb par joueur), `milestone_catalog` (metadata.duckdb cross-titres),
   extension `player_records` (shared_social.duckdb) avec colonne `period`
   (30d/90d/all_time) + `previous_value`/`previous_achieved_at`.

2. **Streaks evaluator** — 4 types (daily_play, daily_perf, weekly_play,
   weekly_kda_threshold) avec 5 transitions (None, Started, Incremented,
   Shielded, Broken). Sémantique « shield ressoude la chaîne » (buckets
   satisfaits avant et après un miss shielded comptent tous en increments).
   1 shield par mois calendaire, multiplicateur PP par paliers (4j/8j/15j/30j).

3. **Records detector** — pour chaque (métrique × fenêtre), compute best,
   compare au PB courant, persiste si new PB (Upsert player_records + Append
   record_history). MinMatchesForRecord=10 (anti faux-positifs). NearMiss
   à 5 % du PB (signal seul, pas de persistance).

4. **Milestones catalog TOML + detector** — référentiel cross-titres chargé
   au boot depuis `config/titles/{slug}/milestones/catalog.toml`. 13 milestones
   seed Halo Infinite sur 6 axes (volume, victoires, kills, headshots, assists,
   régularité). Detector marque earned une seule fois (PK composite garantit
   idempotence).

5. **Coach generator** — 10 types d'alertes positives uniquement (pas de
   feedback négatif, cf. plan §6.1). Mapping AlertType → notifications.Category
   réutilise `personal_record` et `threshold_crossed` existants + ajoute 6
   nouvelles catégories (record_near_miss, milestone_unlocked, milestone_near_miss,
   lusr_tier_approach, streak_milestone, comeback_welcome). Dédup 24h via
   `dedup_key` extrait des params JSON de l'historique notif (pas de table coach
   dédiée).

6. **Hook post-sync orchestrateur** — pipeline 7 étapes appelé après chaque sync
   réussi : load matches récents (120j) + stats agrégés + profile LUSR + comeback
   context → 3 detectors → coach.Generate → FilterRecent → Emitter.Emit. Mini
   service `profile` créé pour exposer μ + sub-tier + slope LOWESS sur μ (aucun
   service PlayerProfile centralisé n'existait, briques en pièces détachées).
   Non bloquant : toute erreur reste slog.Warn.

7. **Endpoints HTTP** — `GET /streaks /records /milestones` sous
   `/api/v1/players/{slug}/`. 1 handler unique avec 3 méthodes (homogénéité).
   DTOs dédiés (pas de leak structs métier). Documenté dans openapi.yaml
   (test contract enforce le plafond 0 routes non documentées).

### Frontend (React) — commits 8-9

8. **UI Streaks** — `features/ascension/` aligné sur `features/notifications/` :
   types, queries, i18n, format helpers, StreakBadge (nav L1), StreakDashboard
   (cards par streak), AscensionPage. Wiring : route file TanStack Router +
   queryKeys + NavL1 mount.

9. **UI Records + Milestones** — RecordsTimeline (PB groupés par métrique ×
   période + timeline historique), MilestonesGrid (responsive earned/locked
   avec tone gold/muted). Skip volontaire du panneau "records proches" : le
   coach émet déjà des notifs `record_near_miss` qui couvrent le besoin avec
   feedback proactif vs panneau statique.

### Glue i18n + ADR — commit 10

10. **i18n manifests** — categoryLabel + categoryDescription + 12 templates
    (6 ×2 keys × 2 locales) FR/EN dans `features/notifications/i18n.ts`. Fix
    icons.tsx (12 entrées manquantes : 6 V2 + 6 catégories Halo 2026-05-16
    déclarées sans icon mapping). Extension `enrichParams` : fallback `metric`
    → `metric_label` + nouvel enrichissement `period` → `period_label`.

## Key architectural choices

### Pas de table `coach_alert` dédiée

Le plan initial proposait une table `coach_alert` (id, user_id, type,
title_key, body_key, payload_json, read_at, dismissed_at). **Décision lors de
l'audit pré-implémentation (§2bis du plan)** : réutiliser
`player_notifications` qui :
- a déjà 18 catégories i18n-keyed, dont `personal_record` et `threshold_crossed`
- a un centre de notifs UI (cloche NavL1, badge count, préférences)
- a une infra TTL, marquage read/dismissed, hook post-sync
- a un manifest i18n keys complet

Dupliquer serait créer deux UX-équivalents. Le coach injecte ses alertes via
`Emitter.Emit` avec une catégorie dédiée par type d'alerte.

### Pas de service PlayerProfile centralisé

Le plan référençait une « V1 PlayerProfile livrée » qui n'existait pas dans
le code. Les briques étaient présentes en pièces détachées (`match_skill_rank`
pour μ/σ, `sync.SkillTiers` pour les paliers, `temporal.LowessSmooth` pour les
tendances). **Décision (commit 6)** : créer `internal/progression/profile/`
mini-service réutilisant ces briques sans construire une nouvelle façade
massive. Service load μ + tier + LOWESS sur μ uniquement — extension
incrémentale possible (composantes LUSR, axes radar) sans refacto majeur.

### Dédup sans table dédiée

`FilterRecent(alerts, recentNotifs, now, window)` filtre côté Go via la clé
`(category, dedup_key)` extraite des params JSON. `AnnotateDedupKey(*Alert)`
injecte la clé dans params juste avant `Emit`. L'historique notif existant
(player_notifications avec TTL et `created_at` indexé) sert d'historique de
dédup naturel — pas besoin d'une nouvelle table.

### Renommage `window` → `period`

`window` est un mot réservé DuckDB (utilisé pour les window functions `OVER`).
Renommage cohérent SQL + Go + JSON + frontend en `period`. Conceptuellement
identique (« fenêtre temporelle 30d/90d/all_time »).

### Coach détecteurs purs

Tous les détecteurs (records, milestones, coach.Generator) sont écrits comme
fonctions pures sans I/O : prennent leurs inputs en argument, produisent des
`[]Result`. L'orchestrateur (commit 6) seul fait l'I/O — DB queries pour le
load des inputs, NotificationsRepo pour la dédup, Emitter pour les émissions.
Permet les tests unitaires fakes + l'observabilité (chaque result peut être
loggué/téléporté ailleurs).

### Couleurs sémantiques uniquement (frontend)

Respect strict de `CLAUDE.md §20` : 0 hex, 0 Tailwind color hardcodé dans
`features/ascension/`. Statuts tones sémantiques (emerald = success, amber =
warning, muted = neutral), tokens du design system (`bg-primary`,
`border-border`, etc.).

## Consequences

### Positives

- Backend V2 complet et idempotent : 10 commits, ~30 fichiers Go, ~9 fichiers TS,
  120+ tests unit/intégration verts à chaque commit.
- Aucune table dupliquée : tout ce qui existait (notifications, player_records,
  match_skill_rank, LowessSmooth, SkillTiers) est réutilisé.
- Feedback positif uniquement (plan §6.1) : aucun toast/alerte sur régression.
  Le système pousse l'utilisateur via stims positifs (paliers atteints,
  approches, retours), pas via culpabilisation.
- Extension future facile : nouveaux types de streak via `StreakType` enum +
  satisfaction predicate, nouveaux milestones via TOML edit, nouvelles
  catégories notif via 1 ligne dans `types.go` + `notifications/types.ts` +
  template i18n FR/EN.

### Négatives / dette

- **Streaks perf-based non câblés** (daily_perf, weekly_kda_threshold) : le hook
  ne les évalue pas faute de seuils personnels exposés par PlayerProfile.
  Le code détecteur les supporte mais l'orchestrateur ne passe que les types
  universels. Follow-up : extension du profile service avec la médiane perso
  par métrique.
- **`accuracy_threshold_days`** : sémantique « 1 match du jour ≥ 0.50 »
  (décision §6 plan), permissif. Si l'UX product change d'avis (« moyenne du
  jour » ou « tous les matchs »), c'est un changement SQL local de 1 ligne.
- **Pas de panneau UI records-near-miss** (skip délibéré commit 9) : si l'UX
  product préfère un panneau statique aux notifs proactives, l'ajouter
  demande un nouvel endpoint + un composant `RecordsNearMiss.tsx` (~0.5j).
- **`metric` vs `metric_key` dans les params coach** : le coach generator
  passe `metric` (pas `metric_key` comme la convention historique). Workaround
  côté frontend dans `enrichParams` (fallback). Renommer côté Go serait plus
  propre mais demande de toucher 5 tests unit du commit 5.

### Observabilité

- `expvar` (ADR 0009) : aucun nouveau compteur ajouté en V2 — les détecteurs
  loggent via `slog.WarnContext` les erreurs individuelles. Si volume notif
  explose en prod, ajouter `notifications_emitted_total{category=X}` via
  expvar.
- `slog.InfoContext` au boot : `progression: catalog synced count=13` pour
  vérifier le chargement du TOML milestones au démarrage server.

## References

- Plan retenu : `.ai/V7/PLAN_PROGRESSION_TRACKING_ASCENSION.md`
- Plan alternatif rejeté : `.ai/V7/PLAN_SEASONS_ASCENSION.md` (DEPRECATED)
- ADR 0004 : narrative engine (composantes consommées par le coach)
- ADR 0006 : canonical indicators (`accuracy_delta`, `kills_vs_expected`, etc.
  référencés pour LOWESS futurs)
- ADR 0009 : expvar monitoring multi-user
- ADR 0013 : leased writer enforcement (toutes les écritures V2 passent par
  les repos qui acquièrent les leases)
- Pull request : feature branch `feat/progression-tracking-ascension` (10 commits)
