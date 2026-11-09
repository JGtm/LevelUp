package grammar

// instruments_partages_section3_test.go — la grammaire de la section 3 (`s3*`) des
// instruments du roster, qu'appellent les lecteurs de profil dont se servent les gardes.
// Deplaces tels quels au J12.7 bis depuis les fichiers tagues `research` (decision DU-5 : le
// tag cache les instruments, jamais une garde) ; chaque declaration garde le corps et le
// commentaire de son fichier d'origine.

import (
	"encoding/binary"
	"strings"
	"unicode/utf16"
)

const (
	// s3rXuidLo / s3rXuidHi : la plage des XUID Xbox Live. Bornes ecrites avant la mesure ;
	// elles servent AUSSI a fabriquer les leurres du controle R-NEG, pour que ceux-ci aient
	// exactement la meme forme que les vraies valeurs cherchees.
	s3rXuidLo = uint64(0x0009000000000000)
	s3rXuidHi = uint64(0x000A000000000000)
	// s3rEnteteBits : les 85 bits d'en-tete qui precedent l'entier de 64 bits (1+1+1+32+2+48).
	s3rEnteteBits = 85
	// s3rCorpsBit : borne basse du balayage, le debut du corps dans le flux.
	s3rCorpsBit = s3wCorpsOff * 8
	// s3rEcartMax : au-dela de cet ecart en bits, deux touches n'appartiennent pas a la meme
	// grappe d'enregistrements. Les longueurs mesurees vont de 16 611 a 28 145 bits ; le seuil
	// est pose a 40 000, soit 1,4 fois la plus longue.
	s3rEcartMax = 40000
	s3rLeurres  = 40
)

// Offsets de la tete, lus dans FUN_14299b198. Ils sont en OCTETS du tampon inflate.
const (
	s3wRegistreDebut = 0x000008 // le registre commence ici, pas a 0
	s3wRegistreBits  = 0x659000 // 6 656 000 bits = 832 000 octets = 50 blocs de 0x4100
	s3wTableDebut    = 0x0CB208 // la table par type
	s3wTableBits     = 0xF60    // 3 936 bits = 492 octets = 123 u32
	s3wVersionOff    = 0x0CB3F4 // trois champs de 32 octets
	s3wBuildOff      = 0x0CB414
	s3wSaveurOff     = 0x0CB434
	s3wU32aOff       = 0x0CB454
	s3wU32bOff       = 0x0CB458
	s3wBoolOff       = 0x0CB45C // UN bit : tout ce qui suit est decale
	s3wNomBits       = 0x800    // 256 octets par champ de nom, deux champs
	s3wCorpsOff      = 0x0CE68C // debut du corps dans le FLUX (FUN_1407ec560) : struct 0x0CE690 - 4
)

// s3rTouche : une occurrence d'entier de 64 bits precedee d'un en-tete conforme.
type s3rTouche struct {
	bit   int
	xuid  uint64
	token uint64 // le champ de 48 bits a slot+0x09
}

// s3rBit lit n bits MSB-first.
func s3rBit(d []byte, bit, n int) uint64 {
	var acc uint64
	for k := range n {
		b := bit + k
		if b>>3 >= len(d) {
			return 0
		}
		acc = acc<<1 | uint64((d[b>>3]>>(7-uint(b&7)))&1)
	}
	return acc
}

// s3bBuild cherche le prefixe `HI_` dans la zone d'en-tete et rend (chaine, offset). Rend
// ("", -1) si le film ne porte pas de section d'identification.
func s3bBuild(d []byte) (string, int) {
	if len(d) < s3bEnteteFin {
		return "", -1
	}
	for off := 0x20; off < s3bEnteteFin; off++ {
		if d[off] == 'H' && d[off+1] == 'I' && d[off+2] == '_' {
			return s3wChaine(d, off, 32), off
		}
	}
	return "", -1
}

// s3sEnr : un enregistrement de slot decode champ par champ.
type s3sEnr struct {
	debut, total            int // position et longueur en BITS dans le flux
	xuid                    uint64
	jeton48                 uint64
	compteMasque, popMasque int
	n, m                    int
	gamertag                string
	repr                    uint32 // "desired-representation"
	q64                     uint64
	f10, f14, f8, f7, f1    uint32
	f6                      int // FUN_1407eddb4 ecrit VALEUR+1 : on rend la valeur signee
	b2                      uint32
	u32Tete                 uint32
	perso                   []byte // 1 852 octets : sub+0xcc0
	bloc104, bloc16, bloc44 []byte
	mots                    []uint32
	masque                  []byte // un octet par bit du masque de presence (sub+0x000)
	octets                  []byte // la liste de N octets (sub+0x108)
}

// s3sDecode decode un enregistrement a partir de son PREMIER bit. Rend nil si le flux est court.
func s3sDecode(d []byte, debut, fin int) *s3sEnr {
	if debut < 0 || debut+s3sFixeSlot > fin {
		return nil
	}
	e := &s3sEnr{debut: debut}
	p := debut
	if s3rBit(d, p, 3) != 4 {
		return nil
	}
	p += 3
	e.u32Tete = uint32(s3rBit(d, p, 32))
	p += 32
	e.b2 = uint32(s3rBit(d, p, 2))
	p += 2
	e.jeton48 = s3rBit(d, p, 48)
	p += 48
	e.xuid = s3rBit(d, p, 64)
	p += 64
	p = s3sDecodeListes(d, e, p, fin)
	if p < 0 {
		return nil
	}
	p = s3sDecodeQueue(d, e, p, fin)
	if p < 0 {
		return nil
	}
	e.total = p - debut
	return e
}

// s3sPredite rend la longueur que la grammaire PREDIT pour un enregistrement, a partir des
// quatre nombres lus dans le flux. C'est le coeur du controle G-CLO.
func s3sPredite(e *s3sEnr) int {
	unites := min(
		// le NUL terminateur est ecrit
		len(utf16.Encode([]rune(e.gamertag)))+1,
		// chaine pleine : FUN_1407ece18 s'arrete sur la borne, sans NUL
		s3sGtMax)
	return s3sFixeSlot + s3sFixeSub + e.compteMasque + e.n*8 + e.m*32 + unites*16
}

// s3sImprimable : trois caracteres ASCII imprimables au moins, rien d'autre.
//
// DELEGUE DEPUIS LE LOT 1.5 : le meme filtre de parasite est devenu du code de PRODUCTION
// (`gamertagImprimable`, player_table.go). L'instrument est l'ORACLE du lecteur : si les deux
// divergeaient d'un caractere, l'oracle cesserait de mesurer ce que le lecteur fait.
func s3sImprimable(s string) bool { return gamertagImprimable(s) }

// s3wChaine lit une chaine ASCII terminee par NUL dans un champ de largeur fixe.
func s3wChaine(d []byte, off, max int) string {
	if off < 0 || off >= len(d) {
		return ""
	}
	end := min(off+max, len(d))
	s := string(d[off:end])
	if i := strings.IndexByte(s, 0); i >= 0 {
		s = s[:i]
	}
	return s
}

// s3bEnteteFin : borne haute de la recherche de la chaine de build. Elle s'arrete AVANT le
// corps (`0x0CE68C`) : le corps est bit-packe et une occurrence de `HI_` y serait fortuite.
// La borne basse est l'octet 0x20, comme chez `lireEntete` — un `HI_` plus tot ne peut pas
// avoir de champ version devant lui.
const s3bEnteteFin = 0x0CE68C

const (
	// Largeurs lues au desassemblage. Aucune n'est devinee.
	s3sPrefixeMasque = 11    // FUN_1424ccf94 : rang du bit haut du masque de 2 048 bits
	s3sLargeurN      = 12    // FUN_1411b1a24 : longueur de la liste d'octets
	s3sLargeurM      = 8     // FUN_1411b198c : longueur de la liste de mots de 32 bits
	s3sBloc104       = 832   // 0x340 bits, sub+0xc48
	s3sGtMax         = 16    // FUN_1407ece18(…, 0x10)
	s3sBloc16        = 128   // 0x80 bits, sub+0xc38
	s3sPersoBits     = 14816 // 0x39e0 bits, sub+0xcc0 : LE BLOC DE PERSONNALISATION
	s3sPersoOctets   = s3sPersoBits / 8
	s3sBloc44        = 352 // 0x160 bits, sub+0x1400
	// s3sFixeSub : tout ce que le sous-enregistrement ecrit hors des trois listes et de la
	// chaine. 11+12+8 (prefixes) + 832 + 128 + 32 + 64 + 10 + 14 + 6 + 8 + 7 + 1 + 14816 + 352.
	s3sFixeSub = s3sPrefixeMasque + s3sLargeurN + s3sLargeurM + s3sBloc104 + s3sBloc16 +
		32 + 64 + 10 + 14 + 6 + 8 + 7 + 1 + s3sPersoBits + s3sBloc44
	// s3sFixeSlot : l'en-tete (85) + le XUID (64) + le u32 de queue a slot+0x1448.
	s3sFixeSlot = s3rEnteteBits + 64 + 32
)

// s3sDecodeListes lit le masque de presence et les deux listes prefixees, puis le bloc de 104 o
// et la chaine UTF-16. Rend la position suivante, ou -1 si le flux est trop court.
func s3sDecodeListes(d []byte, e *s3sEnr, p, fin int) int {
	e.compteMasque = int(s3rBit(d, p, s3sPrefixeMasque)) + 1
	p += s3sPrefixeMasque
	if e.compteMasque > 2048 {
		return -1
	}
	e.masque = make([]byte, e.compteMasque)
	for k := 0; k < e.compteMasque; k++ {
		e.masque[k] = byte(s3rBit(d, p+k, 1))
		if e.masque[k] == 1 {
			e.popMasque++
		}
	}
	p += e.compteMasque
	e.n = int(s3rBit(d, p, s3sLargeurN))
	p += s3sLargeurN
	if e.n > 2048 {
		return -1
	}
	e.octets = s3sOctets(d, p, e.n)
	p += e.n * 8
	e.m = int(s3rBit(d, p, s3sLargeurM))
	p += s3sLargeurM
	if e.m > 192 {
		return -1
	}
	e.mots = make([]uint32, e.m)
	for k := 0; k < e.m; k++ {
		e.mots[k] = binary.LittleEndian.Uint32(s3sOctets(d, p+k*32, 4))
	}
	p += e.m * 32
	if p+s3sBloc104 > fin {
		return -1
	}
	e.bloc104 = s3sOctets(d, p, s3sBloc104/8)
	p += s3sBloc104
	var unites []uint16
	for range s3sGtMax {
		u := uint16(s3rBit(d, p, 16))
		p += 16
		if u == 0 {
			break
		}
		unites = append(unites, u)
	}
	e.gamertag = string(utf16.Decode(unites))
	return p
}

// s3sDecodeQueue lit tout ce qui suit la chaine : les six champs courts, le bloc de
// personnalisation de 1 852 octets et les deux blocs de queue.
func s3sDecodeQueue(d []byte, e *s3sEnr, p, fin int) int {
	if p+s3sBloc16+32+64+46+s3sPersoBits+s3sBloc44+32 > fin {
		return -1
	}
	e.bloc16 = s3sOctets(d, p, s3sBloc16/8)
	p += s3sBloc16
	e.repr = uint32(s3rBit(d, p, 32))
	p += 32
	e.q64 = s3rBit(d, p, 64)
	p += 64
	e.f10 = uint32(s3rBit(d, p, 10))
	p += 10
	e.f14 = uint32(s3rBit(d, p, 14))
	p += 14
	e.f6 = int(s3rBit(d, p, 6)) - 1
	p += 6
	e.f8 = uint32(s3rBit(d, p, 8))
	p += 8
	e.f7 = uint32(s3rBit(d, p, 7))
	p += 7
	e.f1 = uint32(s3rBit(d, p, 1))
	p++
	e.perso = s3sOctets(d, p, s3sPersoOctets)
	p += s3sPersoBits
	e.bloc44 = s3sOctets(d, p, s3sBloc44/8)
	p += s3sBloc44
	p += 32 // le u32 de slot+0x1448
	return p
}

// s3sOctets lit n octets du flux : `FUN_1406d60f4` pousse les octets de la source DANS L'ORDRE
// DES ADRESSES, MSB d'abord, donc n octets relus MSB-first rendent l'image memoire verbatim.
func s3sOctets(d []byte, bit, n int) []byte {
	out := make([]byte, n)
	for i := range n {
		out[i] = byte(s3rBit(d, bit+i*8, 8))
	}
	return out
}
