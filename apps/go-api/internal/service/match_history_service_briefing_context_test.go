package service

import (
	"context"
	"fmt"
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

// Cas sous le seuil (D5) : les deux côtés existent mais un est < seuil → split
// Insufficient qui porte les deux comptes.
func TestBuildBriefingContextSplit_BelowThresholdInsufficient(t *testing.T) {
	var scope []domain.MatchHistoryRawRow
	for i := 0; i < 15; i++ {
		scope = append(scope, briefingCtxRaw("s"+string(rune('a'+i)), i, domain.OutcomeWin, false))
	}
	// Seulement 4 escouade (< minContextSplitMatches = 5).
	for i := 0; i < minContextSplitMatches-1; i++ {
		scope = append(scope, briefingCtxRaw("q"+string(rune('a'+i)), 50+i, domain.OutcomeLoss, true))
	}
	cs := buildBriefingContextSplit(scope)
	if cs == nil || !cs.Insufficient {
		t.Fatalf("sous-groupe escouade < seuil doit donner un split Insufficient, got %+v", cs)
	}
	if cs.Solo.Matches != 15 || cs.Squad.Matches != minContextSplitMatches-1 {
		t.Errorf("comptes solo=%d squad=%d, attendu 15/%d", cs.Solo.Matches, cs.Squad.Matches, minContextSplitMatches-1)
	}
}

// D5 : 3 solo / 63 escouade (cas « Bases » du relevé) → Insufficient avec les comptes.
func TestBuildBriefingContextSplit_ThreeVersusSixtyThree(t *testing.T) {
	var scope []domain.MatchHistoryRawRow
	for i := 0; i < 3; i++ {
		scope = append(scope, briefingCtxRaw(fmt.Sprintf("s%d", i), i, domain.OutcomeWin, false))
	}
	for i := 0; i < 63; i++ {
		scope = append(scope, briefingCtxRaw(fmt.Sprintf("q%d", i), 100+i, domain.OutcomeLoss, true))
	}
	cs := buildBriefingContextSplit(scope)
	if cs == nil || !cs.Insufficient || cs.Solo.Matches != 3 || cs.Squad.Matches != 63 {
		t.Fatalf("attendu Insufficient 3/63, got %+v", cs)
	}
}

// D2 : exactement le seuil de chaque côté (5/5) → comparaison normale.
func TestBuildBriefingContextSplit_ExactThresholdIsNormal(t *testing.T) {
	var scope []domain.MatchHistoryRawRow
	for i := 0; i < 5; i++ {
		scope = append(scope, briefingCtxRaw(fmt.Sprintf("s%d", i), i, domain.OutcomeWin, false))
		scope = append(scope, briefingCtxRaw(fmt.Sprintf("q%d", i), 50+i, domain.OutcomeLoss, true))
	}
	cs := buildBriefingContextSplit(scope)
	if cs == nil || cs.Insufficient || cs.Solo.Matches != 5 || cs.Squad.Matches != 5 {
		t.Fatalf("5/5 doit donner une comparaison normale, got %+v", cs)
	}
}

// D4 : un côté à 0 match (0 solo / 10 escouade) → carte absente.
func TestBuildBriefingContextSplit_ZeroSideNil(t *testing.T) {
	var scope []domain.MatchHistoryRawRow
	for i := 0; i < 10; i++ {
		scope = append(scope, briefingCtxRaw(fmt.Sprintf("q%d", i), i, domain.OutcomeWin, true))
	}
	if cs := buildBriefingContextSplit(scope); cs != nil {
		t.Fatalf("un côté à 0 doit donner nil, got %+v", cs)
	}
}

// Cas low_sample : le split suit les mêmes règles qu'au-delà du seuil (D4/D5) : 4 solo + 4
// escouade → carte servie en état vide.
func TestBuildExplorerBriefing_ContextSplitServedWhenLowSample(t *testing.T) {
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
	if b.ContextSplit == nil || !b.ContextSplit.Insufficient {
		t.Fatalf("ContextSplit attendu en état vide (4 solo + 4 escouade < seuil), got %+v", b.ContextSplit)
	}
	if b.ContextSplit.Solo.Matches != 4 || b.ContextSplit.Squad.Matches != 4 {
		t.Errorf("comptes attendus 4/4, got %+v", b.ContextSplit)
	}
}
