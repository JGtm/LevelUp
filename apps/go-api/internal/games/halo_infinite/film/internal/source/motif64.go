package source

// motif64.go — LA RECHERCHE D UNE FENETRE DE 64 BITS, POSITION DE BIT PAR POSITION DE BIT, SANS
// RELIRE LE TAMPON A CHAQUE PAS (2026-09-28, temps CI du paquet `grammar`).
//
// # POURQUOI UNE PRIMITIVE
//
// La recherche d ancres d image-cle (`grammar.kfRecherche.suivante`) lisait 32 bits a CHAQUE
// position de bit d une fenetre de 120 000 bits, puis 32 de plus pour son filtre fort : 77 % du
// temps CPU du paquet `grammar` sous les options de couverture de la CI (1 043 s mesurees le
// 2026-09-28, pour 79 s sans couverture). Sous `-covermode=atomic`, chaque bloc execute paie un
// increment ATOMIQUE : la boucle d origine en executait une dizaine par position.
//
// [PremierMotif64] fait le meme balayage par BLOC ALIGNE DE HUIT POSITIONS : un mot de 64 bits et
// l octet qui le suit donnent les huit fenetres du bloc par decalage, et un corps SANS BRANCHE
// ([Motif64.huit]) les juge toutes — un seul bloc execute pour huit positions. Il ne DECIDE rien :
// il rend la premiere position ou la fenetre PEUT etre ce que l appelant cherche, et l appelant
// rejoue a cette position son test complet. Un motif doit donc etre un SUR-ENSEMBLE du test de
// l appelant ; le differentiel qui le prouve pour la recherche d ancres est
// `grammar/keyframe_world_motif_test.go`.
//
// # LA SEMANTIQUE
//
// La fenetre a la position `p` est [BitsAt]`(d, p, 64)` : les 64 bits MSB d abord qui commencent
// au bit `p`. Seules les positions dont la fenetre est ENTIEREMENT dans le tampon
// (`p + 64 <= len(d)*8`) sont examinees — aucun bit de bourrage n entre dans un jugement.

import (
	"encoding/binary"
	"math/bits"
)

// Motif64 decrit les fenetres que [PremierMotif64] retient. Une fenetre `w` est retenue si
//
//	w & Nuls == 0  ET, pour chaque masque NON NUL de AuMoinsUn, w & masque != 0   (la forme)
//	OU, quand AvecTete : les 32 premiers bits de w valent Tete
type Motif64 struct {
	// Nuls : les bits qui doivent valoir 0 dans la forme.
	Nuls uint64
	// AuMoinsUn : deux groupes de bits dont chacun doit porter au moins un 1 dans la forme. Un
	// groupe nul n impose rien.
	AuMoinsUn [2]uint64
	// Tete : la valeur des 32 premiers bits d une fenetre retenue sans condition de forme.
	Tete uint32
	// AvecTete : la tete est cherchee.
	AvecTete bool
}

// PremierMotif64 rend la premiere position `p` de [from, limit) dont la fenetre de 64 bits est
// retenue par `m`, ou -1. `limit` est ramene a la derniere position dont la fenetre tient dans
// le tampon. PRECONDITION : `from >= 0` (comme [BitsAt]).
func PremierMotif64(d []byte, from, limit int, m Motif64) int {
	limit = min(limit, len(d)*8-63)
	j := m.jugement()
	// Blocs ALIGNES de huit positions p..p+7. Le premier bloc peut commencer avant `from` et le
	// dernier finir apres `limit` : leurs positions hors de [from, limit) sont masquees. Les
	// fenetres de ces positions masquees peuvent deborder du tampon (bits de bourrage) ; celles
	// des positions gardees, jamais (p < limit <= len(d)*8-63).
	for p := from &^ 7; p < limit; p += 8 {
		w0, nb := blocDeHuit(d, p)
		acc := j.huit(w0, nb)
		if p < from {
			acc &= 0xFF << uint(from-p)
		}
		if limit-p < 8 {
			acc &= 1<<uint(limit-p) - 1
		}
		if acc != 0 {
			return p + bits.TrailingZeros64(acc)
		}
	}
	return -1
}

// blocDeHuit rend le mot de 64 bits qui commence au bit `p` (aligne sur un octet) et l octet qui
// le suit, bits hors du tampon a zero.
func blocDeHuit(d []byte, p int) (w0, nb uint64) {
	if i := p >> 3; i+9 <= len(d) {
		return binary.BigEndian.Uint64(d[i : i+8]), uint64(d[i+8])
	}
	return BitsAt(d, p, 64), BitsAt(d, p+64, 8)
}

// jugement : un [Motif64] mis sous la forme que [jugement.huit] calcule sans branche.
type jugement struct {
	n, u1, u2, t uint64
	// c1, c2 valent 1 quand le groupe AuMoinsUn correspondant est nul (il n impose rien) : `w&u|c`
	// est alors non nul pour toute fenetre. a vaut 1 quand la tete est cherchee.
	c1, c2, a uint64
}

// jugement prepare le motif pour [jugement.huit].
func (m Motif64) jugement() jugement {
	j := jugement{n: m.Nuls, u1: m.AuMoinsUn[0], u2: m.AuMoinsUn[1], t: uint64(m.Tete)}
	j.c1, j.c2 = 1^(j.u1|-j.u1)>>63, 1^(j.u2|-j.u2)>>63
	if m.AvecTete {
		j.a = 1
	}
	return j
}

// huit rend, bit k pour la fenetre k, le jugement des huit fenetres qui commencent aux bits 0 a 7
// du mot `w0`, `nb` etant l octet qui le suit.
//
// SANS BRANCHE ET SANS APPEL, ET C EST LE POINT : ce corps est UN bloc de couverture pour huit
// positions (cf. l en-tete). `(v|-v)>>63` vaut 1 si v != 0, et 0 sinon ; le meme jugement est donc
// ecrit huit fois, a l identique, un par decalage. Le differentiel `motif64_test.go` confronte
// chaque bloc a huit jugements ecrits en clair.
func (j jugement) huit(w0, nb uint64) uint64 {
	w := w0
	z, y, v, x := w&j.n, w&j.u1|j.c1, w&j.u2|j.c2, w>>32^j.t
	acc := (1^(z|-z)>>63)&((y|-y)>>63)&((v|-v)>>63) | (1^(x|-x)>>63)&j.a
	w = w0<<1 | nb>>7
	z, y, v, x = w&j.n, w&j.u1|j.c1, w&j.u2|j.c2, w>>32^j.t
	acc |= ((1^(z|-z)>>63)&((y|-y)>>63)&((v|-v)>>63) | (1^(x|-x)>>63)&j.a) << 1
	w = w0<<2 | nb>>6
	z, y, v, x = w&j.n, w&j.u1|j.c1, w&j.u2|j.c2, w>>32^j.t
	acc |= ((1^(z|-z)>>63)&((y|-y)>>63)&((v|-v)>>63) | (1^(x|-x)>>63)&j.a) << 2
	w = w0<<3 | nb>>5
	z, y, v, x = w&j.n, w&j.u1|j.c1, w&j.u2|j.c2, w>>32^j.t
	acc |= ((1^(z|-z)>>63)&((y|-y)>>63)&((v|-v)>>63) | (1^(x|-x)>>63)&j.a) << 3
	w = w0<<4 | nb>>4
	z, y, v, x = w&j.n, w&j.u1|j.c1, w&j.u2|j.c2, w>>32^j.t
	acc |= ((1^(z|-z)>>63)&((y|-y)>>63)&((v|-v)>>63) | (1^(x|-x)>>63)&j.a) << 4
	w = w0<<5 | nb>>3
	z, y, v, x = w&j.n, w&j.u1|j.c1, w&j.u2|j.c2, w>>32^j.t
	acc |= ((1^(z|-z)>>63)&((y|-y)>>63)&((v|-v)>>63) | (1^(x|-x)>>63)&j.a) << 5
	w = w0<<6 | nb>>2
	z, y, v, x = w&j.n, w&j.u1|j.c1, w&j.u2|j.c2, w>>32^j.t
	acc |= ((1^(z|-z)>>63)&((y|-y)>>63)&((v|-v)>>63) | (1^(x|-x)>>63)&j.a) << 6
	w = w0<<7 | nb>>1
	z, y, v, x = w&j.n, w&j.u1|j.c1, w&j.u2|j.c2, w>>32^j.t
	acc |= ((1^(z|-z)>>63)&((y|-y)>>63)&((v|-v)>>63) | (1^(x|-x)>>63)&j.a) << 7
	return acc
}

// SuiteDeUns rend le nombre de bits a 1 CONSECUTIFS a partir du bit `pos`, borne a `borne` ; les
// bits au-dela du tampon valent zero et arretent donc la suite. PRECONDITION : `pos >= 0`.
//
// C est l autre moitie de la recherche d ancres : une plage de 0xFF porte une SENTINELLE (32 bits
// a 1) a chacune de ses positions de bit, et la recherche les compte une a une. Une suite de `s`
// uns porte exactement `s - 31` sentinelles consecutives.
func SuiteDeUns(d []byte, pos, borne int) int {
	n := 0
	for n < borne {
		k := bits.LeadingZeros64(^BitsAt(d, pos+n, 64))
		n += k
		if k < 64 {
			break
		}
	}
	return min(n, borne)
}
