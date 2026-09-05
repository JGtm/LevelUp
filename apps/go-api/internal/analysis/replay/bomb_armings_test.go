package replay

// bomb_armings_test.go — LA RÈGLE D'ASSEMBLAGE de l'armement, testée à sec (aucun film) :
// segments, quantum plein, déduplication de paire, confrontation locale tout-ou-rien, MÈCHE
// MESURÉE et pauses de désarmement, fenêtre de frames. La chaîne complète sur films réels vit
// dans assaut_armement_gate_test.go (garde ASSAUT_CACHE).

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// baMontee fabrique les lectures d'UN segment qui ARME : n échantillons espacés de 100 ms,
// quanta croissants finissant au QUANTUM PLEIN (254) — la définition mesurée de « bombe
// armée » (le segment finit à son sommet, et ce sommet est le plein).
func baMontee(slot uint32, startMS int32, n int) []filmdec.NavpointRadialRead {
	return baMonteeVers(slot, startMS, n, 254)
}

// baMonteeVers fabrique un segment montant finissant au quantum qEnd (pas de 16 quanta).
func baMonteeVers(slot uint32, startMS int32, n int, qEnd uint8) []filmdec.NavpointRadialRead {
	out := make([]filmdec.NavpointRadialRead, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, filmdec.NavpointRadialRead{
			Slot: slot, TMS: startMS + int32(i)*100, Q: qEnd - uint8(16*(n-1-i)),
		})
	}
	return out
}

// baTenue fabrique une TENUE DE DÉSARMEMENT : segment descendant de q0, 2 quanta par 100 ms
// (20 quanta/s — au milieu des 14-26 mesurés, très en dessous des 138 d'une chute
// d'explosion). durMS fixe sa durée, donc la pause qu'elle retranche à la mèche.
func baTenue(slot uint32, startMS int32, durMS int32, q0 uint8) []filmdec.NavpointRadialRead {
	n := int(durMS/100) + 1
	out := make([]filmdec.NavpointRadialRead, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, filmdec.NavpointRadialRead{
			Slot: slot, TMS: startMS + int32(i)*100, Q: q0 - uint8(2*i),
		})
	}
	return out
}

// horloge : grille de 100 ms, axe large, origine zéro — les tests jugent les ms.
func baClock() scoreClock { return scoreClock{intervalMS: 100, frames: 1 << 20} }

func TestBuildBombArmingsDedupliqueLaPaire(t *testing.T) {
	// Deux navpoints (paire +12) répliquent le MÊME armement, fins à 40 ms d'écart.
	reads := append(baMontee(30, 10_000, 5), baMontee(42, 10_040, 5)...)
	armings, cov, v := buildBombArmings(reads, nil, baClock())
	if cov.Rises != 2 || cov.Armed != 1 || cov.PairMerged != 1 {
		t.Fatalf("paire non fondue : segments=%d armed=%d merged=%d", cov.Rises, cov.Armed, cov.PairMerged)
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
	// AUCUNE explosion : rien à mesurer, la mèche publiée est la référence DÉDUITE.
	if a.FuseMS != BombFuseMS || v.Measured || cov.Detonations != 0 {
		t.Errorf("meche publiee %d (mesuree=%v, explosions=%d), attendu la reference %d deduite",
			a.FuseMS, v.Measured, cov.Detonations, BombFuseMS)
	}
	if a.T != a.TimeMS/100 || a.StartT != a.StartMS/100 {
		t.Errorf("frames fausses : t=%d startT=%d", a.T, a.StartT)
	}
}

func TestBuildBombArmingsEcarteLesSegmentsSousLePlein(t *testing.T) {
	// Une recharge de marqueur (plafond mesuré 253) et un hold relâché (198) : aucun n'arme.
	reads := append(baMonteeVers(30, 10_000, 5, 253), baMonteeVers(30, 30_000, 4, 198)...)
	armings, cov, _ := buildBombArmings(reads, nil, baClock())
	if len(armings) != 0 || cov.Rises != 2 || cov.BelowFull != 2 || cov.Armed != 0 {
		t.Fatalf("segments sous le plein publies : publies=%d segments=%d belowFull=%d armed=%d",
			len(armings), cov.Rises, cov.BelowFull, cov.Armed)
	}
}

// TestBuildBombArmingsEcarteLeCycleDeRecharge est ce que la lecture SEGMENT apporte face à la
// lecture MONTÉE : le cycle complet du marqueur (130 -> 254 -> 127, sans trou) finit à son
// MINIMUM. Découpé en montées, sa moitié montante passait pour un armement plein ; comme
// segment, il sort de lui-même — et il ne devient pas une pause non plus (il remonte
// au-dessus de son départ).
func TestBuildBombArmingsEcarteLeCycleDeRecharge(t *testing.T) {
	reads := baMontee(30, 10_000, 9) // 126 -> 254
	for i := 1; i <= 8; i++ {        // redescente immédiate, même segment : 254 -> 126
		reads = append(reads, filmdec.NavpointRadialRead{
			Slot: 30, TMS: 10_800 + int32(i)*100, Q: uint8(254 - 16*i),
		})
	}
	armings, cov, _ := buildBombArmings(reads, nil, baClock())
	if len(armings) != 0 || cov.Rises != 1 || cov.Armed != 0 || cov.BelowFull != 0 {
		t.Fatalf("cycle de recharge pris pour un armement : publies=%d segments=%d armed=%d belowFull=%d",
			len(armings), cov.Rises, cov.Armed, cov.BelowFull)
	}
}

func TestBuildBombArmingsGardeDeuxArmementsDistincts(t *testing.T) {
	// Deux armements séparés de 20 s : jamais fondus (le hold + la mèche les séparent d'au
	// moins ~6 s dans le jeu réel).
	reads := append(baMontee(30, 10_000, 5), baMontee(30, 30_000, 5)...)
	_, cov, _ := buildBombArmings(reads, nil, baClock())
	if cov.Armed != 2 || cov.PairMerged != 0 {
		t.Fatalf("armements distincts fondus a tort : armed=%d merged=%d", cov.Armed, cov.PairMerged)
	}
}

func TestBuildBombArmingsConfrontationRetientToutOuRien(t *testing.T) {
	// Un armement à 10 400 ; une explosion cohérente et une orpheline (hors fenêtre de sens).
	reads := baMontee(30, 10_000, 5)
	coherente, orpheline := 10_400+BombFuseMS, 10_400+BombFuseSenseWindowMS+1_000
	armings, cov, _ := buildBombArmings(reads, []int{coherente, orpheline}, baClock())
	if !cov.Suppressed || len(armings) != 0 {
		t.Fatalf("explosion orpheline non retenue : suppressed=%v publies=%d (couvertes %d/%d)",
			cov.Suppressed, len(armings), cov.DetonationsCovered, cov.Detonations)
	}
	// La même sans l'orpheline publie.
	armings, cov, v := buildBombArmings(reads, []int{coherente}, baClock())
	if cov.Suppressed || len(armings) != 1 || cov.DetonationsCovered != 1 {
		t.Fatalf("armement coherent non publie : suppressed=%v publies=%d couvertes=%d",
			cov.Suppressed, len(armings), cov.DetonationsCovered)
	}
	if !v.Measured || v.FuseMS != BombFuseMS {
		t.Errorf("meche mesuree %d (mesuree=%v), attendu %d", v.FuseMS, v.Measured, BombFuseMS)
	}
}

// TestBuildBombArmingsMecheMesureeEtPausable est LE PORTAGE de la lecture du 2026-09-01 : la
// mèche n'est pas supposée, elle se mesure sur le film ; et une TENUE DE DÉSARMEMENT du même
// slot la SUSPEND. Ici l'explosion tombe 20 200 ms après l'armement, dont 4 000 ms de tenue :
// la mèche mesurée est 16 200 ms — celle de One Bomb, sans qu'aucun nom de variante n'entre.
func TestBuildBombArmingsMecheMesureeEtPausable(t *testing.T) {
	reads := baMontee(30, 10_000, 5) // armé à 10 400
	reads = append(reads, baTenue(30, 20_000, 4_000, 251)...)
	explosion := 10_400 + 20_200
	armings, cov, v := buildBombArmings(reads, []int{explosion}, baClock())
	if cov.Suppressed || len(armings) != 1 {
		t.Fatalf("meche pausable non reconnue : suppressed=%v publies=%d couvertes=%d/%d",
			cov.Suppressed, len(armings), cov.DetonationsCovered, cov.Detonations)
	}
	if v.FuseMS != 16_200 || !v.Measured {
		t.Errorf("meche mesuree %d (mesuree=%v), attendu 16200 — la pause de 4 000 ms n'est pas deduite",
			v.FuseMS, v.Measured)
	}
	if armings[0].FuseMS != v.FuseMS {
		t.Errorf("meche publiee %d != meche mesuree %d", armings[0].FuseMS, v.FuseMS)
	}
	// CONTRE-ÉPREUVE : sans la tenue, le délai brut de 20 200 ms reste dans la fenêtre de
	// sens — le calque tient, mais la mèche publiée est fausse de la durée de la pause. C'est
	// exactement ce que la correction évite.
	_, _, sansPause := buildBombArmings(baMontee(30, 10_000, 5), []int{explosion}, baClock())
	if sansPause.FuseMS != 20_200 {
		t.Errorf("contre-epreuve : meche %d, attendu 20200 sans correction", sansPause.FuseMS)
	}
}

// TestBuildBombArmingsMechesQuiSeContredisent : la garde 2 ne juge pas que la couverture. Deux
// explosions dont les délais corrigés se contredisent (5 s et 20 s) disent que la lecture ne
// tient pas sur ce film — tout-ou-rien, comme une explosion orpheline.
func TestBuildBombArmingsMechesQuiSeContredisent(t *testing.T) {
	reads := append(baMontee(30, 10_000, 5), baMontee(30, 60_000, 5)...)
	armings, cov, v := buildBombArmings(reads, []int{10_400 + 5_000, 60_400 + 20_000}, baClock())
	if !cov.Suppressed || len(armings) != 0 || !v.Inconsistent {
		t.Fatalf("meches contradictoires publiees : suppressed=%v publies=%d incoherente=%v cv=%.3f",
			cov.Suppressed, len(armings), v.Inconsistent, v.CV)
	}
	if cov.DetonationsCovered != cov.Detonations {
		t.Errorf("les deux explosions sont COUVERTES (%d/%d) : c'est la dispersion qui retient, "+
			"et la couverture doit le dire", cov.DetonationsCovered, cov.Detonations)
	}
}

func TestBuildBombArmingsHorsFenetreCompte(t *testing.T) {
	// Un axe de 5 frames (500 ms) : l'armement à 10,4 s tombe hors fenêtre, compté, non publié.
	reads := baMontee(30, 10_000, 5)
	armings, cov, _ := buildBombArmings(reads, nil, scoreClock{intervalMS: 100, frames: 5})
	if len(armings) != 0 || cov.OutOfWindow != 1 || cov.Published != 0 {
		t.Fatalf("hors fenetre mal compte : publies=%d outOfWindow=%d", len(armings), cov.OutOfWindow)
	}
}

func TestBuildBombArmingsSansLectureResteVide(t *testing.T) {
	armings, cov, _ := buildBombArmings(nil, []int{10_000}, baClock())
	// Aucune lecture : rien à publier, et l'explosion non couverte retient le calque — le
	// document dit POURQUOI (suppressed) au lieu d'un silence ambigu.
	if len(armings) != 0 || !cov.Suppressed {
		t.Fatalf("calque non vide sans lecture : publies=%d suppressed=%v", len(armings), cov.Suppressed)
	}
}
