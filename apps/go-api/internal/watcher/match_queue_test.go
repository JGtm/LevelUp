package watcher

import (
	"testing"
)

func TestMatchQueue_Enqueue_Dequeue(t *testing.T) {
	q := NewMatchQueue(10)
	q.Enqueue(MatchRequest{Gamertag: "p1", XUID: "x1", MatchIDs: []string{"m1", "m2"}})

	if q.Len() != 1 {
		t.Errorf("Len = %d, want 1", q.Len())
	}

	select {
	case req := <-q.Dequeue():
		if req.Gamertag != "p1" {
			t.Errorf("Gamertag = %q", req.Gamertag)
		}
		if len(req.MatchIDs) != 2 {
			t.Errorf("MatchIDs = %v", req.MatchIDs)
		}
	default:
		t.Fatal("expected item in queue")
	}
}

func TestMatchQueue_Dedup(t *testing.T) {
	q := NewMatchQueue(10)
	q.Enqueue(MatchRequest{Gamertag: "p1", XUID: "x1", MatchIDs: []string{"m1", "m2"}})
	q.Enqueue(MatchRequest{Gamertag: "p1", XUID: "x1", MatchIDs: []string{"m2", "m3"}})

	// 1ère requête : m1, m2 (tous nouveaux)
	req1 := <-q.Dequeue()
	if len(req1.MatchIDs) != 2 {
		t.Errorf("req1 MatchIDs = %v", req1.MatchIDs)
	}

	// 2ème requête : m3 seulement (m2 déjà vu)
	req2 := <-q.Dequeue()
	if len(req2.MatchIDs) != 1 || req2.MatchIDs[0] != "m3" {
		t.Errorf("req2 MatchIDs = %v, want [m3]", req2.MatchIDs)
	}
}

func TestMatchQueue_DedupAcrossPlayers(t *testing.T) {
	q := NewMatchQueue(10)
	// Même match_id mais joueurs différents → pas dédupliqué
	q.Enqueue(MatchRequest{Gamertag: "p1", XUID: "x1", MatchIDs: []string{"m1"}})
	q.Enqueue(MatchRequest{Gamertag: "p2", XUID: "x2", MatchIDs: []string{"m1"}})

	if q.Len() != 2 {
		t.Errorf("Len = %d, want 2 (different players, same match)", q.Len())
	}
}

func TestMatchQueue_AllDuplicates(t *testing.T) {
	q := NewMatchQueue(10)
	q.Enqueue(MatchRequest{Gamertag: "p1", XUID: "x1", MatchIDs: []string{"m1"}})
	q.Enqueue(MatchRequest{Gamertag: "p1", XUID: "x1", MatchIDs: []string{"m1"}}) // all dupes

	if q.Len() != 1 {
		t.Errorf("Len = %d, want 1", q.Len())
	}
}

func TestMatchQueue_FullQueue(t *testing.T) {
	q := NewMatchQueue(1)
	q.Enqueue(MatchRequest{Gamertag: "p1", XUID: "x1", MatchIDs: []string{"m1"}})
	q.Enqueue(MatchRequest{Gamertag: "p2", XUID: "x2", MatchIDs: []string{"m2"}}) // devrait être ignoré (queue pleine)

	if q.Len() != 1 {
		t.Errorf("Len = %d, want 1 (queue full)", q.Len())
	}
}

func TestMatchQueue_ClearSeen(t *testing.T) {
	q := NewMatchQueue(10)
	q.Enqueue(MatchRequest{Gamertag: "p1", XUID: "x1", MatchIDs: []string{"m1"}})
	<-q.Dequeue()

	q.ClearSeen()
	q.Enqueue(MatchRequest{Gamertag: "p1", XUID: "x1", MatchIDs: []string{"m1"}})

	if q.Len() != 1 {
		t.Errorf("Len = %d, want 1 after ClearSeen", q.Len())
	}
}
