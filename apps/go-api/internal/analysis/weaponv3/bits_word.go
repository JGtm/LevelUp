package weaponv3

// bits_word.go — LECTURE DE BITS PAR MOT DE 64, pour la resolution xuid -> player_index.
//
// POURQUOI. Le profil CPU de la cuisson d'un rejeu (2026-09-02) place 21 a 26 % du temps
// TOTAL dans `bitReader.bit` / `bitReader.readBits`, appeles par [ResolveXuidToPI] : le
// balayage testait un motif de 64 bits a CHAQUE position de bit du chunk, en relisant ces
// 64 bits UN PAR UN. Le balayage de `playerIndices` pesait a lui seul 38,5 s sur 96 s de
// cuisson (`tmp/refL2_01e1f945.log`).
//
// La semantique est INCHANGEE : bourrage a zero au-dela de la fin du tampon, et la boucle
// d'origine reste seule maitresse des positions negatives (cf. `readBits`).

import "encoding/binary"

// wordBitsAt lit `n` bits (0..64) MSB d'abord a la position bit `pos` (>= 0), cales a
// droite ; les bits au-dela de la fin de `buf` valent zero.
func wordBitsAt(buf []byte, pos int, n uint) uint64 {
	if n == 0 {
		return 0
	}
	i := pos >> 3
	sh := uint(pos & 7)
	var w uint64
	switch {
	case i+8 <= len(buf):
		w = binary.BigEndian.Uint64(buf[i : i+8])
	case i < len(buf):
		for k := 0; k < 8 && i+k < len(buf); k++ {
			w |= uint64(buf[i+k]) << (56 - 8*uint(k))
		}
	}
	if avail := 64 - sh; n <= avail {
		return (w << sh) >> (64 - n)
	}
	extra := n + sh - 64
	var next uint64
	if i+8 < len(buf) {
		next = uint64(buf[i+8])
	}
	return ((w<<sh)>>sh)<<extra | next>>(8-extra)
}

// findPattern64 rend la PREMIERE position de bit `bp` (>= 0, `bp+64 <= len(data)*8`) ou les
// 64 bits qui suivent valent `target`, et si une telle position existe.
//
// C'est le balayage de [ResolveXuidToPI], reecrit pour lire UN MOT par octet au lieu de 64
// bits par position : pour chaque octet `i`, le mot big-endian `w` donne directement la
// fenetre du decalage 0, et les sept decalages suivants se deduisent de `w` et du seul octet
// `i+8`. L'ORDRE DE PARCOURS EST CELUI D'ORIGINE (bp croissant), donc la position rendue est
// exactement celle que rendait la boucle bit a bit.
func findPattern64(data []byte, target uint64) (int, bool) {
	n := len(data)
	for i := 0; i+8 <= n; i++ {
		w := binary.BigEndian.Uint64(data[i : i+8])
		if w == target {
			return i * 8, true
		}
		if i+8 >= n { // les decalages 1..7 exigeraient l'octet i+8, hors tampon
			break
		}
		next := uint64(data[i+8])
		for s := uint(1); s < 8; s++ {
			if w<<s|next>>(8-s) == target {
				return i*8 + int(s), true
			}
		}
	}
	return 0, false
}
