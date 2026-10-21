package duckdb

// resumes_parite_s1_test.go — PARITÉ 9.2 (transitoire) : les lecteurs de frags par paire basculés
// sur S1 rendent exactement ce que rendait la lecture du journal.
//
// Méthode : le lecteur RÉEL tourne sur une base notée (les requêtes qu'il envoie sont gardées) ;
// chaque requête envoyée est réécrite en son ancienne forme (lecture du journal, un frag = une
// ligne) par un remplacement textuel inverse de la bascule, puis rejouée avec les MÊMES arguments :
// les deux jeux de lignes doivent être égaux.

import (
	"context"
	"regexp"
	"strings"
	"testing"
)

var (
	reFragBrut     = regexp.MustCompile(`kv\.kills\b`)
	reSommeFiltree = regexp.MustCompile(`CAST\(COALESCE\(SUM\((?:kv\.)?kills\) FILTER \((WHERE [^()]*)\), 0\) AS BIGINT\)`)
	reAncienS1     = strings.NewReplacer(
		"THEN kv.kills ELSE 0 END", "THEN 1 ELSE 0 END",
		"victim_xuid, kills\n        FROM match_kill_pairs_latest", "victim_xuid\n        FROM match_kill_events_latest",
		"match_kill_pairs_latest", "match_kill_events_latest",
	)
)

// enAncienneFormeS1 réécrit une requête de frags par paire en sa forme d'avant la bascule.
func enAncienneFormeS1(sqlText string) string {
	return reAncienS1.Replace(reSommeFiltree.ReplaceAllString(sqlText, "COUNT(*) FILTER ($1)"))
}

// pariteDesRequetesS1 rejoue, pour chaque requête S1 notée pendant `lire`, la requête réécrite en
// ancienne forme et compare les lignes. Rend le nombre de requêtes comparées.
func pariteDesRequetesS1(t *testing.T, b baseNotee, nom string, lire func()) {
	t.Helper()
	b.carnet.vider()
	lire()
	compare := 0
	for _, r := range b.carnet.vider() {
		if !strings.Contains(r.sql, "match_kill_pairs_latest") {
			continue
		}
		args := make([]any, len(r.args))
		for i, a := range r.args {
			args[i] = a.Value
		}
		ancien := enAncienneFormeS1(r.sql)
		if strings.Contains(ancien, "match_kill_pairs_latest") || reFragBrut.MatchString(ancien) {
			t.Fatalf("%s : la réécriture inverse a laissé du S1 :\n%s", nom, ancien)
		}
		exigerMemesLignes(t, nom, lignesDe(t, b.brute, ancien, args...), lignesDe(t, b.brute, r.sql, args...))
		compare++
	}
	if compare == 0 {
		t.Fatalf("%s : aucune requête S1 notée, la parité ne mesure rien", nom)
	}
}

func TestParite92_LecteursDePaires(t *testing.T) {
	ctx := context.Background()
	b := newBaseNotee(t)
	corpusPariteSur(t, b.brute)
	for _, slug := range []string{"halo_infinite", "halo_5"} {
		b.pdb.XUID, b.pdb.TitleSlug = pJ1, slug
		career := NewCareerRepo(b.pdb)
		pariteDesRequetesS1(t, b, "Q26 top rencontres "+slug, func() {
			if _, _, err := career.GetTopEncountersGlobal(ctx, nil); err != nil {
				t.Fatal(err)
			}
		})
		pariteDesRequetesS1(t, b, "Q27 rivaux "+slug, func() {
			if _, _, err := career.GetRivals(ctx); err != nil {
				t.Fatal(err)
			}
		})
		for _, scope := range [][]string{nil, {"pA", "pB", "pC"}} {
			pariteDesRequetesS1(t, b, "Q28 relations "+slug, func() {
				if _, err := career.GetRelations(ctx, scope); err != nil {
					t.Fatal(err)
				}
			})
		}
		pariteDesRequetesS1(t, b, "Q30 duels "+slug, func() {
			if _, err := career.GetRivalTimeline(ctx, pJ2, nil, 20); err != nil {
				t.Fatal(err)
			}
		})
		pariteDesRequetesS1(t, b, "Q23b rencontres du match "+slug, func() {
			if _, err := NewMatchViewRepo(b.pdb, pJ1).GetMatchEncounterStats(ctx, "pA", pJ1); err != nil {
				t.Fatal(err)
			}
		})
		pariteDesRequetesS1(t, b, "Comparaison "+slug, func() {
			if _, err := NewCompareRepo(b.pdb).GetEncounterStats(ctx, pJ1, pJ2); err != nil {
				t.Fatal(err)
			}
		})
		pariteDesRequetesS1(t, b, "Explorateur "+slug, func() {
			if _, err := NewExplorerRepo(b.pdb, pJ1).GetKillerVictimBetween(ctx, pJ1, pJ2); err != nil {
				t.Fatal(err)
			}
		})
	}
}
