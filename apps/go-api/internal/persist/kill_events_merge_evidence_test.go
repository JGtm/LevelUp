package persist

import "testing"

func mortsDeCredit(victime string, n int) []KillEventInsert {
	out := make([]KillEventInsert, 0, n)
	for k := 1; k <= n; k++ {
		m := mortCredit(1000 * k)
		m.VictimXUID = victime
		out = append(out, m)
	}
	return out
}

func orphelinDeFilm(victime string, t int) KillEventInsert {
	m := mortFilm(t)
	m.VictimXUID = victime
	m.FeedKillerXUID = "xuid(77)"
	return m
}

// Le controle des morts officielles COMPTE et ne retire RIEN : une ligne n est jamais ecartee
// pour ramener un joueur a son compte (on pourrait garder la fausse et perdre la juste). Ici dix
// morts de credit pour sept evenements `death` et sept morts officielles : trois morts de credit
// sans evenement, conservees (le credit fait foi), et le joueur est signale.
// Mutation qui doit rougir : reintroduire une troncature (retirer les morts en trop, les plus
// tardives ou toute autre) -> len(out.Deaths) tombe sous 11.
func TestControleDesMortsOfficiellesNeRetireRien(t *testing.T) {
	const v = "xuid(500)"
	base := batchCredit(mortsDeCredit(v, 10)...)
	base.Evidence = MergeEvidence{
		Known:          true,
		APIKills:       map[int]int{},
		OfficialDeaths: map[string]int{v: 7},
		APIDeaths:      map[string][]int{v: {1000, 2000, 3000, 4000, 5000, 6000, 7000}},
	}
	bot := mortFilm(90000) // une mort de bot : sans xuid de victime, servie sans adossement
	out, st, err := MergeCreditAndFilm(base, batchFilm(bot))
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Deaths) != 11 || st.Orphans != 1 || st.CreditSansMortAPI != 3 {
		t.Fatalf("%d morts / %d orphelins / %d credits sans evenement, attendu 11 / 1 / 3 : le "+
			"controle a retire des lignes", len(out.Deaths), st.Orphans, st.CreditSansMortAPI)
	}
	if len(st.OverOfficial) != 1 || st.OverOfficial[0] != (OverOfficialPlayer{XUID: v, Official: 7, Served: 10}) {
		t.Errorf("exces = %+v, attendu un joueur officiel 7 / servi 10", st.OverOfficial)
	}
}

// Un joueur parti puis revenu en cours de match : quatre morts officielles, mais six evenements
// `death` et six morts servies. L ecart est legitime (l officiel ne couvre pas tout le match) : le
// plafond est le plus grand des deux comptes de l API, rien n est signale.
// Mutation qui doit rougir : comparer les morts servies aux seules morts officielles.
func TestControleDesMortsOfficiellesPartiPuisRevenu(t *testing.T) {
	const v = "xuid(501)"
	base := batchCredit(mortsDeCredit(v, 6)...)
	base.Evidence = MergeEvidence{Known: true, APIKills: map[int]int{},
		OfficialDeaths: map[string]int{v: 4},
		APIDeaths:      map[string][]int{v: {1000, 2000, 3000, 4000, 5000, 6000}}}
	_, st, err := MergeCreditAndFilm(base, batchFilm(mortFilm(90000)))
	if err != nil || len(st.OverOfficial) != 0 || st.CreditSansMortAPI != 0 {
		t.Errorf("exces = %+v, credits sans evenement = %d, err=%v ; attendu aucun", st.OverOfficial,
			st.CreditSansMortAPI, err)
	}
}

// Sans morts officielles connues, ou sans exces : rien n est signale.
func TestControleDesMortsOfficiellesSilencieuxSansExces(t *testing.T) {
	const v = "xuid(500)"
	for nom, ev := range map[string]MergeEvidence{
		"inconnu":       {},
		"non renseigne": {Known: true, OfficialDeaths: map[string]int{}},
		"exact":         {Known: true, OfficialDeaths: map[string]int{v: 3}},
	} {
		base := batchCredit(mortsDeCredit(v, 3)...)
		base.Evidence = ev
		_, st, err := MergeCreditAndFilm(base, batchFilm(orphelinDeFilm("xuid(600)", 90000)))
		if err != nil || len(st.OverOfficial) != 0 {
			t.Errorf("%s : exces=%+v err=%v, attendu aucun", nom, st.OverOfficial, err)
		}
	}
}
