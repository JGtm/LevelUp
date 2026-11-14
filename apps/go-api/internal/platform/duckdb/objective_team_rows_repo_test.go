//go:build cgo

// Package duckdb — objective_team_rows_repo_test.go : LoadObjectiveTeamRowsForPlayer, les
// lignes des DEUX camps des matchs d'un joueur, valeurs brutes, camp joint.
package duckdb

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/analysis/narrative"
)

func insertParticipantTeam(t *testing.T, db *sql.DB, matchID, xuid string, team, tp int) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO match_participants (match_id, xuid, gamertag, team_id, time_played_seconds)
		VALUES (?, ?, ?, ?, ?)`, matchID, xuid, "GT_"+xuid, team, tp); err != nil {
		t.Fatalf("insert participant: %v", err)
	}
}

func TestLoadObjectiveTeamRowsForPlayer_DeuxCampsEtPartDeLEquipe(t *testing.T) {
	_, db := newObjectiveStatsRepoOnMem(t, true)
	// m1 : CTF, équipe 0 = x1 (porteur), x2 (escorte), x3 (tueur) ; équipe 1 = x4.
	insertParticipantTeam(t, db, "m1", "x1", 0, 600)
	insertParticipantTeam(t, db, "m1", "x2", 0, 600)
	insertParticipantTeam(t, db, "m1", "x3", 0, 600)
	insertParticipantTeam(t, db, "m1", "x4", 1, 600)
	insertObjectiveRow(t, db, "m1", "x1", map[string]any{"flag_captures": 2, "flag_steals": 2, "flag_grabs": 9})
	insertObjectiveRow(t, db, "m1", "x2", map[string]any{"flag_capture_assists": 2, "flag_returns": 2, "flag_carriers_killed": 2})
	insertObjectiveRow(t, db, "m1", "x3", map[string]any{"flag_captures": 0})
	insertObjectiveRow(t, db, "m1", "x4", map[string]any{"flag_captures": 4})
	// m2 : un match où x1 n'a pas joué : ne doit pas être rendu.
	insertParticipantTeam(t, db, "m2", "x9", 0, 600)
	insertObjectiveRow(t, db, "m2", "x9", map[string]any{"flag_captures": 1})

	got, err := LoadObjectiveTeamRowsForPlayer(context.Background(), db, "x1")
	if err != nil {
		t.Fatalf("LoadObjectiveTeamRowsForPlayer: %v", err)
	}
	if len(got) != 1 || len(got["m1"]) != 4 {
		t.Fatalf("matchs/lignes = %d/%d, want 1 match de 4 lignes (les deux camps, m2 exclu)", len(got), len(got["m1"]))
	}
	for _, xuid := range []string{"x1", "x2", "x3"} {
		share, ok := narrative.ComputeObjectiveTeamShare(got["m1"], xuid)
		if !ok {
			t.Fatalf("%s : part absente", xuid)
		}
		if share.Family != narrative.FamilyCTF || share.TeamPoints != 24 {
			t.Errorf("%s : %+v, want CTF et 24 points d'équipe (x4 adverse exclu, flag_grabs hors poids)", xuid, share)
		}
	}
	if s, _ := narrative.ComputeObjectiveTeamShare(got["m1"], "x1"); s.Points != 14 {
		t.Errorf("points de x1 = %v, want 14", s.Points)
	}
}

// La lecture passe par la vue `_latest` : la dernière passe écrite d'un joueur sert.
func TestLoadObjectiveTeamRowsForPlayer_VueLatest(t *testing.T) {
	_, db := newObjectiveStatsRepoOnMem(t, true)
	insertParticipantTeam(t, db, "m1", "x1", 0, 600)
	if _, err := db.Exec(`
		INSERT INTO match_objective_stats (match_id, xuid, flag_captures, written_at) VALUES ('m1', 'x1', 1, TIMESTAMP '2026-01-01 00:00:00');
		INSERT INTO match_objective_stats (match_id, xuid, flag_captures, written_at) VALUES ('m1', 'x1', 4, TIMESTAMP '2026-01-02 00:00:00');
	`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	got, err := LoadObjectiveTeamRowsForPlayer(context.Background(), db, "x1")
	if err != nil || len(got["m1"]) != 1 {
		t.Fatalf("rows = %+v err = %v, want 1 ligne", got, err)
	}
	if v := got["m1"][0].Values["flag_captures"]; v != 4 {
		t.Errorf("flag_captures = %v, want 4 (dernière passe)", v)
	}
}

// Titre sans match_objective_stats (Halo 5) : map vide, aucune erreur, aucune panique.
func TestLoadObjectiveTeamRowsForPlayer_SansTableAbsentSansErreur(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	got, err := LoadObjectiveTeamRowsForPlayer(context.Background(), db, "x1")
	if err != nil || len(got) != 0 {
		t.Fatalf("got = %+v err = %v, want map vide et nil", got, err)
	}
	if got, err := LoadObjectiveTeamRowsForPlayer(context.Background(), db, ""); err != nil || len(got) != 0 {
		t.Fatalf("xuid vide : %+v %v", got, err)
	}
}
