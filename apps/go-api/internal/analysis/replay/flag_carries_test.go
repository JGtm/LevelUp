package replay

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/objectiveevents"
)

// flag_carries_test.go — LES REGLES DU DRAPEAU, sans film.
//
// Chaque test fige UNE regle et la fait TOMBER si elle disparait. Les entrees sont synthetiques :
// des evenements de statistique dates, un fil des morts, deux socles et des pistes publiees —
// exactement ce que le constructeur passera. La verite terrain sur films reels vit dans
// l'instrument sous garde `OBJ_FILM` (`objectifs_phase1_drapeau_test.go`), qui appelle CE code.

// flagTestCtx fabrique un contexte a l'echelle 100 ms/frame, sans decalage d'horloge : la frame
// d'un instant du match est donc `ms / 100`, ce qui rend les attentes lisibles.
func flagTestCtx(tracks []Track, deaths []Death, frames int) flagCarryCtx {
	return flagCarryCtx{origin: 0, step: 100_000, frames: frames, tracks: tracks, deaths: deaths}
}

// flagTestTrack fabrique une piste d'un joueur, un point toutes les frames de from a to.
func flagTestTrack(slot uint32, xuid string, from, to int, x, y float32) Track {
	tr := Track{Slot: slot, Team: TeamNeutral, XUID: xuid, StartFrame: from, EndFrame: to}
	for t := from; t <= to; t++ {
		tr.Points = append(tr.Points, Point{T: t, X: x, Y: y})
	}
	return tr
}

// flagTestSignals rend des signaux qui tiennent la regle de mode (le verdict n'est pas le sujet
// de ces tests-ci : il a les siens, dans `objectiveevents/flagfilm_test.go`).
func flagTestSignals() objectiveevents.FlagFilmSignals {
	return objectiveevents.FlagFilmSignals{Bursts: 3, Captures: 3, Steals: 2, Grabs: 4}
}

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
		Identity: map[int]string{12: "aaa"},
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
		Identity: map[int]string{12: "1", 14: "2"},
		Spawns:   []FlagSpawn{{Team: 0, X: 0, Y: 0}, {Team: 1, X: 100, Y: 100}},
	}
	tracks[0].XUID, tracks[1].XUID = "1", "2"
	got, cov := buildFlagCarries(scan, flagTestCtx(tracks, deaths, 100))
	if cov.Carries != 2 || !cov.Balanced() {
		t.Fatalf("couverture %+v : 2 portages attendus, invariant tenu", *cov)
	}
	// Le vol part de (95,95) : le socle le plus proche est celui de l'equipe 1.
	f := flagOfTeam(t, got, 1)
	assertFlagStates(t, f, []string{FlagStateHome, FlagStateCarried, FlagStateDropped, FlagStateCarried})
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

// TestFlagCarriesPriseSansPontComptee — une prise dont le slot statborg n'est pas nomme ne publie
// RIEN et se compte sous `NoBridge`. Se taire vaut mieux qu'attribuer le drapeau au hasard.
func TestFlagCarriesPriseSansPontComptee(t *testing.T) {
	tracks := []Track{flagTestTrack(10, "aaa", 0, 99, 30, 40)}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 2000, Slot: 20, Stat: objectiveevents.StatFlagSteals}, // slot non apparie
		},
		Identity: map[int]string{12: "aaa"},
		Spawns:   []FlagSpawn{{Team: 0}, {Team: 1, X: 100, Y: 100}},
	}
	_, cov := buildFlagCarries(scan, flagTestCtx(tracks, nil, 100))
	if cov.Openings != 2 || cov.NoBridge != 1 || cov.Carries != 1 {
		t.Errorf("couverture %+v : 2 prises, 1 sans pont, 1 publiee attendues", *cov)
	}
	if !cov.Balanced() {
		t.Errorf("invariant de couverture rompu : %+v", *cov)
	}
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
		Identity: map[int]string{12: "aaa"},
		Spawns:   []FlagSpawn{{Team: 0}, {Team: 1, X: 100, Y: 100}},
	}
	_, cov := buildFlagCarries(scan, flagTestCtx(tracks, nil, 100))
	if cov.Openings != 1 {
		t.Errorf("%d prises apres fusion, attendu 1", cov.Openings)
	}
}

// TestFlagCarriesPorteurTueAmbigu — `flag_carriers_killed` est credite au TUEUR : quand deux
// portages sont ouverts, rien ne dit lequel tombe. Il ne doit alors fermer personne, et se
// compter.
func TestFlagCarriesPorteurTueAmbigu(t *testing.T) {
	tracks := []Track{
		flagTestTrack(10, "1", 0, 99, 10, 10),
		flagTestTrack(12, "2", 0, 99, 90, 90),
	}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 1000, Slot: 14, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 3000, Slot: 16, Stat: objectiveevents.StatFlagCarriersKilled},
		},
		Identity: map[int]string{12: "1", 14: "2", 16: "3"},
		Spawns:   []FlagSpawn{{Team: 0}, {Team: 1, X: 100, Y: 100}},
	}
	got, cov := buildFlagCarries(scan, flagTestCtx(tracks, nil, 100))
	if cov.AmbiguousCarrierKills != 1 {
		t.Errorf("%d porteurs tues ambigus, attendu 1", cov.AmbiguousCarrierKills)
	}
	for _, f := range got {
		for _, s := range f.Spans {
			if s.State == FlagStateCarried && s.T1 < 99 {
				t.Errorf("portage ferme a la frame %d par un evenement ambigu", s.T1)
			}
		}
	}
}

// TestFlagCarriesFilmNonCTF — un film que le discriminant n'a pas reconnu ne publie AUCUN
// drapeau, et sa couverture dit pourquoi.
func TestFlagCarriesFilmNonCTF(t *testing.T) {
	scan := FlagCarryScan{
		Scanned: true,
		Signals: objectiveevents.FlagFilmSignals{Bursts: 2, Captures: 6, Steals: 994, Grabs: 1470},
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
		},
		Identity: map[int]string{12: "aaa"},
	}
	got, cov := buildFlagCarries(scan, flagTestCtx(nil, nil, 100))
	if got != nil {
		t.Errorf("%d drapeaux publies sur un film non-CTF", len(got))
	}
	if cov == nil || cov.FlagFilm || cov.Openings != 0 {
		t.Errorf("couverture %+v : le film ne doit pas etre reconnu CTF", cov)
	}
}

// TestFlagCarriesSansBalayage — rien n'a ete lu : le calque est absent, pas vide.
func TestFlagCarriesSansBalayage(t *testing.T) {
	got, cov := buildFlagCarries(FlagCarryScan{}, flagTestCtx(nil, nil, 100))
	if got != nil || cov != nil {
		t.Errorf("calque %v / couverture %v : les deux doivent etre absents", got, cov)
	}
}

// TestFlagCarriesMarqueurConfirme — le CONTROLE independant : une image-cle qui porte le marqueur
// sur le slot de bipede du porteur confirme le portage ; une image-cle sans marqueur ne le
// confirme pas, mais compte au denominateur.
func TestFlagCarriesMarqueurConfirme(t *testing.T) {
	tracks := []Track{flagTestTrack(10, "7", 0, 99, 30, 40), flagTestTrack(12, "8", 0, 99, 60, 60)}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 5000, Slot: 14, Stat: objectiveevents.StatFlagSteals},
		},
		Identity: map[int]string{12: "7", 14: "8"},
		Spawns:   []FlagSpawn{{Team: 0}, {Team: 1, X: 100, Y: 100}},
		Marks: filmdec.CarrierMarkScan{
			KeyframeUS: []uint64{2_000_000, 6_000_000},
			Marks:      []filmdec.CarrierMark{{TimestampUS: 2_000_000, Slot: 10}},
		},
	}
	ctx := flagTestCtx(tracks, nil, 100)
	ctx.slotXUID = map[uint32]uint64{10: 7, 12: 8}
	_, cov := buildFlagCarries(scan, ctx)
	if cov.MarkerObserved != 2 {
		t.Errorf("%d portages avec image-cle, attendu 2", cov.MarkerObserved)
	}
	if cov.MarkerConfirmed != 1 {
		t.Errorf("%d portages confirmes par le marqueur, attendu 1", cov.MarkerConfirmed)
	}
}

// flagOfTeam rend le drapeau d'une equipe, ou echoue.
func flagOfTeam(t *testing.T, carries []FlagCarry, team int) FlagCarry {
	t.Helper()
	for _, c := range carries {
		if c.Team == team {
			return c
		}
	}
	t.Fatalf("aucun drapeau d'equipe %d parmi %d publies", team, len(carries))
	return FlagCarry{}
}

// assertFlagStates compare la suite d'etats d'un drapeau a celle attendue.
func assertFlagStates(t *testing.T, f FlagCarry, want []string) {
	t.Helper()
	got := make([]string, 0, len(f.Spans))
	for _, s := range f.Spans {
		got = append(got, s.State)
	}
	if len(got) != len(want) {
		t.Fatalf("etats %v, attendu %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("etats %v, attendu %v", got, want)
		}
	}
	for i := 1; i < len(f.Spans); i++ {
		if f.Spans[i].T0 != f.Spans[i-1].T1+1 {
			t.Errorf("trou ou recouvrement entre les spans %d et %d : [%d,%d] puis [%d,%d]",
				i-1, i, f.Spans[i-1].T0, f.Spans[i-1].T1, f.Spans[i].T0, f.Spans[i].T1)
		}
	}
}
