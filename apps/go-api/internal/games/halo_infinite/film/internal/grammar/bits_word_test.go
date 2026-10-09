package grammar

// bits_word_test.go — TEST DIFFERENTIEL DU LECTEUR SEQUENTIEL DE LA GRAMMAIRE (decision D6 du
// plan `.ai/V7.5/PLAN_CUISSON_PERF.md`).
//
// LA METHODE. `Lecteur.ReadBits`, reecrit par mot au lot 4, est oppose a une COPIE DE REFERENCE
// de son implementation d'AVANT, recopiee ici et nulle part ailleurs — le code de production
// n'en garde aucune. Chaque cas est joue sur les deux et les resultats doivent coincider bit
// pour bit.
//
// LE DOMAINE COUVERT. Tampons pseudo-aleatoires a GRAINE FIXEE (donc rejouables), toutes les
// largeurs de 0 a 64 (et au-dela), et des positions choisies autour de CHAQUE frontiere qui
// compte : bit, octet, mot de 64, et fin de tampon (avant, sur, apres).
//
// CE FICHIER OPPOSAIT AUSSI TROIS PRIMITIVES SANS CURSEUR (`kfReadBits`, `readBitsAt`,
// `PeekBits`). Elles n'existent plus depuis le lot J4.6 (PLAN_SUITE_AUDIT_DECODEUR_FILM,
// 2026-09-26) : ce sont des conventions nommees de la couche `source` (`BitsBourres`,
// `BitsStricts`, `BitsTolerants`), et leurs differentiels — avec ceux de `kfBitAt`, `invBitAt`
// et `invBits` — vivent a cote d'elles, dans `source/bits_conventions_test.go`.

import (
	"math/rand"
	"testing"
)

// --- Copie de reference (implementation d'AVANT le lot 4, oracle du differentiel) ---

// refReadBitsSeq est `Lecteur.ReadBits` d'avant : lecture bit a bit, zero hors tampon,
// curseur avance de n quoi qu'il arrive.
func refReadBitsSeq(buf []byte, pos int, n uint) (uint64, int) {
	var r uint64
	for range n {
		var bit uint64
		if idx := pos >> 3; idx < len(buf) {
			bit = uint64(buf[idx]>>(7-(uint(pos)&7))) & 1
		}
		r = r<<1 | bit
		pos++
	}
	return r, pos
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

// --- Les deux differentiels ---

func TestReadBitsWordMatchesReference(t *testing.T) {
	for _, buf := range bitsFuzzBuffers() {
		for _, pos := range bitsFuzzPositions(len(buf)) {
			for n := uint(0); n <= 64; n++ {
				wantV, wantPos := refReadBitsSeq(buf, pos, n)
				br := LecteurSur(buf)
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
				br := LecteurSur(buf)
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

// TestReadBitsZeroPadsOutOfBuffer verrouille la semantique HORS TAMPON du lecteur sequentiel :
// des ZEROS au-dela du tampon, et un curseur qui avance quand meme de n.
func TestReadBitsZeroPadsOutOfBuffer(t *testing.T) {
	buf := []byte{0xFF, 0xFF, 0xFF, 0xFF}
	total := len(buf) * 8
	br := LecteurSur(buf)
	br.SetBitPos(total)
	if got := br.ReadBits(64); got != 0 {
		t.Fatalf("ReadBits entierement hors tampon = %#x, attendu 0", got)
	}
	if br.BitPos() != total+64 {
		t.Fatalf("ReadBits hors tampon : curseur = %d, attendu %d", br.BitPos(), total+64)
	}
}
