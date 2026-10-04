package source_test

// bits_conventions_test.go — LES SEPT LECTEURS DE LA GRAMMAIRE SONT DEVENUS DES CONVENTIONS
// NOMMEES DE LA SOURCE, ET CE FICHIER LE PROUVE (lot J4.6 du PLAN_SUITE_AUDIT_DECODEUR_FILM,
// decision DU-3 option S2, ADR 0034 D-2).
//
// # LA METHODE
//
// Chaque lecteur de bits que la couche `grammar` ecrivait a la main est recopie ICI — sa BOUCLE
// D ORIGINE, qui porte sa semantique, et pour `kfReadBits` aussi sa forme exacte a `a15bfc126`
// (dernier etat avant le lot) — et nulle part ailleurs — le code de production
// n en garde aucune copie. Chaque copie est opposee a la convention de la source qui l a
// remplacee, sur des tampons pseudo-aleatoires a GRAINE FIXEE, toutes largeurs de 0 a 64 plus
// des largeurs > 64, et des positions autour de CHAQUE frontiere (bit, octet, mot de 64, fin de
// tampon ; et AVANT le debut pour les conventions qui y ont une regle). Valeur ET panique doivent
// coincider : une convention qui paniquerait moins, ou plus tot, ne serait pas la meme.
//
//	grammar.kfReadBits + kfReadBitsLoop  -> source.BitsBourres
//	grammar.readBitsAt                   -> uint32(source.BitsStricts)
//	grammar.PeekBits                     -> source.BitsTolerants
//	grammar.invBits                      -> uint32(source.BitsTolerants)
//	grammar.invBitAt                     -> uint32(source.BitAt)
//	grammar.kfBitAt                      -> source.BitAt (positions >= 0, cf. son test)
//
// Les tampons, les positions et le detecteur de panique reprennent ceux du differentiel du
// lecteur sequentiel (`grammar/bits_word_test.go`), dont ce fichier a herite les trois tests des
// primitives sans curseur.

import (
	"fmt"
	"math/rand"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// --- Copies de reference : les sept lecteurs de `grammar` a `a15bfc126` ---

// refKfReadBits est `grammar.kfReadBits` d avant, chemin par mot compris.
func refKfReadBits(buf []byte, pos, n int) uint64 {
	if pos >= 0 && n >= 0 && n <= 64 {
		return source.BitsAt(buf, pos, uint(n))
	}
	return refKfReadBitsLoop(buf, pos, n)
}

// refKfReadBitsLoop est `grammar.kfReadBitsLoop` d avant — la boucle d origine, qui est aussi
// l oracle SEMANTIQUE de `kfReadBits` (le differentiel du lot 4 opposait les deux).
func refKfReadBitsLoop(buf []byte, pos, n int) uint64 {
	var r uint64
	for i := range n {
		p := pos + i
		var bit uint64
		if idx := p >> 3; idx < len(buf) {
			bit = uint64(buf[idx]>>(7-uint(p&7))) & 1
		}
		r = r<<1 | bit
	}
	return r
}

// refKfBitAt est `grammar.kfBitAt` d avant.
func refKfBitAt(buf []byte, p int) uint64 {
	if idx := p >> 3; idx < len(buf) {
		return uint64(buf[idx]>>(7-uint(p&7))) & 1
	}
	return 0
}

// refReadBitsAt est la BOUCLE D ORIGINE de `grammar.readBitsAt` : indexation NUE. Son chemin
// par mot n en etait qu une optimisation, prouvee equivalente au lot 4 (`PLAN_CUISSON_PERF` D6).
func refReadBitsAt(b []byte, pos, n int) uint32 {
	var v uint32
	for i := range n {
		p := pos + i
		v = v<<1 | uint32(b[p>>3]>>(7-uint(p&7))&1)
	}
	return v
}

// refPeekBits est la BOUCLE D ORIGINE de `grammar.PeekBits` : zero des DEUX cotes du tampon.
func refPeekBits(d []byte, bp, n int) uint64 {
	var v uint64
	for i := range n {
		p := bp + i
		if p < 0 || p>>3 >= len(d) {
			v <<= 1
			continue
		}
		v = (v << 1) | uint64((d[p>>3]>>uint(7-(p&7)))&1)
	}
	return v
}

// refInvBitAt est `grammar.invBitAt` d avant : un bit, zero hors bornes des deux cotes.
func refInvBitAt(buf []byte, p int) uint32 {
	if idx := p >> 3; idx >= 0 && idx < len(buf) {
		return uint32(buf[idx]>>(7-uint(p&7))) & 1
	}
	return 0
}

// refInvBits est `grammar.invBits` d avant : n bits par la boucle de `invBitAt`.
func refInvBits(pay []byte, p, n int) uint32 {
	var v uint32
	for i := range n {
		v = v<<1 | refInvBitAt(pay, p+i)
	}
	return v
}

// --- Materiel commun ---

// conventionsTampons rend des tampons pseudo-aleatoires a graine FIXEE : plus court qu un mot,
// le mot exact, et des tailles quelconques.
func conventionsTampons() [][]byte {
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

// conventionsPositions rend les positions a eprouver pour un tampon de `nOctets` octets : le
// debut, chaque frontiere d octet et de mot des premiers octets, le voisinage de la fin (dedans,
// dessus, dehors) et, si `negatives`, des positions AVANT le debut.
func conventionsPositions(nOctets int, negatives bool) []int {
	total := nOctets * 8
	vus := map[int]bool{}
	var out []int
	add := func(p int) {
		if !vus[p] {
			vus[p] = true
			out = append(out, p)
		}
	}
	for p := 0; p < 32 && p <= total+16; p++ {
		add(p)
	}
	for _, bord := range []int{8, 16, 56, 64, 72, 120, 128, 136, total} {
		for d := -2; d <= 2; d++ {
			if bord+d >= 0 {
				add(bord + d)
			}
		}
	}
	for _, p := range []int{total + 7, total + 8, total + 64, total + 65} {
		add(p)
	}
	if negatives {
		for _, p := range []int{-1, -2, -7, -8, -9, -64, -65} {
			add(p)
		}
	}
	return out
}

// conventionsLargeurs : 0..64, puis des largeurs > 64 (une largeur tiree du flux peut aller
// jusqu a 4 095, cf. `consumeTrackFrameComponent`).
func conventionsLargeurs() []int {
	out := make([]int, 0, 72)
	for n := 0; n <= 64; n++ {
		out = append(out, n)
	}
	return append(out, 65, 66, 96, 127, 128, 129, 200)
}

// resultat rend la valeur d une lecture, ou le fait qu elle a panique.
func resultat(f func() uint64) (v uint64, panique bool) {
	defer func() {
		if recover() != nil {
			panique = true
		}
	}()
	return f(), false
}

// opposer joue une lecture sur la reference et sur la convention, et exige la MEME issue.
func opposer(t *testing.T, nom string, ref, conv func() uint64) {
	t.Helper()
	rv, rp := resultat(ref)
	cv, cp := resultat(conv)
	if rp != cp || rv != cv {
		t.Fatalf("%s : convention (%#x, panique=%v), reference (%#x, panique=%v)", nom, cv, cp, rv, rp)
	}
}

// --- Les differentiels ---

// TestBitsBourresEgaleKfReadBitsDAvant : `BitsBourres` est `kfReadBits` + `kfReadBitsLoop`,
// positions negatives (panique des deux cotes) et largeurs > 64 comprises.
func TestBitsBourresEgaleKfReadBitsDAvant(t *testing.T) {
	for _, buf := range conventionsTampons() {
		for _, pos := range conventionsPositions(len(buf), true) {
			for _, n := range conventionsLargeurs() {
				nom := fmt.Sprintf("BitsBourres(len=%d, pos=%d, n=%d)", len(buf), pos, n)
				conv := func() uint64 { return source.BitsBourres(buf, pos, n) }
				opposer(t, nom, func() uint64 { return refKfReadBits(buf, pos, n) }, conv)
				opposer(t, nom+" vs boucle", func() uint64 { return refKfReadBitsLoop(buf, pos, n) }, conv)
			}
		}
	}
}

// TestBitsStrictsEgaleReadBitsAtDAvant : `uint32(BitsStricts)` est `readBitsAt`, y compris
// LA OU ELLE PANIQUE — ni plus tot, ni plus tard.
func TestBitsStrictsEgaleReadBitsAtDAvant(t *testing.T) {
	for _, buf := range conventionsTampons() {
		for _, pos := range conventionsPositions(len(buf), true) {
			for _, n := range conventionsLargeurs() {
				opposer(t, fmt.Sprintf("BitsStricts(len=%d, pos=%d, n=%d)", len(buf), pos, n),
					func() uint64 { return uint64(refReadBitsAt(buf, pos, n)) },
					func() uint64 { return uint64(uint32(source.BitsStricts(buf, pos, n))) }) //nolint:gosec // troncature de l appelant
			}
		}
	}
	// Contre-epreuve : sur son domaine legal elle ne panique jamais, hors de lui toujours.
	buf := make([]byte, 4)
	if _, p := resultat(func() uint64 { return source.BitsStricts(buf, 0, 32) }); p {
		t.Fatal("BitsStricts(pos=0, n=32) a panique alors que la lecture tient dans le tampon")
	}
	for _, c := range []struct{ pos, n int }{{31, 2}, {32, 1}, {-1, 1}, {0, 33}} {
		if _, p := resultat(func() uint64 { return source.BitsStricts(buf, c.pos, c.n) }); !p {
			t.Fatalf("BitsStricts(pos=%d, n=%d) hors du tampon n a PAS panique", c.pos, c.n)
		}
	}
}

// TestBitsTolerantsEgalePeekBitsEtInvBitsDAvant : `BitsTolerants` est `PeekBits`, et sa
// troncature a 32 bits est `invBits` — zero des DEUX cotes, jamais de panique.
func TestBitsTolerantsEgalePeekBitsEtInvBitsDAvant(t *testing.T) {
	for _, buf := range conventionsTampons() {
		for _, pos := range conventionsPositions(len(buf), true) {
			for _, n := range conventionsLargeurs() {
				nom := fmt.Sprintf("BitsTolerants(len=%d, pos=%d, n=%d)", len(buf), pos, n)
				opposer(t, nom+" vs PeekBits",
					func() uint64 { return refPeekBits(buf, pos, n) },
					func() uint64 { return source.BitsTolerants(buf, pos, n) })
				opposer(t, nom+" vs invBits",
					func() uint64 { return uint64(refInvBits(buf, pos, n)) },
					func() uint64 { return uint64(uint32(source.BitsTolerants(buf, pos, n))) }) //nolint:gosec // troncature de l appelant
			}
		}
	}
}

// TestBitAtEgaleInvBitAtDAvant : `BitAt` est `invBitAt` — un bit, zero des deux cotes.
func TestBitAtEgaleInvBitAtDAvant(t *testing.T) {
	for _, buf := range conventionsTampons() {
		for _, pos := range conventionsPositions(len(buf), true) {
			opposer(t, fmt.Sprintf("BitAt(len=%d, pos=%d)", len(buf), pos),
				func() uint64 { return uint64(refInvBitAt(buf, pos)) },
				func() uint64 { return uint64(source.BitAt(buf, pos)) }) //nolint:gosec // 0 ou 1
		}
	}
}

// TestBitAtEgaleKfBitAtDAvantSurSonDomaine : `BitAt` est `kfBitAt` sur les positions >= 0 — le
// seul domaine que ses appelants atteignent. `kfBitAt` se documentait « 0 hors borne », mais une
// position NEGATIVE y paniquait (indexation nue, comme `kfReadBitsLoop`) ; `BitAt` y rend 0.
//
// LA DIFFERENCE EST HORS D ATTEINTE, ET VOICI POURQUOI (mesure du 2026-09-26 sur `a15bfc126`) :
// ses six appelants de production lisent a une position >= 0 PAR CONSTRUCTION —
// `marchHasEvents` au bit 1, `marchLocateSignatures` / `marchLocateFallback` au bit `s-1` avec
// `s >= 2`, `ScanKeyframeLoadoutsMarche` au bit `b >= 0` de sa boucle, `lireBitmapsDeDatum`
// (bloc de type 1) a `pos + k` avec `pos` parti de 0. Le test fige les deux faits : l egalite
// sur le domaine, et la difference hors de lui, pour qu elle ne soit pas redecouverte.
func TestBitAtEgaleKfBitAtDAvantSurSonDomaine(t *testing.T) {
	for _, buf := range conventionsTampons() {
		for _, pos := range conventionsPositions(len(buf), false) {
			opposer(t, fmt.Sprintf("BitAt(len=%d, pos=%d)", len(buf), pos),
				func() uint64 { return refKfBitAt(buf, pos) },
				func() uint64 { return uint64(source.BitAt(buf, pos)) }) //nolint:gosec // 0 ou 1
		}
	}
	buf := []byte{0xFF}
	if _, p := resultat(func() uint64 { return refKfBitAt(buf, -1) }); !p {
		t.Fatal("la copie de kfBitAt ne panique plus a -1 : la difference documentee a change")
	}
	if got := source.BitAt(buf, -1); got != 0 {
		t.Fatalf("BitAt(-1) = %d, attendu 0 (zero des deux cotes)", got)
	}
}

// TestConventionsZeroHorsDuTampon : les conventions TOLERANTES rendent des zeros au-dela de la
// fin, et a cheval les bits du dedans restent lus. Herite de
// `grammar.TestReadBitsZeroPadsOutOfBuffer` et `grammar.TestPeekBitsToleranceDesDeuxCotes`.
func TestConventionsZeroHorsDuTampon(t *testing.T) {
	buf := []byte{0xFF, 0xFF, 0xFF, 0xFF}
	total := len(buf) * 8
	if got := source.BitsBourres(buf, total, 64); got != 0 {
		t.Fatalf("BitsBourres entierement hors tampon = %#x, attendu 0", got)
	}
	if got := source.BitsTolerants(buf, total, 64); got != 0 {
		t.Fatalf("BitsTolerants entierement hors tampon = %#x, attendu 0", got)
	}
	if got, want := source.BitsTolerants(buf, total-4, 8), uint64(0xF0); got != want {
		t.Fatalf("BitsTolerants a cheval sur la fin = %#x, attendu %#x", got, want)
	}
	d := []byte{0xFF, 0xFF}
	if got := source.BitsTolerants(d, -8, 8); got != 0 {
		t.Errorf("BitsTolerants entierement avant le debut = %#x, attendu 0", got)
	}
	if got := source.BitsTolerants(d, -4, 8); got != 0x0F {
		t.Errorf("BitsTolerants a cheval sur le debut = %#x, attendu 0x0F", got)
	}
	if got := source.BitsTolerants(nil, -4, 8); got != 0 {
		t.Errorf("BitsTolerants sur tampon vide = %#x, attendu 0", got)
	}
}
