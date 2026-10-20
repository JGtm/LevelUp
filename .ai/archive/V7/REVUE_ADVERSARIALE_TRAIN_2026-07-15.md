# Revue adversariale du train de merge — corrections (2026-07-15)

Revue adversariale multi-agents (6 angles, 3 juges par trouvaille) sur le diff cumulé
du train (`origin/main...test/e2e-fixture-synthetique`, 96 fichiers). 14 défauts
confirmés sur 16 uniques. Branche des corrections : `fix/revue-adversariale`.

Statut : **10 corrigés, 4 écartés** (justifiés). Tous les gates verts (voir §Gates).

## Corrigées (10)

| # | Sévérité | Fichier | Fix |
|---|---|---|---|
| 1 | majeur (ART) | `internal/ops/seed_demo_synthetic_player.go:101` | `INSERT OR REPLACE INTO sync_meta` → `INSERT` pur (DB vierge, aucune collision). Débloque `TestNoARTPatternsOnProtectedTables` (CI rouge garantie sinon). Vérifié : `go test -tags=integration -run '^TestNoARTPatternsOnProtectedTables$' ./internal/sync/` = ok. |
| 2 | majeur (sécu) | `internal/platform/auth/sisu_provider.go:193` | Suppression du slot de flow GLOBAL `p.current`. Le contexte SISU (kp/deviceToken/sessionID/codeVerifier) est désormais porté PAR le `sisuDeviceFlow` (per-flow) et complété via `sisuDeviceFlow.ExchangeFlow` (nouvelle interface `auth.FlowExchanger`, routée par `handlers/auth.go` `exchangeAfterAcquire`). `Exchange` devient TOUJOURS stateless (pool/SSO web). Plus aucun état partagé → plus de course inter-appelants. Single-flight `waitDeviceFlowReady` préservé (le stub ne porte pas de FlowExchanger → fallback `provider.Exchange`). Test de régression `TestSISUProvider_PerFlowContextIsolation`. |
| 3 | majeur (test) | `internal/ops/seed_demo_synthetic_integration_test.go:97` | `TestSeedDemoSynthetic_Deterministic` compare désormais les DONNÉES RÉELLES (dump table-par-table trié, ligne à ligne) des 17 tables écrites par le seeder, sur les 6 DBs, entre deux runs. A débusqué 3 vraies non-déterminations : `player_csr_snapshots.fetched_at`, `match_citations.written_at`, `weapon_kills.written_at` (DEFAULT clock non ancrés) — tous ancrés sur `synthAnchor`. |
| 4 | mineur | `internal/ops/data_quality.go:213` | `metaTableExists` → `(bool, error)` ; l'erreur d'introspection est remontée par les callers (`listUntranslatedModes`, `listOrphanPlaylists`) au lieu d'être avalée en « table absente » (faux vert). Le `slog.Debug` ne ment plus (loggé uniquement quand la table est réellement absente). |
| 5 | mineur | `internal/ops/disk_watch.go:50` | Débounce anti-oscillation : les AGGRAVATIONS notifient immédiatement, les AMÉLIORATIONS (dé-escalade/rétablissement) sont confirmées sur `diskConfirmTicks=2` observations consécutives. Une oscillation ±0,5 % autour d'un seuil n'émet plus de spam. `ShouldNotifyDisk` étendu + 3 nouveaux cas de test (oscillation ok↔warn, warn↔critical, amélioration interrompue) ; recovery devient 2-ticks. |
| 6 | mineur | `apps/web/src/features/explorer/ExplorerBriefing.logic.ts:54` | Suppression du helper mort `perfTierLabelKey` (doublon de `PERF_TIER_KEY`, jamais appelé en prod) + son bloc de test (règle 0 code mort). |
| 7 | mineur | `internal/analysis/indicators.go:37` | Migration de la copie inlinée `explorer_target_stats.go:69` vers le helper canonique `AggregateKDA` + garde-rail grep `TestNoInlinedAggregateKDAInAnalysis` (interdit la réinline dans le package `analysis`). La copie `games/halo_5/mapping_servicerecord.go:88` n'est PAS migrée : `games/` ne doit pas importer `analysis` (inversion de couche) — laissée telle quelle. |
| 8 | mineur | `internal/analysis/breakdown/compare_by_key.go:36` | README catalogue `breakdown/README.md` mis à jour (`CompareByKey` / `KeyedAggregate` / `KeyedDelta` + exemple non-map). |
| 9 | mineur (sécu) | `internal/notify/discord.go:213` | `sanitizeSendError` expurge l'URL du webhook (token = secret d'écriture, dans le path) de `*url.Error` avant tout log ; appliqué aux 2 sites HTTP (`Do`, `NewRequest`). Test `TestSanitizeSendError_RedactsWebhookToken`. |
| 10 | mineur | `apps/web/src/features/match-view/_momentum.ts:79` | Le front CLAMPE désormais les events au-delà du dernier bin dans le dernier bin (parité stricte avec `tug_of_war.go`), et ignore les temps avant le premier bin (parité `TimeMS < 0`). Cas de test (d) ajusté (clamp) + (d2) ajouté (avant premier bin ignoré). |

## Écartées (4) — non corrigées

- **populate-assets locks DuckDB bruts** (majeur, go-correctness — `cmd_populate_assets.go:92`). RÉFUTÉE par 1 juge sur 3 : le lock cross-process est le modèle opérationnel documenté du projet pour les one-offs CLI (`docs/RUNBOOK_OPS_DUCKDB_CLI_TOOLS.md` : serveur ARRÊTÉ d'abord, fail-fast naturel, `OpenReadForQuery` inapplicable car metadata exige du RW). But de la migration = PRÉSENCE du binaire dans l'image prod, pas l'exécution concurrente. Correctif appliqué uniquement au résidu mineur signalé : la ligne imprécise « `docker compose exec` » du `thought_log` (2026-07-13) est corrigée en « serveur arrêté d'abord ».
- **disk notify state avancé sur échec d'envoi** (mineur — `registry_monitoring_diskwatch.go:80`). Design délibéré et testé : le canal PRIMAIRE de l'alerte est la détection persistée + badge admin (insensible à Discord) ; `notify` est documenté failsafe ; aucune règle n'exige la garantie de livraison. Une aggravation retente l'envoi immédiatement. Écartée.
- **specs e2e stale en skip permanent** (mineur — `demoData.ts:80`). Les surfaces testées sont VIVANTES ; seul le chemin d'accès des specs a dérivé → remède = réécriture, déjà tracké au backlog (`thought_log`, `ETAT_CONSOLIDE`, `apps/web/e2e/README.md`). Skip documenté conforme à `plan-execution` n°5 (noter la découverte, ne pas la traiter hors périmètre). Écartée.
- **garde-rail reachability device-code inerte** (mineur — `xbox_device_code_reachability_integration_test.go:43`). Design opt-in délibéré et justifié (double gate tag+env : zéro flake CI, zéro dépendance réseau au boot). Aucune règle n'exige l'invocation auto en CI ; la régression n'est plus silencieuse (lots A/C2 : erreur surfacée UI + `slog.ErrorContext`). Correctif appliqué au résidu : la commande manuelle exacte est ajoutée à `docs/RUNBOOK_DEPLOY_CHECKLIST.md` (post-deploy). Écartée.

## Défauts préexistants du train corrigés au passage (bloquaient le gate)

Le chantier fixture synthétique avait introduit le seeder `seed_demo_synthetic*.go` SANS
mettre à jour 4 garde-rails file-level (dont l'ART, seul trouvé par la revue = item 1).
Les 3 autres rendaient `go test ./...` ROUGE et sont corrigés pour satisfaire les gates :

- `TestNoUnauthorizedSharedSocialMention` : ajout de `seed_demo_synthetic.go` + `_shared.go` à `sharedSocialFilesWhitelist` (créent shared_social.duckdb migré vide, pas d'ATTACH/INSERT RW).
- `TestNoNewHalowaypointLiteral` : `seed_demo_synthetic_player.go` allowlisté (URL blob démo FACTICE, jamais fetchée).
- `TestNoNewRawOutcomeLiteral` : littéraux `outcome == 2/3` → constantes `domain.OutcomeWin/Loss` dans `seed_demo_synthetic.go` + `_shared.go`.

## Gates (2026-07-15)

- `go build ./...` = 0 · `go vet ./...` = 0
- `go test ./...` = vert · `go test -tags=integration -p 1 ./...` = vert
- `golangci-lint run --new-from-rev=test/e2e-fixture-synthetique` = 0 issue
- Front : `tsc --noEmit` = 0 · `eslint` (fichiers touchés) = 0 · `vitest run` = 255 fichiers / 2159 tests passés, 14 skipped
