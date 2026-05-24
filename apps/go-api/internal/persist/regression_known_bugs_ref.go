// Package persist — regression_known_bugs_ref.go : documentation des
// references croisees vers les tests existants qui couvrent les regressions
// ART connues (Bug #1 different configuration, LUSR + CSR ART corruption).
//
// Ce fichier ne contient PAS de code executable — uniquement de la doc.
// Reference J.1 + J.2 du PLAN_FIX_SYNC_TESTS_STRATEGY_2026-05-24 :
// l'engagement "aucun test laisse de cote" exige que les doublons assumes
// (J.1, J.2) aient leur lien de resolution explicite pour traceability.
//
// ─────────────────────────────────────────────────────────────────────────
// J.1 — Regression "different configuration" / ART player_match_enrichment
// ─────────────────────────────────────────────────────────────────────────
//
// Cible : detecter une reapparition du bug DuckDB
//   "Connection Error: Can't open a connection to same database file with
//   a different configuration than existing connections"
//
// Tests couvrant la regression (par ordre de priorite) :
//
//   1. internal/persist/combined_persister_test.go (commit aec3e1ef)
//      - TestCombinedPersister_NoConfigurationConflict
//      - Build tags : `integration bug_repro`
//      - Lance : go test -tags 'integration bug_repro' ./internal/persist/
//      - Comportement : ROUGE si Phase 1 du plan principal a regresse.
//        VERT actuellement (Phase 1 mergee dans commit eec02eb6).
//      - Reproduit le scenario exact observe en prod 2026-05-24 sur
//        Chocoboflor + XxDaemonGamerxX.
//
//   2. internal/sync/csr_art_repro_test.go (commit b0a51b97, WIP user)
//      - TestCSR_ARTRepro_PlayerDB + TestCSR_ARTRepro_SharedDB
//      - Build tag : `art_repro`
//      - Couvre les UpsertCSRRow (csr_writes.go) et UpsertSharedCSRs
//        (csr_shared_writes.go) qui declenchaient le FATAL ART avant
//        bascule vers AppendOnlyLUSRPersister.
//      - Complementaire (target different : CSR, pas le DSN conflict).
//
// Couverture combinee : suffisante pour signaler toute regression de
// Phase 1+2 du plan principal (combined_persister cache + metadata cache).
//
// ─────────────────────────────────────────────────────────────────────────
// J.2 — Regression "ART LUSR" / Failed to delete all rows from index
// ─────────────────────────────────────────────────────────────────────────
//
// Cible : detecter une reapparition du bug DuckDB
//   "Invalid Input Error: Failed to delete all rows from index. Only deleted
//   0 out of 1 rows."
//
// Cause originale : DELETE+INSERT en TX sur match_skill_rank (LUSR) avec
// concurrent writes → corruption ART. Eradiqué Phase 2.A/B/C/D du WIP user
// par bascule vers AppendOnlyLUSRPersister (INSERT-only, schema append-only).
//
// Tests couvrant la regression :
//
//   1. internal/persist/lusr_append_only_persister_test.go (commit 00d37f68, WIP user)
//      - TestAppendOnlyLUSR_PersistAccumulates
//      - TestAppendOnlyLUSR_EmptyBatchNoOp
//      - TestAppendOnlyLUSR_RejectsEmptyMatchID
//      - TestAppendOnlyLUSR_ConcurrentInsertsNoArtCrash ← le test critique
//      - Build tag : `integration`
//      - Lance : go test -tags integration ./internal/persist/
//      - Comportement : ROUGE si quelqu'un re-introduit un DELETE/UPDATE
//        sur match_skill_rank. Le ConcurrentInserts simule 20×5×50
//        ecritures concurrentes — replique le pattern qui crashait.
//
//   2. internal/sync/art_rebuild_e2e_test.go
//      - TestE2E_ARTPipeline_BatchComputeLUSR_ProducesWrites
//      - Couvre BatchComputeLUSR avec dataset 10 matchs synthetiques :
//        run end-to-end avec le AppendOnlyLUSRPersister + verification
//        des writes match_skill_rank.
//      - Validation de la chaine compute → persist post-bascule.
//
// Couverture combinee : suffisante pour signaler toute regression de l'eradication
// ART LUSR. Le test concurrent insert garantit l'invariance "0 ART crash"
// sous charge replicate la concurrence reelle des 4 syncs scheduler.
//
// ─────────────────────────────────────────────────────────────────────────
// Maintenance
// ─────────────────────────────────────────────────────────────────────────
//
// Si un nouveau site sql.Open direct est introduit sur metadata, player ou
// shared DB → re-executer l'audit `grep -rn 'sql.Open("duckdb"'` et fixer
// via cache duckdbpkg.OpenReadOnly / OpenReadWrite / OpenReadWriteShared.
//
// Si un DELETE/UPDATE est introduit sur une table append-only (match_skill_rank,
// match_csrs, match_participants, medals_earned) → re-executer csr_art_repro_test
// avec build tag `art_repro` pour confirmer la non-regression.

package persist
