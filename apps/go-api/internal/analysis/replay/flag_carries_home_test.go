package replay

// flag_carries_home_test.go — LE DRAPEAU QUI RENTRE FERME LE PORTAGE (lot 6.11, item 2).
//
// CE QUE CES TESTS FERMENT. Un joueur qui lache puis REPREND le meme drapeau produit deux prises
// du meme slot ; la seconde borne la premiere, mais le lacher entre les deux n'est date par rien
// quand aucune vie libre ne nait a ses pieds. Tout le sejour AU SOL etait alors publie comme du
// portage — `16ea3668` / 2535417044536883 : 55,1 s pour 0,9 s a l'oracle.
//
// LE DRAPEAU EST RENTRE ENTRE-TEMPS, ET DEUX CHAINES LE DATENT : le RETOUR CREDITE, que l'equipe
// de son auteur nomme, et la RENTREE DE L'OBJET, que le socle nomme. Un drapeau chez lui n'est
// dans la main de personne.

import (
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// flagHomeScan monte un portage de « 1 » a 1 000 ms, ferme par sa mort a 6 000 ms (frame 60), et
// un `flag_returns` credite a « 2 » a 3 000 ms (frame 30).
func flagHomeScan(teams map[string]int, free []flagFreeLife) FlagCarryScan {
	return FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 3000, Slot: 14, Stat: objectiveevents.StatFlagReturns},
		},
		Identity: objectiveevents.FlatRoundIdentity(map[int]string{12: "1", 14: "2"}),
		Spawns:   flagInvariantSpawns(), // [0] equipe 1 en (0,0), [1] equipe 0 en (100,100)
		TeamOf:   teams,
		Free:     free,
	}
}

// flagHomeCtx : la piste du porteur, loin des deux socles, et sa mort a 6 000 ms.
func flagHomeCtx() flagCarryCtx {
	tracks := []Track{flagTestTrack(12, "1", 0, 99, 50, 50)}
	return flagTestCtx(tracks, []Death{{XUID: 1, TimeMS: 6000}}, 100)
}

// TestUnRetourCrediteFermeLePortageDeSonDrapeau — LE POINT DE L'ITEM, chaine creditee. « 1 » est
// de l'equipe 0, il porte donc le drapeau de l'equipe 1. « 2 » est de l'equipe 1 : le retour
// qu'on lui credite est celui de SON drapeau, donc de CELUI-LA. Le portage s'arrete a la frame 30
// et le drapeau est publie CHEZ LUI, pas au sol.
//
// MUTATION : rendre `flagOfOwner` insensible a l'equipe (rendre toujours 0) fait passer le temoin
// negatif ci-dessous au rouge ; supprimer l'appel a [closeByHomecoming] rougit celui-ci.
func TestUnRetourCrediteFermeLePortageDeSonDrapeau(t *testing.T) {
	got, cov := buildFlagCarries(flagHomeScan(map[string]int{"1": 0, "2": 1}, nil), flagHomeCtx())
	f := flagOfTeam(t, got, 1)
	assertFlagStates(t, f, []string{FlagStateHome, FlagStateCarried, FlagStateHome})
	if f.Spans[1].T1 != 30 {
		t.Errorf("portage ferme a la frame %d, attendu 30 (le retour credite)", f.Spans[1].T1)
	}
	if cov.ClosedByReturn != 1 || cov.ClosedByHome != 0 || !cov.Balanced() {
		t.Errorf("couverture %+v, attendu 1 ferme par le retour credite", *cov)
	}
	// LE RETOUR N'EST PAS UNE ABSTENTION : il a servi, il ne doit pas se compter comme perdu.
	if cov.AmbiguousReturns != 0 {
		t.Errorf("ambiguousReturns = %d, attendu 0 : ce retour a ferme un portage",
			cov.AmbiguousReturns)
	}
}

// TestUnRetourDuDRAPEAUADVERSENeFermeRien — LE TEMOIN NEGATIF. « 2 » est cette fois de l'equipe 0,
// comme le porteur : le retour qu'on lui credite est celui du drapeau de l'equipe 0, c'est-a-dire
// l'AUTRE. Il ne dit rien de celui que « 1 » porte.
func TestUnRetourDuDRAPEAUADVERSENeFermeRien(t *testing.T) {
	got, cov := buildFlagCarries(flagHomeScan(map[string]int{"1": 0, "2": 0}, nil), flagHomeCtx())
	f := flagOfTeam(t, got, 1)
	if s := f.Spans[1]; s.T1 != 60 {
		t.Errorf("portage ferme a la frame %d, attendu 60 (la mort) : le retour porte sur "+
			"l'autre drapeau", s.T1)
	}
	if cov.ClosedByReturn != 0 || cov.ClosedByHome != 0 {
		t.Errorf("couverture %+v, attendu aucune fermeture par rentree", *cov)
	}
}

// TestSansEquipeLueLeRetourNeNommeAucunDrapeau — l'artefact hors ligne reste celui d'avant : sans
// equipe, le retour ne nomme rien et ne ferme rien.
func TestSansEquipeLueLeRetourNeNommeAucunDrapeau(t *testing.T) {
	_, cov := buildFlagCarries(flagHomeScan(nil, nil), flagHomeCtx())
	if cov.ClosedByReturn != 0 || cov.ClosedByHome != 0 || cov.CarrierTeamUnknown == 0 {
		t.Errorf("couverture %+v : sans equipe lue, rien n'est ferme et le silence se compte", *cov)
	}
}

// TestUneRentreeDeLObjetFermeLePortage — la seconde chaine, celle du RETOUR AUTOMATIQUE que
// personne ne credite : l'objet est RE-CREE a son socle, et le socle le nomme.
func TestUneRentreeDeLObjetFermeLePortage(t *testing.T) {
	// La vie libre nait au socle du drapeau de l'equipe 1 — celui que « 1 » porte.
	free := []flagFreeLife{flagTestLife(20, [2]float32{0, 0})}
	scan := flagHomeScan(map[string]int{"1": 0, "2": 0}, free) // le retour porte sur l'AUTRE drapeau
	got, cov := buildFlagCarries(scan, flagHomeCtx())
	f := flagOfTeam(t, got, 1)
	assertFlagStates(t, f, []string{FlagStateHome, FlagStateCarried, FlagStateHome})
	if f.Spans[1].T1 != 20 {
		t.Errorf("portage ferme a la frame %d, attendu 20 (la rentree de l'objet)", f.Spans[1].T1)
	}
	if cov.ClosedByHome != 1 || cov.ClosedByReturn != 0 || !cov.Balanced() {
		t.Errorf("couverture %+v, attendu 1 ferme par la rentree de l'objet", *cov)
	}
}

// TestUneRentreeAmbigueNeFermeRien — L'ABSTENTION DE LA RENTREE. Un portage de l'AUTRE drapeau
// s'acheve dans la seconde qui entoure la naissance : celle-ci peut etre sa chute au pied du
// socle, et non le retour de ce drapeau-ci. On se tait.
//
// MUTATION : retirer l'appel a [flagAutreLacherProche] rougit ce test.
func TestUneRentreeAmbigueNeFermeRien(t *testing.T) {
	free := []flagFreeLife{flagTestLife(20, [2]float32{0, 0})}
	scan := flagHomeScan(map[string]int{"1": 0, "2": 1}, free)
	// « 2 » (equipe 1) porte le drapeau de l'equipe 0 et meurt a la frame 20 : sa chute peut
	// avoir produit la naissance lue au socle de l'equipe 1.
	scan.Events = append(scan.Events,
		objectiveevents.NamedEvent{TimeMS: 500, Slot: 14, Stat: objectiveevents.StatFlagSteals})
	tracks := []Track{
		flagTestTrack(12, "1", 0, 99, 50, 50),
		flagTestTrack(14, "2", 0, 99, 0, 0),
	}
	ctx := flagTestCtx(tracks, []Death{{XUID: 1, TimeMS: 6000}, {XUID: 2, TimeMS: 2000}}, 100)

	_, cov := buildFlagCarries(scan, ctx)
	if cov.ClosedByHome != 0 {
		t.Errorf("couverture %+v : la naissance au socle est ambigue, rien ne doit fermer", *cov)
	}
}

// TestUneRentreeApresLaFermetureNeDeplaceRien — LA REGLE NE PEUT QUE RACCOURCIR : une rentree
// posterieure a la borne en place n'allonge ni ne raccourcit.
func TestUneRentreeApresLaFermetureNeDeplaceRien(t *testing.T) {
	free := []flagFreeLife{flagTestLife(80, [2]float32{0, 0})}
	got, cov := buildFlagCarries(flagHomeScan(map[string]int{"1": 0, "2": 0}, free), flagHomeCtx())
	f := flagOfTeam(t, got, 1)
	if f.Spans[1].T1 != 60 || cov.ClosedByHome != 0 {
		t.Errorf("portage ferme a %d, couverture %+v — attendu 60 et aucune fermeture",
			f.Spans[1].T1, *cov)
	}
}
