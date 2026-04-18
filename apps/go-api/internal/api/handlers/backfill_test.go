package handlers

import (
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
	go_sync "levelup/go-api/internal/sync"
)

func TestBuildSyncScope_AllData(t *testing.T) {
	req := domain.BackfillStartRequest{AllData: true, MaxMatches: 50}
	scope := buildSyncScope(req)
	if !scope.AllData {
		t.Fatal("expected AllData")
	}
	if !scope.Medals || !scope.Events || !scope.Skill {
		t.Fatal("expected all types enabled")
	}
	if scope.MaxMatches != 50 {
		t.Fatalf("expected 50, got %d", scope.MaxMatches)
	}
}

func TestBuildSyncScope_NoFlags(t *testing.T) {
	req := domain.BackfillStartRequest{}
	scope := buildSyncScope(req)
	if !scope.AllData {
		t.Fatal("expected AllData when no flags set")
	}
}

func TestBuildSyncScope_PartialFlags(t *testing.T) {
	req := domain.BackfillStartRequest{Medals: true, Events: true}
	scope := buildSyncScope(req)
	if scope.AllData {
		t.Fatal("expected partial, not AllData")
	}
	if !scope.Medals || !scope.Events {
		t.Fatal("expected medals+events")
	}
	if scope.Skill {
		t.Fatal("expected skill false")
	}
}

func TestBuildSyncScope_ForceRescan(t *testing.T) {
	req := domain.BackfillStartRequest{
		Medals:      true,
		ForceRescan: true,
	}
	scope := buildSyncScope(req)
	if !scope.ForceMedals {
		t.Fatal("expected ForceMedals")
	}
	if scope.ForceSkill {
		t.Fatal("expected ForceSkill false (skill not requested)")
	}
}

func TestBuildSyncScope_ForceRescanAllData(t *testing.T) {
	req := domain.BackfillStartRequest{
		AllData:     true,
		ForceRescan: true,
	}
	scope := buildSyncScope(req)
	if !scope.ForceMedals || !scope.ForceSkill || !scope.ForceWeapons {
		t.Fatal("expected all force flags when AllData+ForceRescan")
	}
}

func TestNewBackfillHandler(t *testing.T) {
	cfg := &config.AppConfig{}
	store := jobs.NewStore(t.TempDir())
	h := NewBackfillHandler(cfg, store)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
	if h.cfg != cfg {
		t.Fatal("cfg mismatch")
	}
}

func TestWarnUnimplemented_NoTypes(t *testing.T) {
	store := jobs.NewStore(t.TempDir())
	h := &BackfillHandler{jobStore: store}
	job := store.Create(domain.JobTypeBackfill, "test")

	scope := &go_sync.SyncScope{} // no flags
	h.warnUnimplemented(job.JobID, scope)

	got := store.Get(job.JobID)
	if len(got.Warnings) != 0 {
		t.Fatalf("expected 0 warnings, got %d", len(got.Warnings))
	}
}

func TestWarnUnimplemented_WithTypes(t *testing.T) {
	store := jobs.NewStore(t.TempDir())
	h := &BackfillHandler{jobStore: store}
	job := store.Create(domain.JobTypeBackfill, "test")

	scope := &go_sync.SyncScope{Medals: true, Events: true}
	h.warnUnimplemented(job.JobID, scope)

	got := store.Get(job.JobID)
	if len(got.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(got.Warnings))
	}
}
