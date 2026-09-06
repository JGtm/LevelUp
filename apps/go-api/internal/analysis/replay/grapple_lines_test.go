package replay

// grapple_lines_test.go — les BRANCHES LIMITES de l'assemblage des tractions de grappin.
// Le golden (renderGrapple) verrouille le comportement sur les données réelles du film de
// référence ; ces tests-ci verrouillent ce que le réel n'exerce pas à coup sûr :
// l'appariement (raté compté, accroche orpheline), la fenêtre (clamp à la vie, mort à
// l'accroche), et l'arrivée mesurée (argmin sur la trajectoire, borne de recherche).

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// grappleEntry : une carte de test aux largeurs 10/10/10 et bornes [0, 102.4] — le pas de
// déquantification vaut exactement 0,1 u, les quanta se lisent donc en décimètres.
func grappleEntry() filmdec.MapQuantEntry {
	return filmdec.MapQuantEntry{
		Min:        [3]float32{0, 0, 0},
		Max:        [3]float32{102.4, 102.4, 102.4},
		AxisWidths: [3]uint{10, 10, 10},
	}
}

// grappleTrack : une vie publiée qui S'APPROCHE de l'ancre (100, 100) frame après frame
// puis s'en éloigne — l'arrivée mesurée est au point le plus proche.
func grappleTrack(slot uint32) Track {
	return Track{
		Slot:       slot,
		StartFrame: 0,
		EndFrame:   40,
		Points: []Point{
			{T: 0, X: 10, Y: 10}, {T: 5, X: 40, Y: 40}, {T: 10, X: 80, Y: 80},
			{T: 15, X: 99, Y: 99}, {T: 20, X: 60, Y: 60}, {T: 40, X: 10, Y: 10},
		},
	}
}

// read fabrique une lecture : ts en microsecondes sur une grille origin=0, step=100ms ;
// q1000 = quantum 1000 -> 100,05 u par axe (l'ancre du scénario).
func grappleRead(slot uint32, tsUS uint64, heavy bool) filmdec.GrappleRead {
	return filmdec.GrappleRead{
		Slot: slot, TimestampUS: tsUS, Heavy: heavy, PosQ: [3]uint32{1000, 1000, 0},
	}
}

const grappleStep = uint64(100_000) // 100 ms par frame, origine 0

func TestBuildGrappleLines_PairsFireToAttachAndMeasuresArrival(t *testing.T) {
	tracks := []Track{grappleTrack(5)}
	reads := []filmdec.GrappleRead{
		grappleRead(5, 500_000, false),   // tir à la frame 5
		grappleRead(5, 650_000, true),    // accroche 0,15 s après (frame 6)
		grappleRead(5, 3_000_000, false), // tir SANS accroche : un raté
	}
	lines, cov := buildGrappleLines(reads, grappleEntry(), 0, grappleStep, tracks)
	if len(lines) != 1 {
		t.Fatalf("%d traction(s), attendu 1 — l'appariement a bougé (cov=%+v)", len(lines), cov)
	}
	l := lines[0]
	if l.Slot != 5 || l.T0 != 5 {
		t.Errorf("traction slot=%d t0=%d, attendu slot=5 t0=5 (la fenêtre s'ouvre au TIR)", l.Slot, l.T0)
	}
	// L'arrivée : le point de la trajectoire le plus proche de (100,05, 100,05) est T=15.
	if l.T1 != 15 {
		t.Errorf("t1=%d, attendu 15 — l'arrivée n'est plus l'argmin de la distance à l'ancre", l.T1)
	}
	if l.AX < 100 || l.AX > 100.1 || l.AY < 100 || l.AY > 100.1 {
		t.Errorf("ancre (%v, %v), attendu ~(100,05, 100,05) — la déquantification a bougé", l.AX, l.AY)
	}
	if cov.Pulls != 1 || cov.PullLives != 1 || cov.UnpairedFires != 1 ||
		cov.LightReads != 2 || cov.HeavyReads != 1 {
		t.Errorf("couverture %+v : attendu 1 traction / 1 vie / 1 raté / 2 tirs / 1 accroche", *cov)
	}
}

func TestBuildGrappleLines_AttachWithoutFireOpensAtAttach(t *testing.T) {
	tracks := []Track{grappleTrack(5)}
	reads := []filmdec.GrappleRead{grappleRead(5, 800_000, true)} // accroche seule (frame 8)
	lines, _ := buildGrappleLines(reads, grappleEntry(), 0, grappleStep, tracks)
	if len(lines) != 1 || lines[0].T0 != 8 {
		t.Fatalf("lines=%v : une accroche sans tir lu doit ouvrir la fenêtre à l'ACCROCHE, "+
			"jamais plus tôt (on ne recule pas un début non lu)", lines)
	}
}

func TestBuildGrappleLines_StaleFireIsNotPaired(t *testing.T) {
	tracks := []Track{grappleTrack(5)}
	reads := []filmdec.GrappleRead{
		grappleRead(5, 100_000, false), // tir à la frame 1...
		grappleRead(5, 800_000, true),  // ... accroche 0,7 s après : PAS une paire (> 0,5 s)
	}
	lines, cov := buildGrappleLines(reads, grappleEntry(), 0, grappleStep, tracks)
	if len(lines) != 1 || lines[0].T0 != 8 {
		t.Fatalf("lines=%v : un tir trop vieux ne doit pas ouvrir la fenêtre", lines)
	}
	if cov.UnpairedFires != 1 {
		t.Errorf("ratés=%d, attendu 1 (le tir trop vieux est compté)", cov.UnpairedFires)
	}
}

func TestBuildGrappleLines_UnpublishedLifeAndDeathAtAttachDrawNothing(t *testing.T) {
	tracks := []Track{grappleTrack(5)}
	reads := []filmdec.GrappleRead{
		grappleRead(99, 650_000, true),  // vie non publiée : aucune fiche
		grappleRead(5, 4_000_000, true), // accroche à la frame 40 = fin de vie : fenêtre vide
	}
	lines, cov := buildGrappleLines(reads, grappleEntry(), 0, grappleStep, tracks)
	if len(lines) != 0 {
		t.Fatalf("lines=%v : rien ne doit se tracer (vie non publiée, mort à l'accroche)", lines)
	}
	if cov.HeavyReads != 2 {
		t.Errorf("accroches=%d, attendu 2 — les lectures se comptent même non tracées", cov.HeavyReads)
	}
}

// TestBuildGrappleLines_UneTractionDUneVieAnterieureEstPubliee — LA RÉGRESSION DU BALAYAGE DU
// PARC, FIGÉE AVEC SA PROVENANCE (instruction des régressions, candidate 2, 2026-09-06).
//
// LE CAS RÉEL. Match `879a4dba` (Fortitude, schéma 34 au parc) : 32 tirs, **23 accroches lues
// dans le film**, 23 tractions publiées ; dès `48cf4905d` (schéma 36, « une track = une vie »)
// le compte tombe à **15**, alors que `heavyReads` reste à 23 — la lecture n'a pas bougé, la
// PUBLICATION a jeté. Les 8 perdues sont toutes dans une vie qui n'est pas la dernière de son
// slot (543, 584 x4, 587, 631 x2) : la map slot -> track n'en retenait qu'une, la traction se
// voyait ramenée au début de la dernière vie, et `t1 <= t0` la faisait disparaître.
//
// CE QUE LE TEST VERROUILLE : la traction est posée sur LA VIE QUI COUVRE SON ACCROCHE, et les
// deux vies du slot portent chacune la leur.
func TestBuildGrappleLines_UneTractionDUneVieAnterieureEstPubliee(t *testing.T) {
	// Le slot 5 porte deux vies disjointes ; une accroche tombe dans chacune.
	premiere := grappleTrack(5)
	seconde := Track{
		Slot: 5, StartFrame: 100, EndFrame: 140,
		Points: []Point{{T: 100, X: 10, Y: 10}, {T: 115, X: 99, Y: 99}, {T: 140, X: 10, Y: 10}},
	}
	tracks := []Track{premiere, seconde}
	reads := []filmdec.GrappleRead{
		grappleRead(5, 500_000, false), grappleRead(5, 650_000, true), // vie 1 : frames 5 / 6
		grappleRead(5, 10_500_000, false), grappleRead(5, 10_650_000, true), // vie 2 : frames 105 / 106
	}
	lines, cov := buildGrappleLines(reads, grappleEntry(), 0, grappleStep, tracks)
	if len(lines) != 2 {
		t.Fatalf("%d traction(s), attendu 2 — celle de la vie ANTÉRIEURE a été jetée (cov=%+v)",
			len(lines), cov)
	}
	if lines[0].T0 != 5 || lines[0].T1 != 15 {
		t.Errorf("traction de la première vie [%d..%d], attendu [5..15]", lines[0].T0, lines[0].T1)
	}
	if lines[1].T0 != 105 || lines[1].T1 != 115 {
		t.Errorf("traction de la seconde vie [%d..%d], attendu [105..115]", lines[1].T0, lines[1].T1)
	}
	// Deux vies distinctes du même slot : le compteur en voit DEUX, là où la clé par slot n'en
	// voyait qu'une.
	if cov.Pulls != 2 || cov.PullLives != 2 {
		t.Errorf("couverture %+v, attendu pulls=2 pullLives=2", cov)
	}
}
