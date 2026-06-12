// Package scheduler — post_sync_history_test.go : ring des durées post-sync par
// joueur (sparkline de tendance, P8 monitoring).
package scheduler

import (
	"testing"

	"levelup/go-api/internal/domain"
)

func TestPostSyncHistory_RingBounded(t *testing.T) {
	s := &AutoSyncScheduler{
		playerOutcomes:  make(map[string]PlayerOutcomeDetail),
		postSyncHistory: make(map[string][]int64),
	}

	// 20 post-syncs réussis pour JGtm (durées 10..200) → ring borné à 16.
	for i := 1; i <= 20; i++ {
		s.recordOutcome(PlayerOutcomeDetail{
			Gamertag: "JGtm", Outcome: "ok",
			PostSync: &domain.PostSyncResult{DurationMs: int64(i * 10)},
		})
	}
	// Un joueur skipped (PostSync nil) → aucun point d'historique.
	s.recordOutcome(PlayerOutcomeDetail{Gamertag: "Madina", Outcome: "skipped"})

	hist := s.postSyncHistory["JGtm"]
	if len(hist) != postSyncHistorySize {
		t.Fatalf("ring JGtm = %d (attendu %d, borné)", len(hist), postSyncHistorySize)
	}
	// Après éviction : les 16 derniers = i 5..20 → 50..200 (ordre chronologique).
	if hist[0] != 50 {
		t.Errorf("premier point conservé = %d (attendu 50)", hist[0])
	}
	if last := hist[len(hist)-1]; last != 200 {
		t.Errorf("dernier point = %d (attendu 200)", last)
	}
	if len(s.postSyncHistory["Madina"]) != 0 {
		t.Errorf("Madina (skipped) ne devrait pas avoir d'historique : %v", s.postSyncHistory["Madina"])
	}
}
