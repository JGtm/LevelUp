package handlers

import (
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
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
	if !scope.ForceMedals || !scope.ForceSkill || !scope.ForceEvents {
		t.Fatal("expected all force flags when AllData+ForceRescan")
	}
	if !scope.ForceAliases || !scope.ForceLUSR || !scope.ForcePerformanceScores {
		t.Fatal("expected ForceAliases/ForceLUSR/ForcePerformanceScores when AllData+ForceRescan")
	}
	if !scope.ForceEvents || !scope.ForcePersonalScores {
		t.Fatal("expected ForceEvents/ForcePersonalScores when AllData+ForceRescan")
	}
}

func TestBuildSyncScope_ForceRescanAliasesAndLUSR(t *testing.T) {
	req := domain.BackfillStartRequest{
		Aliases:     true,
		LUSR:        true,
		ForceRescan: true,
	}
	scope := buildSyncScope(req)
	if !scope.ForceAliases {
		t.Fatal("expected ForceAliases when Aliases+ForceRescan")
	}
	if !scope.ForceLUSR {
		t.Fatal("expected ForceLUSR when LUSR+ForceRescan")
	}
	if scope.ForceMedals || scope.ForceSkill {
		t.Fatal("expected non-requested force flags to remain false")
	}
}

func TestBuildSyncScope_ForceRescanEventsAndPersonalScores(t *testing.T) {
	req := domain.BackfillStartRequest{
		Events:         true,
		PersonalScores: true,
		ForceRescan:    true,
	}
	scope := buildSyncScope(req)
	if !scope.ForceEvents {
		t.Fatal("expected ForceEvents when Events+ForceRescan")
	}
	if !scope.ForcePersonalScores {
		t.Fatal("expected ForcePersonalScores when PersonalScores+ForceRescan")
	}
	if scope.ForceMedals || scope.ForceSkill {
		t.Fatal("expected non-requested force flags to remain false")
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

func TestBuildSyncScope_EngagementCoefficientsOnly(t *testing.T) {
	// Coef-only : seul le flag EngagementCoefficients est set, le reste reste off.
	// AllData ne doit PAS s'activer (sinon le user lance un full backfill par
	// surprise).
	req := domain.BackfillStartRequest{EngagementCoefficients: true}
	scope := buildSyncScope(req)
	if scope.AllData {
		t.Fatal("AllData should NOT activate when only EngagementCoefficients is set")
	}
	if !scope.EngagementCoefficients {
		t.Fatal("EngagementCoefficients should be true")
	}
	if scope.EngagementScores {
		t.Fatal("EngagementScores should remain false (coef-only mode)")
	}
	if scope.Medals || scope.Events || scope.Skill {
		t.Fatal("Other flags should remain false in coef-only mode")
	}
}

func TestBuildSyncScope_EngagementScoresImpliesCoefRecompute(t *testing.T) {
	// Quand EngagementScores=true, le recompute coefs est implicite (en queue
	// de RunBackfillEngagementScores). On verifie que le scope explicit
	// EngagementCoefficients reste a false dans ce cas pour eviter le
	// double-passage.
	req := domain.BackfillStartRequest{EngagementScores: true}
	scope := buildSyncScope(req)
	if !scope.EngagementScores {
		t.Fatal("EngagementScores should be true")
	}
	if scope.EngagementCoefficients {
		t.Fatal("EngagementCoefficients should remain false (recompute is implicit)")
	}
}
