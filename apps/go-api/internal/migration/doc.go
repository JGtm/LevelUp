// Package migration — registre ordonné de migrations DuckDB par cible
// (metadata, matchs partagés, données joueur, données sociales).
//
// Contient les rebuilds append-only (ADR 0026 : drop des index ART, recréation
// des vues `<table>_latest`), les seeds de référentiels (career_ranks, citations,
// prestige, milestones) et les réparations de PK legacy. Les migrations sont
// idempotentes et appliquées au boot ; l'ajout d'une table append-only suit la
// recette ADR 0026.
package migration
