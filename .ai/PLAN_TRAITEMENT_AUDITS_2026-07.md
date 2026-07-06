# PLAN — Traitement exhaustif des 4 audits du 2026-07-02

> Date de rédaction : 2026-07-02. Exécutant prévu : Opus. Superviseur : Guillaume.
> Sources (elles FONT FOI — ce plan les ordonne, il ne les remplace pas) :
> - [.ai/AUDIT_ARCHI_GO_API_2026-07.md](../AUDIT_ARCHI_GO_API_2026-07.md) — 173 findings (50 Majeurs numérotés + 118 Mineurs groupés) — cité « ARCHI n° »
> - [.ai/AUDIT_DETTE_DOC_2026-07.md](../AUDIT_DETTE_DOC_2026-07.md) — doc/durabilité/flags — cité « DETTE §x »
> - [.ai/AUDIT_QUALITE_SECURITE_GO_API_2026-07.md](../AUDIT_QUALITE_SECURITE_GO_API_2026-07.md) — sécurité/robustesse/tests — cité « QUALITE Bx/Mx/#x »
> - [.ai/V7/CODE_REVIEW_2026-07-02.md](CODE_REVIEW_2026-07-02.md) — qualité/duplication/code mort — cité « CR Cx/Ax »
>
> Objectif : traiter 100 % des findings des 4 audits, recommandations bonus/optionnelles incluses.
> Critère de succès global : chaque item du tracker porte un statut final ([x], [~] ou [!]) ;
> la matrice de couverture (§5) n'a aucune ligne orpheline ; tous les gates sont passés.

---

## 0. CONTRAT D'EXÉCUTION — À RELIRE AU DÉBUT DE CHAQUE SESSION, AVANT TOUT CODE

Ces règles sont NON NÉGOCIABLES. Elles existent parce que l'exécutant a un historique
documenté de traitement non séquentiel et partiel. Toute dérogation = tâche non terminée.

1. **Ordre strict des lots.** Les lots s'exécutent dans l'ordre du §3. Interdiction de
   commencer le lot N+1 tant que le lot N n'est pas CLOS (définition au point 6).
   Exception unique : D2 est volontairement différé (soak 7 jours) — il se traite quand
   son échéance arrive, sans bloquer la suite.
2. **Un lot = une unité de livraison atomique.** On ne « picore » pas 3 items faciles d'un
   lot pour passer à autre chose. Un lot commencé est terminé avant toute autre action.
3. **Vérifier sur pièces avant de coder ET avant de cocher.** Pour chaque item : rouvrir le
   fichier/la ligne cités par l'audit source, confirmer que le constat tient toujours (le
   code a pu bouger), puis corriger, puis re-vérifier. Jamais de fix « de mémoire ».
4. **Statuts autorisés dans le tracker** (aucune case ne reste vide à la clôture d'un lot) :
   - `[x]` fait et vérifié (commande de gate passée) ;
   - `[~]` couvert par un autre item/lot — indiquer lequel ;
   - `[!]` non traité — justification écrite OBLIGATOIRE dans le Journal (§6). Un `[!]` sans
     justification = lot non clos. Les `[!]` sont revus avec l'utilisateur en fin de lot.
5. **Zéro fix opportuniste hors lot.** Toute découverte en cours de route (bug, dette,
   amélioration) se note dans §7 « Découvertes » et N'EST PAS traitée immédiatement,
   sauf si elle bloque le gate du lot courant.
6. **Clôture d'un lot = 5 actions, dans cet ordre :**
   a. Gate du lot passé (commandes exactes du lot, sorties propres) ;
   b. Toutes les cases du lot statuées ;
   c. Ce fichier mis à jour (cases + Journal §6) et inclus dans le commit ;
   d. Entrée `.ai/thought_log.md` (règle CLAUDE.md) ;
   e. Point d'étape à l'utilisateur (résumé du lot : fait / skippé / découvert).
7. **Commits.** Minimum 1 commit par lot, préfixé `audit(<lot>):`. Les sous-lots de K font
   1 commit chacun. Jamais de `git stash` (règle projet). Demander avant tout push sur
   `main` (auto-déploiement prod).
8. **Si un gate échoue** : corriger dans le lot courant jusqu'au vert. Interdiction de
   désactiver/skipper un test pour passer le gate. Si le vert est réellement impossible
   (dépendance externe), statuer `[!]` + justification + en parler à l'utilisateur.
9. **Pas de nouveaux flags d'activation.** Les corrections se livrent actives (règle
   projet « pas de flags qui laissent une feature OFF »).
10. **Sessions longues / compaction de contexte** : ce fichier est la source de vérité de
    l'avancement. Au début de chaque session : relire §0, puis §6 (Journal) pour trouver le
    lot courant, puis reprendre à la première case non statuée de ce lot.

---

## 1. Pré-requis et branches

- [ ] P1 — Le chantier en cours sur `fix/h5-ui-adjustments-batch` (burst-lease, fichiers
  sync modifiés) est commité/landé AVANT de démarrer ce plan. Ce plan ne se démarre pas
  avec un working tree sale d'un autre sujet.
- [ ] P2 — Partir de `main` à jour.
- Branches (règle « 1 tâche = 1 branche, N commits ») :
  - **Lot S** : branche dédiée `fix/security-unauth-endpoints` — à merger/déployer vite
    (2 Bloquants exploitables). PRÉVENIR l'utilisateur avant le push main (auto-deploy).
  - **Lots A → N** : une seule branche `refactor/audits-2026-07`, un commit (ou plus) par
    lot, dans l'ordre. Merges intermédiaires vers main possibles après accord utilisateur
    (recommandé après B, après G, après K).
  - **Lot D2** (différé ≥7 j) : branche `refactor/adr0023-phase5` le moment venu.

## 2. Décisions — VALIDÉES PAR L'UTILISATEUR le 2026-07-02 (questionnaire interactif)

Ces décisions sont FERMES. Ne pas les re-questionner pendant l'exécution.

| # | Décision | Choix validé | Lot |
|---|---|---|---|
| DEC-1 | Feature `session-compare` morte (25 fichiers front+Go) | SUPPRIMER intégralement (front + endpoint + service + domain + openapi) | G3 |
| DEC-2 | Pipeline sync V1 | **SUPPRIMER V1 ENTIÈREMENT dans ce chantier** (choix utilisateur, plus ambitieux que la reco) : fallback auto + flag `LEVELUP_SYNC_PIPELINE` + code V1 non partagé avec V2. Voir D1c pour la méthode et les garde-fous | D1c |
| DEC-3 | `LEVELUP_PERSIST_BATCH` | Supprimer le flag ET la branche `=0` (chemin UPSERT ART-unsafe) | D1b |
| DEC-4 | Prestige / ADR 0005 | Acter l'activation (défaut ON), unifier les 2 gates sur une source unique (settings.json avec override env), mettre à jour l'ADR | C7 |
| DEC-5 | Politique docs/FR | ADRs + runbooks = EN-only (règle n°18 amendée) ; guides majeurs restent bilingues ; hook lefthook qui liste les paires désynchronisées | C8 |
| DEC-6 | Colonnes mortes `known_teammates_count` / `friends_xuids` | DROP au prochain rebuild (pas de writer à réactiver) | G14 |
| DEC-7 | Charts morts `SessionKDATimeline` / `SessionOcdrScatter` | Supprimer (noter dans le backlog UI si regret) | G11 |
| DEC-8 | Écritures livesync H5 (`csr_match.go`) hors persist/ | Router via la couche persist (convention unique), figée ensuite dans docs/ADD_TITLE.md | E8 |

## 3. Vue d'ensemble des lots (ordre d'exécution)

| Lot | Thème | Sources principales | Effort | Branche |
|---|---|---|---|---|
| S | Sécurité bloquante + majeurs auth | QUALITE B1-B2, M2-M4, mineurs | 0,5-1 j | fix/security-unauth-endpoints |
| A | Bugs utilisateur actifs + XSS + docs flags inversées | CR C1-C4, QUALITE XSS | 1 j | refactor/audits-2026-07 |
| B | Correctness lectures (_latest) + robustesse/avalements | ARCHI 39-45, QUALITE #3/#6/#8/#10 | 1,5 j | idem |
| C | Docs d'orientation (CLAUDE.md, ADRs, invariants, FR) | DETTE §1, §2.3 | 1-1,5 j | idem |
| D1 | Flags & guards (PERSIST_BATCH, suppression pipeline V1, os.Getenv, TODO-expiry) | DETTE §2.1-2.2, CR C4/A6 | 2,5-3,5 j | idem |
| E | ART résiduel (OpenSpartan, bulk UPDATE, tripwire) | ARCHI 15-16 + mineurs SQL | 1 j | idem |
| G | Purge code mort (~40 fichiers + api/gen) | CR A7-A9, ARCHI 26 + mineurs | 1-2 j | idem |
| F | Title-agnosticism (fuites HINF, manifests H5, film) | ARCHI 28-38, 13, 7 ; DETTE §2.4 | 2-3 j | idem |
| H | Repropagation/duplication (SQL, formatters, hooks) | CR A10-A13 | 1,5 j | idem |
| I | i18n (erreurs, scoreboard, streak→série) | CR A17-A19 + mineurs | 1 j | idem |
| J | Performance DuckDB (pools, N+1, LoadAll) | ARCHI 44-49 + mineurs perf | 1,5-2 j | idem |
| K | Structure & couches (extractions, god files/packages) | ARCHI 1-12, 14, 17-25 ; CR A1-A5 | 4-6 j | idem |
| L | Gouvernance/ratchets/contrat (archlint, golangci, OpenAPI) | ARCHI 26-27, CR A16/A20 | 1-1,5 j | idem |
| M | Tests (gaps ciblés) | QUALITE Axe 4 | 1-1,5 j | idem |
| N | Front structurel + résidus bonus | CR A14-A15 + mineurs §5 | 1,5-2 j | idem |
| D2 | ADR 0023 Phase 5 (après soak ≥7 j du log D1a) | DETTE §2.2, ARCHI 5-lié | 3 j | refactor/adr0023-phase5 |

Effort total estimé : ~23-30 jours-homme. Chaque lot est livrable indépendamment.

---

## 4. Lots détaillés

### LOT S — Sécurité (branche dédiée, à déployer vite)

Objectif : plus aucun endpoint mutant ou révélateur d'identité accessible sans auth.
Done : chaque route sous /api/v1 a une garde documentée ; tests httptest 401/403 neufs verts.

- [ ] S1 — QUALITE B1 : envelopper `settingsHandler.Mount` sous `RequireAuth`+`RequireAdmin`
  (`server.go:1271`, `handlers/settings.go:103-114`). Test httptest : PATCH /settings
  anonyme → 401 ; les 4 POST associés idem.
- [ ] S2 — QUALITE B2 : déplacer `NewProgressionBackfillHandler` sous le groupe `/admin`
  (RequireAuth+RequireAdmin) (`server.go:972`). Test 401 anonyme.
- [ ] S3 — QUALITE « cause racine » : revue EXHAUSTIVE de tous les `Mount`/`r.Get`/`r.Post`
  montés sous /api/v1 hors groupe protégé. Produire un tableau route→garde (dans le message
  de commit ou un doc court). Vérifier que le no-op demo/single-user est préservé.
- [ ] S4 — QUALITE M2 : `GET /players` filtré par `filterOwnedPlayers` (comme /bootstrap)
  ou gated RequireAuth (`bootstrap_service.go:346`, `server.go:947`).
- [ ] S5 — QUALITE M3 : `/_diag/auto-sync/probe` → ajouter `RequireAdmin`, retirer
  `refresh_token_head`/`tail` (garder le sha) (`handlers/admin_auto_sync.go:106-146`).
- [ ] S6 — QUALITE M4 : diagnostics par joueur (`HealthHome`, `DiagCSR`, `DiagProgression`,
  `server.go:953-965`) → RequireAuth + ownership.
- [ ] S7 — QUALITE A1-m1 : `RequirePlayerOwnership` — slug inconnu avec session existante →
  403 au lieu de fail-open (`require_player_ownership.go:56-60`) + test.
- [ ] S8 — QUALITE A4-m2 : `/setup/players` & `/setup/smoke-test` → RequireAuth par
  cohérence (gardes internes conservées) (`server.go:1280`).
- [ ] S9 — QUALITE mineurs tokens : `scripts/warm_bp_assets/main.go:176` → logger « OK »
  sans préfixe de token ; `cmd/get-token` → commentaire d'avertissement (sortie à ne
  jamais capturer) ou build tag dev.

Gate S : `go build ./... && go test ./internal/api/...` + les nouveaux tests httptest ;
grep de contrôle : aucun `Mount(r)` nu restant sans justification dans le tableau S3.

### LOT A — Bugs utilisateur actifs, intégrité LUSR, docs de flags, XSS

Objectif : ce qui est faux À L'ÉCRAN aujourd'hui + le risque d'intégrité immédiat.

- [x] A1 — CR C1 : supprimer le `perfTier()` local inversé de
  `TimeseriesFormCharts.tsx:51-57`, importer `perfScale`
  (`lib/accessibility/scales/instances.ts`). Vérification visuelle du tab Forme.
- [x] A2 — CR C2 : `CareerTopMatchesTable.tsx:124-129` — badges outcome par code/`outcomeKey`
  + `getOutcomeColor` (modèle `ExplorerMatchesTable.tsx:317-327`) ; corriger aussi le
  `toLocaleDateString('fr-FR')` figé (l.99).
- [x] A3 — CR C3 : rerouter le backfill LUSR v1 → `RecomputeLUSRCanonicalForPlayer` (v2)
  dans `handlers/backfill.go:334` et `cmd/levelup/cmd_backfill.go:411,431` ; supprimer
  `RunBackfillLUSR` (`engine_backfills.go:182`) et `upsertLUSRRatingsLegacy`
  (`skill_rating_loaders.go:232`).
- [x] A4 — CR C4 + DETTE TOP2 : réécrire les 5 docs inversées des flags :
  `engine_options.go:130-131`, `engine.go:128-133`, `engine_batch_path.go:16`,
  `sync/v2/doc.go:17`, `cmd/server/main.go:1108-1111` — défaut réel (ON / V2), `=0`/`=v1`
  = kill-switch. (Le RETRAIT des flags est en D1 ; ici on rend la doc vraie.)
- [x] A5 — QUALITE XSS (#9) : promouvoir `escapeHtml()` dans `apps/web/src/components/charts/_utils.ts`
  et l'appliquer à toute interpolation non constante des formatters tooltip ECharts
  (~40 sites ; les 7 à contenu tiers en premier : `squadMapHeatmapChart.ts:78`,
  `squadEfficiencyChart.ts:88`, `OutcomeSequenceTape.tsx:103`,
  `RelationsMomentsHeatmap.tsx:83`, `squadFragBreakdownChart.ts:117` + 2 autres relevés).
  Réutiliser/déplacer le helper existant de `BarStackedChart.tsx:210`.

Gate A : `go test ./internal/sync/... ./internal/api/...` ;
`cd apps/web && npm run typecheck && npm run lint && npm run test` (vitest hors sandbox) ;
grep : `RunBackfillLUSR` → 0 occurrence.

### LOT B — Lectures rating `_latest` + robustesse (avalements silencieux)

Objectif : plus aucune lecture brute de `match_skill_rank` sur un chemin de lecture ; plus
aucun avalement silencieux à impact durable.

Famille _latest (piège ADR 0026) :
- [x] B1 — ARCHI 39 : `queries_home_citations.go:100` (Q26) → join `match_skill_rank_latest`.
- [x] B2 — ARCHI 40 : `queries_career.go:172` (Q5) → `_latest` ; retirer le bricolage
  winProb de `mergeHistorySkillRanks` devenu inutile.
- [x] B3 — ARCHI 41 : `compare_repo.go:129` et `:170` (ATH LUSR) → `_latest`.
- [x] B4 — ARCHI 42 : `leaderboard_repo.go:65` → `_latest`.
- [x] B5 — ARCHI 43 : `patterns_repo.go:245` (`loadSkillRanks`) → `_latest`.
- [x] B6 — ARCHI 2 (partie lecture) : `post_sync_deltas_snapshot.go:142-153` → `_latest`
  + tiebreak déterministe à `start_time` égal dans le détecteur de transition (l.151).
  (L'extraction en `PlayerSnapshotRepo` part en K1a — ne pas la faire ici.)
- [x] B7 — ARCHI mineurs « ré-implémentations du latest » (5 sites) :
  `queries_home_citations.go:448` (Q26g), `player_matches_loaders.go:190` (match_csrs),
  `queries_squad.go:10` (MAX expected_win_prob), `halo5_career_source.go:34`,
  `csr_coverage_repo.go:63`.
- [x] B8 — Garde-rail : test grep interdisant `FROM match_skill_rank` (et `match_csrs`,
  `player_csr_snapshots`, `pve_match_stats`) hors vues `_latest`, writers persist et
  allowlist explicite datée (modèle `no_art_patterns_test.go`).
- [x] B9 — ARCHI 45 : fan-out notifications `registry_notifications.go:162` →
  `OpenReadForQuery` (incident 2026-06-01 documenté dans db.go).

Robustesse (QUALITE Axe 3) :
- [x] B10 — QUALITE #3 : `worldenrich/wiring.go:64` — ne plus avaler l'échec de persistance
  du RT roté : `slog.ErrorContext` + compteur expvar + retry simple. (Classe d'incident
  ADR 0023 « auth morte définitivement ».)
- [x] B11 — QUALITE #6 : `sync/engagement.go:470-483` — erreur de requête historique →
  logger + échouer le calcul du match (pas de score faux persisté) ; vérifier `rows.Err()`.
- [x] B12 — QUALITE #8 : `api/server.go:99-103` — `familyXUIDResolver` : logger le fichier
  groups corrompu au lieu de retourner nil muet.
- [x] B13 — QUALITE #10 : `scheduler/data_health_check.go:215-283` — les sondes en erreur
  remontent en « unhealthy » (ou au minimum loggées + compteur), plus de `continue` muet.
- [x] B14 — QUALITE : `sync/skill_chain_provider.go:61` — valider les classifiers de titre
  au boot (fail-fast) au lieu du panic au 1er match.
- [x] B15 — QUALITE : mapper HTTP central pour `ErrCapabilityNotSupported` (middleware/helper
  Huma → 503/204) ; brancher les handlers existants dessus.
- [x] B16 — QUALITE dette logging (mineurs, mécanique) : ~42 `slog.Error` → `ErrorContext` ;
  15 sites `"err", err.Error()` → error brute ; params HTTP invalides défaultés → log warn
  (`notifications.go:320`, `catalog.go:107,140`) ; best-effort journalisés
  (`sync/career.go:137`, `sync/backfill_weapons.go:57,149`, `sync/engagement.go:169`,
  `auth_xbox_oauth.go:229,245`, `catalog_fetcher_service.go:282`, `ops/catalog_refresh.go:278`).

Gate B : `go build ./... && go test ./...` + B8 vert ; grep `FROM match_skill_rank[^_]`
hors allowlist → 0.

### LOT C — Documents d'orientation & invariants

Objectif : les documents lus « avant toute action » redeviennent vrais ; les invariants
critiques sont écrits aux points de mutation.

- [x] C1 — PRÉ-EXÉCUTÉ le 2026-07-02 (session de planification : CLAUDE.md réécrit +
  skills mis à jour). Au passage du lot : VÉRIFIER la fraîcheur (le code aura bougé via
  les lots S-D1) et statuer `[~]` si rien à corriger. Contenu original de l'item : réécrire
  CLAUDE.md — purger : § Environnement Python, § Commandes
  Utiles Python, règles 2-17 mortes, § src/ai/, § Modules Supprimés v4.1, § Stack Python,
  anti-patterns à exemples Python ; corriger § Architecture des Données →
  `data/titles/{slug}/warehouse/` + `data/titles/{slug}/players/{gt}/` (ADR 0008) ; exemple
  MCP ATTACH corrigé ; conserver/rafraîchir : liste ADRs, règle auth 0023 (sans le marqueur
  fantôme tant que D1a n'est pas livré — coordonner), règle Collect→Persist, branches Git,
  skills, règles front (§20 couleurs). Les seuils (500 L/80 L) restent, reformulés Go/TS.
- [x] C2 — DETTE TOP5 : `.ai/project_map.md` — soit mise à jour réelle, soit bandeau
  « HISTORIQUE — gelé au 2026-04-28, ne fait plus foi » + suppression des affirmations
  démenties (« Film Chunks : NON EXPLOITABLES », règles Python).
- [x] C3 — DETTE §1.1 : rotation du thought_log — archiver les entrées avant 2026-Q2 vers
  `.ai/archive/thought_log_2025-2026Q1.md` (pattern archive existant) ; noter la règle de
  rotation trimestrielle dans CLAUDE.md.
- [x] C4 — DETTE TOP8 : corriger les 2 pointeurs `0014-b-swap` → 0016
  (`sharedprovider/doc.go:23`, `baseline_red_integration_test.go:93`) + rafraîchir le
  statut « commit 2/9 » (B-swap livré).
- [x] C5 — DETTE TOP9 : écrire les 4 invariants aux points de mutation :
  INSERT-only sur `SharedPersister.Persist` ; « pas de shared writer lease pendant la
  phase 6 » dans le runner post-sync V2 ; « jamais sql.Open direct sur ce chemin —
  mono-process RO/RW » sur l'API publique `sharedprovider/provider.go` ; recette 3-étapes
  ADR 0019 référencée depuis `enrichmentFields()`. Bonus : rappel du piège de scope dans
  `soloFilterStore.ts` (le récit est dans createFilterStore.ts).
- [x] C6 — DETTE §1.3 : doc.go/README manquants : `internal/sync` (v1), `internal/migration`,
  `internal/games` (+ `halo_5`), `internal/progression`, `internal/domain`,
  `internal/api/handlers` (patron : `persist/doc.go`) ; compléter
  `analysis/temporal/README.md` (ComputeEngagementScore, ComputeEngagementCoefficient,
  types + erreurs sentinelles).
- [x] C7 — DETTE TOP7 / DEC-4 : Prestige — mettre à jour ADR 0005 (activation actée),
  unifier `prestige.IsEnabled()` et `loadPrestigeEnabled()` sur une source unique
  (settings.json + override env), le hook de sync lit la même source que les surfaces HTTP.
- [x] C8 — DETTE TOP10 / DEC-5 : politique docs/FR écrite dans CLAUDE.md (ADRs+runbooks
  EN-only ; guides bilingues) ; rattraper `docs/CITATIONS.md` EN (FR a 4 mois d'avance) ;
  statuer COMMENDATIONS.md et RUNBOOK_OPS_DUCKDB (EN-only assumé) ; réparer les liens
  relatifs de `docs/FR/ARCHITECTURE_V6.md` ; hook lefthook qui liste les paires
  désynchronisées dans le diff (warning, pas bloquant).

Gate C : grep dans CLAUDE.md → 0 occurrence de `pytest|pandas|polars|streamlit|src/|.venv|
data/warehouse|scripts/sync.py` ; liens docs/FR valides ; relecture par l'utilisateur
proposée (c'est SON fichier d'instructions).

### LOT D1 — Flags & guards (cycle de vie)

Objectif : plus aucun « forever guard » sur le chemin critique ; flags découvrables.

- [x] D1a — DETTE TOP4 (pré-requis de D2) : implémenter le warn log `legacy_source_used`
  (+ compteur expvar) sur TOUTES les lectures de fallback legacy auth
  (`sync_meta.oauth_refresh_token`, `msal_token_cache`, `SPNKR_OAUTH_REFRESH_TOKEN_*`) —
  tel que CLAUDE.md le documente déjà. Noter la date de mise en prod dans le Journal §6 :
  D2 se déclenche ≥7 jours après.
- [x] D1b — DETTE TOP3 / DEC-3 : supprimer `LEVELUP_PERSIST_BATCH` et la branche `=0`
  (chemin `insertFetchedMatch` / UPSERT legacy `sync/writes.go:68`) — 8 sites recensés
  (`cmd/levelup/cmd_sync.go:68-70`, `engine.go:338`, `scheduler/auto_sync.go:336`,
  `handlers/sync_handler.go:210`, `engine_batch_path.go`, +3). Supprimer aussi
  `processMatch` legacy test-only (CR A9) et ses tests V1 qui exercent un chemin mort.
- [x] D1c — DEC-2 (VALIDÉ : suppression complète) : supprimer le pipeline V1 ENTIÈREMENT —
  fallback automatique (ARCHI 50), flag `LEVELUP_SYNC_PIPELINE` et sa lecture
  (`auto_sync.go:437-444`), chemins d'orchestration V1 non partagés avec V2.
  MÉTHODE OBLIGATOIRE (gros diff dans sync/, prudence) :
  (1) cartographier d'abord ce que V2 RÉUTILISE du moteur (`SyncEngine`, engine.run,
  mixins) vs ce qui est V1-only (orchestration séquentielle, fallback) — ne supprimer QUE
  le V1-only ; si `engine.run` s'avère V1-only, le noter : K2b devient `[~]` ;
  (2) suppression par commits séparés (flag/fallback, puis orchestration, puis tests V1) ;
  (3) gate renforcé : `go test -tags=integration ./internal/sync/...` + un sync live
  complet en local (delta + backfill court) AVANT de clore l'item ;
  (4) mettre à jour ADR 0027 (V1 supprimé, date) + `sync/v2/doc.go`.
  En cas de découverte bloquante (couplage V1/V2 plus profond que prévu) : repli autorisé
  sur « fallback auto supprimé + retrait daté » MAIS en informer l'utilisateur au point
  d'étape, pas en silence.
- [x] D1d — DETTE §2.1 : documenter le cycle de vie des autres flags
  (`LEVELUP_PERSIST_BATCH_ASYNC`, `MULTI_TITLE_API_ENABLED`, `LEVELUP_EVENTS_CONVERGENCE`,
  `LEVELUP_CONTRACT_VALIDATE`) — modèle `shared_reader_legacy.go:30-34` (date de
  basculement + date cible de retrait + critère mesurable). FAIT : triplet ajouté aux 4
  sites de lecture (2 vrais kill-switches BATCH_ASYNC/EVENTS_CONVERGENCE avec date cible
  >= 2026-Q4 + critère mesurable ; MULTI_TITLE classé gate de rollout ; CONTRACT_VALIDATE
  classé diagnostic dev/CI permanent sans retrait). docs/CONFIGURATION.md (+FR) : défaut
  `(off)`→`on` corrigé pour BATCH_ASYNC + 4 lignes de flags ajoutées. Tension règle 11 sur
  MULTI_TITLE notée en §7.
- [x] D1e — CR A6 : centraliser les 41 `os.Getenv` hors `internal/config` dans
  `config.AppConfig` au boot, injecter (élimine la double lecture scheduler/handler de
  PERSIST_BATCH — devient sans objet après D1b pour ce flag, reste vrai pour les autres).
  FAIT (cible réelle = les lectures DIVERGENTES/multi-sites, pas le littéral ~0) : (1)
  supprimé le mort+divergent `handlers.MultiTitleAPIEnabled()` (server lit `cfg.MultiTitleAPIEnabled`)
  + son test ; (2) supprimé le mort `notify.EnvWebhookURL()` + son test ; (3) extrait
  `config.DiscordWebhookURLFromEnv()` (précédence env UNIQUE), consommé par le loader config
  ET par notify/validation (fin de la triple lecture `DISCORD_WEBHOOK_URL` qui bypassait
  `LEVELUP_DISCORD_WEBHOOK_URL`) ; (4) centralisé les kill-switches scheduler
  `PersistBatchAsync`/`EventsConvergence`/`EventsConvergenceMax` dans AppConfig (fin de la
  triple lecture `LEVELUP_PERSIST_BATCH_ASYNC` main/wiring/scheduler) ; (5) garde-rail
  `internal/config/env_centralization_test.go` interdit toute relecture `os.Getenv` de ces
  flags hors config. Baseline prod hors config : 34 → 29 (résidu justifié §7). Gate :
  build+vet OK, tests config/notify/validation/handlers/scheduler verts,
  `-tags=integration -p 1 ./...` = exit 0.
- [x] D1f — DETTE reco 7 : généraliser `TODO(expiry:YYYY-MM-DD)` + lint léger qui échoue à
  date dépassée (précédent : `season_pass_repo_tracks.go:254`). Trier les 513 TODO/FIXME :
  dater ceux qui référencent des phases mortes, supprimer les caducs (passe rapide, pas
  d'exhaustivité ligne-à-ligne exigée — l'outillage est le livrable). FAIT : lint
  `internal/archlint/todo_expiry_test.go` (calque `no_slug_comparison_test.go` ; regex
  `TODO\(expiry:YYYY-MM-DD\)` ; `now` injectable via `LEVELUP_TODO_EXPIRY_NOW` ; scanne toute
  la racine go-api ; auto-exclusion du scanner). Vérifié DANS LES DEUX SENS : vert au
  2026-07-03, ROUGE à `now=2026-09-01` (attrape `season_pass_repo_tracks.go:254` échu
  2026-08-01). Triage rapide : 1 caduc supprimé (`persist/worker.go` — marqueurs « TODO Phase
  1.5+ » sur PlayerPersister/PVEPersister/MetadataPersister, désormais tous implémentés).
  Résidu documenté §7 (cluster P4 ADR 0006 « retirer *100 », TODO session_compare = fichiers
  DEC-1, stub WithPrestigeHook). Gate : build OK, `go test ./internal/archlint/...` vert.

Gate D1 : `go build ./... && go test ./...` ; grep `LEVELUP_PERSIST_BATCH` → 0 ;
grep `os.Getenv` hors `internal/config` → baseline notée au Journal (cible : ~0 hors
main/cmd bootstrap).

### LOT E — ART résiduel & écritures à risque

Objectif : plus aucun chemin prod du pattern déclencheur ART #23046 ; tripwire étendu.

- [x] E1 — ARCHI 15 : router `writeOneMatch` de l'import OpenSpartan
  (`openspartan_import_service.go:318`) vers `persist.SharedPersister` (INSERT-only +
  pre-check registry), comme le livesync H5. Retirer l'entrée allowlist du tripwire.
  FAIT : `writeOneMatch` construit un `persist.MatchBatch` (SetMatch+AddParticipants+
  AddMedals+AddMatchCSRs) et appelle `NewSharedPersister(sharedDB).Persist` — 1 transaction
  INSERT-only atomique + idempotence (skip si match_id existe), remplace les 4 appels
  `sync.Insert*/Upsert*` per-helper (dont ON CONFLICT). Converter `SharedCSRRowToMatchCSRInsert`
  exporté (réutilisé, pas dupliqué). Entrée allowlist : SANS OBJET (les ON CONFLICT vivent dans
  `writes.go`, allowlisté là ; openspartan n'avait pas d'entrée). Fixture test complétée
  (colonnes batch match_intensity/backfill_completed/backfill_bits/kill-mechanics). Gate :
  `TestOpenSpartanImport_EndToEnd_WritesAllRowFamilies` vert ; `-tags=integration -p 1`
  persist/sync/service vert.
- [x] E2 — ARCHI 16 : `backfill_registry_names.go:98` — convertir l'UPDATE bulk multi-row
  nu sur match_registry en row-by-row par match_id (ou réserver au CLI serveur-arrêté,
  décision au code) ; ÉTENDRE la regex du tripwire aux UPDATE multi-row nus sur les
  tables critiques. FAIT (choix A1 = row-by-row, garde le chemin API admin) :
  `backfillPairNamesByConstruction` fait un SELECT des match_ids + pair_name construit puis
  N `UPDATE ... WHERE match_id = ?` sérialisés. Tripwire étendu : `reUpdateRawSQL` +
  détection « littéral SQL `UPDATE <table critique>` SANS placeholder `?` » = bulk set-based
  nu (ancrage backtick, faux positifs évités — vérifié sur events_completion/pve/writes qui
  lient tous match_id=?). Signature précise (seul le bare set-based n'a aucun `?`).
  `TestBareBulkUpdateDetection_Sanity` valide les 2 sens. Gate : tripwire + backfill_registry
  vert, `-tags=integration -p 1 ./internal/sync/...` = exit 0.
- [x] E3 — ARCHI mineur : `no_art_patterns_test.go:184` — retirer l'exclusion `ops/`
  (hypothèse « exécuté hors serveur » fausse : plomberie média ops tourne in-process).
  FAIT : exclusion `ops/` retirée des 3 tests tripwire + 3 commentaires de justification
  corrigés (doc-inversion). DÉCOUVERTE MATÉRIELLE : retirer l'exclusion exposait un VRAI
  risque ART que E2 venait de rendre détectable — `ops/lying_bits_reset.go` faisait 3 bulk
  UPDATE multi-row nus sur match_registry (bits inlinés `%d`, aucun `?`), IN-PROCESS (action
  admin data-quality), avec un commentaire affirmant à tort « pas de risque ART ». Convertis
  en row-by-row par match_id (SELECT match_ids → UPDATE `WHERE match_id = ?`). Vérif
  empirique : lying_bits était la SEULE exposition (catalog_refresh = SELECT-then-write tables
  non protégées ; seed/archive/restore = non protégées/dynamiques) → aucune allowlist ajoutée.
  Gate : `TestResetLyingBits` vert (comportement préservé), tripwire complet + ops verts, vet OK.
- [x] E4 — DETTE §2.5 : `TestAllowlistJustifiesEverything` passe de warning à erreur.
  FAIT + NUANCE : le test lisait le contenu BRUT (commentaires inclus), donc une
  « justification » vivant uniquement dans un commentaire passait — alors que le scan
  principal strippe les commentaires. Rendu (1) BLOQUANT (`t.Logf`→`t.Errorf`) ET (2)
  cohérent avec le scan (`stripGoComments`). Entrée `internal/persist/doc.go` retirée
  (ses seuls patterns étaient en commentaire → morte). `internal/sync/writes.go` conservée
  (ON CONFLICT réel en code, l.68/133). Gate : allowlist + scan principal + tripwire verts.
- [x] E5 — ARCHI mineur : `progression/profile/queries.go:40` — SQL de lecture HTTP →
  platform/duckdb + filtre temporel canonique (COALESCE timezone, croisé H1).
  FAIT (filtre canonique) : les 5 sites `start_time` bruts (count/radar/awards-PhaseA/FKFD/
  engagement) passent au fragment timezone-canonique. Helper EXPORTÉ
  `duckdb.StartTimeCanonicalSQL(alias)` créé (const existante refactorée pour le réutiliser →
  source unique côté platform/duckdb, rule 6). [~] pour le déplacement des LITTÉRAUX SQL vers
  un repo platform/duckdb : les 5 méthodes sont de l'ORCHESTRATION cross-DB (Phase A shared +
  Phase B player + mapping awards) dont la CONNEXION passe déjà par le SharedReader (ADR 0016,
  couche correcte) ; l'extraction pure-repo chevauche H1 (refactor read-layer timezone) → à
  faire en H1 pour éviter double-travail. `analysis/match_filter.go` garde sa copie inline
  (analysis ne peut importer platform/duckdb) → unification H1. Gate : build+vet OK, profile +
  platform/duckdb `-tags=integration -p 1` verts.
- [x] E6 — ARCHI mineur : bare connects RO sur player DB potentiellement tenue RW :
  `worldenrich/wiring.go:33`, `platform/auth/pool/discovery.go:229` → pattern
  `OpenReadForQuery` / provider. FAIT : Site 1 (`sql.Open ...access_mode=read_only`) et
  Site 2 (`duckdb.OpenReadOnly`) → `duckdb.OpenReadForQuery` (réutilise un handle en cache
  si la DB est tenue RW, sinon ouvre RO — release func). GOTCHA résolu (le `&duckdb.DB{}` de
  la carto était une hallucination) : `OpenReadForQuery` rend `*sql.DB` mais les helpers auth
  prenaient `*DB` → ajout de variantes `Read{MSALCacheJSON,OAuthRefreshToken}FromSQL(*sql.DB)`
  + helper unique `readSyncMetaValue` ; les versions `*DB` délèguent (zéro impact sur leurs
  ~10 callers). worldenrich : imports morts retirés (`database/sql`, driver blank) + lectures
  sync_meta DRY via les helpers. Gate : build+vet OK, queries_auth + pool `-tags=integration -p 1` verts.
- [!] E7 — ARCHI mineur : `sync/schema.go:22` — DDL bootstrap → `internal/migration`.
  DIFFÉRÉ (blocage réel, règle plan-execution 9 — l'item est MAL LABELLISÉ « mineur »).
  Vérif sur pièces : (1) `EnsurePlayerSchema`/`EnsureSharedSchema` s'exécutent à CHAQUE
  `OpenPlayerDB`/`OpenSharedDB` (quick-provision idempotent) — mécanisme DISTINCT du migration
  runner ; les outils/tests qui ouvrent une DB SANS runner en dépendent. (2) Le DDL est
  dupliqué-mais-aligné avec les steps `create_base_*_schema`, eux-mêmes EN TRANSITION
  (Phase 1.5 b23/b25 : `create_base_player_schema` existe dans `internal/migration` ET
  `games/halo_infinite/migrations` — ownership en cours de bascule global→title-owned).
  (3) `EnsureSharedSchema` porte une logique de création de vues AU BOOT qui corrige des bugs
  prod documentés (attach RO/RW 2026-05-27 « Cannot execute CREATE on read-only », xuid bruts
  2026-05-30). Déplacer = recâbler le boot/provisioning de TOUTES les DBs sur le migration
  runner → risque de régression de ces bugs + collision avec la transition b23/b25 en cours.
  Disproportionné pour « ARCHI mineur ». À FAIRE en chantier dédié testé APRÈS stabilisation
  b23/b25. Signalé à l'utilisateur.
- [x] E8 — DEC-8 : `games/halo_5/livesync/csr_match.go:69` — router les écritures
  per-match H5 via la couche persist (BatchBuilder/Persister dédié). Convention à figer
  en F14 (ADD_TITLE.md). FAIT : `writePerMatchCSR`/`writeCareerSR` (ExecContext directs sur
  match_skill_rank + career_progression, hors couche persist) → builders
  `buildPerMatchCSRInsert`/`buildCareerProgressionInsert` + `PlayerPersister.PersistPerMatchRating`
  (Persister DÉDIÉ, nouveau). BLOCAGE RÉSOLU que la carto avait manqué : `PlayerPersister.Persist`
  exige `Enrichment != nil` comme ancre d'idempotence et SKIP si l'enrichment 'live' du match
  existe → inutilisable pour un hook POST-SCORE (l'enrichment est déjà écrit). D'où le persister
  dédié INSERT-only SANS ancre enrichment : match_skill_rank append-only + career_progression
  dédupliqué par (xuid, recorded_at). csr_match : 0 ExecContext direct restant. Convention
  ADD_TITLE.md = [~]→F14. Gate : `TestPerMatchCareerSR` (rewire) + H5 livesync + persist
  `-tags=integration -p 1` verts.

Gate E : `go test ./... && go test -tags=integration ./internal/persist/... ./internal/sync/...` ;
tripwire étendu vert ; allowlist ART réduite et bloquante.

### LOT G — Purge du code mort (« dead code museum »)

Objectif : 0 module mort (règle projet) ; exécuté AVANT F/H pour réduire la surface des
migrations suivantes. Chaque suppression : retirer code + tests + imports + entrées
openapi/migrations associées, puis build+tests.

- [x] G1 — ARCHI 26 : supprimer `internal/api/gen` (2 536 L, 0 importeur) + cible make +
  3 exclusions tooling + corriger le message du drift-test (`make gen` → chaîne Huma).
  FAIT : `internal/api/gen/types.gen.go` (0 importeur, confirmé carto) + sa config génératrice
  `api/oapi-codegen-types.yaml` supprimés ; Makefile (var OAPI_CODEGEN + cible `gen` + exclusion
  `api/gen` du test-unit), `.golangci.yml` (exclusion lll/revive), `scripts/coverage_filter.sh`
  (comment + exclusion), `docs/testing.md` (ligne tableau), allowlist `no_attach_on_social_test.go`
  nettoyés ; message drift-test corrigé (`make gen`→ Huma dérive le contrat). NB : `internal/ln/`
  (output stale de l'ancienne config oapi, dir absent) = Découverte §7, hors périmètre G1. Gate :
  build+vet OK, 0 ref `api/gen` Go, tests no_attach + drift verts.
- [x] G2 — CR A7 : cluster home legacy — supprimer les 10 exports morts (`ComputeKPIs`,
  `ComputeTrend`, `BuildHeroCard`, `BuildHighlights`, `BuildSessionSummaries`,
  `BuildRecentMatches*` x4) + tiles legacy transitifs + leurs tests ; CONSERVER
  `mapImageURLFromRegistry`, `mmrDelta`, `float64PtrVal`, `intPtrIfPos` ; corriger la doc
  fausse de `home_canonical.go:4-18`. FAIT (chirurgie fonction-par-fonction — la carto Haiku
  s'était TROMPÉE en annonçant des suppressions de FICHIERS entiers alors que `home_kpis.go`
  mêle du VIVANT `BuildSpartanIdentity`/career-rank). Vérifié sur pièces : cluster à 0 caller
  externe (les `*FromCanonical` ne délèguent plus). Supprimé les 10 exports + `bestKDAMatch`/
  `bestMMRUnderdogWin` + `home_highlights_tiles.go` (6 tiles) ; CONSERVÉ les helpers PARTAGÉS
  (dominantKey, selectHighlightWindow, highlightPerf/KDAColor, distinctSessionLabels, latest*,
  earliest*, + les 4 nommés). Tests morts retirés des 5 fichiers (chirurgie sélective, KEPT
  BuildSpartanIdentity/BuildRecentMedia/ComputeKPIStats). Doc `home_canonical.go` corrigée
  (délégation legacy → autonomie canonical). Gate : build OK, `go test ./internal/analysis/` vert.
- [x] G3 — CR A8 / DEC-1 : supprimer la feature session-compare entière : 17 fichiers
  front (`features/session-compare/`), `handlers/session_compare.go`,
  `service/session_compare_service.go` + 3 helpers, `domain/session_compare.go`, entrée
  openapi.yaml, query key, manifest §compare. (~25 fichiers.)
  FAIT — CORRECTION DE PLAN MAJEURE (signalée §7) : le plan disait « supprimer
  domain/session_compare.go + service + 3 helpers » MAIS ces types+helpers sont PARTAGÉS
  avec la page SESSION-DETAIL vivante (session_page) : `SessionCompareEntry`,
  `SessionCompareMetricRow`, `SessionParticipationAxis`, `SessionMatchPoint`,
  `SessionCompareSuggestion` + les builders (buildCompareEntryWithObjectives, buildCompareMetrics,
  extractSessionLabels…) + les stat/table/participation helpers. Les supprimer aurait CASSÉ
  session-detail + le build front. G3 corrigé = supprimer SEULEMENT la couche compare-only :
  handler+test, orchestration `Compare`/`NewSessionCompareService` (struct), buildMapTable/
  buildModeTable, types Request/Response/MapRow/ModeRow, route openapi /pages/session-compare,
  17 fichiers front, query key ; PRÉSERVER l'infra session-summary partagée (trimmée dans
  domain/session_compare.go + doc mise à jour). generated.ts régénéré, types.ts nettoyé.
  Gate : go build+vet+`-tags=integration -p 1` service/api/domain vert (drift openapi OK) ;
  front typecheck + build verts. NB dette mineure : fichiers `session_compare_*.go` gardent leur
  nom bien qu'ils contiennent désormais l'infra session-summary partagée (rename = follow-up).
- [x] G4 — CR A9 : `SquadV2RouteHost.tsx` + `SquadV2Page.tsx` (ATTENTION : `squad/v2/types.ts`
  reste vivant). FAIT : 2 pages orphelines (0 importeur, 0 route) supprimées ; `squad/v2/types.ts`
  (consommé par SessionBriefing/SquadVerdict) + les autres composants squad/v2 (HistoryTable,
  MedalsGallery, WeaponsTable…) conservés. Gate : front typecheck vert.
- [x] G5 — CR A9 + ARCHI mineur : chaîne `NotifyNewMedia`. **PÉRIMÈTRE ÉLARGI vs plan**
  (signalé) : la fonctionnalité était câblée END-TO-END jusqu'à un toggle réglages
  user-facing (`discord_notify_new_media`) MAIS sans AUCUN déclencheur (0 caller de
  `NotifyNewMedia`) — une demi-feature morte (toggle qui ne déclenche rien). Laisser le
  backend supprimé + le toggle vivant aurait violé la règle 11 (pas de toggle no-op). Donc
  suppression COMPLÈTE full-stack :
  - Go : `NotifyNewMedia` + helpers uniques (`queryUnnotifiedMedia`, `markMediaNotified`,
    `buildMediaEmbed`, `mediaRow`, `mediaKindVideo`, `min`), imports `database/sql`+driver
    duckdb devenus morts, `NotifyConfig.NotifyNewMedia` + ligne loader, clés i18n Go
    `discord_media_*`, champ `domain.Settings.DiscordNotifyNewMedia` (+ patch struct) +
    store (3 sites) + migration `add_media_discord_notified` (steps_player_base + order.go)
    + colonne CREATE TABLE `discord_notified_at` (shared_social) + addIfMissing/backfill.
    Tests supprimés : `media_notify_test.go`, `notifiers_extra_test.go` (100% média) +
    bloc `TestMin`/`TestBuildMediaEmbed` de `embeds_test.go`.
  - Front : toggle `_settingsCards.tsx`, i18n settings (type+FR+EN), `types.ts`,
    `openapi.yaml` + `generated.ts` régénéré, fixtures (handlers/AddFriendFlow/WatcherCard).
  Gate : go build+vet + notify/settings/migration tests verts ; front typecheck+build+vitest
  (42) verts ; intégration `-p 1` (voir journal).
- [x] G6 — ARCHI mineur : `ReassociateMedia` (`media_service.go:338`) — méthode + interface
  + types (route supprimée 2026-04-29). FAIT : méthode `MediaService.ReassociateMedia`, entrée
  interface `port`, types `domain.ReassociateRequest/Result`, test + mock supprimés (prod-orphelin :
  handler HTTP retiré 2026-04-29, gardé vivant seulement par les tests). Gate : build+vet OK,
  service/handlers/title tests verts.
- [x] G7 — ARCHI mineur : `ServiceConfigIDFor` (`domain/title/registry.go:249`, fonction morte).
  FAIT : stub (return "") 0 caller supprimé. Gate : build+vet OK.
- [x] G8 — ARCHI mineurs : constantes SQL mortes + doc fausse. FAIT : supprimé
  `Q4MatchesForFilters` + `Q4MVMatchesForFilters` (variantes monolithiques mortes ;
  les variantes `Q4Shared*`/`Q4MVShared*` restent LIVE, cf. filters_repo.go) et
  `Q26eHomeSkillPeakByType` (monolithique remplacée par le split Phase A/B
  Q26ePeakPhaseAPlayer + classifyPeakType). Corrigé la doc inversée référençant Q26e
  (home_repo_skill_peak.go, csr_backfill.go, player_repos_test.go, queries.go catalog).
  Q24 (`Q24LUSRHistory`) : PAS morte (2 lecteurs) — c'était la doc fausse qui annonçait
  un param `?1=xuid` inexistant (query sans WHERE, player DB mono-joueur) → doc corrigée.
  Gate : build+vet OK, `go test ./internal/platform/duckdb/...` vert.
- [x] G9 — ARCHI mineur : entrées mortes `assets.toml`. FAIT : supprimé les 8 entrées
  `[assets.mode.*]` placeholder H5 (slugs lowercase slayer/ctf/… divergents de la
  convention — clé = catégorie de mode réelle comme halo_infinite ; AUCUN lecteur
  mode-kind en prod, seuls des tests lisent "mode"/"ranked" côté Infinite). Header
  documenté (réintroduction correcte = Phase 1). Gate : `go test ./internal/games/...` vert.
- [~] G10 — CR A9 : `upsertLUSRRatingsLegacy` — vérifier déjà supprimé en A3, sinon supprimer.
  VÉRIFIÉ : déjà supprimé (grep = 0 occurrence). Rien à faire.
- [x] G11 — CR A9 / DEC-7 : `SessionKDATimeline.tsx`, `SessionOcdrScatter.tsx` supprimés
  (orphelins confirmés : 0 importeur). Noté dans le backlog UI (mémoire
  project-backlog-deferred-tasks item 3 : récupérables via git si regret). Gate : front
  typecheck vert.
- [x] G12 — CR A9 : `MapMuToLegacyRating`/`MapTierSubToLegacyRating` dé-exportés →
  `mapMuToLegacyRating`/`mapTierSubToLegacyRating` (test-only confirmé : 0 caller prod).
  Noms de tests `Test*` conservés (convention Go). Gate : build+vet + skill_v2 test verts.
- [~] G13 — CR A9 : `processMatch` legacy — vérifier déjà supprimé en D1b, sinon supprimer.
  VÉRIFIÉ : la fonction `processMatch` est déjà supprimée (D1b, engine_process_match.go) ; il ne
  reste que des mentions en COMMENTAIRE (backfill_personal_scores.go, csr_writes.go, engine.go…) —
  commentaires stale, non bloquants (nettoyage cosmétique noté, hors périmètre suppression).
- [x] G14 — DETTE §2.5 / DEC-6 : `known_teammates_count` + `friends_xuids` retirées. FAIT :
  confirmé 0 writer (persister ne les écrit pas) et 0 lecture applicative (`ExcludeFriendsXUIDs`
  = param requête distinct, pas la colonne). Retiré de la vue `player_match_enrichment_latest`
  (buildPMELatestViewSQL, baseline-only), de `ensurePMEColumns`, du CREATE TABLE `sync/schema.go`
  et de la doc `persist/doc.go`. DROP physique = au prochain rebuild append-only (DEC-6, pas de
  writer à réactiver ; la vue ne les projette plus donc plus aucune surface de lecture). Fixtures
  de test conservées AVEC les colonnes (couvre le cas réel « DB legacy = colonnes orphelines
  physiques présentes »). Gate : build+vet + sync/migration/persist tests verts + intégration -p 1.
- [x] G15 — ARCHI mineur perf : `mv_map_stats` rebuild supprimé. FAIT : confirmé 0 lecteur Go
  (seule réf = la définition). Retiré de `playerMaterializedViews` (plus de DROP+CREATE TABLE à
  chaque sync). Ajout d'un nettoyage self-healing `deprecatedPlayerAggregates` (DROP TABLE IF
  EXISTS mv_map_stats, no-op après 1er passage) pour purger les DBs prod existantes — commenté
  façon kill-switch (critère de retrait mesurable). Gate : idem G14.
- [~] G16 — PRÉ-EXÉCUTÉ le 2026-07-02. VÉRIFIÉ : la section « ## 0. Complétude » est bien
  présente dans `.claude/skills/delivery-checklist/SKILL.md` (couvre suppression routing =>
  suppression code+tests+imports+types+openapi/i18n). Rien à refaire.

Gate G : `go build ./... && go test ./...` ; `npm run typecheck && npm run build` ;
grep de chaque symbole supprimé → 0 ; diff openapi cohérent (suppression session-compare).

### LOT F — Title-agnosticism (fuites HINF, manifests H5)

Objectif : plus aucune donnée/label/URL Infinite servie sous titre H5 ; manifests H5
complets ; ratchet anti-slug étanche. (Priorité audit : ces items bloquent Halo 5, les
traiter AVANT les refactors structurels.)

- [x] F1 — ARCHI 28 : couplage platform/duckdb → games/halo_infinite (classification
  média des modes) SUPPRIMÉ. PÉRIMÈTRE ÉLARGI vs synthèse : q37_enrich couplait 3 symboles
  (pas 1) — `PairNamePrefixesForCategory` + `ModeCategoryOther` + `AllKnownPairNamePrefixes`,
  via une chaîne de fonctions LIBRES (`applyCrossDBMediaFilters`→`mediaRowMatchesMode`). Créé
  un seam unique `analysis.ModeTaxonomy` (struct + helpers nil-safe Classify/Prefixes/
  KnownPrefixes + champ Other), injecté dans MediaRepo (`WithModeTaxonomy`) et threadé aux
  fonctions libres via `runMediaPipeline` (méthode `r`). Les 3 fichiers media_repo* n'importent
  PLUS halo_infinite (import retiré ×3). Câblage au wiring `registry_media.go` (helper
  `mediaModeTaxonomy()`, racine DI autorisée à importer games) — byte-identique (Infinite =
  seul titre média en prod). NOTE per-titre : quand un 2e titre exposera du média, résoudre
  la taxonomie via le title resolver (comme assetURLFor). Test seam nil-safe + délégation.
  Gate : build+vet + duckdb/api/service/analysis tests verts. (F15-2 neighbors_skill = même
  seam `ModeCategoryPrefixes`, item distinct.)
- [x] F2 — ARCHI 29 : providers CSR Explorer/Compare gatés par capability. FAIT : les 2
  providers (`newExplorerCSRProvider`/`newExplorerSeasonCSRProvider`) sont spécifiques au
  live Infinite (`rankedplaylists.Active()` + client + endpoints HINF). Nouveau helper
  `titleSupportsLiveCSR(pdb)` = `dataAdapter.Capabilities().Has(CapMatchSkillSnapshot)` (pas
  de comparaison de slug → pas de violation ratchet). Infinite `match.skill.snapshot=degraded`
  → Has=true → providers injectés (0 changement). H5 `=not_exposed` → Has=false → providers
  nil → le service dégrade (encart CSR vide, `s.csr==nil` guard) : plus de fuite CSR/playlists
  Infinite sous H5. Appliqué aux 2 sites (Explorer deps + Compare WithCSR). NOTE Phase 2 : quand
  H5 exposera match.skill.snapshot, ces providers Infinite devront devenir title-aware (playlists
  H5) — la capability les réactiverait à tort en l'état (à traiter au câblage CSR H5). Gate :
  build+vet + api/service tests verts.
- [x] F3 — ARCHI 32 : URL Waypoint derrière `TitleAssetURLAdapter`. FAIT : 2 méthodes ajoutées
  à l'interface — `MatchWebURL(matchID)` (forme `.../matches/{id}`, site MatchView) et
  `PlayerMatchWebURL(gt, matchID)` (forme `.../players/{gt}/matches/{id}`, site MatchHistory).
  Implémentées : halo_infinite (Waypoint, byte-identique) ; halo_5 + synthetic_title_b → "".
  MatchView `builders_header.go` : `assetURL.MatchWebURL(matchID)` (nil-guard) — H5 (assetURL nil)
  → plus de lien Waypoint Infinite MORT en Match View H5. MatchHistory : `buildMatchURL` (URL
  hardcodée) SUPPRIMÉ ; nouveau `WithAssetURL` + closure `matchURLFn()` threadée dans enrichRows/
  enrichRow (remplace le param `waypoint string`) ; câblé au wiring (registry_pages:682). Tests
  `buildMatchURL` migrés vers l'adapter halo_infinite (Match/PlayerMatchWebURL). 3 stubs de test
  complétés (resolver + 2 service). Gate : build+vet + `go test ./...` (non-intégration) verts.
- [x/~] F4 — ARCHI 31 : labels d'outcome FR en dur x3. FAIT (site 1 + site 3) :
  - Site 1 `match_history` : `MatchHistoryService.WithSemantic` + résolution via
    `semantic.Outcomes().Get(code→key).Label("fr")` avec fallback FR canonique
    (`outcomeLabel`). Le seam `matchURLFor` (F3) + `outcomeLabel` (F4) consolidés dans une
    struct nil-safe `rowFormatters` (réduit l'explosion de params + churn de tests). Câblé au
    wiring (registry_pages `WithSemantic`). Map `outcomeCodeToKey` (1→tie/2→win/3→loss/4→dnf).
  - Incohérence corrigée : `outcomes.toml` Infinite avait dnf fr="Non terminé" alors que H5 +
    les 2 surfaces UI (match_history/home) utilisaient "Abandon" → aligné Infinite→"Abandon"
    (byte-identique UI, source de vérité unifiée cross-titre).
  - Site 3 (notify/discord) : `[~]` déjà via le seam `Outcomes()` (labels.go, testé). Désormais
    "Abandon" pour dnf (unifié).
  - Site 2 (`analysis/home_locale.go`) : `[~]` → K1. Reste en littéraux FR (analysis = pur, la
    migration resolver relève de la couche service K1) ; valeur déjà cohérente ("Abandon").
  Gate : build+vet + service/api/games/mappings tests verts.
- [x/~] F5 — ARCHI 33 : labels KPI en dur. FAIT (bugs concrets) : `timeseries_service_tabs.go`
  portait des ANGLICISMES dans l'UI FR (« Win Rate » → « Taux de victoire », « Kills/game » →
  « Frags / partie », viole `feedback_fr_ui_no_anglicisms`) + une CORRUPTION UTF-8 réelle
  (« PrÃ©cision » double-encodé → « Précision »). Corrigés + gate service vert.
  DIFFÉRÉ `[~]` → K1 : le routage complet des ~20 labels `compare_service.go:472` (et
  timeseries) via `FieldMappingSet.Get(FieldKey).Label(locale)` (ou key-only + labelling front)
  est un refactor architectural de couche (K1). Les libellés actuels sont FR-canoniques Halo
  (corrects pour Infinite ET H5, même famille) → faible valeur title-agnosticism, gros diff.
  `session_compare_service.go` : `[~]` réf G3 (helper partagé conservé ; migration = K1).
- [x/~] F6 — ARCHI 34 : `config/titles/halo_5/mappings/fields.toml` COMPLÉTÉ (5 → 52 FieldKeys).
  FAIT : généré par transform d'Infinite (59) MOINS les 7 champs PvE/Firefight (`group="pve"`) —
  labels/unités canoniques Halo partagés. 52 = combat+match+career+skill+derived. Gate :
  `go test -count=1 ./internal/games/mappings/ ./internal/games/halo_5/` VERT (validate + loader).
  Le test de parité générique (« chaque titre déclare les FieldKeys de ses capability-groups ») →
  **L2** avec les autres garde-rails (F15-12/14) : cross-source capabilities↔fields, home gouvernance.
  **DÉCISION (2026-07-04) : sous-ensemble par capability, PAS strict-59.** H5 déclare les
  FieldKeys des groupes qu'il EXPOSE (combat, match, career, skill, aggregate) mais PAS les
  7 champs PvE/Firefight (`waves_completed`, `bosses_killed`, `grunt/elite/jackal/brute/
  hunter_kills`) — H5 = modèle Warzone Firefight distinct, `pve.firefight_stats=not_exposed`.
  Test de parité générique : « chaque titre actif déclare les FieldKeys des capability-groups
  qu'il expose » (un FieldKey manquant dégrade gracieusement via `Get()→ok=false`, donc pas
  de casse ; la parité garde la cohérence capabilities↔fields).
- [~] F7 — ARCHI 37 : DÉCISION (2026-07-04). Réconciliation seule dans ce plan ; l'ACTIVATION
  de l'engagement H5 est **HORS PÉRIMÈTRE** (chantier dédié — cf. mémoire
  [[project-h5-engagement-canonicalization-chantier]]). Constat vérifié : le calcul
  (`temporal.ComputeEngagementScore`) est PUR + title-agnostic et le sync H5 reconstruit déjà
  l'enrichment « …/engagement/… » → la machine tourne pour H5 ; c'est l'adapter H5 qui coupe
  (`CapEngagement=not_exposed`) par prudence (score non canonicalisé de 1er ordre + coefficients
  cold-start calés Infinite → risque de mauvaise calibration H5 non validée). Ce plan livre
  UNIQUEMENT le test miroir coarse↔fine en règle SOUPLE : pour tout titre, une capability coarse
  ayant un pendant fine ⟹ le fine est DÉCLARÉ (n'importe quel statut). H5 (coarse `engagement`
  ON + fine `engagement.score`=not_exposed) est alors COHÉRENT — pas de changement de comportement.
  Test à livrer avec F15-12 (complétude capabilities). L'activation `not_exposed→degraded` +
  câblage adapter + validation calibration = chantier futur (impacte Halo 7 et tout titre).
- [~] F8 — ARCHI 36 : DIFFÉRÉ Phase 1b / MT-02 (justifié, règle 9 — territoire auth ADR 0023
  sensible). Vérifié sur pièces : `config/titles/halo_5/auth.toml` DOCUMENTE explicitement que
  H5 RÉUTILISE les audiences Infinite (« ZÉRO audience séparée requise, les bonnes ») ; seule
  divergence = `clearance_url=""` — or H5 IGNORE le clearance (`ClearanceAware:false`, confirmé
  sonde live cmd/probe-h5). Donc `DefaultHaloAuthDescriptor()` (Infinite) est FONCTIONNELLEMENT
  correct pour H5 aujourd'hui (il fait juste un appel clearance supplémentaire que H5 ignore) —
  H5 sync fonctionne. L'infra existe (`LoadAuthDescriptor(root,slug)` + variantes `*WithDescriptor`)
  mais threader le descripteur par-titre à travers la chaîne d'échange de tokens (par-joueur) est
  un refactor auth-sensible pour titres FUTURS à auth réellement divergente = MT-02 / activation
  multi-titre. Risque de casser l'auth (règle : jamais toucher l'auth à la légère) pour valeur
  H5 NULLE. Le seam est prêt ; le câblage se fait au chantier d'activation. Défaut byte-identique
  Infinite (0 régression).
- [~] F9 — ARCHI 35 : DIFFÉRÉ Phase 1b / MT-19 (justifié, règle 9). Vérifié sur pièces :
  les 6 handlers Ascension/Prestige (progression/coach/profile/awards/patterns/campaign,
  `server.go:~1554-1614`) sont construits AU BOOT avec `DefaultSlug` figé et montés
  (`.Mount(r)`). Le fix « a minima RequireCapability → 503 » N'EST PAS applicable proprement :
  Ascension/Prestige (progression V2) est Infinite-only mais il N'EXISTE PAS de capability
  Ascension/Prestige, et H5 déclare `CapCareer` → un gate `CapCareer` ne bloquerait PAS H5.
  Les 2 vrais fixes — (a) créer `CapAscension` + gater, ou (b) rendre chaque handler
  ctx-title-aware (`ctxkeys.TitleSlug` + resolver → 503 pour titre sans Ascension) — sont du
  câblage boot multi-handlers = chantier d'activation multi-titre (le plan lui-même le renvoie
  à MT-19 / Phase 1b). Risque élevé pour valeur audit faible (le front ne lie pas Ascension
  hors Infinite → pas de fuite active constatée). Non traité ici ; à reprendre en Phase 1b.
- [x] F10 — ARCHI 30 : regex ratchet élargie (`(?:\w+\.)?DefaultSlug` + forme d'appel
  `TitleSlug(ctx)`), FERME le trou d'un feature-gate `TitleSlug(ctx) == "halo_infinite"`.
  PÉRIMÈTRE : l'audit citait 3 sites (comeback:34/95, coordinator:316) mais l'élargissement
  en détecte **11** (+ `api/server.go:461`, `ops/seed_demo_multititle.go` ×6, `ops/seed_demo.go:391`)
  — tous vérifiés = gardes de PARITÉ de base (défaut HINF byte-identique) et NON des feature-gates.
  Grandfathered dans l'allowlist (par fichier, justif datée catégorisée ; comeback:34 suivi F15-15).
  Test de sanité positif ajouté (regex attrape les formes élargies, pas l'égalité slug↔slug).
  Gate : `go test ./internal/archlint/... -run Slug` VERT (ratchet + sanité).
- [x] F11 — ARCHI 38 : DÉCISION TRANCHÉE = WARN (défaut recommandé, moins risqué qu'un
  parity-test qui figerait le built-in au TOML). FAIT : `config_loader.go` émet
  `title_builtin_toml_ignored` (WARN) quand un `title.toml` existe pour un titre déjà
  enregistré (built-in) — plus de skip muet. Test `TestLoadTitlesIntoRegistry_BuiltinTOMLWarns`
  (capture slog). Gate : `go test ./internal/domain/title/... -run LoadTitles` VERT.
- [~] F12 — ARCHI 13 / DETTE §2.4.1 : DIFFÉRÉ → **LOT K** (structure & couches). Vérifié :
  18 fichiers (9 src + 9 test) à extraire de `package analysis` vers un nouveau `package film`
  (`internal/games/halo_infinite/film/`). L'audit l'annonçait « mécanique » mais c'est une
  EXTRACTION DE PACKAGE d'un sous-système délicat (pipeline film RE-verse, keyframe/weapon
  attribution) : les fichiers vivent dans `package analysis`, leurs symboles exportés sont
  consommés par ~3 callers externes via `analysis.X` (sync/backfill_weapons, sync/collect,
  games/halo_infinite/events) + dépendances film→analysis (`RawEvent`). Renommage package +
  requalification des refs + MAJ imports callers = refactor structurel = domaine LOT K, pas
  audit-mineur. Risque de casser le pipeline film (délicat). À faire en chantier dédié K.
- [~] F13 — DETTE §2.4.2 : DIFFÉRÉ → **LOT M** (Tests). Paramétrer les goldens par slug
  (`golden_output_<slug>.json`, fixtures H5) pour distinguer régression vs divergence de titre =
  amélioration d'INFRASTRUCTURE DE TEST (nouvelles fixtures H5 + chemins slug-aware), home naturel
  = LOT M (gaps de tests ciblés). Pas un défaut fonctionnel ; les goldens Infinite actuels restent
  valides. À livrer avec les autres travaux tests de M.
- [x] F14 — DETTE §2.4.3 : convention « nouveau titre » figée. FAIT : section « Data writes:
  the Collect → Persist architecture (ADR 0019) » ajoutée à `docs/ADD_TITLE.md` (Étape 4) EN
  **et** FR — invariant INSERT-only, hiérarchie client→livesync→persist, tables append-only +
  vues `_latest`, impl de référence `halo_5/livesync/csr_match.go`. Doc-only (ADD_TITLE.md
  hors hook docs-fr-sync mais bilingue → MAJ des 2 versions).
- [x] F15 — ARCHI mineurs title-agnosticism : toutes les puces statuées (voir Sous-progrès
  ci-dessous). Bilan : 8 `[x]` (couplages retirés / paramétrés) + 9 `[~]` (déjà title-agnostic
  vérifié, ou seams différés à leur lot naturel). Les 2 garde-rails génériques (F15-12 miroir
  coarse↔fine = livrable F7 souple ; F15-14 cap⟺scalaire) → **L2** (gouvernance/ratchets), leur
  home naturel (cross-source registre+capabilities+damage_model, cf. renvoi F7). Détail :
  `home_repo_skill_peak.go:516` (fallback badge CSR cross-titre),
  `match_view_repo_neighbors_skill.go:63` (préfixes HINF), `halo_ranks_loader.go:132`
  (DefaultSlug forcé), `server.go:780` (callbacks Asset Drawer), `world_stats_enricher.go:15`
  (leaderboard mondial sans gate), `handlers/sync_handler.go:29` (couplage livesync H5),
  `registry_catalog_adapter_check.go:48` (HasCatalogAdapter), `registry_career.go:220`
  (commentaire obsolète), `halo_5/outcomes.toml:9` (raw_code absent → chemin nominal MT-06),
  `halo_5/capabilities.toml:10` (header périmé), `synthetic_title_b` (fixture dans l'arbre
  runtime + test de parité), `games/capabilities.go:13` (exigence de complétude des clés
  fines), `server_titles_additional.go:110` (fallback silencieux → erreur ou WARN fort),
  `domain/title/registry.go:76` ([damage_model] miroir manuel — garde de cohérence, croisé
  L2), `sync/comeback.go:33-38` (ID médaille par slug → mapping TOML — CR mineur),
  `domain/achievement_categories.go:27`, `domain/job.go:92` (littéral halo_infinite).
  (Les items « chemins physiques » de cette section partent en K1l — statuer `[~]`.)
  **Sous-progrès F15** (case F15 fermée quand toutes les puces sont statuées) :
  `home_repo_skill_peak.go` (F15-1) `[~]` seam déjà câblé (server.go:812) ·
  `registry_career.go:220` (F15-8) `[x]` commentaire corrigé (title-agnostic MT-09) ·
  `capabilities.toml:10` (F15-10) `[x]` header MAJ (8 caps supported, pas « seul career ») ·
  `job.go:92` (F15-17) `[x]` littéral → `title.DefaultSlug` (pas de cycle) ·
  `sync/comeback.go:33-38` (F15-15) allowlisté ratchet F10 (mapping TOML = suivi) ·
  `match_view_repo_neighbors_skill.go:63` (F15-2) `[x]` couplage halo_infinite retiré, injecte
  le seam `analysis.ModeTaxonomy` (réutilise F1) via `WithModeTaxonomy` au wiring + fix test
  intégration ·
  `halo_ranks_loader.go` (F15-3) `[x]` `LoadRankCatalog` paramétrisé `(ctx, metaDB, titleSlug)`
  au lieu de `DefaultSlug` figé ; callers (boot + 3 tests) MAJ ; import titlePkg retiré du loader ·
  `halo_5/outcomes.toml` (F15-9) `[x]` `raw_code` ajouté (win=2/loss=3/tie=1/dnf=4) — seam
  int↔canonique MT-06 complet pour H5 ·
  `domain/achievement_categories.go` (F15-16) `[~]` DÉJÀ title-agnostic : `achievementCategoriesByTitle`
  est un registre map[slug] avec dégradation gracieuse (titre absent → catégorie vide, front masque
  le filtre) — pas de littéral en dur à corriger ; décision « statu quo » respectée ·
  `handlers/sync_handler.go` (F15-6) `[~]` `livesync.RunnerForTitle(titleSlug)` est DÉJÀ le point
  de dispatch title-aware (retourne nil hors titre live → fallback engine par défaut) ; seul
  l'emplacement de l'import couple (mineur) — déplacer `RunnerForTitle` en package agnostique = K-couches.
  `server.go loadTitleAssetDrawerData` (F15-4) `[~]` DÉJÀ title-paramétré `(metaPath, slug)`, appelé
  pour H5 (h5Maps/Weapons/Medals) → pas de hardcoding Infinite ·
  `registry_catalog_adapter_check.go` (F15-7) `[~]` `HasCatalogAdapter(slug)` DÉJÀ title-aware +
  dégradation gracieuse (H5 → false, « pas une erreur ») ·
  `server_titles_additional.go:110` (F15-13) `[~]` DÉJÀ un `slog.Warn("capabilities_convert_failed")`
  (pas un fallback SILENCIEUX) ; décision retenue = WARN (télémétrie expvar = nice-to-have L1/L2) ·
  `world_stats_enricher.go` (F15-5) `[~]` → suivi activation leaderboard mondial multi-titre :
  `RankedPlaylistSet()` couple `rankedplaylists.Active()` (Infinite) mais le classement mondial est
  CSR/Infinite-only en prod (H5 non câblé) ; le seam `RankedPlaylistProvider(slug)` (pattern F2) se
  posera au câblage leaderboard H5 ·
  `synthetic_title_b` (F15-11) `[~]` fixture de test multi-titre (config/titles + package) ; les tests
  d'isolation en DÉPENDENT ; à couvrir/retirer avec le test miroir capabilities (L2) ·
  F15-12 (miroir coarse↔fine, livrable F7 souple) + F15-14 (garde-rail cap⟺scalaire damage_model/
  team_mmr) → **L2** (gouvernance/ratchets) : garde-rails génériques cross-source ; invariants déjà
  documentés dans le code (`registry.go` « SSI ProvidesDamageTaken(slug) »).
- [~] F16 — ARCHI 7 : DÉPLACÉ → **LOT H** (repropagation/dédup — home naturel). Dédup
  `augmentWithActiveRankedCSRs` : original `sync/career.go` vs copie DI dans
  `registry_pages.go` (Explorer/newExplorerCSRProvider, déjà touchée en F2), divergence
  NameFR/NameEN réelle. C'est un item de DÉDUPLICATION (« plus de copie locale divergente
  d'un helper ») = exactement le thème de LOT H → traité là avec H1-H7 (une impl unique,
  nom résolu via semantic adapter + locale, + garde-rail anti-régression).

Gate F : archlint étendu vert (F10) ; test parité fields.toml (F6) et coarse/fine (F7)
verts ; grep `halowaypoint` hors `games/halo_infinite` + `platform/halo` → 0 (le client
sync bouge en K3e — allowlist temporaire datée si besoin) ; smoke test des pages H5
(Médias, Explorer, Match View) via `verify`/run local.

> **STRATÉGIE H→N (décidée 2026-07-04, investigation workflow 8 agents sur pièces)** :
> Lots SÛRS d'abord (H, I, J sauf J5, M, L, N) exécutés en autonomie sur les défauts recommandés
> de l'investigation (helpers dans `sql_fragments.go`/`util/pointers`, i18n par-feature, etc. —
> low-stakes, notés au journal). **LOT K (26 items, ~4-6 j, refactors god-functions/packages
> haut-risque : NewRouter 1470L, SyncEngine.run 483L, extractions packages 143/127/112 fichiers)
> + J5 (cache Match-History) + F12 (film) = CHANTIER DÉDIÉ** (mini-commits séquentiels + smoke-run,
> en DERNIER ou planifié à part). Vraies décisions produit escaladées au moment voulu : J2 (budgets
> mémoire DuckDB, après mesure J1), J5 (sémantique cache), N4 (politique purge migrations).
> Ordre : H → I → M → L → J(sauf J5) → **K dédié** → J5 → N. Comptes réels > audit (H1=115 littéraux/
> 52 fichiers vs 87/33 ; H2=58/30 vs 36/19). Résultats complets : workflow `lotHN-investigate`.

### LOT H — Repropagation & duplication (source de vérité unique)

Objectif : plus de copie locale divergente d'un helper canonique existant ; chaque helper
embarque son garde-rail anti-régression le jour de sa livraison (CR reco 1).

> **CALIBRATION H (2026-07-04, vérifiée sur pièces)** : les comptes de l'audit sont PÉRIMÉS
> (sous-évalués) et plusieurs « dédups » cachent des DIVERGENCES SÉMANTIQUES. Règles
> d'exécution H : (a) vérif PER-COPIE avant migration (pas de grep-replace aveugle) ;
> (b) les `migrations/steps_*.go` sont de l'HISTORIQUE GELÉ — ne jamais y réécrire les
> littéraux, les ALLOWLISTER dans les garde-rails ; (c) chaque helper livré = garde-rail
> grep dans le MÊME commit.

- [x] H1 — CR A10 : helper start_time canonique. **LIVRÉ (2026-07-04)**. Source unique
  `analysis.SQLStartTimeCanonical(alias string) string` (sql_fragments.go) ;
  `duckdb.StartTimeCanonicalSQL` délègue (appel local sans préfixe pour les repos duckdb).
  Comptes RÉELS : le « 115 » de l'audit conflatait le pattern canonique avec
  `real_start_time` (colonne DISTINCTE, epoch/durée), des commentaires-prose et la
  définition — le vrai pattern `COALESCE(x.start_time_utc, x.start_time AT TIME ZONE 'UTC')`
  = **97 sites migrés** (92 backtick scriptés + 5 double-quote/analysis manuels), sur
  ~46 fichiers (internal + cmd). Effets de bord traités : **21 `const`→`var`** (une valeur
  bâtie par appel de fonction n'est plus constante) + 2 `const q` locaux → `:=` + 2
  commentaires démanglés. Garde-rail `archlint/no_raw_start_time_literal_test.go` (scanne
  internal/ + cmd/, saute migrations/ + la définition, allowlist VIDE, regex précis qui
  n'attrape ni `real_start_time` ni l'offset-diagnostic backfill_first_joined_tz). Gate :
  build+vet OK, unit duckdb/sync/ops verts, **intégration `-p 1` VERTE** (duckdb 111 s,
  sync 109 s, 0 FAIL, exit 0), garde-rail vert, grep hors allowlist → 0.
- [x] H2 — CR A11 : prédicat bot. **LIVRÉ (2026-07-04)**. Les ex-const nues
  `SQLIsBot`/`SQLIsNotBot` avaient **0 consommateur SQL** (centralisation abandonnée,
  34 copies re-divergées — leçon CLAUDE.md règle 6) → remplacées par
  `SQLIsBotCol(col)`/`SQLIsNotBotCol(col)` (paramétrées, préfixe d'alias inclus).
  **33 sites single-% migrés** (34 − 1 : le site `gamertag NOT LIKE 'bid(%'` de
  diag_recent_match_sync RÉVERTÉ — colonne distincte, wrapper aurait blanchi un bug latent
  puisque les bots ont un gamertag "343 …", pas "bid…" ; noté en Découvertes). PIÈGE `%%`
  confirmé : 10 sites sont des templates `fmt.Sprintf` (`'bid(%%'`) répartis sur 6 fichiers
  → prédicat statique correct, migration = threading d'un `%s`-arg positionnel à travers
  plusieurs call sites (fragile, SQL identique) → **allowlistés** dans le garde-rail
  (politique identique à no_raw_outcome_literal_test.go). Garde-rail
  `archlint/no_raw_isbot_literal_test.go` (regex ciblant les colonnes xuid — ignore
  `gamertag` et la forme paramétrée `%s LIKE` d'identity.go ; allowlist décroissante =
  6 fichiers Sprintf + media.go comment). Effets de bord : 8 `const`→`var` ; test de
  régression B2 (grep `bid(`) mis à jour pour accepter le helper. Gate : build+vet OK,
  unit verts, **intégration `-p 1` VERTE** (duckdb 109 s, sync 106 s, 0 FAIL, exit 0).
- [x] H3 — CR A12 : SynthesisPage / useLocalFilterBar. **Étape 1 LIVRÉE (2026-07-04)**.
  Nouveau module partagé `features/_shared/experienceCascade.ts` (EXPERIENCE_TO_CASCADE +
  setsEqual) importé par le hook ET SynthesisPage → dédup des 13 L à l'identique
  (1 def chacune, vérifié grep). Étape 2 (migration d'état complète : faire consommer le
  hook par SynthesisPage) = `[!]` NON traité — gros refactor conditionnel (le hook devrait
  couvrir l'état pending/committed de SynthesisPage, mais SynthesisPage a des besoins
  synthesis-spécifiques à valider) → §7 Découvertes. Le fix du matching couplé aux libellés
  FR (substring `'classé'` dans experienceCounts) = `[!]` → §7 (design, pas dédup).
  Gate : typecheck OK, eslint OK (0 err), vitest 46 verts (dont useLocalFilterBar + Synthesis).
- [x] H4 — CR A13 : formatters front. **LIVRÉ (2026-07-04)**. Comme H5-safeDiv, l'audit a
  SUR-compté : les « copies » sont des HOMONYMES DIVERGENTS (locale-aware, fallbacks/formats/
  contrats d'entrée distincts), pas des doublons. **1 seul vrai doublon** :
  `MatchEncountersTable.formatPercent` === `ExplorerEncounterBriefing.formatPercent`
  (`Math.round(v*100)%`) → centralisé `lib/formatters.formatPercentInt(ratio)` (sans espace,
  entier ; distinct de formatPercent qui a l'espace typo FR + décimales). Homonymes divergents
  RENOMMÉS (nom descriptif, satisfait le gate sans fusion destructrice) : PeriodSessionRail
  `formatDateShort`→`formatDateMonthDay` ; lab `formatDate`→`formatLabDateTime` (date+heure) ;
  ascension `formatDate`→`formatAscensionDate` (PIÈGE évité : cru mort au grep, typecheck a
  prouvé qu'il est appelé par 3 composants → restauré+renommé, PAS supprimé).
  `session-detail._shared.formatPercent` = `[~]` GARDÉ : legacy DOCUMENTÉE (entrée 0..100,
  TODO ADR 0006 de bascule vers lib quand l'API passera en 0..1, testée). Gate : typecheck OK,
  eslint 0 err, vitest 334 verts, grep `function formatDate` hors lib → 0 (formatPercent : ne
  reste que la legacy session-detail documentée).
- [x] H5 — CR A13 : helpers Go. **LIVRÉ (2026-07-04)**. (a) `safeDiv` `[~] faux positif`
  CONFIRMÉ non touché (≠ SafeRatio : remplacer = bug KD). (b) Créé `internal/util/pointers`
  `Ptr[T any](v T) *T`. Migrés vers `pointers.Ptr` : les **3 copies PURES** `strPtr`
  (cmd/server, api/handlers/openspartan_import, games/halo_5/livesync/appearance_persist)
  + `strPtrH5` (games/halo_5, clone identique). PIÈGE évité (sur pièces) : le `strPtr` de
  `sync/transforms_helpers.go` N'EST PAS pur — il renvoie `nil` sur chaîne vide → migrer =
  bug (`&""` au lieu de nil). RENOMMÉ `strPtrNonEmpty` (13 call sites prod + tests) pour
  refléter sa sémantique et laisser le garde-rail propre. `strPtrOrNil` (openspartan,
  TrimSpace+nil) + `strPtrEq`/`strPtrDeref` (test) restent distincts. Garde-rail
  `archlint/no_local_ptr_helper_test.go` (regex `func strPtr(H5)?\(` — ignore les variantes
  nommées). Gate : build+vet OK, unit verts, intégration `-p 1` VERTE (0 FAIL).
- [x] H6 — CR A13 : **LIVRÉ (2026-07-04)**. Volet icône = `[~]` faux positif confirmé
  (1 seule def `OpenMatchIcon` MediaViewer.tsx). Volet ECharts : `_utils.ts` factorisait
  DÉJÀ axis/tooltip/legend/xAxis (getAxisBase/getTooltipBase/getLegendBase) ; seul le
  littéral `grid: { top, right, bottom, left, containLabel: true }` restait recopié **8×**
  (4 identiques + 3 variantes de marge). Ajouté `getGridBase(overrides)` à
  `components/charts/_utils.ts` ; 8 littéraux → `getGridBase()` / `getGridBase({...})` avec
  overrides EXACTS (objets identiques → rendu visuel inchangé). Gate : typecheck OK,
  eslint 0 err, vitest 131 verts (timeseries + charts), 0 littéral grid brut restant.
- [x] H7 — CR A13 : **LIVRÉ (2026-07-04)**. Nouveau module `lib/colors/outcomePalette.ts`
  (défaut approuvé D-A.3) — TOKENS sémantiques uniquement via `tokenCssVar` (règle 12).
  Comme H4, l'audit sur-comptait : les helpers avaient des SIGNATURES DIVERGENTES (`ratioColor`
  seul en 3 variantes) → canonicalisés en fonctions distinctes selon la sémantique du seuil :
  `ratioColor(v)` (seuil 1), `winRateColor(v)` (seuil 0.5), `kdaNetColor(v)` (net signé, seuil 0),
  `kdRatioColor(kills,deaths)` (garde deaths=0, délègue à ratioColor), `ratioColorGuarded(deaths,ratio)`
  (variante ratio pré-calculé), `winRateClass(v)` (classes Tailwind sémantiques). 6 fichiers migrés
  (Explorer, MatchEncounters, PalmaresRelationsPage, RelationsTable, RelationsRivalryCards,
  CareerRivalsSection) — noms locaux incohérents (winLossColor/wrColor/kdaColor) → noms canoniques.
  Famille outcome-ENUM (`outcomeColor`/`outcomeColorVar`, mécanisme `resolveToken` distinct) hors
  périmètre H7 (ratio/winrate). Gate : typecheck OK, eslint 0 err, **vitest 2070 verts (suite
  complète)**, grep fns couleur locales hors lib → 0.
- [x] H8 (ex-F16) — ARCHI 7 : **LIVRÉ (2026-07-04)**. La « copie » de registry_pages.go
  n'était pas une fonction nommée mais une **boucle inline** (newExplorerCSRProvider,
  ~21 L) divergeant sur `pl.NameFR` (vs `NameEN` côté sync) + un message de log distinct.
  `augmentWithActiveRankedCSRs` (sync/career.go) EXPORTÉ → `AugmentWithActiveRankedCSRs`
  + **param `locale`** ("fr"→NameFR, "en"/autre→NameEN, ""→skip label ; défaut approuvé
  D-D.11) ; les 2 CSR ont le même type `PlayerPlaylistCSR` donc appel direct. Sync passe
  "en", provider DI Explorer passe "fr" (parité comportement préservée). Boucle inline
  supprimée. Pas de garde-rail grep : le fingerprint (Active()+GetPlaylistCsr) collisionne
  avec `newExplorerSeasonCSRProvider` (autre logique légitime) → la fonction exportée unique
  EST le mécanisme anti-divergence (gate `= 1 def` vérifié). Gate : build+vet OK, unit
  api/sync verts, intégration `-p 1` VERTE (0 FAIL).

Gate H : garde-rails grep H1/H2/H5/H7 verts (allowlists migrations datées) ;
`npm run typecheck && npm run test` ; go build+test ; comptes avant/après au Journal.

### LOT I — i18n

Objectif : purge FR monolingue + anglicismes ; règle lint passée en `error` à la fin.

> **CALIBRATION I (2026-07-04, RÉVISÉE 2026-07-05 sur pièces)** : l'audit s'est
> RÉVÉLÉ FAUX sur le couplage gate↔migration. La règle lint `no-hardcoded-strings`
> ne remonte qu'**1 warning** (pas « >100 ») et ne flague QUE le texte JSX (≥3 mots
> /≥15 car) + 5 attributs (title/aria-*/placeholder/alt) — **PAS** les args de
> fonction (`setError('…')`) ni les libellés courts (« Connexion Xbox » = 2 mots/14 car)
> que visent I1/I2/I4. Donc **I5 n'est PAS couplé à I1-I4** : la migration manuelle des
> strings brutes (bilinguisme réel, règle CLAUDE.md n°1) est volumineuse mais NON exigée
> par le gate lint. **Décision utilisateur 2026-07-05 (option A) : verrouiller le gate
> (I3 + I5 livrés) et sortir I1/I2/I4 en chantier i18n manuel séparé** (§7 + handoff), pour
> enchaîner sur M/L/J/N mieux calibrés. I3 confirme aussi le sur-comptage (13 valeurs FR
> réelles vs « 68+ »). Le « 88 ternaires » d'I4 reste réel mais hors gate.

- [x] I1 — CR A17 : **LIVRÉ (2026-07-05)** (chantier i18n manuel, go utilisateur). Les 5
  composants onboarding/auth passés bilingues FR+EN via manifest `common.toml` +
  `build_i18n_manifests.mjs` : XboxLoginPage (commit 6462887f4), StepDeviceCode +
  StepInitialSync (81fed7aad), RegisterPage + OpenSpartanImportCard (d39cc5d1a).
  `failureMessageFromCode(err, t)` refactoré pour injecter le traducteur (test unité MAJ).
  +43 clés `common.toml` (2402 total). Gate : typecheck 0, eslint 0 (règle no-hardcoded en
  error, I5), vitest 18/18 sur auth+onboarding, 0 résiduel FR user-facing.
- [~] I2 — CR A18 : **LABELS LIVRÉS (2026-07-05) ; figement nombre/date résiduel scopé.**
  Fait `[x]` : (a) MatchScoreboard 11 libellés colonnes + header + tooltip + sbFormatScore
  locale-aware (MatchViewText fr/en) ; (b) heatmaps activité explorer+synthesis (DOW/heures/
  axes/tooltips bilingues) + **centralisation `lib/formatters/calendar.ts`** (4 copies DOW
  dédupliquées, garde-rail `calendar.guard.test.ts`, CLAUDE.md n°6) ; (c) filtres Analyser/
  Appliqué (`common.filter.*`) + « Par carte/mode » (`synthesis.breakdown.*`).
  Fait partiel `[~]` : figement `toLocaleString('fr-FR')` — audit sur-comptait (« ~100 » →
  **39 réels dont ~9 légitimes** : valeurs objet `fr` d'i18n bilingue + `formatDateShort`
  verrou chart documenté). Pont canonique **`lib/formatters/intlLocale.ts`** créé +
  SynthesisPage flagship (15 sites) migré. **RESTE ~24 sites** `[!]` : helpers PURS /
  builders ECharts / consts module SANS locale en scope (career gauges, media/home dates,
  session-detail `fmtInt`+_shared, prestige LeaderboardPP, timeseries SquadAdapted,
  synthesis Weapon*Chart, MatchScoreboard.logic) → threading `locale` par signature requis
  (cosmétique : séparateurs nombre / ordre date). Helper prêt. Détail DETTE_ASSUMEE §I2b.
- [x] I3 — CR A19 : **LIVRÉ (2026-07-04)**. Le « 68+ » sur-comptait ~5× : la quasi-totalité
  sont des CLÉS (`streaksSectionTitle`, `streak_milestone`), des identifiants de code
  (`StreakType`, `StreakCard`, `win_streak`), des valeurs EN (à garder) et le terme de
  glossaire `'Série (Streak)'` (intentionnel — help/i18n.ts documente la traduction).
  **13 VALEURS FR réelles** avec l'anglicisme corrigées → « série » (ascension 5,
  notifications 4, help 1, match-view 1, squad 1, coach-comment 1), dont 2 reformulées
  (double « série » évité ; « streak shield » → « bouclier de série »). CLÉS intactes.
  Gate : typecheck OK, vitest 425 verts, grep valeur FR + « streak » → 0 (ne restent que
  clés/EN/glossaire/commentaires de code).
- [!] I4 — CR mineurs : **SCOPÉ PRÉCISÉMENT (2026-07-05), non exécuté ce tour.** Comptage
  réel = **155 ternaires** `locale === 'en'/'fr' ?` (audit disait 88), MAIS distinction
  clé vérifiée sur pièces : **tous DÉJÀ bilingues** (les 2 langues présentes dans le
  ternaire) → I4 est un refactor **d'ORGANISATION** (centralisation + parité typée), PAS
  un correctif de lacune bilingue user-facing (contrairement à I1/I2 qui comblaient de vraies
  strings FR-only). Priorité moindre une fois I1/I2 livrés. Deux sous-ensembles :
  - **40 ternaires = pont locale→BCP-47** — ✅ **LIVRÉ (2026-07-05)**. **41 sites** migrés
    vers `intlLocale(locale)` (helper I2b) : home/explorer/prestige/career/donuts/admin+ascension
    helpers/leaderboard/PeriodSessionRail/citations/media/settings/achievements. Collisions
    `const intlLocale` résolues par import aliasé ; `LeaderboardBlock` params resserrés
    `string`→`ManifestLocale`. 6 sites `'en-GB'` (date EU délibérée) conservés. Gate :
    typecheck 0, eslint 0, vitest 261 verts, 0 ternaire pont restant.
  - **Libellés** — SUR-COMPTÉ comme tous les items (vérifié 2026-07-05). Le « 114 » incluait
    dict-selection (`MATCH_VIEW_TEXT[…]`), locale-prop-normalization (`locale === 'en' ? 'en' : 'fr'`)
    et data-selection (`title_en : title_fr`, backend — LÉGITIME). Vrais libellés scattered ≈ 40.
    - ✅ **LIVRÉ (2026-07-05)** : les fichiers HAUTE DENSITÉ (le vrai anti-pattern) — **26
      ternaires** migrés : AscensionProfileTab(16), MatchViewPage(3), PrestigeSquadProgress(3),
      AscensionRealisationsTab(3), AscensionCoachingTab(1) → feature i18n (`getAscensionText`,
      `MatchViewText`). 3 commits `i18n(I4b)`, typecheck/tests verts.
    - **RESTE (accepté / scopé)** : (i) `ArcPresetPicker` — dict local `t={…}` consolidé, typé,
      bilingue = pattern ACCEPTÉ (pas du scattered) ; (ii) **cluster tooltips paramétrés
      DUPLIQUÉ** `MatchEncountersTable`+`ExplorerEncounterBriefing` (`${n} matches as ally`…,
      ~9 sites, #6 cross-feature) = sous-tâche distincte (i18n paramétré partagé) ; (iii) longue
      traîne 1-2/fichier (FeatureUnavailable, SettingsPage, HomePage, HomeAscensionWidget,
      ExplorerTargetSeasonCSR) = **tolérable** (règle plan). Détail DETTE_ASSUMEE §2 I4.
- [x] I5 — CR reco 4 : **LIVRÉ (2026-07-05)**. RECALIBRÉ : la règle remontait **1 warning**
  (pas « >100 ») et n'est PAS couplée à I1-I4. Fix du seul warning (`AscensionProfileTab`
  title `Phase 5 minimale` → clé bilingue `prestigeDisabledHint` FR+EN), puis
  `@levelup/no-hardcoded-strings` passé de `'warn'` à `'error'` (eslint.config.js). Portée
  documentée dans le commentaire de la règle (texte JSX + attributs ; hors args de fonction).
  Gate : typecheck OK, **`npm run lint` = 0 erreur** (rule en error), vitest ascension 62 verts.

Gate I (RÉVISÉ) : `npm run lint` vert AVEC la règle en error → **ATTEINT** (I3+I5 livrés).
La revue visuelle EN des pages I1/I2 (scoreboard, heatmap, onboarding) fait partie du
chantier i18n manuel différé (§7), pas du gate de ce lot.

### LOT J — Performance DuckDB

Objectif : mesurer d'abord (règle audit), puis desserrer les goulots ; batcher les N+1
des chemins HTTP chauds.

> **CALIBRATION J (2026-07-04, vérifiée sur pièces)** : les 9 items sont VALIDES (0 déjà
> fait) — `sql.DBStats` jamais exporté (0 hit), DSN nu sans memory_limit/threads
> (db.go:493-517), `GetHistoryForAvgBulk`/`LoadSquadMatchesBulk` inexistants, magic 4/2
> confirmés (db.go:188-196), emprunt cross-titre non possédant réel (use-after-free).
> **Chemin critique : J1 (mesure) AVANT J2** — les budgets mémoire/threads de J2 sont une
> DÉCISION PRODUIT (stabilité VPS) à escalader APRÈS lecture des mesures J1.
> **J5 = CHANTIER DÉDIÉ** (haut risque : sémantique d'invalidation de cache = décision
> produit ; à traiter avec le chantier K, PAS au fil de l'eau).
> Ordre calibré : J8 → J7 → J1 → [décision J2] → J2 → J3 → J4 → J9 → J6 ; J5 sorti.

- [~] J1 — ARCHI 48 : **(1) LIVRÉ (2026-07-05)** : `duckdb.PoolStatsSnapshot()` (sql.DBStats
  par handle ouvert, sous verrou) + `observability.PublishDuckDBPoolStats` (injection anti-cycle)
  câblé au boot → exposé sous `/debug/vars` `levelup/duckdb_pool_stats` (WaitCount/WaitDuration/
  InUse). Tests : handle présent avec MaxOpenConns=4, absent après Close. **(2) BLOQUÉ RUNTIME** :
  le pool lecture 2-4 conns pour player DBs dépend de la LECTURE des stats SOUS CHARGE (obs.
  runtime VPS) + audit des UPSERT reposant sur MaxOpenConns(1) — impossible en test local.
  Ordre respecté (ne pas inverser). Follow-up runtime.
- [ ] J2 — ARCHI 49 : configurer memory_limit/threads par classe de DB dans
  `openSQLDBFor` (params DSN), valeurs exposées dans /health (8-15 instances x défauts
  80 % RAM = surengagement VPS).
- [ ] J3 — ARCHI 46 : `GetHistoryForAvgBulk` (IN + ROW_NUMBER PARTITION BY xuid), un seul
  Get du SharedReader — remplace jusqu'à ~8 exécutions par clic Match View
  (`match_view_data_loaders.go:386`).
- [ ] J4 — ARCHI 47 : `LoadSquadMatchesBulk` groupé par teammate_xuid + lookup gamertags
  batch (`teammates_service.go:185`).
- [ ] J5 — ARCHI 44 [CHANTIER DÉDIÉ — décidé 2026-07-04] : `LoadAll` full-history par hit
  (`match_history_repo.go:32`, confirmé sans LIMIT ni cache) → cache par joueur invalidé
  post-sync (option A recommandée : entrée PlayerCache invalidée si MatchesInserted>0 via
  finalizer) OU matérialisation du placement (option B). Le push-LIMIT naïf est impossible
  (le placement exige l'historique chronologique complet — nuance vérifiée). DÉCISIONS
  PRODUIT à trancher au chantier : TTL (invalidation immédiate vs 30 min), propriétaire de
  l'invalidation (finalizer engine.run vs sync result handler). À traiter avec le chantier
  K (risque données périmées servies).
- [ ] J6 — ARCHI mineurs N+1 batchables (8 sites, lecture seule) : `sync/engine.go:696`,
  `sync/skill_v2_helpers.go:28`, `relations_moments_service.go:140`,
  `fanout_service.go:73`, `sync/session_recalc.go:80`,
  `sync/backfill_registry_names.go:157` (croisé E2), `handlers/prestige.go:718`,
  `registry_catalog_expand.go:94` (croisé K1d).
- [ ] J7 — ARCHI mineur : CTE perfect de Q26 bornée (agrège tout l'historique pour un
  LIMIT 150) (`queries_home_citations.go:26`).
- [x] J8 — ARCHI mineur : **LIVRÉ (2026-07-05)**. Magic 4/2/1 → constantes nommées
  `poolMaxOpenShared`/`poolMaxIdleShared`/`poolSingleConn` (db.go) sur les 3 sites
  (OpenReadOnly, OpenReadWriteShared, OpenReadWrite) + commentaire expliquant le lien mono-
  process (ADR 0013). Observabilité d'attente = couverte par J1(1). Gate : build+vet OK.
- [ ] J9 — ARCHI mineur : `registry_relations_cross_game.go:81` — emprunt non-possédant
  d'un handle cross-titre → acquisition sûre (refcount/provider du titre visé).

**J2/J3/J4/J6/J7/J9 — DIFFÉRÉS (measure-first, 2026-07-05)** : J1(1) livre l'instrumentation
(expvar DBStats) qui est le PRÉ-REQUIS de la règle « mesurer d'abord ». Les optimisations
(J2 budgets mémoire = **DÉCISION PRODUIT** VPS ; J3 GetHistoryForAvgBulk ; J4 LoadSquadMatchesBulk ;
J6 8 N+1 batchables ; J7 CTE Q26 bornée ; J9 emprunt cross-titre B-swap-safe) doivent être
VALIDÉES par une mesure avant/après SOUS CHARGE (runtime), pas faites à l'aveugle — sinon on
optimise un chemin non mesuré + risque de changement de résultat (J3/J4/J7) / wiring provider
(J9). Chacune a son approche confirmée dans les items ci-dessus. À traiter en tâches ciblées
avec les mesures J1. J5 = chantier K.

Gate J (PARTIEL) : J8 + J1(1) livrés (expvar `duckdb_pool_stats` visible /debug/vars, tests
verts). Optimisations J2-J9 = follow-ups measure-first (voir ci-dessus). J5 = chantier K.

### LOT K — Structure & couches (le plus gros — sous-lots commités séparément)

Objectif : la racine api/ cesse d'être une 2e couche service ; god functions/packages
découpés ; chemins via PathResolver. Chaque sous-lot = 1 commit + build/tests verts.

> **CALIBRATION K (2026-07-04, vérifiée sur pièces)** : contrairement à H/I, les comptes
> de l'audit sont EXACTS (143 fichiers duckdb / 127 service / 112 sync / 40 api, non-test)
> — le lot est bien calibré, il est juste GROS (~4-6 j, 26 items, god-functions/packages
> haut-risque). **DÉCISION (2026-07-04) : K = CHANTIER DÉDIÉ**, exécuté EN DERNIER (ou
> planifié à part), mini-commits séquentiels + smoke-run local après K2a — PAS au fil de
> l'eau avec les lots sûrs. S'y rattachent : **F12** (extraction package film 18 fichiers →
> à faire pendant K3, même nature), **J5** (cache Match History), **F15-6** (RunnerForTitle
> → package agnostique, K-couches), **L7-reste** (double-load boot → K1g). Contraintes
> d'ordre internes : K1 → K2 → K3 (les extractions K3 bougent les fichiers sous les pieds
> de K1/K2 sinon) ; K2a prérequis de K3d ; K1a croise B1-B7/J6 ; K1b prérequis de D2.
> Défauts approuvés : orchestrateurs table-driven `{name, gate, fn}` (K1a/K1f) ;
> PathResolver reste dans domain/ (K1l — réutiliser l'existant registry.go, ne pas en
> créer un 2e) ; DI dans `api/wire/` (K3d) ; client Infinite → `games/halo_infinite/client/`
> sur le modèle games/halo_5/client.go (K3e).

> **BILAN SESSION /goal 2026-07-06** (mis à jour après reprise post-hook — les items durs
> aussi ont été attaqués). **Fait (gated build+vet+intégration/e2e+guards, commité + poussé
> sur `refactor/audits-2026-07`)** : K1a (6 sous-étapes), K1c (sync_meta), K1d (upsert
> ART-safe + guard), K1e (dataQualityHandles B-swap), K1f (BackfillOrchestrator), K1g
> (asset-drawer/CSR → duckdb + dédup double-load), K1i (interfaces consumer-side), K1k
> (**DTO career-live → domain**, 4/5 fichiers décoplés de sync, alias zéro-churn), K1l
> (PlayersRootDir + CacheRootDir, 2 guards), K1n (MedianFloat + EngagementCoefModes), K1h
> (partiel : slug SQL weapon-coverage paramétré), K2b (**pagination de SyncEngine.run
> extraite**, e2e sync vert), K2c (**auto_sync scindé** engine+convergence), K2d (**SeedDemo
> → 4 phases**, intégration ops verte), K2e (strings.Title + goLoad). **RESTE** (items dont
> l'extraction propre est soit infaisable en méthode simple, soit multi-heures/énorme) :
> - `[!]` **K1a cœur** (extraction `service/postsync/`) : `buildPostSyncDeltaHook` couple ~10
>   capacités `*ServiceRegistry` (resolve/homeCache/emitter/bundles/settings) → inversion de
>   dépendance large + cycle `streaks↔duckdb`. Tourne après CHAQUE sync. Multi-heures.
> - `[!]` **K1b** (cascade auth ~130 L) : les 2 cascades DIVERGENT (registry ne marque PAS
>   `reauth_required` sur échec révoqué, le helper canonique SI) → déléguer changerait le
>   comportement bannière prod. Réconciliation comportementale sur l'auth (merge=deploy), pas
>   un dédup mécanique.
> - `[!]` **K1h (reste) / K1j+K1m** : handlers→services+ports (collection) ; catalog/media
>   repos (D-MV2, ~250 L SQL + port + wiring).
> - `[!]` **K2a** : `NewRouter` ~1470 L (huge) ; **K2b drain** non extrait (defer-lifecycle
>   load-bearing, cf. commit K2b) ; **K2c reste** (run-loop → auto_sync_run.go pour < 500 L).
> - `[!]` **K3a-f** : scissions god-packages (duckdb 143 / service 127 / sync 111 fichiers) —
>   énormes, « 1 domaine = 1 commit », mécaniques mais volumineuses.
> Reprise recommandée : K1a → K2a → K3d (ordre du plan).

K1 — Extractions de couches (ROI d'abord) :
- [~] K1a — ARCHI 1 + 2 (reste) + CR A2 (partie post-sync). **« au passage » FAIT+VÉRIFIÉ
  (2026-07-06)** : `outcome = 2` → `duckdb.OutcomeSQLEqSlug(pdb.TitleSlug, …)` title-aware
  (`post_sync_progression_queries.go:306`, plus aucun littéral en requête) ; seuils nommés
  `kdRatioThresholdStep`/`winrateThresholdStep` (0.05) + `bestKDARecordEpsilon` (0.01)
  (`post_sync_deltas.go:56-60`) ; `EmitPostSyncDeltas` déjà réduit 247 → 147 L
  (`//nolint:funlen,gocyclo` justifié : émetteur multi-événements comparant ~12 snapshots).
  `[!]` **BestKDA quotient** = DETTE DOCUMENTÉE prod-gated : la requête calcule
  `(kills+assists)/GREATEST(deaths,1)` ; le fix ADR 0006 (kda natif) ne peut PAS s'appliquer
  seul (records best_kda persistés sur l'échelle quotient → resteraient bloqués), exige un
  RE-BACKFILL coordonné (op data/prod, hors branche refacto) — commentaire explicite en code.
  `[!]` **cœur RESTE** : extraction pipeline post-sync api/ → `service/postsync/` + repos →
  duckdb + formules → analysis/ = inversion de dépendance large (relocation
  `CoachAdvisorBundle`/`PrestigeBundle` hors api pour éviter le cycle api↔postsync +
  interface 6-capacités) — gros déplacement cross-package, session dédiée.
- [~] K1b — ARCHI 5 : cascade refresh tokens de `registry_auth.go` dupliquant le pipeline
  MSAL→OAuth. **FAIT (2026-07-06) — dédup de la cascade store (cœur), ZÉRO changement de
  comportement** : la cascade MSAL silent → OAuth refresh+rotation est extraite en source
  unique `auth.RefreshFromStoreEntry` (ex-`tryRefreshFromUserEntry`, param `store` élargi à
  l'interface `UserTokenStore` — seul `UpdateOAuthRefreshToken` y est appelé, PAS besoin
  d'élargir l'interface). `registry.tryRefreshFromAuthStore` délègue et CONSERVE sa politique
  exacte (clear-on-success ; JAMAIS de marquage reauth sur le chemin serveur haute-fréquence ;
  erreur désormais LOGUÉE, jamais avalée). Les 4 tests registry (rotation persistée / no-write /
  clear-on-success / no-clear-on-failure) passent INCHANGÉS = preuve de non-régression.
  Gate : build+vet 0, tests auth cascade (canonique+registry) + intégration -p 1 auth verts.
  `[!]` RESTE : dédup du CHEMIN LEGACY (sync_meta DuckDB + env var) NON faite — la déléguer à
  `tryRefreshFromLegacyInputs` introduirait 3 divergences (marquage reauth, rotation→DuckDB si
  store nil, granularité télémétrie env-vs-duckdb) sur un chemin DÉPRÉCIÉ (retrait Phase 5/D2).
  Réconciliation à faire AVEC D2 quand le legacy disparaît (élimine les divergences). Suit la
  logique « pré-requis D2 » du plan.
- [x] K1c — ARCHI 3 (2026-07-06) : helper unique `duckdb.WriteSyncMeta` (+ `ReadSyncMeta`)
  déjà source unique des 2 ex-copies (`notifications_title_ready`/`_boot`, dédup #6). **Reste
  livré ici** : le durcissement « écriture SOUS LEASE dblease » (ADR 0013, un seul writer
  par DB) — `WriteSyncMeta` acquiert `AcquirePlayerWriterTimeout(dblease.PlayerLeaseTimeout)`
  + `defer Release()` AVANT l'`OpenReadWrite` (modèle `match_exclusion_repo.go`). Sûr :
  les 2 seuls appelants (boot `EmitAppReleaseForAllPlayers` via `reg.resolve` handle RO ;
  notifier title-ready « à la fin d'un cycle ») ne tiennent PAS le lease player →
  acquisition libre, zéro ré-entrance. Gate : build+vet 0, intégration -p 1 duckdb (99 s)
  + api (17 s) vertes séparément (combiné = flake SharedProvider reopen documenté).
- [~] K1d — ARCHI 4 : `ExpandPlaylistChildren` (`registry_catalog_expand.go:94`) → ops/ ou
  service/ ; DDL → internal/migration ; factoriser la 3e copie du pattern upsert ART-safe ;
  batcher ses 3 requêtes/entry (croisé J6).
  - [x] **Dédup upsert ART-safe FAIT (2026-07-05)** : canonique `duckdb.UpsertRowNoConflict`
    (`db.go` — la méthode `*DB.UpsertNoConflict` délègue) ; les 3 copies (`ops/catalog_refresh`,
    `service/catalog_fetcher_service`, `api/registry_catalog_expand.upsertPlaylistWeight`)
    pointent dessus, SAUF la copie service — gardée volontairement car ADR 0025 D-MV2 interdit
    à service d'importer duckdb (allowlistée). Garde-rail #6 :
    `archlint/no_local_upsert_helper_test.go`. Build+vet+gate intégration (duckdb+ops+api) verts.
  - [!] Reste (relocation `ExpandPlaylistChildren` hors racine api/, DDL→migration, batch 3-req)
    : couplé à la sortie du post-sync de la racine api/ (famille K1a) — session dédiée, pas
    fait ici pour ne pas mélanger un déplacement de package avec la dédup gated.
- [x] K1e — ARCHI 6 (2026-07-06) : `dataQualityHandles(ctx, titleSlug)` route la lecture
  shared via `cfg.SharedProvider.Get` (réutilise `acquireProgressionSharedRead`, drain
  RO↔RW résilient) QUAND le provider tient ce fichier (titre par défaut = seul shared pris
  en RW en process) — élimine le conflit "different configuration" des 5+ runners admin
  concurrents à un sync. Autres titres (aucune fenêtre RW) → `OpenReadOnly` conservé. ctx
  threadé aux 8 callers (registry_actions ×2, asset_names_sweep, catalog_drain,
  catalog_expand, data_quality ×2, weapon_coverage). Gate : build+vet 0, intégration -p 1
  (api) verte.
- [x] K1f — CR A3 (2026-07-06) : `service.BackfillOrchestrator` extrait
  (`backfill_orchestrator.go`) ; `handleStartBackfill` réduit à validation (400/404/409) +
  wiring SyncEngine + création job + 202 (~50 L, plus de nolint:funlen). Décomposé en
  phases `Run`/`runCitationsComeback`/`runWeaponsEngagement`/`runEventsLusr`/`runCsrPerfPsa`/
  `warnUnimplemented` (chacune ≤ 80 L). Extraction FIDÈLE (mêmes libellés d'étape, warnings,
  résumé, ordre de phase — pas de refonte table-driven qui aurait risqué un écart de
  comportement sur ce pipeline hétérogène). Tests `warnUnimplemented` migrés vers service
  (package `service`). Gate : build+vet 0, tests handlers+service verts, intégration -p 1
  (service+handlers) verte.
- [x] K1g — CR A2 + A1 (partie) (2026-07-06) : SQL de `loadTitleAssetDrawerData` +
  `loadCSRBadgeResolver` → `platform/duckdb/title_asset_drawer_loader.go`
  (`LoadTitleAssetDrawerData` + `LoadCSRBadgeMap`) ; les wrappers server.go ne gardent que
  open+delegate (modèle `loadTitleRankImageURLs`). Double chargement metadata H5 au boot
  SUPPRIMÉ : `loadTitleAssetDrawerData(h5)` était appelé 2× (AssetMetadataHandler + adapter
  TitleAssetURLAdapter) → hoisté UNE fois avant le bloc, réutilisé aux 2 endroits. Gate :
  build+vet 0, intégration -p 1 (api+handlers) verte.
- [~] K1h — CR A5 + ARCHI mineurs handlers : handlers qui construisent repos/métier →
  services + ports : `progression.go:162-251` (ProgressionService, jointure catalog x
  earned hors handler), `campaign.go:221`, `admin_auto_sync.go:206`, `sync_handler.go:165`,
  `player_profile.go`, `home.go` ; `bootstrap.go:23`/`title_sync.go:30` → port.* ;
  `commendation_handler.go` → api/handlers/.
  - [x] **`registry_weapon_coverage.go:102` FAIT (2026-07-06)** : slug concaténé dans le SQL
    → **paramétré** (`title_slug=?` + arg). Injection-hardening. Build+vet 0.
  - [x] **`progression.go` jointure milestones FAIT (2026-07-06)** : `handleMilestones` réduit
    à resolvePlayer + 202 ; la jointure catalog × earned → DTO extraite en helper `milestoneDTOs`
    (logique hors méthode HTTP). Build+vet 0. (Le passage en SERVICE + port reste bloqué par
    D-MV2 : les repos milestones sont des types duckdb ; un vrai service exigerait des ports
    par repo — churn disproportionné, `[!]` reste de K1h en collection incrémentale.)
- [x] K1i — ARCHI 8 + CR A5 (2026-07-06) : interfaces consumer-side étroites (1 méthode
  chacune) remplacent les champs concrets. `HomeService.careerLive` → `homeSpartanIdentityProvider`
  (`GetSpartanIdentity`). `CareerService.seasonsCatalog` + `FiltersService.catalog` →
  `seasonsCatalogLoader` (`Load`) partagé (défini dans seasons_catalog.go). Setters gardent
  le param concret + garde `if x != nil` (nil-check concret fiable → évite le piège
  interface typed-nil) ; `*CareerLiveService`/`*SeasonsCatalog` satisfont structurellement,
  mockables en test (même package). Gate : build+vet 0, tests service verts.
- [~] K1j — ARCHI 9 (2026-07-06) : **persistance catalogue extraite** —
  `catalog_fetcher_service` ne tient PLUS de *sql.DB ni de SQL : `port.CatalogWriter`
  (interface consumer-side étroite : SelectPending + Upsert{Playlist,Pair,Map,GameVariant})
  implémenté par `duckdb.CatalogWriterDB` (nouveau, ~180 L SQL déplacé, utilise
  `UpsertRowNoConflict` canonique). Le service ne garde que l'orchestration (drain loop +
  policy is_ranked rankedplaylists). **Ferme la boucle K1d** : la copie locale forcée
  `upsertRowNoConflict` est SUPPRIMÉE → garde-rail archlint durci (plus aucune exception hors
  duckdb/db.go). 2 constructeurs + test rewirés. Gate : build+vet 0, D-MV2 + upsert guards
  verts, **intégration catalog drain end-to-end verte** (playlist+pair+labels+re-enqueue),
  duckdb+api intégration verts. `[!]` RESTE : `openspartan_post_import_service.go` (même
  dérive *sql.DB, 2e service) ; gate-titre explicite sur rankedplaylists (actuellement
  effectif car les IDs H5 ne matchent pas l'allowlist HI — clarification mineure).
- [~] K1k — ARCHI 10 (2026-07-06) : **types promus dans domain** — `domain.CareerRankSnapshot`
  (ex-`sync.CareerRankData`, renommé pour éviter la collision avec `domain.CareerRankData`
  calculé) + `domain.SpartanCustomizationData` (`domain/career_live.go`). sync garde des
  **alias** (`type CareerRankData = domain.CareerRankSnapshot`) → ses ~55 usages (+ duckdb/
  games/port) INCHANGÉS. Les 5 fichiers `career_live_*` référencent les types domain ; **4/5
  cessent d'importer internal/sync** (cache/merge/partial/service décoplés). Gate : build+vet
  0, tests career-live (service+sync) verts. `[!]` RESTE : `career_live_fetcher.go` importe
  encore sync pour la FACTORY (`sync.NewHaloAPIClient` + compile-check `*HaloAPIClient`) →
  déplacer la factory côté registry (api) est le dernier pas du découplage (débloque H5).
- [~] K1l — ARCHI 11 + 14 + mineurs chemins : TOUS les chemins via PathResolver :
  `openspartan_import_service.go:481` + `server.go:1344` (stash friends layout legacy),
  `server.go:290/651/1112` (data/cache — ajouter `CacheRootDir()` au resolver),
  `ops/seed_demo.go:392` (MediaDataDir existant), `seed_demo_multititle.go:43` (layout
  réimplémenté), helper `PlayersRootDir(slug)` (6 copies du filepath.Join, dont
  `data_health_check.go:257`), `config.go:205` (double mécanisme data/auth + data/sessions).
  Garde-rail archlint en L2.
  - [x] **`PlayersRootDir(slug)` FAIT (2026-07-05)** : `PathResolver.PlayersRootDir` ajouté
    (registry.go), `PlayerDir` délègue dessus ; **7 copies** migrées (le garde-rail a
    débusqué une 7e hors du grep initial : `cmd/levelup/cmd_title.go:137`) —
    ops/backup_service, ops/healthcheck ×2, scheduler/data_health_check,
    service/media_index_service ×2, cmd_title. Garde-rail #6 :
    `archlint/no_players_root_join_test.go`. Build+vet+gate intégration
    (ops+scheduler+service+domain/title) verts.
  - [x] **`CacheRootDir()` FAIT (2026-07-06)** : `PathResolver.CacheRootDir()` ajouté,
    `JobsCachePath` délègue dessus ; 3 sites `data/cache` de server.go (jobsPath, assetCfg
    CacheRootDir, HelpHandler) migrés via `NewPathResolver(cfg.RepoRoot).CacheRootDir/JobsCachePath`.
    Build+vet 0, test resolver vert.
  - [!] Reste K1l (stash friends, seed_demo layout, config.go double mécanisme data/auth+sessions,
    garde-rail data-path global L2) : non fait ici — chemins hétérogènes nécessitant chacun une
    décision de resolver (session chemins dédiée).
- [x] K1m — ARCHI 12 (2026-07-06) : **allowlist D-MV2 VIDÉE** (`no_duckdb_import_test.go` →
  `map[string]bool{}`, zéro service important `internal/platform/duckdb`). Résolu par SUPPRESSION
  de code mort plutôt qu'extraction d'un repo : `media_index_service.resetPlayerMediaIndex`
  (seul importeur de duckdb) était un NO-OP (ouvrait lease + RW handle pour `_ = db; return nil`)
  depuis `drop_media_from_player_db` (media → shared_social append-only) → fonction + cérémonie
  « reset » (progression 0-50 %) supprimées ; réindexation passe en 0-100 %. `media_service.go`
  n'importait DÉJÀ plus le package data (entrée d'allowlist périmée). `ResetAndReindex`/`ScanAllMedia`
  (contrat MediaIndexer, sentinelle) INCHANGÉS. Gate : build+vet 0, `TestServicesDoNotImportDuckDB`
  vert (allowlist vide), tests media indexer verts. Comportement identique (le reset était déjà
  no-op) hors libellé de progression.
- [~] K1n — ARCHI mineurs couches service/analysis/domain. Médiane centralisée FAITE
  (`analysis.MedianFloat`, 3 copies, commité plus tôt). Liste de modes dupliquée FAITE
  (2026-07-06) : `domain.EngagementCoefModes()` source unique — `sync.engagementCoefModes`
  + `service.engagementCoefModesService` s'y alignent. **Reste STATUÉ** (règle plan
  « statuer : toléré si documenté, sinon déplacer ») :
  - `combat_yield.go:28` (état global atomique) : `[~]` TOLÉRÉ — impureté DOCUMENTÉE et
    délibérée (réglage app-unique, évite de threader le flag dans ~13 agrégateurs purs ;
    commentaire explicite « PAS un guard de compatibilité »). Conforme à la règle.
  - `comeback.go:130` / `sql_fragments.go:26` / `perfect_kills.go:28` (slog + fragments
    SQL en analysis/) : `[~]` TOLÉRÉ documenté — logs diagnostic best-effort / fragments
    SQL partagés déjà commentés ; les extraire exigerait de threader un canal diagnostic
    pour un bénéfice marginal.
  - Algos purs → analysis/ (`engagement_timeseries_binning`, `timeseries_service_aggregations`,
    `teammates_squad_charts_intensity_perminute`, `match_history_placement` regex 2 copies
    intra-≤2) ; `friends_orchestrator_service.go:102` → port ; `mode_label.go:49`,
    `identity.go:162`, `citations.go:174`, `world_stats.go:153`, `home_kpis.go:12` (croisé
    G2), `domain/achievement_categories.go:27` + `domain/job.go:92` (croisé F15) : `[!]`
    déplacements de couche de valeur structurelle FAIBLE (chacun un mini-refactor +
    callers) — reportés au profit des items structurels à fort levier (K1e/K1h/K1a). Non
    bloquants, aucun n'excède les seuils lint.

K2 — God functions à risque (tâche dédiée, passer la grille plan-review avant chaque item) :
- [~] K2a — ARCHI 22 + CR A1 : `NewRouter` ~1 470 L → extractions par bloc (compiler-vérifié,
  gate api intégration après chaque). **FAIT (2026-07-06)** : `buildAssetMetadataHandler` +
  `wireHalo5AssetAdapters` (~134 L) PUIS `buildTitleRuntime` (~148 L : tout le bloc « Phase B
  multi-titres » — resolver d'adapters semantic/data/assetURL HI + images de rang par titre
  actif + capabilities HI, sorties regroupées en struct `titleRuntime`). **NewRouter ~1 470 →
  ~1 197 L**. Gate à chaque : build+vet 0, intégration -p 1 api verte (NewRouter boot OK).
  `[!]` RESTE : poursuivre par bloc (`buildStores`, `applyMiddlewares`, `mountXxx` par domaine,
  `waypointExplore`→service, purge sessions goroutine) — la cible < 100 L exige une bascule
  builder-pattern (assemblage DI intrinsèquement séquentiel) ; réduction incrémentale continue.
- [~] K2b — ARCHI 23 + CR A4 (2026-07-06) : **boucle de pagination EXTRAITE** de `run()` →
  méthode `paginateAndPersistHistory(ctx, historyPaginationInputs, *SyncResult)` (~155 L,
  `//nolint:funlen` justifié : filtre/fetch/persist par page partagent processed/toFetch/
  stopAfterFlush ; découper disperserait le critère d'arrêt delta). État groupé en struct
  (≤5 params). **Gate : build+vet 0, e2e sync `-tags=integration -p 1` VERT (103 s)** —
  comportement byte-identique. `[!]` Bloc drain/ré-acquisition (l.505-597) NON extrait :
  **techniquement infaisable en méthode simple** — il gère les leases via des `defer` au scope
  `run()` (postPH.Close / wp.Release / postRls) dont le timing LIFO au retour de run() est
  load-bearing (prévention auto-deadlock, ADR 0016) ; les déplacer en méthode les ferait
  tourner tôt → deadlock/corruption. Laissé en place (documenté).
- [~] K2c — ARCHI 24 (2026-07-06) : `auto_sync.go` scindé — `auto_sync_engine.go` (150 L :
  `BuildEngine` factory + `defaultRunnerFactory`/`acquireLiveTitleRunner`/`resolveTitleSlug`)
  + `auto_sync_convergence.go` (89 L : passe convergence events + `warnStaleGateClaims`).
  auto_sync.go 1101 → 887 L. Gate : build+vet 0, tests scheduler unit+intégration -p 1 verts
  (déplacement pur, compiler-vérifié). **run-loop extrait (2026-07-06)** : `Run`/`RunOnce`/
  `RunOnceTrigger`/`syncPlayersConcurrent`/`syncPlayer`/`checkSyncPreconditions` + type
  `syncOutcome` → `auto_sync_run.go` (455 L, même package = zéro churn d'import). **auto_sync.go
  887 → 445 L (< 500 ✓)**. Gate : build+vet 0, intégration -p 1 scheduler (4,8 s) verte.
  `[!]` RESTE mineur : `BuildEngine`/`RunOnceTrigger` > 80 L portent leur `//nolint:funlen`
  justifié (cohésion) — décomposition optionnelle, seuil couvert par exemption.
- [x] K2d — ARCHI 25 (2026-07-06) : `SeedDemo()` ~203 L → orchestrateur (~55 L) + 4 phases
  nommées : `resolveDemoCorpusAndRoster` (0/1/1b : corpus figé/dynamique + roster),
  `buildDemoWarehouse` (2-4 : metadata+shared+anonymisation+migration), `seedDemoPlayerDBs`
  (5), `seedDemoMediaFiles` (6). Extraction FIDÈLE (mêmes libellés/erreurs/ordre ; sémantique
  `res.*` sur erreur préservée — MetadataCopied/Frozen positionnés avant le retour d'erreur).
  Gate : build+vet 0, **intégration -p 1 ops (SeedDemo end-to-end sur sources synthétiques)
  VERTE** (13 s) — c'est le gate « regen demo » demandé (les tests construisent les DB
  sources + lancent le pipeline + vérifient le résultat).
- [x] K2e — CR mineurs (2026-07-06) : `strings.Title` déprécié (`engine.go`) supprimé →
  `mode, modeTitle := "full","Full"` (dual-assign inline, plus simple qu'une map pour le
  site unique). ~18 blocs `g.Go` copiés (`match_view_data_loaders.go`) → helper `goLoad(gctx,
  g, matchID, label, load)` (best-effort + `slog.WarnContext`, jamais fatal) ; les 2 blocs
  non-uniformes (eventsRepo Validate/ErrCapabilityNotSupported) restent en g.Go brut. Gate :
  build+vet 0, tests match_view verts.

K3 — God packages & structure (mécanique, 1 domaine = 1 PR/commit) :
- [ ] K3a — ARCHI 17 : platform/duckdb (143 fichiers) → extraction par domaine, commencer
  par prestige ; `halo5_*.go` → games/halo_5 ou duckdb/halo5.
- [ ] K3b — ARCHI 18 : service/ (127 fichiers) → sous-packages par feature, commencer par
  teammates (13 fichiers) ; archlint interdit les imports croisés entre features.
- [~] K3c — ARCHI 19 : sync/ (111 fichiers) → extraire sync/skill/ (17) et sync/snapshot/
  (6) ; ratchet de gel sur la racine ; le neuf va dans v2. **FAIT (2026-07-06) — ratchet de
  gel** : `archlint/sync_root_freeze_test.go` gèle la racine sync/ à 112 fichiers .go non-test
  (baseline DÉCROISSANTE : échoue si un nouveau fichier racine est ajouté ET si le compte
  descend sans abaisser la baseline). Doctrine ADR 0027 : le neuf va en v2/ ou sous-package.
  **sync/snapshot EXTRAIT (2026-07-06) ✅** : le cluster snapshot (11 fichiers : cutter/metrics/
  readiness/readiness_eval/report/shared_reader + tests) → `internal/sync/snapshot` (package
  `snapshot`). Cycle bidirectionnel rompu par la **technique du package feuille + ré-export** :
  (1) les constantes `MBit*` (pures `1<<N`) → package feuille `internal/sync/matchflags` ;
  `sync/backfill_flags.go` les RÉ-EXPORTE (const alias) → **ZÉRO requalification** des ~centaines
  d'usages (sync/domain/ops/scheduler/cmd) ; (2) snapshot importe `matchflags` (feuille) au lieu
  de sync ; (3) `slugHasLUSR` localisé dans snapshot (2 copies triviales, ≤2) ; (4)
  `evaluateSnapshotReadiness` → exporté `EvaluateSnapshotReadiness`, `engine_postsync` câblé sur
  `snapshot.EvaluateSnapshotReadiness` ; (5) 2 callers externes requalifiés (registry_pages,
  sync_v2_wiring). **Aucun cycle** (snapshot→matchflags feuille ; sync→snapshot uni-directionnel).
  Baseline ratchet 112 → 106. **Gate : build+vet 0, archlint + auth-sentinel + duckdb-ratchets
  verts, snapshot intégration + sync intégration (101 s, anti-ART+e2e) + api intégration verts** —
  comportement préservé. **LEÇON (corrige la mesure « ~400 refs impossible » précédente)** : la
  ré-export des constantes partagées rend les scissions cross-package FAISABLES sans requalification
  de masse. Applicable à K3a/b/d/e. `[!]` RESTE : sync/skill/ (17 fichiers, même technique).
  **2e tentative (2026-07-06, backfillflags feuille) → 3e NIVEAU DE CASCADE prouvé** :
  `backfill_flags.go` n'est PAS auto-contenu — sa map `BackfillFlags` référence `MetricKeyAccuracy`
  + `BackfillType*` (définis DISPERSÉS dans citations/performance/scope/skill_config, MÊLÉS à de
  la logique). Extraire les flags exige donc d'extraire AUSSI MetricKey*/BackfillType*, qui
  cascadent encore. **CONCLUSION EMPIRIQUE DÉFINITIVE** : le god-package sync/ est
  STRUCTURELLEMENT non-subdivisable proprement — constantes/helpers tissés sur 3+ niveaux de
  cascade et des centaines de références. Les scissions K3 ne sont PAS un travail de session mais
  un refactor pluri-semaines à requalification de masse. Revert propre, branche verte.
- [ ] K3d — ARCHI 20 : racine api/ (39 fichiers) → api/wire/ pour la DI ; cible < 10
  fichiers racine (le post-sync est déjà parti en K1a).
- [ ] K3e — ARCHI 21 : client HTTP Halo Infinite (halo_client*.go, 7 fichiers) →
  platform/halo/ ou games/halo_infinite/client/ (cible montrée par games/halo_5/client.go).
- [~] K3f — ARCHI mineurs structure. **FAIT (2026-07-06) — magic numbers** : `rows[:50]`/
  `matches[:50]` (analysis home highlights) → const partagée `maxSessionlessHighlights = 50`
  (home_highlights.go, réutilisée par la variante canonique) ; `window := 15` (media
  match-candidates) → `defaultMediaMatchWindowMinutes = 15` (media.go). Build+vet 0, tests
  analysis+handlers verts. **Débris nolint/doc** : les lignes citées par l'audit ont dérivé —
  ce qui s'y trouve désormais (`post_sync_deltas.go`, `teammates_squad_charts_impact_events.go`)
  sont des `//nolint:funlen` JUSTIFIÉS + doc de rationale valide, pas des orphelins → rien à
  purger. `[!]` RESTE (god-files 10 items + décisions de packages notify/prestige/campaign/
  metadata/openspartan) : structurel, suite dédiée.
  - `[!]` détail RESTE : doublon `notify/` vs `notifications/` (renommer ou
    documenter) ; `prestige/` + `campaign/` → progression/ (ou documenter le choix) ;
    `worldenrich/` câblage top-level ; frontières `assets/`/`assetnames/`/`media/`
    documentées ; `metadata/` renommé ; `openspartan/` → platform/ ; `legacymatch` échéance
    documentée [TRACKÉ] ; `migration/steps_*` Halo-specific [TRACKÉ ADR 0025 Ph 1.5 — statuer
    seulement] ; PathResolver dans domain/ (documenter le choix) ; god-files secondaires
    → découper OU exemption. **FAIT (2026-07-06)** : (1) `handlers/prestige.go` 1 019 L → 4
    fichiers même-package (prestige.go 353 + prestige_arcs 142 + prestige_squad_challenges 92 +
    prestige_squads 470), tous < 500 ; (2) `games/halo_infinite/adapter_data.go` 746 → 472 L +
    `adapter_data_career.go` 287 L (méthodes carrière/historique) ; (3) `games/halo_5/adapter_data.go`
    641 → 379 L + `adapter_data_loaders.go` 277 L (carrière + chargement matchs) ; (4)
    `api/registry_pages.go` 851 → 294 L + `registry_pages_explorer.go` 323 + `registry_pages_home.go`
    259 (factories Explorer / Home-MatchHistory-Squad) ; (5) `platform/duckdb/persist_sink.go`
    745 → 312 L + `persist_sink_items.go` 203 + `persist_sink_challenges.go` 259 (INSERT-only
    ART-safe INCHANGÉ, ADR 0019). Gates : build+vet 0, tests package + intégration api verts ;
    **persist_sink : intégration -p 1 duckdb anti-ART verte (100 s)** — logique persist intacte.
    **Piège noté** : goimports STRIP l'alias custom `sync_pkg` → import ajouté à la main +
    `gofmt` seul (jamais goimports) sur les fichiers à alias non-inférable.
    (6) `platform/duckdb/db.go` 757 → 484 L + `db_recovery.go` 152 (invalidation/reopen) +
    `db_query.go` 142 (Query/Exec/*Recovered). Gate build+vet 0 + intégration -p 1 duckdb verte
    (anti-ART OK ; ratchet `TestNoUnauthorizedSharedSocialMention` mis à jour : commentaire policy
    déplacé vers db_query.go → allowlisté). (7) **steps.go / steps_player_base.go** (god-FONCTIONS `Steps()`/`playerBaseSteps()` =
    slice littéral de migrations ORDONNÉ, ordre audité par `order_audit_test.go`) → **EXEMPTION
    JUSTIFIÉE** (plan « découper OU exemption ») : exclusion `.golangci.yml` ajoutée pour
    `internal/games/.*/migrations/steps` (funlen/gocyclo/lll), MIROIR de l'exemption existante
    `internal/migration/steps_`. Découper fragmenterait le registre ordonné sans gain ; la
    longueur = nombre de migrations. (8) **pool.go ×3 fns** (`NewPool` constructeur DI,
    `OnHTTPError` machine à états backoff, `refresherLoop` boucle goroutine) + **prestige/service.go
    `CreateChallenge`** → **EXEMPTIONS `//nolint:funlen` JUSTIFIÉES** (fonctions cohésives
    lifecycle/orchestration ; décomposer fragmenterait un flux unique). build+vet 0. **✅ K3f
    god-files : TOUS traités** (6 splits même-package + steps exemptés config + 4 fns exemptées
    nolint). **NOTE** : `sync/skill_v2_shadow.go` NON splittable en
    place — le ratchet de gel K3c interdit un nouveau fichier racine sync/ (doit aller en
    sous-package, cf. K3c reste). RESTE (7) : steps.go (migration — ordering sensible),
    persist_sink.go (ART-critique), db.go, registry_pages.go, pool.go x3 fns, prestige/service.go
    CreateChallenge, steps_player_base.go → splits même-package OU exemption, suite mécanique.

Gate K (par sous-lot) : `go build ./... && go test ./...` ; pour K2a/K2b :
`go test -tags=integration ./internal/sync/...` + diff openapi VIDE (aucune route
changée) ; archlint verts ; smoke run local du serveur après K2a.

### LOT L — Gouvernance, ratchets, contrat HTTP

Objectif : chaque règle d'architecture de l'audit finit soit corrigée (lots précédents),
soit ENCODÉE en ratchet à allowlist décroissante datée (reco centrale ARCHI).

> **CALIBRATION L (2026-07-04, vérifiée sur pièces)** : L2 HÉRITE des garde-rails différés
> par F : **F15-12** (miroir coarse↔fine souple — livrable F7), **F15-14** (cap⟺scalaire
> damage_model/team_mmr), **F6-parité** (chaque titre déclare les FieldKeys de ses
> capability-groups). La règle L2-(1) (SQL dans api/) dépend de K → livrer (2)+(3)+hérités
> d'abord, (1) après le chantier K (baseline datée sinon).

- [!] L1 — ARCHI 27 [TRACKÉ] : **APPROCHE INVALIDÉE (2026-07-05), à re-scoper**. Testé sur
  pièces : le mode emit résout bien les 22 DIVERGENT à **0** (script de remplacement des
  blocs de schéma, vérifié par le drift-test) MAIS **la régénération de generated.ts CASSE
  le typecheck front** (`appShellStore.ts` : `data.auth_state`/`setup_state`/`auth_mode`/
  `registration_mode` deviennent `string` au lieu des UNIONS d'énums ; `available_players`
  perd `| null`). CAUSE : Huma dérive `string` des champs Go `string` — l'openapi.yaml
  MANUEL est INTENTIONNELLEMENT PLUS RICHE (énums + nullabilité ajoutés à la main). Donc
  une partie des 22 DIVERGENT = **enrichissement voulu, PAS de la dérive** ; les adopter
  DÉGRADE le contrat, et durcir DIVERGENT→0 (t.Errorf) FORCERAIT à supprimer ces
  enrichissements. Le log-only actuel est donc ~correct. Re-scope requis : catégoriser les
  22 (dérive réelle → fix Huma ; enrichissement → GARDER + allowlist) et durcir avec
  allowlist, PAS à 0. Reverté (openapi.yaml + generated.ts intacts). Voir §7. Follow-up ciblé.
- [~] L2 — ARCHI TOP3 + hérités F : **PARTIEL (2026-07-05)**. (2) **LIVRÉ** :
  `archlint/no_data_path_join_test.go` (interdit `filepath.Join(..."data"...)` hors
  PathResolver dans internal/ ; allowlist décroissante datée = 9 sites bootstrap légitimes :
  registry.go=resolver, config/ (défauts data-root), api/server.go (DI), ops/seed+migrate+
  backup, testfixtures/paths.go). (1) SQL/`Open*` dans api/ = **après K** (dépendance
  documentée). (3)(4)(5) hérités F (parité coarse↔fine, cap⟺scalaire, fields.toml par
  capability-group) = `[!]` À BÂTIR — ratchets de parité capability/config, plus impliqués
  (nécessitent l'API CapabilityMap + resolvers scalaires) → follow-up L2-suite (§7).
- [x] L3 — CR A20 : **LIVRÉ (2026-07-05)**. Exclusion blanket `argument-limit` RETIRÉE (la
  règle était DÉSACTIVÉE) + funlen 100→80. **Mesure sur pièces** (golangci-lint 2.12.2, run
  complet) : 43 funlen>80, 151 argument-limit>5 dont **89 à 6 args + 29 à 7** (idiome
  orchestration) puis queue nette de **33 à ≥8** (monstres jusqu'à 14). DÉCISION (mienne,
  non-triviale) : argument-limit=**7** (pas 5-Python) — cible la queue ≥8 sans refactorer
  l'idiome établi. Baseline = `only-new-issues: true` DÉJÀ en CI (grandfather ~479 dette) →
  pas besoin de `--new-from-rev` séparé. Vérifié via `--new-from-rev=main` : **0 issue
  L3-causée** sur la branche (1 seul funlen-caused corrigé : `enrichRow` 83→<80 par
  extraction `enrichMapWinRate`). Config `golangci-lint config verify` OK.
- [x] L4 — ARCHI mineurs contrat : **LIVRÉ (2026-07-05)**. `contract_validate.go` relu
  (confirmé : Content-Type/JSON/error-shape mais AUCUN schéma, bufferise tous les corps),
  SUPPRIMÉ (D-E.15) avec son test `contract_validate_test.go`, le `r.Use(ContractValidate)`
  (server.go) et les commentaires. Tests contract-validation retirés de
  `middleware_internal_test.go` (`TestResolveTitleSlug` conservé). Vérifs : `errKey*` restent
  (définis dans require_auth.go), flag LEVELUP_CONTRACT_VALIDATE plus référencé. `read_budget.go`
  couplé sharedprovider : exception DOCUMENTÉE (M4 le teste, commentaire déjà en place). Gate :
  build+vet OK, api+handlers+middleware verts, 0 ref résiduelle (hors commentaires « retiré »).
- [x] L5 — CR A16 : **LIVRÉ (2026-07-05, commit 91492e360)**. Le « ~180 queryKey » de
  l'audit = la plupart consommaient DÉJÀ `queryKeys` ; le vrai chantier = **7 registres
  feature-local** (prestige/arc/challenge/squad/profileKeys + watcher/adminKeys) + **8
  littéraux inline** (les 4 restants = extensions légitimes `[...queryKeys.X]`).
  - 7 registres repliés en namespaces `queryKeys.{prestige,arc,challenge,squad,
    playerProfile,watcher}` + `queryKeys.adminUsers`. **Clés IDENTIQUES au byte** → zéro
    changement de cache ; typecheck = filet (réf manquée = erreur compile, pas bug muet).
  - 8 littéraux → clés dédiées (+ préfixes broad `filtersResolveAll`/`adminDataQualityIssuesAll`).
  - Garde-rail `lib/query/keys.guard.test.ts` (fs-grep node) : interdit tout registre
    `*Keys` feature-local ET tout `queryKey: ['…']` littéral (autorise `[...queryKeys]`).
  Gate : typecheck 0, eslint 0, vitest 438 verts (11 features), garde-rail vert (mord).
  NB : garde-rail vitest fs-grep plutôt que règle ESLint custom (précédent `calendar.guard`,
  moins d'infra) — couvre le même invariant.
- [~] L6 — PRÉ-EXÉCUTÉ, VÉRIFIÉ 2026-07-04 : la convention kill-switch est bien dans
  CLAUDE.md règle 11 (date de basculement + date cible de retrait + critère mesurable,
  modèle shared_reader_legacy.go) + skill arch-rules §Feature flags + delivery-checklist
  §5. Rien à faire.
- [~] L7 — PARTIEL, réf K1g : la title-paramétrisation est FAITE (F15-4 :
  `loadTitleAssetDrawerData(metaPath, slug)`, pas de fallback HINF) ; il RESTE la double
  lecture au boot (`server.go:777` vs `:820`, 2 appels) dont la fusion appartient à K1g
  (extraction platform/duckdb + dédup). Suivi au chantier K.

Gate L : CI verte avec les nouvelles règles actives et baselines commitées datées ;
`TestOpenAPISchemaDrift` strict (0 divergent).

### LOT M — Tests (gaps ciblés)

- [!] M1 — QUALITE : **DIFFÉRÉ (follow-up ciblé, 2026-07-05)**. Item le plus lourd du lot.
  `RecomputeLUSRCanonicalForPlayer` est mince (INSERT sentinelle `is_reset=TRUE` +
  délégation à `RunLUSRV2ShadowOwnerOnly`) ; **le replay délégué est DÉJÀ couvert par 30+
  `TestRunLUSRV2Shadow_*`** (skill_v2_shadow_test.go). La valeur marginale = la sentinelle
  de reset, qui exige une fixture au schéma COURANT (`is_reset` + vue `_latest` filtrant
  `WHERE NOT is_reset`, ADR 0026). **DÉCOUVERTE : le scaffolding existant `openShadowTestDB`
  est EN RETARD sur le schéma prod** (player_skill_state_v2 sans `is_reset`, vue sans le
  filtre) → un test M1 direct échouerait sur l'INSERT sentinelle. À écrire avec un setup
  is_reset-aware dédié (copie corrigée d'openShadowTestDB) — chantier ciblé, hors budget
  de cette session vu son ratio effort/valeur (replay déjà testé). Voir §7.
- [x] M2 — QUALITE : **LIVRÉ (2026-07-05)**. Les 2 jobs coverage integration de ci.yml
  (l.206 « couverture complète » + l.260 « internal/sync ») reçoivent **`-p 1`** + passage
  **`-timeout 300s`→`600s`** + commentaire NON-NÉGOCIABLE référençant l'incident 2026-07-03.
  L'échec sur code de sortie est déjà inhérent (steps `run:` GitHub Actions, pas de grep).
  Note : `-timeout` est PAR-package (chaque binaire de test) — 600s est une marge généreuse
  (duckdb ~111s, sync ~106s en local). YAML vérifié (indentation + continuations `\`).
- [x] M3 — QUALITE : **LIVRÉ (2026-07-05)**. Les 2 fonctions 0-test CIBLÉES sont couvertes :
  `medal_exploit_test.go` (8 cas table : poids par difficulté, normal/nil/index-inconnu
  ignorés, mixte) ; `weapon_data_test.go` (GetTiming known-vs-default, AVEC commentaire
  « NOTE F12 : relocatable → games/halo_infinite/film »). Renforcement : `ComputeMVPLVP`
  cas-garde (`scoreboard_extremes_test.go` : vide/1 humain/que-bots/humain+bots → extrêmes
  vides). `ComputeTrend` = **n'existe pas** (réf audit périmée). `ComputeImpactSummary` +
  `ComputeSquadPerformanceScore` gardent leur baseline (squad_test.go) — renforcement
  edge-case plus profond = follow-up basse priorité (baseline présente). Gate : package
  analysis vert ; helper `intPtr` réutilisé (perf_score_test).
- [x] M4 — QUALITE : **LIVRÉ (2026-07-05)**. `http_cache_test.go` (9 tests httptest :
  CacheMaxAge GET vs mutations, NoStore, ETagFromBytes déterministe/distinct/format,
  WriteJSONCached 200/304/stale + **NoTitleLeak MT-25** = pas de fuite de cache inter-titre)
  + `read_budget_test.go` (câblage : enveloppe le contexte via WithSwapWaitBudget + appelle
  next ; budget≤0 → défaut ; la sémantique fail-fast reste couverte par l'intégration
  sharedprovider). Mutation-check vérifié (casser le header max-age → FAIL). Gate : package
  middleware vert. Note : helper renommé `cacheBackend` (collision `okHandler` auth_test.go).
- [!] M5 (ex-F13) — DETTE §2.4.2 : **DIFFÉRÉ (follow-up ciblé, 2026-07-05)**. Vérifié sur
  pièces : les goldens sont des CAPTURES de réponses d'endpoints (`tests/fixtures/golden_values/
  *.json` : career_page_chocoboflor, match_view_slayer… = données halo_infinite). Paramétrer
  par slug exige de **GÉNÉRER des goldens H5** (mêmes endpoints pour un joueur/match halo_5
  → data H5 traversant les handlers + capture), pas seulement de restructurer. C'est une
  infra de test substantielle (comparable à M1). Structure cible confirmée (helper
  `goldenPathForSlug` + subtests `t.Run(slug)`). Voir §7. Note : `cmd/refresh_golden_fixture`
  concerne un AUTRE golden (chunk highlight events), pas ces goldens d'endpoint.

Gate M (PARTIEL) : M2+M3+M4 livrés et verts (mutation-check M4 vérifié ; CI `-p 1` posé).
M1 + M5 = follow-ups ciblés différés (infra de test lourde, ratio effort/valeur défavorable
en fin de session ; le replay LUSR de M1 est déjà couvert par 30+ tests) — voir §7.

### LOT N — Front structurel + résidus (bonus/optionnels)

> **CALIBRATION N (2026-07-04, vérifiée sur pièces)** : N1/N2 confirmés (table native
> l.323-379 ; SquadLayout l.97-729 ≈630 L god component avec double sync localStorage).
> N3 contient un FAUX POSITIF (voir item). N4 = la seule vraie DÉCISION
> PRODUIT/OPÉRATEUR restante du lot — à escalader au moment de le traiter.

- [!] N1 — CR A14 : **DIFFÉRÉ → session front (visual review)**. Vérifié sur pièces :
  `LeaderboardBlock.tsx` = 576 L, colonnes CONDITIONNELLES (isWorld/hasEnrichment),
  `LeaderboardRow` à rendu RICHE (hover/extremes/trends/rank-delta/badges), masquage
  responsive, tri manuel. Migration TanStack = refactor DÉLICAT + **Gate N exige une revue
  visuelle** non faisable à l'aveugle. Bilan N5 §6.
- [!] N2 — CR A15 : **DIFFÉRÉ → session front (visual review)**. `SquadLayout` ~630 L god
  component → 3 hooks + SquadFilterBar (défaut D-G.19 : localStorage reste local aux hooks).
  Refactor + revue visuelle. Bilan N5 §6.
- [~] N3 — CR mineurs web : (a) `[~]` FAUX POSITIF confirmé (React.lazy code-split standard,
  consomme déjà `_utils`). (b) skeleton, (c) deps listener clavier, (e) `joinAndSort` rename
  = petits fixes front, (d) MatchCard split = refactor → **DIFFÉRÉS session front** ((e) sans
  nom cible spécifié, (c) nécessite analyse deps `navigate` — mieux avec revue). Bilan N5 §6.
- [x] N4 — DETTE §2.5 : **LIVRÉ (2026-07-05)**. Politique de cycle-out documentée dans
  `internal/migration/doc.go` comme **PROPOSITION par défaut à confirmer par l'opérateur**
  (déclenchement manuel, squash par version majeure, préserver 10 derniers steps, archive
  `.ai/migrations/squashed/`, invariant schéma bit-identique). Le squash DESTRUCTIF lui-même
  reste un chantier opérateur distinct (décision non prise ici — livrable = la politique).
- [x] N5 — DETTE §2.6 : **LIVRÉ (2026-07-05)**. `.ai/V7/DETTE_ASSUMEE_2026-Q3.md` créé —
  bilan des reports PLANIFIÉS (chantier K/J5/F12, i18n I1/I2/I4, tests M1/M5, perf J1(2)/J2-J9
  measure-first, gouvernance L1-rescope/L2-345/L5, front N1/N2/N3d, + antérieurs E7/F7/F8-9/D2)
  avec condition de reprise. Les découvertes incidentes restent en §7 (backlog séparé).

Gate N (PARTIEL) : N4 (doc) + N5 (bilan) livrés ; N3(a) faux positif. N1/N2/N3(b-e) =
différés session front (le Gate « revue visuelle » leaderboard/escouade n'est pas faisable
à l'aveugle en fin de session — bilan N5 §6).

### LOT D2 — ADR 0023 Phase 5 (DIFFÉRÉ : ≥7 jours après mise en prod de D1a)

Pré-conditions : D1a en prod depuis ≥7 j ; compteur `legacy_source_used` observé à 0 (ou
sites résiduels identifiés et traités) ; K1b livré (implémentation unique de la cascade).

- [ ] D2a — Lire la télémétrie `legacy_source_used` ; lister les sources encore actives.
- [ ] D2b — Supprimer les fallbacks legacy (~28 fichiers / 65+ appels :
  `worldenrich/wiring.go:31-41`, `cmd/server/main.go:2227`, 13 CLIs,
  `platform/auth/cli_refresh.go:29-53`, `migration.go:93-103`) ; supprimer la whitelist
  `sentinel_test.go:150` ; supprimer le 2e store legacy `data/auth/watcher_tokens.json` ;
  colonnes `sync_meta` auth : arrêt des lectures (le drop physique suit la recette
  ADR 0026 au prochain rebuild).
- [ ] D2c — Mettre à jour ADR 0023 (Phase 5 livrée) + CLAUDE.md (retirer la mention des
  sources legacy tolérées).

Gate D2 : `go build ./... && go test ./...` ; grep `SPNKR_OAUTH_REFRESH_TOKEN` +
`oauth_refresh_token` (lectures) → 0 hors migration one-shot ; sync live OK pour les 6
joueurs en local/prod (RT jamais re-capturés — règle projet).

---

## 5. Matrice de couverture (audit → lots) — AUCUNE ligne ne doit rester orpheline

| Source | Section/IDs | Lot(s) |
|---|---|---|
| ARCHI | Majeurs 1-7 (racine api/) | K1a, B6, K1c, K1d, K1b, K1e, F16→H8 |
| ARCHI | Majeurs 8-12 (service/) | K1i, K1j, K1k, K1l, K1m |
| ARCHI | Majeurs 13-14 (analysis/chemins) | F12→K3, K1l |
| ARCHI | Majeurs 15-16 (ART) | E1, E2 |
| ARCHI | Majeurs 17-25 (structure) | K3a-e, K2a-d |
| ARCHI | Majeurs 26-27 (contrat) | G1, L1 |
| ARCHI | Majeurs 28-38 (title-agnosticism) | F1-F11 |
| ARCHI | Majeurs 39-45 (_latest + notifications) | B1-B9 |
| ARCHI | Majeurs 46-47 (N+1 HTTP) | J3, J4 |
| ARCHI | Majeurs 48-50 (connexions) | J1, J2, D1c |
| ARCHI | Mineurs couches handlers/api (6) | K1h, K1a (BestKDA), K1g |
| ARCHI | Mineurs couches service/analysis/domain (14) | K1n, F15 (domain) |
| ARCHI | Mineurs SQL hors platform (7) | D1b (writes.go), G5, E5, E6, E8, E3, E7 |
| ARCHI | Mineurs structure (~15) | K3f, G7 (ServiceConfigIDFor via registre) |
| ARCHI | Mineurs title-agnosticism (~20) | F15, K1l (chemins), G9 |
| ARCHI | Mineurs performance (~15) | J6-J9, B7, G15, G8 |
| ARCHI | Mineurs contrat/gouvernance (4) | L4, G6, K1g (boot H5), L7 |
| ARCHI | TOP 10 + recos techniques | répartis (L2 ratchets, F, J, K, G) — vérifier en fin de plan |
| DETTE | TOP 1-10 | C1, A4/D1c, D1b, D1a/D2, C2, F12→K3, C7, C4, C5, C8 |
| DETTE | §1.3 READMEs / §1.4 invariants | C6, C5 |
| DETTE | §2.1 guards / §2.2 Phase 5 / §2.3 Prestige | D1b-f, D2, C7 |
| DETTE | §2.4 multi-titre (film, goldens, template) | F12→K3, F13→M5, F14/E8 |
| DETTE | §2.5 schéma DB (colonnes, migrations, allowlist) | G14, N4, E4 |
| DETTE | §2.6 dette assumée / recos 1-9 + gouvernance | N5, C3, C8, D1f |
| QUALITE | B1-B2 + cause racine | S1, S2, S3 |
| QUALITE | M2-M4 + mineurs accès | S4-S8 |
| QUALITE | Tokens/secrets (probe, CLIs) | S5, S9, D2 (dette Phase 5) |
| QUALITE | XSS frontend | A5 |
| QUALITE | Robustesse #3/#6/#8/#10 + autres | B10-B16 |
| QUALITE | Axe 4 tests (gaps) | M1-M5 |
| CR | C1-C4 | A1-A4 |
| CR | A1-A6 (archi Go) | K2a, K1g, K1f, K2b, K1h/K1i, D1e |
| CR | A7-A9 (code mort) | G2-G5, G10-G13 |
| CR | A10-A13 (duplication) | H1-H8 |
| CR | A14-A16 (React) | N1, N2, L5 |
| CR | A17-A19 (i18n) | I1-I3 |
| CR | A20 (gouvernance lint) | L3 |
| CR | Mineurs §5 (condensés) | K2e, K1a, K3f (débris), F15 (comeback), N3, I4, D1d (guards) |
| CR | Recos 1-5 | H (garde-rails), G16, L6, I5/L5, N5 |

Vérification finale (dernière action du plan, après N) : relire les 4 audits en diagonale,
confirmer que chaque finding a un statut dans ce fichier, compléter la matrice si un item
a été oublié, puis produire le bilan final à l'utilisateur.

---

## 6. Journal d'exécution (à remplir par l'exécutant, OBLIGATOIRE à chaque clôture de lot)

Format par entrée :
```
[YYYY-MM-DD] LOT X — CLOS
- Commits : <hashes>
- Items [x] : n / [~] : n / [!] : n
- Justifications [!] : ...
- Gate : <commandes passées + résultat>
- Mesures/notes : ...
```

```
[2026-07-02] LOT A — CLOS (branche refactor/audits-2026-07 ; commit à suivre)
- NOTE BRANCHES : LOT S est livré sur sa branche dédiée fix/security-unauth-endpoints
  (plan + journal S marqués là-bas, commit 0c5982111). Cette branche part de main → le
  plan y est vierge côté S ; RÉCONCILIER le journal/les cases au merge des deux branches.
- Items [x] : 5 (A1-A5) / [~] : 0 / [!] : 0.
- Gate A : go build ./... OK ; go test ./internal/sync/... ./internal/api/... ./cmd/levelup/... OK ;
  go test -tags=integration ./internal/sync/... ./internal/persist/... OK (anti-ART, A3 touche sync/) ;
  grep RunBackfillLUSR( → 0 ; cd apps/web : typecheck OK, lint OK (0 err, 70 warn baseline),
  vitest OK (2070 passed, +2 = escapeHtml fonctionnel + garde-rail).
- A1 : perfTier local inversé supprimé → perfScale canonique ; import `type SemanticToken` retiré (inutilisé).
- A2 : badge outcome piloté par outcomeKey(outcome_code). Badge (pill) CONSERVÉ (préférence UI « pills
  pleines ») plutôt que le span coloré du modèle Explorer — décision consignée. Date via formatDate
  locale-dynamique. Le type front CareerTopMatch n'exposait pas outcome_code → ajouté au schéma openapi.yaml
  + `make generate-types` (TopMatchDTO Go le peuple déjà au runtime).
- A3 : chemin LUSR v1 supprimé. RunBackfillLUSR (v1→batchComputeLUSR) renommé RecomputeLUSRCanonical
  (reroute v2 RecomputeLUSRCanonicalForPlayer, param force retiré) — le scaffolding lease+OpenPlayerDB+
  acquireSharedWriter est conservé, donc rename+reroute plutôt que suppression brute. upsertLUSRRatingsLegacy
  (dead code, 0 caller vérifié) supprimé + import slog orphelin retiré. 3 callers adaptés. RunBackfillLUSRDryRun
  (read-only, ne WRITE pas) CONSERVÉ (hors scope C3). ForceLUSR reste câblé (flag/scope/tests) mais n'est plus
  consommé par le backfill (replay v2 toujours complet). batchComputeLUSR CONSERVÉ (post-sync + fallback).
- A4 : 5 docs corrigées (engine_options, engine, engine_batch_path, sync/v2/doc, cmd/server/main :1108) —
  PERSIST_BATCH défaut ON (=0 kill-switch ART-unsafe), SYNC_PIPELINE défaut V2 (=v1 kill-switch). Retrait des
  flags eux-mêmes = lots D1b/D1c.
- A5 : escapeHtml promu dans components/charts/_utils.ts (+ échappement apostrophe absent de la version locale) ;
  BarStackedChart refactoré ; garde-rail escapeHtml.test.ts (interdit toute redéfinition locale + test
  fonctionnel) ; ~30 sites de formatters tooltip enveloppés (sweep 4 agents), dont les 8 à contenu tiers
  (gamertags / cartes UGC) confirmés ; MatchWeaponCharts string-template {b} converti en formatter fonction.
  SessionPlacementBreakdown skippé (label = '#'+entier, sûr).
```

```
[2026-07-02] LOT B — CLOS (branche refactor/audits-2026-07 ; 2 commits)
- Items [x] : 16 (B1-B16) / [~] : 0 / [!] : 0. (B7-squad : MAX winProb IS NOT NULL laissé tel
  quel — stale-safe, documenté + allowlisté ; pas un site ADR-0026.)
- Gate B : go build ./... OK ; go test ./... OK (suite complète) ; garde-rail B8 vert
  (no_raw_rating_reads_test) ; garde-rail B15 vert (no_capability_error_dup_test) ;
  golangci --new-from-rev = 0 issue ; seed patterns_repo_db_test adapté (vue _latest).
- B1-B9 (famille _latest) : lectures rating (match_skill_rank/match_csrs/player_csr_snapshots)
  → vues _latest ; Q26g gardé RAW (filtre H5 CSR=0 non réplicable par la vue) + tie-break
  written_at déterministe ; B2 retire le workaround winProb mort ; B6 snapshot + tiebreak ;
  B9 fan-out OpenReadForQuery.
- B10-B15 (robustesse) : RT roté log+retry (B10) ; engagement skip si history en erreur, pas de
  score faux (B11) ; family resolver log (B12) ; data_health sondes → ProbeErrors + cycle WARN
  (B13) ; ValidateLUSRChainClassifierWired fail-fast boot (B14) ; MapCapabilityError central +
  2 sites migrés + garde-rail (B15).
- B16 (dette logging) : sweep 3 agents — slog.Error→ErrorContext (où ctx dispo), err.Error()→err,
  best-effort journalisés (career.go SetSyncMeta, backfill_weapons MarkWeaponKillsDone, catalog.go
  only_played invalide). Sites sans ctx laissés (documentés).
- 2 commits : 0077142bb (partie 1 : B1-B12/B14 + garde-rail B8) + le commit de clôture (B13/B15/B16).
- RÉCONCILIER plan/journal S+A+B au merge (S sur sa branche ; A+B sur refactor/audits-2026-07).

[CORRECTION 2026-07-03 — gate B incomplet à la clôture] Le "Gate B" ci-dessus n'a fait
tourner que `go test ./...` (NON-intégration) ; le `-tags=integration ./...` était pourtant
OBLIGATOIRE (B touche sync/ en B10-B14 et les lecteurs platform/duckdb en B8). Découvert au
gate de LOT C : 20 tests d'intégration `internal/platform/duckdb` étaient ROUGES — fixtures
périmées par la migration B8 vers les vues _latest (vues `match_skill_rank_latest` /
`match_csrs_latest` absentes, colonnes append-only `written_at`/`id` manquantes) — plus 1
build break `internal/service` pré-campagne (collision `stubResolver`). Le rouge avait été
masqué par (1) le flake concurrent DuckDB mono-process et (2) un filtre `Select-String "FAIL"`
attrapant les logs « Failure while replaying WAL ». RÉPARÉ 2026-07-03 (commit fix dédié) :
fixtures alignées sur le schéma prod + `stubResolver`→`stubCatalogResolver` ; suite
d'intégration complète VERTE en run sérialisé `-p 1` (exit 0). Le code livrable de B était
correct — seule sa vérification était incomplète. Garde-fou process ajouté au skill
delivery-checklist (`-p 1` obligatoire + filtre ancré `^--- FAIL:`).
```

```
[2026-07-03] LOT C — CLOS (branche refactor/audits-2026-07)
- Items [x] : 8 (C1-C8) / [~] : 0 / [!] : 0.
- Statuts détaillés :
  - C1 [x] : CLAUDE.md (pré-écrit 07-02) vérifié frais + 2 corrections mineures (formulation
    de la purge Python ligne 19 sans littéraux `src/`//`.venv` ; chemin `generated.ts` ligne
    164 qualifié `apps/web/`). Gate grep : 0 token Python-mort sans ambiguïté ; 3 hits bruts
    résiduels LÉGITIMES (2× `apps/web/src` frontend, 1× la règle 2 qui INTERDIT pandas/polars).
  - C2 [x] : `.ai/project_map.md` bandeau « HISTORIQUE — GELÉ, NE FAIT PLUS FOI ».
  - C3 [x] : règle de rotation trimestrielle écrite dans CLAUDE.md (archive
    `.ai/archive/thought_log_<AAAA>-Q<N>.md`). Archivage effectif = NO-OP : le journal actif
    ne contient que 2026-05/06/07 = Q2+Q3 (fenêtre courant+précédent), rien avant Q2 à sortir.
  - C4 [x] : 2 pointeurs `0014`→`0016` (`sharedprovider/doc.go`, `baseline_red_integration_test.go`)
    + statut « commit 2/9 »→« livré ».
  - C5 [x] : 4 invariants écrits (INSERT-only `SharedPersister.Persist` ; pas de write-lease
    shared en phase 6 du runner post-sync V2 ; jamais `sql.Open` direct sur `provider.go`
    mono-process ; recette 3-étapes ADR 0019 depuis `enrichmentFields()`/player_persister).
  - C6 [x] : doc.go créés — sync, migration, games, progression (README), domain, api/handlers ;
    `temporal/README.md` complété (engagement). `halo_5/doc.go` déjà présent (33a288783).
  - C7 [x] : Prestige unifié — `prestige.IsEnabled(settingsPath)` source unique
    (app_settings.json + override env), `loadPrestigeEnabled()` supprimé, hook post-sync et
    surfaces HTTP lisent la même source. ADR 0005 → Accepted + clause d'expiration annulée.
    `prestige_expiry_test.go` supprimé. Pas de cycle d'import (config→prestige, vérifié).
  - C8 [x] : politique docs/FR dans CLAUDE.md règle 15 (ADRs+runbooks EN-only, 4 guides
    bilingues) ; liens `docs/FR/ARCHITECTURE_V6.md` validés (3 cibles existent) ; hook
    lefthook `docs-fr-sync` (warning non bloquant). SOUS-ITEMS audit revus sur pièces :
    « rattraper CITATIONS.md EN » = SANS OBJET — CITATIONS.md (EN et FR) sont des STUBS de
    redirection vers la source unique `docs/COMMENDATIONS.md` (122 L, à jour) ; la prémisse
    « FR 4 mois d'avance » lisait les dates git des stubs. `RUNBOOK_OPS_DUCKDB.md` n'existe pas.
- Gate C : grep CLAUDE.md tokens Python-morts → 0 (3 hits légitimes documentés) ; liens
  docs/FR valides ; `go build ./...` OK ; `go test ./...` OK ; `go vet ./...` OK ;
  `go test -tags=integration -p 1 ./...` = exit 0 (suite complète VERTE, sérialisée).
- NOTE : le gate d'intégration a révélé que celui des LOTS A/B était masqué (voir §7 +
  correction LOT B ci-dessus). Réparation dans un commit fix dédié (07ee3546d) séparé du
  commit de clôture C.
- Commits : 07ee3546d (fix gate B) + <hash clôture C>.
- RÉCONCILIER plan/journal S+A+B+C au merge (S sur sa branche ; A+B+C sur refactor/audits-2026-07).

### LOT D1 — Flags & guards (cycle de vie) — CLOS 2026-07-03

- D1a [x] : télémétrie `legacy_source_used` (`internal/observability/legacy_source.go` +
  `RecordLegacySourceUsed` + 4 constantes) instrumentée sur 6 sites legacy auth. Pré-requis D2.
  Commit 9b2d07870.
- D1b [x] : suppression COMPLÈTE de `LEVELUP_PERSIST_BATCH` + chemin legacy `insertFetchedMatch`
  (batch INSERT-only = unique voie). −1249 L. A révélé le piège `batchMode=false` (défaut
  silencieux des tests) — 2 lacunes de SETUP corrigées (contract_v1 nil-provider, 4 E2E provider
  via patchSharedSchemaForBatch). Commit d4343dce4. Voir §7.
- D1c [x] : suppression pipeline V1 (flag `LEVELUP_SYNC_PIPELINE` + fallback auto). AUDIT
  H5-sous-V2 (3 agents) → BUG PRÉ-EXISTANT corrigé : V2 mono-titre ne routait pas H5 →
  `RunOnceTrigger` partitionne par `livesync.HandlesTitle`. `syncPlayer`/`engine.run` CONSERVÉS
  (live-only + filet boot ; engine.run PARTAGÉ, K2b reste valide). Commits b30eb9fe5 + 52a8920c3.
  Voir §7. ATTENTION : kill-switch rollback V1 retiré sur branche — gate live-sync manuel avant
  land main.
- D1d [x] : cycle de vie documenté des 4 flags restants (modèle `shared_reader_legacy.go`). 2
  vrais kill-switches (BATCH_ASYNC, EVENTS_CONVERGENCE) → triplet complet (retrait >= 2026-Q4 +
  critère mesurable) ; MULTI_TITLE = gate de rollout ; CONTRACT_VALIDATE = diagnostic dev/CI
  permanent. `docs/CONFIGURATION.md` (+FR) : défaut `(off)`→`on` corrigé pour BATCH_ASYNC + 4
  lignes de flags. Commit 268d600f2. Tension règle 11 (MULTI_TITLE) en §7.
- D1e [x] : centralisation des lectures `os.Getenv` DIVERGENTES (CR A6). Suppression de 2
  fonctions mortes+divergentes (`handlers.MultiTitleAPIEnabled`, `notify.EnvWebhookURL`) ;
  `config.DiscordWebhookURLFromEnv()` = précédence env unique (notify/validation cessent de
  bypasser `LEVELUP_DISCORD_WEBHOOK_URL`) ; kill-switches scheduler (`PersistBatchAsync`/
  `EventsConvergence`/`EventsConvergenceMax`) dans AppConfig ; garde-rail
  `env_centralization_test.go`. Baseline prod `os.Getenv` hors config 34→29 (résidu justifié §7).
  Commit ec5335afd.
- D1f [x] : lint `internal/archlint/todo_expiry_test.go` (`TODO(expiry:YYYY-MM-DD)`, `now`
  injectable, scanne go-api). Validé dans les 2 sens (vert 2026-07-03, rouge à now=2026-09-01).
  1 caduc supprimé (`persist/worker.go` marqueurs Phase 1.5+). Résidu TODO + BUG latent
  `WithPrestigeHook` en §7. Commit <hash clôture D1f>.
- Gate D1 : `go build ./...` OK ; `go test ./...` OK ; `go test -tags=integration -p 1 ./...`
  = exit 0 (D1e, sérialisé) ; grep `LEVELUP_PERSIST_BATCH` (flag exact, hors _ASYNC) → 0 ;
  os.Getenv baseline notée (34→29, reads `LEVELUP_PERSIST_BATCH_ASYNC` → 0).
- RÉCONCILIER plan/journal S+A+B+C+D1 au merge. NE PAS merger main sans feu vert + gate
  live-sync manuel (retrait rollback V1).

### LOT E — ART résiduel & écritures à risque — CLOS 2026-07-03 (E7 différé)

- Cartographie read-only préalable : 8 agents Explore (1/item), 562k tokens — a mappé
  chaque item sur pièces avant implémentation (dépendances croisées + gotchas Haiku à
  vérifier).
- E1 [x] : import OpenSpartan `writeOneMatch` → `persist.SharedPersister` (1 tx INSERT-only
  atomique, remplace 4 `sync.Insert*/Upsert*` dont ON CONFLICT). Converter CSR exporté.
  Commit 0a27412f7 (+ fixture E2E api/handlers 9c211a6f3). Découverte §7 : helpers sync.Insert*
  orphelins prod → LOT G.
- E2 [x] : `backfillPairNamesByConstruction` bulk UPDATE nu → row-by-row par match_id +
  garde-fou tripwire « bare-bulk » (littéral SQL `UPDATE <critique>` sans `?` = violation,
  ancrage backtick) + sanity test. Commit 7262df3e0. Découverte §7 : backfillOneColumn per-asset-id.
- E3 [x] : exclusion `ops/` retirée du tripwire (3 tests + 3 commentaires). BUG ART RÉEL
  découvert+corrigé : `lying_bits_reset` faisait 3 bulk UPDATE in-process sur match_registry
  (« pas de risque ART » — faux) → row-by-row. Vérif empirique : seule exposition. Commit cdd1e970d.
- E4 [x] : `TestAllowlistJustifiesEverything` warning→erreur + cohérent avec le scan
  (stripGoComments — une justif commentaire-only ne compte plus) ; entrée `persist/doc.go`
  retirée (morte). Commit 58c5542dd.
- E5 [x/~] : filtre temporel canonique dans `progression/profile/queries.go` (5 sites
  start_time bruts) via helper exporté `duckdb.StartTimeCanonicalSQL(alias)`. Déplacement des
  littéraux SQL → repo platform/duckdb = [~]→H1 (orchestration cross-DB, connexion déjà via
  SharedReader). Commit 461532340.
- E6 [x] : bare RO connects (`worldenrich/wiring.go` sql.Open RO, `auth/pool/discovery.go`
  OpenReadOnly) → `OpenReadForQuery`. Gotcha résolu (hallucination carto) via variantes
  `Read*FromSQL(*sql.DB)` + helper `readSyncMetaValue`. Commit deb6f8e98.
- E7 [!] : DDL bootstrap `sync/schema.go` → migration. DIFFÉRÉ (règle 9) : item MAL LABELLISÉ
  « mineur » — en réalité refactor profond du boot/provisioning de TOUTES les DBs, DDL
  dupliqué-mais-aligné avec `create_base_*_schema` en transition b23/b25 (title-ownership),
  logique de vues au boot corrigeant des bugs prod documentés (attach RO/RW, xuid bruts).
  À faire en chantier dédié APRÈS stabilisation b23/b25. Signalé utilisateur.
- E8 [x/~] : écritures per-match H5 CSR/SR → `PlayerPersister.PersistPerMatchRating` (persister
  DÉDIÉ, nouveau). Blocage résolu : `Persist()` exige l'ancre enrichment + skip si présent →
  inutilisable post-score. ADD_TITLE.md = [~]→F14. Commit e84853e70.
- Gate E : `go test -tags=integration -p 1 ./...` = exit 0 (105 packages, suite complète VERTE
  sérialisée) ; tripwire étendu (bulk-from-values + bare-bulk) vert ; allowlist ART réduite
  (1 entrée) et BLOQUANTE ; ops/ scanné.
- RÉCONCILIER plan/journal E au merge. E7 reste [!] à planifier.
```

### LOT G — Purge du code mort — CLOS 2026-07-03

```
- Méthode : chaque suppression VÉRIFIÉE sur pièces (la cartographie Haiku antérieure s'est
  révélée peu fiable) — 0 caller/reader confirmé par grep avant chaque delete.
- G1 [x] : gen/types.gen.go + oapi-codegen retirés (Makefile, golangci, coverage, docs, ratchets).
- G2 [x] : analysis/home_* — fonctions mortes supprimées chirurgicalement (BuildKPIs/Trend/
  HeroCard/Highlights/SessionSummaries/RecentMatches*) ; helpers vivants + BuildSpartanIdentity gardés.
- G3 [x] : feature session-compare supprimée MAIS infra partagée préservée (DEC-1, correction
  de plan : les types domain + builders sont partagés avec session-detail vivant). Openapi + front purgés.
- G4 [x] : SquadV2RouteHost + SquadV2Page orphelins supprimés. Commit(s) G1-G4.
- G5 [x] : feature « notif Discord nouveaux médias » morte — PÉRIMÈTRE ÉLARGI (signalé) : câblée
  end-to-end jusqu'à un toggle réglages user-facing mais SANS déclencheur (0 caller). Suppression
  COMPLÈTE full-stack (backend notify+settings+migration + front toggle/i18n/openapi/fixtures) pour
  ne pas laisser un toggle no-op (règle 11). Découverte §7 : colonne bool `discord_notified` orpheline.
  Commit 25f9c3581. Gate intégration -p 1 vert.
- G6/G7 [x] : ReassociateMedia (prod-orphelin, route retirée 2026-04-29) + ServiceConfigIDFor
  (stub mort) supprimés. Commit 5d14fa19f.
- G8 [x] : constantes SQL mortes Q4/Q4MV monolithiques + Q26eHomeSkillPeakByType (remplacée par
  split Phase A/B) supprimées ; doc inversée corrigée ; Q24 = doc fausse (param inexistant) corrigée,
  const vivante. G9 [x] : 8 entrées [assets.mode.*] H5 placeholder (slugs divergents, 0 lecteur)
  supprimées. Commit 9c6c2a9cc.
- G10 [~] (déjà fait A3) · G13 [~] (déjà fait D1b) · G16 [~] (pré-exécuté, section 0 delivery-checklist).
- G11 [x] : SessionKDATimeline + SessionOcdrScatter orphelins supprimés (notés backlog UI mémoire).
  G12 [x] : Map{Mu,TierSub}ToLegacyRating dé-exportés (test-only). Commit(s) G11/G12.
- G14 [x] : known_teammates_count + friends_xuids (PME) retirées de la vue _latest + ensurePMEColumns
  + CREATE TABLE + doc (0 writer/lecture ; DROP physique au prochain rebuild, DEC-6). G15 [x] :
  mv_map_stats rebuild par-sync supprimé (0 lecteur Go) + nettoyage self-healing deprecatedPlayerAggregates.
  Commit a4fb7bcad. Gate intégration -p 1 vert (233 lignes, 0 FAIL ancré).
- Gate G : go build+vet OK ; suites unitaires par package vertes ; front typecheck+build+vitest verts ;
  intégration -p 1 exit 0 (G5 + G14/G15) ; grep de chaque symbole supprimé → 0.
- RÉCONCILIER plan/journal G au merge. Suivi LOT F ensuite (title-agnosticism).
```

### LOT F — Title-agnosticism — CLOS 2026-07-04

```
- Méthode : investigation on-pièces des 15 items (workflow 16 agents) puis exécution linéaire,
  chaque seam vérifié + gaté. Pattern récurrent : injection de seam au wiring (racine DI autorisée
  à importer games) pour découpler platform/service de games/halo_infinite.
- LIVRÉS [x] : F1 (media_repo → seam `analysis.ModeTaxonomy`), F2 (CSR Explorer/Compare gated
  capability → 0 fuite H5), F3 (URL Waypoint → 2 méthodes `TitleAssetURLAdapter`, lien mort H5
  supprimé), F4 (labels outcome → outcomes.toml + unif « Abandon » cross-titre), F5 (anglicismes +
  corruption UTF-8 timeseries ; routage FieldMappingSet → K1), F6 (H5 fields.toml 5→52, sous-ensemble
  par capability, transform Infinite-moins-PvE), F10 (ratchet anti-slug élargi, FERME un feature-gate
  `TitleSlug(ctx)=="halo_infinite"` ; 11 sites parité grandfathered), F11 (WARN builtin toml),
  F14 (doc Collect→Persist EN+FR), F15 (17 puces : 8 [x] + 9 [~]).
- DÉCISIONS UTILISATEUR (2026-07-04) : F6 = sous-ensemble par capability (pas strict-59) ;
  F7 = réconciliation SEULE (test miroir souple) — l'ACTIVATION engagement H5 est un chantier futur
  (canonicalisation + calibration, impacte Halo 7), HORS audit → mémoire dédiée.
- DIFFÉRÉS [~] justifiés : F7 (activation = chantier futur), F8 (auth ADR 0023, H5 réutilise audiences
  Infinite → défaut fonctionnel, per-titre = MT-02/Phase 1b), F9 (Ascension épinglé DefaultSlug, pas de
  cap Ascension + H5 a CapCareer → Phase 1b), F12 (extraction package film 18 fichiers = structurel → LOT K),
  F13 (goldens par slug = infra test → LOT M). Garde-rails génériques F15-12/14 + parité fields F6 → LOT L2.
- Gate F : go build+vet OK ; suites unitaires par package + front verts ; intégration -p 1 exit 0
  (F1/F15-2, 233 pkgs). ~14 commits (F1 9e602c638 … F6 ecdb0b4e0).
- RÉCONCILIER plan/journal F au merge. Suivi LOTS H→N.
```

## 7. Découvertes hors périmètre (à remplir — NE PAS traiter sans accord)

- [FOLLOW-UPS L — L2-(3/4/5) parités capability + L5 queryKey + L1 re-scope] Différés
  2026-07-05. **L2-(4)** cap⟺scalaire : invariant RÉEL et confirmé (registry.go documente
  « CapDamageTaken déclarée SSI ProvidesDamageTaken(slug) », idem TeamMMR) ; test de parité
  buildable mais exige de charger DEUX sous-systèmes de config en test (title.Registry via
  LoadTitlesIntoRegistry + games.NewMappingsEndpointResolver depuis mappings.Registry) →
  intégration modérée. **L2-(3)** coarse↔fine + **L2-(5)** fields.toml par capability-group :
  même besoin de chargement config. **L5** : 180 queryKey (gros front). **L1** : re-scope
  (catégoriser dérive-réelle vs enrichissement-voulu, cf. plus haut). Tous ont API/invariant
  confirmés ; à traiter en tâches ciblées (config-integration tests / front).
- [DETTE lint pré-existante branche — révélée par L3] `--new-from-rev=main` (config L3)
  remonte 2 issues PRÉ-EXISTANTES (hors périmètre L3, config goconst/gocyclo inchangée par
  L3) : (a) `match_history_service.go:107` goconst — `"loss"` ×4, à remplacer par la const
  existante `duelLabelLoss` ; (b) `halo_ranks_loader.go:55` gocyclo `LoadRankCatalog`
  complexité 16>15 (introduite par F15-3) — extraire une branche. Non bloquantes pour le
  commit L3 (only-new-issues compare par push ; ces fichiers ne sont pas dans le push L3).
  À nettoyer avant le merge final vers main (ou vérifier qu'only-new-issues les a déjà
  acceptées à F4/F15). Règle 7 : non traitées ici.
- [FOLLOW-UPS tests lourds — M1 + M5] Différés (2026-07-05, fin de session). **M1** : test
  intégration `RecomputeLUSRCanonicalForPlayer` — le replay délégué est déjà couvert (30+
  `TestRunLUSRV2Shadow_*`) ; la sentinelle `is_reset` exige une fixture au schéma courant, et
  DÉCOUVERTE le scaffolding `openShadowTestDB` est en retard (pas d'`is_reset`) → setup dédié à
  écrire. **M5** : goldens par slug — exige de GÉNÉRER des captures d'endpoints H5 (infra
  substantielle), pas juste `goldenPathForSlug` + `t.Run(slug)`. Les deux ont un ratio
  effort/valeur défavorable en fin de session ; à planifier comme tâches ciblées.
- [CHANTIER i18n manuel différé — I1/I2/I4] Décision utilisateur A (2026-07-05). La règle
  lint `no-hardcoded-strings` (I5, désormais `error`) est volontairement ciblée : texte JSX
  ≥3 mots/≥15 car + attributs title/aria-*/placeholder/alt. Elle NE flague PAS : (a) les args
  de fonction (`setError('Impossible…')`), (b) les libellés JSX courts (« Connexion Xbox »),
  (c) les ternaires `locale === 'en' ? … : …`. Reste donc un vrai chantier de couverture
  bilingue (CLAUDE.md n°1), NON exigé par le gate : **I1** (onboarding : XboxLoginPage,
  StepDeviceCode, StepInitialSync, RegisterPage, OpenSpartanImportCard) ; **I2** (scoreboard
  MatchScoreboard, heatmap DOW/HOUR, `toLocaleString('fr-FR')` figés, « Par carte/mode/
  Analyser ») ; **I4** (~88 ternaires `locale===`). Système cible = manifests TOML
  `src/lib/i18n/manifests/*.toml` + `node scripts/build_i18n_manifests.mjs` → `generated/*.ts`,
  consommés via `formatMessage(manifest, key, locale)`. À planifier comme chantier dédié.
- [LOT H / H3 — Étape 2 + matching FR-label] Deux follow-ups au-delà du dédup 13 L :
  (1) Faire consommer `useLocalFilterBar` par `SynthesisPage` (supprimerait ~200 L d'état
  pending/committed dupliqué) — conditionné à vérifier que le hook couvre les besoins
  synthesis-spécifiques (rôles, armes, accuracy). (2) `experienceCounts`
  (`useLocalFilterBar.tsx`) classe les options via substring FR `'classé'`/`'non classé'`
  — couplage fragile aux libellés ; à dériver d'un champ canonique d'expérience côté
  contrat backend. Aucun des deux n'est un dédup → hors périmètre H3 (règle 7).
- [LOT H / H2 — bug latent filtre bot par gamertag] `cmd/diag_recent_match_sync/main.go:333`
  et `migrations/steps_shared_core.go:387` filtrent `gamertag [NOT] LIKE 'bid(%'`. Or les bots
  ont un xuid `bid(N.0)` mais un GAMERTAG "343 Meowlnir/Ellis/…" (cf. `analysis/identity.go`
  botDisplayNames) — donc `gamertag LIKE 'bid(%'` ne matche JAMAIS un bot réel. Le prédicat
  bot correct est sur `xuid`. Ces 2 sites sont soit un no-op (diag : compte "vrais gamertags"
  — l'exclusion bot y est redondante avec `gamertag != xuid`), soit un filtre inopérant.
  Non traité en H2 (règle 7 ; le site diag a été laissé littéral, migrations = gelé). À
  clarifier : supprimer le prédicat gamertag inopérant ou le corriger en xuid.
- [LOT E / E2 — bulk résiduel per-asset-id] `backfillOneColumn` (`backfill_registry_names.go:177`)
  fait `UPDATE match_registry SET <name> = ? WHERE <id_col> = ? AND <name> = ?` — multi-row
  par asset_id (tous les matchs d'une même map), donc bulk-ish, MAIS lie des `?` (pas « nu »).
  Hors cible nommée de E2 (« l'UPDATE bulk multi-row NU », ligne 98) et non attrapé par le
  garde-fou bare-bulk (il a des placeholders). Résiduel ART-adjacent : à convertir en
  row-by-row par match_id dans un follow-up si on veut match_registry 100% row-serialized.
  Non traité (règle 7).
- [LOT E / E1 — code mort transitif] Après E1, `sync.InsertRegistryIfNotExists`,
  `sync.InsertParticipants`, `sync.InsertMedals` n'ont PLUS aucun caller prod (openspartan
  était le dernier) — seulement des tests (dont `concurrent_upsert_*_test.go` qui caractérisent
  le comportement ART de l'ancien UPSERT). Dead-code-museum (diagnostic #1). Candidat suppression
  LOT G (retirer fn + tests + entrée allowlist `writes.go` si le pattern ON CONFLICT disparaît
  avec). NON traité en E1 (règle 7). `sync.UpsertSharedCSRs` reste vivant (backfill CSR
  `csr_shared_backfill.go:149`).
- [LOT A / A2] Dette de type front pré-existante : `CareerTopMatchesResponse` hand-written
  (`types.ts:516`) = `{ items: CareerTopMatch[] }` ne correspond PAS à la réponse réelle du
  backend `{ best_matches, worst_matches: TopMatchDTO[] }` ; et `data.top_matches_preview`
  (lu par CareerPage.tsx) n'existe pas dans le `CareerPageResponse` Go. Conséquence :
  `fullTopMatches.items` / `data.top_matches_preview` sont `undefined` au runtime sur ces
  chemins. Hors scope A2 (le fix badge/date fonctionne car getOutcomeColor dégrade
  proprement sur outcome_code absent). À traiter : aligner les types front sur le contrat
  réel (openapi) + corriger le flux de données CareerPage.
- [LOT A / A5] MatchWeaponCharts.tsx : string-template ECharts `{b}` converti en formatter
  fonction avec escapeHtml — vérifier visuellement que l'affichage (nom d'arme + valeur +
  pourcentage) est préservé lors d'une revue UI.
- [LOT B / B7] Lectures rating brutes VOLONTAIRES à sémantique nuancée (allowlistées dans
  `no_raw_rating_reads_test`) : `queries_career_encounters.go` Q24 (LUSR enemy strength) et
  `queries_home_citations.go` Q26f (effective_type CSR/LUSR) — une migration `_latest`
  (priorité CSR>LUSR) changerait la valeur LUSR sur les matchs ranked. À trancher (décision
  produit LUSR vs CSR) avant toute migration.
- [LOT C / gate — TRAITÉ] Le gate `-tags=integration` des LOTS A/B n'était pas réellement
  vert (masqué par le flake concurrent DuckDB + un filtre `Select-String "FAIL"` attrapant
  « Failure » de logs). 20 tests `platform/duckdb` rouges (fixtures sans vues `_latest` /
  colonnes append-only, régression B8) + 1 collision `stubResolver` service (pré-campagne
  Phase F). RÉPARÉ dans le commit fix 07ee3546d. Prémisse process : lancer les tests
  d'intégration DuckDB avec `-p 1` et filtrer les FAIL avec l'ancre `^--- FAIL:` (consigné
  au skill delivery-checklist).
- [LOT C → LOT M] Aucun gate CI n'exécute la suite d'intégration aujourd'hui (le pre-push
  a retiré les tests Go). C'est la cause racine de la non-détection de la casse B. ACTION
  planifiée en LOT M (Tests) : câbler un job CI `go test -tags=integration -p 1 ./...`
  (sérialisé) — décision utilisateur 2026-07-03.
- [pré-campagne — TRAITÉ (fixture)] `TestGetOrOpen_RunsPlayerMigrationsForLegacySchema` :
  `seedSharedDBForPoolTest.match_registry` sans `game_variant_id`/`game_variant_name`
  (lus par Q5SharedHistory depuis f7c7885b69, 2026-07-01) — colonnes ajoutées au fixture
  (commit 07ee3546d). Pas de défaut de prod (schéma prod expose ces colonnes).
- [LOT D1 / D1b — TRAITÉ] `batchMode=false` était le DÉFAUT SILENCIEUX des tests : tout test
  basé sur run()/RunDelta (y compris la suite E2E provider concurrency) utilisait le chemin
  legacy `insertFetchedMatch` (écriture inline), JAMAIS le chemin batch. Supprimer `batchMode`
  (batch INSERT-only = unique voie) a rerouté ces tests vers `submitMatchAsBatch` →
  SharedPersister, révélant 2 lacunes de SETUP de test (PAS des bugs prod — le batch est le
  défaut prod, correct) : (1) `contract_v1_test.go` à provider nil bloquait sur le writer-lease
  (corrigé : le contrat ne teste que le format d'URL `xuid(NNN)`, vérifiable sans persist) ;
  (2) les 4 E2E provider (`newE2EEnv`/`newMultiUserEnv`) créaient le shared via
  `EnsureSharedSchema` statique, sans les colonnes écrites par le SharedPersister
  (`match_intensity`, `backfill_bits`, mécaniques H5 — ajoutées par migrations title-owned) →
  corrigé via `patchSharedSchemaForBatch` dans les 2 env. LEÇON : un flag à défaut-false peut
  masquer un chemin de code entier en test.
- [LOT D1 / D1b — À TRAITER (transitif)] `insertHighlightEventsFromData`
  (`engine_highlight_events.go`) devient orphelin test-only après la suppression
  d'`insertFetchedMatch` (son seul caller prod) — le chemin batch persiste les events via
  SharedPersister. À supprimer avec ses 2 tests (`highlight_events_orchestration_test.go`)
  après vérif du sibling `ProcessHighlightEvents`. Hors périmètre nommé de D1b.
- [LOT D1 / D1c — TRAITÉ + BUG PRÉ-EXISTANT CORRIGÉ] L'audit H5-sous-V2 (3 agents + vérif sur
  pièces) a révélé que le pipeline V2 (défaut prod) est MONO-TITRE (orchestrator câblé
  halo_infinite en dur) et ne route PAS les titres live-only : les joueurs Halo 5 (SyncEnabled
  par défaut → dans SyncablePlayers) étaient traités comme Infinite sous V2, leur chemin
  liveRunner (testé, correct) bypassé. D1c étape 1 corrige ce BUG : RunOnceTrigger partitionne
  par `livesync.HandlesTitle` (H5 → syncPlayer→liveRunner ; Infinite → orchestrator). Puis
  suppression du flag `LEVELUP_SYNC_PIPELINE` + du fallback auto V2→V1.
- [LOT D1 / D1c — REPLI documenté] `syncPlayer` + sa branche moteur (RunnerFactory→engine.run)
  NON supprimés : `main.go:1104` câble l'orchestrator V2 CONDITIONNELLEMENT (si pool+queue+
  metaDB présents) → orchestrator-nil est un scénario boot RÉEL. `syncPlayer` conservé comme
  (a) chemin des titres live-only et (b) filet structurel de boot — ce n'est PLUS le pipeline
  V1 flag-sélectionnable (supprimé). C'est le « repli autorisé » du plan (fallback auto
  supprimé), pas une suppression totale de syncPlayer.
- [LOT D1 / D1c — K2b NON impacté] `engine.run`/`RunDelta` confirmé PARTAGÉ (watcher, HTTP,
  handlers, CLI, admin convergence) → NON supprimé. La condition « si engine.run V1-only →
  K2b [~] » n'est PAS remplie : K2b (refactor de run() 483 L) reste un item LOT K valide.
- [LOT D1 / D1c — GATE live-sync différé] Le gate D1c (3) « sync live complet en local » exige
  tokens/réseau réels, non exécutable par l'agent. Couvert par `-tags=integration -p 1 ./...`
  (vert) ; le sync live reste un contrôle MANUEL avant le land sur main.
- [LOT D1 / D1f — BUG latent : hook Prestige HTTP droppé] `SyncHandler.WithPrestigeHook`
  (`sync_handler.go:226`) est un STUB no-op (`return h`), mais `server.go:1292` lui passe
  `prestigeBundle.RunPostSync` — le hook est SILENCIEUSEMENT ignoré. Conséquence probable : un
  sync manuel déclenché via l'endpoint HTTP ne lance pas le post-sync Prestige (l'auto-sync
  scheduler passe, lui, par le hook du SyncEngine `engine_options.go:78`, correct). À VÉRIFIER
  puis câbler (ou retirer le stub + le call si redondant avec le chemin engine). Hors périmètre
  D1f (règle 7) — candidat LOT K (couches) ou fix dédié.
- [LOT D1 / D1f — résidu TODO non traité] Passe rapide (outillage = livrable) : NON datés/
  supprimés (aucun n'est échu, le lint reste vert) : (a) cluster « TODO P4 ADR 0006 : retirer
  *100 » (~10 occurrences service/analysis) = migration unité canonique 0..1 cohérente, à
  traiter en bloc (pas de dating piecemeal arbitraire) ; (b) TODO dans les fichiers
  `session_compare_*` = supprimés avec les fichiers au titre de DEC-1 ; (c) TODO Phase 2/3
  Prestige/squad = travaux futurs réels. Convention `TODO(expiry:)` désormais outillée pour
  toute NOUVELLE dette datée.
- [LOT D1 / D1e — baseline résiduelle + report justifié] Après centralisation, 29 `os.Getenv`
  prod subsistent hors `internal/config` (34 avant). AUCUN n'est plus une lecture divergente
  multi-sites d'un flag de déploiement (le défaut CR A6 est éradiqué). Résidu classé :
  (a) sentinels/secrets auth `SPNKR_*` + `LEVELUP_OAUTH_CLIENT_ID` + `SPNKR_AZURE_CLIENT_SECRET`
  — gardés par `sentinel_test.go` (ADR 0023), retrait Phase 5 ; (b) bootstrap logging
  (`observability/logging/config.go` — c'est le loader de config logging, analogue à config) ;
  (c) fixtures de test (`testfixtures/external_chunks.go`) ; (d) flags LUSR expérimentaux
  shadow (`skill_v2_*.go` : `lusrV2EnvFlag`/`lusrCanonicalEnvFlag`/`lusrModeCouplingEnvFlag`/
  `lusrSquadOffsetEnvFlag`) — harnais de test dédié, tuning expérimental, pas de la config
  opérateur ; (e) knobs de tuning sync mono-lecteur profonds dans l'engine
  (`LEVELUP_SYNC_RESOLVE_ASSETS`, `LEVELUP_SNAPSHOT_GRACE_HOURS`, `LEVELUP_POSTSYNC_BURST`,
  `LEVELUP_PERSIST_NO_FETCH_CACHE`, `LEVELUP_FRESH_FILM_RETRY`, `LEVELUP_FILM_RETRY_WINDOW_HOURS`,
  `LEVELUP_NOTIFY_VERSIONS`) — lecteur unique chacun, zéro divergence ; les centraliser
  exigerait de plomber `cfg` dans les internals du SyncEngine (gros diff, faible valeur).
  REPORT justifié (règle 3 : pas de valeur/accès bloquant, mais périmètre disproportionné
  pour un lot « flags & guards ») — follow-up possible en LOT K (structure/couches).
- [LOT D1 / D1d — tension règle 11] `MULTI_TITLE_API_ENABLED` défaut OFF est un gate de rollout
  qui « laisse une feature OFF pour plus tard » — ce que la règle CLAUDE.md n°11 proscrit. NON
  corrigé en D1d (doc-only) : la bascule ON relève du chantier d'activation multi-titre
  (phase 1b, cf. `HANDOFF` multi-titre), pas de la campagne d'audits. D1d a
  documenté son cycle de vie (critère de bascule + renvoi règle 11 pour le retrait) sans changer
  le défaut.
- [LOT G / G5 — colonne `discord_notified` bool orpheline transitive] La suppression de
  G5 a retiré le backfill `discord_notified` (bool legacy) → `discord_notified_at`. La
  colonne `discord_notified` (bool) n'a plus qu'un rôle de schéma : listée dans les rebuild
  `media_files` (`steps_shared_social_media_files_drop_filepath_unique.go:221`, `ops/media_store.go`)
  + un test persister. Aucun lecteur/writer applicatif. Candidat DROP (recette ADR 0026
  comme G14) — NON traité en G5 (règle 7, hors cible nommée `discord_notified_at`).
- [LOT F / F7 — DÉCISION PRODUIT À TRANCHER] H5 `engagement` : le coarse (title.toml) déclare
  la capability `engagement` mais le fine `engagement.score = not_exposed` (capabilities.toml),
  commentaire « events présents, coefficients à recalibrer Phase 2 ». Vérifié : l'adapter H5
  ne câble PAS engagement → `not_exposed` = état honnête. Choix requis : (A) exposer un score
  H5 non calibré → `degraded` (montre une valeur peu fiable) ; (B) le garder caché → `not_exposed`
  ET retirer `engagement` du coarse (le titre ne l'expose pas). Le test miroir coarse↔fine
  (générique, tous titres) sera livré AVEC la décision (H5 le violerait en l'état). Reco : (B)
  — ne pas montrer de donnée non fiable ; réactiver quand la calibration Phase 2 est faite.
- [LOT H — l'audit SOUS-ESTIME la nuance des dédups (vérifié 2026-07-04)] Les items H sont
  présentés comme des dédups « mécaniques » mais la vérif on-pièces révèle des DIVERGENCES
  SÉMANTIQUES qui interdisent le remplacement aveugle : (1) **H5 `safeDiv` ≠ `analysis.SafeRatio`** :
  `safeDiv(a,0)` renvoie le NUMÉRATEUR (KD=kills quand deaths=0) + arrondit à 2 déc. ; `SafeRatio(n,0)`
  renvoie `0.0` NON arrondi. Les remplacer = BUG de comportement (KD à 0 au lieu de kills sur les
  matchs sans mort). safeDiv N'EST PAS un doublon → garder, statuer `[~] faux positif`. (2) **H2**
  littéraux `bid(` à préfixe VARIABLE (`xuid`/`mp.xuid`/`opp.xuid`/`gamertag`, `%`/`%%`) ; le const
  `SQLIsBot` suppose `xuid` nu → migration NON triviale (helper paramétré `SQLIsBotCol(col)` requis
  pour les alias, sinon allowlist). (3) comptes réels > audit (H1=115/52, H2=58/30). CONSÉQUENCE :
  LOT H (et probablement I/J) demande une vérif per-copie AVANT migration + garde-rail — pas un
  grep-replace. Exécution careful multi-session (l'investigation workflow `lotHN-investigate` a la
  carte complète ; la stratégie safe-first + K-dédié est posée en tête de §4 LOT H).
