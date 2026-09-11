package replay

// flag_carries_handoff_test.go — LE PASSAGE DE MAIN EN MAIN (lot 6.11, item 1).
//
// CE QUE CES TESTS FERMENT. `boundFlagCarries` ne connaissait que la prise SUIVANTE DU MEME SLOT
// (`nextOpeningOfSlot`) : un drapeau qui passe a un COEQUIPIER laissait le portage precedent
// courir jusqu'au fait suivant (mort, capture, fin de match), c'est-a-dire bien apres que la
// main l'eut lache. Le calque publiait alors DEUX porteurs du meme drapeau au meme instant —
// l'incoherence que `overlaps` ne faisait que compter (cf. `flag_invariant_test.go`).
//
// LE DRAPEAU EST NOMME PAR L'EQUIPE, ET C'EST CE QUI REND LA REGLE SURE : en CTF on ne porte
// jamais son propre drapeau (invariant de `flag_assign.go`), donc deux coequipiers portent le
// MEME et deux adversaires portent des drapeaux DIFFERENTS. Aucune geometrie n'intervient — ce
// qui est indispensable, le bornage precedant l'attribution.
//
// LES TESTS PORTENT SUR [closeByHandoff] DIRECTEMENT, et c'est deliberé : dans le document
// PUBLIE, un portage borne par la prise suivante est de toute facon coupe a la frame de cette
// prise (`spansOfTransitions`), si bien qu'un recouvrement et une fermeture se ressemblent a
// l'oeil. Ce qui change est la BORNE du portage — et elle se lit ici. L'effet de bout en bout,
// lui, est fige par `TestFlagOverlapsComptesParDrapeau`.

import (
	"testing"
)

// flagHandoffScan : deux socles d'equipe, et la table des equipes de l'appelant.
func flagHandoffScan(teams map[string]int) FlagCarryScan {
	return FlagCarryScan{Scanned: true, Signals: flagTestSignals(),
		Spawns: flagInvariantSpawns(), TeamOf: teams}
}

// flagHandoffCas monte le cas nominal : « 1 » prend a 1 000 ms et son portage court jusqu'a sa
// mort a 6 000 ms ; « 2 » prend a 3 000 ms.
func flagHandoffCas() ([]flagCarryRaw, []flagOpening) {
	raws := []flagCarryRaw{
		{xuid: "1", t0: 1000, t1: 6000, steal: true, closed: true, flagIndex: -1},
		{xuid: "2", t0: 3000, t1: 9000, closed: true, flagIndex: -1},
	}
	ops := []flagOpening{
		{slot: 12, xuid: "1", t0: 1000, steal: true},
		{slot: 14, xuid: "2", t0: 3000},
	}
	return raws, ops
}

// TestUnePriseDUnCoequipierFermeLePortage — LE POINT DE L'ITEM. « 1 » et « 2 » sont de l'equipe 0 :
// ils ne peuvent porter que le drapeau de l'equipe 1, donc le MEME. La prise de « 2 » a 3 000 ms
// borne le portage de « 1 », qui courait jusqu'a sa mort a 6 000 ms.
//
// MUTATION : retirer l'appel a [closeByHandoff] dans `buildFlagCarries` rougit
// `TestFlagOverlapsComptesParDrapeau` ; vider le corps de [flagFirstOtherOpening] rougit
// celui-ci.
func TestUnePriseDUnCoequipierFermeLePortage(t *testing.T) {
	raws, ops := flagHandoffCas()
	closed, unnamed := closeByHandoff(raws, ops, flagHandoffScan(map[string]int{"1": 0, "2": 0}))
	if closed != 1 || unnamed != 0 {
		t.Fatalf("closeByHandoff rend (%d, %d), attendu (1, 0)", closed, unnamed)
	}
	if raws[0].t1 != 3000 || !raws[0].closed || raws[0].captured {
		t.Errorf("portage de « 1 » borne a %d (closed=%v) — attendu 3000, ferme et non capture",
			raws[0].t1, raws[0].closed)
	}
	if raws[1].t1 != 9000 {
		t.Errorf("portage de « 2 » borne a %d — aucune prise ne lui est posterieure et interieure",
			raws[1].t1)
	}
}

// TestUnePriseDUnADVERSAIRENeFermeRien — LE TEMOIN NEGATIF, et il porte tout le poids de la
// regle. Un adversaire qui prend un drapeau prend L'AUTRE : il ne dit rien de celui-ci. Sans le
// filtre par equipe, la regle fermerait la moitie des portages d'un CTF ordinaire — mesure du
// lot sur les 11 films CTF du parc : 57 portages et 775,1 s retires a tort, soit un ratio
// publie/oracle de 0,739 la ou l'oracle vaut 1,000.
//
// MUTATION : supprimer le test `drapeaux[o.xuid] != mien` de [flagFirstOtherOpening] rougit ce
// test, et lui seul.
func TestUnePriseDUnADVERSAIRENeFermeRien(t *testing.T) {
	raws, ops := flagHandoffCas()
	closed, unnamed := closeByHandoff(raws, ops, flagHandoffScan(map[string]int{"1": 0, "2": 1}))
	if closed != 0 || unnamed != 0 {
		t.Fatalf("closeByHandoff rend (%d, %d), attendu (0, 0)", closed, unnamed)
	}
	if raws[0].t1 != 6000 {
		t.Errorf("portage borne a %d, attendu 6000 : l'adversaire a pris L'AUTRE drapeau", raws[0].t1)
	}
}

// TestUnSeulDrapeauEnJeuToutePriseDUnAutreFerme — la carte hors catalogue d'objectifs (aucun
// socle) et la variante DRAPEAU NEUTRE ne mettent en jeu qu'UN drapeau : il n'appartient a
// personne, et toute prise d'un autre joueur le borne, coequipier ou non.
func TestUnSeulDrapeauEnJeuToutePriseDUnAutreFerme(t *testing.T) {
	raws, ops := flagHandoffCas()
	scan := flagHandoffScan(map[string]int{"1": 0, "2": 1}) // ADVERSAIRES
	scan.Spawns = nil
	closed, unnamed := closeByHandoff(raws, ops, scan)
	if closed != 1 || unnamed != 0 || raws[0].t1 != 3000 {
		t.Errorf("(%d, %d) et borne %d — attendu (1, 0) et 3000 : un seul drapeau est en jeu",
			closed, unnamed, raws[0].t1)
	}
}

// TestSansEquipeLueLaRegleSeTaitEtSeCompte — l'artefact hors ligne (CLI sans faits, ouvrier sans
// base) doit rester celui d'avant : sans equipe LUE, aucun drapeau n'est nomme, rien n'est
// ferme, et le silence SE COMPTE.
func TestSansEquipeLueLaRegleSeTaitEtSeCompte(t *testing.T) {
	raws, ops := flagHandoffCas()
	closed, unnamed := closeByHandoff(raws, ops, flagHandoffScan(nil))
	if closed != 0 || unnamed != 2 || raws[0].t1 != 6000 {
		t.Errorf("(%d, %d) et borne %d — attendu (0, 2) et 6000", closed, unnamed, raws[0].t1)
	}
}

// TestUnePriseHorsDuPortageNeDeplaceRien — LA REGLE NE PEUT QUE RACCOURCIR. Une prise de
// coequipier POSTERIEURE a la fermeture, ou tombant exactement sur l'une des deux bornes,
// n'allonge ni ne raccourcit : seul un instant STRICTEMENT interieur a `]t0, t1[` borne.
func TestUnePriseHorsDuPortageNeDeplaceRien(t *testing.T) {
	teams := map[string]int{"1": 0, "2": 0}
	for _, cas := range []struct {
		nom string
		at  int64
	}{
		{"apres la fermeture", 9000},
		{"sur la borne haute", 6000},
		{"sur la borne basse", 1000},
	} {
		raws, ops := flagHandoffCas()
		ops[1].t0, raws[1].t0 = cas.at, cas.at
		closed, _ := closeByHandoff(raws, ops, flagHandoffScan(teams))
		if closed != 0 || raws[0].t1 != 6000 {
			t.Errorf("%s : (%d) et borne %d — attendu 0 passage et 6000", cas.nom, closed, raws[0].t1)
		}
	}
}

// TestUneCarteAPlusDeDeuxDrapeauxAdversesSeTait — l'abstention, et elle SE COMPTE. Quatre socles,
// deux par camp : l'equipe du porteur ne designe plus un drapeau unique, donc la regle ne juge
// rien.
func TestUneCarteAPlusDeDeuxDrapeauxAdversesSeTait(t *testing.T) {
	raws, ops := flagHandoffCas()
	scan := flagHandoffScan(map[string]int{"1": 0, "2": 0})
	scan.Spawns = []FlagSpawn{
		{Team: 0, X: 0, Y: 0}, {Team: 0, X: 10, Y: 0},
		{Team: 1, X: 100, Y: 100}, {Team: 1, X: 300, Y: 300},
	}
	closed, unnamed := closeByHandoff(raws, ops, scan)
	if closed != 0 || unnamed != 2 {
		t.Errorf("(%d, %d) — attendu (0, 2) : deux drapeaux adverses ne se departagent pas",
			closed, unnamed)
	}
}
