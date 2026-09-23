//go:build integration

// Package duckdb — squad_repo_departages_test.go : LES DEPARTAGES STABLES du lot perf L8
// (2026-09-23).
//
// Q29 (LoadTopTeammates) rend des coequipiers a egalite ; sans ordre total, leur ordre ET la
// coupe du LIMIT 50 parmi eux etaient ceux du plan d'execution de DuckDB, et la liste des
// coequipiers connus que la composition exacte exclut changeait d'une lecture a l'autre.
// Chaque test lit DEUX fois la meme base : meme resultat, dans l'ordre ecrit en clair. Les
// lignes sont inserees dans un ordre qui n'est ni l'ordre voulu ni son inverse, pour que
// retirer un departage rende le test rouge au lieu de le laisser passer par chance.
package duckdb

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// premierEcart decrit la premiere position ou deux listes different (message de test lisible
// sur des listes de 50 lignes).
func premierEcart(obtenu, attendu []string) string {
	for i := 0; i < len(obtenu) && i < len(attendu); i++ {
		if obtenu[i] != attendu[i] {
			return fmt.Sprintf("position %d : obtenu %q, attendu %q", i, obtenu[i], attendu[i])
		}
	}
	return fmt.Sprintf("longueurs : obtenu %d, attendu %d", len(obtenu), len(attendu))
}

// matchAvecAmis pose un match « avec amis » du joueur principal (equipe 0, issue donnee) :
// Q29 ne lit que ceux-la.
func matchAvecAmis(t *testing.T, pdb *PlayerDB, matchID string, issue int) {
	t.Helper()
	ctx := context.Background()
	if _, err := pdb.Player.Exec(ctx, `INSERT INTO player_match_enrichment
		(match_id, performance_score, session_id, session_label, dominance_flag, is_with_friends, is_excluded)
		VALUES (?, 50.0, 7, 'Session 7', 0, TRUE, FALSE)`, matchID); err != nil {
		t.Fatalf("enrichment %s : %v", matchID, err)
	}
	execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.match_participants (match_id, xuid, gamertag, outcome, team_id)
		VALUES (?, ?, ?, ?, 0)`, matchID, pTestXUID, pTestGamertag, issue)
}

// coequipier pose un coequipier (equipe 0) sur des matchs ; l'issue d'un match est celle du
// joueur principal, que Q29 compte (wins_together).
func coequipier(t *testing.T, pdb *PlayerDB, xuid string, matchs ...string) {
	t.Helper()
	for _, m := range matchs {
		execOnSharedDBs(t, pdb, context.Background(), `INSERT INTO shared.match_participants (match_id, xuid, gamertag, outcome, team_id)
			VALUES (?, ?, ?, 0, 0)`, m, xuid, "GT"+xuid)
	}
}

// seedDepartagesQ29 : 54 coequipiers sur six matchs « avec amis » (trois victoires du joueur
// principal, trois defaites), dont trois groupes a egalite :
//   - 45 « piliers » a (3 matchs, 3 victoires), inseres par xuid DECROISSANT ;
//   - 3 a 3 matchs et 2, 1, 0 victoire, dont l'ordre des xuids est l'INVERSE de celui des
//     victoires (retirer `wins_together DESC` les retourne) ;
//   - 5 a (2 matchs, 1 victoire), inseres 5, 3, 1, 4, 2 : la coupe du LIMIT 50 passe au
//     milieu du groupe, seuls les deux plus petits xuids entrent ;
//   - 1 a un seul match, sous la coupe.
func seedDepartagesQ29(t *testing.T, pdb *PlayerDB) {
	t.Helper()
	for _, m := range []struct {
		id    string
		issue int
	}{{"mq_l2", 3}, {"mq_w3", 2}, {"mq_w1", 2}, {"mq_l3", 3}, {"mq_w2", 2}, {"mq_l1", 3}} {
		matchAvecAmis(t, pdb, m.id, m.issue)
	}
	for i := 44; i >= 0; i-- {
		coequipier(t, pdb, fmt.Sprintf("25332748100000%02d", i), "mq_w1", "mq_w2", "mq_w3")
	}
	coequipier(t, pdb, "2533274820000001", "mq_l1", "mq_l2", "mq_l3") // 0 victoire
	coequipier(t, pdb, "2533274820000003", "mq_w1", "mq_w2", "mq_l1") // 2 victoires
	coequipier(t, pdb, "2533274820000002", "mq_w1", "mq_l1", "mq_l2") // 1 victoire
	for _, n := range []int{5, 3, 1, 4, 2} {
		coequipier(t, pdb, fmt.Sprintf("253327483000000%d", n), "mq_w1", "mq_l1")
	}
	coequipier(t, pdb, "2533274840000001", "mq_w2")
}

// TestSquadRepo_Q29_DepartageStable : deux lectures du top 50, meme liste dans le meme ordre —
// matchs communs DESC, victoires DESC, xuid ASC — et la coupe du LIMIT 50 garde les plus
// petits xuids du groupe a egalite qu'elle traverse.
func TestSquadRepo_Q29_DepartageStable(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedDepartagesQ29(t, pdb)
	repo := NewSquadRepo(pdb)

	premiere, err := repo.LoadTopTeammates(context.Background(), pTestXUID)
	if err != nil {
		t.Fatalf("LoadTopTeammates (1re lecture) : %v", err)
	}
	seconde, err := repo.LoadTopTeammates(context.Background(), pTestXUID)
	if err != nil {
		t.Fatalf("LoadTopTeammates (2e lecture) : %v", err)
	}
	if !reflect.DeepEqual(premiere, seconde) {
		t.Errorf("deux lectures consecutives different :\n1re : %+v\n2e : %+v", premiere, seconde)
	}

	attendu := make([]string, 0, 50)
	for i := 0; i <= 44; i++ {
		attendu = append(attendu, fmt.Sprintf("25332748100000%02d 3/3", i))
	}
	attendu = append(attendu,
		"2533274820000003 3/2", "2533274820000002 3/1", "2533274820000001 3/0",
		"2533274830000001 2/1", "2533274830000002 2/1")
	obtenu := make([]string, 0, len(premiere))
	for _, r := range premiere {
		obtenu = append(obtenu, fmt.Sprintf("%s %d/%d", r.XUID, r.GamesTogether, r.WinsTogether))
	}
	if !slices.Equal(obtenu, attendu) {
		t.Errorf("top 50 hors de l'ordre (matchs DESC, victoires DESC, xuid ASC) — %s\nobtenu :\n%s",
			premierEcart(obtenu, attendu), strings.Join(obtenu, "\n"))
	}
}
