package duckdb

// match_history_team_score_test.go — LA PERMUTATION « MON CAMP / CAMP ADVERSE », éprouvée.
//
// `applyTeamScore` reproduit en Go le `CASE WHEN team_id = 0 THEN … ELSE …` du SQL, pour les
// POINTS et pour les MANCHES. Deux erreurs y sont invisibles à la relecture et se voient
// seulement à l'écran :
//   - permuter les points sans permuter les manches : la ligne afficherait les manches d'un
//     camp à côté des points de l'autre, et donnerait « 1 - 2 » sur une victoire ;
//   - tester `== 1` au lieu de `!= 0` : les matchs FFA (Halo Infinite numérote un camp par
//     joueur, donc `team_id >= 2`) basculeraient dans l'autre branche, en contradiction avec
//     le chemin jumeau de l'escouade qui garde le CASE SQL.
//
// TOUTES LES VALEURS DE CES TÉMOINS SONT DISTINCTES : avec un 2-2 ou un 50-50, une
// permutation ratée passerait inaperçue.

import (
	"testing"

	"levelup/go-api/internal/domain"
)

func ip(v int) *int { return &v }

// scoresFixture : quatre nombres tous différents, et un total qui ne ressemble à aucun.
func scoresFixture() teamScorePair {
	return teamScorePair{
		team0: ip(181), team1: ip(186),
		rounds0: ip(2), rounds1: ip(1),
		total: ip(3),
	}
}

func TestApplyTeamScore_Camp0(t *testing.T) {
	var row domain.MatchHistoryRawRow
	applyTeamScore(&row, ip(0), scoresFixture())
	assertSide(t, row, 181, 186, 2, 1)
}

// Témoin 293a763e vu du camp 1 : points ET manches doivent basculer ENSEMBLE.
func TestApplyTeamScore_Camp1PermuteLesDeux(t *testing.T) {
	var row domain.MatchHistoryRawRow
	applyTeamScore(&row, ip(1), scoresFixture())
	assertSide(t, row, 186, 181, 1, 2)
}

// Un joueur sans camp transmis est traité comme le camp 0 (comportement historique).
func TestApplyTeamScore_CampInconnu(t *testing.T) {
	var row domain.MatchHistoryRawRow
	applyTeamScore(&row, nil, scoresFixture())
	assertSide(t, row, 181, 186, 2, 1)
}

// FFA : `team_id >= 2` tombe dans le ELSE du CASE SQL, comme le chemin de l'escouade.
func TestApplyTeamScore_FFASuitLeCaseSQL(t *testing.T) {
	var row domain.MatchHistoryRawRow
	applyTeamScore(&row, ip(3), scoresFixture())
	assertSide(t, row, 186, 181, 1, 2)
}

// Le total de manches est une grandeur du MATCH : il ne se permute pas.
func TestApplyTeamScore_TotalNePermutePas(t *testing.T) {
	for _, teamID := range []*int{nil, ip(0), ip(1), ip(5)} {
		var row domain.MatchHistoryRawRow
		applyTeamScore(&row, teamID, scoresFixture())
		if row.RoundsTotal == nil || *row.RoundsTotal != 3 {
			t.Errorf("team_id %v : total = %v, want 3", teamID, row.RoundsTotal)
		}
	}
}

// Colonnes NULL : elles le restent. Un nil n'est pas un zéro.
func TestApplyTeamScore_NullResteNull(t *testing.T) {
	var row domain.MatchHistoryRawRow
	applyTeamScore(&row, ip(1), teamScorePair{team0: ip(50), team1: ip(43)})
	if row.MyRoundsWon != nil || row.EnemyRoundsWon != nil || row.RoundsTotal != nil {
		t.Error("manches absentes : les champs doivent rester nil, jamais devenir 0")
	}
	if row.MyTeamScore == nil || *row.MyTeamScore != 43 {
		t.Errorf("les points doivent tout de même permuter : got %v, want 43", row.MyTeamScore)
	}
}

func assertSide(t *testing.T, row domain.MatchHistoryRawRow, myPts, enemyPts, myRounds, enemyRounds int) {
	t.Helper()
	got := [4]*int{row.MyTeamScore, row.EnemyTeamScore, row.MyRoundsWon, row.EnemyRoundsWon}
	want := [4]int{myPts, enemyPts, myRounds, enemyRounds}
	names := [4]string{"MyTeamScore", "EnemyTeamScore", "MyRoundsWon", "EnemyRoundsWon"}
	for i := range got {
		if got[i] == nil || *got[i] != want[i] {
			t.Errorf("%s = %v, want %d", names[i], got[i], want[i])
		}
	}
}
