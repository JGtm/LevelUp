package weaponv3

// bits_word_test.go — TEST DIFFERENTIEL de la lecture de bits par mot du resolveur
// xuid -> player_index (decision D6 du plan `.ai/V7.5/PLAN_CUISSON_PERF.md`).
//
// Deux proprietes, deux oracles, tous deux recopies ici depuis l'implementation d'AVANT :
//   - `readBits` doit rendre la meme valeur que la lecture bit a bit, sur tout le domaine
//     reel — largeurs 0..64, positions autour des frontieres d'octet, de mot et de fin de
//     tampon, positions NEGATIVES comprises (le resolveur relit les 5 bits qui PRECEDENT le
//     motif trouve : sur un motif en tete de chunk, il recule sous zero et doit y lire des
//     zeros de bourrage, jamais paniquer) ;
//   - `findPattern64` doit rendre la MEME PREMIERE POSITION que le balayage bit a bit
//     d'origine — c'est cette position qui decide de l'index de joueur publie.

import (
	"math/rand"
	"testing"
)

// refBit est `bitReader.bit` : zero des deux cotes du tampon.
func refBit(data []byte, total, p int) int {
	if p < 0 || p >= total {
		return 0
	}
	return int((data[p>>3] >> uint(7-(p&7))) & 1)
}

// refReadBits est `bitReader.readBits` d'avant : lecture bit a bit.
func refReadBits(data []byte, total, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		v = (v << 1) | uint64(refBit(data, total, bp+i))
	}
	return v
}

// refFindPattern64 est le balayage d'origine de `ResolveXuidToPI` : premiere position de bit
// dont les 64 bits suivants valent `target`.
func refFindPattern64(data []byte, target uint64) (int, bool) {
	total := len(data) * 8
	for bp := 0; bp+64 <= total; bp++ {
		if refReadBits(data, total, bp, 64) == target {
			return bp, true
		}
	}
	return 0, false
}

func piFuzzBuffers() [][]byte {
	rng := rand.New(rand.NewSource(0x51A7_0B17))
	sizes := []int{0, 1, 7, 8, 9, 16, 17, 23, 31, 64, 65}
	out := make([][]byte, 0, len(sizes))
	for _, n := range sizes {
		b := make([]byte, n)
		rng.Read(b)
		out = append(out, b)
	}
	return out
}

func TestPIReadBitsWordMatchesReference(t *testing.T) {
	for _, buf := range piFuzzBuffers() {
		br := newBitReader(buf)
		total := br.total
		var positions []int
		for p := -PIBits - 8; p < 40; p++ {
			positions = append(positions, p)
		}
		for _, edge := range []int{56, 64, 72, 120, 128, total} {
			for d := -2; d <= 2; d++ {
				positions = append(positions, edge+d)
			}
		}
		positions = append(positions, total+7, total+64)
		for _, bp := range positions {
			for n := 0; n <= 64; n++ {
				got := br.readBits(bp, n)
				want := refReadBits(buf, total, bp, n)
				if got != want {
					t.Fatalf("readBits(len=%d, bp=%d, n=%d) = %#x, reference %#x",
						len(buf), bp, n, got, want)
				}
			}
		}
	}
}

// TestFindPattern64MatchesReference eprouve le balayage sur des motifs REELLEMENT presents,
// implantes a chaque decalage de bit possible (0..7) et a plusieurs profondeurs — dont la
// toute premiere position, ou la relecture des 5 bits precedents recule sous zero.
func TestFindPattern64MatchesReference(t *testing.T) {
	rng := rand.New(rand.NewSource(0x1701_B175))
	const target = uint64(0x0123456789ABCDEF)
	for _, size := range []int{9, 16, 24, 40, 41} {
		for shift := 0; shift < 8; shift++ {
			for _, atByte := range []int{0, 1, 3} {
				buf := make([]byte, size)
				rng.Read(buf)
				bp := atByte*8 + shift
				if bp+64 > size*8 {
					continue
				}
				writeBitsBE(buf, bp, 64, target)
				wantPos, wantOK := refFindPattern64(buf, target)
				gotPos, gotOK := findPattern64(buf, target)
				if gotOK != wantOK || gotPos != wantPos {
					t.Fatalf("findPattern64(size=%d, implante a bp=%d) = (%d, %v), reference (%d, %v)",
						size, bp, gotPos, gotOK, wantPos, wantOK)
				}
			}
		}
	}
}

// TestFindPattern64AbsentMatchesReference : motif absent, et tampons plus courts que le
// motif lui-meme.
func TestFindPattern64AbsentMatchesReference(t *testing.T) {
	for _, buf := range piFuzzBuffers() {
		for _, target := range []uint64{0, ^uint64(0), 0x0123456789ABCDEF} {
			wantPos, wantOK := refFindPattern64(buf, target)
			gotPos, gotOK := findPattern64(buf, target)
			if gotOK != wantOK || gotPos != wantPos {
				t.Fatalf("findPattern64(len=%d, target=%#x) = (%d, %v), reference (%d, %v)",
					len(buf), target, gotPos, gotOK, wantPos, wantOK)
			}
		}
	}
}

// TestResolveXuidToPIMatchesReference : le resolveur complet, sur un chunk ou le motif d'un
// xuid est implante a un decalage non aligne, y compris en TETE de tampon (les 5 bits d'index
// se lisent alors sous zero et doivent valoir 0).
func TestResolveXuidToPIMatchesReference(t *testing.T) {
	const xuid = uint64(2533274790395904)
	target := xuidTargetPattern(xuid)
	for _, bp := range []int{0, 1, 5, 6, 7, 8, 13, 64, 71} {
		buf := make([]byte, 32)
		writeBitsBE(buf, bp, 64, target)
		want := map[uint64]int{}
		if p, ok := refFindPattern64(buf, target); ok {
			want[xuid] = int(refReadBits(buf, len(buf)*8, p-PIBits, PIBits))
		}
		got := ResolveXuidToPI([]uint64{xuid}, buf)
		if len(got) != len(want) || got[xuid] != want[xuid] {
			t.Fatalf("ResolveXuidToPI (motif a bp=%d) = %v, reference %v", bp, got, want)
		}
	}
}

// writeBitsBE ecrit les n bits de poids faible de v a la position bit bp, MSB d'abord.
func writeBitsBE(buf []byte, bp, n int, v uint64) {
	for i := 0; i < n; i++ {
		p := bp + i
		bit := byte((v >> uint(n-1-i)) & 1)
		mask := byte(1) << uint(7-(p&7))
		buf[p>>3] &^= mask
		if bit == 1 {
			buf[p>>3] |= mask
		}
	}
}
