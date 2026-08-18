package replay

// objectifs_phase1_ctf_test.go — LE DISCRIMINANT DE MODE, MESURE SUR FILMS DE MODE CONNU.
//
// LE PROBLEME QUE LA PHASE 1 A DU TRANCHER, ET QUE LE PLAN NE POSAIT PAS. L'artefact de rejeu
// est construit HORS LIGNE, a partir des SEULS chunks du film. Le film ne nomme ni la carte ni
// le mode (`document.go`, en-tete) : `objectiveevents.ObjectiveTypeOf` prend un
// `game_variant_name` qui vient de la base, et la base n'est pas ouverte ici. Or la table
// d'emplacements de statistiques du DRAPEAU, appliquee au film d'un AUTRE mode, rendrait des
// « prises » qui n'en sont pas — un calque de drapeau sur une partie de Bastion.
//
// PREMIER DISCRIMINANT ESSAYE, ET REFUTE PAR CETTE MESURE : le BURST DE CAPTURE seul
// (`objectiveevents.CaptureBurstTimes`), l evenement de score a 6 tiers distincts qui accompagne
// une capture de drapeau. Il etait deja mesure « 0 manque / 0 faux positif » sur les matchs de
// verite terrain de son propre chantier — mais SUR DES FILMS CTF. L autre moitie, mesuree ici,
// le refute : QUATRE films non-CTF en portent (Oddball 2, une colline 4, un Slayer 2).
//
// LE DISCRIMINANT RETENU EST L ACCORD DE TROIS SIGNAUX, tous du film
// (`objectiveevents.FlagFilmSignals`) : bursts > 0, captures > 0, captures <= bursts, vols > 0.
// La regle et son corpus gele vivent dans `objectiveevents/flagfilm.go` ; cet instrument est ce
// qui l a mesuree, et il la REJOUE sur les films.
//
// SEUIL ECRIT AVANT LA MESURE : ZERO faux positif ET ZERO film CTF ecarte, sur tout le corpus.
//
// GARDE : `OBJ_FILM` (racine du cache film), comme toute la phase 0. Lecture seule, aucune base.

import (
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// objModeCorpus — les films de mode CONNU. Le mode vient de `game_variant_name`, releve dans
// les tests de verite terrain d'`objectiveevents` (extract_test.go, named_test.go) et dans le
// corpus du plan (`.ai/V7.5/replay2d/PLAN_OBJECTIFS_VIVANTS_2E_LECTURE.md`, ligne « Corpus »).
var objModeCorpus = []struct {
	ID   string
	Mode string // "" = mode sans table d'objectif (Slayer et apparentes)
}{
	{"64e8adfa", objectiveevents.ObjectiveTypeFlag},
	{"530820e5", objectiveevents.ObjectiveTypeFlag},
	{"53ce4390", objectiveevents.ObjectiveTypeFlag},
	{"0f9550e5", objectiveevents.ObjectiveTypeFlag},
	{"1bc77d2e", objectiveevents.ObjectiveTypeFlag},
	{"24dbb67d", objectiveevents.ObjectiveTypeSkull},
	{"696a9d7c", objectiveevents.ObjectiveTypeZone},
	{"7344d24f", objectiveevents.ObjectiveTypeZone},
	{"0a247154", objectiveevents.ObjectiveTypeHill},
	{"01e1f945", objectiveevents.ObjectiveTypeHill},
	{"606d9844", objectiveevents.ObjectiveTypeHill},
	{"8076f97f", objectiveevents.ObjectiveTypeHill},
	{"bcb6d393", objectiveevents.ObjectiveTypeFlag},
	{"000d5950", ""},
	{"00162144", ""},
}

// TestObjectifsPhase1DiscriminantCTF — le discriminant de production distingue-t-il le CTF du
// reste, sur quinze films de mode connu ?
func TestObjectifsPhase1DiscriminantCTF(t *testing.T) {
	root := objRequireRoot(t)
	joues, fauxPositifs, ctfSansBurst := 0, 0, 0
	for _, f := range objModeCorpus {
		src, ok := objOpenFilm(t, root, f.ID)
		if !ok {
			continue
		}
		joues++
		evs := objectiveevents.NamedEvents(src, objectiveevents.ObjectiveTypeFlag)
		sig := objectiveevents.FlagFilmSignalsFrom(objectiveevents.CaptureBurstTimes(src), evs)
		compte := objCompteStats(evs)
		verdict, attendu := sig.IsFlagFilm(), f.Mode == objectiveevents.ObjectiveTypeFlag
		t.Logf("%s (mode %q) : bursts %d ; table DRAPEAU appliquee -> grabs %d, steals %d, "+
			"captures %d, returns %d, porteurs tues %d ; VERDICT CTF %v (attendu %v)",
			f.ID, f.Mode, sig.Bursts, compte[objectiveevents.StatFlagGrabs],
			compte[objectiveevents.StatFlagSteals], compte[objectiveevents.StatFlagCaptures],
			compte[objectiveevents.StatFlagReturns], compte[objectiveevents.StatFlagCarriersKilled],
			verdict, attendu)
		switch {
		case attendu && !verdict:
			ctfSansBurst++
			t.Errorf("%s : film CTF ECARTE par le discriminant — signaux %+v", f.ID, sig)
		case !attendu && verdict:
			fauxPositifs++
			t.Errorf("%s : mode %q RETENU comme CTF — signaux %+v", f.ID, f.Mode, sig)
		}
	}
	if joues == 0 {
		t.Skipf("aucun film du corpus de mode dans le cache (%s=%q)", objFilmEnv, root)
	}
	t.Logf("DISCRIMINANT CTF : %d films joues, %d faux positifs (seuil 0), %d films CTF ecartes "+
		"(seuil 0)", joues, fauxPositifs, ctfSansBurst)
}

// objCompteStats compte les evenements nommes par statistique.
func objCompteStats(evs []objectiveevents.NamedEvent) map[string]int {
	out := map[string]int{}
	for _, e := range evs {
		out[e.Stat]++
	}
	return out
}
