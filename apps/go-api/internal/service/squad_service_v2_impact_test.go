package service

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

func killEv(matchID, killerXUID, victimXUID string, timeMS int64) canonical.HighlightEvent {
	k, v := killerXUID, victimXUID
	return canonical.HighlightEvent{
		MatchID:    matchID,
		EventType:  string(canonical.EventKill),
		TimeMS:     timeMS,
		KillerXUID: &k,
		VictimXUID: &v,
	}
}

func mkSharedMatchImpact(matchID string, players map[string]canonical.Outcome) domain.SquadSharedMatch {
	playerRows := make(map[string]canonical.PlayerMatchRow, len(players))
	for gt, oc := range players {
		playerRows[gt] = canonical.PlayerMatchRow{
			Self: canonical.MatchParticipant{Outcome: oc},
		}
	}
	mainOutcome := canonical.Outcome("")
	for _, oc := range players {
		mainOutcome = oc
		break
	}
	return domain.SquadSharedMatch{
		MatchID:   matchID,
		StartedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
		Outcome:   mainOutcome,
		Players:   playerRows,
	}
}

func TestBuildImpactRolesMatrix_FirstBloodAndLastCasualty(t *testing.T) {
	t.Parallel()
	// p1 = main, p2 = friend1. m1 : p1 fait first kill, dernier kill victime = p2.
	events := []canonical.HighlightEvent{
		killEv("m1", "x_p1", "x_p2", 1000), // first
		killEv("m1", "x_p1", "x_p2", 5000),
		killEv("m1", "x_p2", "x_p1", 3000),
		killEv("m1", "x_p1", "x_p2", 9000), // last : victime = p2
	}
	squadOrder := []string{"main", "f1"}
	squadXUIDs := map[string]string{"main": "x_p1", "f1": "x_p2"}
	shared := []domain.SquadSharedMatch{
		mkSharedMatchImpact("m1", map[string]canonical.Outcome{
			"main": canonical.OutcomeWin,
			"f1":   canonical.OutcomeLoss,
		}),
	}
	matrix := BuildImpactRolesMatrix(events, squadOrder, squadXUIDs, shared)
	if len(matrix.MatchRows) != 1 {
		t.Fatalf("want 1 match row, got %d", len(matrix.MatchRows))
	}
	row := matrix.MatchRows[0]
	mainRoles := row.RolesByPlayer["main"]
	hasFirstBlood := false
	for _, c := range mainRoles {
		if c.RoleKey == "first_blood" {
			hasFirstBlood = true
		}
	}
	if !hasFirstBlood {
		t.Errorf("main should have first_blood, got cells %v", mainRoles)
	}
	f1Roles := row.RolesByPlayer["f1"]
	hasLastCasualty := false
	for _, c := range f1Roles {
		if c.RoleKey == "last_casualty" {
			hasLastCasualty = true
			if !c.Inverted {
				t.Error("last_casualty should be inverted")
			}
		}
	}
	if !hasLastCasualty {
		t.Errorf("f1 should have last_casualty, got cells %v", f1Roles)
	}
}

func TestBuildImpactRolesMatrix_PreservesSquadOrder(t *testing.T) {
	t.Parallel()
	matrix := BuildImpactRolesMatrix(
		nil,
		[]string{"main", "f1", "f2"},
		map[string]string{"main": "x1", "f1": "x2", "f2": "x3"},
		nil,
	)
	if len(matrix.SquadGamertags) != 3 || matrix.SquadGamertags[0] != "main" {
		t.Errorf("SquadGamertags should preserve order: %v", matrix.SquadGamertags)
	}
}

func TestBuildImpactRolesMatrix_MainOutcomeAndStartedAt(t *testing.T) {
	t.Parallel()
	shared := []domain.SquadSharedMatch{
		{
			MatchID:   "m1",
			StartedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
			Outcome:   canonical.OutcomeLoss,
			Players: map[string]canonical.PlayerMatchRow{
				"main": {Self: canonical.MatchParticipant{Outcome: canonical.OutcomeLoss}},
			},
		},
	}
	matrix := BuildImpactRolesMatrix(nil, []string{"main"}, map[string]string{"main": "x1"}, shared)
	if matrix.MatchRows[0].MainOutcome != canonical.OutcomeLoss {
		t.Errorf("MainOutcome want loss, got %s", matrix.MatchRows[0].MainOutcome)
	}
	if !matrix.MatchRows[0].StartedAt.Equal(shared[0].StartedAt) {
		t.Errorf("StartedAt mismatch")
	}
}

func TestBuildImpactRanking_OrderAndInverted(t *testing.T) {
	t.Parallel()
	matrix := domain.ImpactRolesMatrix{
		SquadGamertags: []string{"main", "f1", "f2"},
		MatchRows: []domain.ImpactRolesMatchRow{
			{
				MatchID: "m1",
				RolesByPlayer: map[string][]domain.ImpactRoleCell{
					"main": {{RoleKey: "first_blood"}, {RoleKey: "top_killer"}},
					"f1":   {{RoleKey: "false_brother", Inverted: true}},
				},
			},
			{
				MatchID: "m2",
				RolesByPlayer: map[string][]domain.ImpactRoleCell{
					"main": {{RoleKey: "first_blood"}},
					"f2":   {{RoleKey: "top_killer"}},
				},
			},
		},
	}
	ranking := BuildImpactRanking(matrix)
	if len(ranking) != 8 {
		t.Fatalf("want 8 role columns, got %d", len(ranking))
	}
	// Premiere colonne : first_blood
	if ranking[0].RoleKey != "first_blood" {
		t.Errorf("first column should be first_blood, got %s", ranking[0].RoleKey)
	}
	// main = 2 first_blood, f1 = 0, f2 = 0 -> ordre attendu : main, f1, f2 (alphabetic tiebreak)
	if ranking[0].Entries[0].Gamertag != "main" || ranking[0].Entries[0].Count != 2 {
		t.Errorf("first_blood: want main=2 first, got %+v", ranking[0].Entries[0])
	}
	// false_brother colonne : f1 a 1, marquee Inverted=true
	for _, r := range ranking {
		if r.RoleKey == "false_brother" {
			if !r.Inverted {
				t.Error("false_brother should be inverted")
			}
			if r.Entries[0].Gamertag != "f1" {
				t.Errorf("false_brother top: want f1, got %s", r.Entries[0].Gamertag)
			}
		}
	}
}

func TestBuildImpactRanking_AllPlayersAppearWithZero(t *testing.T) {
	t.Parallel()
	// Scenario : aucun role attribue. Tous les joueurs apparaissent quand meme
	// avec count=0 dans chaque colonne (le front rend une ligne vide).
	matrix := domain.ImpactRolesMatrix{
		SquadGamertags: []string{"main", "f1"},
		MatchRows:      nil,
	}
	ranking := BuildImpactRanking(matrix)
	if len(ranking[0].Entries) != 2 {
		t.Errorf("each role column should have entries for all squad members, got %d",
			len(ranking[0].Entries))
	}
	for _, e := range ranking[0].Entries {
		if e.Count != 0 {
			t.Errorf("entry count should be 0, got %d", e.Count)
		}
	}
}

func TestBuildImpactRanking_EmptySquadReturnsNil(t *testing.T) {
	t.Parallel()
	matrix := domain.ImpactRolesMatrix{}
	if got := BuildImpactRanking(matrix); got != nil {
		t.Errorf("empty squad: want nil, got %v", got)
	}
}
