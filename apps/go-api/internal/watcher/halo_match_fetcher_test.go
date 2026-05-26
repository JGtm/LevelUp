package watcher

import (
	"context"
	"errors"
	"testing"

	syncpkg "levelup/go-api/internal/sync"
)

// stubMatchHistoryClient mocke matchHistoryClient pour les tests.
type stubMatchHistoryClient struct {
	gotGamertag  string
	gotMatchType string
	gotStart     int
	gotCount     int
	returnRows   []syncpkg.MatchHistoryEntry
	returnErr    error
}

func (s *stubMatchHistoryClient) GetMatchHistory(
	_ context.Context,
	gamertag, matchType string,
	start, count int,
) ([]syncpkg.MatchHistoryEntry, error) {
	s.gotGamertag = gamertag
	s.gotMatchType = matchType
	s.gotStart = start
	s.gotCount = count
	return s.returnRows, s.returnErr
}

func TestHaloMatchFetcher_FormatsXUIDCorrectly(t *testing.T) {
	stub := &stubMatchHistoryClient{}
	f := NewHaloMatchFetcher(stub)

	_, err := f.FetchRecentMatchIDs(context.Background(), "2535470000000001", 10)
	if err != nil {
		t.Fatalf("FetchRecentMatchIDs: unexpected err: %v", err)
	}
	if stub.gotGamertag != "xuid(2535470000000001)" {
		t.Errorf("gamertag = %q, want %q", stub.gotGamertag, "xuid(2535470000000001)")
	}
	if stub.gotMatchType != "all" {
		t.Errorf("matchType = %q, want %q", stub.gotMatchType, "all")
	}
	if stub.gotStart != 0 {
		t.Errorf("start = %d, want 0", stub.gotStart)
	}
	if stub.gotCount != 10 {
		t.Errorf("count = %d, want 10", stub.gotCount)
	}
}

func TestHaloMatchFetcher_MapsResponse(t *testing.T) {
	stub := &stubMatchHistoryClient{
		returnRows: []syncpkg.MatchHistoryEntry{
			{MatchID: "match-1", StartTime: "2026-05-26T12:00:00Z"},
			{MatchID: "match-2", StartTime: "2026-05-26T12:30:00Z"},
			{MatchID: "match-3", StartTime: "2026-05-26T13:00:00Z"},
		},
	}
	f := NewHaloMatchFetcher(stub)

	ids, err := f.FetchRecentMatchIDs(context.Background(), "123", 25)
	if err != nil {
		t.Fatalf("FetchRecentMatchIDs: unexpected err: %v", err)
	}
	want := []string{"match-1", "match-2", "match-3"}
	if len(ids) != len(want) {
		t.Fatalf("len(ids) = %d, want %d", len(ids), len(want))
	}
	for i, id := range ids {
		if id != want[i] {
			t.Errorf("ids[%d] = %q, want %q", i, id, want[i])
		}
	}
}

func TestHaloMatchFetcher_EmptyResponse(t *testing.T) {
	stub := &stubMatchHistoryClient{returnRows: nil}
	f := NewHaloMatchFetcher(stub)

	ids, err := f.FetchRecentMatchIDs(context.Background(), "123", 25)
	if err != nil {
		t.Fatalf("FetchRecentMatchIDs: unexpected err: %v", err)
	}
	if ids == nil {
		t.Error("ids = nil, want non-nil empty slice")
	}
	if len(ids) != 0 {
		t.Errorf("len(ids) = %d, want 0", len(ids))
	}
}

func TestHaloMatchFetcher_PropagatesError(t *testing.T) {
	sentinel := errors.New("boom")
	stub := &stubMatchHistoryClient{returnErr: sentinel}
	f := NewHaloMatchFetcher(stub)

	ids, err := f.FetchRecentMatchIDs(context.Background(), "123", 25)
	if err == nil {
		t.Fatal("FetchRecentMatchIDs: expected error, got nil")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("err does not wrap sentinel: %v", err)
	}
	if ids != nil {
		t.Errorf("ids = %v, want nil", ids)
	}
}

func TestHaloMatchFetcher_RejectsEmptyXUID(t *testing.T) {
	stub := &stubMatchHistoryClient{}
	f := NewHaloMatchFetcher(stub)

	_, err := f.FetchRecentMatchIDs(context.Background(), "   ", 25)
	if err == nil {
		t.Fatal("FetchRecentMatchIDs: expected error for empty xuid, got nil")
	}
	if stub.gotGamertag != "" {
		t.Errorf("client called with gamertag=%q, expected no call", stub.gotGamertag)
	}
}

func TestHaloMatchFetcher_ClampsCountToOne(t *testing.T) {
	stub := &stubMatchHistoryClient{}
	f := NewHaloMatchFetcher(stub)

	_, err := f.FetchRecentMatchIDs(context.Background(), "123", 0)
	if err != nil {
		t.Fatalf("FetchRecentMatchIDs: unexpected err: %v", err)
	}
	if stub.gotCount != 1 {
		t.Errorf("count = %d, want 1 (clamped from 0)", stub.gotCount)
	}
}
