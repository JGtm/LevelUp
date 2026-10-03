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

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/objectives"
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
		{xuid: "1", t0: 1000, t1: 6000, steal: true, closed: true, closedBy: flagCloserBound, flagIndex: -1},
		{xuid: "2", t0: 3000, t1: 9000, closed: true, closedBy: flagCloserBound, flagIndex: -1},
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
	unnamed := closeByHandoff(raws, ops, flagHandoffScan(map[string]int{"1": 0, "2": 0}))
	closed := flagComptePar(raws, flagCloserHandoff)
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
	unnamed := closeByHandoff(raws, ops, flagHandoffScan(map[string]int{"1": 0, "2": 1}))
	closed := flagComptePar(raws, flagCloserHandoff)
	if closed != 0 || unnamed != 0 {
		t.Fatalf("closeByHandoff rend (%d, %d), attendu (0, 0)", closed, unnamed)
	}
	if raws[0].t1 != 6000 {
		t.Errorf("portage borne a %d, attendu 6000 : l'adversaire a pris L'AUTRE drapeau", raws[0].t1)
	}
}

// TestUnSeulDrapeauEnJeuToutePriseDUnAutreFerme — la variante DRAPEAU NEUTRE (un seul socle
// retenu, cf. flag_neutral.go) ne met en jeu qu'UN drapeau : il n'appartient a personne, et toute
// prise d'un autre joueur le borne, coequipier ou non.
//
// CE TEST COUVRAIT AUSSI LA CARTE HORS CATALOGUE (aucun socle) jusqu'au lot J9.1-J9.2 : sa premisse
// y etait fausse (constat RB1-5), cf. [TestHorsCatalogueLeNombreDeDrapeauxNeSeSupposePas].
func TestUnSeulDrapeauEnJeuToutePriseDUnAutreFerme(t *testing.T) {
	raws, ops := flagHandoffCas()
	scan := flagHandoffScan(map[string]int{"1": 0, "2": 1}) // ADVERSAIRES
	scan.Spawns = []FlagSpawn{{Team: TeamNeutral, Neutral: true, X: 50, Y: 50}}
	unnamed := closeByHandoff(raws, ops, scan)
	closed := flagComptePar(raws, flagCloserHandoff)
	if closed != 1 || unnamed != 0 || raws[0].t1 != 3000 {
		t.Errorf("(%d, %d) et borne %d — attendu (1, 0) et 3000 : un seul drapeau est en jeu",
			closed, unnamed, raws[0].t1)
	}
}

// TestHorsCatalogueLeNombreDeDrapeauxNeSeSupposePas — CONSTAT RB1-5 (audit du 2026-09-24, lot
// J9.2). Une carte hors du catalogue d'objectifs ne donne AUCUN socle : rien ne dit combien de
// drapeaux sont en jeu. Le supposer UNIQUE faisait de toute prise d'un ADVERSAIRE un passage de
// main en main — c'est exactement la regle sans son filtre d'equipe, que la mesure du lot 6.11
// chiffre a 57 portages et 775,1 s retires a tort sur les 11 films CTF du parc. Le nombre n'est
// donc plus suppose : aucun drapeau n'est nomme par l'equipe, rien n'est ferme, le silence se
// compte (`carrierTeamUnknown`) et le repli NOMME se declenche une fois par portage non juge.
//
// MUTATION : `len(spawns) <= 1` dans [flagSingleInPlay] rougit ce test et
// [TestHorsCatalogueUnRetourCrediteNeFermeRien].
func TestHorsCatalogueLeNombreDeDrapeauxNeSeSupposePas(t *testing.T) {
	for _, cas := range []struct {
		nom   string
		teams map[string]int
	}{
		{"adversaires", map[string]int{"1": 0, "2": 1}},
		{"coequipiers", map[string]int{"1": 0, "2": 0}},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			raws, ops := flagHandoffCas()
			scan := flagHandoffScan(cas.teams)
			scan.Spawns = nil
			unnamed := closeByHandoff(raws, ops, scan)
			closed := flagComptePar(raws, flagCloserHandoff)
			if closed != 0 || unnamed != 2 || raws[0].t1 != 6000 {
				t.Errorf("(%d, %d) et borne %d — attendu (0, 2) et 6000 : le nombre de drapeaux "+
					"d'une carte hors catalogue n'est pas lu", closed, unnamed, raws[0].t1)
			}
		})
	}
}

// TestHorsCatalogueLeRepliSeCompteParPortage — le silence de la regle n'est pas muet : le repli
// NOMME se declenche une fois par portage publie, et zero fois des qu'un socle est connu.
func TestHorsCatalogueLeRepliSeCompteParPortage(t *testing.T) {
	for _, cas := range []struct {
		nom    string
		spawns []FlagSpawn
		veut   int
	}{
		{"hors catalogue", nil, 2},
		{"deux socles", flagInvariantSpawns(), 0},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			scan := FlagCarryScan{
				Scanned: true, Signals: flagTestSignals(),
				Events: []objectives.NamedEvent{
					{TimeMS: 1000, Slot: 12, Stat: objectives.StatFlagSteals},
					{TimeMS: 3000, Slot: 14, Stat: objectives.StatFlagSteals},
				},
				Identity: objectives.FlatRoundIdentity(map[int]string{12: "1", 14: "2"}),
				Spawns:   cas.spawns, TeamOf: map[string]int{"1": 0, "2": 1},
			}
			ctx := flagTestCtx([]Track{flagTestTrack(10, "1", 0, 99, 2, 2),
				flagTestTrack(11, "2", 0, 99, 98, 98)}, nil, 100)
			ctx.fb = fallback.NouveauCompteur()
			if _, cov := buildFlagCarries(scan, ctx); cov == nil || cov.Carries != 2 {
				t.Fatalf("couverture %+v : deux portages publies attendus", cov)
			}
			if n := ctx.fb.Compte(fallback.NomNombreDrapeauxHorsCatalogueSansPassage); n != cas.veut {
				t.Errorf("repli declenche %d fois, attendu %d", n, cas.veut)
			}
		})
	}
}

// TestHorsCatalogueUnRetourCrediteNeFermeRien — le MEME postulat nommait le drapeau d'un retour
// credite : hors catalogue, le retour d'un joueur de l'equipe 0 (qui rend SON drapeau) fermait le
// portage de son coequipier, qui tient l'AUTRE.
func TestHorsCatalogueUnRetourCrediteNeFermeRien(t *testing.T) {
	raws, _ := flagHandoffCas()
	scan := flagHandoffScan(map[string]int{"1": 0, "2": 0, "3": 0})
	scan.Spawns = nil
	scan.Identity = objectives.FlatRoundIdentity(map[int]string{16: "3"})
	scan.Events = []objectives.NamedEvent{{TimeMS: 2000, Slot: 16, Stat: objectives.StatFlagReturns}}
	closeByHomecoming(raws, scan, flagTestCtx(nil, nil, 100))
	if n := flagComptePar(raws, flagCloserReturn); n != 0 || raws[0].t1 != 6000 {
		t.Errorf("%d portages fermes par un retour, borne %d — attendu 0 et 6000 : hors catalogue "+
			"le retour ne nomme aucun drapeau", n, raws[0].t1)
	}
}

// TestSansEquipeLueLaRegleSeTaitEtSeCompte — l'artefact hors ligne (CLI sans faits, ouvrier sans
// base) doit rester celui d'avant : sans equipe LUE, aucun drapeau n'est nomme, rien n'est
// ferme, et le silence SE COMPTE.
func TestSansEquipeLueLaRegleSeTaitEtSeCompte(t *testing.T) {
	raws, ops := flagHandoffCas()
	unnamed := closeByHandoff(raws, ops, flagHandoffScan(nil))
	closed := flagComptePar(raws, flagCloserHandoff)
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
		closeByHandoff(raws, ops, flagHandoffScan(teams))
		closed := flagComptePar(raws, flagCloserHandoff)
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
	unnamed := closeByHandoff(raws, ops, scan)
	closed := flagComptePar(raws, flagCloserHandoff)
	if closed != 0 || unnamed != 2 {
		t.Errorf("(%d, %d) — attendu (0, 2) : deux drapeaux adverses ne se departagent pas",
			closed, unnamed)
	}
}
