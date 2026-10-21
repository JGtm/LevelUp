//go:build integration

package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/port"
)

// Match classé abandonné : l'API rend un MMR à 0 pour tout le monde. Un 0 est une valeur
// ABSENTE (analysis.MMRKnown) : aucune lecture ne doit le servir comme un niveau, sinon les
// moyennes, les écarts et les pics du MMR sont faussés. La ligne stockée n'est pas réécrite ;
// c'est la lecture qui la neutralise, quelle que soit la requête qui la porte.
//
// Le test pose le MMR à 0 sur la ligne ET le laisse au repos sur un second joueur à 1200 (le
// coéquipier de Squad / TeammateMatches) pour vérifier qu'un MMR réel passe toujours.
func seedZeroMMRMatch(t *testing.T) *PlayerDB {
	t.Helper()
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	execOnSharedDBs(t, pdb, ctx, `UPDATE shared.match_participants SET team_mmr = 0, enemy_mmr = 0`)
	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.match_participants
		 (match_id,xuid,gamertag,outcome,kills,deaths,team_id,kda,accuracy,time_played_seconds,team_mmr,enemy_mmr)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		"m1", "xuid_mate_002", "TeamMate", 2, 8, 4, 1, 1.2, 0.55, 600, 0.0, 0.0)
	return pdb
}

func TestZeroMMR_HistoireDuJoueur(t *testing.T) {
	pdb := seedZeroMMRMatch(t)
	rows, err := NewMatchHistoryRepo(pdb).LoadAll(context.Background())
	if err != nil || len(rows) != 1 {
		t.Fatalf("LoadAll: rows=%d err=%v", len(rows), err)
	}
	if rows[0].TeamMMR != nil || rows[0].EnemyMMR != nil {
		t.Errorf("historique : MMR à 0 attendu absent, got team=%v enemy=%v", rows[0].TeamMMR, rows[0].EnemyMMR)
	}
}

func TestZeroMMR_StatsEtAccueil(t *testing.T) {
	pdb := seedZeroMMRMatch(t)
	ctx := context.Background()

	stats, err := NewStatsRepo(pdb).LoadStatsMatches(ctx)
	if err != nil || len(stats) != 1 {
		t.Fatalf("LoadStatsMatches: rows=%d err=%v", len(stats), err)
	}
	if stats[0].TeamMMR != nil || stats[0].EnemyMMR != nil {
		t.Errorf("stats : MMR à 0 attendu absent, got team=%v enemy=%v", stats[0].TeamMMR, stats[0].EnemyMMR)
	}

	home, err := NewHomeRepo(pdb).LoadHomeMatches(ctx)
	if err != nil || len(home) != 1 {
		t.Fatalf("LoadHomeMatches: rows=%d err=%v", len(home), err)
	}
	if home[0].TeamMMR != nil || home[0].EnemyMMR != nil {
		t.Errorf("accueil : MMR à 0 attendu absent, got team=%v enemy=%v", home[0].TeamMMR, home[0].EnemyMMR)
	}
}

func TestZeroMMR_CarriereEtMeilleursMatchs(t *testing.T) {
	pdb := seedZeroMMRMatch(t)
	ctx := context.Background()
	execOnSharedDBs(t, pdb, ctx, `UPDATE shared.match_participants SET time_played_seconds = 600`)

	top, err := NewCareerRepo(pdb).GetTopMatches(ctx)
	if err != nil {
		t.Fatalf("GetTopMatches: %v", err)
	}
	for _, m := range top {
		if m.TeamMMR != nil || m.EnemyMMR != nil {
			t.Errorf("meilleurs matchs : MMR à 0 attendu absent, got team=%v enemy=%v", m.TeamMMR, m.EnemyMMR)
		}
	}
}

func TestZeroMMR_VueDuMatch(t *testing.T) {
	pdb := seedZeroMMRMatch(t)
	ctx := context.Background()
	repo := NewMatchViewRepo(pdb, pTestXUID)

	stats, err := repo.GetPlayerMatchStats(ctx, pTestXUID, "m1")
	if err != nil || stats == nil {
		t.Fatalf("GetPlayerMatchStats: %v (stats=%v)", err, stats)
	}
	if stats.TeamMMR != nil || stats.EnemyMMR != nil {
		t.Errorf("stats du joueur : MMR à 0 attendu absent, got team=%v enemy=%v", stats.TeamMMR, stats.EnemyMMR)
	}

	board, err := repo.GetMatchScoreboard(ctx, "m1")
	if err != nil || len(board) == 0 {
		t.Fatalf("GetMatchScoreboard: rows=%d err=%v", len(board), err)
	}
	for _, s := range board {
		if s.TeamMMR != nil || s.EnemyMMR != nil {
			t.Errorf("tableau des scores %s : MMR à 0 attendu absent, got team=%v enemy=%v", s.XUID, s.TeamMMR, s.EnemyMMR)
		}
	}
}

func TestZeroMMR_LignesCanoniques(t *testing.T) {
	pdb := seedZeroMMRMatch(t)
	rows, err := NewPlayerMatchesRepo(pdb).Load(context.Background(), port.PlayerMatchFilters{})
	if err != nil || len(rows) != 1 {
		t.Fatalf("Load: rows=%d err=%v", len(rows), err)
	}
	e := rows[0].Enrichment
	if e.TeamMMR != nil || e.EnemyMMR != nil {
		t.Errorf("lignes canoniques : MMR à 0 attendu absent, got team=%v enemy=%v", e.TeamMMR, e.EnemyMMR)
	}
}

func TestZeroMMR_Escouade(t *testing.T) {
	pdb := seedZeroMMRMatch(t)
	ctx := context.Background()
	repo := NewSquadRepo(pdb)

	squad, err := repo.LoadSquadMatches(ctx, pTestXUID, "xuid_mate_002")
	if err != nil || len(squad) != 1 {
		t.Fatalf("LoadSquadMatches: rows=%d err=%v", len(squad), err)
	}
	if squad[0].TeamMMR != 0 {
		t.Errorf("escouade : le MMR à 0 reste 0 (= absent), got %v", squad[0].TeamMMR)
	}
	if squad[0].EnemyMMR != nil {
		t.Errorf("escouade : MMR adverse à 0 attendu absent, got %v", *squad[0].EnemyMMR)
	}

	mate, err := repo.LoadTeammateMatches(ctx, pTestXUID, "xuid_mate_002")
	if err != nil {
		t.Fatalf("LoadTeammateMatches: %v", err)
	}
	for _, m := range mate {
		if m.TeamMMR != 0 {
			t.Errorf("coéquipier : le MMR à 0 reste 0 (= absent), got %v", m.TeamMMR)
		}
	}
}

// Un MMR réel traverse toujours la lecture : la règle écarte le 0, pas le MMR.
func TestZeroMMR_UnMMRReelPasse(t *testing.T) {
	pdb := newTestPlayerDB(t) // m1 : 1200 / 1100
	rows, err := NewMatchHistoryRepo(pdb).LoadAll(context.Background())
	if err != nil || len(rows) != 1 {
		t.Fatalf("LoadAll: rows=%d err=%v", len(rows), err)
	}
	if rows[0].TeamMMR == nil || *rows[0].TeamMMR != 1200 || rows[0].EnemyMMR == nil || *rows[0].EnemyMMR != 1100 {
		t.Errorf("MMR réel attendu 1200/1100, got team=%v enemy=%v", rows[0].TeamMMR, rows[0].EnemyMMR)
	}
}
