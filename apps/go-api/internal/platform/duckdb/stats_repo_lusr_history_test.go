package duckdb

// stats_repo_lusr_history_test.go — ce que rend Q24LUSRHistory (StatsRepo.LoadLUSRHistory) sur la
// table append-only match_skill_rank du schéma réel des migrations : UNE ligne LUSR par match, la
// plus récemment écrite (written_at, puis id), jamais une ligne CSR (autre échelle) ni la ligne
// d'audit LUSR_V2.

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/migration"
)

// openMigratedMemoryPlayer ouvre une player DB en mémoire au schéma réel des migrations. Une seule
// connexion : chaque connexion d'une base :memory: serait une base distincte.
func openMigratedMemoryPlayer(t *testing.T) *PlayerDB {
	t.Helper()
	sqlDB, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open :memory: : %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := migration.RunForDB(sqlDB, migration.TargetPlayer); err != nil {
		t.Fatalf("RunForDB(player) : %v", err)
	}
	return &PlayerDB{Player: newTestDB(sqlDB, ":memory:"), XUID: "xuid_lusr_history"}
}

// insertMSR ajoute une ligne à match_skill_rank (INSERT pur, comme les persisters). written est un
// littéral TIMESTAMP UTC.
func insertMSR(t *testing.T, pdb *PlayerDB, matchID, ratingType string, value, deviation float64, group, written string) {
	t.Helper()
	q := fmt.Sprintf(`INSERT INTO match_skill_rank
		(match_id, rating_type, rating_value, rating_deviation, playlist_group, written_at)
		VALUES (?, ?, ?, ?, ?, TIMESTAMP '%s')`, written)
	if _, err := pdb.Player.Exec(context.Background(), q, matchID, ratingType, value, deviation, group); err != nil {
		t.Fatalf("insert msr %s/%s : %v", matchID, ratingType, err)
	}
}

func TestStatsRepo_LoadLUSRHistory_UneLigneLUSRParMatch(t *testing.T) {
	pdb := openMigratedMemoryPlayer(t)

	// (a) un match à une seule ligne LUSR.
	insertMSR(t, pdb, "a_simple", "LUSR", 1200, 80, "arena", "2026-01-01 10:00:00")
	// (b) un match classé à double ligne : CSR (échelle ~1500) et LUSR (échelle mu).
	insertMSR(t, pdb, "b_classe", "CSR", 1650, 0, "ranked", "2026-01-02 10:00:00")
	insertMSR(t, pdb, "b_classe", "LUSR", 1210, 70, "ranked", "2026-01-02 10:00:00")
	// (c) un match dont la ligne LUSR a été réécrite : deux written_at.
	insertMSR(t, pdb, "c_reecrit", "LUSR", 1100, 90, "arena", "2026-01-03 09:00:00")
	insertMSR(t, pdb, "c_reecrit", "LUSR", 1180, 60, "arena", "2026-01-03 11:00:00")
	// (d) la ligne d'audit LUSR_V2 (même valeur) ne double pas le point.
	insertMSR(t, pdb, "d_audit", "LUSR", 1300, 50, "arena", "2026-01-04 10:00:00")
	insertMSR(t, pdb, "d_audit", "LUSR_V2", 1300, 50, "arena", "2026-01-04 10:00:00")
	// (e) deux écritures au même written_at : la dernière insérée (id le plus grand) gagne.
	insertMSR(t, pdb, "e_egalite", "LUSR", 1400, 40, "arena", "2026-01-05 10:00:00")
	insertMSR(t, pdb, "e_egalite", "LUSR", 1450, 35, "arena", "2026-01-05 10:00:00")

	got, err := NewStatsRepo(pdb).LoadLUSRHistory(context.Background())
	if err != nil {
		t.Fatalf("LoadLUSRHistory : %v", err)
	}
	for _, r := range got {
		t.Logf("rendu : %s valeur=%v ecart=%v groupe=%s", r.MatchID, r.RatingValue, r.RatingDeviation, r.PlaylistGroup)
	}

	want := []domain.LUSRMatchRating{
		{MatchID: "a_simple", RatingValue: 1200, RatingDeviation: 80, PlaylistGroup: "arena"},
		{MatchID: "b_classe", RatingValue: 1210, RatingDeviation: 70, PlaylistGroup: "ranked"},
		{MatchID: "c_reecrit", RatingValue: 1180, RatingDeviation: 60, PlaylistGroup: "arena"},
		{MatchID: "d_audit", RatingValue: 1300, RatingDeviation: 50, PlaylistGroup: "arena"},
		{MatchID: "e_egalite", RatingValue: 1450, RatingDeviation: 35, PlaylistGroup: "arena"},
	}
	if len(got) != len(want) {
		t.Fatalf("%d ligne(s) rendue(s), attendu %d (une par match)", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ligne %d : %+v, attendu %+v", i, got[i], want[i])
		}
	}
}
