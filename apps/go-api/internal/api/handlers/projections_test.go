// Package handlers — projections_test.go : BuildExplorerRowFromMatchHistory.
//
// V72-32 : vérifie que le signal de placement (PlacementDone/PlacementTotal),
// déjà utilisé par la colonne "Rang" de l'Explorer, traverse bien la
// projection MatchHistoryRow → ExplorerMatchesRow AUX CÔTÉS d'un PerfScore et
// d'un RatingType nuls — c'est ce triplet (placement présent + note absente)
// que consomme ExplorerMatchesTable.placement.tsx (front) pour afficher le
// badge « En placement » sur Perf/ΔPerf/Note à la place du "-".
package handlers

import (
	"testing"

	"levelup/go-api/internal/domain"
)

func intPtr(v int) *int { return &v }

func TestBuildExplorerRowFromMatchHistory_placementPropagatesWithNilScoreAndRating(t *testing.T) {
	item := domain.MatchHistoryRow{
		MatchID:                  "btb-1",
		PerformanceScoreRelative: nil, // en placement → pas encore de note relative
		PerfTier:                 0,
		SkillRatingType:          nil, // en placement → pas encore de LUSR/CSR
		PlacementDone:            intPtr(3),
		PlacementTotal:           intPtr(10),
	}
	row := BuildExplorerRowFromMatchHistory(item)

	if row.PerfScore != nil {
		t.Errorf("PerfScore want nil (en placement), got %v", *row.PerfScore)
	}
	if row.DeltaPerf != nil {
		t.Errorf("DeltaPerf want nil (dérivé de PerfScore nil), got %v", *row.DeltaPerf)
	}
	if row.RatingType != nil {
		t.Errorf("RatingType want nil (en placement), got %v", *row.RatingType)
	}
	if row.PlacementDone == nil || *row.PlacementDone != 3 {
		t.Errorf("PlacementDone want 3, got %v", row.PlacementDone)
	}
	if row.PlacementTotal == nil || *row.PlacementTotal != 10 {
		t.Errorf("PlacementTotal want 10, got %v", row.PlacementTotal)
	}
}

// TestBuildExplorerRowFromMatchHistory_perfPlacementPropagatesIndependently (V72-34)
// vérifie que le signal DÉDIÉ aux colonnes Perf/ΔPerf traverse la projection
// INDÉPENDAMMENT du placement de classement — cas JGtm : LUSR établi (rating_type
// non nul, PAS de placement_done) MAIS chaîne perf sous le seuil → perf_placement 8/10
// et perf_score nul. C'est ce couple que consomme hasPerfPlacementSignal côté front.
func TestBuildExplorerRowFromMatchHistory_perfPlacementPropagatesIndependently(t *testing.T) {
	item := domain.MatchHistoryRow{
		MatchID:                  "btb-jgtm",
		PerformanceScoreRelative: nil, // chaîne perf en calibration
		PerfTier:                 0,
		SkillRatingType:          strPtrHandlers("LUSR"), // Note établie
		SkillTierLabel:           strPtrHandlers("Or II"),
		PlacementDone:            nil, // pas en placement de CLASSEMENT
		PlacementTotal:           nil,
		PerfPlacementDone:        intPtr(8),
		PerfPlacementTotal:       intPtr(10),
	}
	row := BuildExplorerRowFromMatchHistory(item)

	if row.PerfScore != nil || row.DeltaPerf != nil {
		t.Errorf("PerfScore/DeltaPerf want nil (chaîne perf en calibration), got %v/%v", row.PerfScore, row.DeltaPerf)
	}
	if row.PerfPlacementDone == nil || *row.PerfPlacementDone != 8 {
		t.Errorf("PerfPlacementDone want 8, got %v", row.PerfPlacementDone)
	}
	if row.PerfPlacementTotal == nil || *row.PerfPlacementTotal != 10 {
		t.Errorf("PerfPlacementTotal want 10, got %v", row.PerfPlacementTotal)
	}
	// La colonne Note garde son propre signal : rating_type présent, placement absent.
	if row.RatingType == nil || *row.RatingType != "LUSR" {
		t.Errorf("RatingType want LUSR (Note inchangée), got %v", row.RatingType)
	}
	if row.PlacementDone != nil || row.PlacementTotal != nil {
		t.Errorf("placement de classement doit rester nil, got %v/%v", row.PlacementDone, row.PlacementTotal)
	}
}

func TestBuildExplorerRowFromMatchHistory_deltaPerfDerivedFromScoreMinus50(t *testing.T) {
	score := 73
	item := domain.MatchHistoryRow{
		MatchID:                  "rated-1",
		PerformanceScoreRelative: &score,
		PerfTier:                 2,
		SkillRatingType:          strPtrHandlers("LUSR"),
	}
	row := BuildExplorerRowFromMatchHistory(item)

	if row.PerfScore == nil || *row.PerfScore != 73 {
		t.Errorf("PerfScore want 73, got %v", row.PerfScore)
	}
	if row.DeltaPerf == nil || *row.DeltaPerf != 23 {
		t.Errorf("DeltaPerf want 23 (73-50), got %v", row.DeltaPerf)
	}
	if row.RatingType == nil || *row.RatingType != "LUSR" {
		t.Errorf("RatingType want LUSR, got %v", row.RatingType)
	}
	if row.PlacementDone != nil {
		t.Errorf("PlacementDone want nil (hors placement), got %v", *row.PlacementDone)
	}
}

func strPtrHandlers(s string) *string { return &s }
