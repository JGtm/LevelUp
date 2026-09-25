package grammar

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// default_state_ti41_test.go — l etat par defaut du projectile (`FUN_1408efb58`, lot M4b), branche
// par branche, sur des flux synthetiques. La base ne le lisait pas (repli « 0 bit ») : la
// traversee d un record NEW de projectile partait trop tot.

// longueurMPPSurZeros rend la largeur que le bloc MPP (`FUN_14080cfe8`) consomme sur un flux nul :
// le test l ecrit en zeros et en deduit sa longueur par le lecteur lui-meme, pour n eprouver que
// ce qui SUIT le bloc.
func longueurMPPSurZeros() int {
	br := lecteurDInstrument(make([]byte, 64))
	consumeMultiplayerPropertiesBlock(br)
	return br.BitPos()
}

// TestEtatParDefautDuProjectileToutesPortesFermees : version absente, bloc MPP, puis chaque porte
// a 0 — 14 bits apres le bloc (R(1) +0x70, drapeau 2, porte de FUN_1408eff64, drapeau 4, porte
// des deux echelles, R(5), drapeaux 8, 0x10 et 0x20, porte de FUN_141fcf730).
func TestEtatParDefautDuProjectileToutesPortesFermees(t *testing.T) {
	mpp := longueurMPPSurZeros()
	w := &bitw{}
	w.put(0, 1) // version absente
	w.pad(mpp)
	w.put(0, 1)  // +0x70 absent
	w.put(0, 1)  // drapeau 2
	w.put(0, 1)  // FUN_1408eff64 : porte fermee
	w.put(0, 1)  // drapeau 4
	w.put(0, 1)  // echelles absentes
	w.put(17, 5) // +0x90
	w.put(0, 1)  // drapeau 8
	w.put(0, 1)  // drapeau 0x10
	w.put(0, 1)  // drapeau 0x20
	w.put(0, 1)  // FUN_141fcf730 absent
	w.put(0x2a, 8)
	br := lecteurDInstrument(append(w.buf, make([]byte, 64)...))
	consumeDefaultStateTI41(br, true)
	if got, attendu := br.BitPos(), 1+mpp+14; got != attendu {
		t.Fatalf("%d bits consommes, %d attendus", got, attendu)
	}
	if m := br.ReadBits(8); m != 0x2a {
		t.Fatalf("marqueur %#x, 0x2a attendu", m)
	}
}

// TestEtatParDefautDuProjectileToutesPortesOuvertes : version 3 (le drapeau 0x40 est lu), chaque
// porte ouverte, chaque lecteur de reference dans sa branche la plus courte. MUTATION : largeur
// des echelles 5 -> 4 (`largeurEchelleProjectile`), ou le drapeau 0x40 lu sans condition de
// version : ROUGE.
func TestEtatParDefautDuProjectileToutesPortesOuvertes(t *testing.T) {
	mpp := longueurMPPSurZeros()
	w := &bitw{}
	n := 0
	put := func(v uint64, k int) { w.put(v, k); n += k }
	put(1, 1)
	put(3, 8) // version 3
	w.pad(mpp)
	put(1, 1)
	put(9, 5) // +0x70
	put(1, 1) // drapeau 2
	put(0, 1) // FUN_1408f0ac4 : porte fermee
	put(1, 1) // FUN_1406d00ec : sentinelle, pas de R(2)
	put(1, 1) // FUN_1408eff64 : porte ouverte
	put(0, 2) // genre 0 : rien
	put(1, 1) // drapeau 4
	put(1, 1) // echelles
	put(3, 5)
	put(4, 5)
	put(17, 5) // +0x90
	put(1, 1)  // drapeau 8 : position de niveau 16, index absent
	put(1, 1)
	axes := profile.LargeursAxeParDefautDuBuild(profile.NiveauPositionDObjet)
	for _, a := range axes {
		put(0, int(a))
	}
	put(0, 19) // FUN_14076dc04
	put(0, 12) // FUN_1406d84b4 0xc
	put(1, 1)  // drapeau 0x10
	put(0, 1)  // FUN_1408f0ac4 categorie 0 : porte fermee
	put(1, 1)  // drapeau 0x20
	put(1, 1)  // c = 1
	put(2, 2)
	put(0, 3*13)
	put(1, 1)
	put(0xbeef, 16)
	put(1, 1) // drapeau 0x40 (version 3 > 2)
	put(1, 1) // FUN_141fcf730
	put(1, 1) // FUN_1407f2058 : sentinelle
	put(5, 7)
	put(1, 1)
	put(6, 4)
	w.put(0x2a, 8)
	br := lecteurDInstrument(append(w.buf, make([]byte, 64)...))
	consumeDefaultStateTI41(br, true)
	if got, attendu := br.BitPos(), n+mpp; got != attendu {
		t.Fatalf("%d bits consommes, %d attendus", got, attendu)
	}
	if m := br.ReadBits(8); m != 0x2a {
		t.Fatalf("marqueur %#x, 0x2a attendu", m)
	}
}

// TestLecteurDeReferenceDuProjectileSelonLeCinquiemeArgument : `FUN_1408eff64` lit l entier a
// largeur variable (record NEW, `param_5 = 1`) ou un R(32) (etat complet d image-cle, 0).
func TestLecteurDeReferenceDuProjectileSelonLeCinquiemeArgument(t *testing.T) {
	for _, c := range []struct {
		p       bool
		largeur int
	}{
		{true, 1 + 2 + int(varWidthBits(genreCibleCategorie2)) + 2},
		{false, 1 + 2 + 32},
	} {
		w := &bitw{}
		w.put(1, 1)
		w.put(genreCibleCategorie2, 2)
		w.pad(40)
		br := lecteurDInstrument(append(w.buf, make([]byte, 16)...))
		consume1408eff64(br, c.p)
		if got := br.BitPos(); got != c.largeur {
			t.Errorf("param_5 %v : %d bits, %d attendus", c.p, got, c.largeur)
		}
	}
}

// TestLeRecordNeufDuProjectileTraverseSonEtatParDefaut : [TraverseEntity] (le lecteur de record
// NEW) lit l etat par defaut du projectile avant la porte et le masque. La base sautait 0 bit :
// sa porte et son masque etaient lus dans la version et le bloc MPP — ROUGE.
func TestLeRecordNeufDuProjectileTraverseSonEtatParDefaut(t *testing.T) {
	archs := make([]Archetype, ProjectileTypeIndex+1)
	for i := range archs {
		archs[i] = Archetype{Index: i}
	}
	reg := &Registry{Archetypes: archs}
	mpp := longueurMPPSurZeros()
	w := &bitw{}
	w.put(ProjectileTypeIndex, 6)
	w.put(1, 1) // version posee...
	w.put(0xff, 8)
	w.pad(mpp)
	w.pad(5)     // +0x70, drapeau 2, FUN_1408eff64, drapeau 4, echelles : fermes
	w.put(17, 5) // +0x90
	w.pad(4)     // drapeaux 8, 0x10, 0x20 ; version > 2 : drapeau 0x40
	w.put(0, 1)  // FUN_141fcf730 absent
	w.put(0, 1)  // porte du record NEW
	w.put(0, 1)  // masque clairseme
	w.put(0, 3)  // aucun composant
	w.put(0x2a, 8)
	br := lecteurDInstrument(append(w.buf, make([]byte, 64)...))
	tr := TraverseEntity(br, reg, 0)
	attendu := 6 + 9 + mpp + 5 + 5 + 4 + 1 + 1 + 4
	if tr.DesyncAt != -1 || tr.EndBit != attendu {
		t.Fatalf("fin %d (desync %d), attendu %d", tr.EndBit, tr.DesyncAt, attendu)
	}
	if m := br.ReadBits(8); m != 0x2a {
		t.Fatalf("marqueur %#x, 0x2a attendu", m)
	}
}
