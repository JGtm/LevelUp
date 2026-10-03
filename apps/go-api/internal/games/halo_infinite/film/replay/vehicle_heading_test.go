package replay

import (
	"math"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// vehicle_heading_test.go — LA PRESEANCE DES DEUX SOURCES DU CAP (lot 5.4.3).
//
// Les huit fixtures d entrees ne couvrent PAS ce chemin : elles sont rejouees depuis des
// positions enregistrees sur des mini-bobines, qui ne portent aucun delta `ti=40` et donc aucun
// roulis. Sans ces tests, le cap lu dans le film n aurait aucun filet.

// posAvecRoulis fabrique une position de vehicule dont le film ecrit l orientation : chassis A
// PLAT (la porte de direction posee vaut `(0,0,1)`), donc le cap au sol EGALE l angle de roulis.
func posAvecRoulis(mode uint8, rollRaw uint32) grammar.BipedPosition {
	var p grammar.BipedPosition
	p.HasRoll, p.RollRaw, p.FwdMode, p.AimDefault = true, rollRaw, mode, true
	return p
}

// TestCapVientDuFilmSurLeModePublie : sur le mode prouve, le cap sort du film — et il vaut
// l angle de roulis, puisque le chassis est a plat.
func TestCapVientDuFilmSurLeModePublie(t *testing.T) {
	for _, raw := range []uint32{0, 64, 128, 200, (1 << 30) - 1} {
		p := posAvecRoulis(grammar.FwdUpModeConfig, raw)
		h, ok := vehicleHeadingOf(p)
		if !ok {
			t.Fatalf("roulis %d : aucun cap rendu alors que le film l ecrit", raw)
		}
		theta := float64(grammar.RollAngleFromRaw(raw, grammar.FwdUpRollBits(grammar.FwdUpModeConfig)))
		attendu := headingDegOf(math.Cos(theta), math.Sin(theta))
		if ecart := math.Abs(float64(h - attendu)); ecart > 0.01 && math.Abs(ecart-360) > 0.01 {
			t.Fatalf("roulis %d : cap %.3f, attendu %.3f", raw, h, attendu)
		}
	}
}

// TestCapDuFilmNeDependPasDeLaVitesse — LE GAIN DU LOT, en un test : a l arret, en marche
// arriere, le cap reste celui du film. La velocite ne commande plus que le repli.
func TestCapDuFilmNeDependPasDeLaVitesse(t *testing.T) {
	p := posAvecRoulis(grammar.FwdUpModeConfig, 128)
	sansVitesse, ok := vehicleHeadingOf(p)
	if !ok {
		t.Fatalf("aucun cap sans velocite")
	}
	// La MEME position, mais avec une velocite qui pointe a l oppose (marche arriere).
	p.HasVel, p.VelRaw, p.VelScale = true, 0, 0
	avecVitesse, ok := vehicleHeadingOf(p)
	if !ok {
		t.Fatalf("aucun cap avec velocite")
	}
	if sansVitesse != avecVitesse {
		t.Fatalf("le cap du film a bouge avec la velocite : %.3f -> %.3f", sansVitesse, avecVitesse)
	}
}

// TestCapReplieSurLaVelociteHorsModePublie : sur le mode REFUTE et sur le chemin sans roulis
// absolu, rien n est publie depuis le film — le repli reprend, seuil compris.
func TestCapReplieSurLaVelociteHorsModePublie(t *testing.T) {
	// mode 0 : refute par la mesure, donc jamais publie depuis le film.
	if _, ok := vehicleFilmHeadingOf(posAvecRoulis(0, 128)); ok {
		t.Fatalf("le mode 0 ne doit PAS servir de cap : il est refute (mediane 95,5 deg, temoin 94,5)")
	}
	// chemin « delta » : aucun angle absolu ecrit.
	var sansRoulis grammar.BipedPosition
	sansRoulis.HasAim, sansRoulis.AimRaw, sansRoulis.FwdMode = true, 1234, 0
	if _, ok := vehicleFilmHeadingOf(sansRoulis); ok {
		t.Fatalf("sans roulis absolu, aucun cap ne peut etre reconstruit")
	}
	// Et sans velocite non plus : aucun cap du tout, l appelant reportera le dernier connu.
	if _, ok := vehicleHeadingOf(sansRoulis); ok {
		t.Fatalf("ni film ni velocite : aucun cap ne doit etre rendu")
	}
}
