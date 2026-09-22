package grammar

// roster_type8.go — LE PAQUET DE TYPE 8, LU CHEZ L ECRIVAIN : C EST LA LISTE DES JOUEURS DE LA
// SESSION, PAS UNE TABLE DE COMPOSANTS.
//
// # LE REPARTITEUR, ET CE QU IL PASSE AU LECTEUR
//
// `FUN_1428e22c0` aiguille sur `*(short *)paquet` (le type de l en-tete de 16 octets) ; sa
// branche `sVar2 == 8` est la seule qui interroge `&DAT_144c23178` :
//
//	cVar3 = FUN_1428e1e94(&DAT_144c23178)                 ; ou vit la structure du film charge
//	lVar7 = cVar3 ? *(param_1+0x120) + 0x130 : *(param_1+0x108)
//	FUN_142987bd4(session, paquet, *(u32 *)(lVar7 + 4))   ; le 3e argument = LA VERSION DE FORMAT
//
// `*(u32 *)(lVar7 + 4)` est exactement ce que rend `FUN_1428e1c0c` — la VERSION DE FORMAT du
// registre inflate de `chunk_00`, que le depot lit par [FilmFormatVersionFromHeader]
// (`chunk_00 + 4`). `&DAT_144c23178` n est donc PAS un porteur de table de composants : c est le
// singleton du film charge, et la branche de type 8 ne lui demande QUE l adresse de cette
// structure. Le filtre de composants de `FUN_14076cb60` interroge le MEME singleton, mais pour y
// lire le REGISTRE de `chunk_00` (cf. `registry.go` et la note 5.17).
//
// # LA GRAMMAIRE, LARGEUR PAR LARGEUR (toutes lues au desassemblage, aucune inventee)
//
//	FUN_142987bd4(session, paquet, version) :
//	  FUN_142988338(session, tampon, taille, 0)     ; memcpy PUR du curseur d octets de la
//	                                                ; session (session+0xf8) — aucune detente
//	  FUN_1424c7b4c(&lecteur, tampon, taille) ; FUN_1406d5cc0(&lecteur, 3)
//	  N = R(32)                                     ; @142987d18 -> FUN_142975788(vec, N)
//	  pour chaque entree (stride 0x1440 octets) :
//	     si version >= 0x15 : FUN_1407f2058 = R(1) porte ; si 0 -> R(5)     ; entree+0x00
//	     FUN_1406d676c(..., entree+0x08, 0x40) = R(64)                      ; entree+0x08
//	     FUN_1407eeba4(entree+0x10, &lecteur) :
//	        FUN_1407f01c4 :  n  = R(11)   (FUN_142bdeddc) ; n x R(1)
//	                         L1 = R(12)   (FUN_1411b1bd8) ; R(L1 * 8)
//	                         L2 = R(8)    (FUN_1411b1b04) ; R(L2 * 32)
//	        R(832)                                                          ; rec+0xc48
//	        FUN_1407f0094(..., rec+0xc14, 0x10) : jusqu a 16 x R(16), ARRET APRES le mot NUL
//	        R(128)                                                          ; rec+0xc38
//	        FUN_14080dec4 = R(32)  (« desired-representation »)             ; rec+0xcb0
//	        R(64)                                                           ; rec+0xcb8
//	        FUN_1407effb8 = R(10)                                           ; rec+0xc12
//	        FUN_1407efedc = R(14)                                           ; rec+0xc36
//	        FUN_1407ef724 = R(6), valeur - 1                                ; rec+0xc35
//	        R(8) inline                                                     ; rec+0xc10
//	        FUN_1407eed64 = R(7)                                            ; rec+0xc34
//	        FUN_1406cf008 = R(1)                                            ; rec+0xc11
//	        R(14816)                                                        ; rec+0xcc0
//	        R(352)                                                          ; rec+0x1400
//
// LA CLE DE JOINTURE EST `rec+0xcb8` : apres la boucle, `FUN_142987bd4` apparie chaque entree du
// paquet avec la liste VIVE des joueurs par `*(longlong *)(entree+0x10+0xcb8) ==
// *(longlong *)(objet+0xcb8)` (@142987e64), ajoute les entrees sans correspondant
// (`FUN_1424d8a8c`, index de manette 0..0x1f), met a jour les autres (`FUN_1424d512c`) et RETIRE
// les joueurs vivants absents du paquet (`FUN_142b7f6b8`). Le type 8 est donc la population de la
// session, rafraichie a chaque emission.
//
// # LE JEU NE VERIFIE QUE LE DEBORDEMENT (@142987d6e)
//
//	MOV [RBP-0x40],0x5 ; MOV EAX,[RBP-0x48] ; SHL EAX,0x3 ; CMP [RBP-0x34],EAX ; SETG CL
//
// soit `bitsLus > taille*8` (plus un drapeau d erreur de lecteur) -> le paquet est REJETE. Aucune
// exigence de reste nul : le bourrage de queue n est pas relu. [RosterBilan.Reste] est donc
// publie, mais c est le DEBORDEMENT qui est le verdict du jeu.

import (
	"unicode/utf16"
)

// PacketTypeRoster : le type d en-tete du paquet de population de session (`FUN_1428e22c0`,
// branche 8). Les deux autres types deja nommes du depot sont [PacketTypeDelta] et
// [PacketTypeKeyframe].
const PacketTypeRoster uint16 = 8

const (
	// rosterCountBits : le mot de tete, lu inline @142987c5a..142987d18 puis passe a
	// `FUN_142975788` comme cardinal du vecteur d entrees.
	rosterCountBits = 32
	// rosterOptGateVersion : la version de format a partir de laquelle chaque entree porte la
	// porte `FUN_1407f2058` (`CMP R12D,0x15 ; JL` @142987d37).
	rosterOptGateVersion = 0x15
	// rosterOptWidth : la largeur derriere la porte de `FUN_1407f2058` (`ADD [param_1+0x2c],5`).
	rosterOptWidth = 5
	// rosterIdentityBits : `FUN_1406d676c(..., entree+0x08, 0x40)` @142987d4c.
	rosterIdentityBits = 64
	// rosterFlagCountBits : `FUN_142bdeddc` — le cardinal du champ de drapeaux de bits, rendu
	// `R(11) + 1`. Le `+ 1` n est pas un detail : sans lui tout ce qui suit dans l entree glisse
	// d un bit, et les deux longueurs prefixees sortent de leurs bornes de structure (mesure du
	// 5.17 sur `bfecd02b` : L1 = 2 399 pour un champ de 2 048 octets).
	rosterFlagCountBits = 11
	// rosterByteLenBits : `FUN_1411b1bd8` — la longueur, EN OCTETS, du premier tableau.
	rosterByteLenBits = 12
	// rosterWordLenBits : `FUN_1411b1b04` — la longueur, EN MOTS DE 32 BITS, du second tableau.
	rosterWordLenBits = 8
	// rosterBlobABits : `FUN_1406d676c(..., rec+0xc48, 0x340)`.
	rosterBlobABits = 0x340
	// rosterLabelMaxWords : `MOV R9D,0x10` @1407eebd5 — le cardinal passe a `FUN_1407f0094`.
	rosterLabelMaxWords = 0x10
	// rosterLabelUnitBits : une unite de la chaine, `ADD [RBX+0x2c],0x10` @1407f00e9.
	rosterLabelUnitBits = 16
	// rosterBlobBBits : `FUN_1406d676c(..., rec+0xc38, 0x80)`.
	rosterBlobBBits = 0x80
	// rosterRepresentationBits : `FUN_14080dec4` (« desired-representation »), R(32) plein.
	rosterRepresentationBits = 32
	// rosterKeyBits : `FUN_1406d676c(..., rec+0xcb8, 0x40)` — LA CLE DE JOINTURE.
	rosterKeyBits = 64
	// rosterF10Bits, rosterF14Bits, rosterF6Bits, rosterF7Bits : les quatre lecteurs etroits
	// (`FUN_1407effb8`, `FUN_1407efedc`, `FUN_1407ef724`, `FUN_1407eed64`).
	rosterF10Bits = 10
	rosterF14Bits = 14
	rosterF6Bits  = 6
	rosterF7Bits  = 7
	// rosterByteBits : le R(8) inline @1407eec65.
	rosterByteBits = 8
	// rosterBlobCBits : `FUN_1406d676c(..., rec+0xcc0, 0x39e0)`.
	rosterBlobCBits = 0x39e0
	// rosterBlobDBits : la queue, `FUN_1406d676c(..., rec+0x1400, 0x160)` en appel terminal.
	rosterBlobDBits = 0x160
)

// RosterEntry : une entree du paquet de type 8 — un joueur de la session. Seuls les champs que la
// suite de `FUN_142987bd4` CONSULTE sont nommes ; les trois blocs opaques (`0x340`, `0x39e0`,
// `0x160` bits) sont consommes et non publies, parce qu aucun lecteur du jeu ne les redecoupe a
// cet endroit.
type RosterEntry struct {
	// Cle : `rec+0xcb8`, R(64). C est la cle d appariement avec la liste vive (@142987e64) et
	// avec l objet joueur (`objet+0xcb8`).
	Cle uint64
	// Identite : `entree+0x08`, R(64), lu AVANT le corps.
	Identite uint64
	// Etiquette : `rec+0xc14`, chaine UTF-16 NUL-terminee d au plus 16 unites.
	Etiquette string
	// Representation : `rec+0xcb0`, le R(32) de « desired-representation ».
	Representation uint32
	// Option : la valeur derriere la porte `FUN_1407f2058` (version >= 0x15), ou -1 quand la
	// porte est fermee ou que la version ne la porte pas.
	Option int
	// F10, F14, F6, Octet, F7, Bit : les six champs etroits, dans l ordre de lecture.
	F10   uint16
	F14   uint16
	F6    uint8
	Octet uint8
	F7    uint8
	Bit   bool
	// Drapeaux : les `n` bits de `FUN_1407f01c4`, `n` lu en R(11).
	Drapeaux []bool
	// Octets, Mots : les deux tableaux a longueur prefixee de `FUN_1407f01c4`.
	Octets []byte
	Mots   []uint32
}

// RosterBilan : ce qu une passe de [DecodeRoster] a consomme. `Debordement` est LE verdict du
// jeu (@142987d6e) ; `Reste` est publie pour la mesure, le jeu ne le relit pas.
type RosterBilan struct {
	Annonce         int // le R(32) de tete
	Entrees         int // les entrees effectivement lues
	BitsLus         int
	BitsDisponibles int
	Reste           int
	Debordement     bool
}

// DecodeRoster lit un paquet de type 8 (`FUN_142987bd4`) et rend sa population.
//
// `formatVersion` est la version de format du registre de `chunk_00`
// ([FilmFormatVersionFromHeader]) : c est le 3e argument que le repartiteur passe, et il decide
// de la porte de tete de chaque entree. La lecture s arrete des que le curseur sort du payload —
// le jeu rejette le paquet entier dans ce cas, et le bilan le dit.
func DecodeRoster(pay []byte, formatVersion int) ([]RosterEntry, RosterBilan) {
	br := LecteurSur(pay)
	b := RosterBilan{BitsDisponibles: len(pay) * 8}
	//nolint:gosec // le cardinal est un u32 du film ; la boucle est bornee par le payload
	b.Annonce = int(uint32(br.ReadBits(rosterCountBits)))
	out := make([]RosterEntry, 0, rosterCap(b.Annonce, len(pay)))
	for i := 0; i < b.Annonce; i++ {
		if br.Remaining() <= 0 {
			break
		}
		out = append(out, rosterLireEntree(br, formatVersion))
		b.Entrees++
	}
	b.BitsLus = br.BitPos()
	b.Reste = b.BitsDisponibles - b.BitsLus
	b.Debordement = b.Reste < 0
	return out, b
}

// rosterCap borne la pre-allocation par ce que le payload peut PHYSIQUEMENT porter : le cardinal
// annonce est un mot du film, et un mot aberrant ne doit pas reserver un gigaoctet.
func rosterCap(annonce, nOctets int) int {
	const minBitsParEntree = rosterIdentityBits + rosterFlagCountBits + rosterByteLenBits +
		rosterWordLenBits + rosterBlobABits + rosterLabelUnitBits + rosterBlobBBits +
		rosterRepresentationBits + rosterKeyBits + rosterF10Bits + rosterF14Bits + rosterF6Bits +
		rosterByteBits + rosterF7Bits + 1 + rosterBlobCBits + rosterBlobDBits
	plafond := nOctets*8/minBitsParEntree + 1
	if annonce < plafond {
		return annonce
	}
	return plafond
}

// rosterLireEntree porte UNE entree : la porte de version, l identite, puis `FUN_1407eeba4`.
func rosterLireEntree(br *Lecteur, formatVersion int) RosterEntry {
	e := RosterEntry{Option: -1}
	if formatVersion >= rosterOptGateVersion && !br.ReadBit() {
		//nolint:gosec // R(5) : cinq bits, jamais negatif
		e.Option = int(br.ReadBits(rosterOptWidth))
	}
	e.Identite = rosterLireMot(br, rosterIdentityBits)
	rosterLireCorps(br, &e)
	return e
}

// rosterLireCorps porte `FUN_1407eeba4` dans son ordre exact.
func rosterLireCorps(br *Lecteur, e *RosterEntry) {
	rosterLirePrefixes(br, e)
	br.Skip(rosterBlobABits)
	e.Etiquette = rosterLireEtiquette(br)
	br.Skip(rosterBlobBBits)
	//nolint:gosec // R(32) dans un u32
	e.Representation = uint32(br.ReadBits(rosterRepresentationBits))
	e.Cle = rosterLireMot(br, rosterKeyBits)
	//nolint:gosec // R(10) tient dans un u16
	e.F10 = uint16(br.ReadBits(rosterF10Bits))
	//nolint:gosec // R(14) tient dans un u16
	e.F14 = uint16(br.ReadBits(rosterF14Bits))
	// `FUN_1407ef724` DECREMENTE sa valeur (`DEC R9B` @1407ef764) : le champ porte `R(6) - 1`,
	// modulo 256 comme l octet du jeu.
	//nolint:gosec // R(6) dans un u8
	e.F6 = uint8(br.ReadBits(rosterF6Bits)) - 1
	//nolint:gosec // R(8) dans un u8
	e.Octet = uint8(br.ReadBits(rosterByteBits))
	//nolint:gosec // R(7) dans un u8
	e.F7 = uint8(br.ReadBits(rosterF7Bits))
	e.Bit = br.ReadBit()
	br.Skip(rosterBlobCBits)
	br.Skip(rosterBlobDBits)
}

// rosterLireMot porte UN mot de `FUN_1406d676c` : ce lecteur DEPOSE LES OCTETS DU FLUX DANS
// L ORDRE DU FLUX (`*param_3 = BSWAP64(accumulateur)`, @1406d67cb), et le champ de structure est
// relu en LITTLE-ENDIAN. La valeur du champ est donc l INVERSION D OCTETS de la lecture
// MSB-first. C est ce qui fait de `entree+0x08` un XUID Xbox lisible (`0x0009...`) au lieu de son
// image renversee — mesure du 5.17 sur `dad793c7`.
//
// Reserve aux largeurs multiples de 8 : les six champs etroits de l entree passent par des
// lecteurs DEDIES (`FUN_1407effb8` et compagnie) qui, eux, stockent la valeur MSB-first telle
// quelle, et `FUN_14080dec4` fait de meme pour son R(32).
func rosterLireMot(br *Lecteur, bits uint) uint64 {
	v := br.ReadBits(bits)
	var out uint64
	for i := uint(0); i < bits/8; i++ {
		// L octet `i` du FLUX est le `i`-eme en partant du haut de la lecture MSB-first ; il pese
		// `8*i` dans le champ relu en little-endian.
		out |= ((v >> (bits - 8*(i+1))) & 0xff) << (8 * i)
	}
	return out
}

// rosterLirePrefixes porte `FUN_1407f01c4` : un champ de bits a cardinal lu, puis deux tableaux a
// longueur prefixee (octets, puis mots de 32 bits).
func rosterLirePrefixes(br *Lecteur, e *RosterEntry) {
	// `FUN_142bdeddc` rend `R(11) + 1` (`LEA EAX,[R9 + 0x1]` @142bdeea4) : le cardinal est
	// TOUJOURS au moins 1, et son maximum 2048 remplit EXACTEMENT les 0x100 octets de bitmap que
	// `FUN_1407f01c4` met a zero avant de la remplir par `FUN_1407688b0(dest, k, bit)`.
	n := int(br.ReadBits(rosterFlagCountBits)) + 1
	e.Drapeaux = make([]bool, n)
	for k := 0; k < n; k++ {
		e.Drapeaux[k] = br.ReadBit()
	}
	l1 := int(br.ReadBits(rosterByteLenBits))
	e.Octets = make([]byte, l1)
	for k := 0; k < l1; k++ {
		//nolint:gosec // R(8) dans un octet
		e.Octets[k] = uint8(br.ReadBits(8))
	}
	l2 := int(br.ReadBits(rosterWordLenBits))
	e.Mots = make([]uint32, l2)
	for k := 0; k < l2; k++ {
		//nolint:gosec // R(32) dans un u32
		e.Mots[k] = uint32(br.ReadBits(32))
	}
}

// rosterLireEtiquette porte `FUN_1407f0094(..., 0x10)` : au plus 16 unites de 16 bits, et la
// lecture S ARRETE APRES l unite NULLE (@1407f010a : `JNZ` continue, la chute sort). Une chaine
// pleine de 16 unites non nulles consomme donc 256 bits et n a pas de terminateur.
func rosterLireEtiquette(br *Lecteur) string {
	u := make([]uint16, 0, rosterLabelMaxWords)
	for k := 0; k < rosterLabelMaxWords; k++ {
		//nolint:gosec // R(16) dans un u16
		w := uint16(br.ReadBits(rosterLabelUnitBits))
		if w == 0 {
			break
		}
		u = append(u, w)
	}
	return string(utf16.Decode(u))
}
