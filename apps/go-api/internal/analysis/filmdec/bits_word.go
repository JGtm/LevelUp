package filmdec

// bits_word.go — LECTURE DE BITS PAR MOT DE 64.
//
// POURQUOI CE FICHIER EXISTE. Le profil CPU de la cuisson d'un rejeu (2026-09-02,
// `tmp/01e1f945.cpu.prof`, 143 s d'echantillons) place 58 a 61 % du temps TOTAL dans une
// seule fonction : `kfReadBits`, qui lisait ses 32 bits d'identifiant UN BIT A LA FOIS, avec
// un test de borne par bit. Les autres primitives de la meme famille (`BitReader.ReadBits`,
// `readBitsAt`, `PeekBits`) portaient la meme boucle. Toutes lisent desormais leurs bits par
// un mot de 64 bits + deux decalages.
//
// LA SEMANTIQUE HORS TAMPON EST PRESERVEE PAR FONCTION, et c'est la seule chose qui compte
// ici : chaque primitive garde EXACTEMENT le comportement qu'elle avait au-dela (et en deca)
// des bornes de son tampon — `readBitsAt` panique, les trois autres rendent des zeros de
// bourrage. Le chemin rapide n'est donc pris que sur le domaine ou les deux implementations
// coincident trivialement ; tout le reste retombe sur la boucle bit-a-bit d'origine, recopiee
// telle quelle. Les tests differentiels (`bits_word_test.go`) opposent chaque primitive a une
// copie de reference de l'ancienne implementation, sur tampons aleatoires a graine fixee,
// toutes largeurs 0..64, autour de chaque frontiere d'octet, de mot et de fin de tampon,
// cas hors tampon compris.

import "encoding/binary"

// wordBitsAt lit `n` bits (0..64) big-endian, MSB d'abord, a la position bit `pos`, et les
// rend cales a droite. Les bits au-dela de la fin de `buf` valent ZERO — c'est le bourrage
// de queue du moteur.
//
// PRECONDITIONS (l'appelant les garantit, la fonction ne les verifie pas) : `pos >= 0` et
// `n <= 64`. En dehors, les appelants retombent sur leur boucle d'origine, qui seule connait
// leur convention (panique ou zero, troncature au-dela de 64 bits).
// Le corps est volontairement COURT : cette fonction doit rester INLINABLE (elle est
// appelee des dizaines de millions de fois par film, une par position de bit candidate, et
// le seul cout d'appel se verrait au profil). Tout ce qui ne sert que la queue du tampon vit
// donc dans [wordBitsAtTail], hors du budget d'inlining.
func wordBitsAt(buf []byte, pos int, n uint) uint64 {
	i := pos >> 3       // premier octet touche
	sh := uint(pos & 7) // bits a jeter en tete de cet octet
	// Cas dominant, et le seul qui doit tenir dans le budget d'inlining : le champ tient
	// dans le mot de 64 bits qui commence a l'octet i, et ce mot est entierement lisible.
	// `w << sh` amene le bit `pos` en tete du mot.
	if n+sh <= 64 && i+8 <= len(buf) && n > 0 {
		return (binary.BigEndian.Uint64(buf[i:i+8]) << sh) >> (64 - n)
	}
	return wordBitsAtEdge(buf, i, sh, n)
}

// wordBitsAtEdge traite les trois cas que [wordBitsAt] laisse de cote : largeur nulle,
// champ a cheval sur un NEUVIEME octet (`n+sh > 64`, donc n proche de 64 et sh > 0), et
// queue du tampon (moins de huit octets lisibles a partir de l'octet i, les manquants
// valant zero).
func wordBitsAtEdge(buf []byte, i int, sh, n uint) uint64 {
	if n == 0 {
		return 0
	}
	var w uint64
	if i+8 <= len(buf) {
		w = binary.BigEndian.Uint64(buf[i : i+8])
	} else {
		for k := 0; k < 8 && i+k < len(buf); k++ {
			w |= uint64(buf[i+k]) << (56 - 8*uint(k))
		}
	}
	if n+sh <= 64 {
		return (w << sh) >> (64 - n)
	}
	extra := n + sh - 64 // 1..7
	var next uint64
	if i+8 < len(buf) {
		next = uint64(buf[i+8])
	}
	return ((w<<sh)>>sh)<<extra | next>>(8-extra)
}
