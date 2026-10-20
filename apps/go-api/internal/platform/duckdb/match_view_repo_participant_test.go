package duckdb

import (
	"context"
	"database/sql"
	"testing"
)

// newParticipantsReader rend un SharedReader adossé à une DB in-memory isolée portant
// `match_participants` (colonnes lues par Q17bIsParticipant), remplie des couples donnés.
func newParticipantsReader(t *testing.T, rows ...[2]string) SharedReader {
	t.Helper()
	raw, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { raw.Close() })
	db := newTestDB(raw, ":memory:")
	ctx := context.Background()
	if _, err := db.Exec(ctx, `CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR)`); err != nil {
		t.Fatalf("schema: %v", err)
	}
	for _, r := range rows {
		if _, err := db.Exec(ctx, `INSERT INTO match_participants VALUES (?, ?)`, r[0], r[1]); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	return LegacySharedReader(db)
}

// TestIsParticipant_SnapshotMissRecheckedOnLive : un match absent du snapshot (joué après
// le dernier cut) mais présent dans le live est une participation confirmée — sans quoi la
// garde de participation répond 404 sur la vue et le rejeu d'un match récent. Un refus
// du live reste un refus.
func TestIsParticipant_SnapshotMissRecheckedOnLive(t *testing.T) {
	ctx := context.Background()
	live := newParticipantsReader(t, [2]string{"recent", "x1"}, [2]string{"old", "x1"})
	snap := newParticipantsReader(t, [2]string{"old", "x1"})
	pdb := &PlayerDB{SharedReader: live, XUID: "x1"}

	cases := []struct {
		match string
		want  bool
	}{
		{"old", true},      // confirmé par le snapshot
		{"recent", true},   // absent du snapshot, confirmé par le live
		{"foreign", false}, // absent des deux : refus
	}
	for _, c := range cases {
		repo := NewMatchViewRepo(pdb, "x1").WithSharedReader(snap)
		got, err := repo.IsParticipant(ctx, "x1", c.match)
		if err != nil {
			t.Fatalf("%s: %v", c.match, err)
		}
		if got != c.want {
			t.Errorf("IsParticipant(%s) = %v, want %v", c.match, got, c.want)
		}
	}
}
