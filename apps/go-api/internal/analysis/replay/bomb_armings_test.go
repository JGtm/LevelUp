package replay

// bomb_armings_test.go — LA RÈGLE D'ASSEMBLAGE de l'armement, testée à sec (aucun film) :
// déduplication de paire, confrontation locale tout-ou-rien, fenêtre de frames, mèche publiée.
// La chaîne complète sur films réels vit dans assaut_armement_gate_test.go (garde ASSAUT_CACHE).

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// baMontee fabrique les lectures d'UNE montée contiguë qui ARME : n échantillons espacés de
// 100 ms, quanta croissants finissant au QUANTUM PLEIN (254) — la définition mesurée de
// « bombe armée ».
func baMontee(slot uint32, startMS int32, n int) []filmdec.NavpointRadialRead {
	return baMonteeVers(slot, startMS, n, 254)
}

// baMonteeVers fabrique une montée contiguë finissant au quantum qEnd (pas de 16 quanta).
func baMonteeVers(slot uint32, startMS int32, n int, qEnd uint8) []filmdec.NavpointRadialRead {
	out := make([]filmdec.NavpointRadialRead, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, filmdec.NavpointRadialRead{
			Slot: slot, TMS: startMS + int32(i)*100, Q: qEnd - uint8(16*(n-1-i)),
		})
	}
	return out
}

// horloge : grille de 100 ms, axe large, origine zéro — les tests jugent les ms.
func baClock() scoreClock { return scoreClock{intervalMS: 100, frames: 1 << 20} }

func TestBuildBombArmingsDedupliqueLaPaire(t *testing.T) {
	// Deux navpoints (paire +12) répliquent la MÊME montée, fins à 40 ms d'écart.
	reads := append(baMontee(30, 10_000, 5), baMontee(42, 10_040, 5)...)
	armings, cov := buildBombArmings(reads, nil, baClock())
	if cov.Rises != 2 || cov.Armed != 1 || cov.PairMerged != 1 {
		t.Fatalf("paire non fondue : rises=%d armed=%d merged=%d", cov.Rises, cov.Armed, cov.PairMerged)
	}
	if len(armings) != 1 {
		t.Fatalf("attendu 1 armement publie, obtenu %d", len(armings))
	}
	a := armings[0]
	// Début retenu = le plus tôt des deux ; fin = la plus tardive (le miroir tardif date la mèche).
	if a.StartMS != 10_000 || a.TimeMS != 10_440 {
		t.Errorf("bornes fondues fausses : start=%d (attendu 10000), armed=%d (attendu 10440)",
			a.StartMS, a.TimeMS)
	}
	if a.FuseMS != BombFuseMS {
		t.Errorf("meche publiee %d, attendu %d", a.FuseMS, BombFuseMS)
	}
	if a.T != a.TimeMS/100 || a.StartT != a.StartMS/100 {
		t.Errorf("frames fausses : t=%d startT=%d", a.T, a.StartT)
	}
}

func TestBuildBombArmingsEcarteLesMonteesSousLePlein(t *testing.T) {
	// Une recharge de marqueur (plafond mesuré 253) et un hold relâché (198) : aucune n'arme.
	reads := append(baMonteeVers(30, 10_000, 5, 253), baMonteeVers(30, 30_000, 4, 198)...)
	armings, cov := buildBombArmings(reads, nil, baClock())
	if len(armings) != 0 || cov.Rises != 2 || cov.BelowFull != 2 || cov.Armed != 0 {
		t.Fatalf("montees sous le plein publiees : publies=%d rises=%d belowFull=%d armed=%d",
			len(armings), cov.Rises, cov.BelowFull, cov.Armed)
	}
}

func TestBuildBombArmingsGardeDeuxArmementsDistincts(t *testing.T) {
	// Deux armements séparés de 20 s : jamais fondus (le hold + la mèche les séparent d'au
	// moins ~6 s dans le jeu réel).
	reads := append(baMontee(30, 10_000, 5), baMontee(30, 30_000, 5)...)
	_, cov := buildBombArmings(reads, nil, baClock())
	if cov.Armed != 2 || cov.PairMerged != 0 {
		t.Fatalf("armements distincts fondus a tort : armed=%d merged=%d", cov.Armed, cov.PairMerged)
	}
}

func TestBuildBombArmingsConfrontationRetientToutOuRien(t *testing.T) {
	// Un armement à 10 400 ; une explosion cohérente (10 400 + 4 930) et une orpheline.
	reads := baMontee(30, 10_000, 5)
	coherente, orpheline := 10_400+BombFuseMS, 60_000
	armings, cov := buildBombArmings(reads, []int{coherente, orpheline}, baClock())
	if !cov.Suppressed || len(armings) != 0 {
		t.Fatalf("explosion orpheline non retenue : suppressed=%v publies=%d (couvertes %d/%d)",
			cov.Suppressed, len(armings), cov.DetonationsCovered, cov.Detonations)
	}
	// La même sans l'orpheline publie.
	armings, cov = buildBombArmings(reads, []int{coherente}, baClock())
	if cov.Suppressed || len(armings) != 1 || cov.DetonationsCovered != 1 {
		t.Fatalf("armement coherent non publie : suppressed=%v publies=%d couvertes=%d",
			cov.Suppressed, len(armings), cov.DetonationsCovered)
	}
}

func TestBuildBombArmingsHorsFenetreCompte(t *testing.T) {
	// Un axe de 5 frames (500 ms) : l'armement à 10,4 s tombe hors fenêtre, compté, non publié.
	reads := baMontee(30, 10_000, 5)
	armings, cov := buildBombArmings(reads, nil, scoreClock{intervalMS: 100, frames: 5})
	if len(armings) != 0 || cov.OutOfWindow != 1 || cov.Published != 0 {
		t.Fatalf("hors fenetre mal compte : publies=%d outOfWindow=%d", len(armings), cov.OutOfWindow)
	}
}

func TestBuildBombArmingsSansLectureResteVide(t *testing.T) {
	armings, cov := buildBombArmings(nil, []int{10_000}, baClock())
	// Aucune lecture : rien à publier, et l'explosion non couverte retient le calque — le
	// document dit POURQUOI (suppressed) au lieu d'un silence ambigu.
	if len(armings) != 0 || !cov.Suppressed {
		t.Fatalf("calque non vide sans lecture : publies=%d suppressed=%v", len(armings), cov.Suppressed)
	}
}
