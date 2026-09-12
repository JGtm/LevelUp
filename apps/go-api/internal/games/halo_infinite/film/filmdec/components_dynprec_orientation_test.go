package filmdec

import "testing"

// bitWriterMSB écrit des champs MSB-first, comme les lit BitReader. Local au test : il
// n'existe que pour fabriquer les chemins de FUN_140c5f7ec sans film.
type bitWriterMSB struct {
	buf []byte
	pos int
}

func (w *bitWriterMSB) put(v uint64, n uint) {
	for i := int(n) - 1; i >= 0; i-- {
		if w.pos>>3 >= len(w.buf) {
			w.buf = append(w.buf, 0)
		}
		if v>>uint(i)&1 == 1 {
			w.buf[w.pos>>3] |= 1 << (7 - uint(w.pos)&7)
		}
		w.pos++
	}
}

// TestFwdUpDynPrecBitCost fige le COÛT EN BITS de chaque chemin de
// `object-forward-and-up-dynamic-precision-component` (FUN_140c5f7ec). Chaque attendu cite
// la pièce de désassemblage dans components_dynprec_orientation.go ; ce test est le
// garde-rail qui empêche un « nettoyage » de déplacer une largeur en silence.
func TestFwdUpDynPrecBitCost(t *testing.T) {
	cases := []struct {
		name  string
		param uint32
		bits  []struct {
			v uint64
			n uint
		}
		want int
	}{
		// A=1 -> mode 2 : R(1) + deux vec3 bruts (2 x 96).
		{"A=1 keep (mode 2)", 1, []struct {
			v uint64
			n uint
		}{{1, 1}}, 1 + 192},
		// A=0, B=0 -> FUN_140c5fa84 : R(1)+R(1) + [gate R(1) = 0 -> R(19)] + R(8).
		{"A=0 B=0 direction presente", 1, []struct {
			v uint64
			n uint
		}{{0, 1}, {0, 1}, {0, 1}}, 2 + 1 + 19 + 8},
		// idem, gate interne = 1 : pas de R(19).
		{"A=0 B=0 direction absente", 1, []struct {
			v uint64
			n uint
		}{{0, 1}, {0, 1}, {1, 1}}, 2 + 1 + 8},
		// A=0, B=1 -> FUN_14076e744, g1=1 : queue R(1) seule (t=0).
		{"A=0 B=1 g1=1 t=0", 1, []struct {
			v uint64
			n uint
		}{{0, 1}, {1, 1}, {1, 1}, {0, 1}}, 2 + 1 + 1},
		// A=0, B=1 -> FUN_14076e744, g1=0 g2=0 : R(19) puis queue R(1)+R(4).
		{"A=0 B=1 g1=0 g2=0 t=1", 1, []struct {
			v uint64
			n uint
		}{{0, 1}, {1, 1}, {0, 1}, {0, 1}, {0, 19}, {1, 1}}, 2 + 2 + 19 + 1 + 4},
		// A=0, B=1 -> FUN_14076e744, g1=0 g2=1 : deux quartets puis queue R(1) (t=0).
		{"A=0 B=1 g1=0 g2=1 t=0", 1, []struct {
			v uint64
			n uint
		}{{0, 1}, {1, 1}, {0, 1}, {1, 1}, {0, 8}, {0, 1}}, 2 + 2 + 8 + 1},
		// param>=2 : le bit C est lu. C=0 -> chemin normal, un bit de plus qu'en param=1.
		{"param=2 C=0", 2, []struct {
			v uint64
			n uint
		}{{0, 1}, {0, 1}, {0, 1}, {1, 1}}, 3 + 1 + 8},
		// param>=2, C=1 -> mode 1 = FUN_142e29bac : R(1) g ; g=1 -> pas de R(30) ; puis R(30).
		{"param=2 C=1 g=1", 2, []struct {
			v uint64
			n uint
		}{{0, 1}, {0, 1}, {1, 1}, {1, 1}}, 3 + 1 + 30},
		// param>=2, C=1, g=0 : R(30) direction + R(30) scalaire.
		{"param=2 C=1 g=0", 2, []struct {
			v uint64
			n uint
		}{{0, 1}, {0, 1}, {1, 1}, {0, 1}}, 3 + 1 + 30 + 30},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := &bitWriterMSB{}
			for _, b := range c.bits {
				w.put(b.v, b.n)
			}
			for len(w.buf) < 64 { // marge : les chemins longs lisent au-delà des bits posés
				w.buf = append(w.buf, 0)
			}
			br := NewBitReader(w.buf)
			if !consumeObjectForwardAndUpDynPrec(br, c.param) {
				t.Fatalf("%s : le porteur rend non-porte", c.name)
			}
			if got := br.BitPos(); got != c.want {
				t.Errorf("%s : %d bits consommes, attendu %d", c.name, got, c.want)
			}
		})
	}
}

// TestAngVelDynPrecBitCost fige le coût d'i3 dyn.-préc. (FUN_140d87740) et, surtout, le
// SÉPARE de celui du composant sans « dynamic-precision » (FUN_140d70998 =
// consumeDynPrecVec3) : c'est cette confusion qui désynchronisait ti=40.
func TestAngVelDynPrecBitCost(t *testing.T) {
	cases := []struct {
		name     string
		bits     []struct{ v, n uint64 }
		wantDyn  int
		wantPlat int
	}{
		{"gate externe pose : copie brute", []struct{ v, n uint64 }{{1, 1}}, 1 + 96, 1},
		{"gate externe nul, vec3 absent", []struct{ v, n uint64 }{{0, 1}, {1, 1}}, 2, 1 + 19 + 8},
		{"gate externe nul, vec3 present", []struct{ v, n uint64 }{{0, 1}, {0, 1}}, 2 + 19 + 8, 1 + 19 + 8},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := &bitWriterMSB{}
			for _, b := range c.bits {
				w.put(b.v, uint(b.n))
			}
			for len(w.buf) < 32 {
				w.buf = append(w.buf, 0)
			}
			br := NewBitReader(w.buf)
			consumeObjectAngularVelocityDynPrec(br)
			if got := br.BitPos(); got != c.wantDyn {
				t.Errorf("dyn.-prec. : %d bits, attendu %d", got, c.wantDyn)
			}
			br2 := NewBitReader(w.buf)
			consumeDynPrecVec3(br2, angularMagBits, angularScaleBits)
			if got := br2.BitPos(); got != c.wantPlat {
				t.Errorf("sans dyn.-prec. : %d bits, attendu %d", got, c.wantPlat)
			}
		})
	}
}
