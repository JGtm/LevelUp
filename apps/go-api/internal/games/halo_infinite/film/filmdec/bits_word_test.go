package filmdec

// bits_word_test.go — TEST DIFFERENTIEL DES PRIMITIVES DE LECTURE DE BITS (decision D6 du
// plan `.ai/V7.5/PLAN_CUISSON_PERF.md`).
//
// LA METHODE. Les quatre primitives reecrites par mot (`BitReader.ReadBits`, `kfReadBits`,
// `readBitsAt`, `PeekBits`) sont opposees a une COPIE DE REFERENCE de leur implementation
// d'AVANT, recopiee ici et nulle part ailleurs — le code de production n'en garde aucune.
// Chaque cas est joue sur les deux et les resultats doivent coincider bit pour bit.
//
// LE DOMAINE COUVERT. Tampons pseudo-aleatoires a GRAINE FIXEE (donc rejouables), toutes les
// largeurs de 0 a 64, et des positions choisies autour de CHAQUE frontiere qui compte : bit,
// octet, mot de 64, et fin de tampon (avant, sur, apres). Les cas HORS TAMPON sont joues
// explicitement, et la semantique attendue y est celle de CHAQUE fonction, pas une
// semantique commune : `readBitsAt` PANIQUE, les trois autres rendent des zeros de bourrage.
// C'est la propriete que le lot 4 devait preserver, et c'est celle-ci qui la verrouille.

import (
	"fmt"
	"math/rand"
	"testing"
)

// --- Copies de reference (implementations d'AVANT le lot 4, oracles du differentiel) ---

// refReadBitsSeq est `BitReader.ReadBits` d'avant : lecture bit a bit, zero hors tampon,
// curseur avance de n quoi qu'il arrive.
func refReadBitsSeq(buf []byte, pos int, n uint) (uint64, int) {
	var r uint64
	for i := uint(0); i < n; i++ {
		var bit uint64
		if idx := pos >> 3; idx < len(buf) {
			bit = uint64(buf[idx]>>(7-(uint(pos)&7))) & 1
		}
		r = r<<1 | bit
		pos++
	}
	return r, pos
}

// refKfReadBits est `kfReadBits` d'avant.
func refKfReadBits(buf []byte, pos, n int) uint64 {
	var r uint64
	for i := 0; i < n; i++ {
		p := pos + i
		var bit uint64
		if idx := p >> 3; idx < len(buf) {
			bit = uint64(buf[idx]>>(7-uint(p&7))) & 1
		}
		r = r<<1 | bit
	}
	return r
}

// refReadBitsAt est `readBitsAt` d'avant : AUCUNE garde, indexation nue.
func refReadBitsAt(b []byte, pos, n int) uint32 {
	var v uint32
	for i := 0; i < n; i++ {
		p := pos + i
		v = v<<1 | uint32(b[p>>3]>>(7-uint(p&7))&1)
	}
	return v
}

// refPeekBits est `PeekBits` d'avant : zero des DEUX cotes du tampon.
func refPeekBits(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		p := bp + i
		if p < 0 || p>>3 >= len(d) {
			v <<= 1
			continue
		}
		v = (v << 1) | uint64((d[p>>3]>>uint(7-(p&7)))&1)
	}
	return v
}

// --- Materiel commun ---

// bitsFuzzBuffers rend des tampons pseudo-aleatoires a graine FIXEE, de tailles choisies
// pour couvrir le tampon plus court qu'un mot, le mot exact, et les tailles quelconques.
func bitsFuzzBuffers() [][]byte {
	rng := rand.New(rand.NewSource(0x5EED_1104))
	sizes := []int{0, 1, 2, 3, 7, 8, 9, 15, 16, 17, 23, 31, 64, 65}
	out := make([][]byte, 0, len(sizes))
	for _, n := range sizes {
		b := make([]byte, n)
		rng.Read(b)
		out = append(out, b)
	}
	return out
}

// bitsFuzzPositions rend les positions de bit a eprouver pour un tampon de `nBytes` octets :
// le debut, chaque frontiere d'octet et de mot des 24 premiers octets, et le voisinage
// immediat de la fin du tampon (dedans, dessus, dehors).
func bitsFuzzPositions(nBytes int) []int {
	total := nBytes * 8
	seen := map[int]bool{}
	var out []int
	add := func(p int) {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	for p := 0; p < 32 && p <= total+16; p++ {
		add(p)
	}
	for _, edge := range []int{8, 16, 56, 64, 72, 120, 128, 136, total} {
		for d := -2; d <= 2; d++ {
			if edge+d >= 0 {
				add(edge + d)
			}
		}
	}
	for _, p := range []int{total + 7, total + 8, total + 64, total + 65} {
		if p >= 0 {
			add(p)
		}
	}
	return out
}

// --- Les quatre differentiels ---

func TestReadBitsWordMatchesReference(t *testing.T) {
	for _, buf := range bitsFuzzBuffers() {
		for _, pos := range bitsFuzzPositions(len(buf)) {
			for n := uint(0); n <= 64; n++ {
				wantV, wantPos := refReadBitsSeq(buf, pos, n)
				br := NewBitReader(buf)
				br.SetBitPos(pos)
				gotV := br.ReadBits(n)
				if gotV != wantV || br.BitPos() != wantPos {
					t.Fatalf("ReadBits(len=%d, pos=%d, n=%d) = (%#x, %d), reference (%#x, %d)",
						len(buf), pos, n, gotV, br.BitPos(), wantV, wantPos)
				}
			}
		}
	}
}

// TestReadBitsWordWideMatchesReference couvre les largeurs > 64 : la valeur ne garde que les
// 64 DERNIERS bits lus et le curseur avance quand meme de n. Ce n'est pas un cas theorique —
// `consumeTrackFrameComponent` (components_batch7.go) lit une largeur tiree de 12 bits du
// flux, donc jusqu'a 4 095.
func TestReadBitsWordWideMatchesReference(t *testing.T) {
	for _, buf := range bitsFuzzBuffers() {
		for _, pos := range []int{0, 1, 7, 8, 63, 64, 65, len(buf)*8 - 1, len(buf) * 8} {
			if pos < 0 {
				continue
			}
			for _, n := range []uint{65, 66, 96, 127, 128, 129, 200} {
				wantV, wantPos := refReadBitsSeq(buf, pos, n)
				br := NewBitReader(buf)
				br.SetBitPos(pos)
				gotV := br.ReadBits(n)
				if gotV != wantV || br.BitPos() != wantPos {
					t.Fatalf("ReadBits large (len=%d, pos=%d, n=%d) = (%#x, %d), reference (%#x, %d)",
						len(buf), pos, n, gotV, br.BitPos(), wantV, wantPos)
				}
			}
		}
	}
}

func TestKfReadBitsWordMatchesReference(t *testing.T) {
	for _, buf := range bitsFuzzBuffers() {
		for _, pos := range bitsFuzzPositions(len(buf)) {
			for n := 0; n <= 64; n++ {
				if got, want := kfReadBits(buf, pos, n), refKfReadBits(buf, pos, n); got != want {
					t.Fatalf("kfReadBits(len=%d, pos=%d, n=%d) = %#x, reference %#x",
						len(buf), pos, n, got, want)
				}
			}
		}
	}
}

func TestPeekBitsWordMatchesReference(t *testing.T) {
	for _, buf := range bitsFuzzBuffers() {
		positions := bitsFuzzPositions(len(buf))
		// Depart NEGATIF : `PeekBits` promet le zero des DEUX cotes, et c'est la seule des
		// quatre primitives a le promettre.
		positions = append(positions, -1, -2, -7, -8, -9, -64, -65)
		for _, pos := range positions {
			for n := 0; n <= 64; n++ {
				if got, want := PeekBits(buf, pos, n), refPeekBits(buf, pos, n); got != want {
					t.Fatalf("PeekBits(len=%d, bp=%d, n=%d) = %#x, reference %#x",
						len(buf), pos, n, got, want)
				}
			}
		}
	}
}

// TestReadBitsAtWordMatchesReference oppose `readBitsAt` a sa reference SUR SON DOMAINE
// LEGAL (lecture entierement dans le tampon) : c'est le seul ou la reference rend une
// valeur au lieu de paniquer.
func TestReadBitsAtWordMatchesReference(t *testing.T) {
	for _, buf := range bitsFuzzBuffers() {
		total := len(buf) * 8
		for _, pos := range bitsFuzzPositions(len(buf)) {
			if pos < 0 || pos > total {
				continue
			}
			for n := 0; n <= 64 && pos+n <= total; n++ {
				if got, want := readBitsAt(buf, pos, n), refReadBitsAt(buf, pos, n); got != want {
					t.Fatalf("readBitsAt(len=%d, pos=%d, n=%d) = %#x, reference %#x",
						len(buf), pos, n, got, want)
				}
			}
		}
	}
}

// TestReadBitsAtPanicsOutOfBufferLikeReference verrouille la semantique HORS TAMPON de
// `readBitsAt` : elle panique, exactement la ou la reference panique. C'est la garantie
// explicitement demandee par D6 — la reecriture ne devait la changer ni dans un sens
// (avaler la panique) ni dans l'autre (paniquer plus tot).
func TestReadBitsAtPanicsOutOfBufferLikeReference(t *testing.T) {
	buf := make([]byte, 4)
	total := len(buf) * 8
	cases := []struct{ pos, n int }{
		{total - 1, 2}, {total, 1}, {total, 64}, {total + 1, 1}, {total + 64, 8},
		{-1, 1}, {-8, 1}, {-1, 64}, {0, 33}, // 33 bits sur 32 : deborde d'un bit
	}
	for _, c := range cases {
		name := fmt.Sprintf("pos=%d/n=%d", c.pos, c.n)
		refPanics := panics(func() { _ = refReadBitsAt(buf, c.pos, c.n) })
		gotPanics := panics(func() { _ = readBitsAt(buf, c.pos, c.n) })
		if refPanics != gotPanics {
			t.Fatalf("%s : panique=%v, reference=%v", name, gotPanics, refPanics)
		}
	}
	// Contre-epreuve : sur son domaine legal, elle ne panique jamais.
	if panics(func() { _ = readBitsAt(buf, 0, 32) }) {
		t.Fatal("readBitsAt(pos=0, n=32) a panique alors que la lecture tient dans le tampon")
	}
}

// TestReadBitsZeroPadsOutOfBuffer verrouille l'AUTRE semantique : les trois primitives
// gardees rendent des ZEROS au-dela du tampon, la ou `readBitsAt` panique.
func TestReadBitsZeroPadsOutOfBuffer(t *testing.T) {
	buf := []byte{0xFF, 0xFF, 0xFF, 0xFF}
	total := len(buf) * 8
	if got := kfReadBits(buf, total, 64); got != 0 {
		t.Fatalf("kfReadBits entierement hors tampon = %#x, attendu 0", got)
	}
	if got := PeekBits(buf, total, 64); got != 0 {
		t.Fatalf("PeekBits entierement hors tampon = %#x, attendu 0", got)
	}
	br := NewBitReader(buf)
	br.SetBitPos(total)
	if got := br.ReadBits(64); got != 0 {
		t.Fatalf("ReadBits entierement hors tampon = %#x, attendu 0", got)
	}
	if br.BitPos() != total+64 {
		t.Fatalf("ReadBits hors tampon : curseur = %d, attendu %d", br.BitPos(), total+64)
	}
	// A cheval sur la fin : les bits dedans valent 1, ceux dehors valent 0.
	if got, want := PeekBits(buf, total-4, 8), uint64(0xF0); got != want {
		t.Fatalf("PeekBits a cheval = %#x, attendu %#x", got, want)
	}
}

func panics(f func()) (did bool) {
	defer func() {
		if recover() != nil {
			did = true
		}
	}()
	f()
	return false
}
