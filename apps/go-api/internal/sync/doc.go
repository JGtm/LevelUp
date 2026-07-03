// Package sync — pipeline de synchronisation des matchs Halo (Collect→Persist
// anti-ART, ADR 0019).
//
// Orchestre le fetch de l'API Halo, les transforms vers les types canonical, les
// écritures per-match via BatchBuilder/persist (INSERT-only, un seul writer sous
// lease dblease mono-process), les backfills (citations, CSR, weapons, PSA) et les
// runners post-sync (LUSR v2, sessions, progression). Le nouveau code d'orchestration
// va dans sync/v2 (cycle orchestrator, ADR 0027) ; la racine sync/ est gelée.
package sync
