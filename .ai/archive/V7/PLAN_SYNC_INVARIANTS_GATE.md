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
| `registry_without_participants` | FAIL | registry sans aucun participant (écriture partielle) |
| `medals_without_registry` | FAIL | medals_earned orphelines |
| `lusr_v2_orphan` | FAIL | row LUSR_V2 sans row LUSR (contrat dual-row, cf. RunDualRowSentinel) |
| `session_missing` | WARN | enrichment sans session_id post-pipeline |
| `performance_score_missing` | WARN | scores NULL (cold-start toléré ; croissance = batch en panne) |
| `skill_rank_missing` | WARN | match du joueur sans row match_skill_rank (PvE toléré ; croissance = désync watermark LUSR, incident 2026-06-03) |
| `citations_missing` | WARN | match avec médailles sans match_citations |
| `psa_missing` | WARN | match enrichi sans personal_score_awards — **confirmé par le gate : les PSA ne convergent PAS après delta-skip** (audit Phase 3) |
| `xuid_alias_missing` | WARN | xuid humain de participants sans alias gamertag. **Caveat** : vérifie shared.xuid_aliases ; post-ADR-0008 la source canonique est la DB globale → migrer le check quand le handle global sera câblé (Phase 4) |
| `pair_name_uuid` | WARN | pair_name UUID brut (trou catalogue assets, 2026-06-09) |

Extension 2026-06-10 (2e passe, depuis `.ai/ENRICHMENTS_CATALOG.md`) : +7
invariants couvrant les classes désync watermark LUSR, orphelins ART
(registry/medals), citations, PSA et alias gamertag.

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

### Phase 3 — PARTIELLEMENT LIVRÉE : étendre la couverture du gate

- [x] Invariant LUSR dual-row (`lusr_v2_orphan`) — définition unique
  test/runtime, mêmes sémantiques que `RunDualRowSentinel`.
- [x] `psa_missing` (WARN) — le gate a CONFIRMÉ que les PSA ne convergeaient
  pas après delta-skip. **FIX LIVRÉ (2026-06-10)** : étape `convergePSA` dans
  le pipeline post-sync (sélection `selectMatchesMissingPSA` bornée
  convergenceHorizon → GetMatchStats → ExtractPersonalScoreAwards → insert
  idempotent), marqueur terminal `psa_checked_at` sur player_match_enrichment
  (migration `player_match_enrichment_psa_checked_v1` + bootstrap schema.go)
  pour empêcher le re-fetch infini des matchs sans PSA extractibles. Backlog
  PSA intégré à hasConvergenceBacklog. Le rattrapage historique se fait par
  cycles (auto-backfill convergent, aucun backfill manuel). Validé rouge→vert
  par le gate (assertion stricte psa_missing=0 dans le scénario delta-skip) +
  compteurs expvar convergence_psa_pending/processed_total.
- [x] Scénario gate concurrent (`TestGate_ConcurrentSquadSync_Converges`) :
  3 RunDelta simultanés + passe d'idempotence + registry unique + zéro FAIL.
- [x] Tolérance PvE de `skill_rank_missing` (exclusion mode_category=firefight).
  Promotion WARN→FAIL toujours différée : le reliquat légitime (matchs
  non-2-équipes / déséquilibres skippés EP, 5-11 par joueur post-backfill)
  n'est pas distinguable par les colonnes actuelles.

### Phase 4 — LIVRÉE (2026-06-10) : dashboard monitoring admin

Décision user : PAS de notification in-app (« on ne peut rien y faire ») —
un **dashboard monitoring dans la partie admin** de l'app à la place.

- [x] Backend : `GET /api/v1/admin/invariants?title=` —
  `ServiceRegistry.RunDataInvariants` (LoadPlayers × resolveByGT ×
  CheckPlayer, best-effort par joueur avec `check_error`), handler
  admin-gated (RequireAuth + RequireAdmin + NoStore), 2 tests httptest.
- [x] Frontend : section « Intégrité des données » dans AdminPage — badge
  FAIL/WARN/OK par joueur, violations avec samples, bouton Vérifier,
  timestamp. i18n FR/EN via manifest common, query key centralisée.
- [x] Vérifié en live (smoke-boot) : **0 FAIL sur les 4 joueurs suivis** ;
  WARNs = classes connues (psa_missing → résorbé par la convergence PSA ;
  skill_rank_missing → backlog watermark LUSR ; xuid_alias_missing ×113 →
  caveat ADR-0008 ; pair_name_uuid ×12 → matchs du 09/06).

Reste (v2) — LIVRÉ 2026-06-10 (passe 5) :
- [x] Compteurs expvar : `invariants_runs_total` + gauges `invariants_fail_last`
  / `invariants_warn_last` (`observability.SetInt` ajouté) + WARN log si FAIL.
- [x] `xuid_alias_missing` lit `global.xuid_aliases` via la conn player
  (fallback legacy étiqueté). Découverte : les 113 manquants sont absents des
  DEUX sources → vrai backlog d'alias (pas du bruit).
- [x] Tendance : snapshot localStorage par (joueur, invariant), delta +N/-N
  affiché au run suivant.

## Backlog données restant (hors code — actions ponctuelles)

- [x] `lusr_v2_canonical_backfill --commit` exécuté (2436 matchs, 4 joueurs).
- [x] `backfill_registry_names` exécuté (24 noms, pair_name_uuid 12→1).
- [x] Backlog alias : convergence OPPORTUNISTE livrée (2026-06-10 soir) —
  convergePSA upserte shared.xuid_aliases pour TOUS les participants de
  chaque JSON match fetché (coût API nul, helper upsertAliasesFromMatchJSON,
  testé). Le backlog se résorbe avec le backlog PSA ; le reliquat éventuel
  (alias hors matchs PSA) reste visible au dashboard. Constat au passage :
  le chemin live UpsertXUIDAlias écrit vers globalDB dont le handle est nil
  en pratique (fichier global figé au 29/04) — à investiguer séparément.
- [ ] `psa_missing` (34/79/88) : se résorbe automatiquement par cycles de
  convergence dès que le serveur dev tourne (50/cycle).

## Garde-fous

- Le package invariants ne DOIT jamais dépendre d'internal/service ou
  internal/platform (importable partout, zéro cycle).
- Tout nouvel incident de données → réflexe : ajouter l'invariant
  correspondant AVANT le fix (red → green), comme un test de régression.
- Les sévérités WARN ne bloquent pas le gate ; toute promotion WARN→FAIL doit
  être justifiée dans ce document.

## Vérification finale (2026-06-10, revue 20 agents + fixes)

4 majeurs confirmés et corrigés : ordre engagement inversé (reverse ASC),
baseline tendance figée au mount (roulante via generated_at), invariants
globaux dupliqués ×N joueurs (split CheckPlayer/CheckShared + carte
« Données partagées »), test handler GetSquadEngagementSession manquant.
Couverture ajoutée : 12 checks exercés EN VIOLATION (invariants_violation_test),
convergePSA chemins d'erreur, SetInt, ChartFromOption, invariantsTrend, ordre
matchIds. Logs invariants routés vers logs/invariants.log (module dédié).

Limites assumées (documentées, non corrigées) :
- GET /admin/invariants peut créer/migrer une player DB déclarée absente
  (résolution pool = même effet qu'une visite de page joueur).
- Pas de timeout middleware dédié (WriteTimeout 30s serveur ; mesuré 0.1-0.2s
  pour 4 joueurs) ni de garde anti-runs-concurrents.
- Clé localStorage de tendance non namespacée par titre (latent, mono-titre).
- convergePSA : pas de borne de retries sur échec fetch persistant (atténué
  par rate-limiter + fetch-cache + ORDER BY récents d'abord).
