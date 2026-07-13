// Package migration — registre ordonné de migrations DuckDB par cible
// (metadata, matchs partagés, données joueur, données sociales).
//
// Contient les rebuilds append-only (ADR 0026 : drop des index ART, recréation
// des vues `<table>_latest`), les seeds de référentiels (career_ranks, citations,
// prestige, milestones) et les réparations de PK legacy. Les migrations sont
// idempotentes et appliquées au boot ; l'ajout d'une table append-only suit la
// recette ADR 0026.
//
// ─── Politique de cycle-out des migrations (N4) — APPLIQUÉE le 2026-07-12 ───────
//
// 1re application : baseline PLAYER v1 (registre player, cible du 1er squash — cf.
// plan `.ai/PLAN_MIGRATION_SQUASH_BASELINE_2026-07.md` M3-M5). Politique confirmée par
// l'opérateur (GO 2026-07-12). Le registre accumulait ~une centaine de steps appliqués
// au boot ; un squash contrôlé réduit le coût de boot et la charge cognitive. Le squash
// est DESTRUCTIF sur l'historique de schéma actif → chaque invariant ci-dessous est
// NON NÉGOCIABLE.
//
//  1. Déclenchement MANUEL uniquement (jamais au boot, jamais automatique).
//  2. Squash d'un BLOC CONTIGU d'UN SEUL monde (title-owned OU global, jamais à cheval —
//     DM-4) : les steps du bloc sont fusionnés en un seul step de création « à plat »
//     (`create_baseline_<cible>_v<N>`), positionné à la place du bloc dans canonicalOrder.
//  3. PRÉSERVER les steps hors bloc (post-borne) — fenêtre de rollback récent (DM-2 ;
//     pour la baseline player v1 la borne est un PRÉFIXE, tout le reste est préservé).
//  4. ARCHIVER les steps squashés sous `.ai/migrations/squashed/<version>/` (audit +
//     reconstruction d'un état legacy), jamais supprimés du dépôt. Player v1 :
//     `.ai/migrations/squashed/player_v1/`.
//  5. Invariant BIT-IDENTIQUE (test de non-régression obligatoire, VERT avant tout
//     commit) : `SchemaSnapshot(baseline)` == `SchemaSnapshot(steps squashés)` (golden
//     figé de l'historique réel). Player v1 : `TestSquashInvariant_PlayerBaselineEquivalent`.
//  6. ÉQUIVALENCE LEDGER (DM-5) : une DB EXISTANTE portant la sentinelle (dernier step
//     squashé) est réputée porter la baseline → le runner l'enregistre sans rejouer son
//     DDL (`Migration.SupersededByAll` + `supersededBaselineSatisfied`, registry.go).
//     Tests : `internal/migration/squash_dm5_test.go`.
//
// Reste à squasher (chantiers ultérieurs, même chemin) : metadata (attention SEEDS —
// le snapshot ne capture pas les données), shared, la base sociale, halo_5.
package migration
