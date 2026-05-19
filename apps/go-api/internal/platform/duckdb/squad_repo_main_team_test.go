//go:build integration

// Package duckdb — squad_repo_main_team_test.go : tests Q32b
// SquadRepo.LoadMainTeamParticipants (shared-only via SharedReader).
package duckdb

import (
	"context"
	"sort"
	"testing"
)

// TestSquadRepo_LoadMainTeamParticipants_EmptyInputs : matchIDs vide ou mainXUID
// vide → nil sans erreur (court-circuit défensif).
func TestSquadRepo_LoadMainTeamParticipants_EmptyInputs(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewSquadRepo(pdb)
	ctx := context.Background()

	got, err := repo.LoadMainTeamParticipants(ctx, pTestXUID, nil)
	if err != nil {
		t.Fatalf("nil matchIDs: %v", err)
	}
	if got != nil {
		t.Errorf("nil matchIDs: want nil, got %v", got)
	}

	got, err = repo.LoadMainTeamParticipants(ctx, "", []string{"m1"})
	if err != nil {
		t.Fatalf("empty mainXUID: %v", err)
	}
	if got != nil {
		t.Errorf("empty mainXUID: want nil, got %v", got)
	}
}

// TestSquadRepo_LoadMainTeamParticipants_TeamFilter : pour chaque match,
// retourne uniquement les participants ayant le même team_id que le main XUID
// (le main inclus). Les ennemis (team_id différent) sont exclus.
func TestSquadRepo_LoadMainTeamParticipants_TeamFilter(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()

	// m1 (seed) : main team_id=1 (cf. seedPlayerSchema, avant-dernier arg = 1).
	// Ajout ally1 (team=1, même que main) + enemy1 (team=0).
	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.match_participants (match_id, xuid, gamertag, kills, deaths, assists, outcome, team_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"m1", "xuid_ally1", "Ally1", 5, 2, 3, 2, 1)
	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.match_participants (match_id, xuid, gamertag, kills, deaths, assists, outcome, team_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"m1", "xuid_enemy1", "Enemy1", 8, 4, 1, 3, 0)

	// m_mt2 : new match, main team_id=0 + ally2 team_id=0 + enemy2 team_id=1.
	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.match_registry (match_id) VALUES (?)`, "m_mt2")
	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.match_participants (match_id, xuid, gamertag, kills, deaths, assists, outcome, team_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"m_mt2", pTestXUID, pTestGamertag, 12, 6, 4, 2, 0)
	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.match_participants (match_id, xuid, gamertag, kills, deaths, assists, outcome, team_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"m_mt2", "xuid_ally2", "Ally2", 9, 5, 2, 2, 0)
	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.match_participants (match_id, xuid, gamertag, kills, deaths, assists, outcome, team_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"m_mt2", "xuid_enemy2", "Enemy2", 7, 3, 0, 3, 1)

	repo := NewSquadRepo(pdb)
	got, err := repo.LoadMainTeamParticipants(ctx, pTestXUID, []string{"m1", "m_mt2"})
	if err != nil {
		t.Fatalf("LoadMainTeamParticipants: %v", err)
	}
	// Attendu : 4 rows (m1: main + ally1) + (m_mt2: main + ally2).
	if len(got) != 4 {
		t.Fatalf("attendu 4 rows, obtenu %d : %v", len(got), got)
	}

	// Reconstruction map (match_id, xuid) → row pour vérifications stables.
	keys := make([]string, 0, len(got))
	by := make(map[string]struct{ K, D, A int })
	for _, r := range got {
		key := r.MatchID + "|" + r.XUID
		keys = append(keys, key)
		by[key] = struct{ K, D, A int }{r.Kills, r.Deaths, r.Assists}
	}
	sort.Strings(keys)
	want := []string{
		"m1|" + pTestXUID,
		"m1|xuid_ally1",
		"m_mt2|" + pTestXUID,
		"m_mt2|xuid_ally2",
	}
	sort.Strings(want)
	for i, k := range want {
		if keys[i] != k {
			t.Errorf("row[%d] key = %q, want %q", i, keys[i], k)
		}
	}
	// Vérification ennemis exclus.
	if _, ok := by["m1|xuid_enemy1"]; ok {
		t.Error("enemy1 ne devrait pas être retourné (team_id différent)")
	}
	if _, ok := by["m_mt2|xuid_enemy2"]; ok {
		t.Error("enemy2 ne devrait pas être retourné (team_id différent)")
	}
}
