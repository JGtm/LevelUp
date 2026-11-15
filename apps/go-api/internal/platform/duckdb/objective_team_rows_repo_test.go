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

// ForMatches : les lignes des DEUX camps des seuls matchs demandés, famille par ligne (dont le split
// KOTH / Bases par zone_scoring_ticks), camp et durée joints, pour plus d'un lot de requête.
func TestLoadObjectiveTeamRowsForMatches_FamillesDeuxCampsEtLots(t *testing.T) {
	repo, db := newObjectiveStatsRepoOnMem(t, true)
	insertParticipantTeam(t, db, "m1", "x1", 0, 600)
	insertParticipantTeam(t, db, "m1", "x2", 1, 600)
	insertObjectiveRow(t, db, "m1", "x1", map[string]any{"flag_captures": 2, "time_as_flag_carrier_seconds": 30.0})
	insertObjectiveRow(t, db, "m1", "x2", map[string]any{"flag_captures": 1})
	insertParticipantTeam(t, db, "m2", "x1", 0, 600)
	insertObjectiveRow(t, db, "m2", "x1", map[string]any{"zone_captures": 4, "zone_scoring_ticks": 0})
	insertParticipantTeam(t, db, "m3", "x1", 0, 600)
	insertObjectiveRow(t, db, "m3", "x1", map[string]any{"zone_captures": 2, "zone_scoring_ticks": 12})
	insertParticipantTeam(t, db, "hors", "x1", 0, 600)
	insertObjectiveRow(t, db, "hors", "x1", map[string]any{"flag_captures": 50})

	// Plus d'un lot : des identifiants sans ligne ne changent rien et ne cassent pas la requête.
	ids := []string{"m1", "m2", "m3"}
	for i := 0; i < objectiveTeamRowsChunk+10; i++ {
		ids = append(ids, "vide_"+string(rune('a'+i%26))+string(rune('a'+i/26%26))+string(rune('a'+i/676)))
	}
	got, err := repo.LoadObjectiveTeamRows(context.Background(), ids)
	if err != nil {
		t.Fatalf("LoadObjectiveTeamRows: %v", err)
	}
	if len(got) != 3 || len(got["m1"]) != 2 || len(got["hors"]) != 0 {
		t.Fatalf("matchs = %d, lignes de m1 = %d, want 3 matchs, 2 lignes (les deux camps), match hors lot absent", len(got), len(got["m1"]))
	}
	byXUID := map[string]narrative.ObjectiveTeamRow{}
	for _, r := range got["m1"] {
		byXUID[r.XUID] = r
	}
	if byXUID["x1"].TeamID != 0 || byXUID["x2"].TeamID != 1 || byXUID["x1"].Family != narrative.FamilyCTF ||
		byXUID["x1"].Values["time_as_flag_carrier_seconds"] != 30 || byXUID["x1"].TimePlayedSeconds != 600 {
		t.Errorf("ligne x1 = %+v", byXUID["x1"])
	}
	if got["m2"][0].Family != narrative.FamilyZonesStrongholds || got["m3"][0].Family != narrative.FamilyZonesKOTH {
		t.Errorf("familles m2/m3 = %s / %s, want Bases / Colline", got["m2"][0].Family, got["m3"][0].Family)
	}
	if empty, err := repo.LoadObjectiveTeamRows(context.Background(), nil); err != nil || len(empty) != 0 {
		t.Errorf("aucun match : (%v, %v), want (vide, nil)", empty, err)
	}
}

// Vue absente (titre sans match.objective.stats) : map vide, aucune erreur.
func TestLoadObjectiveTeamRowsForMatches_VueAbsenteSansErreur(t *testing.T) {
	repo, _ := newObjectiveStatsRepoOnMem(t, false)
	got, err := repo.LoadObjectiveTeamRows(context.Background(), []string{"m1"})
	if err != nil || len(got) != 0 {
		t.Fatalf("(%v, %v), want (vide, nil)", got, err)
	}
}

// Les coéquipiers non suivis se résolvent par shared.xuid_aliases ; un gamertag inconnu est absent.
func TestObjectiveStatsRepo_XUIDsByGamertag(t *testing.T) {
	repo, db := newObjectiveStatsRepoOnMem(t, true)
	if _, err := db.Exec(`INSERT INTO xuid_aliases (xuid, gamertag) VALUES ('x9', 'MateGT'), ('x10', 'MateGT')`); err != nil {
		t.Fatalf("seed alias: %v", err)
	}
	got, err := repo.XUIDsByGamertag(context.Background(), []string{"MateGT", "Inconnu"})
	if err != nil {
		t.Fatalf("XUIDsByGamertag: %v", err)
	}
	if len(got["MateGT"]) != 2 || len(got["Inconnu"]) != 0 {
		t.Errorf("got = %v, want deux xuids pour MateGT et rien pour Inconnu", got)
	}
}
