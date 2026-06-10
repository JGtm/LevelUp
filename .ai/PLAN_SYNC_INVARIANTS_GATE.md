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
- [ ] Scénario gate concurrent (errgroup) plutôt que séquentiel, pour couvrir
  la course réelle des watchers.
- [ ] Promotion WARN→FAIL de `skill_rank_missing` une fois la tolérance PvE
  affinée (filtre lifecycle/playlist).

### Phase 4 — À FAIRE : dashboard monitoring admin (décision user 2026-06-10)

Décision : PAS de notification in-app (« on ne peut rien y faire ») —
un **dashboard monitoring dans la partie admin** de l'app à la place.

- Backend : `GET /api/v1/admin/invariants` — exécute `invariants.CheckPlayer`
  pour chaque profil du titre (lectures RO, best-effort), retourne la liste
  des Reports + timestamp. Handler admin-gated (même garde que les autres
  routes admin).
- Frontend : section « Intégrité des données » dans la page admin — tableau
  joueur × invariant (sévérité, count, sample cliquable), badge global
  vert/orange/rouge, bouton refresh. Tendance simple (count précédent vs
  courant) en v2.
- Compteurs expvar `levelup.invariants.*` publiés au passage (gratuit, déjà
  le pattern RunDualRowSentinel).
- Câbler le handle DB globale pour migrer `xuid_alias_missing` vers la source
  canonique post-ADR-0008.

## Garde-fous

- Le package invariants ne DOIT jamais dépendre d'internal/service ou
  internal/platform (importable partout, zéro cycle).
- Tout nouvel incident de données → réflexe : ajouter l'invariant
  correspondant AVANT le fix (red → green), comme un test de régression.
- Les sévérités WARN ne bloquent pas le gate ; toute promotion WARN→FAIL doit
  être justifiée dans ce document.
