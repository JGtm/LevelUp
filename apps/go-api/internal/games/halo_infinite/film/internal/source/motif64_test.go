package source

// motif64_test.go — [PremierMotif64] CONTRE LE BALAYAGE NAIF, POSITION PAR POSITION.
//
// La reference lit chaque fenetre BIT A BIT (aucune lecture par mot, aucun decalage partage) et
// juge le motif par des comparaisons ecrites en clair. Tampons pseudo-aleatoires a graine fixee,
// dont des tampons a longues plages de 0x00 et de 0xFF (ce que la recherche d ancres rencontre :
// corps nuls, slots liberes) ; toutes les positions de depart et toutes les bornes autour des
// frontieres d octet et de fin de tampon.

import (
	"math/rand"
	"testing"
)

// refFenetre lit les 64 bits a la position p, un bit a la fois.
func refFenetre(d []byte, p int) uint64 {
	var w uint64
	for k := range 64 {
		q := p + k
		w = w<<1 | uint64(d[q>>3]>>(7-uint(q&7)))&1
	}
	return w
}

// refRetenue juge une fenetre en clair.
func refRetenue(m Motif64, w uint64) bool {
	if w&m.Nuls != 0 {
		return false
	}
	for _, u := range m.AuMoinsUn {
		if u != 0 && w&u == 0 {
			return false
		}
	}
	return true
}

// refPremierMotif64 est le balayage naif.
func refPremierMotif64(d []byte, from, limit int, m Motif64) int {
	for p := from; p < limit && p+64 <= len(d)*8; p++ {
		if refRetenue(m, refFenetre(d, p)) {
			return p
		}
	}
	return -1
}

// motifTampons : aleatoires, a plages nulles, a plages pleines, et melanges.
func motifTampons() [][]byte {
	rng := rand.New(rand.NewSource(0x6D6F_7469))
	var out [][]byte
	for _, n := range []int{0, 1, 7, 8, 9, 15, 16, 17, 23, 24, 31, 40, 64, 97} {
		for v := range 4 {
			d := make([]byte, n)
			for i := range d {
				switch v {
				case 0:
					d[i] = byte(rng.Intn(256))
				case 1: // plages nulles, parsemees
					if rng.Intn(6) == 0 {
						d[i] = byte(rng.Intn(256))
					}
				case 2: // plages pleines, parsemees
					d[i] = 0xFF
					if rng.Intn(6) == 0 {
						d[i] = byte(rng.Intn(256))
					}
				default: // petites valeurs : peu de bits a 1
					d[i] = byte(1 << uint(rng.Intn(8)) & rng.Intn(256))
				}
			}
			out = append(out, d)
		}
	}
	return out
}

// motifsDeTest : le motif de la recherche d ancres, des variantes de ses groupes, et des masques
// aleatoires peu exigeants (pour que les retenues soient frequentes).
func motifsDeTest() []Motif64 {
	rng := rand.New(rand.NewSource(0x4D36_3434))
	out := []Motif64{
		{Nuls: 0x3FFFE000_FFFFFFC0, AuMoinsUn: [2]uint64{0xC0000000_00000000, 0x3FFFFE00_00000000}},
		{Nuls: 0x3FFFE000_FFFFFFC0, AuMoinsUn: [2]uint64{0xC0000000_00000000}},
		{Nuls: 0x3FFFE000_FFFFFFC0, AuMoinsUn: [2]uint64{0, 0x3FFFE000_00000000}},
		{AuMoinsUn: [2]uint64{^uint64(0), 1}},
		{Nuls: ^uint64(0)},
		{},
	}
	for i := range 12 {
		out = append(out, Motif64{
			Nuls:      rng.Uint64() & rng.Uint64() & rng.Uint64(),
			AuMoinsUn: [2]uint64{rng.Uint64() * uint64(i%3), rng.Uint64() >> uint(rng.Intn(64))},
		})
	}
	return out
}

// TestPremierMotif64EgaleLeBalayageNaif : meme position rendue, pour tout depart et toute borne.
func TestPremierMotif64EgaleLeBalayageNaif(t *testing.T) {
	cas := 0
	for ti, d := range motifTampons() {
		nb := len(d) * 8
		for mi, m := range motifsDeTest() {
			for from := 0; from <= nb; from++ {
				for _, limit := range []int{from - 1, from, from + 1, from + 7, from + 8, from + 9,
					from + 63, from + 200, nb - 64, nb - 63, nb - 62, nb, nb + 5} {
					cas++
					got, want := PremierMotif64(d, from, limit, m), refPremierMotif64(d, from, limit, m)
					if got != want {
						t.Fatalf("tampon %d (%d octets), motif %d %+v, [%d, %d) : %d, attendu %d",
							ti, len(d), mi, m, from, limit, got, want)
					}
				}
			}
		}
	}
	if cas < 100_000 {
		t.Fatalf("%d cas seulement : le differentiel ne balaie plus rien", cas)
	}
}

// TestHuitEgaleHuitRetenues : le bloc sans branche juge chaque fenetre comme le jugement en clair.
func TestHuitEgaleHuitRetenues(t *testing.T) {
	rng := rand.New(rand.NewSource(0x0808))
	for _, m := range motifsDeTest() {
		for i := range 20_000 {
			w0, nb := rng.Uint64(), uint64(rng.Intn(256))
			switch i % 4 {
			case 1:
				w0 &= rng.Uint64() & rng.Uint64()
			case 2:
				w0 |= rng.Uint64() | rng.Uint64()
			}
			var want uint64
			for k := range uint(8) {
				w := w0 << k
				if k > 0 {
					w |= nb >> (8 - k)
				}
				if refRetenue(m, w) {
					want |= 1 << k
				}
			}
			if got := m.jugement().huit(w0, nb); got != want {
				t.Fatalf("motif %+v w0=%#x nb=%#x : %08b, attendu %08b", m, w0, nb, got, want)
			}
		}
	}
}
