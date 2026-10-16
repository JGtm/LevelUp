//go:build integration

package sync

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/migration"
)

// TestWithFriends_MatchesSQLVerdict : pour chaque cas de casAmis, le prédicat pur et la
// requête du recalcul (loadMatchesWithFriends) rendent le même verdict. C'est ce qui permet
// au premier écrit de poser la valeur que le recalcul confirmera (0 promotion, 0 démotion).
func TestWithFriends_MatchesSQLVerdict(t *testing.T) {
	shared := openBatchPathTestDB(t, migration.TargetShared)
	for i, c := range casAmis {
		matchID := fmt.Sprintf("sql-%d", i)
		for _, p := range c.participants {
			if _, err := shared.Exec(`INSERT INTO match_participants (match_id, xuid, team_id) VALUES (?, ?, ?)`,
				matchID, p.XUID, p.TeamID); err != nil {
				t.Fatalf("%s: seed participant: %v", c.nom, err)
			}
		}
		ids, err := loadMatchesWithFriends(context.Background(), shared, c.joueur, c.amis)
		if err != nil {
			t.Fatalf("%s: loadMatchesWithFriends: %v", c.nom, err)
		}
		sqlVerdict := false
		for _, id := range ids {
			sqlVerdict = sqlVerdict || id == matchID
		}
		if pure := withFriends(c.participants, c.joueur, c.amis); pure != sqlVerdict {
			t.Errorf("%s: prédicat pur = %v, requête SQL = %v", c.nom, pure, sqlVerdict)
		}
		if sqlVerdict != c.attendu {
			t.Errorf("%s: verdict SQL = %v, attendu du cas = %v", c.nom, sqlVerdict, c.attendu)
		}
	}
}

func lireIsWithFriends(t *testing.T, playerDB *sql.DB, matchID string) bool {
	t.Helper()
	var v bool
	if err := playerDB.QueryRow(`SELECT is_with_friends FROM player_match_enrichment_latest WHERE match_id = ?`, matchID).Scan(&v); err != nil {
		t.Fatalf("lecture is_with_friends %s: %v", matchID, err)
	}
	return v
}

// TestFirstWrite_PosesIsWithFriends_RecomputeConfirms : un match d'escouade (un ami dans
// l'équipe) porte is_with_friends = TRUE dès le premier écrit, AVANT tout post-sync ; un match
// où l'ami est adverse porte FALSE ; le recalcul de l'étape escouade qui suit ne promeut et ne
// démote aucun des deux (le premier écrit est déjà la valeur convergente).
func TestFirstWrite_PosesIsWithFriends_RecomputeConfirms(t *testing.T) {
	sharedDB := openBatchPathTestDB(t, migration.TargetShared)
	playerDB := openBatchPathTestDB(t, migration.TargetPlayer)
	e := &SyncEngine{gamertag: "Alice", xuid: "1111", titleSlug: "halo_infinite"}
	ctx := context.Background()

	if _, err := sharedDB.Exec(`INSERT INTO xuid_aliases (xuid, gamertag) VALUES ('2222', 'Bob')`); err != nil {
		t.Fatalf("seed alias: %v", err)
	}
	amis := ResolveFriendXUIDs(ctx, sharedDB, func() ([]string, error) { return []string{"Bob"}, nil }, "1111")
	if len(amis) != 1 {
		t.Fatalf("amis résolus = %v, want [2222]", amis)
	}

	persister := func(matchID string, bobTeam int) {
		t.Helper()
		team0, team1 := 0, bobTeam
		fm := &fetchedMatch{
			MatchID:  matchID,
			Registry: sampleRegistry(matchID),
			Participants: []domain.MatchParticipantRow{
				{MatchID: matchID, XUID: "1111", TeamID: &team0},
				{MatchID: matchID, XUID: "2222", TeamID: &team1},
				{MatchID: matchID, XUID: "3333", TeamID: &team1},
			},
			FriendXUIDs: amis,
		}
		if err := e.persistFetchedMatch(ctx, sharedDB, playerDB, &domain.SyncResult{}, fm); err != nil {
			t.Fatalf("persistFetchedMatch %s: %v", matchID, err)
		}
	}
	persister("squad-1", 0)
	persister("opposed-1", 1)

	if !lireIsWithFriends(t, playerDB, "squad-1") {
		t.Error("squad-1: is_with_friends = FALSE avant tout post-sync, want TRUE (ami dans l'équipe)")
	}
	if lireIsWithFriends(t, playerDB, "opposed-1") {
		t.Error("opposed-1: is_with_friends = TRUE, want FALSE (ami adverse)")
	}

	res, err := RecomputeIsWithFriendsCore(ctx, playerDB, sharedDB, "1111", []string{"Bob"}, false)
	if err != nil {
		t.Fatalf("RecomputeIsWithFriendsCore: %v", err)
	}
	if res.MatchesPromoted != 0 || res.MatchesDemoted != 0 {
		t.Errorf("recalcul après le premier écrit: promus=%d démotés=%d, want 0 et 0", res.MatchesPromoted, res.MatchesDemoted)
	}
}

// TestFirstWrite_NoFriendList_KeepsDefault : sans liste d'amis, le premier écrit ne pose rien
// (comportement inchangé : FALSE par défaut) et le recalcul promeut ensuite l'escouade.
func TestFirstWrite_NoFriendList_KeepsDefault(t *testing.T) {
	sharedDB := openBatchPathTestDB(t, migration.TargetShared)
	playerDB := openBatchPathTestDB(t, migration.TargetPlayer)
	e := &SyncEngine{gamertag: "Alice", xuid: "1111", titleSlug: "halo_infinite"}
	ctx := context.Background()
	if _, err := sharedDB.Exec(`INSERT INTO xuid_aliases (xuid, gamertag) VALUES ('2222', 'Bob')`); err != nil {
		t.Fatalf("seed alias: %v", err)
	}
	team := 0
	fm := &fetchedMatch{
		MatchID:  "squad-2",
		Registry: sampleRegistry("squad-2"),
		Participants: []domain.MatchParticipantRow{
			{MatchID: "squad-2", XUID: "1111", TeamID: &team},
			{MatchID: "squad-2", XUID: "2222", TeamID: &team},
		},
	}
	if err := e.persistFetchedMatch(ctx, sharedDB, playerDB, &domain.SyncResult{}, fm); err != nil {
		t.Fatalf("persistFetchedMatch: %v", err)
	}
	if lireIsWithFriends(t, playerDB, "squad-2") {
		t.Fatal("liste inconnue: is_with_friends doit rester FALSE au premier écrit")
	}
	res, err := RecomputeIsWithFriendsCore(ctx, playerDB, sharedDB, "1111", []string{"Bob"}, false)
	if err != nil {
		t.Fatalf("RecomputeIsWithFriendsCore: %v", err)
	}
	if res.MatchesPromoted != 1 {
		t.Errorf("promus = %d, want 1 (le recalcul reste l'autorité convergente)", res.MatchesPromoted)
	}
}
