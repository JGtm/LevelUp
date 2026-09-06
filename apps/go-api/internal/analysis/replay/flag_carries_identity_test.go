package replay

import (
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// flag_carries_identity_test.go — QUI NOMME LE PORTEUR : ce paquet, ou son appelant ?
//
// Depuis le schema 42, l'appelant PEUT poser un pont deja resolu sur `FlagInput.Identity` — le
// meme que celui des actions d'objectif, COMPLETE par le triplet la ou le pont par morts se
// tait. Ce fichier fige les trois cas de la regle : l'appelant l'emporte, son absence fait
// resoudre localement, et son SILENCE est une reponse — pas une absence.

// flagIdentityRecs — deux slots dont le pont PAR MORTS ne nomme que le premier : le slot 10
// aligne trois progressions du compteur de morts sur le fil de "aaa", le slot 12 n'en aligne
// que deux (sous `deathInstantMin` = 3).
func flagIdentityRecs() ([]objectiveevents.StatRecord, []Death) {
	rec := func(t, slot int, kills, deaths int64) objectiveevents.StatRecord {
		return objectiveevents.StatRecord{TimeMS: t, Slot: slot, Round: 0,
			Comps: map[int]objectiveevents.StatValue{2: {A: kills, B: deaths}}}
	}
	recs := []objectiveevents.StatRecord{
		rec(1000, 10, 1, 1), rec(2000, 10, 2, 2), rec(3000, 10, 3, 3),
		rec(5000, 12, 1, 1), rec(6000, 12, 2, 2),
	}
	deaths := []Death{
		{XUID: 111, TimeMS: 1000}, {XUID: 111, TimeMS: 2000}, {XUID: 111, TimeMS: 3000},
		{XUID: 222, TimeMS: 5000}, {XUID: 222, TimeMS: 6000},
	}
	return recs, deaths
}

// TestFlagIdentityOfResoutLocalementSansPontFourni — L'ARTEFACT HORS LIGNE RESTE ENTIER.
//
// `FlagInput.Identity` a zero, ce paquet resout par les seuls instants de mort, comme avant le
// schema 42 : un CLI qui n'ouvre aucune base construit le meme calque qu'hier.
func TestFlagIdentityOfResoutLocalementSansPontFourni(t *testing.T) {
	recs, deaths := flagIdentityRecs()
	got := flagIdentityOf(FlagInput{Records: recs}, Options{Deaths: deaths})
	if !got.Resolved() {
		t.Fatalf("aucun pont resolu : le calque ne nommerait plus personne hors ligne")
	}
	if x := got.At(10, 1000); x != "111" {
		t.Errorf("slot 10 = %q, attendu \"111\" (trois morts coincidentes)", x)
	}
	if x := got.At(12, 5000); x != "" {
		t.Errorf("slot 12 = %q, attendu vide : DEUX morts sont sous le seuil du pont par morts — "+
			"c'est precisement le trou que l'appelant vient combler", x)
	}
}

// TestFlagIdentityOfPrefereLePontDeLAppelant — LE PONT FOURNI L'EMPORTE, et c'est ce qui fait
// tomber le plafond du calque : le slot 12 que la resolution locale laisse tomber est nomme.
func TestFlagIdentityOfPrefereLePontDeLAppelant(t *testing.T) {
	recs, deaths := flagIdentityRecs()
	fourni := objectiveevents.FlatRoundIdentity(map[int]string{10: "111", 12: "222"})
	got := flagIdentityOf(FlagInput{Records: recs, Identity: fourni}, Options{Deaths: deaths})
	if x := got.At(12, 5000); x != "222" {
		t.Errorf("slot 12 = %q, attendu \"222\" : le pont de l'appelant n'a pas ete retenu", x)
	}
	if x := got.At(10, 1000); x != "111" {
		t.Errorf("slot 10 = %q, attendu \"111\"", x)
	}
}

// TestFlagIdentityOfRespecteUnPontMuet — « PERSONNE N'A RESOLU » ET « LA RESOLUTION N'A NOMME
// PERSONNE » NE SONT PAS LA MEME CHOSE.
//
// Un appelant qui rend un pont VIDE a repondu : ce paquet ne doit pas passer outre en resolvant
// pour son compte. Le test le prouve avec des enregistrements que la resolution locale saurait
// nommer — s'il retombait dessus, le slot 10 porterait un nom.
func TestFlagIdentityOfRespecteUnPontMuet(t *testing.T) {
	recs, deaths := flagIdentityRecs()
	muet := objectiveevents.FlatRoundIdentity(nil)
	if !muet.Resolved() {
		t.Fatalf("le pont temoin doit se declarer RESOLU, sans quoi ce test ne prouve rien")
	}
	got := flagIdentityOf(FlagInput{Records: recs, Identity: muet}, Options{Deaths: deaths})
	if x := got.At(10, 1000); x != "" {
		t.Errorf("slot 10 = %q, attendu vide : un pont fourni muet est une reponse, "+
			"pas une invitation a resoudre soi-meme", x)
	}
}

// TestFlagCarriesPontFourniPublieLePortage — LE BOUT DE LA CHAINE : le meme film, le meme
// oracle, deux ponts. Celui par morts laisse la prise `noBridge` et ne publie AUCUN intervalle ;
// celui de l'appelant la publie, au bon joueur.
//
// C'est la mutation du lot, ecrite comme un test : le premier cas EST l'etat du schema 41.
func TestFlagCarriesPontFourniPublieLePortage(t *testing.T) {
	tracks := []Track{flagTestTrack(12, "222", 0, 99, 30, 40)}
	scanDe := func(identity objectiveevents.RoundIdentity) FlagCarryScan {
		return FlagCarryScan{
			Scanned: true, Signals: flagTestSignals(),
			Events: []objectiveevents.NamedEvent{
				{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
				{TimeMS: 4000, Slot: 12, Stat: objectiveevents.StatFlagCaptures},
			},
			Identity: identity,
			Spawns:   []FlagSpawn{{Team: 0, X: 0, Y: 0}, {Team: 1, X: 100, Y: 100}},
		}
	}
	recs, deaths := flagIdentityRecs()

	// Schema 41 : le pont par morts se tait sur le slot 12, la prise est perdue.
	_, cov := buildFlagCarries(scanDe(flagIdentityOf(FlagInput{Records: recs}, Options{Deaths: deaths})),
		flagTestCtx(tracks, nil, 100))
	if cov.Carries != 0 || cov.NoBridge != 1 || !cov.Balanced() {
		t.Fatalf("pont par morts seul : couverture %+v, attendu 0 portage et 1 sans pont", *cov)
	}

	// Schema 42 : le pont complete de l'appelant nomme le slot, et le portage est publie.
	complet := objectiveevents.FlatRoundIdentity(map[int]string{12: "222"})
	got, cov := buildFlagCarries(scanDe(complet), flagTestCtx(tracks, nil, 100))
	if cov.Carries != 1 || cov.NoBridge != 0 || !cov.Balanced() {
		t.Fatalf("pont complete : couverture %+v, attendu 1 portage et 0 sans pont", *cov)
	}
	f := flagOfTeam(t, got, 0)
	assertFlagStates(t, f, []string{FlagStateHome, FlagStateCarried, FlagStateHome})
	if f.Spans[1].XUID == nil || *f.Spans[1].XUID != "222" {
		t.Errorf("porteur %v, attendu \"222\"", f.Spans[1].XUID)
	}
}
