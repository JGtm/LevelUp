package source

// octets.go — LES LECTURES D OCTETS DU FILM : ORDRE DES OCTETS, OFFSETS NON ALIGNES, MOTIFS.
//
// # POURQUOI ELLES SONT ICI (lot 2.4.2, ADR 0034 D-2)
//
// « Une seule porte aux octets » ne parle pas que du lecteur de BITS. Avant ce lot, neuf
// paquets appliquaient `binary.LittleEndian` / `binary.BigEndian` a des octets de film chacun
// de son cote — l en-tete du registre, la section d identification, l en-tete de paquet, les
// horodatages, le motif de xuid. Ces lectures vivent desormais ici, et le ratchet
// `archlint/no_raw_film_bytes_outside_source_test.go` refuse `binary.*Endian` dans les racines
// du film.
//
// # LES QUATRE CONVENTIONS DE BORD, ET POURQUOI ELLES NE SE FONDENT PAS EN UNE
//
// Le film est lu par des balayages qui ne se mefient pas des memes choses, et leur convention
// AU BORD du tampon fait partie de la grammaire mesuree : la fondre changerait des valeurs
// decodees, ce que D4 interdit. Elles sont donc NOMMEES, ici, au meme endroit :
//
//	[BitsAt]         pos >= 0, n <= 64. Bourrage a ZERO au-dela de la fin. C est la convention
//	                 du MOTEUR, celle du lecteur canonique [Bits].
//	[BitAt]          un bit, ZERO des deux cotes. Les localisateurs de `killsource` testent le
//	                 bit `s-1` d une position candidate.
//	[BitsTolerants]  ZERO des deux cotes, sur n bits. `weaponv3.ResolveXuidToPI` RECULE de cinq
//	                 bits devant un motif trouve : sur un motif au tout debut du chunk, elle
//	                 passe sous zero.
//	[BitsTronques]   s ARRETE a la fin du tampon SANS bourrer : la valeur rendue ne porte que
//	                 les bits reellement lus, cales a droite. C est la convention du lecteur du
//	                 PIED DE FILM (`objectives`), et elle n est pas interchangeable avec
//	                 [BitsAt] — sur les derniers bits d un bloc, les deux rendent des valeurs
//	                 DIFFERENTES.

import "encoding/binary"

// U16LE / U32LE / U64LE : les entiers little-endian du film, a un offset en OCTETS.
//
// Elles ne verifient pas les bornes, exactement comme `binary.LittleEndian.UintNN` qu elles
// remplacent : l appelant garde SA garde de longueur, et une lecture hors tampon panique au
// meme endroit qu avant.
func U16LE(d []byte, off int) uint16 { return binary.LittleEndian.Uint16(d[off:]) }

// U32LE : cf. [U16LE].
func U32LE(d []byte, off int) uint32 { return binary.LittleEndian.Uint32(d[off:]) }

// U64LE : cf. [U16LE].
func U64LE(d []byte, off int) uint64 { return binary.LittleEndian.Uint64(d[off:]) }

// OctetAuBit lit UN octet a un offset exprime en BITS, alignement quelconque. ZERO quand
// l octet ne tient pas entierement dans le tampon — c est la garde du lecteur du pied de film,
// et elle differe de [BitsAt], qui bourrerait a zero les bits manquants et rendrait donc une
// valeur NON nulle sur une fin de tampon partielle.
func OctetAuBit(d []byte, pos int) byte {
	if pos < 0 || pos+8 > len(d)*8 {
		return 0
	}
	i, sh := pos/8, uint(pos%8)
	if sh == 0 {
		return d[i]
	}
	return d[i]<<sh | d[i+1]>>(8-sh)
}

// U64LEAuBit lit un uint64 little-endian a un offset exprime en BITS : huit [OctetAuBit]
// consecutifs, du moins significatif au plus significatif.
func U64LEAuBit(d []byte, pos int) uint64 {
	var x uint64
	for i := 0; i < 8; i++ {
		x |= uint64(OctetAuBit(d, pos+i*8)) << (uint(i) * 8)
	}
	return x
}

// ChercherMotif64 rend la PREMIERE position de bit `bp` (>= 0, `bp+64 <= len(d)*8`) ou les 64
// bits qui suivent valent `cible`, et si une telle position existe.
//
// C est le balayage de `weaponv3.ResolveXuidToPI` — le xuid d un joueur est ecrit en huit
// octets LITTLE-ENDIAN dans un flux NON aligne sur l octet, donc on cherche au niveau BIT le
// motif obtenu en relisant ces huit octets en big-endian.
//
// IL LIT UN MOT PAR OCTET, pas 64 bits par position : pour chaque octet `i`, le mot big-endian
// `w` donne directement la fenetre du decalage 0, et les sept decalages suivants se deduisent
// de `w` et du seul octet `i+8`. L ORDRE DE PARCOURS EST CELUI D ORIGINE (bp croissant), donc
// la position rendue est exactement celle que rendait la boucle bit a bit — qui pesait 21 a
// 26 % du temps total d une cuisson (profil du 2026-09-02).
func ChercherMotif64(d []byte, cible uint64) (int, bool) {
	n := len(d)
	for i := 0; i+8 <= n; i++ {
		w := binary.BigEndian.Uint64(d[i : i+8])
		if w == cible {
			return i * 8, true
		}
		if i+8 >= n { // les decalages 1..7 exigeraient l octet i+8, hors tampon
			break
		}
		suivant := uint64(d[i+8])
		for s := uint(1); s < 8; s++ {
			if w<<s|suivant>>(8-s) == cible {
				return i*8 + int(s), true
			}
		}
	}
	return 0, false
}
