//go:build integration

// Package duckdb — squad_repo_departages_test.go : LES DEPARTAGES STABLES du lot perf L8
// (2026-09-23).
//
// Q29 (LoadTopTeammates) rend des coequipiers a egalite ; sans ordre total, leur ordre ET la
// coupe du LIMIT 50 parmi eux etaient ceux du plan d'execution de DuckDB, et la liste des
// coequipiers connus que la composition exacte exclut changeait d'une lecture a l'autre.
// Q32b (LoadMainTeamParticipants) n'avait pas d'ORDER BY : les badges d'impact ex aequo
// (Bourreau, Faux-frere, Heros silencieux) allaient au premier allie rencontre, dans un ordre
// que choisissait le plan. Chaque test lit DEUX fois la meme base : meme resultat, dans
// l'ordre ecrit en clair. Les lignes sont inserees dans un ordre qui n'est ni l'ordre voulu
// ni son inverse, pour que retirer un departage rende le test rouge au lieu de le laisser
// passer par chance.
package duckdb

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
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

// allieQ32b : une ligne de match_participants telle que le test la pose.
type allieQ32b struct {
	match, xuid           string
	equipe, issue         int
	frags, morts, assists int
}

// alliesDepartagesQ32b : trois matchs, inseres mb2, mb3, mb1. Dans chacun, trois allies a
// egalite sur le critere d'un badge, inseres de sorte que le plus petit xuid ne soit ni le
// premier ni le dernier du groupe :
//   - mb2, victoire : Heros silencieux ex aequo (6 assistances, 2 morts, hors Bourreau) entre
//     ...07, ...05 et ...09 ; porteur attendu ...05 ;
//   - mb3, defaite : Bourreau ex aequo (20 frags) entre ...03, ...01 et ...05 ; porteur ...01 ;
//   - mb1, defaite : Faux-frere ex aequo (11 morts, 0 assistance, hors Bourreau) entre ...13,
//     ...11 et ...15 ; porteur ...11.
//
// Un adversaire par match (autre equipe que le joueur principal) : Q32b ne le rend pas.
var alliesDepartagesQ32b = []allieQ32b{
	{"mb2", pTestXUID, 0, 2, 15, 5, 1},
	{"mb2", "2533274850000007", 0, 2, 5, 2, 6},
	{"mb2", "2533274850000005", 0, 2, 6, 2, 6},
	{"mb2", "2533274850000009", 0, 2, 4, 2, 6},
	{"mb2", "2533274850000006", 0, 2, 3, 4, 2},
	{"mb2", "2533274859000001", 1, 3, 20, 9, 0},
	{"mb3", "2533274860000003", 1, 3, 20, 5, 1},
	{"mb3", pTestXUID, 1, 3, 10, 6, 2},
	{"mb3", "2533274860000001", 1, 3, 20, 7, 3},
	{"mb3", "2533274860000005", 1, 3, 20, 8, 0},
	{"mb3", "2533274869000001", 0, 2, 30, 2, 0},
	{"mb1", "2533274870000013", 1, 3, 3, 11, 0},
	{"mb1", "2533274870000011", 1, 3, 2, 11, 0},
	{"mb1", "2533274870000015", 1, 3, 4, 11, 0},
	{"mb1", pTestXUID, 1, 3, 14, 8, 3},
	{"mb1", "2533274870000012", 1, 3, 5, 6, 4},
	{"mb1", "2533274879000001", 0, 2, 9, 3, 1},
}

// badgesParMatch rejoue la matrice d'impact (buildSquadImpactMatrix) sur les lignes de Q32b :
// les allies d'un match, DANS L'ORDRE DES LIGNES, donnes a analysis.ComputeMatchImpactFull.
// Rend, par match, le porteur de chaque badge.
func badgesParMatch(lignes []domain.AllyParticipant) map[string]map[string]string {
	parMatch := map[string][]analysis.ParticipantSnap{}
	for _, a := range lignes {
		parMatch[a.MatchID] = append(parMatch[a.MatchID], analysis.ParticipantSnap{
			XUID: a.XUID, Outcome: a.Outcome, Kills: a.Kills, Deaths: a.Deaths, Assists: a.Assists,
		})
	}
	out := make(map[string]map[string]string, len(parMatch))
	for m, snaps := range parMatch {
		out[m] = map[string]string{}
		for _, b := range analysis.ComputeMatchImpactFull(analysis.MatchImpactInput{Participants: snaps}) {
			out[m][b.BadgeKey] = b.PlayerXUID
		}
	}
	return out
}

// TestSquadRepo_Q32b_OrdreStableEtPorteursExAequo : deux lectures de l'equipe alliee, memes
// lignes dans le meme ordre — (match_id, xuid) — et, consequence voulue, le porteur d'un badge
// ex aequo est le plus petit xuid parmi les ex aequo.
func TestSquadRepo_Q32b_OrdreStableEtPorteursExAequo(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	for _, a := range alliesDepartagesQ32b {
		execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.match_participants
			(match_id, xuid, gamertag, team_id, outcome, kills, deaths, assists)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			a.match, a.xuid, "GT"+a.xuid, a.equipe, a.issue, a.frags, a.morts, a.assists)
	}
	repo := NewSquadRepo(pdb)
	matchs := []string{"mb3", "mb1", "mb2"}

	premiere, err := repo.LoadMainTeamParticipants(ctx, pTestXUID, matchs)
	if err != nil {
		t.Fatalf("LoadMainTeamParticipants (1re lecture) : %v", err)
	}
	seconde, err := repo.LoadMainTeamParticipants(ctx, pTestXUID, matchs)
	if err != nil {
		t.Fatalf("LoadMainTeamParticipants (2e lecture) : %v", err)
	}
	if !reflect.DeepEqual(premiere, seconde) {
		t.Errorf("deux lectures consecutives different :\n1re : %+v\n2e : %+v", premiere, seconde)
	}

	attendu := []string{
		"mb1 2533274870000011", "mb1 2533274870000012", "mb1 2533274870000013",
		"mb1 2533274870000015", "mb1 " + pTestXUID,
		"mb2 2533274850000005", "mb2 2533274850000006", "mb2 2533274850000007",
		"mb2 2533274850000009", "mb2 " + pTestXUID,
		"mb3 2533274860000001", "mb3 2533274860000003", "mb3 2533274860000005",
		"mb3 " + pTestXUID,
	}
	obtenu := make([]string, 0, len(premiere))
	for _, a := range premiere {
		obtenu = append(obtenu, a.MatchID+" "+a.XUID)
	}
	if !slices.Equal(obtenu, attendu) {
		t.Errorf("lignes hors de l'ordre (match_id, xuid) — %s\nobtenu :\n%s",
			premierEcart(obtenu, attendu), strings.Join(obtenu, "\n"))
	}

	// Les trois badges ex aequo ; le Bourreau sans egalite (joueur principal) en temoin.
	badges := badgesParMatch(premiere)
	for _, c := range []struct{ match, badge, porteur string }{
		{"mb2", "silent_hero", "2533274850000005"},
		{"mb3", "top_killer", "2533274860000001"},
		{"mb1", "false_brother", "2533274870000011"},
		{"mb1", "top_killer", pTestXUID},
		{"mb2", "top_killer", pTestXUID},
	} {
		if got := badges[c.match][c.badge]; got != c.porteur {
			t.Errorf("%s, %s : porteur %q, attendu %q (le plus petit xuid parmi les ex aequo)",
				c.match, c.badge, got, c.porteur)
		}
	}
}
