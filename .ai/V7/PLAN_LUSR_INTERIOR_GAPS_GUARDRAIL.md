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

### Lot 1 — Prédicat d'éligibilité factorisé (Go, backend) — [x] FAIT 2026-07-21
Statut : `classifyLUSREligibility` + `lusrEligibility`/`lusrSkipReason` ajoutés dans
`skill_v2_shadow.go` ; `processOneShadowMatch` refactoré (checks rosters/équilibre/outcome
délégués, ordre + compteurs `shadowRunStats` inchangés, prédicat appelé APRÈS le watermark
donc pas de query rosters sur l'historique déjà vu). Garde-rail
`lusr_eligibility_guardrail_test.go` (allowlist `skill_rating_loaders.go` + `skill_v2_shadow.go`).
Gate : `go build ./internal/sync/skill/` OK ; tests non-régression shadow + garde-rail verts.
Signature finale : `classifyLUSREligibility(ctx, sharedDB, m) lusrEligibility` (le `group` est
pré-résolu par le caller via `GetLUSRChainForTitle`, maillon mono-source distinct ; le filtre SQL
reste dans `loadShadowMatches`). Les 3 maillons d'éligibilité (filtre SQL, chaîne, prédicat rosters)
sont couverts par le garde-rail.

- Extraire `classifyLUSREligibility(...)` depuis `processOneShadowMatch` (mêmes checks, ordre
  identique : chain non vide, ownerHasTeam, 2 équipes via `buildTwoTeamRosters`, balance
  `concurrentTeamSize`/`isTeamImbalanceTooHigh`, outcome ∈ {1,2,3}, owner présent). Le scoreur
  appelle ce prédicat ; comportement inchangé (test de non-régression sur les compteurs
  `shadowRunStats`).
- Garde-rail grep : interdire la ré-écriture des littéraux de filtre (is_ranked/is_firefight/
  duration>=30) hors du helper + `loadShadowMatches`.

### Lot 2 — Détecteur read-only (Go) — [x] FAIT 2026-07-21
Statut : `lusr_gap_scan.go` (`ScanLUSRGaps(ctx, playerDB, sharedDB, xuid) (*LUSRGapReport, error)`)
réutilise `loadShadowMatches` + `classifyLUSREligibility`, LEFT-set via `match_skill_rank_latest`
(`rating_type='LUSR'`, vue _latest ART n°2), watermark via `player_skill_state_v2_latest`.
Types `LUSRGapReport`/`LUSRGroupGaps`/`LUSRGapMatch`. Trou intérieur = éligible + non noté +
`!start_time.After(last_match_at)` (sémantique exacte du `skippedAlready` du scoreur — cf.
journal ci-dessous) ; sans watermark → pending, pas trou. Tests `lusr_gap_scan_test.go` :
dataset hétérogène (bot xuid vide, quitter, FFA, 4v2) → eligible=3/rated=1/interior=1/pending=1 ;
cas sans watermark → tout pending. Gate : suite `./internal/sync/skill/` verte.
Note conception : le plan écrivait `start_time < last_match_at` ; retenu `<=` (`!After`) pour
coller au `skippedAlready` du scoreur — le match-frontière (== watermark) est de toute façon
noté donc exclu par le set rated, les deux bornes donnent le même résultat en pratique.

### Lot 2 — Détecteur read-only (Go) — spéc d'origine
- `internal/sync/skill/lusr_gap_scan.go` : `ScanLUSRGaps(ctx, playerDB, sharedDB, xuid) (GapReport, error)`
  — réutilise le SQL `loadShadowMatches` + le prédicat du Lot 1 + `LEFT JOIN match_skill_rank
  (rating_type='LUSR')`, croise avec le watermark. Retourne par groupe : `{eligible, rated,
  interiorGaps[], pendingRecent}`. Read-only, best-effort, timeout-gardé — même forme que
  `runDualRowSentinelBestEffort` (`apps/go-api/internal/sync/engine_postsync_scoring.go:210`).
- Test avec dataset hétérogène (bots, quitters, FFA, BTB déséquilibré) validant : `ac313879`-like
  compté ; FFA/imbalance NON comptés ; récent au-dessus watermark → `pendingRecent`, pas trou.

### Lot 3 — Métriques + planification (Go) — [x] FAIT 2026-07-21
Statut : jauge `levelup.lusr_v2.interior_gaps` + accesseurs `LUSRInteriorGapsGaugeValue` /
`LUSRCanonicalWriteHeldWatermarkValue` / `LUSRCanonicalOwnerMissingValue` (ré-exposition pour le
DTO Lot 5) + setter `SetLUSRInteriorGapsGauge` dans `skill_v2_metrics.go`. Accroche dans
`HealthScheduler.auditTitle` → `auditTitleLUSRGaps` (itère player dirs, résout xuid via `xuid.txt`,
`ScanLUSRGaps` timeout 60s/joueur, agrège) ; champs `LUSRInteriorGaps`/`LUSRPendingRecent`/
`LUSRPlayersScanned` sur `DataHealthCheckResult` ; jauge publiée + loggée par cycle. Trous NON
comptés dans WarningsTotal (signal distinct). Gate : build + `./internal/scheduler/` +
`./internal/sync/skill/` verts.

### Lot 3 — Métriques + planification (Go) — spéc d'origine
- Expvar (namespace existant `apps/go-api/internal/sync/skill/skill_v2_metrics.go:30`) :
  `levelup.lusr_v2.interior_gaps` (gauge, dernier scan) ; **re-exposer** les compteurs
  aujourd'hui limités à `/debug/vars` : `canonical_write_held_watermark_total`,
  `canonical_owner_missing_total`.
- Accrocher `ScanLUSRGaps` dans **`HealthScheduler`** (cron 24h read-only qui itère déjà
  titres+joueurs, `apps/go-api/internal/scheduler/data_health_check.go`) : ajouter un champ
  `LUSRGaps` au `DataHealthCheckResult` + `ReportCronRun`.

### Lot 4 — Remédiation — [x] BACKEND FAIT 2026-07-21
Statut : action manuelle `POST /api/v1/admin/monitoring/lusr-gaps/{player}/recompute` →
`ServiceRegistry.RecomputeLUSRGapsForPlayer` (construit un `SyncEngine` in-server avec
`SharedProvider` → `RecomputeLUSRCanonical`, leases coordonnés B-swap). Handler
`admin_lusr_gaps.go` (RequireAdmin + NoStore hérités), monté dans `server_admin_monitoring.go`.
Auto-heal borné : `HealthScheduler.maybeAutoHealLUSR` (1 joueur/cycle, le plus impacté, seuil
`lusrAutoHealMinGaps=3`), kill-switch `LEVELUP_LUSR_AUTOHEAL_ENABLED` **défaut OFF** (commentaire
daté conforme CLAUDE.md n°11 : flip 2026-07-21 OFF, retrait cible après ≥2 sem. stables, critère
gauge→0 post-heal). Hook injecté depuis `main.go` vers `reg.RecomputeLUSRGapsForPlayer`. Gate :
`go build ./...` + tests scheduler/skill/wire/handlers verts. **DÉCISION UTILISATEUR requise** :
activer l'auto-heal (flag ON) après observation — démarré OFF (alerte seule) comme prévu au plan.

### Lot 4 — Remédiation — spéc d'origine
- **Action admin manuelle** (défaut) : `POST /api/v1/admin/monitoring/lusr-gaps/{player}/recompute`
  → `SyncEngine.RecomputeLUSRCanonical`. Loggé, NoStore, RequireAdmin.
- **Auto-heal optionnel borné** : dans `HealthScheduler`, si trous permanents > seuil pour un
  joueur, déclencher le replay (1 joueur/cycle max, loggé, kill-switch daté commenté selon
  règle CLAUDE.md n°11). **Démarrer OFF (alerte seule), activer après observation** — décision
  utilisateur.

### Lot 5 — Exposition monitoring (Go DTO + API + Web) — [x] FAIT 2026-07-21
Backend : DTO `domain.AdminLUSRGaps` (+ `LUSRGapPlayer`/`LUSRGapItem`/`LUSRGuardrailHealth`/
`AdminLUSRRecomputeResponse`), runners `ServiceRegistry.LUSRGapsReport` (par joueur via
`resolveMonitoringDBs` + `ScanLUSRGaps`, tri par impact, garde-fou via accesseurs expvar) +
`RecomputeLUSRGapsForPlayer`. Handler `admin_lusr_gaps.go` (GET + POST), monté. Champ
`lusr_interior_gaps` (jauge) ajouté à `AdminMonitoringOverview` (Go + openapi.yaml + regen) pour le
badge. openapi.yaml : 2 paths + champ overview ; `TestContractOpenAPIYAMLValid` vert.
Web : `useLusrGaps` + types hand-typed + clé `adminLusrGaps` ; `useRecomputeLusrGaps` (POST +
invalidations) ; `LusrGapsSection.tsx` (une carte/titre actif : barre couverture rated/pending/
interior via tokens, stats, ligne garde-fou, joueurs impactés + `AdminActionButton` « Recalculer ») ;
montée dans `AdminDataPage`. i18n `admin.data.section_lusr_gaps` + bloc `admin.lusr.*` (FR+EN,
manifests régénérés). Badge : `computeTabBadges` cumule `lusr_interior_gaps` dans le badge
`/admin/data` (warning ; FAIL invariant masque). Réconciliation `checkSkillRankMissing` : voir
Découvertes ci-dessous. Gates : `check-types` ✓, eslint 0 + 0 hex ✓, tests admin (80) + tabBadges
(21, dont 2 nouveaux) ✓, `go build ./...`/`go vet`/`go-api-test` ✓.

### Lot 5 — Exposition monitoring (Go DTO + API + Web) — spéc d'origine
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

### Lot 6 — Livraison — [x] EN COURS 2026-07-21
Gates rejoués localement (verts) : `go build ./...`, `go vet`, `make go-api-test` (dont
`TestContractOpenAPIYAMLValid`), `./internal/sync` complet + `./internal/sync/skill` +
`./internal/scheduler` + `./internal/api/{wire,handlers}` + `./internal/domain` ; `make check-types`
(cache `.tsbuildinfo` purgé), eslint 0 + 0 hex, suite `vitest` complète (2422 tests, 0 échec après
renommage `LusrRecomputeResult`). Tests d'intégration `-tags="integration cgo" -p 1
./internal/sync/... ./internal/persist/...` : **VERTS** (exit 0, 0 FAIL — sync 191s, persist 27s,
skill, invariants, v2 tous ok ; anti-ART confirmé). Refactor taille-fichier :
prédicat → `lusr_eligibility.go` (skill_v2_shadow.go 844→781, sous son 792 d'origine) ; auto-heal →
`data_health_lusr.go` (data_health_check.go 501→385). Entrée `.ai/thought_log.md` ajoutée.
[!] **Commit/merge NON faits** — attente autorisation user (CLAUDE.md n°16 ; push main = deploy prod).

### Découvertes (hors périmètre — notées, non traitées)
- Le chantier LUSR est empilé sur la branche `feat/frag-distribution-v2` (frags). Découpage
  commits / séparation de branche à décider avec l'user au moment du commit.
- `AdminLUSRGaps`/`AdminLUSRRecomputeResponse` sont hand-typed côté web (pas de schéma openapi pour
  ces réponses admin — cohérent avec `AdminWeaponCoverage`). Migration vers mirror généré = dette
  optionnelle si un jour les réponses admin gagnent un schéma openapi complet.

### Lot 6 — Livraison — spéc d'origine
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

## Addendum revue 2026-07-22 (post-merge)

Lot 6 : RÉSOLU — commité `b8024ea83` sur `feat/monitoring-lusr-fixes`. La revue complète de la
branche a confirmé 2 défauts de la jauge, corrigés le 2026-07-22 :

- `[x]` **Jauge sous-comptée sur scan partiel** : un joueur dont la player DB était tenue RW par un
  sync était sauté en Debug sans compteur, puis la jauge était republiée quand même → le badge
  `/admin/data` pouvait s'éteindre alors que les trous existaient (« unmeasured ≠ sain » violé).
  Fix : champ `LUSRPlayersUnmeasured` (WARN par joueur non mesuré ; `os.ReadDir` KO = titre non
  mesuré), `publishLUSRGaugeIfComplete` ne republie la jauge que si le scan est COMPLET, cycle
  loggé WARN sinon. Tests scheduler + e2e (`seedUnmeasurableLUSRPlayer`, jauge figée).
- `[x]` **Badge périmé jusqu'à 24 h après « Recalculer »** : `RecomputeLUSRGapsForPlayer` ne
  rafraîchissait jamais la jauge (seul écrivain = cron). Fix : scan du joueur avant/après replay +
  `AddLUSRInteriorGapsGauge(delta)` clampé ≥ 0 (course avec le Set du cron documentée bénigne) ;
  best-effort, un scan raté n'ajuste pas la jauge et ne fait pas échouer le replay. Doc de la jauge
  (2 écrivains) mise à jour dans `skill_v2_metrics.go`.

Réfuté par la même revue (conception validée) : l'interruption d'un auto-heal mi-replay est bénigne
(reset watermark = sentinelle INSERT append-only, reprise propre au run suivant).

- `[x]` **CI branche — drift OpenAPI** (découvert au go/no-go, causé par le Lot 5) : les 2 paths
  admin étaient dans `openapi.yaml` mais PAS les 5 schémas de composants (`AdminLUSRGaps`,
  `AdminLUSRRecomputeResponse`, `LUSRGapItem`, `LUSRGapPlayer`, `LUSRGuardrailHealth`) →
  `TestOpenAPISchemaDrift_AggregatesAndReports` rouge en CI. Ajoutés (recette `OPENAPI_EMIT_OUT` du
  test) + `generate-types` rejoué (generated.ts additif). La note « hand-typed côté web, cohérent
  avec AdminWeaponCoverage » des Découvertes reste vraie pour les paths (description-only), mais le
  gate de drift exige les schémas de composants. + gofmt `internal/domain/admin_lusr_gaps.go`
  (lint ratchet CI, alignement de commentaires).
