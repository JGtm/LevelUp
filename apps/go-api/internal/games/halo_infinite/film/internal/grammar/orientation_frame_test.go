package grammar

import (
	"math"
	"testing"
)

// orientation_frame_test.go — LES INVARIANTS DE LA RECONSTRUCTION DE L AVANT.
//
// Aucun film, aucune bobine : ce sont les proprietes que l executable garantit par
// construction, et le predicat `FUN_140501798` les teste en jeu a chaque record. Si l un de ces
// tests tombe, le port de `FUN_1406d8678` a derive.

// TestAvantEstUnitaireEtPerpendiculaire porte le predicat `FUN_140501798` lui-meme :
// ||avant|| = 1, ||haut|| = 1 et avant.haut = 0, a 1,0e-3 pres (`DAT_143cd84bc`).
func TestAvantEstUnitaireEtPerpendiculaire(t *testing.T) {
	const tol = 1e-3
	hauts := [][3]float32{
		{0, 0, 1}, {0, 0, -1}, {1, 0, 0}, {0, 1, 0},
		{0.6, 0.8, 0}, {0.267, 0.535, 0.802}, {-0.577, 0.577, 0.577},
	}
	for _, up := range hauts {
		for raw := uint32(0); raw < 256; raw += 7 {
			roll := RollAngleFromRaw(raw, 8)
			f := ForwardFromUpRoll(up, roll)
			if n := float64(norm3(f)); math.Abs(n-1) > tol {
				t.Fatalf("haut %v, roulis %.4f : ||avant|| = %.6f, attendu 1", up, roll, n)
			}
			if d := math.Abs(float64(dot3(f, up))); d > tol {
				t.Fatalf("haut %v, roulis %.4f : avant.haut = %.6f, attendu 0", up, roll, d)
			}
		}
	}
}

// TestChassisAPlatLeCapEstLAngle — LA PROPRIETE QUI REND LA MESURE LISIBLE, et elle tombe du
// port, elle n est pas posee a la main : quand le haut vaut (0, 0, 1) — un vehicule a plat,
// c est-a-dire le defaut `DAT_143b8f860` et le cas dominant au sol — la base choisie est
// `(0,1,0) x haut = (1,0,0)`, et la rotation de Rodrigues autour de +Z d un angle `theta` rend
// exactement `(cos theta, sin theta, 0)`. LE CAP AU SOL EST DONC L ANGLE LUI-MEME.
//
// C est ce qui explique que le lot 5.2b.2 ne pouvait RIEN trouver : il mesurait l azimut du
// HAUT (indetermine a plat) et jetait l angle, qui EST le cap.
func TestChassisAPlatLeCapEstLAngle(t *testing.T) {
	up := [3]float32{0, 0, 1}
	for raw := uint32(0); raw < 256; raw++ {
		roll := RollAngleFromRaw(raw, 8)
		f := ForwardFromUpRoll(up, roll)
		got := math.Atan2(float64(f[1]), float64(f[0]))
		if ecart := math.Abs(math.Remainder(got-float64(roll), 2*math.Pi)); ecart > 1e-4 {
			t.Fatalf("roulis brut %d (%.6f rad) : cap reconstruit %.6f, ecart %.6f", raw, roll, got, ecart)
		}
		if math.Abs(float64(f[2])) > 1e-6 {
			t.Fatalf("roulis brut %d : avant hors du plan du sol, z = %.6f", raw, f[2])
		}
	}
}

// TestDequantRoulisEgaleLesConstantesRelues : `theta = raw * pi/128 - pi + pi/256` est la forme
// INLINEE dans FUN_140c5f9c8, avec ses trois constantes relues dans le binaire. Ce test fige
// l egalite avec `dequantMidpoint`, qui est la forme generique employee par le port.
func TestDequantRoulisEgaleLesConstantesRelues(t *testing.T) {
	const (
		pasHuitBits  = math.Pi / 128 // DAT_143cd891c = 0x3cc90fdb
		demiPasHuit  = math.Pi / 256 // DAT_143cd97a0 = 0x3c490fdb
		borneSuperio = math.Pi       // DAT_143cd8918 = 0x40490fdb
	)
	for raw := uint32(0); raw < 256; raw++ {
		attendu := float32(float64(raw)*pasHuitBits - borneSuperio + demiPasHuit)
		if got := RollAngleFromRaw(raw, 8); math.Abs(float64(got-attendu)) > 1e-6 {
			t.Fatalf("raw %d : RollAngleFromRaw = %.8f, constantes relues = %.8f", raw, got, attendu)
		}
	}
	// Les bornes : le premier quantum est a un demi-pas de -pi, le dernier a un demi-pas de +pi.
	if got := RollAngleFromRaw(0, 8); math.Abs(float64(got)-(-math.Pi+demiPasHuit)) > 1e-6 {
		t.Fatalf("premier quantum = %.8f", got)
	}
	if got := RollAngleFromRaw(255, 8); math.Abs(float64(got)-(math.Pi-demiPasHuit)) > 1e-6 {
		t.Fatalf("dernier quantum = %.8f", got)
	}
}

// TestBaseChoisieEstLaMoinsAlignee fige la regle de selection de l executable (et son ordre de
// produit vectoriel, qui fixe le SENS de l avant) : un haut presque aligne sur +X doit passer
// par la branche Y, et reciproquement.
func TestBaseChoisieEstLaMoinsAlignee(t *testing.T) {
	// haut ~ +X : |haut.X| > |haut.Y| -> branche Y -> v = Y x haut = (haut.z, 0, -haut.x)
	up := [3]float32{1, 0, 0}
	f := ForwardFromUpRoll(up, 0)
	if f[0] != 0 || f[1] != 0 || f[2] != -1 {
		t.Fatalf("haut +X, roulis 0 : avant = %v, attendu (0, 0, -1)", f)
	}
	// haut ~ +Y : |haut.X| < |haut.Y| -> branche X -> v = haut x X = (0, haut.z, -haut.y)
	up = [3]float32{0, 1, 0}
	f = ForwardFromUpRoll(up, 0)
	if f[0] != 0 || f[1] != 0 || f[2] != -1 {
		t.Fatalf("haut +Y, roulis 0 : avant = %v, attendu (0, 0, -1)", f)
	}
}
