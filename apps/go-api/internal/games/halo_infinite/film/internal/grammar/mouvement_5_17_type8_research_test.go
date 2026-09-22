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

// t817Perso rend la largeur, en bits, du bloc de personnalisation du build du film — la MEME
// valeur que la table de `chunk_00` emploie (`profile.PersonnalisationOctets`). Un build inconnu
// est un arret : lire le type 8 au profil d un build voisin est exactement ce que l ADR 0034
// interdit.
func t817Perso(t *testing.T, fc *FilmContext) int {
	t.Helper()
	reg, ok := FilmRegistryChunk(fc.Film())
	if !ok {
		t.Fatal("le film ne porte pas son chunk_00")
	}
	ident, err := ReadFilmIdentity(reg)
	if err != nil {
		t.Fatalf("identite : %v", err)
	}
	octets, connu := profile.PersonnalisationOctets(ident.Build)
	if !connu {
		t.Fatalf("build %q inconnu : le film est mis de cote", ident.Build)
	}
	t.Logf("BUILD %q · bloc de personnalisation %d octets", ident.Build, octets)
	return octets * 8
}

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
	persoBits := t817Perso(t, fc)
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
			t817Paquet(pk.Payload(data), version, persoBits, &b, cles, ident, etiquettes)
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
func t817Paquet(pay []byte, version, persoBits int, b *t817Bilan,
	cles, ident map[uint64]int, etiquettes map[string]int) {
	b.paquets++
	b.octets += len(pay)
	entrees, bilan := DecodeRoster(pay, version, persoBits)
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
		cles[e.Joueur.Shorts.Q64]++
		ident[e.Identite]++
		if e.Joueur.Gamertag != "" {
			etiquettes[e.Joueur.Gamertag]++
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

// TestType817Etapes publie, entree par entree, la POSITION et la LONGUEUR que le lecteur de
// production consomme sur le premier paquet de type 8 — plus les champs nommes. C est
// l instrument qui a LOCALISE le maillon court du lot (`FUN_142bdeddc` rend `R(11) + 1`, pas
// `R(11)`) : sous le cadrage faux les longueurs d entree sautaient de 16 382 a 45 588 bits et les
// deux longueurs prefixees sortaient de leurs bornes de structure ; sous le cadrage juste elles
// se suivent et le paquet ferme.
//
// Il ne recopie PAS la grammaire : il lit par [DecodeRoster] et publie ce que le port rend.
func TestType817Etapes(t *testing.T) {
	fc := t817Cadre(t)
	persoBits := t817Perso(t, fc)
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
			pay := pk.Payload(data)
			entrees, bilan := DecodeRoster(pay, version, persoBits)
			t.Logf("payload %d octets = %d bits · %+v", len(pay), len(pay)*8, bilan)
			for k, e := range entrees {
				t.Logf("  entree %d : bit %d, %d bits · XUID %#016x · %q · repr %#x · cle %#016x",
					k, e.Joueur.Bit, e.Joueur.TotalBits, e.Identite, e.Joueur.Gamertag,
					e.Joueur.Shorts.Repr, e.Joueur.Shorts.Q64)
			}
			return
		}
	}
	t.Skip("aucun paquet de type 8")
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
