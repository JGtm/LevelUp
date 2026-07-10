// Package migration — registre ordonné de migrations DuckDB par cible
// (metadata, matchs partagés, données joueur, données sociales).
//
// Contient les rebuilds append-only (ADR 0026 : drop des index ART, recréation
// des vues `<table>_latest`), les seeds de référentiels (career_ranks, citations,
// prestige, milestones) et les réparations de PK legacy. Les migrations sont
// idempotentes et appliquées au boot ; l'ajout d'une table append-only suit la
// recette ADR 0026.
//
// ─── Politique de cycle-out des migrations (N4, 2026-07-05 — PROPOSITION) ──────
//
// Le registre accumule les steps historiques (~une centaine, appliqués au boot).
// À terme, un squash contrôlé réduit le coût de boot et la charge cognitive. La
// politique PROPOSÉE PAR DÉFAUT ci-dessous est à CONFIRMER par l'opérateur avant
// toute exécution — le squash est une opération DESTRUCTIVE sur l'historique de
// schéma (risque : perte de la reproductibilité d'un état legacy).
//
//  1. Déclenchement MANUEL uniquement (jamais au boot, jamais automatique).
//  2. Squash par VERSION MAJEURE (ex. baseline "7.0.0") : les steps antérieurs à
//     la baseline sont fusionnés en un seul step de création de schéma « à plat ».
//  3. PRÉSERVER les 10 derniers steps hors baseline (fenêtre de rollback récent).
//  4. ARCHIVER les steps squashés sous `.ai/migrations/squashed/<version>/` (audit
//     + reconstruction d'un état legacy si besoin), jamais supprimés du dépôt.
//  5. Invariant : un boot sur une base VIERGE après squash produit un schéma
//     BIT-IDENTIQUE à un boot sur l'historique complet (test de non-régression
//     obligatoire avant de committer un squash).
//
// Livrable N4 = cette politique documentée. Le squash lui-même reste un chantier
// opérateur distinct, hors de ce commit.
package migration
