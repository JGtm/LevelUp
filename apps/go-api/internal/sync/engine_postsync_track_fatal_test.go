// Package sync — engine_postsync_track_fatal_test.go : test du helper
// trackFatalErr (Phase 5 ART — Status sync honnête).
//
// Vérifie que les erreurs FATAL DuckDB (IsInvalidatedError) sont
// collectées dans PostSyncResult.FatalErrors, et que les erreurs non
// fatales (network timeout, parse, etc.) sont ignorées par ce tracking
// (elles restent en WARN logs).

package sync

import (
	"errors"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
)

func TestTrackFatalErr_NilError_NoOp(t *testing.T) {
	r := &domain.PostSyncResult{}
	trackFatalErr(r, "LUSR", nil)
	if len(r.FatalErrors) != 0 {
		t.Errorf("FatalErrors = %v, want empty (nil err)", r.FatalErrors)
	}
}

func TestTrackFatalErr_NonFatalError_NoOp(t *testing.T) {
	r := &domain.PostSyncResult{}
	trackFatalErr(r, "weapon kills", errors.New("network timeout"))
	trackFatalErr(r, "citations", errors.New("metadata table missing"))
	if len(r.FatalErrors) != 0 {
		t.Errorf("FatalErrors = %v, want empty (non-fatal errors)", r.FatalErrors)
	}
}

func TestTrackFatalErr_FatalInvalidated_Appended(t *testing.T) {
	r := &domain.PostSyncResult{}
	fatal := errors.New("FATAL Error: database has been invalidated because of a previous fatal error")
	trackFatalErr(r, "LUSR", fatal)
	if len(r.FatalErrors) != 1 {
		t.Fatalf("FatalErrors count = %d, want 1", len(r.FatalErrors))
	}
	if !strings.HasPrefix(r.FatalErrors[0], "LUSR: ") {
		t.Errorf("FatalErrors[0] = %q, want prefix 'LUSR: '", r.FatalErrors[0])
	}
}

func TestTrackFatalErr_FailedToDeleteRows_Appended(t *testing.T) {
	r := &domain.PostSyncResult{}
	// Signature exacte du crash prod 2026-05-24 20:41:04 (Chocoboflor LUSR).
	fatal := errors.New("Invalid Input Error: Failed to delete all rows from index. Only deleted 0 out of 1 rows.")
	trackFatalErr(r, "LUSR", fatal)
	if len(r.FatalErrors) != 1 {
		t.Fatalf("FatalErrors count = %d, want 1 (ART deletion signature)", len(r.FatalErrors))
	}
}

func TestTrackFatalErr_MultipleSites_Cascade(t *testing.T) {
	// Reproduit la cascade post-sync observée 2026-05-24 20:41 :
	// LUSR FATAL → friends/aggregates/achievements re-hit le même FATAL.
	r := &domain.PostSyncResult{}
	fatal := errors.New("FATAL Error: database has been invalidated")
	trackFatalErr(r, "LUSR", fatal)
	trackFatalErr(r, "friends recompute", fatal)
	trackFatalErr(r, "aggregates", fatal)
	trackFatalErr(r, "shared views", fatal)

	if len(r.FatalErrors) != 4 {
		t.Errorf("FatalErrors count = %d, want 4 (cascade)", len(r.FatalErrors))
	}

	// Chaque entrée doit avoir son préfixe de step pour traçabilité.
	expectedSteps := []string{"LUSR", "friends recompute", "aggregates", "shared views"}
	for i, step := range expectedSteps {
		if !strings.HasPrefix(r.FatalErrors[i], step+": ") {
			t.Errorf("FatalErrors[%d] = %q, want prefix %q",
				i, r.FatalErrors[i], step+": ")
		}
	}
}
