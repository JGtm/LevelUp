// Package port — post_sync_runner.go : interface du runner post-sync invoqué
// par le SyncEngine après chaque sync réussi (HTTP, auto-sync, CLI).
//
// Phase 4 plan stabilisation 2026-05-22 — solution B1 de
// AUDIT_ASCENSION_PIPELINE_DISCONNECTED_2026-05-21.
//
// Contexte : avant ce fix, le hook post-sync (deltas + progression V2
// Ascension) était attaché uniquement au SyncHandler HTTP via
// `WithPostSyncDeltaHook`. L'auto-sync scheduler et les CLI court-circuitaient
// le handler en appelant `SyncEngine.RunDelta` directement → 100% des syncs
// en condition réelle (auto-sync 15 min) sautaient le pipeline progression.
// Conséquence : tables `streak`, `record_history`, `player_records`,
// `milestone_earned` restaient vides indéfiniment ; toutes les notifications
// post-sync (career_rank, skill_tier, citation_tier, threshold_crossed, etc.)
// étaient désactivées.
//
// Fix : déplacer l'invocation dans SyncEngine.runPostSyncPipeline. Toute
// sync (peu importe l'entry point) déclenche désormais le runner.
package port

import "context"

// PostSyncFinalizer est la closure invoquée APRÈS la sync (le caller a déjà
// committé les writes shared+player). Elle compare l'état avant/après et émet
// les deltas + lance le pipeline progression V2.
type PostSyncFinalizer func(ctx context.Context)

// PostSyncRunner est l'interface implémentée par l'adapter
// `api.buildPostSyncRunner` (qui wrappe `buildPostSyncDeltaHook` + la couche
// progression). Injecté dans SyncEngine via `WithPostSyncRunner`.
//
// Contrat :
//   - BeforeSync est appelé AVANT la sync. Il capture un snapshot du joueur
//     (career_rank, citations, challenges, skill_rank, etc.) et retourne un
//     PostSyncFinalizer.
//   - Le SyncEngine invoque le PostSyncFinalizer APRÈS la sync (uniquement
//     si la sync a réussi). Le finalizer recapture le snapshot, émet les
//     deltas via notifications.Emitter, et lance EvaluateProgressionAfterSync.
//   - Si BeforeSync retourne nil (ex: resolve player échoué), aucun
//     finalizer n'est invoqué — la sync continue normalement.
//
// Non-bloquant : toute erreur dans le runner est loguée mais n'échoue jamais
// la sync.
type PostSyncRunner interface {
	BeforeSync(ctx context.Context, slug string) PostSyncFinalizer
}
