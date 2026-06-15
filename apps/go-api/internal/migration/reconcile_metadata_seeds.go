package migration

// reconcile_metadata_seeds.go — ReconcileMetadataSeeds a été déplacé vers
// internal/games/halo_infinite/migrations/mode_playlist_fr.go (Phase 1.5 b7) :
// il dépend de applyModeNameTr/applyPlaylistFRSeeds, désormais title-owned, et le
// package migration ne peut pas importer le package titre (cycle).
// Caller boot : cmd/server/main.go appelle halomigrations.ReconcileMetadataSeeds.
