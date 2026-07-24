# Garde-fou « trous LUSR » + exposition monitoring

> Créé 2026-07-21. Diagnostic + mesure faits (lecture directe prod via `ssh lvelup`, DBs locales
> serveur arrêté). NON implémenté. Voir mémoire `project_lusr_interior_gaps_guardrail`.

## Context

Signalé : le match `ac313879` a un LUSR (Platine I) sur le VPS mais **aucun** LUSR en local,
et JGtm est **Or VI en local / Platine I en prod** sans changement de calibration.

**Diagnostic prouvé** (lecture directe des deux bases — copie prod via `ssh lvelup`, DBs
locales serveur arrêté, `cmd/diag_q` read-only) :

- Le LUSR v2 est une note μ **incrémentale, à état, par `(xuid, playlist_group)`**, gardée
  par un **watermark chronologique** (`player_skill_state_v2`, DB partagée). Le libellé de
  palier est **cuit à l'écriture** dans `match_skill_rank` (player DB), jamais recalculé à la
  lecture. Local et VPS maintiennent chacun leur propre état — jamais synchronisé.
- `ac313879` = « Super Fiesta:Slayer » (groupe `chaos`), roster **identique** prod/local
  (16 participants dont 3 bots, team 1 à gros turnover). **Prod l'a scoré en live, en ordre**
  (ligne écrite 18/07 20:21, Platine I) ; **local l'a sauté** (arrivée hors-ordre : quand il a
  été vu, le watermark `chaos` local était déjà passé → `skippedAlready`, jamais noté).
- Effet : μ `chaos` **prod 25,181 vs local 24,940**, de part et d'autre de la frontière
  Or→Platine (**μ = 25,0**). Un seul match sauté, pile sur la crête, bascule tout le palier.
- Ce n'est ni calibration, ni données manquantes (local est même plus à jour), ni bug
  d'affichage. C'est le **même code + mêmes données traités dans un ordre différent**.

**Mesure de l'ampleur** (trous = éligibles+2équipes+équilibrés SANS ligne LUSR) :

| Joueur | Trous PROD | Trous LOCAL | Note |
|---|---|---|---|
| JGtm | 10 (dont ~3 récents en attente, ~7 permanents) | 1 (= `ac313879`) | ensembles **disjoints** |
| Madina97294 | 5 | 5 | |
| Chocoboflor | 4 | 5 | |

→ Les trous sont **réels, permanents, par-environnement et non-déterministes** (dépendent de
l'ordre d'arrivée). Il en existe des deux côtés ; aucun n'est « la vérité ». D'où le besoin
d'un **garde-fou** : détecter les trous d'intérieur, les exposer, et pouvoir les combler par
replay chronologique (déterministe, `ac313879` est calculable — prod l'a prouvé).

Doctrine du pipeline (à préserver) : **« plutôt un trou qu'une note fausse »** — un compute EP
raté n'écrit jamais de note. Le garde-fou ne change PAS cette règle ; il rend les trous
**visibles et réparables**.

## Objectif

1. **Détecter** les trous d'intérieur LUSR (par joueur, par groupe), en distinguant
   *permanent* (sous le watermark) de *récent-en-attente* (au-dessus).
2. **Exposer** l'état (trous + santé du garde-fou) dans la section monitoring `/admin/data`.
3. **Réparer** : action admin manuelle (replay owner-only in-server) + option d'auto-heal
   quotidien borné.

## Décision de conception

- **Source unique d'éligibilité** (anti-dérive, règle CLAUDE.md n°6) : aujourd'hui
  l'éligibilité est implicite dans le flot de `processOneShadowMatch`
  (`apps/go-api/internal/sync/skill/skill_v2_shadow.go:251-335`). **Extraire un prédicat pur**
  `classifyLUSREligibility(ctx, match, rosters) (group string, eligible bool, reason skipReason)`
  réutilisé par **le scoreur ET le détecteur** + garde-rail grep interdisant de dupliquer les
  filtres. Sans ça, le détecteur re-diverge du scoreur (over/under-count).
- **Définition « trou d'intérieur »** : match éligible, SANS ligne `rating_type='LUSR'`, ET
  `start_time < last_match_at` du groupe (`player_skill_state_v2_latest`). Au-dessus du
  watermark = récent-en-attente (exclu du compteur d'alerte, compté séparément).
- **Replay** : `SyncEngine.RecomputeLUSRCanonical(ctx)`
  (`apps/go-api/internal/sync/engine_backfills.go:187`) — in-server, prend les leases
  `dblease.KindPlayer`+`KindSharedMatches`, **tourne serveur up**, owner-only (reset watermark
  + rejeu chrono complet). PAS `cmd/lusr_v2_canonical_backfill` (serveur arrêté uniquement).
  Replay = plein historique (pas de replay partiel « juste le trou »).

## Plan d'implémentation

### Lot 1 — Prédicat d'éligibilité factorisé (Go, backend)
- Extraire `classifyLUSREligibility(...)` depuis `processOneShadowMatch` (mêmes checks, ordre
  identique : chain non vide, ownerHasTeam, 2 équipes via `buildTwoTeamRosters`, balance
  `concurrentTeamSize`/`isTeamImbalanceTooHigh`, outcome ∈ {1,2,3}, owner présent). Le scoreur
  appelle ce prédicat ; comportement inchangé (test de non-régression sur les compteurs
  `shadowRunStats`).
- Garde-rail grep : interdire la ré-écriture des littéraux de filtre (is_ranked/is_firefight/
  duration>=30) hors du helper + `loadShadowMatches`.

### Lot 2 — Détecteur read-only (Go)
- `internal/sync/skill/lusr_gap_scan.go` : `ScanLUSRGaps(ctx, playerDB, sharedDB, xuid) (GapReport, error)`
  — réutilise le SQL `loadShadowMatches` + le prédicat du Lot 1 + `LEFT JOIN match_skill_rank
  (rating_type='LUSR')`, croise avec le watermark. Retourne par groupe : `{eligible, rated,
  interiorGaps[], pendingRecent}`. Read-only, best-effort, timeout-gardé — même forme que
  `runDualRowSentinelBestEffort` (`apps/go-api/internal/sync/engine_postsync_scoring.go:210`).
- Test avec dataset hétérogène (bots, quitters, FFA, BTB déséquilibré) validant : `ac313879`-like
  compté ; FFA/imbalance NON comptés ; récent au-dessus watermark → `pendingRecent`, pas trou.

### Lot 3 — Métriques + planification (Go)
- Expvar (namespace existant `apps/go-api/internal/sync/skill/skill_v2_metrics.go:30`) :
  `levelup.lusr_v2.interior_gaps` (gauge, dernier scan) ; **re-exposer** les compteurs
  aujourd'hui limités à `/debug/vars` : `canonical_write_held_watermark_total`,
  `canonical_owner_missing_total`.
- Accrocher `ScanLUSRGaps` dans **`HealthScheduler`** (cron 24h read-only qui itère déjà
  titres+joueurs, `apps/go-api/internal/scheduler/data_health_check.go`) : ajouter un champ
  `LUSRGaps` au `DataHealthCheckResult` + `ReportCronRun`.

### Lot 4 — Remédiation
- **Action admin manuelle** (défaut) : `POST /api/v1/admin/monitoring/lusr-gaps/{player}/recompute`
  → `SyncEngine.RecomputeLUSRCanonical`. Loggé, NoStore, RequireAdmin.
- **Auto-heal optionnel borné** : dans `HealthScheduler`, si trous permanents > seuil pour un
  joueur, déclencher le replay (1 joueur/cycle max, loggé, kill-switch daté commenté selon
  règle CLAUDE.md n°11). **Démarrer OFF (alerte seule), activer après observation** — décision
  utilisateur.

### Lot 5 — Exposition monitoring (Go DTO + API + Web)
- Modèle : `AdminWeaponCoverage` (coverage % + top offenders).
- Endpoint `GET /api/v1/admin/monitoring/lusr-gaps` monté dans
  `apps/go-api/internal/api/handlers/admin_monitoring.go` ; DTO dans
  `internal/domain/admin_monitoring.go` : par titre/joueur `{coveragePct, ratedCount,
  interiorGaps, pendingRecent, topGaps[]{matchId,playlist,startTime,group}, guardrail{lastAuditAt,
  lastAutoHealAt, heldWatermark, ownerMissing}}` ; runner sur `ServiceRegistry`
  (`apps/go-api/internal/api/wire/registry_monitoring.go`).
- Web : section dans **`/admin/data`** (`apps/web/src/features/admin/data/AdminDataPage.tsx`)
  à côté de `InvariantsSection` — panneau « Notes LUSR — trous & garde-fou » : jauge coverage,
  liste top trous, état garde-fou, bouton « Recalculer » (action Lot 4). Query `useLusrGaps`
  dans `monitoring/queries.ts` + clé dans `apps/web/src/lib/query/keys.ts`. Badge d'onglet via
  `AdminMonitoringOverview`/`tabBadges.ts` si trous > 0. Strings FR+EN, tokens couleur
  sémantiques (pas de hex).
- **Réconcilier avec l'existant** : l'invariant `checkSkillRankMissing`
  (`apps/go-api/internal/sync/invariants/invariants.go:351`, « croissance = désync watermark
  LUSR ») couvre partiellement le sujet — soit l'enrichir pour pointer vers ce panneau, soit
  documenter le partage de responsabilité (éviter double signal divergent).

### Lot 6 — Livraison
- Tests Go (`make go-api-test` + `-tags=integration` pour persist/replay), `make check-types`,
  `make test-web`, `make go-api-lint`. Entrée `.ai/thought_log.md`. Skill `delivery-checklist`.
- **Prod = deploy auto sur push main** : prévenir avant merge.

## Correctif immédiat séparé (optionnel, hors garde-fou)
Converger le local sur prod maintenant : serveur arrêté →
`cd apps/go-api && go run -tags cgo ./cmd/lusr_v2_canonical_backfill --commit --data-root ../.. JGtm`
→ rejoue en ordre → `ac313879` noté, JGtm remonte Platine en local. À faire sur demande.

## Vérification (end-to-end)
1. Détecteur : test unitaire dataset hétérogène (Lot 2) ; sur données réelles, `ScanLUSRGaps`
   pour JGtm doit rendre **10 prod / 1 local** (conformes à la mesure ci-dessus) et classer les
   3 Fiesta 21/07 en `pendingRecent`.
2. Replay : après `RecomputeLUSRCanonical(JGtm)` en local, re-scan → **0 trou d'intérieur**,
   `ac313879` a une ligne LUSR, μ `chaos` ≥ 25,0 (Platine), convergence avec prod.
3. Monitoring : `GET /admin/monitoring/lusr-gaps` renvoie les compteurs ; le panneau `/admin/data`
   les affiche ; le bouton « Recalculer » déclenche le replay et le compteur retombe à 0.
4. Garde-rail : test grep anti-duplication du prédicat d'éligibilité (Lot 1).

## Fichiers principaux
- Backend : `internal/sync/skill/skill_v2_shadow.go` (extraction prédicat),
  `internal/sync/skill/lusr_gap_scan.go` (neuf), `internal/sync/skill/skill_v2_metrics.go`,
  `internal/scheduler/data_health_check.go`, `internal/sync/engine_backfills.go` (réutilisé),
  `internal/api/handlers/admin_monitoring.go`, `internal/domain/admin_monitoring.go`,
  `internal/api/wire/registry_monitoring.go`.
- Web : `features/admin/data/AdminDataPage.tsx` (+ nouveau panneau), `features/admin/monitoring/queries.ts`,
  `lib/query/keys.ts`, i18n manifests.
