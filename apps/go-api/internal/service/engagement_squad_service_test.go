// Package service — engagement_squad_service_test.go : GetSquadSession remplit
// DurationsSeconds aligne sur Labels (P4, ecart d'engagement cumule par joueur).
package service

import (
	"context"
	"testing"
)

// TestGetSquadSession_FillsDurationsAlignedToLabels verifie que chaque match
// ajoute a la session porte sa duree (secondes), alignee 1:1 sur Labels. Le mock
// memoCountingEngagementRepo sert des matchs de 720 s (EndTimeMS 720_000).
func TestGetSquadSession_FillsDurationsAlignedToLabels(t *testing.T) {
	repo := &memoCountingEngagementRepo{}
	svc := NewPlayerEngagementService(repo, "xuid-test", "GT-test")

	ids := []string{"ranked-1", "ranked-2", "ranked-3"}
	sess, err := svc.GetSquadSession(context.Background(), ids, nil)
	if err != nil {
		t.Fatalf("GetSquadSession: %v", err)
	}
	if len(sess.Labels) == 0 {
		t.Fatal("aucun match ajoute a la session (bundle non calculable)")
	}
	if len(sess.DurationsSeconds) != len(sess.Labels) {
		t.Fatalf("durations desalignees : %d durations vs %d labels",
			len(sess.DurationsSeconds), len(sess.Labels))
	}
	for i, d := range sess.DurationsSeconds {
		if d != 720 {
			t.Errorf("duration[%d] = %d, attendu 720 (720_000 ms / 1000)", i, d)
		}
	}
}

// TestGetSquadSession_EmptyMatchesNoDurations verifie le cas vide : aucune duree.
func TestGetSquadSession_EmptyMatchesNoDurations(t *testing.T) {
	repo := &memoCountingEngagementRepo{}
	svc := NewPlayerEngagementService(repo, "xuid-test", "GT-test")

	sess, err := svc.GetSquadSession(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("GetSquadSession: %v", err)
	}
	if len(sess.DurationsSeconds) != 0 {
		t.Errorf("attendu 0 duration pour une session vide, got %d", len(sess.DurationsSeconds))
	}
}
