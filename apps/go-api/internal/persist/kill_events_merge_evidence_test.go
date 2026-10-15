package persist

import "testing"

// credit de la victime `v` : n morts de credit, aux instants 1000*k.
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

// KA4 — le cas DANIELBOIMEXICO : 7 morts officielles, 7 de credit, trois orphelines de film dont
// une sur un instant partage. Le plafond ramene la victime a 7.
func TestFusionPlafonneLesMortsAuxMortsOfficielles(t *testing.T) {
	const v = "xuid(500)"
	autre := mortCredit(500) // une mort d un AUTRE joueur a l instant 500
	base := batchCredit(append(mortsDeCredit(v, 7), autre)...)
	base.Evidence = MergeEvidence{
		Known:          true,
		APIKills:       map[int]int{500: 2}, // deux kills : l orpheline de l instant 500 est prouvee
		OfficialDeaths: map[string]int{v: 7},
	}
	film := batchFilm(
		orphelinDeFilm(v, 500),   // instant partage, prouve par l API
		orphelinDeFilm(v, 50000), // instant libre
		orphelinDeFilm(v, 60000), // instant libre
	)
	out, st, err := MergeCreditAndFilm(base, film)
	if err != nil {
		t.Fatal(err)
	}
	servies := 0
	for _, d := range out.Deaths {
		if d.VictimXUID == v {
			servies++
		}
	}
	if servies != 7 {
		t.Fatalf("%d morts servies, attendu 7 (morts officielles)", servies)
	}
	if st.CappedOverOfficial != 3 || st.Orphans != 0 {
		t.Errorf("retirees=%d orphelins=%d, attendu 3 / 0", st.CappedOverOfficial, st.Orphans)
	}
	if out.CreditBaseCount != 8 || len(out.Deaths) != 8 {
		t.Errorf("plancher de credit = %d, %d morts, attendu 8 / 8 : le plafond ne retranche jamais le credit",
			out.CreditBaseCount, len(out.Deaths))
	}
}

// Le plafond retire d abord les orphelines d instant partage, puis les plus tardives.
func TestPlafondRetireLesPartageesPuisLesPlusTardives(t *testing.T) {
	const v = "xuid(500)"
	base := batchCredit(append(mortsDeCredit(v, 2), mortCredit(500))...)
	base.Evidence = MergeEvidence{Known: true, APIKills: map[int]int{500: 2}, OfficialDeaths: map[string]int{v: 4}}
	film := batchFilm(
		orphelinDeFilm(v, 500), // partagee
		orphelinDeFilm(v, 9000),
		orphelinDeFilm(v, 8000),
	)
	out, st, err := MergeCreditAndFilm(base, film)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Deaths) != 5 || st.CappedOverOfficial != 1 {
		t.Fatalf("%d morts / %d retirees, attendu 5 / 1", len(out.Deaths), st.CappedOverOfficial)
	}
	for _, d := range out.Deaths {
		if d.TimeMS == 500 && d.VictimXUID == v {
			t.Error("l orpheline d instant partage aurait du etre retiree en premier")
		}
	}
}

// Sans morts officielles connues, aucun plafond ; surplus de credit seul : signale, jamais retranche.
func TestPlafondInconnuOuCreditSeul(t *testing.T) {
	const v = "xuid(500)"
	base := batchCredit(mortsDeCredit(v, 3)...)
	base.Evidence = MergeEvidence{Known: true, APIKills: map[int]int{}, OfficialDeaths: map[string]int{}}
	out, st, err := MergeCreditAndFilm(base, batchFilm(orphelinDeFilm(v, 70000)))
	if err != nil || len(out.Deaths) != 4 || st.CappedOverOfficial != 0 {
		t.Fatalf("sans mort officielle connue : %d morts / %d retirees / err=%v", len(out.Deaths), st.CappedOverOfficial, err)
	}

	base.Evidence.OfficialDeaths = map[string]int{v: 2}
	out, st, err = MergeCreditAndFilm(base, batchFilm(orphelinDeFilm(v, 70000)))
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Deaths) != 3 || st.CappedOverOfficial != 1 || st.OverOfficialUncapped != 1 {
		t.Errorf("%d morts / %d retirees / %d residuelles, attendu 3 / 1 / 1 : le credit (3) ne se retranche pas",
			len(out.Deaths), st.CappedOverOfficial, st.OverOfficialUncapped)
	}
}
