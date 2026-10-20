package duckdb

// annuaire_fenetres_test.go — L'ANNUAIRE DES NOMS LIE SES LISTES EN UN PARAMÈTRE (item B.7 du plan
// perf « compaction et périmètre joueur », 2026-09-27).
//
// Ce que ce test verrouille, et pourquoi il peut échouer :
//
//  1. un xuid que seul le kill-feed nomme reçoit ce nom, en portée LECTURE (rivaux) comme en
//     portée BASE (Relations) : rouge si une liste de l'annuaire est vide, mal ordonnée ou mal
//     typée (le nom tombe au libellé masqué, ou la requête échoue) ;
//  2. la jambe kill-feed de l'annuaire ne lit, sous sa fenêtre `_latest`, que les matchs de la
//     lecture : rouge si la liste de match_id ne borne plus la fenêtre (la semi-jointure, par
//     exemple, ne descend pas sous une fenêtre — cf. analysis/sql_liste.go) ;
//  3. aucune requête de l'annuaire ne lie plus d'une liste par paramètre : rouge si un
//     `IN (?, ?, …)` revient (le nombre d'arguments suivrait la taille des listes).

import (
	"context"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
)

// xuidNommeParLeKillFeed : présent dans deux de mes matchs, sans alias ni gamertag de
// participant ; seul le journal des morts porte son nom (le gamertag de la ligne = son xuid).
const xuidNommeParLeKillFeed = "2533274000000199"

func TestAnnuaire_ListesLieesEnUnParametre(t *testing.T) {
	b := newBaseNotee(t)
	seedFenetresTactiques(t, b.pdb, matchsDuJoueurFenetres)
	seedMatchsDesAutres(t, b.pdb, matchsDesAutresFenetres)
	for i, id := range []string{"f01", "f02"} {
		tacExec(t, b.pdb, `INSERT INTO match_participants (match_id, xuid, gamertag, team_id, outcome)
			VALUES (?, ?, NULL, 1, 3)`, id, xuidNommeParLeKillFeed)
		tacKill(t, b.pdb, id, tacXUIDMoi, xuidNommeParLeKillFeed, 9000+i, true)
	}
	// Les lignes de mes matchs dans le journal : trois morts par match, plus les deux ci-dessus.
	borne := matchsDuJoueurFenetres*mortsParMatchFenetres + 2
	ctx := context.Background()
	repo := NewCareerRepo(b.pdb)
	b.carnet.vider()

	_, vic, err := repo.GetRivals(ctx)
	if err != nil {
		t.Fatalf("GetRivals: %v", err)
	}
	exigerNomDuKillFeed(t, "GetRivals (portée lecture)", vic2noms(vic))
	exigerUnArgumentParListe(t, b)
	exigerFenetresBornees(t, b, "GetRivals (portée lecture)", borne, 1)

	rel, err := repo.GetRelations(ctx, nil)
	if err != nil {
		t.Fatalf("GetRelations: %v", err)
	}
	noms := map[string]string{}
	for _, r := range rel {
		noms[r.XUID] = r.Gamertag
	}
	exigerNomDuKillFeed(t, "GetRelations (portée base)", noms)
	exigerUnArgumentParListe(t, b)
	exigerFenetresBornees(t, b, "GetRelations (portée base)", borne, 1)
}

// vic2noms indexe des rivaux par xuid.
func vic2noms(rivaux []domain.CareerRivalRawRow) map[string]string {
	noms := map[string]string{}
	for _, r := range rivaux {
		noms[r.XUID] = r.Gamertag
	}
	return noms
}

// estRequeteDAnnuaire reconnaît les quatre requêtes de l'annuaire (squad_repo_annuaire.go).
func estRequeteDAnnuaire(q string) bool {
	for _, marque := range []string{"FROM xuid_aliases", "SELECT xuid, gamertag FROM (", "WITH cherches(xuid)",
		"SELECT DISTINCT match_id FROM match_participants WHERE xuid"} {
		if strings.Contains(q, marque) {
			return true
		}
	}
	return false
}

// exigerNomDuKillFeed : le xuid que seul le kill-feed nomme porte ce nom, pas le libellé masqué.
func exigerNomDuKillFeed(t *testing.T, lecture string, noms map[string]string) {
	t.Helper()
	if got := noms[xuidNommeParLeKillFeed]; got != xuidNommeParLeKillFeed {
		t.Fatalf("%s : %s nommé %q, want %q (le gamertag de son kill-feed)", lecture,
			xuidNommeParLeKillFeed, got, xuidNommeParLeKillFeed)
	}
}

// exigerUnArgumentParListe : les requêtes de l'annuaire notées depuis le dernier vidage ne portent
// que quelques arguments (au plus cinq listes), jamais un argument par valeur. Les requêtes sont
// remises au carnet pour exigerFenetresBornees.
func exigerUnArgumentParListe(t *testing.T, b baseNotee) {
	t.Helper()
	requetes := b.carnet.vider()
	annuaire := 0
	for _, r := range requetes {
		if estRequeteDAnnuaire(r.sql) {
			annuaire++
			if len(r.args) > 5 {
				t.Errorf("requête d'annuaire à %d arguments (une liste doit être UN paramètre) :\n%.300s",
					len(r.args), r.sql)
			}
		}
		b.carnet.noter(r.sql, r.args)
	}
	if annuaire == 0 {
		t.Fatal("aucune requête d'annuaire notée : le test ne mesure rien")
	}
}
