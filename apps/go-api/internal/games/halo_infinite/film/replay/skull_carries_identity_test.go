package replay

import (
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// skull_carries_identity_test.go — QUI NOMME LE PORTEUR DU CRANE : ce paquet, ou son appelant ?
//
// MEME REGLE QUE LE DRAPEAU (flag_carries_identity_test.go), et pour la MEME cause mesuree. Le
// calque nommait son porteur par les SEULS instants de mort, qui en exigent TROIS coincidents :
// un joueur qui meurt moins de trois fois dans la manche echappe au pont par construction, son
// train de tics part en `noBridge` et AUCUN intervalle n'est publie pour lui.
//
// LA MESURE QUI FONDE CE CABLAGE (2026-09-10, quatre films Oddball du parc cuits hors ligne) :
// `43716616` perd 62,3 s de portage sur son PLUS GROS porteur (2533274978052136, oracle API
// `time_as_skull_carrier_seconds`), `c88ec007` 25,8 s, `d9781168` 1 train. Le pont COMPLETE que
// l'appelant resout deja pour les actions d'objectif et le drapeau ferme ce trou.
//
// Ces tests figent la REGLE de preference, pas la correction du pont lui-meme : celle-la est
// prouvee a sa source (`objectiveevents/slotidentity_completion_test.go`).

// skullIdentityRecs — deux slots dont le pont PAR MORTS ne nomme que le premier : le slot 10
// aligne trois progressions du compteur de morts sur le fil de "111", le slot 12 n'en aligne que
// deux (sous `deathInstantMin` = 3). Le slot 12 porte les tics de score de mode.
func skullIdentityRecs() ([]objectiveevents.StatRecord, []Death) {
	mort := func(t, slot int, kills, deaths int64) objectiveevents.StatRecord {
		return objectiveevents.StatRecord{TimeMS: t, Slot: slot, Round: 0,
			Comps: map[int]objectiveevents.StatValue{2: {A: kills, B: deaths}}}
	}
	tic := func(t, slot int, v int64) objectiveevents.StatRecord {
		return objectiveevents.StatRecord{TimeMS: t, Slot: slot, Round: 0,
			Comps: map[int]objectiveevents.StatValue{0: {A: v}}}
	}
	recs := []objectiveevents.StatRecord{
		mort(1000, 10, 1, 1), mort(2000, 10, 2, 2), mort(3000, 10, 3, 3),
		mort(5000, 12, 1, 1), mort(6000, 12, 2, 2),
		tic(7000, 12, 1), tic(8000, 12, 2), tic(9000, 12, 3),
	}
	deaths := []Death{
		{XUID: 111, TimeMS: 1000}, {XUID: 111, TimeMS: 2000}, {XUID: 111, TimeMS: 3000},
		{XUID: 222, TimeMS: 5000}, {XUID: 222, TimeMS: 6000},
	}
	return recs, deaths
}

// TestSkullIdentityOfResoutLocalementSansPontFourni — L'ARTEFACT HORS LIGNE RESTE ENTIER.
//
// `SkullInput.Identity` a zero, ce paquet resout par les seuls instants de mort, comme avant ce
// lot : un CLI qui n'ouvre aucune base construit le meme calque qu'hier.
func TestSkullIdentityOfResoutLocalementSansPontFourni(t *testing.T) {
	recs, deaths := skullIdentityRecs()
	got := skullIdentityOf(SkullInput{Records: recs}, Options{Deaths: deaths})
	if !got.Resolved() {
		t.Fatalf("aucun pont resolu : le calque ne nommerait plus personne hors ligne")
	}
	if x := got.AtRound(0, 10); x != "111" {
		t.Errorf("slot 10 = %q, attendu \"111\" (trois morts coincidentes)", x)
	}
	if x := got.AtRound(0, 12); x != "" {
		t.Errorf("slot 12 = %q, attendu vide : DEUX morts sont sous le seuil du pont par morts — "+
			"c'est precisement le trou que l'appelant vient combler", x)
	}
}

// TestSkullIdentityOfPrefereLePontDeLAppelant — LE PONT FOURNI L'EMPORTE : le slot 12 que la
// resolution locale laisse tomber est nomme.
func TestSkullIdentityOfPrefereLePontDeLAppelant(t *testing.T) {
	recs, deaths := skullIdentityRecs()
	fourni := objectiveevents.FlatRoundIdentity(map[int]string{10: "111", 12: "222"})
	got := skullIdentityOf(SkullInput{Records: recs, Identity: fourni}, Options{Deaths: deaths})
	if x := got.AtRound(0, 12); x != "222" {
		t.Errorf("slot 12 = %q, attendu \"222\" : le pont de l'appelant n'a pas ete retenu", x)
	}
	if x := got.AtRound(0, 10); x != "111" {
		t.Errorf("slot 10 = %q, attendu \"111\"", x)
	}
}

// TestSkullIdentityOfRespecteUnPontMuet — « PERSONNE N'A RESOLU » ET « LA RESOLUTION N'A NOMME
// PERSONNE » NE SONT PAS LA MEME CHOSE.
//
// Un appelant qui rend un pont VIDE a repondu : ce paquet ne doit pas passer outre en resolvant
// pour son compte. Le test le prouve avec des enregistrements que la resolution locale saurait
// nommer — s'il retombait dessus, le slot 10 porterait un nom.
func TestSkullIdentityOfRespecteUnPontMuet(t *testing.T) {
	recs, deaths := skullIdentityRecs()
	muet := objectiveevents.FlatRoundIdentity(nil)
	if !muet.Resolved() {
		t.Fatalf("le pont temoin doit se declarer RESOLU, sans quoi ce test ne prouve rien")
	}
	got := skullIdentityOf(SkullInput{Records: recs, Identity: muet}, Options{Deaths: deaths})
	if x := got.AtRound(0, 10); x != "" {
		t.Errorf("slot 10 = %q, attendu vide : un pont fourni muet est une reponse, "+
			"pas une invitation a resoudre soi-meme", x)
	}
}

// TestSkullCarriesPontFourniPublieLePortage — LE BOUT DE LA CHAINE : le meme film, deux ponts.
// Celui par morts laisse le train `noBridge` et ne publie AUCUN intervalle ; celui de l'appelant
// le publie, au bon joueur.
//
// C'est la mutation du lot, ecrite comme un test : le premier cas EST l'etat d'avant ce lot.
func TestSkullCarriesPontFourniPublieLePortage(t *testing.T) {
	recs, deaths := skullIdentityRecs()
	scanDe := func(identity objectiveevents.RoundIdentity) SkullCarryScan {
		return SkullCarryScan{Scanned: true, Records: recs, Identity: identity}
	}
	clock := matchClock{origin: 0, step: 1000, frames: 100000}

	// Avant le lot : le pont par morts se tait sur le slot 12, le train est perdu.
	carries, cov := buildSkullCarries(
		scanDe(skullIdentityOf(SkullInput{Records: recs}, Options{Deaths: deaths})),
		clock, carrierPresence{})
	if len(carries) != 0 || cov.NoBridge != 1 || !cov.Balanced() {
		t.Fatalf("pont par morts seul : %d portage(s), couverture %+v, attendu 0 portage et "+
			"1 sans pont", len(carries), *cov)
	}

	// Apres le lot : le pont complete de l'appelant nomme le slot, et le portage est publie.
	complet := objectiveevents.FlatRoundIdentity(map[int]string{10: "111", 12: "222"})
	carries, cov = buildSkullCarries(scanDe(complet), clock, carrierPresence{})
	if len(carries) != 1 || cov.NoBridge != 0 || !cov.Balanced() {
		t.Fatalf("pont complete : %d portage(s), couverture %+v, attendu 1 portage et "+
			"0 sans pont", len(carries), *cov)
	}
	// Bornes = tics +/- la demi-fenetre de tic mesuree sur ce film (lot 6.7-B1, item 2).
	demi := skullHalfTickFrames(recs, clock)
	if carries[0].XUID != "222" || carries[0].T0 != 7000-demi || carries[0].T1 != 9000+demi {
		t.Errorf("portage = %+v, attendu {222 %d %d}", carries[0], 7000-demi, 9000+demi)
	}
}
