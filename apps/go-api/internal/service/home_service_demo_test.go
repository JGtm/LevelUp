package service

import (
	"context"
	"testing"
)

// TestGetChallenges_DemoMode : en DemoMode, GetChallenges sert la fixture embarquée
// (bypass provider live + cache) avec des items renderables.
func TestGetChallenges_DemoMode(t *testing.T) {
	svc := NewHomeService(nil).WithDemoMode(true)

	resp := svc.GetChallenges(context.Background())

	if !resp.Available {
		t.Fatal("demo challenges: Available = false, want true")
	}
	if len(resp.Items) == 0 {
		t.Fatal("demo challenges: aucun item (la fixture doit peupler Items)")
	}
	if resp.Total == nil || *resp.Total != len(resp.Items) {
		t.Errorf("demo challenges: Total = %v, want %d", resp.Total, len(resp.Items))
	}
	if resp.SnapshotAt == nil {
		t.Error("demo challenges: SnapshotAt nil (fraîcheur attendue)")
	}
	// Chaque item doit avoir un titre non vide (rendu UI).
	for i, it := range resp.Items {
		if it.Title == "" {
			t.Errorf("demo challenges: item %d sans titre", i)
		}
	}
}
