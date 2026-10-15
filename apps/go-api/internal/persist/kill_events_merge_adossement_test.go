package persist

// kill_events_merge_adossement_test.go — TOUTE MORT D HUMAIN SERVIE S ADOSSE A UN EVENEMENT DEATH DE
// L API, UN EVENEMENT PAR LIGNE (kill_events_merge_adossement.go).

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

const victimeHumaine = "xuid(700)"

// orphelineHumaine : une ligne de film a victime humaine, tuee par `tueurXUID` (vide pour un bot).
func orphelineHumaine(t int, tueurXUID string) KillEventInsert {
	m := mortFilm(t)
	m.VictimXUID = victimeHumaine
	m.FeedKillerXUID = tueurXUID
	return m
}

// preuvesDesMorts : les preuves lues, avec les evenements `death` de la victime humaine.
func preuvesDesMorts(instants ...int) MergeEvidence {
	return MergeEvidence{Known: true,
		APIDeaths: map[string][]int{victimeHumaine: instants}}
}

// Le temoin du parc `0762ae7c` : la mort de M est au credit a 227 994 (l instant du kill) et le film
// la sert aussi a 227 995 au nom d un bot (« tueur-bot »). Un seul evenement `death` de M : la
// mort de credit le prend, la ligne de film est refusee et comptee.
//
// Mutation qui doit rougir : servir l orpheline sans consulter l adossement
// ([adossement.orphelineServie] qui rend toujours vrai) -> deux morts de M.
func TestAdossementRefuseLaMortServieDeuxFois(t *testing.T) {
	credit := mortCredit(227_994)
	credit.VictimXUID = victimeHumaine
	base := batchCredit(credit)
	base.Evidence = preuvesDesMorts(227_995)
	out, st, err := MergeCreditAndFilm(base, batchFilm(orphelineHumaine(227_995, "")))
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Deaths) != 1 || st.OrphansSansMortAPI != 1 || st.Orphans != 0 {
		t.Fatalf("%d morts / %d refusees / %d orphelins, attendu 1 / 1 / 0", len(out.Deaths),
			st.OrphansSansMortAPI, st.Orphans)
	}
}

// Une mort infligee par un bot : le kill-feed de l API porte la mort (evenement libre) et pas le
// kill, le credit n en a donc pas de ligne. L orpheline de film s adosse a l evenement et se sert.
func TestAdossementServeLaMortInfligeeParUnBot(t *testing.T) {
	base := batchCredit(mortCredit(1000))
	base.Evidence = preuvesDesMorts(50_001)
	out, st, err := MergeCreditAndFilm(base, batchFilm(orphelineHumaine(50_001, "")))
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Deaths) != 2 || st.Orphans != 1 || st.OrphansSansMortAPI != 0 {
		t.Fatalf("%d morts / %d orphelins / %d refusees, attendu 2 / 1 / 0", len(out.Deaths), st.Orphans,
			st.OrphansSansMortAPI)
	}
}

// Deux lignes de film pour UN evenement `death` : une seule est servie.
//
// Mutation qui doit rougir : ne pas marquer l evenement pris dans [adossement.prendre].
func TestAdossementUnEvenementUneLigne(t *testing.T) {
	base := batchCredit(mortCredit(1000))
	base.Evidence = preuvesDesMorts(3001)
	out, st, err := MergeCreditAndFilm(base,
		batchFilm(orphelineHumaine(3000, ""), orphelineHumaine(3002, "xuid(9)")))
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Deaths) != 2 || st.Orphans != 1 || st.OrphansSansMortAPI != 1 {
		t.Fatalf("%d morts / %d orphelins / %d refusees, attendu 2 / 1 / 1", len(out.Deaths), st.Orphans,
			st.OrphansSansMortAPI)
	}
	if out.Deaths[1].TimeMS != 3000 {
		t.Errorf("ligne servie a %d, attendu 3000 (la premiere du film)", out.Deaths[1].TimeMS)
	}
}

// Hors de la fenetre, un evenement n adosse pas ; sans preuve lue, rien n est filtre.
func TestAdossementFenetreEtPreuveInconnue(t *testing.T) {
	base := batchCredit(mortCredit(1000))
	base.Evidence = preuvesDesMorts(3006)
	_, st, err := MergeCreditAndFilm(base, batchFilm(orphelineHumaine(3000, "")))
	if err != nil || st.OrphansSansMortAPI != 1 {
		t.Errorf("a 6 ms : %d refusees (err=%v), attendu 1", st.OrphansSansMortAPI, err)
	}
	base.Evidence = MergeEvidence{}
	out, st, err := MergeCreditAndFilm(base, batchFilm(orphelineHumaine(3000, "")))
	if err != nil || len(out.Deaths) != 2 || st.OrphansSansMortAPI != 0 {
		t.Errorf("sans preuve : %d morts, %d refusees (err=%v), attendu 2 / 0", len(out.Deaths),
			st.OrphansSansMortAPI, err)
	}
}

// Les instants ambigus se journalisent en WARN avec le `match_id` (le compteur seul ne dit pas ou).
//
// Mutation qui doit rougir : retirer le WARN des instants ambigus de [PublishMergeStats].
func TestPublishMergeStatsJournaliseLesInstantsAmbigus(t *testing.T) {
	var buf bytes.Buffer
	avant := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})))
	t.Cleanup(func() { slog.SetDefault(avant) })
	PublishMergeStats(context.Background(), "m-ambigu", MergeStats{AmbiguousInstants: 2})
	if s := buf.String(); !strings.Contains(s, "match_id=m-ambigu") || !strings.Contains(s, "instants_ambigus=2") {
		t.Errorf("journal = %q, attendu un WARN portant match_id et instants_ambigus", s)
	}
}
