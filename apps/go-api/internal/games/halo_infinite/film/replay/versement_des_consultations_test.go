package replay

// versement_des_consultations_test.go — L ENREGISTREUR DU DOCUMENT ARRIVE AU COMPTEUR DE LA CUISSON
// (lot J8.7-bis, 2026-09-28) : les deux replis a la consultation sont verses par la table, a la
// cloture, sous leurs noms — une fois par evenement distinct, quel que soit le nombre de calques qui
// l ont lu.
//
// MUTATION JOUEE (2026-09-28) : retirer `.Plus(r.Consultations.ComptesDesReplis())` de
// [versementDeLAssemblage] fait rougir ce test (« 0, attendu 1 »).

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/objectives"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

func TestLesReplisALaConsultationSontVersesAlaCloture(t *testing.T) {
	recs, deaths := skullFixture()
	// Une emission NEGATIVE du score de mode du slot 22, manche 0 : jetee par le filtre de domaine.
	recs = append(recs, types.StatRecord{TimeMS: 2500, Slot: 22, Round: 0,
		Comps: map[int]types.StatValue{0: {A: -115}}})
	cons := &objectives.ReplisALaConsultation{}
	// TROIS LECTURES DE LA MEME EMISSION, par deux calques (score, crane) : un evenement.
	loadScoreSeries(recs, objectives.ModeScoreComponent(), false, cons)
	loadScoreSeries(recs, objectives.ModeScoreComponent(), false, cons)
	skullCarryIntervals(recs, objectives.ResolveRoundIdentity(recs, deaths, cons), cons)
	// UN INSTANT anterieur a toute manche, lu par le resolveur puis par une copie : un evenement.
	ri := objectives.ResolveRoundIdentity(recs, deaths, cons)
	ri.At(22, 10)
	copie := ri
	copie.RoundAt(10)

	fb := fallback.NouveauCompteur()
	versementDeLAssemblage(fb, ReplisHorsBalayage{Consultations: cons})
	got := replisDe(fb)
	if got[fallback.NomEmissionHorsDomaineJetee] != 1 || got[fallback.NomInstantSurLaPremiereManche] != 1 {
		t.Fatalf("verses : %v, attendu 1 emission jetee et 1 instant sur la premiere manche", got)
	}
}
