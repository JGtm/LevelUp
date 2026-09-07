package replay

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/objectiveevents"
)

// TestFlagCarriesVolPuisCapture — un vol ouvre un `carried` du BON joueur, et la capture renvoie
// le drapeau a sa base.
func TestFlagCarriesVolPuisCapture(t *testing.T) {
	tracks := []Track{flagTestTrack(10, "aaa", 0, 99, 30, 40)}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 4000, Slot: 12, Stat: objectiveevents.StatFlagCaptures},
		},
		Identity: objectiveevents.FlatRoundIdentity(map[int]string{12: "aaa"}),
		Spawns:   []FlagSpawn{{Team: 0, X: 0, Y: 0}, {Team: 1, X: 100, Y: 100}},
	}
	got, cov := buildFlagCarries(scan, flagTestCtx(tracks, nil, 100))
	if len(got) != 2 {
		t.Fatalf("%d drapeaux publies, attendu 2 (un par socle)", len(got))
	}
	if cov.Openings != 1 || cov.Carries != 1 || !cov.Balanced() {
		t.Fatalf("couverture %+v : 1 prise attendue, publiee, et l'invariant tenu", *cov)
	}
	// Le vol s'est fait a (30,40), donc au socle de l'equipe 0 : c'est SON drapeau.
	f := flagOfTeam(t, got, 0)
	wantStates := []string{FlagStateHome, FlagStateCarried, FlagStateHome}
	assertFlagStates(t, f, wantStates)
	if f.Spans[1].T0 != 10 || f.Spans[1].T1 != 40 {
		t.Errorf("portage sur les frames [%d,%d], attendu [10,40]", f.Spans[1].T0, f.Spans[1].T1)
	}
	if f.Spans[1].XUID == nil || *f.Spans[1].XUID != "aaa" {
		t.Errorf("porteur %v, attendu \"aaa\"", f.Spans[1].XUID)
	}
	if f.Spans[2].X != 0 || f.Spans[2].Y != 0 {
		t.Errorf("retour a la base en (%v,%v), attendu le socle (0,0)", f.Spans[2].X, f.Spans[2].Y)
	}
}

// TestFlagCarriesMortLachePuisReprise — la mort du porteur lache le drapeau LA OU IL EST, et une
// prise a cet endroit reprend LE MEME drapeau, pas celui du socle le plus proche.
//
// C'est la regle qui tient la continuite de l'objet : sans elle, un drapeau lache pres du socle
// adverse serait « repris » comme s'il venait de ce socle.
func TestFlagCarriesMortLachePuisReprise(t *testing.T) {
	tracks := []Track{
		flagTestTrack(10, "aaa", 0, 40, 95, 95), // meurt pres du socle de l'equipe 1
		flagTestTrack(12, "bbb", 41, 99, 96, 96),
	}
	deaths := []Death{{XUID: 1, TimeMS: 4000}}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 5000, Slot: 14, Stat: objectiveevents.StatFlagGrabs},
		},
		Identity: objectiveevents.FlatRoundIdentity(map[int]string{12: "1", 14: "2"}),
		Spawns:   []FlagSpawn{{Team: 0, X: 0, Y: 0}, {Team: 1, X: 100, Y: 100}},
	}
	tracks[0].XUID, tracks[1].XUID = "1", "2"
	got, cov := buildFlagCarries(scan, flagTestCtx(tracks, deaths, 100))
	if cov.Carries != 2 || !cov.Balanced() {
		t.Fatalf("couverture %+v : 2 portages attendus, invariant tenu", *cov)
	}
	// Le vol part de (95,95) : le socle le plus proche est celui de l'equipe 1.
	f := flagOfTeam(t, got, 1)
	// La REPRISE n'est fermee par rien (ni capture, ni mort, ni nouvelle prise) : elle se publie
	// donc `carried_open`, pas `carried` — l'intervalle court jusqu'a la fin de l'axe et c'est
	// une borne haute.
	assertFlagStates(t, f,
		[]string{FlagStateHome, FlagStateCarried, FlagStateDropped, FlagStateCarriedOpen})
	if f.Spans[2].X != 95 || f.Spans[2].Y != 95 {
		t.Errorf("drapeau lache en (%v,%v), attendu la derniere position du porteur (95,95)",
			f.Spans[2].X, f.Spans[2].Y)
	}
	if f.Spans[3].XUID == nil || *f.Spans[3].XUID != "2" {
		t.Errorf("repreneur %v, attendu \"2\"", f.Spans[3].XUID)
	}
	// L'autre drapeau n'a jamais bouge : un seul span, a la maison.
	assertFlagStates(t, flagOfTeam(t, got, 0), []string{FlagStateHome})
}

// TestFlagCarriesFusionneLesPrisesJumelles — un vol emet AUSSI `flag_grabs` a la meme
// milliseconde ; les deux sont LA MEME action et ne doivent ouvrir qu'un portage.
func TestFlagCarriesFusionneLesPrisesJumelles(t *testing.T) {
	tracks := []Track{flagTestTrack(10, "aaa", 0, 99, 30, 40)}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagGrabs},
			{TimeMS: 1050, Slot: 12, Stat: objectiveevents.StatFlagSteals},
		},
		Identity: objectiveevents.FlatRoundIdentity(map[int]string{12: "aaa"}),
		Spawns:   []FlagSpawn{{Team: 0}, {Team: 1, X: 100, Y: 100}},
	}
	_, cov := buildFlagCarries(scan, flagTestCtx(tracks, nil, 100))
	if cov.Openings != 1 {
		t.Errorf("%d prises apres fusion, attendu 1", cov.Openings)
	}
}

// TestFlagCarriesPortageOuvert — un portage que RIEN ne ferme se publie `carried_open` jusqu'a la
// fin de l'axe, et la couverture le compte a part. Publier ce doute sous le nom d'un portage
// etabli serait affirmer une fin que le film ne date pas.
func TestFlagCarriesPortageOuvert(t *testing.T) {
	tracks := []Track{flagTestTrack(10, "aaa", 0, 99, 30, 40)}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
		},
		Identity: objectiveevents.FlatRoundIdentity(map[int]string{12: "aaa"}),
		Spawns:   []FlagSpawn{{Team: 0}, {Team: 1, X: 100, Y: 100}},
	}
	got, cov := buildFlagCarries(scan, flagTestCtx(tracks, nil, 100))
	if cov.Closed != 0 || cov.Open != 1 || cov.Carries != 1 {
		t.Fatalf("couverture %+v : 1 portage, ouvert, attendu", *cov)
	}
	f := flagOfTeam(t, got, 0)
	assertFlagStates(t, f, []string{FlagStateHome, FlagStateCarriedOpen})
	if f.Spans[1].T1 != 99 {
		t.Errorf("portage ouvert borne a la frame %d, attendu la derniere (99)", f.Spans[1].T1)
	}
}

// TestFlagCarriesMarqueurConfirme — le CONTROLE independant : une image-cle qui porte le marqueur
// sur le slot de bipede du porteur confirme le portage ; une image-cle sans marqueur ne le
// confirme pas, mais compte au denominateur.
//
// LES DEUX PORTAGES SONT FERMES PAR UNE CAPTURE, et c'est le sujet du test suivant : le controle
// publie sous `markerObserved` ne compte QUE les portages fermes.
func TestFlagCarriesMarqueurConfirme(t *testing.T) {
	tracks := []Track{flagTestTrack(10, "7", 0, 99, 30, 40), flagTestTrack(12, "8", 0, 99, 60, 60)}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 3000, Slot: 12, Stat: objectiveevents.StatFlagCaptures},
			{TimeMS: 5000, Slot: 14, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 7000, Slot: 14, Stat: objectiveevents.StatFlagCaptures},
		},
		Identity: objectiveevents.FlatRoundIdentity(map[int]string{12: "7", 14: "8"}),
		Spawns:   []FlagSpawn{{Team: 0}, {Team: 1, X: 100, Y: 100}},
		Marks: filmdec.CarrierMarkScan{
			KeyframeUS: []uint64{2_000_000, 6_000_000},
			Marks:      []filmdec.CarrierMark{{TimestampUS: 2_000_000, Slot: 10}},
		},
	}
	ctx := flagTestCtx(tracks, nil, 100)
	ctx.slotXUID = map[uint32]uint64{10: 7, 12: 8}
	_, cov := buildFlagCarries(scan, ctx)
	if cov.Closed != 2 {
		t.Fatalf("%d portages fermes, attendu 2 (deux captures)", cov.Closed)
	}
	if cov.MarkerObserved != 2 {
		t.Errorf("%d portages avec image-cle, attendu 2", cov.MarkerObserved)
	}
	if cov.MarkerConfirmed != 1 {
		t.Errorf("%d portages confirmes par le marqueur, attendu 1", cov.MarkerConfirmed)
	}
}

// TestFlagCarriesMarqueurSurLesFermesSeuls — un portage OUVERT observe par une image-cle ne
// compte PAS au denominateur du controle : il compte au sien.
//
// C'EST LA MESURE QUI L'IMPOSE, PAS UNE COMMODITE. Un portage ouvert est trop long par
// construction (le lacher volontaire n'est date par rien) : ses images-cles tardives tombent
// apres que le drapeau a ete lache. Les melanger fait passer le controle de 37/37 a 37/42 —
// c'est-a-dire qu'il ferait juger la justesse des bornes par des portages qui n'en ont pas.
func TestFlagCarriesMarqueurSurLesFermesSeuls(t *testing.T) {
	tracks := []Track{flagTestTrack(10, "7", 0, 99, 30, 40)}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
		},
		Identity: objectiveevents.FlatRoundIdentity(map[int]string{12: "7"}),
		Spawns:   []FlagSpawn{{Team: 0}, {Team: 1, X: 100, Y: 100}},
		Marks:    filmdec.CarrierMarkScan{KeyframeUS: []uint64{2_000_000}},
	}
	ctx := flagTestCtx(tracks, nil, 100)
	ctx.slotXUID = map[uint32]uint64{10: 7}
	_, cov := buildFlagCarries(scan, ctx)
	if cov.MarkerObserved != 0 || cov.MarkerConfirmed != 0 {
		t.Errorf("controle des FERMES %d/%d, attendu 0/0 : ce portage n'est pas ferme",
			cov.MarkerConfirmed, cov.MarkerObserved)
	}
	if cov.OpenObserved != 1 || cov.OpenConfirmed != 0 {
		t.Errorf("controle des OUVERTS %d/%d, attendu 0/1 — publie, jamais tu",
			cov.OpenConfirmed, cov.OpenObserved)
	}
}
