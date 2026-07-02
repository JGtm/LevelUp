// Package v2 — sync pipeline V2 : cycle orchestrator (ADR 0027).
//
// V2 orchestre le sync au niveau cycle (process-wide) au lieu du niveau
// player. Un seul CycleOrchestrator traite tous les joueurs en 6 phases :
//
//  1. Discovery   — parallèle N joueurs, read-only : loadKnown + paginate API
//  2. Dedup       — single : union des unknownIDs cross-player
//  3. FetchShared — errgroup(8) : GetMatchStats par match unique
//  4. FetchPlayer — parallèle par joueur : awards/scores requérant son token
//  5. Persist     — single writer : 1 méga-batch shared + player.* en une TX
//  6. PostSync    — parallèle N joueurs : heals + films + citations + ...
//
// Élimine la sérialisation sur le shared writer lease qui dominait V1 et
// garantit la cross-player dedup correcte (visibilité immédiate des writes
// avant le prochain loadKnown).
//
// Activation : V2 PAR DÉFAUT (shouldUseV2 = LEVELUP_SYNC_PIPELINE != "v1").
// LEVELUP_SYNC_PIPELINE=v1 = kill-switch de rollback vers le legacy V1 (retrait
// planifié lot D1c). Rollback instantané.
// Coexistence : V1 et V2 partagent les Persisters, le schéma DB et le WAL.
//
// Tests anti-régression critiques : cf. tests/integration/sync_v2/ et
// internal/sync/contract_test.go.
package v2
