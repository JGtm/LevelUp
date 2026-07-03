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
- [ ] D1c — DEC-2 (VALIDÉ : suppression complète) : supprimer le pipeline V1 ENTIÈREMENT —
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
- [ ] D1d — DETTE §2.1 : documenter le cycle de vie des autres flags
  (`LEVELUP_PERSIST_BATCH_ASYNC`, `MULTI_TITLE_API_ENABLED`, `LEVELUP_EVENTS_CONVERGENCE`,
  `LEVELUP_CONTRACT_VALIDATE`) — modèle `shared_reader_legacy.go:30-34` (date de
  basculement + date cible de retrait + critère mesurable).
- [ ] D1e — CR A6 : centraliser les 41 `os.Getenv` hors `internal/config` dans
  `config.AppConfig` au boot, injecter (élimine la double lecture scheduler/handler de
  PERSIST_BATCH — devient sans objet après D1b pour ce flag, reste vrai pour les autres).
- [ ] D1f — DETTE reco 7 : généraliser `TODO(expiry:YYYY-MM-DD)` + lint léger qui échoue à
  date dépassée (précédent : `season_pass_repo_tracks.go:254`). Trier les 513 TODO/FIXME :
  dater ceux qui référencent des phases mortes, supprimer les caducs (passe rapide, pas
  d'exhaustivité ligne-à-ligne exigée — l'outillage est le livrable).

Gate D1 : `go build ./... && go test ./...` ; grep `LEVELUP_PERSIST_BATCH` → 0 ;
grep `os.Getenv` hors `internal/config` → baseline notée au Journal (cible : ~0 hors
main/cmd bootstrap).

### LOT E — ART résiduel & écritures à risque

Objectif : plus aucun chemin prod du pattern déclencheur ART #23046 ; tripwire étendu.

- [ ] E1 — ARCHI 15 : router `writeOneMatch` de l'import OpenSpartan
  (`openspartan_import_service.go:318`) vers `persist.SharedPersister` (INSERT-only +
  pre-check registry), comme le livesync H5. Retirer l'entrée allowlist du tripwire.
- [ ] E2 — ARCHI 16 : `backfill_registry_names.go:98` — convertir l'UPDATE bulk multi-row
  nu sur match_registry en row-by-row par match_id (ou réserver au CLI serveur-arrêté,
  décision au code) ; ÉTENDRE la regex du tripwire aux UPDATE multi-row nus sur les
  tables critiques.
- [ ] E3 — ARCHI mineur : `no_art_patterns_test.go:184` — retirer l'exclusion `ops/`
  (hypothèse « exécuté hors serveur » fausse : plomberie média ops tourne in-process).
- [ ] E4 — DETTE §2.5 : `TestAllowlistJustifiesEverything` passe de warning à erreur.
- [ ] E5 — ARCHI mineur : `progression/profile/queries.go:40` — SQL de lecture HTTP →
  platform/duckdb + filtre temporel canonique (COALESCE timezone, croisé H1).
- [ ] E6 — ARCHI mineur : bare connects RO sur player DB potentiellement tenue RW :
  `worldenrich/wiring.go:33`, `platform/auth/pool/discovery.go:229` → pattern
  `OpenReadForQuery` / provider.
- [ ] E7 — ARCHI mineur : `sync/schema.go:22` — DDL bootstrap → `internal/migration`.
- [ ] E8 — DEC-8 : `games/halo_5/livesync/csr_match.go:69` — router les écritures
  per-match H5 via la couche persist (BatchBuilder/Persister dédié). Convention à figer
  en F14 (ADD_TITLE.md).

Gate E : `go test ./... && go test -tags=integration ./internal/persist/... ./internal/sync/...` ;
tripwire étendu vert ; allowlist ART réduite et bloquante.

### LOT G — Purge du code mort (« dead code museum »)

Objectif : 0 module mort (règle projet) ; exécuté AVANT F/H pour réduire la surface des
migrations suivantes. Chaque suppression : retirer code + tests + imports + entrées
openapi/migrations associées, puis build+tests.

- [ ] G1 — ARCHI 26 : supprimer `internal/api/gen` (2 536 L, 0 importeur) + cible make +
  3 exclusions tooling + corriger le message du drift-test (`make gen` → chaîne Huma).
- [ ] G2 — CR A7 : cluster home legacy — supprimer les 10 exports morts (`ComputeKPIs`,
  `ComputeTrend`, `BuildHeroCard`, `BuildHighlights`, `BuildSessionSummaries`,
  `BuildRecentMatches*` x4) + tiles legacy transitifs + leurs tests ; CONSERVER
  `mapImageURLFromRegistry`, `mmrDelta`, `float64PtrVal`, `intPtrIfPos` ; corriger la doc
  fausse de `home_canonical.go:4-18`.
- [ ] G3 — CR A8 / DEC-1 : supprimer la feature session-compare entière : 17 fichiers
  front (`features/session-compare/`), `handlers/session_compare.go`,
  `service/session_compare_service.go` + 3 helpers, `domain/session_compare.go`, entrée
  openapi.yaml, query key, manifest §compare. (~25 fichiers.)
- [ ] G4 — CR A9 : `SquadV2RouteHost.tsx` + `SquadV2Page.tsx` (ATTENTION : `squad/v2/types.ts`
  reste vivant).
- [ ] G5 — CR A9 + ARCHI mineur : chaîne `NotifyNewMedia` → `queryUnnotifiedMedia` →
  `markMediaNotified` (`notify/notifiers.go:88-190`) + la migration de colonne
  `discord_notified_at` qui ne vit que pour elle + tests.
- [ ] G6 — ARCHI mineur : `ReassociateMedia` (`media_service.go:338`) — méthode + interface
  + types (route supprimée 2026-04-29).
- [ ] G7 — ARCHI mineur : `ServiceConfigIDFor` (`domain/title/registry.go:249`, fonction morte).
- [ ] G8 — ARCHI mineurs : constantes SQL mortes Q4/Q4MV (`queries_career.go:7`), Q26e
  (`queries_home_citations.go:283`), Q24 + doc fausse (`queries_career_encounters.go:446`).
- [ ] G9 — ARCHI mineur : entrées mortes `config/titles/halo_5/mappings/assets.toml:13`
  (slugs divergents de la convention).
- [ ] G10 — CR A9 : `upsertLUSRRatingsLegacy` — vérifier déjà supprimé en A3, sinon supprimer.
- [ ] G11 — CR A9 / DEC-7 : `SessionKDATimeline.tsx`, `SessionOcdrScatter.tsx` — supprimer,
  noter dans `.ai/` backlog UI.
- [ ] G12 — CR A9 : `MapMuToLegacyRating`/`MapTierSubToLegacyRating` → dé-exporter
  (test-only).
- [ ] G13 — CR A9 : `processMatch` legacy — vérifier déjà supprimé en D1b, sinon supprimer.
- [ ] G14 — DETTE §2.5 / DEC-6 : `known_teammates_count` + `friends_xuids` — DROP au
  prochain rebuild (suivre la recette ADR 0026) ; retirer toute lecture résiduelle.
- [ ] G15 — ARCHI mineur perf : `sync/aggregates.go:37` — `mv_map_stats` rebuildée à chaque
  sync sans AUCUN lecteur Go → supprimer le rebuild (et la vue si rien ne la lit).
- [ ] G16 — PRÉ-EXÉCUTÉ le 2026-07-02 (section 0 « Complétude » ajoutée à
  delivery-checklist : suppression routing => suppression code+tests). Statuer `[~]` au
  passage du lot après vérification.

Gate G : `go build ./... && go test ./...` ; `npm run typecheck && npm run build` ;
grep de chaque symbole supprimé → 0 ; diff openapi cohérent (suppression session-compare).

### LOT F — Title-agnosticism (fuites HINF, manifests H5)

Objectif : plus aucune donnée/label/URL Infinite servie sous titre H5 ; manifests H5
complets ; ratchet anti-slug étanche. (Priorité audit : ces items bloquent Halo 5, les
traiter AVANT les refactors structurels.)

- [ ] F1 — ARCHI 28 : `media_repo.go:112` (+ :240, q37_enrich) — injecter la
  classification de modes par titre au wiring (seam `analysis.PairNamePrefixesFunc`
  existant) ; plus de couplage platform/duckdb → games/halo_infinite.
- [ ] F2 — ARCHI 29 : providers CSR Explorer/Compare (`registry_pages.go:360`, `:812`) —
  gate capability (`csr.live`) ou map slug→provider (pattern MT-09) ; un joueur H5 ne
  voit plus le CSR Infinite.
- [ ] F3 — ARCHI 32 : URL Waypoint (`match_view_builders_header.go:59`,
  `match_history_service_enrich.go:278`) → derrière `TitleAssetURLAdapter` (déjà injecté) ;
  capability/None pour les titres sans page match — plus de lien mort en Match View H5.
- [ ] F4 — ARCHI 31 : labels d'outcome FR en dur x3 (`match_history_service.go:34`,
  `analysis/home_locale.go:55`, notify/discord) → `resolver.Semantic(slug).Outcomes().Label(...)`,
  littéraux en failsafe. Corrige l'incohérence Victory/Defeat vs Win/Loss.
- [ ] F5 — ARCHI 33 : labels KPI en dur (`compare_service.go:472` ~20 labels,
  `timeseries_service_tabs.go:48`) → FieldMappingSet du titre ou key-only + labelling
  front via /field-mappings. (`session_compare_service.go:414` : sans objet si G3 a
  supprimé — statuer `[~]` réf G3.)
- [ ] F6 — ARCHI 34 : compléter `config/titles/halo_5/mappings/fields.toml` (5 → ~59
  FieldKeys) + test de parité « FieldKeys requis vs déclarés » pour tout titre actif.
- [ ] F7 — ARCHI 37 : réconcilier coarse `engagement` (title.toml:48) vs fine
  `engagement.score = not_exposed` (capabilities.toml:28) + test générique des paires
  miroir coarse↔fine (croisé L2).
- [ ] F8 — ARCHI 36 : câbler `LoadAuthDescriptor` au boot par titre actif ;
  `DefaultHaloAuthDescriptor()` réservé au titre par défaut (`halo_exchange.go:71`).
  Corriger le statut MT-02 au registre.
- [ ] F9 — ARCHI 35 : les 5-6 handlers Ascension épinglés DefaultSlug
  (`server.go:1554,1579,1584,1588,1609,1614`) → `ctxkeys.TitleSlug(ctx)` ou a minima
  `RequireCapability` → 503 propre. (Débloque MT-19 / Phase 1b.)
- [ ] F10 — ARCHI 30 : élargir la regex du ratchet `no_slug_comparison_test.go:35`
  (`(?:\w+\.)?DefaultSlug` + cas `TitleSlug(ctx)`) ; allowlister les sites détectés
  (`sync/comeback.go:34/95`, `sync/coordinator.go:316`) avec justification datée.
- [ ] F11 — ARCHI 38 : titre par défaut hors TOML — WARN explicite au boot si un
  `title.toml` infinite existe (skip muet `config_loader.go:185`), ou parity-test
  built-in vs TOML versionné. Trancher et documenter dans ADR 0025.
- [ ] F12 — ARCHI 13 / DETTE §2.4.1 : migrer le pipeline film Infinite entier
  (`weapon_data.go`, `weapon_scanner/parser/correlation/reconciliation`,
  `highlight_event_parser.go`, `spawn_detection.go`, `kill_attribution.go`) de
  `internal/analysis/` vers `internal/games/halo_infinite/film/` (migration mécanique
  vérifiée par l'audit) + entrée au registre MT.
- [ ] F13 — DETTE §2.4.2 : paramétrer les goldens par slug
  (`analysis/timeline/golden_test.go`, `home_canonical_test.go`,
  `synthesis_canonical_test.go`) pour distinguer régression vs divergence de titre.
- [ ] F14 — DETTE §2.4.3 : figer la convention « nouveau titre » dans `docs/ADD_TITLE.md`
  (hiérarchie client → livesync → migrations, écritures via persist — cf. E8).
- [ ] F15 — ARCHI mineurs title-agnosticism (traiter CHAQUE puce de la section, ~20) :
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
- [ ] F16 — ARCHI 7 : dédupliquer `augmentWithActiveRankedCSRs` (copie DI
  `registry_pages.go:380` vs original `sync/career.go:255`, divergence NameFR/NameEN déjà
  réelle) → une implémentation unique, nom résolu via semantic adapter + locale.

Gate F : archlint étendu vert (F10) ; test parité fields.toml (F6) et coarse/fine (F7)
verts ; grep `halowaypoint` hors `games/halo_infinite` + `platform/halo` → 0 (le client
sync bouge en K3e — allowlist temporaire datée si besoin) ; smoke test des pages H5
(Médias, Explorer, Match View) via `verify`/run local.

### LOT H — Repropagation & duplication (source de vérité unique)

Objectif : plus de copie locale divergente d'un helper canonique existant ; chaque helper
embarque son garde-rail anti-régression le jour de sa livraison (CR reco 1).

- [ ] H1 — CR A10 : helper `SQLStartTimeUTC(alias)` dans `analysis/sql_fragments.go` +
  migration mécanique des 87 copies (33 fichiers) + test grep interdisant le littéral
  hors sql_fragments.go.
- [ ] H2 — CR A11 : adoption `SQLIsBot` sur les 36 littéraux résiduels (19 fichiers) +
  test grep interdisant `LIKE 'bid(%` hors sql_fragments.go.
- [ ] H3 — CR A12 : migrer SynthesisPage sur `useLocalFilterBar` (~250 L supprimées,
  `SynthesisPage.tsx:51-61` === `useLocalFilterBar.tsx:23-33`) ; en profiter pour corriger
  le matching de filtres couplé aux libellés FR (`useLocalFilterBar.tsx:234-236`, CR mineur).
- [ ] H4 — CR A13 : compléter `lib/formatters/{date,duration,percent}.ts` et purger les
  copies (4 formatDate, 4 formatPercent, 3 durées) ; absorber le dictionnaire 60 L inline
  + `formatDateShort` homonyme de `PeriodSessionRail.tsx:55-114` (CR mineur).
- [ ] H5 — CR A13 : côté Go — `Ptr[T]` générique (remplace 5 `strPtr`) ; `safeDiv`
  (`teammates_service_kpis.go:224`) → `analysis.SafeRatio`.
- [ ] H6 — CR A13 : composant `<OpenMatchIcon/>` (9 copies, 8 fichiers) ; factoriser le
  socle d'option ECharts répété x7 dans `TimeseriesFormCharts.tsx`.
- [ ] H7 — CR A13 : helpers couleur win/loss/ratio recodés x4 (palmares/career) →
  module partagé (respect règle couleurs §20).

Gate H : tests grep H1/H2 verts ; `npm run typecheck && npm run test` ; go test ; compte
des copies au Journal (avant/après).

### LOT I — i18n

Objectif : purge FR monolingue + anglicismes ; règle lint passée en `error` à la fin.

- [ ] I1 — CR A17 : chemins d'erreur auth/setup/onboarding → i18n.ts (FR+EN) :
  `XboxLoginPage.tsx:106-418` (~12 strings), `StepDeviceCode.tsx:129-142`,
  `StepInitialSync.tsx:81-87`, `RegisterPage.tsx:71-79`, `OpenSpartanImportCard.tsx:346-365`.
- [ ] I2 — CR A18 : surfaces de données — 9 colonnes scoreboard
  (`MatchScoreboard.tsx:51-76`, finir la migration `t.sbCol*`) ; heatmap Explorer entière
  (`ExplorerActivityHeatmapChart.tsx:18-84`) ; 24 `toLocaleString('fr-FR')` figés
  (7 fichiers synthesis/career/session) ; « Par carte »/« Par mode »/« Analyser »
  (SynthesisPage, SquadLayout).
- [ ] I3 — CR A19 : « streak » → « série » dans les textes FR (`notifications/i18n.ts:171,
  201,283,286`, `ascension/i18n.ts:162-169`, `settings/i18n.ts:456` ranked/casual,
  `match-view/i18n.ts:239`, `squad/i18n.ts:429`, `help/i18n.ts:200`) — le glossaire
  officiel (`help/i18n.ts:326`) fait foi.
- [ ] I4 — CR mineurs : ~33 ternaires `locale === 'en' ? ... : ...` inline → i18n.ts ;
  aria-label FR figé (`NotificationItem.tsx:72`) ; labels hardcodés
  (`TimeseriesFormCharts.tsx:207`, `LeaderboardBlock.tsx:435`, `LeaderboardPP.tsx:82-87`).
- [ ] I5 — CR reco 4 : passer `@levelup/no-hardcoded-strings` en `error` (après purge
  I1-I4).

Gate I : `npm run lint` vert AVEC la règle en error ; revue visuelle EN des pages touchées
(badges, scoreboard, heatmap).

### LOT J — Performance DuckDB

Objectif : mesurer d'abord (règle audit), puis desserrer les goulots ; batcher les N+1
des chemins HTTP chauds.

- [ ] J1 — ARCHI 48 : (1) exporter `sql.DBStats` (WaitCount/WaitDuration) par handle via
  expvar (ADR 0009) ; (2) APRÈS lecture des stats et audit des UPSERT player reposant sur
  l'effet de bord MaxOpenConns(1) : petit pool lecture 2-4 conns pour les player DBs (ou
  généralisation du modèle sharedprovider). Ne pas inverser l'ordre.
- [ ] J2 — ARCHI 49 : configurer memory_limit/threads par classe de DB dans
  `openSQLDBFor` (params DSN), valeurs exposées dans /health (8-15 instances x défauts
  80 % RAM = surengagement VPS).
- [ ] J3 — ARCHI 46 : `GetHistoryForAvgBulk` (IN + ROW_NUMBER PARTITION BY xuid), un seul
  Get du SharedReader — remplace jusqu'à ~8 exécutions par clic Match View
  (`match_view_data_loaders.go:386`).
- [ ] J4 — ARCHI 47 : `LoadSquadMatchesBulk` groupé par teammate_xuid + lookup gamertags
  batch (`teammates_service.go:185`).
- [ ] J5 — ARCHI 44 : `LoadAll` full-history par hit (`match_history_repo.go:32`) → cache
  par joueur invalidé post-sync OU matérialisation du placement (le push-LIMIT naïf est
  impossible — nuance vérifiée par l'audit).
- [ ] J6 — ARCHI mineurs N+1 batchables (8 sites, lecture seule) : `sync/engine.go:696`,
  `sync/skill_v2_helpers.go:28`, `relations_moments_service.go:140`,
  `fanout_service.go:73`, `sync/session_recalc.go:80`,
  `sync/backfill_registry_names.go:157` (croisé E2), `handlers/prestige.go:718`,
  `registry_catalog_expand.go:94` (croisé K1d).
- [ ] J7 — ARCHI mineur : CTE perfect de Q26 bornée (agrège tout l'historique pour un
  LIMIT 150) (`queries_home_citations.go:26`).
- [ ] J8 — ARCHI mineur : `db.go:192` — limites de pool 4/2 en constantes nommées +
  observabilité d'attente (couvert en partie par J1).
- [ ] J9 — ARCHI mineur : `registry_relations_cross_game.go:81` — emprunt non-possédant
  d'un handle cross-titre → acquisition sûre (refcount/provider du titre visé).

Gate J : expvar DBStats visibles dans /debug/vars ; go test ; mesure avant/après notée au
Journal (temps de réponse Match View + page escouade en local).

### LOT K — Structure & couches (le plus gros — sous-lots commités séparément)

Objectif : la racine api/ cesse d'être une 2e couche service ; god functions/packages
découpés ; chemins via PathResolver. Chaque sous-lot = 1 commit + build/tests verts.

K1 — Extractions de couches (ROI d'abord) :
- [ ] K1a — ARCHI 1 + 2 (reste) + CR A2 (partie post-sync) : extraire le pipeline
  post-sync de api/ → `service/postsync/` ; SQL → repos platform/duckdb
  (`PlayerSnapshotRepo`, `post_sync_progression_queries.go:139`,
  `post_sync_deltas_records.go:21-48` loadPlayerRecord/upsertPlayerRecord) ; formules
  produit → analysis/ ; au passage : `EmitPostSyncDeltas` 247 L → table-driven (CR mineur),
  « BestKDA » en quotient → formule ADR 0006 (ARCHI mineur), seuils 0.05/0.01 nommés
  (CR mineur), `outcome = 2` → `outcomeSQLEq` (`post_sync_progression_queries.go:301`).
- [ ] K1b — ARCHI 5 : cascade refresh tokens de `registry_auth.go:169` (~130 L dupliquant
  `RefreshHaloTokensViaStoreFirst`) → platform/auth, implémentation unique (pré-requis D2).
- [ ] K1c — ARCHI 3 : helper unique côté platform pour les écritures sync_meta SOUS LEASE
  dblease (ADR 0013) — remplace les 2 copies (`notifications_title_ready.go:141`,
  `notifications_boot.go:112`) ; pattern `prestige_lazy_service.go:119`.
- [ ] K1d — ARCHI 4 : `ExpandPlaylistChildren` (`registry_catalog_expand.go:94`) → ops/ ou
  service/ ; DDL → internal/migration ; factoriser la 3e copie du pattern upsert ART-safe ;
  batcher ses 3 requêtes/entry (croisé J6).
- [ ] K1e — ARCHI 6 : `dataQualityHandles` (`registry_data_quality.go:33`) → lire via
  `cfg.SharedProvider` (pattern `acquireProgressionSharedRead`) — élimine le conflit avec
  les fenêtres RW du B-swap pour les 5+ runners admin.
- [ ] K1f — CR A3 : extraire `service.BackfillOrchestrator` table-driven
  (`{nom, gate, fn}`) du handler `handleStartBackfill` (368 L, `backfill.go:76-443`) ;
  le handler ne garde que validation + 202.
- [ ] K1g — CR A2 + A1 (partie) : `loadTitleAssetDrawerData` + `loadCSRBadgeResolver`
  (`server.go:137-222`) → platform/duckdb (modèle `loadTitleRankImageURLs`) ; supprimer le
  double chargement metadata H5 au boot (`server.go:773` vs `:816`).
- [ ] K1h — CR A5 + ARCHI mineurs handlers : handlers qui construisent repos/métier →
  services + ports : `progression.go:162-251` (ProgressionService, jointure catalog x
  earned hors handler), `campaign.go:221`, `admin_auto_sync.go:206`, `sync_handler.go:165`,
  `player_profile.go`, `home.go` ; `bootstrap.go:23`/`title_sync.go:30` → port.* ;
  `commendation_handler.go` → api/handlers/ ; `registry_weapon_coverage.go:102` (SQL slug
  concaténé → paramétré + déplacé).
- [ ] K1i — ARCHI 8 + CR A5 : interfaces consumer-side étroites pour les couplages
  service→service concrets : `home_service.go:264` (spartanIdentityProvider, pattern
  `explorer_service.go:65`), `career_service.go:74`, `filters_service.go:26`.
- [ ] K1j — ARCHI 9 : persistance catalogue (~200 L SQL + *sql.DB dans
  `catalog_fetcher_service.go:73` + même dérive `openspartan_post_import_service.go:170`)
  → CatalogRepository (platform/duckdb) ou Persister dédié via port.* ; + gate titre sur
  le référentiel rankedplaylists dans le drain (`catalog_fetcher_service.go:197`, mineur).
- [ ] K1k — ARCHI 10 : `career_live_fetcher.go:150` — factory client Halo injectée côté
  registry ; promouvoir CareerRankData/SpartanCustomizationData en types domain/ (5
  fichiers career_live_* cessent d'importer internal/sync). Débloque career-live H5.
- [ ] K1l — ARCHI 11 + 14 + mineurs chemins : TOUS les chemins via PathResolver :
  `openspartan_import_service.go:481` + `server.go:1344` (stash friends layout legacy),
  `server.go:290/651/1112` (data/cache — ajouter `CacheRootDir()` au resolver),
  `ops/seed_demo.go:392` (MediaDataDir existant), `seed_demo_multititle.go:43` (layout
  réimplémenté), helper `PlayersRootDir(slug)` (6 copies du filepath.Join, dont
  `data_health_check.go:257`), `config.go:205` (double mécanisme data/auth + data/sessions).
  Garde-rail archlint en L2.
- [ ] K1m — ARCHI 12 [TRACKÉ] : exécuter le plan de l'allowlist media — extraire
  MediaRepository/MediaStore de `media_service.go:357` + `media_index_service.go`, vider
  l'allowlist de `no_duckdb_import_test.go`.
- [ ] K1n — ARCHI mineurs couches service/analysis/domain : algos purs → analysis/
  (`engagement_timeseries_binning.go:65`, `timeseries_service_aggregations.go:58`,
  `teammates_squad_charts_intensity_perminute.go:141`, `match_history_placement.go:35` —
  résoudre le cycle regex dupliquée) ; `friends_orchestrator_service.go:102` → port ;
  liste de modes dupliquée (`engagement_admin_service.go:28`) ; impuretés analysis/ :
  `combat_yield.go:28` (état global), `comeback.go:130` (slog), `sql_fragments.go:26` +
  `perfect_kills.go:28` (fragments SQL — statuer : tolérés si documentés, sinon déplacer),
  `identity.go:162` (générateur DDL), `citations.go:174` (URLs assets),
  `world_stats.go:153` (TitleDataAdapter de fait), `mode_label.go:49` (playlists dupliquées
  vs games/), `home_kpis.go:12` (dépendance legacymatch — croisé G2) ;
  `domain/achievement_categories.go:27` + `domain/job.go:92` (croisé F15).

K2 — God functions à risque (tâche dédiée, passer la grille plan-review avant chaque item) :
- [ ] K2a — ARCHI 22 + CR A1 : `NewRouter` ~1 470 L → `buildStores` / `buildTitleRuntime` /
  `applyMiddlewares` / `mountXxx` par domaine ; cible < 100 L ; `waypointExplore` (closure
  46 L) → service ; purge sessions goroutine et retry-loop metadata extraites ; corriger le
  doc-comment rattaché à la mauvaise fonction.
- [ ] K2b — ARCHI 23 + CR A4 : `SyncEngine.run()` 483 L → extraire au minimum le bloc
  drain/ré-acquisition (l.505-597) et la boucle de pagination (l.346-497) en méthodes
  nommées (précédent : engine_backfills.go). Tests e2e sync verts obligatoires.
- [ ] K2c — ARCHI 24 : `auto_sync.go` 1 083 L → scission en 3 (scheduler / factory engine
  / métriques-convergence) ; exemptions 80 L résolues (BuildEngine, RunOnceTrigger).
- [ ] K2d — ARCHI 25 : `SeedDemo()` 203 L → scission extract/configs/identity + phases
  nommées (chemin de déploiement sensible — tester via regen demo local).
- [ ] K2e — CR mineurs : `strings.Title` déprécié (`engine.go:191`) → map 2 entrées ;
  ~15 blocs `g.Go` copiés (`match_view_data_loaders.go:68-256`) → helper + `WarnContext(gctx)`.

K3 — God packages & structure (mécanique, 1 domaine = 1 PR/commit) :
- [ ] K3a — ARCHI 17 : platform/duckdb (143 fichiers) → extraction par domaine, commencer
  par prestige ; `halo5_*.go` → games/halo_5 ou duckdb/halo5.
- [ ] K3b — ARCHI 18 : service/ (127 fichiers) → sous-packages par feature, commencer par
  teammates (13 fichiers) ; archlint interdit les imports croisés entre features.
- [ ] K3c — ARCHI 19 : sync/ (111 fichiers) → extraire sync/skill/ (17) et sync/snapshot/
  (6) ; ratchet de gel sur la racine (aucun nouveau fichier racine) ; le neuf va dans v2.
- [ ] K3d — ARCHI 20 : racine api/ (39 fichiers) → api/wire/ pour la DI ; cible < 10
  fichiers racine (le post-sync est déjà parti en K1a).
- [ ] K3e — ARCHI 21 : client HTTP Halo Infinite (halo_client*.go, 7 fichiers) →
  platform/halo/ ou games/halo_infinite/client/ (cible montrée par games/halo_5/client.go).
- [ ] K3f — ARCHI mineurs structure : doublon `notify/` vs `notifications/` (renommer ou
  documenter) ; `prestige/` + `campaign/` → progression/ (ou documenter le choix) ;
  `worldenrich/` câblage top-level ; frontières `assets/`/`assetnames/`/`media/`
  documentées ; `metadata/` renommé ; `openspartan/` → platform/ ; `legacymatch` échéance
  documentée [TRACKÉ] ; `migration/steps_*` Halo-specific [TRACKÉ ADR 0025 Ph 1.5 — statuer
  seulement] ; PathResolver dans domain/ (documenter le choix) ; god-files secondaires
  (10 items : steps.go, handlers/prestige.go 1 019 L, skill_v2_shadow.go, persist_sink.go,
  adapter_data.go, db.go, registry_pages.go, pool.go x3 fns, prestige/service.go
  CreateChallenge, steps_player_base.go) → découper OU exemption justifiée par commentaire ;
  débris de splits (doc/nolint orphelins : `teammates_squad_charts_impact_events.go:444-463`,
  `post_sync_deltas.go:133,404`, nolint brouillés 5+ sites) ; `rows[:50]` + fenêtre 15 min
  nommées (CR mineurs magic numbers).

Gate K (par sous-lot) : `go build ./... && go test ./...` ; pour K2a/K2b :
`go test -tags=integration ./internal/sync/...` + diff openapi VIDE (aucune route
changée) ; archlint verts ; smoke run local du serveur après K2a.

### LOT L — Gouvernance, ratchets, contrat HTTP

Objectif : chaque règle d'architecture de l'audit finit soit corrigée (lots précédents),
soit ENCODÉE en ratchet à allowlist décroissante datée (reco centrale ARCHI).

- [ ] L1 — ARCHI 27 [TRACKÉ] : résorber les 22 schémas OpenAPI DIVERGENT (mode emit prévu
  par le drift-test), régénérer generated.ts, puis durcir : `t.Errorf` si divergent > 0.
- [ ] L2 — ARCHI TOP3 : 3 nouvelles règles archlint : (1) pas de SQL/`Open*` dans api/
  (baseline = sites restants après K, décroissante datée) ; (2) pas de
  `filepath.Join(..."data"...)` hors PathResolver ; (3) parité coarse↔fine des
  capabilities (générique, croisé F7).
- [ ] L3 — CR A20 : `.golangci.yml` — remplacer les exclusions par répertoire
  (funlen/gocyclo sur sync/, analysis/, service/, handlers/) par une baseline gelée
  (`--new-from-rev` ou baseline explicite) ; réactiver `argument-limit` (l'exclusion
  `text:` l.117-118 tue la règle) ; funlen 100 → 80 avec baseline ; supprimer le
  commentaire périmé l.92.
- [ ] L4 — ARCHI mineurs contrat : `contract_validate.go:34` — le middleware ne valide
  aucun schéma et bufferise TOUS les corps → soit implémenter la validation (flag
  CONTRACT_VALIDATE, croisé D1d), soit supprimer ; `read_budget.go:7` couplé sharedprovider
  → découpler ou documenter l'exception.
- [ ] L5 — CR A16 : query keys front — rapatrier les 11 clés inline + invalidations par
  littéral (`admin/data-quality/mutations.ts:22`) dans `lib/query/keys.ts` ; consolider ou
  documenter les 7 registres locaux (squadKeys, prestigeKeys, watcherKeys...) dans le
  skill frontend-patterns ; règle ESLint : `queryKey:` littéral interdit hors keys.ts.
- [ ] L6 — PRÉ-EXÉCUTÉ le 2026-07-02 (convention kill-switch écrite dans CLAUDE.md
  règle 11 + arch-rules § Feature flags + delivery-checklist §5). Statuer `[~]` au
  passage du lot après vérification.
- [ ] L7 — ARCHI mineur : `api/server.go:773` câblage Halo 5 copy-paste au boot (double
  lecture + fallbacks image HINF) — statuer `[~]` si déjà résolu par K1g/F15, sinon traiter.

Gate L : CI verte avec les nouvelles règles actives et baselines commitées datées ;
`TestOpenAPISchemaDrift` strict (0 divergent).

### LOT M — Tests (gaps ciblés)

- [ ] M1 — QUALITE : test d'intégration sur `RecomputeLUSRCanonicalForPlayer`
  (`lusr_full_recompute.go` — orchestrateur à 3 callers, 0 test) : dataset réaliste
  hétérogène (règle mémoire projet), ordre des matchs + watermark assertés.
- [ ] M2 — QUALITE : garantir que la CI n'est JAMAIS verte sans `-tags=integration`
  (vérifier workflows ; ajouter un job dédié ou un garde qui échoue si les tests
  integration n'ont pas tourné). MOTIVÉ PAR INCIDENT (2026-07-03, décision utilisateur) :
  le gate intégration de LOT B avait été validé à tort (20 tests `platform/duckdb` rouges
  non vus). Le job CI DOIT lancer `go test -tags=integration -p 1 ./...` (SÉRIALISÉ — sinon
  flake DuckDB mono-process + durées fantômes masquant les FAIL) et échouer sur code de
  sortie ≠ 0 (pas sur un grep de sortie). Cf. skill delivery-checklist (règle `-p 1` +
  filtre ancré `^--- FAIL:`).
- [ ] M3 — QUALITE : tests manquants : `ComputeMedalExploitScore` (`medal_exploit.go:22`),
  `GetTiming` (`weapon_data.go:224` — ATTENTION : aura bougé vers games/halo_infinite/film
  en F12) ; renforcer `ComputeImpactSummary`, `ComputeMVPLVP`, `ComputeTrend`,
  `ComputeSquadPerformanceScore` (1 seul test référent chacun).
- [ ] M4 — QUALITE : tests middleware manquants : `http_cache.go`, `read_budget.go`
  (touche la contention DB).

Gate M : `go test ./... && go test -tags=integration ./...` verts ; les nouveaux tests
échouent si on casse volontairement le code testé (vérif mutation rapide à la main).

### LOT N — Front structurel + résidus (bonus/optionnels)

- [ ] N1 — CR A14 : `LeaderboardBlock.tsx:325` → TanStack Table (règle projet, 8 tables
  de référence dans le repo).
- [ ] N2 — CR A15 : `SquadLayout.tsx` ~630 L → 3 hooks (`useSquadSessionSync`,
  `useSquadCompositionAnchor`, `useSquadPendingFilters`) + `SquadFilterBar` ; l'écriture
  localStorage sort de l'updater setState (l.143-149) vers un `useEffect`.
- [ ] N3 — CR mineurs web : bypass ECharts (`CumulativeFragGapChart.tsx:23`) ;
  `isLoading → return null` remplacé par skeleton (SynthesisPage:631, HomePage:123) ;
  listener clavier deps (`CoverFlowModal.tsx:474-482`) ; `MatchCard` ~470 L découpé
  (`match-card.tsx:65-537`) ; `joinAndSort` renommé (`mapPerfVsHistoryChart.ts:56`).
- [ ] N4 — DETTE §2.5 : politique de cycle-out des ~105 migrations — décision documentée
  (squash par version majeure à date fixe, ou TTL) dans internal/migration/doc.go (créé
  en C6).
- [ ] N5 — DETTE §2.6 : consigner la dette assumée résiduelle (ce qui reste volontairement
  non traité à l'issue du plan) dans un court doc `.ai/V7/DETTE_ASSUMEE_2026-Q3.md`.

Gate N : typecheck + vitest + revue visuelle des pages touchées (leaderboard, escouade).

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
| ARCHI | Majeurs 1-7 (racine api/) | K1a, B6, K1c, K1d, K1b, K1e, F16 |
| ARCHI | Majeurs 8-12 (service/) | K1i, K1j, K1k, K1l, K1m |
| ARCHI | Majeurs 13-14 (analysis/chemins) | F12, K1l |
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
| DETTE | TOP 1-10 | C1, A4/D1c, D1b, D1a/D2, C2, F12, C7, C4, C5, C8 |
| DETTE | §1.3 READMEs / §1.4 invariants | C6, C5 |
| DETTE | §2.1 guards / §2.2 Phase 5 / §2.3 Prestige | D1b-f, D2, C7 |
| DETTE | §2.4 multi-titre (film, goldens, template) | F12, F13, F14/E8 |
| DETTE | §2.5 schéma DB (colonnes, migrations, allowlist) | G14, N4, E4 |
| DETTE | §2.6 dette assumée / recos 1-9 + gouvernance | N5, C3, C8, D1f |
| QUALITE | B1-B2 + cause racine | S1, S2, S3 |
| QUALITE | M2-M4 + mineurs accès | S4-S8 |
| QUALITE | Tokens/secrets (probe, CLIs) | S5, S9, D2 (dette Phase 5) |
| QUALITE | XSS frontend | A5 |
| QUALITE | Robustesse #3/#6/#8/#10 + autres | B10-B16 |
| QUALITE | Axe 4 tests (gaps) | M1-M4 |
| CR | C1-C4 | A1-A4 |
| CR | A1-A6 (archi Go) | K2a, K1g, K1f, K2b, K1h/K1i, D1e |
| CR | A7-A9 (code mort) | G2-G5, G10-G13 |
| CR | A10-A13 (duplication) | H1-H7 |
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
```

## 7. Découvertes hors périmètre (à remplir — NE PAS traiter sans accord)

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
