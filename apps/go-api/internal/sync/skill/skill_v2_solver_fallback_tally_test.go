package skill

import (
	"context"
	"strings"
	"testing"

	skillv2 "levelup/go-api/internal/analysis/skill_v2"
)

// Dans un rejeu (tally sur le ctx), N matchs en repli n'écrivent AUCUN WARN par match : le détail
// passe en DEBUG, le compteur expvar compte chaque repli, et la fin du rejeu écrit UNE synthèse
// (matchs rejoués, matchs en repli, au plus repliExemplesMax identifiants).
func TestSolveWithCountFallback_UneSyntheseParRejeu(t *testing.T) {
	logs, restore := captureSlog(t)
	defer restore()
	before := solverCountFallbackTotal.Value()
	ctx, tally := withRepliTally(context.Background())

	const matchs = 36
	for i := 0; i < matchs; i++ {
		in := matchDivergent()
		in.matchID = "m_" + string(rune('a'+i%26)) + string(rune('a'+i/26))
		if _, _, err := solveWithCountFallback(ctx, skillv2.UpdateTwoTeamWithCountsEP, in); err != nil {
			t.Fatalf("repli %d : %v", i, err)
		}
	}
	if delta := solverCountFallbackTotal.Value() - before; delta != matchs {
		t.Errorf("compteur du repli +%d, attendu +%d", delta, matchs)
	}
	if n := strings.Count(logs.String(), "level=WARN"); n != 0 {
		t.Errorf("%d WARN avant la synthèse, attendu 0 :\n%s", n, logs.String())
	}
	if n := strings.Count(logs.String(), "level=DEBUG"); n != matchs {
		t.Errorf("%d lignes DEBUG de détail, attendu %d", n, matchs)
	}

	tally.logSummary(ctx, "xuid(1)", 120)

	if n := strings.Count(logs.String(), "level=WARN"); n != 1 {
		t.Fatalf("%d WARN après la synthèse, attendu exactement 1 :\n%s", n, logs.String())
	}
	var synthese string
	for _, ligne := range strings.Split(logs.String(), "\n") {
		if strings.Contains(ligne, "level=WARN") {
			synthese = ligne
		}
	}
	for _, want := range []string{"matchs_rejoues=120", "matchs_en_repli=36", "m_aa", "m_ba", "m_ca"} {
		if !strings.Contains(synthese, want) {
			t.Errorf("synthèse sans %q :\n%s", want, synthese)
		}
	}
	if strings.Contains(synthese, "m_da") {
		t.Errorf("plus de %d exemples dans la synthèse :\n%s", repliExemplesMax, synthese)
	}
}

// Sans repli, le rejeu n'écrit aucune synthèse.
func TestRepliTally_SansRepliSilencieux(t *testing.T) {
	logs, restore := captureSlog(t)
	defer restore()
	ctx, tally := withRepliTally(context.Background())
	tally.logSummary(ctx, "xuid(1)", 50)
	if logs.Len() != 0 {
		t.Errorf("synthèse écrite sans repli :\n%s", logs.String())
	}
}
