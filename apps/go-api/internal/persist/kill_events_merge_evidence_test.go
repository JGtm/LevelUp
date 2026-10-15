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
// pour ramener un joueur a son compte (on pourrait garder la fausse et perdre la juste).
// Mutation qui doit rougir : reintroduire une troncature (retirer les lignes de film en trop,
// les plus tardives ou toute autre) -> len(out.Deaths) tombe sous 10.
func TestControleDesMortsOfficiellesNeRetireRien(t *testing.T) {
	const v = "xuid(500)"
	base := batchCredit(mortsDeCredit(v, 7)...)
	base.Evidence = MergeEvidence{
		Known:          true,
		APIKills:       map[int]int{},
		OfficialDeaths: map[string]int{v: 7},
	}
	// Trois orphelines d instant libre (aucun credit a ces instants) : conservees par la fusion.
	film := batchFilm(orphelinDeFilm(v, 50000), orphelinDeFilm(v, 60000), orphelinDeFilm(v, 70000))
	out, st, err := MergeCreditAndFilm(base, film)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Deaths) != 10 || st.Orphans != 3 {
		t.Fatalf("%d morts / %d orphelins, attendu 10 / 3 : le controle a retire des lignes", len(out.Deaths), st.Orphans)
	}
	if len(st.OverOfficial) != 1 || st.OverOfficial[0] != (OverOfficialPlayer{XUID: v, Official: 7, Served: 10}) {
		t.Errorf("exces = %+v, attendu un joueur officiel 7 / servi 10", st.OverOfficial)
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
