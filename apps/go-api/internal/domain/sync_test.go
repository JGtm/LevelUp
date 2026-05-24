// Package domain — sync_test.go : tests SyncResult.Status() et PostSyncResult.
//
// Phase 5 ART — Status sync honnête : valide que les FATAL post-sync
// propagés via PostSyncResult.FatalErrors → SyncResult.Errors font
// basculer Status() de "success" à "partial_success".

package domain

import "testing"

func TestSyncResult_Status_NoErrors_Success(t *testing.T) {
	r := &SyncResult{MatchesInserted: 5}
	if got := r.Status(); got != "success" {
		t.Errorf("Status() = %q, want %q (pas d'erreurs)", got, "success")
	}
}

func TestSyncResult_Status_WithErrorsAndInsertions_PartialSuccess(t *testing.T) {
	r := &SyncResult{MatchesInserted: 5}
	r.AddError("post-sync LUSR: FATAL Error: database has been invalidated")
	if got := r.Status(); got != "partial_success" {
		t.Errorf("Status() = %q, want %q", got, "partial_success")
	}
}

func TestSyncResult_Status_OnlyErrors_Failure(t *testing.T) {
	r := &SyncResult{MatchesInserted: 0}
	r.AddError("ingestion: API timeout")
	if got := r.Status(); got != "failure" {
		t.Errorf("Status() = %q, want %q", got, "failure")
	}
}

// TestSyncResult_Status_PostSyncFatalDegradesStatus simule le scénario
// observé en prod 2026-05-24 20:41:08 sur Chocoboflor :
//   - 15 matchs insérés OK (ingestion réussie)
//   - LUSR FATAL ART → invalide la player DB → cascade friends/aggregates/achievements
//   - Avant Phase 5 ART : Status() = "success" (mensonge, monitoring aveugle)
//   - Après Phase 5 ART : Status() = "partial_success" via propagation
//     PostSyncResult.FatalErrors → SyncResult.Errors par engine.go.
func TestSyncResult_Status_PostSyncFatalDegradesStatus(t *testing.T) {
	r := &SyncResult{MatchesInserted: 15}
	postSync := PostSyncResult{
		LUSRUpdated:        0, // post-sync LUSR a foiré
		ViewsRefreshed:     2, // pas tout : aggregates échoué
		AchievementsSynced: false,
		FatalErrors: []string{
			"LUSR: FATAL Error: Failed to delete all rows from index. Only deleted 0 out of 1 rows.",
			"aggregates: drop mv_player_matches: database has been invalidated because of a previous fatal error",
			"friends recompute: updateIsWithFriendsBatch: database has been invalidated",
		},
	}
	r.PostSync = &postSync

	// Simulation de la propagation faite par engine.go run() :
	for _, fatalErr := range postSync.FatalErrors {
		r.AddError("post-sync " + fatalErr)
	}

	if got := r.Status(); got != "partial_success" {
		t.Errorf("Status() = %q, want %q (ingestion OK mais post-sync FATAL)", got, "partial_success")
	}
	if len(r.Errors) != 3 {
		t.Errorf("Errors count = %d, want 3 (un par FATAL)", len(r.Errors))
	}
}

func TestPostSyncResult_FatalErrors_DefaultsEmpty(t *testing.T) {
	r := PostSyncResult{}
	if r.FatalErrors != nil {
		t.Errorf("FatalErrors default = %v, want nil", r.FatalErrors)
	}
}
