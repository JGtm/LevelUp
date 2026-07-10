// teammates_friends_test.go : tests purs pour le filtre amis-only du dropdown
// (§3 plan Squad/Sessions). Indépendant de DuckDB.
package teammates

import (
	"testing"

	"levelup/go-api/internal/domain"
)

func TestFilterTopRowsToFriends_EmptyFriendsReturnsNil(t *testing.T) {
	rows := []domain.TopTeammateRow{
		{XUID: "x1", Gamertag: "Alice"},
		{XUID: "x2", Gamertag: "Bob"},
	}
	out := filterTopRowsToFriends(rows, nil)
	if out != nil {
		t.Fatalf("expected nil, got %v", out)
	}
	out = filterTopRowsToFriends(rows, []string{})
	if out != nil {
		t.Fatalf("expected nil for empty slice, got %v", out)
	}
}

func TestFilterTopRowsToFriends_KeepsOnlyMatchingGamertags(t *testing.T) {
	rows := []domain.TopTeammateRow{
		{XUID: "x1", Gamertag: "Alice"},
		{XUID: "x2", Gamertag: "Bob"},
		{XUID: "x3", Gamertag: "Charlie"},
	}
	out := filterTopRowsToFriends(rows, []string{"Bob", "Alice"})
	if len(out) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(out))
	}
	gts := map[string]bool{out[0].Gamertag: true, out[1].Gamertag: true}
	if !gts["Alice"] || !gts["Bob"] {
		t.Fatalf("expected Alice + Bob, got %v", gts)
	}
}

func TestFilterTopRowsToFriends_CaseInsensitiveAndTrim(t *testing.T) {
	rows := []domain.TopTeammateRow{
		{XUID: "x1", Gamertag: "Alice"},
		{XUID: "x2", Gamertag: "Bob"},
	}
	out := filterTopRowsToFriends(rows, []string{"  ALICE  ", "bob"})
	if len(out) != 2 {
		t.Fatalf("expected 2 rows after trim+lowercase match, got %d", len(out))
	}
}

func TestFilterTopRowsToFriends_NonFriendDropped(t *testing.T) {
	rows := []domain.TopTeammateRow{
		{XUID: "x1", Gamertag: "Alice"},
		{XUID: "x2", Gamertag: "Random"},
	}
	out := filterTopRowsToFriends(rows, []string{"Alice"})
	if len(out) != 1 || out[0].Gamertag != "Alice" {
		t.Fatalf("expected only Alice, got %v", out)
	}
}

func TestFilterTopRowsToFriends_EmptyGamertagInListIgnored(t *testing.T) {
	rows := []domain.TopTeammateRow{
		{XUID: "x1", Gamertag: "Alice"},
	}
	// "" et "   " ne doivent pas matcher Alice
	out := filterTopRowsToFriends(rows, []string{"", "   "})
	if len(out) != 0 {
		t.Fatalf("expected 0 rows when only empty entries, got %d", len(out))
	}
}
