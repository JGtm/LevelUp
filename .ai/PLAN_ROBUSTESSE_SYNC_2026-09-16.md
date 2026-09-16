# PLAN — Robustesse du sync par le pool : verdict fidèle, 429, plafond de slots, migrations CLI, rang de carrière public

> Créé le 2026-09-16 (soir), révisé le même soir après revue fraîche (`plan-review`, 8 P1 / 7 P2
> intégrés). Branche : `wt/sync-robustesse`. Worktree dédié :
> `C:/Users/Guillaume/Downloads/Scripts/LevelUp-wt-sync-robustesse`. Base : `feat/v75` @ `ab7fc5695`.
> **Push sur `main` = déploiement prod : interdit.** Push de `feat/v75` : décision utilisateur.
>
> **Contrat d'exécution : skill `plan-execution` fait foi.** Ordre strict, une étape à la fois,
> aucun report d'étape exécutable, chaque item statué `[x]` / `[~]` (référence) / `[!]`
> (justification écrite), zéro fix hors périmètre (découvertes en §10). Vérifier sur pièces
> avant de coder et avant de cocher : les numéros de ligne datent du 2026-09-16 sur `ab7fc5695`.
>
> **Interdits absolus** : aucun démarrage de serveur, aucune ouverture d'une base sous `data/`
> (le serveur local tient les bases ; mono-process ADR 0013). Tests sur `:memory:` /
> `t.TempDir()`, build, vet, grep. Aucun `time.Sleep` réel dans un test : les attentes passent
> par le seam nommé en 1.1.
>
> **Règle de baseline (dans le contrat, leçon du 2026-09-16)** : tout `func Test*` renommé ou
> supprimé par le lot est retiré de `.ai/baselines/tests_pre_migration.jsonl` dans le MÊME commit,
> et le retrait est daté dans l'en-tête de `scripts/check_test_baseline.sh`. Vérification
> reproductible (paires `Package::Test`) :
>
> ```
> extrait() { sed -n 's/.*"Package":"\([^"]*\)".*"Test":"\([^"]*\)".*/\1::\2/p' "$1" | sort -u; }
> comm -3 <(git show <base>:.ai/baselines/tests_pre_migration.jsonl | extrait /dev/stdin) <(extrait .ai/baselines/tests_pre_migration.jsonl)
> ```
> Le résultat (paires retirées) est copié dans « Avancement ».

---

## 1. Constat (première exécution réelle du sync par le pool, Nuzzles, 2026-09-16)

**C-A — Un `429` termine la passe qui se déclare réussie.** 6 slots × 3 req/s : `HTTP 429` sur
`GetMatchHistory(start=225)`. Le pool met le slot fautif en cooldown AIMD et `acquireAnyPublic`
le saute (il ne bloque jamais : quand TOUS les slots sont en pause il rend immédiatement
`fmt.Errorf("pool: aucun slot sain disponible (PolicyAnyPublic)")`, `pool.go:269`, enveloppé par
`doPublic` en `"pooled: Acquire failed: %w"`, `pooled_client.go:146-162`). Mais
`paginateAndPersistHistory` (`internal/sync/engine.go:245-251`) fait `result.AddWarning` puis
`break` sur TOUTE erreur d'historique, et `SyncResult.Status()` (`internal/domain/sync.go:93-101`)
ne regarde que `Errors` → `sync full OK … status=success inserted=0`. Le client HTTP rend
`*haloclient.HTTPError{StatusCode: 429, RetryAfter}` sans retenter (`halo_client_http.go:296-300`,
« le caller retentera ») — le caller ne retente pas. `errors.As` fonctionne : `GetMatchHistory`
enveloppe en `%w` (`halo_client.go:258`), `doPublic` et `cachedHaloClient` passent l'erreur telle
quelle. Les 503 sont DÉJÀ retentés en interne par `doGet` (`halo_client_http.go:293-310`).
Côté serveur (`sync_handler.go:516-520/599-603`, `auto_sync_run.go:379-389`) : le job reste
`succeeded`, `Errors > 0` → WARN « erreurs partielles » + `detail.SyncStatus` exposé au web
(`types.ts sync_status`) ; aucune notification d'erreur — passer les erreurs d'historique en
`Errors` les rend VISIBLES au monitoring sans basculer aucun job en échec.

**C-B — `--token-pool-size N` compte les sources, pas les slots sains.** `NewPool`
(`internal/platform/auth/pool/pool.go:137-183`) tronque `sources[:MaxSize]` AVANT de résoudre ;
avec `1`, la première source du scan (Chocoboflor, révoquée) est la seule tentée → « aucun slot
créé ». L'ordre du scan n'est pas stable (map).

**C-C — `sync-full` / `sync-delta` n'appliquent aucune migration.** `cmd/levelup/cmd_sync.go` :
zéro appel ; seuls le boot du serveur (`cmd/server/main.go:1686`) et les `backfill`
(`cmd/levelup/cmd_backfill.go:375 applyMigrationsOnDB`, qui passe par `duckdbpkg.OpenReadWrite`,
cache `rw:`, mono-connexion — pas un « bare connect ») le font. Vécu : élargissement livré mais
non appliqué pendant 83 min de sync → 2 matchs rejetés une seconde fois.

**C-D — `PolicyPinnedPlayer` sur `GetCareerRank` repose sur une prémisse fausse.** Mesuré le
2026-09-16 (sonde non versionnée, serveur arrêté) : `GET /careerranks` pour un xuid TIERS avec
trois prêteurs différents (JGtm, DankerGlue, Trimbutton) → `200` ; rang ET XP identiques à
l'appel du propriétaire (JGtm : `rank=202 xp=2555` vu par DankerGlue et Trimbutton ; Nuzzles :
`rank=272 xp=0` — 272 est le rang maximal, l'XP nulle est sa vraie valeur). L'endpoint est
entièrement public. Aucun appelant de production n'utilise `GetCareerRank` (`career.go:56`
`//nolint:unused`, tests seulement ; `fetch_cache.go:124` passe-plat ; `v2/fetch_player.go:43`
commentaire). `pinnedGamertag` / `pinnedXUID` du `PooledHaloClient` ne sont lus QUE par
`GetCareerRank` (`pooled_client.go:33-35, 50-59, 300-308`). Restent réellement pinned, HORS
périmètre : le cron de personnalisation Spartan (`spartan_customization_cron.go:338`,
`/customization/appearance` 403 pour un tiers — mesuré, `career_live_target.go:24`) et le
live-sync Halo 5 (`halo_5/livesync/acquire.go:41`).

**C-E — Deux résidus du post-sync CLI.** (1) `engine.go:494` ouvre `metadata.duckdb` par
`OpenReadForQuery` (clé `ro:` en CLI), handle tenu par `defer` jusqu'à la fin de `run()`
(`:500-503`) et LU pendant le post-sync (`engine_postsync.go:337-342`, `assetnames_wiring.go:49`,
`citations_backfill.go:422`, `convergence.go:657`) ; puis `engine_postsync_csr.go:305
OpenReadWriteShared` (clé `rw:`) et `runAchievementsSync:170` (même défaut) → « different
configuration ». (2) `resolveAchievementsAccessToken` (`engine_postsync_csr.go:327-329`) résout le
token PROPRE du joueur pour les succès Xbox Live ; le store rend le sentinelle
`auth.ErrUserTokensNotFound` (`multi_user_token_store.go:111`), déjà journalisé en Debug à
`engine_postsync_csr.go:136`, MAIS le helper `access_token_store_first.go:56` émet un
`ErrorContext` avant — un ERROR par passe pour un cas normal (profil suivi sans token).

**C-F — Hygiène.** Aide du drapeau `--gamertag` de `backfill-killsource --online`
(`cmd_backfill_killsource.go:152`) en retard sur le comportement ; 12 fichiers de test déclarent
`match_registry` avec `SMALLINT`, dont le test de la migration d'élargissement qui DOIT le faire
(DDL legacy) ; le ratchet `titleseams_wired_test.go` ne voit que l'import direct de
`internal/sync` (10 binaires transitifs : `backfill-world-player-stats`, `h5-appearance-backfill`,
`h5-csr-backfill`, `h5-events-backfill`, `h5-kill-kind-backfill`, `h5-roster-refetch`,
`levelup-titles`, `openapi-gen`, `populate-playlists-catalog`, `replay-worker`).

## 2. Décisions tranchées

- **D1** — Rejeu d'une page d'historique : (i) `*haloclient.HTTPError` avec `StatusCode == 429`
  → rejouer IMMÉDIATEMENT la même page (le pool saute le slot en cooldown ; avec un seul slot le
  rejeu retombe sur le cas ii) ; (ii) erreur d'acquisition « aucun slot sain » → attendre
  `min(pool.GlobalCooldown, 60 s)` puis rejouer ; 3 tentatives au total ; les 503 ne sont PAS
  rejoués ici (déjà retentés par `doGet`). Échec définitif ou toute autre erreur →
  `result.AddError` (plus `AddWarning`), `break` ; `Status()` inchangé (`partial_success` /
  `failure`) ; la CLI rend un code de sortie ≠ 0 avec « historique interrompu à start=N ».
  L'attente passe par un seam de paquet `var historyRetrySleep = time.Sleep` (tests sans sommeil).
  L'erreur du pool devient typée : `pool.ErrNoHealthySlot` (sentinelle) rendue par
  `acquireAnyPublic`, enveloppée `%w` par `doPublic`.
- **D2** — `MaxSize` plafonne les slots SAINS : sources triées par gamertag, parcours complet,
  `break` quand `len(slots) == MaxSize` (`> 0`) ; `0` = toutes les saines.
- **D3** — `sync-full` / `sync-delta` (mono et `--all`) appliquent les migrations shared du titre
  avant de synchroniser via `migration.RunForTitleDB(db, titleSlug, TargetShared)` (même
  ouverture que `applyMigrationsOnDB`). La base joueur reste à `EnsurePlayerSchema`.
- **D4** — `GetCareerRank` passe en `PolicyAnyPublic`. `ErrNoPinnedToken`, la branche de
  dégradation de `syncCareerRank`, les champs `pinnedGamertag` / `pinnedXUID` et les deux
  paramètres correspondants de `NewPooledHaloClient` disparaissent (0 code mort) ; les six
  appelants de production et les mocks sont retouchés. Cron Spartan et live-sync H5 inchangés.
- **D5** — Post-sync CLI : (1) `engine.go:494` ouvre metadata par `OpenReadWriteShared` (clé
  `rw:`, refcount : dans le serveur c'est le handle déjà en cache, en CLI une instance unique)
  et `e.metaDB` sert TOUT le run ; le seed de catalogue (`engine_postsync_csr.go:305`) et
  `runAchievementsSync:170` réutilisent ce handle au lieu d'ouvrir. Les leases (`KindPlayer`
  `engine.go:449`, `KindMetadata` `:297`) ne bougent pas — ouvrir un handle n'est pas prendre un
  lease. Jamais l'autre variante (libérer avant le post-sync : elle éteint quatre étapes en
  silence). (2) `access_token_store_first.go:56` : sur `errors.Is(err, auth.ErrUserTokensNotFound)`
  retourner SANS `ErrorContext` ; `engine_postsync_csr.go:136` : Debug → `InfoContext` « succès
  Xbox Live sautés : aucun token propre pour ce profil ». Un seul journal par passe.
- **D6** — Hygiène : aide du drapeau alignée sur `docs/COMMANDS.md` ; fixtures `SMALLINT`
  alignées SAUF les DDL legacy volontaires, marquées `// match_registry-ddl: legacy — <raison>`
  et exclues du ratchet ; ratchet étendu aux `*_test.go` non marqués, colonnes communes
  seulement, sans plancher ; ratchet `titleseams` sur les dépendances transitives par UN SEUL
  `go list -f '{{.ImportPath}} {{.Deps}}' ./cmd/...` (2,7 s mesuré, Windows OK).
- **D7** — Hors périmètre, refusés : rotation des RT par la CLI pendant qu'un serveur tourne
  (note d'exploitation) ; `snapshot ready DE FORCE` (voulu, grâce 60 j) ; trois RT révoqués
  (ADR 0023) ; faire basculer un job serveur en échec sur `Errors > 0` (comportement actuel
  documenté en C-A, décision produit distincte).

## 3. Étape 0 — Préparation (rapide)

- [ ] 0.1 Worktree `../LevelUp-wt-sync-robustesse` sur `wt/sync-robustesse` (créé par le pilote) ;
      `git branch --show-current`. Pas de `npm install`.
- [ ] 0.2 Lire `CLAUDE.md`, ce plan, `internal/platform/auth/pool/README.md`,
      `.ai/PLAN_SYNC_POOL_SEAMS_SCHEMA_2026-09-16.md` §8, skills `plan-execution`, `arch-rules`.
- [ ] 0.3 Baseline : `cd apps/go-api && go build ./... && go test ./cmd/levelup/... ./internal/platform/auth/pool/... ./internal/sync/ -count=1 -timeout 30m` → 0 (noter la durée).

**Gate G0** : branche correcte, baseline verte notée dans « Avancement ».

## 4. Étape 1 — Verdict fidèle et rejeu du 429 (C-A, D1 ; moyen, cœur du plan)

- [ ] 1.1 `internal/platform/auth/pool/pool.go:269` : `var ErrNoHealthySlot = errors.New("pool: aucun slot sain disponible (PolicyAnyPublic)")`,
      rendu par `acquireAnyPublic` ; `pooled_client.go:146-162` `doPublic` : `fmt.Errorf("pooled: Acquire failed: %w", err)`
      (vérifier que c'est déjà `%w`). Test : `errors.Is(err, pool.ErrNoHealthySlot)` traverse `doPublic`.
- [ ] 1.2 Nouveau fichier `internal/sync/engine_history_retry.go` (le fichier `engine.go` dépasse
      500 L : ne pas l'accroître) : `var historyRetrySleep = time.Sleep` ;
      `func (e *SyncEngine) fetchHistoryPage(ctx context.Context, in *historyPaginationInputs, start int) ([]domain.MatchHistoryEntry, error)`
      (≤ 5 paramètres — `historyPaginationInputs` existe, `engine.go:206-213` ; vérifier le type
      des entrées rendu par `GetMatchHistory`) : boucle de 3 tentatives ; `*haloclient.HTTPError`
      429 → rejeu immédiat ; `errors.Is(err, pool.ErrNoHealthySlot)` → `historyRetrySleep(min(cooldown, 60 s))`
      puis rejeu, `cooldown` = `pool.GlobalCooldown` si accessible depuis le client (sinon 30 s
      constante nommée) ; autre erreur → retour immédiat. `slog.WarnContext` à chaque rejeu
      (`start`, `attempt`, `cause`).
- [ ] 1.3 `engine.go:245-251` : appel remplacé par `fetchHistoryPage` ; en cas d'erreur définitive
      `result.AddError(fmt.Sprintf("historique interrompu à start=%d: %v", start, err))` puis
      `break`. `SyncResult.Status()` non modifié.
- [ ] 1.4 `cmd/levelup/cmd_sync.go` : fonction pure `reportSyncResult(w io.Writer, mode string, r *domain.SyncResult) error`
      dans `cmd/levelup/sync_report.go` : imprime `sync <mode> <STATUT>: …` (+ `first_error=` si
      `Errors` non vide) et rend une erreur quand `r.Status() != "success"`. Les quatre runners
      (`runSyncDelta`, `runSyncFull`, `runSyncDeltaAll`, `runSyncFullAll`) l'appellent ; en `--all`
      un joueur en `failure` compte dans `failed`.
- [ ] 1.5 Tests (aucun sommeil réel : `historyRetrySleep` remplacé dans le test, durées demandées
      enregistrées) : `engine_history_retry_test.go` — (a) 429 puis 200 → page obtenue, 0 attente ;
      (b) `ErrNoHealthySlot` puis 200 → une attente = `min(cooldown, 60 s)`, page obtenue ; (c) 429
      × 3 → erreur, `AddError`, `Status()=failure` si rien inséré (via la boucle de pagination avec
      client factice) ; (d) 500 → erreur immédiate, 0 rejeu, 0 attente ; (e) 503 → pas rejoué ici.
      `sync_report_test.go` — `success` → nil ; `failure` → erreur et texte contenant
      `first_error`. Chaque test rougit si la correction est retirée (le noter dans le test).
- [ ] 1.6 Baseline : paires renommées/supprimées retirées (commande du contrat) — a priori aucune.

**Gate G1** : `go test ./internal/sync/ -run 'History|Pagin|Retry' -count=1 -timeout 30m` → 0 ;
`go test ./cmd/levelup/ ./internal/platform/auth/pool/ -count=1` → 0 ; `go vet ./internal/sync/ ./cmd/levelup/ ./internal/platform/auth/pool/` → 0 ;
`wc -l internal/sync/engine.go` ≤ valeur de base (`git show ab7fc5695:apps/go-api/internal/sync/engine.go | wc -l`).

## 5. Étape 2 — Plafond de slots sains (C-B, D2 ; rapide)

- [ ] 2.1 `pool.go:137-183` : `sort.Slice(sources, by Gamertag)` ; boucle sur TOUTES les sources,
      `continue` sur échec (journal inchangé), `break` quand `MaxSize > 0 && len(slots) == MaxSize`.
      « aucun slot créé » conservé pour 0 slot.
- [ ] 2.2 Doc `PoolOptions.MaxSize` (`pool.go:114`), aide `--token-pool-size` (`cmd_sync.go`),
      `docs/COMMANDS.md` + `docs/FR/COMMANDS.md` : « nombre maximal de slots SAINS ».
- [ ] 2.3 Tests `pool_test.go` : sources [révoquée, saine, saine] (résolveur factice) — `MaxSize 1`
      → 1 slot = première saine par ordre alphabétique ; `2` → 2 ; `0` → toutes ; même résultat
      quel que soit l'ordre d'entrée.

**Gate G2** : `go test ./internal/platform/auth/pool/... -count=1` → 0.

## 6. Étape 3 — Rang de carrière public (C-D, D4 ; moyen)

- [ ] 3.1 `internal/sync/pooled_client.go` : `GetCareerRank` via `doPublic` (`PolicyAnyPublic`) ;
      supprimer `ErrNoPinnedToken` (`:287-293`), les champs `pinnedGamertag` / `pinnedXUID`
      (`:33-35`), les paramètres correspondants de `NewPooledHaloClient` (`:50-59`) et le
      commentaire `:24`. `grep -rn "ErrNoPinnedToken\|pinnedGamertag\|pinnedXUID" apps/go-api` → vide.
- [ ] 3.2 Appelants de `NewPooledHaloClient` retouchés (signature sans pin) : `cmd/levelup/pool_engine.go:96,162`,
      `cmd/server/main.go:2179`, `cmd/server/sync_v2_wiring.go:135,337`,
      `internal/scheduler/auto_sync_engine.go:67` ; mocks `mockPool` (`pooled_client_test.go`),
      `poolUnSlot` (`cmd/levelup/pool_engine_test.go`) ; doc de `newPooledEngine`
      (`pool_engine.go`, « ne sert qu'à épingler ») réécrite ; `cmd/server/main.go:753`
      commentaire mis à jour. Vérifier par `go build ./...` qu'aucun autre appelant n'existe.
- [ ] 3.3 `internal/sync/career.go:25-60` : retirer la branche `errors.Is(err, ErrNoPinnedToken)`
      et la doc « seul endpoint privacy-gated » ; la validation du xuid reste.
- [ ] 3.4 Tests : supprimer `career_no_pinned_token_test.go` (`TestSyncCareerRank_SansTokenPropre_DegradeSansEchouer`,
      `TestSyncCareerRank_AutreErreur_Remontee` — le second déplacé dans `career_integration_test.go`
      s'il teste encore quelque chose) et `pool_engine_test.go TestRangDeCarriereSeDegradeSeul` ;
      `pooled_client_test.go` : `_PinnedToken` et `_NoPinnedToken` remplacés par
      `TestPooledHaloClientGetCareerRank_AcquiertEnPublic` (le `mockPool` gagne un champ
      `lastPolicy` ; assertion = `PolicyAnyPublic`, pas de réponse HTTP simulée) ; `_PoolError`
      conservé. **Baseline** : paires supprimées/renommées retirées (`_PinnedToken`,
      `_NoPinnedToken`, `_PoolError` si renommé, `TestSyncCareerRank_*` supprimés — ceux du
      16/09 n'y sont pas), en-tête daté.
- [ ] 3.5 Docs : `internal/platform/auth/pool/README.md` (tableau des appelants : `GetCareerRank`
      → `PolicyAnyPublic` ; pinned = cron Spartan + live-sync H5 ; mesure du 2026-09-16 consignée :
      trois prêteurs, rang et XP identiques à l'appel du propriétaire, Nuzzles `272/0` = rang
      max), `docs/COMMANDS.md` + `docs/FR/COMMANDS.md` (paragraphe carrière), `cmd/levelup/pool_engine.go:5-10`,
      `cmd/levelup/cmd_sync.go` (deux commentaires), `.ai/PLAN_SYNC_POOL_SEAMS_SCHEMA_2026-09-16.md`
      §8 (ligne « prémisse contredite » → « tranchée par mesure, D4 du plan robustesse »).

**Gate G3** : `grep -rn "ErrNoPinnedToken\|pinnedGamertag\|pinnedXUID" apps/go-api` → vide ;
`grep -rn "PolicyPinnedPlayer" apps/go-api --include=*.go | grep -v _test | grep -v "auth/pool/" | grep -v "^\s*//" | grep "Acquire("`
→ exactement `spartan_customization_cron.go` et `halo_5/livesync/acquire.go` ;
`go build ./... && go test ./internal/sync/ ./cmd/levelup/ ./internal/scheduler/ -count=1 -timeout 30m` → 0.

## 7. Étape 4 — Cohérence CLI / serveur (C-C, C-E, D3, D5 ; moyen)

- [ ] 4.1 `cmd/levelup/migrations_cli.go` : `applyMigrationsOnDB` déplacé depuis
      `cmd_backfill.go:375` (un seul exemplaire) ; nouveau `applySharedMigrationsForTitle(cfg, titleSlug) error`
      = ouverture identique + `migration.RunForTitleDB(db, titleSlug, migration.TargetShared)`
      (pas `RunForDB`, qui force `DefaultSlug`) ; chemin via `PathResolver.SharedDBPath(titleSlug)`.
      Appelé en tête des quatre runners de sync, `slog.InfoContext` « migrations shared : N
      appliquées » (0 = à jour).
- [ ] 4.2 `internal/sync/engine.go:482-503` : `OpenReadForQuery` → `duckdbpkg.OpenReadWriteShared`
      (même durée de vie, même `defer`) ; `engine_postsync_csr.go:295-312` (seed de catalogue) et
      `runAchievementsSync:170` : utiliser `e.metaDB` s'il est non nil, sinon ouvrir comme avant.
      Le WARN « metadata inaccessible » ne doit plus apparaître sur une passe CLI normale.
      Vérifier sur pièces que les quatre lecteurs de `e.metaDB` (C-E) ne posent pas de
      contrainte RO.
- [ ] 4.3 `internal/platform/auth/access_token_store_first.go:50-60` : `errors.Is(err, ErrUserTokensNotFound)`
      → retour sans `ErrorContext` ; `internal/sync/engine_postsync_csr.go:133-138` : Debug →
      `slog.InfoContext(ctx, "post-sync: succès Xbox Live sautés — aucun token propre pour ce profil", "gamertag", …)`.
      Un seul journal par passe.
- [ ] 4.4 Tests : `migrations_cli_test.go` — `applySharedMigrationsForTitle` sur une base
      temporaire (`t.TempDir()`) crée `match_registry` avec `team_0_score INTEGER` (preuve que
      les steps title-owned jouent : `cmd/levelup/main.go:55` câble `titleseams.RegisterAll`) ;
      l'ORDRE « migrations avant `RunFull` » est vérifié sur pièces et noté `[~]` (pas de seam
      de moteur à inventer). `access_token_store_first_test.go` — store vide → `("", nil)` et
      aucune ligne ERROR capturée (handler slog de test). 4.2 : `go test -tags=integration -p 1 ./internal/sync/ -run 'Catalog|Achievements|PostSync'`.
- [ ] 4.5 Baseline : paires renommées/supprimées retirées si besoin.

**Gate G4** : `go test ./cmd/levelup/ ./internal/sync/ ./internal/platform/auth/ -count=1 -timeout 30m` → 0 ;
`go test -tags=integration -p 1 ./internal/sync/ -timeout 30m` → 0 ; `grep -rn "func applyMigrationsOnDB" cmd/`
→ une seule définition ; `grep -rn "OpenReadForQuery" internal/sync/engine.go` → vide.

## 8. Étape 5 — Hygiène (C-F, D6 ; rapide)

- [ ] 5.1 `cmd/levelup/cmd_backfill_killsource.go:152` : aide de `--gamertag` = « joueur dont
      les films sont traités (les plus récents d'abord) ; les tokens viennent du pool ».
- [ ] 5.2 Fixtures : `internal/migration/steps_shared_rebuild_match_participants_test.go:106-107`
      et `internal/sync/art_rebuild_regression_test.go:110-111` → `INTEGER`. Le test de
      l'élargissement (`steps_shared_core_widen_test.go`) et tout autre test qui DOIT porter la
      DDL legacy reçoivent le marqueur `// match_registry-ddl: legacy — <raison>` sur la ligne
      précédant la DDL. Ratchet `internal/archlint/match_registry_ddl_types_test.go` étendu :
      pour chaque `*_test.go` contenant `CREATE TABLE match_registry` SANS marqueur, les colonnes
      communes avec la DDL de production ont le même type ; aucun plancher de colonnes.
      Lister les 12 fichiers touchés dans « Avancement » (alignés vs marqués).
- [ ] 5.3 `internal/archlint/titleseams_wired_test.go` : critère = dépendances transitives
      (`go list -f '{{.ImportPath}} {{.Deps}}' ./cmd/...`, une seule exécution, `t.Skip` si `go`
      absent du PATH avec message), allowlist toujours vide ; les 10 binaires listés en C-F
      reçoivent `titleseams.RegisterAll("")` s'ils rougissent.
- [ ] 5.4 `docs/COMMANDS.md` FR + EN, note d'exploitation : « la CLI de sync tient la base
      partagée en écriture : serveur arrêté ; ne pas faire tourner les tokens du parc pendant
      qu'un serveur tourne ».

**Gate G5** : `go test ./internal/archlint/ ./internal/migration/ ./internal/sync/ -run 'Ddl|DDL|Seams|Titleseams|ArtRebuild|RebuildMatchParticipants|Widen' -count=1 -timeout 30m` → 0 ;
`go build ./cmd/...` → 0.

## 9. Étape 6 — Clôture

- [ ] 6.1 Gates complets : `go build ./... && go vet ./... && go test ./... -count=1 -timeout 30m` → 0 ;
      `go test -tags=integration -p 1 ./... -timeout 30m` → 0 ; `gofmt -l ./cmd ./internal` → vide.
- [ ] 6.2 Baseline : commande du contrat exécutée ; paires retirées listées dans « Avancement » ;
      en-tête de `scripts/check_test_baseline.sh` daté.
- [ ] 6.3 `.ai/thought_log.md` : entrée `[2026-09-1x]` Complété.
- [ ] 6.4 Revue adversariale (pilote, 2 relecteurs : pool/moteur ; post-sync/migrations/hygiène)
      et CI : `[~]`. Push et fusion : décision utilisateur.

Journal de phase : section « Avancement » en fin de fichier. Reprise : la lire, puis
`git log --oneline -10` dans le worktree.

---

## 10. Découvertes hors périmètre (ne pas traiter ici)

- Le serveur ne bascule jamais un job de sync en échec : `sync_handler.go:516-520/599-603` marque
  `succeeded` quoi qu'il arrive ; `auto_sync_run.go:379-389` journalise WARN et expose
  `sync_status` au web. Décision produit distincte (D7).
- `halo_5/livesync/acquire.go:41` : `PolicyPinnedPlayer` pour le live-sync Halo 5 — légitime ou
  même prémisse que C-D ? Hors titre, hors plan.
- `HaloAPIClient.GetCareerRank` avale déjà 401/403 (`doPlayerGatedGet`) : à revoir si un
  endpoint réellement gated y passe un jour.

## Avancement

(vide — plan révisé, non exécuté au 2026-09-16 19:xx)
