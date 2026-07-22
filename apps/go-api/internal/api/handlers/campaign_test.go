package handlers

import (
	"testing"
	"time"

	"levelup/go-api/internal/campaign"
)

// TestToCampaignHistoryItem vérifie la projection campagne close → DTO historique
// (axe, delta snapshot→final, playlist, dates, statut).
func TestToCampaignHistoryItem(t *testing.T) {
	started := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ended := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	lowess := 1.8
	c := campaign.ImprovementCampaign{
		ID: "c1", UserID: "u1", TitleSlug: "halo_infinite",
		Axis: "combat", AxisKind: campaign.AxisKindRadar,
		PlaylistGroup: "ranked", Status: campaign.StatusCompleted,
		StartedAt: started, EndedAt: &ended,
		SnapshotValue: 1.0, CurrentValueLOWESS: &lowess,
	}

	item := toCampaignHistoryItem(&c)

	if item.ID != "c1" || item.Axis != "combat" || item.AxisKind != "radar" {
		t.Errorf("champs de base incorrects: %+v", item)
	}
	if item.PlaylistGroup != "ranked" || item.Status != "completed" {
		t.Errorf("playlist/statut incorrects: %+v", item)
	}
	if item.EndedAt == nil || !item.EndedAt.Equal(ended) {
		t.Errorf("ended_at incorrect: %v", item.EndedAt)
	}
	if item.FinalValue == nil || *item.FinalValue != 1.8 {
		t.Errorf("final_value: got %v, want 1.8", item.FinalValue)
	}
	if item.Delta == nil || *item.Delta != 0.8 {
		t.Errorf("delta (final-snapshot): got %v, want 0.8", item.Delta)
	}
}

// TestToCampaignHistoryItem_NoEvaluation : campagne jamais évaluée → final/delta nil.
func TestToCampaignHistoryItem_NoEvaluation(t *testing.T) {
	c := campaign.ImprovementCampaign{
		ID: "c2", Axis: "survival", AxisKind: campaign.AxisKindLUSRComponent,
		Status: campaign.StatusAbandoned, SnapshotValue: 2.0,
	}
	item := toCampaignHistoryItem(&c)
	if item.FinalValue != nil {
		t.Errorf("final_value: want nil, got %v", item.FinalValue)
	}
	if item.Delta != nil {
		t.Errorf("delta: want nil, got %v", item.Delta)
	}
}
