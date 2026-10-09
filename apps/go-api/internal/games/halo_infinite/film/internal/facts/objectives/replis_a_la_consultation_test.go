package objectives

// replis_a_la_consultation_test.go — LES DEUX REPLIS A LA CONSULTATION SE COMPTENT PAR EVENEMENT
// DISTINCT (lot J8.7-bis, 2026-09-28).
//
// La regle tenue ici : le compte ne depend ni du NOMBRE ni de l ORDRE des lectures. Deux lectures du
// meme evenement — depuis deux copies du resolveur, ou par deux marches des series — comptent UNE
// fois ; deux evenements distincts comptent deux fois.
//
// MUTATIONS JOUEES (2026-09-28) : noter chaque consultation dans une liste au lieu d un ensemble fait
// rougir les deux tests (« 2, attendu 1 ») ; retirer la note de [RoundIdentity.roundOfTime], ou celle
// de [rawSeriesByKey], les fait rougir aussi (« 0, attendu 1 »).

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// avecEmissionsNegatives rend les enregistrements de la fixture a deux manches, plus une emission
// NEGATIVE du score de mode du slot 22 a chacun des instants donnes (manche 0).
func avecEmissionsNegatives(instants ...int) []types.StatRecord {
	recs, _ := twoRoundReassignedFixture()
	for _, t := range instants {
		recs = append(recs, modeRec(t, 22, 0, -115))
	}
	return recs
}

// TestUneEmissionJeteeSeCompteUneFoisQuelQueSoitLeNombreDeLectures : la meme emission negative, lue
// par la marche par emplacement (deux fois, par deux entrees publiques) et par la marche par table,
// compte UNE fois ; une seconde emission negative, a un autre instant, en fait deux.
func TestUneEmissionJeteeSeCompteUneFoisQuelQueSoitLeNombreDeLectures(t *testing.T) {
	cons := &ReplisALaConsultation{}
	recs := avecEmissionsNegatives(1500)
	SeriesByRound(recs, ModeScoreComponent(), false, cons)
	SeriesTotal(recs, ModeScoreComponent(), false, cons)
	rawSeriesByKey(recs, map[statSlotKey]statSlot{ModeScoreComponent().key(): {Stat: "sonde"}}, cons)
	if got := cons.ComptesDesReplis().EmissionsHorsDomaineJetees; got != 1 {
		t.Fatalf("une emission jetee lue trois fois : %d, attendu 1", got)
	}

	deux := &ReplisALaConsultation{}
	recs = avecEmissionsNegatives(1500, 2500)
	SeriesTotal(recs, ModeScoreComponent(), false, deux)
	SeriesByRound(recs, ModeScoreComponent(), false, deux)
	if got := deux.ComptesDesReplis().EmissionsHorsDomaineJetees; got != 2 {
		t.Fatalf("deux emissions jetees distinctes : %d, attendu 2", got)
	}
}

// TestLaMarcheParTableEstUnSiteDuRepli : [rawSeriesByKey] (les actions nommees) note l emission
// qu il jette, sans aucune autre lecture.
func TestLaMarcheParTableEstUnSiteDuRepli(t *testing.T) {
	cons := &ReplisALaConsultation{}
	rawSeriesByKey(avecEmissionsNegatives(1500), map[statSlotKey]statSlot{ModeScoreComponent().key(): {Stat: "sonde"}}, cons)
	if got := cons.ComptesDesReplis().EmissionsHorsDomaineJetees; got != 1 {
		t.Fatalf("emission jetee par la marche par table : %d, attendu 1", got)
	}
}

// TestUnInstantAvantLesManchesNAppartientAAucune : un instant anterieur a toute manche connue n est
// range dans aucune — `RoundAt` le dit, `At` ne nomme personne, sur le resolveur comme sur une
// completion. Un instant DANS une manche y reste range.
func TestUnInstantAvantLesManchesNAppartientAAucune(t *testing.T) {
	recs, deaths := twoRoundReassignedFixture()
	ri := ResolveRoundIdentity(recs, deaths, nil)
	completee := ri.CompletedByElimination(recs, nil)
	if round, ok := ri.RoundAt(12000); !ok || round != 1 {
		t.Fatalf("RoundAt(12000) = %d, %v — attendu 1, vrai", round, ok)
	}
	for _, r := range []RoundIdentity{ri, completee} {
		if _, ok := r.RoundAt(100); ok {
			t.Errorf("RoundAt(100) range un instant anterieur a toute manche")
		}
		if got := r.At(22, 100); got != "" {
			t.Errorf("At(22, 100) = %q, attendu vide : l instant n appartient a aucune manche", got)
		}
	}
}

// TestSansEnregistreurRienNEstNote : nil (outils hors production) consulte sans compter ni paniquer.
func TestSansEnregistreurRienNEstNote(t *testing.T) {
	var cons *ReplisALaConsultation
	SeriesTotal(avecEmissionsNegatives(1500), ModeScoreComponent(), false, cons)
	recs, deaths := twoRoundReassignedFixture()
	ResolveRoundIdentity(recs, deaths, nil).At(22, 100)
	if got := cons.ComptesDesReplis(); got != (ComptesDesReplis{}) {
		t.Fatalf("enregistreur nil : %+v, attendu zero", got)
	}
}
