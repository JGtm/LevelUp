//go:build research

package grammar

// mouvement_5_17_type8_research_test.go — LE PAQUET DE TYPE 8 SUR LES DEUX TEMOINS.
//
// Il mesure ce que le port de `roster_type8.go` consomme sur des paquets REELS : combien de
// paquets de type 8 le film porte, combien d entrees chacun annonce, si le curseur DEBORDE (le
// seul verdict que le jeu rende, @142987d6e), et ce que les entrees portent (identites, cles,
// etiquettes). Il publie aussi le RESTE, qui n est pas un verdict du jeu mais dit si la grammaire
// est complete.
//
// Rejouable :
//
//	MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> MOUV511_CARTE=<carte> \
//	  go test -tags=research -count=1 -v -timeout 30m -run '^TestType817' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// t817Cadre ouvre le film temoin SANS entree de catalogue de carte : le paquet de type 8 ne porte
// aucune position, donc aucune largeur d axe n entre dans sa grammaire. Poser une carte substitut
// ici serait exactement le piege du report D3 (5.11) — des largeurs fausses sans erreur.
func t817Cadre(t *testing.T) *FilmContext {
	t.Helper()
	return m511Contexte(m511Film(t), profile.MapQuantEntry{})
}

// t817Bilan : le cumul d une passe sur un film.
type t817Bilan struct {
	paquets, octets               int
	entreesAnnoncees, entreesLues int
	debordements                  int
	resteNul, resteNonNul         int
	resteMin, resteMax            int
	bourrageNul, bourrageSale     int
}

// TestType817Population mesure les paquets de type 8 du film courant.
func TestType817Population(t *testing.T) {
	fc := t817Cadre(t)
	version, ok := FilmFormatVersion(fc.Film())
	if !ok {
		t.Fatalf("version de format illisible : sans elle la porte de tete est indecidable")
	}
	t.Logf("VERSION DE FORMAT (chunk_00 + 4) = %d — porte de tete par entree : %v",
		version, version >= rosterOptGateVersion)
	b := t817Bilan{resteMin: 1 << 30, resteMax: -(1 << 30)}
	cles := map[uint64]int{}
	ident := map[uint64]int{}
	etiquettes := map[string]int{}
	for _, c := range fc.ChunkNumbers() {
		data, pks, present := fc.ChunkAt(c)
		if !present {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeRoster {
				continue
			}
			t817Paquet(pk.Payload(data), version, &b, cles, ident, etiquettes)
		}
	}
	t.Logf("PAQUETS DE TYPE 8 : %d (%d octets) · entrees annoncees %d, lues %d · DEBORDEMENTS %d",
		b.paquets, b.octets, b.entreesAnnoncees, b.entreesLues, b.debordements)
	if b.paquets > 0 {
		t.Logf("  reste : NUL %d paquets · non nul %d · etendue [%d ; %d] bits",
			b.resteNul, b.resteNonNul, b.resteMin, b.resteMax)
		t.Logf("  FERMES A BOURRAGE NUL : %d / %d paquets (sales %d)",
			b.bourrageNul, b.paquets, b.bourrageSale)
	}
	t.Logf("IDENTITES DISTINCTES (entree+0x08) : %d", len(ident))
	for _, k := range t817TrierCles(ident) {
		t.Logf("  %#016x  x%d  (decimal %d)", k, ident[k], k)
	}
	t.Logf("CLES DISTINCTES (rec+0xcb8) : %d", len(cles))
	for _, k := range t817TrierCles(cles) {
		t.Logf("  %#016x  x%d", k, cles[k])
	}
	t.Logf("ETIQUETTES DISTINCTES (rec+0xc14) : %d", len(etiquettes))
	for _, s := range t817TrierEtiquettes(etiquettes) {
		t.Logf("  %-24q x%d", s, etiquettes[s])
	}
}

// t817Paquet mesure UN paquet de type 8 et cumule.
func t817Paquet(pay []byte, version int, b *t817Bilan,
	cles, ident map[uint64]int, etiquettes map[string]int) {
	b.paquets++
	b.octets += len(pay)
	entrees, bilan := DecodeRoster(pay, version)
	b.entreesAnnoncees += bilan.Annonce
	b.entreesLues += bilan.Entrees
	if bilan.Debordement {
		b.debordements++
	}
	if bilan.Reste == 0 {
		b.resteNul++
	} else {
		b.resteNonNul++
	}
	// LE GATE DU DEPOT : le reste tient dans un octet ET tous ses bits sont a ZERO (le bourrage
	// d octet est ecrit a zero). `c514ResteNul` est celui du lot 5.14.
	if bilan.Reste >= 0 && bilan.Reste < 8 && c514ResteNul(pay, bilan.BitsLus) {
		b.bourrageNul++
	} else {
		b.bourrageSale++
	}
	if bilan.Reste < b.resteMin {
		b.resteMin = bilan.Reste
	}
	if bilan.Reste > b.resteMax {
		b.resteMax = bilan.Reste
	}
	for _, e := range entrees {
		cles[e.Cle]++
		ident[e.Identite]++
		if e.Etiquette != "" {
			etiquettes[e.Etiquette]++
		}
	}
}

// t817TrierCles rend les cles par compte decroissant, bornees a 40 lignes.
func t817TrierCles(m map[uint64]int) []uint64 {
	out := make([]uint64, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool {
		if m[out[i]] != m[out[j]] {
			return m[out[i]] > m[out[j]]
		}
		return out[i] < out[j]
	})
	if len(out) > 40 {
		out = out[:40]
	}
	return out
}

// t817TrierEtiquettes rend les etiquettes par compte decroissant, bornees a 40 lignes.
func t817TrierEtiquettes(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool {
		if m[out[i]] != m[out[j]] {
			return m[out[i]] > m[out[j]]
		}
		return out[i] < out[j]
	})
	if len(out) > 40 {
		out = out[:40]
	}
	return out
}

// TestType817Types publie la POPULATION DE TYPES de paquet du film — le tableau que le §2.2 de la
// note 5.16 avait releve chez l ecrivain, re-mesure hors ligne, pour que « 700 868 octets de type
// 8 » soit un chiffre rejouable.
func TestType817Types(t *testing.T) {
	fc := t817Cadre(t)
	parType := map[uint16][2]int{} // type -> [paquets, octets]
	for _, c := range fc.ChunkNumbers() {
		data, pks, present := fc.ChunkAt(c)
		if !present {
			continue
		}
		for _, pk := range pks {
			e := parType[pk.Type]
			e[0]++
			e[1] += len(pk.Payload(data))
			parType[pk.Type] = e
		}
	}
	types := make([]int, 0, len(parType))
	for k := range parType {
		types = append(types, int(k))
	}
	sort.Ints(types)
	for _, k := range types {
		e := parType[uint16(k)] //nolint:gosec // k vient des cles de la carte, un u16
		t.Logf("  type %2d : %7d paquets · %10d octets", k, e[0], e[1])
	}
}

// TestType817Etapes DUMP les positions de bit etape par etape sur la premiere entree du premier
// paquet de type 8 : c est la mesure qui LOCALISE un maillon court, pas un reglage. Chaque ligne
// nomme le lecteur de l ecrivain et la position ou il finit.
func TestType817Etapes(t *testing.T) {
	fc := t817Cadre(t)
	version, _ := FilmFormatVersion(fc.Film())
	for _, c := range fc.ChunkNumbers() {
		data, pks, present := fc.ChunkAt(c)
		if !present {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeRoster {
				continue
			}
			t817Etapes(t, pk.Payload(data), version)
			return
		}
	}
	t.Skip("aucun paquet de type 8")
}

// t817Etapes rejoue la premiere entree en publiant chaque position.
func t817Etapes(t *testing.T, pay []byte, version int) {
	t.Helper()
	br := LecteurSur(pay)
	n32 := br.ReadBits(rosterCountBits)
	t.Logf("payload %d octets = %d bits · N = %d (bit %d)", len(pay), len(pay)*8, n32, br.BitPos())
	for k := 0; k < int(n32); k++ { //nolint:gosec // n32 borne par le payload de l instrument
		if br.Remaining() <= 0 {
			break
		}
		t.Logf(" --- entree %d, debut bit %d", k, br.BitPos())
		t817Entree(t, br, version)
	}
	t.Logf("FIN bit %d · RESTE %d bits", br.BitPos(), len(pay)*8-br.BitPos())
}

// t817Entree publie les positions d UNE entree.
func t817Entree(t *testing.T, br *Lecteur, version int) {
	t.Helper()
	if version >= rosterOptGateVersion {
		if ouverte := br.ReadBit(); !ouverte {
			t.Logf("  porte FERMEE -> R(5) = %d (bit %d)", br.ReadBits(rosterOptWidth), br.BitPos())
		} else {
			t.Logf("  porte OUVERTE (bit %d)", br.BitPos())
		}
	}
	t.Logf("  identite = %#016x (bit %d)", br.ReadBits(rosterIdentityBits), br.BitPos())
	nf := br.ReadBits(rosterFlagCountBits) + 1
	br.Skip(int(nf)) //nolint:gosec // nf est un R(11) + 1
	t.Logf("  FUN_142bdeddc n = %d -> %d bits de drapeaux (bit %d)", nf, nf, br.BitPos())
	l1 := br.ReadBits(rosterByteLenBits)
	br.Skip(int(l1) * 8) //nolint:gosec // l1 est un R(12)
	t.Logf("  FUN_1411b1bd8 L1 = %d octets (bit %d)", l1, br.BitPos())
	l2 := br.ReadBits(rosterWordLenBits)
	br.Skip(int(l2) * 32) //nolint:gosec // l2 est un R(8)
	t.Logf("  FUN_1411b1b04 L2 = %d mots (bit %d)", l2, br.BitPos())
	br.Skip(rosterBlobABits)
	t.Logf("  R(0x340) @rec+0xc48 (bit %d)", br.BitPos())
	t.Logf("  etiquette = %q (bit %d)", rosterLireEtiquette(br), br.BitPos())
	br.Skip(rosterBlobBBits)
	t.Logf("  R(0x80) @rec+0xc38 (bit %d)", br.BitPos())
	t.Logf("  representation = %#x (bit %d)", br.ReadBits(rosterRepresentationBits), br.BitPos())
	t.Logf("  CLE = %#016x (bit %d)", br.ReadBits(rosterKeyBits), br.BitPos())
	t.Logf("  R(10)=%d R(14)=%d R(6)=%d R(8)=%d R(7)=%d",
		br.ReadBits(rosterF10Bits), br.ReadBits(rosterF14Bits), br.ReadBits(rosterF6Bits),
		br.ReadBits(rosterByteBits), br.ReadBits(rosterF7Bits))
	t.Logf("  bit = %v (bit %d)", br.ReadBit(), br.BitPos())
	br.Skip(rosterBlobCBits)
	t.Logf("  R(0x39e0) @rec+0xcc0 (bit %d)", br.BitPos())
	br.Skip(rosterBlobDBits)
	t.Logf("  R(0x160) @rec+0x1400 (bit %d)", br.BitPos())
}

// TestType817Dump publie le payload du premier paquet de type 8 en hexadecimal et signale les
// motifs UTF-16 ASCII : c est la mesure qui dit OU la chaine se tient reellement.
func TestType817Dump(t *testing.T) {
	fc := t817Cadre(t)
	for _, c := range fc.ChunkNumbers() {
		data, pks, present := fc.ChunkAt(c)
		if !present {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeRoster {
				continue
			}
			pay := pk.Payload(data)
			t817Motifs(t, pay)
			for off := 0; off < len(pay) && off < 256; off += 32 {
				fin := off + 32
				if fin > len(pay) {
					fin = len(pay)
				}
				t.Logf("  %04x  %x", off, pay[off:fin])
			}
			t.Logf("  ... queue :")
			for off := len(pay) - 128; off < len(pay); off += 32 {
				t.Logf("  %04x  %x", off, pay[off:off+32])
			}
			return
		}
	}
	t.Skip("aucun paquet de type 8")
}

// t817Motifs signale les suites `XX 00 XX 00` d ASCII imprimable : une chaine UTF-16 alignee.
func t817Motifs(t *testing.T, pay []byte) {
	t.Helper()
	for i := 0; i+8 <= len(pay); i++ {
		ok := true
		for k := 0; k < 4; k++ {
			c := pay[i+2*k]
			if pay[i+2*k+1] != 0 || c < 0x20 || c > 0x7e {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		var s []rune
		for j := i; j+1 < len(pay); j += 2 {
			c := pay[j]
			if pay[j+1] != 0 || c < 0x20 || c > 0x7e {
				break
			}
			s = append(s, rune(c))
		}
		t.Logf("  UTF-16 a l octet %d (bit %d) : %q", i, i*8, string(s))
		i += 2 * len(s)
	}
	// Les plages de zeros de plus de 64 octets : elles cadrent les blocs opaques.
	debut := -1
	for i := 0; i <= len(pay); i++ {
		if i < len(pay) && pay[i] == 0 {
			if debut < 0 {
				debut = i
			}
			continue
		}
		if debut >= 0 && i-debut > 64 {
			t.Logf("  ZEROS octets %d..%d (%d octets, bits %d..%d)",
				debut, i-1, i-debut, debut*8, i*8)
		}
		debut = -1
	}
}
