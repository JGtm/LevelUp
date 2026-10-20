package grammar

// roster_type8.go — LE PAQUET DE TYPE 8 : LA POPULATION DE LA SESSION, ET SON CORPS EST CELUI DE
// LA TABLE DE `chunk_00`.
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
// lire le REGISTRE de `chunk_00` (cf. `masque_cadre_registre_test.go`).
//
// # CE QUE LE TYPE 8 PORTE
//
// `FUN_142987bd4` lit `R(32)` entrees de `0x1440` octets, puis apparie chacune avec la liste VIVE
// des joueurs par la cle de 8 octets `sub+0xcb8` (@142987e64) : il AJOUTE les entrees sans
// correspondant (`FUN_1424d8a8c`, index de manette 0..0x1f), met a jour les autres
// (`FUN_1424d512c`) et RETIRE les joueurs vivants que le paquet ne nomme plus
// (`FUN_142b7f6b8`). C est la population de la session, rafraichie a chaque emission.
//
// # LE CORPS EST DEJA LU PAR LE DEPOT (lot 1.5.2), ET C EST LA MEME GRAMMAIRE
//
//	FUN_142987bd4(session, paquet, version) :
//	  FUN_142988338(session, tampon, taille, 0)   ; memcpy PUR du curseur d octets de la session
//	                                              ; (session+0xf8) — aucune detente
//	  FUN_1424c7b4c(&lecteur, tampon, taille) ; FUN_1406d5cc0(&lecteur, 3)
//	  N = R(32)                                   ; @142987d18 -> FUN_142975788(vec, N)
//	  pour chaque entree :
//	     si version >= 0x15 : FUN_1407f2058 = R(1) porte ; si 0 -> R(5)     ; entree+0x00
//	     FUN_1406d676c(..., entree+0x08, 0x40) = R(64)                      ; entree+0x08
//	     FUN_1407eeba4(entree+0x10, &lecteur)                               ; LE CORPS
//
// et `FUN_1407eeba4` est, champ pour champ et largeur pour largeur, le corps que `FUN_1407edea8`
// ecrit dans la table de 32 joueurs de `chunk_00` : masque `R(11) + 1` (`FUN_142bdeddc` rend
// `R(11) + 1`, `LEA EAX,[R9 + 0x1]` @142bdeea4), les deux listes prefixees `R(12)` et `R(8)`, le
// bloc de 104 octets, le gamertag UTF-16, le bloc de 16 octets, `desired-representation`, la cle
// de 64 bits, les six champs courts (10+14+(6)-1+8+7+1), le bloc de personnalisation et le bloc
// de 44 octets. Ce fichier ne porte donc QUE l en-tete du paquet et celui d une entree : le corps
// passe par [decodeSlotListes] puis [decodeSlotCorps] (`player_table_record.go`), dont l en-tete
// de `player_table.go` porte la provenance de chaque largeur.
//
// DEUX DIFFERENCES AVEC `chunk_00`, LES DEUX MESUREES :
//
//  1. L EN-TETE. `chunk_00` ecrit 85 bits de champs puis le XUID ; le type 8 ecrit la porte
//     `R(1)[R(5)]` (a partir de la version de format `0x15`) puis le XUID, et rien d autre. Il n
//     a pas non plus le `u32` de queue de `slot+0x1448`.
//  2. L ORDRE D OCTETS DU XUID. `FUN_1406d676c` DEPOSE les octets du flux dans l ordre du flux
//     (`*param_3 = BSWAP64(accumulateur)`) et le champ est relu en little-endian : la valeur est
//     l INVERSION D OCTETS de la lecture MSB-first. C est ce qui fait de `entree+0x08` un XUID
//     Xbox lisible (`0x0009...`) au lieu de son image renversee — la table de `chunk_00`, elle,
//     est ecrite par un lecteur de bits et se lit MSB-first ([slotXuidLo], [slotXuidHi]).
//
// ET LA CLE DE JOINTURE `sub+0xcb8` EST LE MEME XUID, dans le meme ordre d octets que
// `entree+0x08` : mesure du 5.17 sur les huit joueurs de `bfecd02b`
// (`cle 0x6945f23ff0010900` <-> `XUID 0x000901f03ff24569`, octet pour octet). Le champ que la
// table de `chunk_00` publie en [types.PlayerSlotShorts.Q64] sans le nommer est donc, lui aussi,
// le XUID vu par `FUN_1406d676c` — ce qui explique que le jeu apparie sur lui.
//
// # LE JEU NE VERIFIE QUE LE DEBORDEMENT (@142987d6e)
//
//	MOV [RBP-0x40],0x5 ; MOV EAX,[RBP-0x48] ; SHL EAX,0x3 ; CMP [RBP-0x34],EAX ; SETG CL
//
// soit `bitsLus > taille*8` (plus un drapeau d erreur de lecteur) -> le paquet est REJETE. Aucune
// exigence de reste nul : le bourrage de queue n est pas relu. [RosterBilan.Reste] est donc
// publie, mais c est le DEBORDEMENT qui est le verdict du jeu.

import (
	"levelup/go-api/internal/games/halo_infinite/film/types"
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
)

// RosterEntry : une entree du paquet de type 8. Le joueur est un [types.PlayerSlot], le MEME type
// que la table de `chunk_00` rend — c est la meme population, lue a un autre endroit du film.
type RosterEntry struct {
	// Joueur : le corps de l entree. `Bit` et `TotalBits` sont relatifs au payload du paquet.
	Joueur types.PlayerSlot
	// Identite : `entree+0x08`, le XUID, en ordre d octets du flux (cf. l en-tete du fichier).
	Identite uint64
	// Option : la valeur derriere la porte `FUN_1407f2058`, ou -1 quand la porte est OUVERTE ou
	// que la version de format ne la porte pas.
	Option int
}

// RosterBilan : ce qu une passe de [DecodeRoster] a consomme. `Debordement` est LE verdict du
// jeu (@142987d6e) ; `Reste` est publie pour la mesure, le jeu ne le relit pas.
type RosterBilan struct {
	Annonce         int // le R(32) de tete
	Entrees         int // les entrees effectivement lues
	Refusees        int // les entrees dont le corps a bute sur la borne du payload
	BitsLus         int
	BitsDisponibles int
	Reste           int
	Debordement     bool
}

// DecodeRoster lit un paquet de type 8 (`FUN_142987bd4`) et rend sa population.
//
// `formatVersion` est la version de format du registre de `chunk_00`
// ([FilmFormatVersionFromHeader]) : c est le 3e argument que le repartiteur passe, et il decide
// de la porte de tete de chaque entree. `persoBits` est la largeur du bloc de personnalisation
// pour le build du film — la MEME valeur que la table de `chunk_00` emploie
// ([PlayerTableReport.PersoBytes] x 8).
func DecodeRoster(pay []byte, formatVersion, persoBits int) ([]RosterEntry, RosterBilan) {
	fin := len(pay) * 8
	r := &slotReader{br: LecteurSur(pay), fin: fin, ok: true}
	b := RosterBilan{BitsDisponibles: fin}
	//nolint:gosec // le cardinal est un u32 du film ; la boucle est bornee par la borne du lecteur
	b.Annonce = int(uint32(r.bits(rosterCountBits)))
	var out []RosterEntry
	for i := 0; i < b.Annonce && r.ok; i++ {
		e, ok := rosterLireEntree(r, formatVersion, persoBits)
		if !ok {
			b.Refusees++
			break
		}
		out = append(out, e)
		b.Entrees++
	}
	b.BitsLus = r.br.BitPos()
	b.Reste = b.BitsDisponibles - b.BitsLus
	b.Debordement = b.Reste < 0 || !r.ok
	return out, b
}

// rosterLireEntree porte UNE entree : la porte de version, le XUID, puis le corps commun.
func rosterLireEntree(r *slotReader, formatVersion, persoBits int) (RosterEntry, bool) {
	e := RosterEntry{Option: -1}
	debut := r.br.BitPos()
	if formatVersion >= rosterOptGateVersion && r.bits(1) == 0 {
		//nolint:gosec // R(5) : cinq bits, jamais negatif
		e.Option = int(r.bits(rosterOptWidth))
	}
	e.Identite = rosterOctetsDuFlux(r.bits(slotXUIDBits), slotXUIDBits)
	var enr slotEnr
	if !decodeSlotListes(r, &enr) || !decodeSlotCorps(r, &enr, persoBits) {
		return e, false
	}
	enr.slot.XUID = e.Identite
	enr.slot.Bit = debut
	enr.slot.TotalBits = r.br.BitPos() - debut
	e.Joueur = enr.slot
	return e, r.ok
}

// rosterOctetsDuFlux rend la valeur d un champ ecrit par `FUN_1406d676c` : ce lecteur depose les
// octets du flux dans l ORDRE DU FLUX (`*param_3 = BSWAP64(accumulateur)`) et le champ de
// structure est relu en little-endian, donc la valeur est l inversion d octets de la lecture
// MSB-first. Reserve aux largeurs multiples de 8.
func rosterOctetsDuFlux(v uint64, bits uint) uint64 {
	var out uint64
	for i := uint(0); i < bits/8; i++ {
		// L octet `i` du FLUX est le `i`-eme en partant du haut de la lecture MSB-first ; il pese
		// `8*i` dans le champ relu en little-endian.
		out |= ((v >> (bits - 8*(i+1))) & 0xff) << (8 * i)
	}
	return out
}
