package analysis_test

import (
	"testing"

	"levelup/go-api/internal/analysis"
)

func TestComputeKDTimeline_Empty(t *testing.T) {
	result := analysis.ComputeKDTimeline(nil, "xuid1")
	if result != nil {
		t.Errorf("attendu nil, obtenu %v", result)
	}
}

func TestComputeKDTimeline_NoPlayerEvents(t *testing.T) {
	events := []analysis.KDEvent{
		{TimeMS: 1000, IsKill: true, ActorXUID: "other"},
	}
	result := analysis.ComputeKDTimeline(events, "xuid1")
	if result != nil {
		t.Errorf("attendu nil, obtenu %v", result)
	}
}

func TestComputeKDTimeline_KillsOnly(t *testing.T) {
	events := []analysis.KDEvent{
		{TimeMS: 1000, IsKill: true, ActorXUID: "xuid1"},
		{TimeMS: 2000, IsKill: true, ActorXUID: "xuid1"},
	}
	pts := analysis.ComputeKDTimeline(events, "xuid1")
	if len(pts) != 2 {
		t.Fatalf("attendu 2 points, obtenu %d", len(pts))
	}
	if pts[1].CumKills != 2 {
		t.Errorf("attendu CumKills=2, obtenu %d", pts[1].CumKills)
	}
	if pts[1].CumDeaths != 0 {
		t.Errorf("attendu CumDeaths=0, obtenu %d", pts[1].CumDeaths)
	}
	// KDRatio = kills (0 deaths)
	if pts[1].KDRatio != 2.0 {
		t.Errorf("attendu KDRatio=2.0, obtenu %f", pts[1].KDRatio)
	}
}

func TestComputeKDTimeline_MixedEvents(t *testing.T) {
	events := []analysis.KDEvent{
		{TimeMS: 1000, IsKill: true, ActorXUID: "xuid1"},
		{TimeMS: 2000, IsKill: false, ActorXUID: "xuid1"},
		{TimeMS: 3000, IsKill: true, ActorXUID: "xuid1"},
	}
	pts := analysis.ComputeKDTimeline(events, "xuid1")
	if len(pts) != 3 {
		t.Fatalf("attendu 3 points, obtenu %d", len(pts))
	}
	last := pts[2]
	if last.CumKills != 2 || last.CumDeaths != 1 {
		t.Errorf("attendu K=2 D=1, obtenu K=%d D=%d", last.CumKills, last.CumDeaths)
	}
	if last.KDRatio < 1.9 || last.KDRatio > 2.1 {
		t.Errorf("attendu KDRatio≈2.0, obtenu %f", last.KDRatio)
	}
}

func TestComputeKDTimeline_OrderedByTime(t *testing.T) {
	events := []analysis.KDEvent{
		{TimeMS: 3000, IsKill: true, ActorXUID: "xuid1"},
		{TimeMS: 1000, IsKill: false, ActorXUID: "xuid1"},
		{TimeMS: 2000, IsKill: true, ActorXUID: "xuid1"},
	}
	pts := analysis.ComputeKDTimeline(events, "xuid1")
	if pts[0].TimeMS != 1000 {
		t.Errorf("attendu TimeMS=1000 en premier, obtenu %d", pts[0].TimeMS)
	}
}
