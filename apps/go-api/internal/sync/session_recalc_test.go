//go:build integration

// Package sync — session_recalc_test.go : tests d'intégration du recalcul de sessions.
package sync

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/sync/testutil"
)

// seedSharedForRecalc est réservé à de futurs helpers de test.

func TestLookupFriendXUIDs_Found(t *testing.T) {
	db := testutil.NewInMemoryShared(t)

	_, err := db.Exec(`INSERT INTO xuid_aliases (xuid, gamertag) VALUES ('xuid-1', 'PlayerOne')`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	xuids := LookupFriendXUIDs(t.Context(), db, []string{"PlayerOne"})
	if len(xuids) != 1 || xuids[0] != "xuid-1" {
		t.Errorf("expected [xuid-1], got %v", xuids)
	}
}

func TestLookupFriendXUIDs_CaseInsensitive(t *testing.T) {
	db := testutil.NewInMemoryShared(t)

	_, err := db.Exec(`INSERT INTO xuid_aliases (xuid, gamertag) VALUES ('xuid-2', 'PlayerTwo')`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	xuids := LookupFriendXUIDs(t.Context(), db, []string{"playertwo"})
	if len(xuids) != 1 || xuids[0] != "xuid-2" {
		t.Errorf("expected [xuid-2], got %v", xuids)
	}
}

func TestLookupFriendXUIDs_NotFound(t *testing.T) {
	db := testutil.NewInMemoryShared(t)

	xuids := LookupFriendXUIDs(t.Context(), db, []string{"Inconnu"})
	if len(xuids) != 0 {
		t.Errorf("expected empty, got %v", xuids)
	}
}

func TestLookupFriendXUIDs_EmptySlice(t *testing.T) {
	db := testutil.NewInMemoryShared(t)
	xuids := LookupFriendXUIDs(t.Context(), db, nil)
	if xuids != nil {
		t.Errorf("expected nil, got %v", xuids)
	}
	xuids = LookupFriendXUIDs(t.Context(), db, []string{})
	if xuids != nil {
		t.Errorf("expected nil for empty slice, got %v", xuids)
	}
}

func TestLookupFriendXUIDs_PartialMatch(t *testing.T) {
	db := testutil.NewInMemoryShared(t)

	_, err := db.Exec(`INSERT INTO xuid_aliases (xuid, gamertag) VALUES ('xuid-3', 'PlayerThree')`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Un gamertag trouvé, un absent → seul le trouvé retourné.
	xuids := LookupFriendXUIDs(t.Context(), db, []string{"PlayerThree", "GhostPlayer"})
	if len(xuids) != 1 || xuids[0] != "xuid-3" {
		t.Errorf("expected [xuid-3], got %v", xuids)
	}
}

func TestLoadSessionMatchRowsDirect_EmptyDB(t *testing.T) {
	db := testutil.NewInMemoryShared(t)

	rows, err := loadSessionMatchRowsDirect(context.Background(), db, "xuid-nobody")
	if err != nil {
		t.Fatalf("loadSessionMatchRowsDirect: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("expected empty result, got %d rows", len(rows))
	}
}

func TestLoadSessionMatchRowsDirect_WithData(t *testing.T) {
	db := testutil.NewInMemoryShared(t)

	t1 := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)

	_, err := db.Exec(`INSERT INTO match_registry (match_id, start_time, is_ranked) VALUES ('m1', ?, FALSE), ('m2', ?, TRUE)`,
		t1, t2)
	if err != nil {
		t.Fatalf("seed registry: %v", err)
	}
	_, err = db.Exec(`INSERT INTO match_participants (match_id, xuid, gamertag, team_id, outcome, rank, score, time_played_seconds) VALUES
		('m1', 'xuid-a', 'PlayerA', 0, 2, 1, 100, 600),
		('m2', 'xuid-a', 'PlayerA', 0, 3, 2, 80, 600)`)
	if err != nil {
		t.Fatalf("seed participants: %v", err)
	}

	rows, err := loadSessionMatchRowsDirect(context.Background(), db, "xuid-a")
	if err != nil {
		t.Fatalf("loadSessionMatchRowsDirect: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	// Vérifie l'ordre chronologique.
	if rows[0].MatchID != "m1" || rows[1].MatchID != "m2" {
		t.Errorf("expected m1, m2 order; got %q, %q", rows[0].MatchID, rows[1].MatchID)
	}
	if rows[1].IsRanked != true {
		t.Errorf("expected m2 is_ranked=true, got %v", rows[1].IsRanked)
	}
}
