# PLAN — Invariants de données sync : gate au commit + convergence déterministe

> Créé le 2026-06-10 suite à la discussion « pourquoi un problème de sync tous
> les 3 jours ». Décision user : la détection doit avoir lieu AU COMMIT (CI),
> pas le soir dans les logs (trop tard, le end-user a déjà vécu le problème).
> Branche : `fix/chart-empty-states` (stackée après les fixes Escouade du jour).

## Objectif et critère de succès

Toute violation d'un contrat de données du pipeline sync multi-joueurs doit
faire échouer la CI (`go test -tags=integration`, job `go-coverage`, CGO) avant
merge. Critère : le scénario d'incident réel (delta-skip cross-player sans
convergence, 2026-05-27 et 2026-06-10) est reproduit en test et passe
uniquement si la convergence fonctionne.

## Constat fondateur (audit 2026-06-10)

- Le delta-skip cross-player est un design assumé (`loadKnownMatchIDs` source 2)
  avec un filet : `ensurePlayerEnrichmentRows` au début de `runPostSyncPipeline`.
- MAIS le déclencheur du pipeline en cycle « 0 match inséré »
  (`runConditionalPostSync`) ne couvrait pas le cas « enrichment manquant » :
  `hasMatchesNeedingScoreRefresh` ne voit que les rows existantes à scores NULL,
  `hasConvergenceBacklog` ne voyait que events/weapons. La convergence observée
  en prod reposait ACCIDENTELLEMENT sur des scores NULL cold-start permanents
  qui maintenaient `needsScoreRefresh=true`.
- Les invariants du pipeline n'étaient déclarés nulle part ; les violations
  étaient silencieuses (nil sans log) et découvertes par l'utilisateur final.

## Phases

### Phase 1 — LIVRÉE (2026-06-10) : bibliothèque d'invariants

`apps/go-api/internal/sync/invariants/` — package autonome (database/sql +
context uniquement), consommé par les tests ET, à terme, par la sentinelle
runtime. Une seule source de définitions.

| Clé | Sévérité | Contrat |
|---|---|---|
| `enrichment_missing` | FAIL | match en shared.match_participants (xuid) sans row player_match_enrichment |
| `participants_without_registry` | FAIL | row participants orpheline (classe ART) |
| `session_missing` | WARN | enrichment sans session_id post-pipeline |
| `performance_score_missing` | WARN | scores NULL (cold-start toléré ; croissance = batch en panne) |
| `pair_name_uuid` | WARN | pair_name UUID brut (trou catalogue assets) |

API : `invariants.CheckPlayer(ctx, playerDB, sharedDB, xuid) (Report, error)` ;
`Report.Failures()` pour le gate.

### Phase 2 — LIVRÉE (2026-06-10) : convergence déterministe + gate

- Fix moteur : `countSharedMatchesMissingEnrichment` ajouté à
  `hasConvergenceBacklog` (`internal/sync/convergence.go`) → un cycle
  « pur skip » déclenche désormais le pipeline (et donc le heal ensure-rows)
  de façon déterministe, plus par accident.
- Gate : `internal/sync/invariants_gate_integration_test.go` (tag integration)
  - `TestGate_DeltaSkip_EnrichmentConverges_integration` : 3 joueurs, MÊME
    match squad (xuids réels en participants), user0 insère, users 1-2
    skippent → `CheckPlayer` ×3 sans violation FAIL.
  - `TestGate_PureSkipCycle_TriggersConvergence_integration` : cadenasse le
    déclencheur (player DB vierge + match shared → missing ≥ 1).
- Fixture `TestConvergence_NothingWhenAllComplete` mise à jour : « complet »
  inclut désormais l'enrichment ; + `TestConvergence_BacklogWhenEnrichmentMissing`.
- CI : aucun changement requis — le job `go-coverage` exécute déjà
  `-tags=integration` avec CGO. Le gate est actif dès le merge.

### Phase 3 — À FAIRE : étendre la couverture du gate

- Invariant LUSR : intégrer la logique `RunDualRowSentinel` comme invariant
  (clé `lusr_dual_row`), pour une définition unique test/runtime.
- Invariant médailles/PSA : audit de la convergence de `personal_score_awards`
  pour les matchs delta-skippés (axe « objectif » du radar) ; ajouter
  `psa_missing` (sévérité à décider après audit).
- Scénario gate supplémentaire : sync simultané (errgroup) plutôt que
  séquentiel, pour couvrir la course réelle des watchers.
- Test page Escouade 2 coéquipiers au niveau service (la régression
  `851e10ef5` en test de service complet — les 3 tests unitaires de
  régression existent déjà dans teammates_squad_charts_test.go).

### Phase 4 — À FAIRE : sentinelle runtime + alerting

- Brancher `invariants.CheckPlayer` dans le cycle sync (cadence : 1/cycle ou
  1/heure), publier compteurs expvar (`levelup.invariants.*`).
- Remonter les FAIL en **notification in-app** (pas seulement logs) — exigence
  user : être prévenu sans aller chercher.
- Le WARN `performance_score_missing` devient utile ici : suivre la tendance
  (croissance = batch en panne).

## Garde-fous

- Le package invariants ne DOIT jamais dépendre d'internal/service ou
  internal/platform (importable partout, zéro cycle).
- Tout nouvel incident de données → réflexe : ajouter l'invariant
  correspondant AVANT le fix (red → green), comme un test de régression.
- Les sévérités WARN ne bloquent pas le gate ; toute promotion WARN→FAIL doit
  être justifiée dans ce document.
