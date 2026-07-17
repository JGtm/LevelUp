package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
)

// briefingCtxRaw : raw row du briefing avec le contexte social (IsWithFriends).
func briefingCtxRaw(id string, daysAgo, outcome int, withFriends bool) domain.MatchHistoryRawRow {
	r := briefingRaw(id, daysAgo, outcome, 10, 5, 2, 60, "map1", "Aquarius", "Slayer", "Arène classée")
	r.IsWithFriends = withFriends
	return r
}

// Cas pertinent : les deux sous-groupes ≥ seuil → bloc présent avec valeurs justes.
func TestBuildBriefingContextSplit_RelevantBothAboveThreshold(t *testing.T) {
	var scope []domain.MatchHistoryRawRow
	// 10 solo victorieux (WinRate 1.0), 10 escouade perdus (WinRate 0.0).
	for i := 0; i < minContextSplitMatches; i++ {
		scope = append(scope, briefingCtxRaw("s"+string(rune('a'+i)), i, domain.OutcomeWin, false))
	}
	for i := 0; i < minContextSplitMatches; i++ {
		scope = append(scope, briefingCtxRaw("q"+string(rune('a'+i)), 50+i, domain.OutcomeLoss, true))
	}
	cs := buildBriefingContextSplit(scope)
	if cs == nil {
		t.Fatal("split attendu non nil (les deux contextes ≥ seuil)")
	}
	if cs.Solo.Matches != minContextSplitMatches || cs.Squad.Matches != minContextSplitMatches {
		t.Fatalf("matches solo=%d squad=%d, attendu %d/%d", cs.Solo.Matches, cs.Squad.Matches, minContextSplitMatches, minContextSplitMatches)
	}
	if cs.Solo.WinRate != 1.0 {
		t.Errorf("solo win_rate = %v, attendu 1.0", cs.Solo.WinRate)
	}
	if cs.Squad.WinRate != 0.0 {
		t.Errorf("squad win_rate = %v, attendu 0.0", cs.Squad.WinRate)
	}
	// KDA agrégat ADR 0006 : ((100 + 20/3) − 50)/10 ≈ 5.6667 (identique pour les deux).
	if cs.Solo.KDA < 5.6 || cs.Solo.KDA > 5.7 {
		t.Errorf("solo kda = %v, attendu ≈ 5.667", cs.Solo.KDA)
	}
	if cs.Solo.AvgPerf == nil || cs.Squad.AvgPerf == nil {
		t.Error("avg_perf attendu non nil (perf renseignée)")
	}
}

// Cas mono-contexte : un sous-groupe vide (tout solo) → nil.
func TestBuildBriefingContextSplit_MonoContextNil(t *testing.T) {
	var scope []domain.MatchHistoryRawRow
	for i := 0; i < 20; i++ {
		scope = append(scope, briefingCtxRaw("s"+string(rune('a'+i)), i, domain.OutcomeWin, false))
	}
	if cs := buildBriefingContextSplit(scope); cs != nil {
		t.Fatalf("scope mono-contexte (tout solo) doit donner nil, got %+v", cs)
	}
}

// Cas sous le seuil : un sous-groupe < seuil → nil.
func TestBuildBriefingContextSplit_BelowThresholdNil(t *testing.T) {
	var scope []domain.MatchHistoryRawRow
	for i := 0; i < 15; i++ {
		scope = append(scope, briefingCtxRaw("s"+string(rune('a'+i)), i, domain.OutcomeWin, false))
	}
	// Seulement 9 escouade (< minContextSplitMatches = 10).
	for i := 0; i < minContextSplitMatches-1; i++ {
		scope = append(scope, briefingCtxRaw("q"+string(rune('a'+i)), 50+i, domain.OutcomeLoss, true))
	}
	if cs := buildBriefingContextSplit(scope); cs != nil {
		t.Fatalf("sous-groupe escouade < seuil doit donner nil, got %+v", cs)
	}
}

// Cas low_sample : modules omis (retour anticipé) → ContextSplit nil.
func TestBuildExplorerBriefing_ContextSplitOmittedWhenLowSample(t *testing.T) {
	var filtered []domain.MatchHistoryRawRow
	for i := 0; i < 8; i++ {
		filtered = append(filtered, briefingCtxRaw("m"+string(rune('a'+i)), i, domain.OutcomeWin, i%2 == 0))
	}
	b := svcWithRanked(false).buildExplorerBriefing(context.Background(), filtered, filtered)
	if b == nil {
		t.Fatal("briefing nil")
	}
	if !b.LowSample {
		t.Fatal("attendu LowSample (scope < seuil modules)")
	}
	if b.ContextSplit != nil {
		t.Errorf("ContextSplit doit être nil sous low_sample, got %+v", b.ContextSplit)
	}
}
