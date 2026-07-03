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
// Activation : V2 est l'UNIQUE moteur de sync des joueurs moteur (pipeline V1
// supprimé au lot D1c, 2026-07). shouldUseV2 = orchestrator câblé ; s'il ne l'est
// pas (prérequis boot manquants), le scheduler bascule sur le filet structurel
// syncPlayer. Les titres live-only (Halo 5) passent par syncPlayer→liveRunner,
// jamais par V2. V2 réutilise les Persisters, le schéma DB et le WAL (écriture
// unique) + le SyncEngine pour le post-sync (engine.run reste partagé).
//
// Tests anti-régression critiques : cf. tests/integration/sync_v2/ et
// internal/sync/contract_test.go.
package v2
