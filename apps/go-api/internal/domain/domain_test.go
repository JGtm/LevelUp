// Package domain — domain_test.go : tests des types et helpers domain.
//
// Sprint 39 — couverture des fonctions pures du package domain.
package domain

import (
	"testing"
)

// ---------------------------------------------------------------------------
// APIError
// ---------------------------------------------------------------------------

func TestAPIError_Error(t *testing.T) {
	e := &APIError{Code: "not_found", Message: "player introuvable : foo"}
	got := e.Error()
	want := "not_found: player introuvable : foo"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestErrNotFound(t *testing.T) {
	e := ErrNotFound("match", "abc-123")
	if e.Code != "not_found" {
		t.Errorf("Code = %q, want not_found", e.Code)
	}
	if e.Retryable {
		t.Error("ErrNotFound should not be retryable")
	}
}

func TestErrBadRequest(t *testing.T) {
	e := ErrBadRequest("invalid field")
	if e.Code != "bad_request" {
		t.Errorf("Code = %q, want bad_request", e.Code)
	}
	if e.Retryable {
		t.Error("ErrBadRequest should not be retryable")
	}
}

func TestErrInternal(t *testing.T) {
	e := ErrInternal("db crash")
	if e.Code != "internal_error" {
		t.Errorf("Code = %q, want internal_error", e.Code)
	}
	if !e.Retryable {
		t.Error("ErrInternal should be retryable")
	}
}

// ---------------------------------------------------------------------------
// AsyncJobStatus
// ---------------------------------------------------------------------------

func TestAsyncJobStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status JobStatus
		want   bool
	}{
		{JobStatusRunning, false},
		{JobStatusQueued, false},
		{JobStatusSucceeded, true},
		{JobStatusFailed, true},
		{JobStatusCancelled, true},
		{JobStatusInterrupted, true},
	}
	for _, tt := range tests {
		job := &AsyncJobStatus{Status: tt.status}
		if got := job.IsTerminal(); got != tt.want {
			t.Errorf("IsTerminal(%q) = %v, want %v", tt.status, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// SyncResult
// ---------------------------------------------------------------------------

func TestSyncResult_Status_Success(t *testing.T) {
	r := &SyncResult{MatchesInserted: 5}
	if got := r.Status(); got != "success" {
		t.Errorf("Status() = %q, want success", got)
	}
}

func TestSyncResult_Status_PartialSuccess(t *testing.T) {
	r := &SyncResult{MatchesInserted: 3, Errors: []string{"timeout"}}
	if got := r.Status(); got != "partial_success" {
		t.Errorf("Status() = %q, want partial_success", got)
	}
}

func TestSyncResult_Status_Failure(t *testing.T) {
	r := &SyncResult{MatchesInserted: 0, Errors: []string{"auth failed"}}
	if got := r.Status(); got != "failure" {
		t.Errorf("Status() = %q, want failure", got)
	}
}

func TestSyncResult_AddError(t *testing.T) {
	r := &SyncResult{}
	r.AddError("err1")
	r.AddError("err2")
	if len(r.Errors) != 2 {
		t.Errorf("Errors len = %d, want 2", len(r.Errors))
	}
}

func TestSyncResult_AddWarning(t *testing.T) {
	r := &SyncResult{}
	r.AddWarning("warn1")
	if len(r.Warnings) != 1 {
		t.Errorf("Warnings len = %d, want 1", len(r.Warnings))
	}
}

func TestDefaultSyncOptions(t *testing.T) {
	opts := DefaultSyncOptions()
	if opts.MatchType != "matchmaking" {
		t.Errorf("MatchType = %q, want matchmaking", opts.MatchType)
	}
	if opts.MaxMatches != 200 {
		t.Errorf("MaxMatches = %d, want 200", opts.MaxMatches)
	}
	if !opts.WithParticipants {
		t.Error("WithParticipants should be true")
	}
	if !opts.WithMedals {
		t.Error("WithMedals should be true")
	}
	if opts.RequestsPerSecond != 10 {
		t.Errorf("RequestsPerSecond = %d, want 10", opts.RequestsPerSecond)
	}
}

// ---------------------------------------------------------------------------
// Outcomes constants
// ---------------------------------------------------------------------------

func TestOutcomeConstants(t *testing.T) {
	if OutcomeWin != 2 {
		t.Errorf("OutcomeWin = %d, want 2", OutcomeWin)
	}
	if OutcomeLoss != 3 {
		t.Errorf("OutcomeLoss = %d, want 3", OutcomeLoss)
	}
	if OutcomeDraw != 1 {
		t.Errorf("OutcomeDraw = %d, want 1", OutcomeDraw)
	}
	if OutcomeUnknown != 0 {
		t.Errorf("OutcomeUnknown = %d, want 0", OutcomeUnknown)
	}
}

// --- JobMeta ---

func TestJobMeta_GetTitleSlug_Default(t *testing.T) {
	m := JobMeta{}
	if got := m.GetTitleSlug(); got != "halo_infinite" {
		t.Errorf("GetTitleSlug() = %q, want halo_infinite", got)
	}
}

func TestJobMeta_GetTitleSlug_Custom(t *testing.T) {
	m := JobMeta{TitleSlug: "halo_mcc"}
	if got := m.GetTitleSlug(); got != "halo_mcc" {
		t.Errorf("GetTitleSlug() = %q, want halo_mcc", got)
	}
}

func TestJobMeta_WithTitleSlug(t *testing.T) {
	m := JobMeta{}
	m2 := m.WithTitleSlug("halo_mcc")
	if m2.TitleSlug != "halo_mcc" {
		t.Errorf("WithTitleSlug failed: %q", m2.TitleSlug)
	}
	if m.TitleSlug != "" {
		t.Error("original should not be modified")
	}
}
