package source

// bits.go — LE LECTEUR DE BITS CANONIQUE : LA SEULE PORTE AUX BITS D UN FILM.
//
// # POURQUOI IL VIT ICI (lot 2.4, ADR 0034 D-2, arbitrage V15 (1))
//
// Avant le lot 2.4, SEPT lecteurs de bits distincts lisaient les memes octets de film, chacun
// avec sa propre idee du bourrage de queue, du debordement et de l ordre des bits :
// `grammar.Lecteur`, `killsource.evReader`, les trois primitives de position de `killsource`
// (`bitAt` / `bits32` / `bitsN`, plus `bitsWide`), `objectives.readBitsBE`,
// `weaponv3.bitReader`, `analysis.scanEvents` et la copie de `bitAt` d `analysis/positions`.
// Une largeur corrigee d un cote et pas de l autre est exactement la divergence silencieuse que
// le chantier ferme. Ils sont ramenes ICI, dans la couche `source` — le seul paquet dont un
// ratchet dedie (`archlint/filmsource_leaf_test.go`) garantit qu il n importe RIEN du depot.
//
// Le lot 2.5.a le deplacera en bloc sous `film/internal/source` par `git mv` pur.
//
// # LA SEMANTIQUE, ET C EST CELLE DU MOTEUR (arbitrage V15 (3))
//
// Lecture MSB-first big-endian : le i-eme bit consomme est le bit (7-(i mod 8)) de l octet
// (i div 8). LES BITS AU-DELA DU TAMPON VALENT ZERO, silencieusement — c est le bourrage de
// queue du moteur, et c est la convention que le lecteur canonique garde. Le lecteur ne porte
// AUCUN drapeau d erreur : un consommateur qui a besoin de se mefier d un flux desynchronise
// (la chaine d evenements de `killsource`) teste [Bits.Remaining] AVANT sa lecture et porte son
// propre drapeau. Le drapeau appartient au marcheur, pas au lecteur.
//
// # LECTURE PAR MOT DE 64 BITS
//
// Le profil CPU de la cuisson d un rejeu (2026-09-02, `tmp/01e1f945.cpu.prof`, 143 s
// d echantillons) placait 58 a 61 % du temps TOTAL dans une seule fonction, qui lisait ses
// 32 bits d identifiant UN BIT A LA FOIS avec un test de borne par bit. [BitsAt] lit par un mot
// de 64 bits + deux decalages. LA SEMANTIQUE HORS TAMPON EST PRESERVEE : le chemin rapide n est
// pris que sur le domaine ou les deux implementations coincident trivialement, tout le reste
// retombe sur la boucle bit a bit d origine, recopiee telle quelle. Les tests differentiels
// (`bits_test.go`) opposent chaque primitive a une copie de reference de l ancienne
// implementation, sur tampons aleatoires a graine fixee, toutes largeurs 0..64, autour de chaque
// frontiere d octet, de mot et de fin de tampon, cas hors tampon compris.

import "encoding/binary"

// Bits : un lecteur de bits sequentiel sur un tampon d octets. C est LE lecteur du depot ; les
// couches du dessus (grammaire, faits) l EMBARQUENT pour y ajouter ce qui leur appartient — un
// profil de largeurs, un observateur, un drapeau de debordement — jamais pour le recopier.
//
// Il ne possede pas son tampon : la tranche est celle du chunk decompresse ou du payload de
// paquet ([Packet.Payload] est une SOUS-TRANCHE du chunk, jamais une copie).
type Bits struct {
	buf []byte
	pos int // position EN BITS du prochain bit a lire
}

// NewBits rend un lecteur positionne sur le premier bit de `buf`.
func NewBits(buf []byte) *Bits { return &Bits{buf: buf} }

// Octets rend le tampon lu, sans copie. Reserve aux consommateurs qui doivent repartir des
// octets eux-memes (une relecture a une autre position, une mesure de longueur) ; lire par
// [Bits.ReadBits] est la voie normale.
func (b *Bits) Octets() []byte { return b.buf }

// NbBits rend la taille du tampon, en bits.
func (b *Bits) NbBits() int { return len(b.buf) * 8 }

// BitPos rend la position, en bits, du prochain bit a lire.
func (b *Bits) BitPos() int { return b.pos }

// SetBitPos deplace la lecture a une position de bit ABSOLUE (resynchronisation d un lecteur
// partage entre plusieurs vues de replication, sans en allouer un second).
func (b *Bits) SetBitPos(p int) { b.pos = p }

// Remaining rend le nombre de bits non lus. C EST LA PORTE DE LA MEFIANCE : un marcheur qui
// refuse de lire au-dela du tampon teste `Remaining() >= n` avant sa lecture (cf. l en-tete).
func (b *Bits) Remaining() int { return len(b.buf)*8 - b.pos }

// Skip avance la lecture de `n` bits sans les decoder. SANS BORNE : la lecture suivante lira des
// zeros de bourrage, comme le moteur.
func (b *Bits) Skip(n int) { b.pos += n }

// ReadBit lit un seul bit, MSB d abord.
func (b *Bits) ReadBit() bool { return b.ReadBits(1) != 0 }

// ReadBits lit `n` bits (0..64) MSB d abord et les rend cales a droite. Les bits au-dela de la
// fin du tampon valent ZERO (bourrage de queue du moteur).
//
// Lecture par mot de 64 bits sur le domaine ou elle coincide avec la boucle d origine :
// position courante non negative et largeur <= 64. Hors de ce domaine — position negative
// (l indexation panique, comme avant) ou largeur > 64 (le resultat ne garde que les 64 DERNIERS
// bits lus, et le curseur avance quand meme de n) — la boucle d origine reste seule maitresse.
func (b *Bits) ReadBits(n uint) uint64 {
	if b.pos >= 0 && n <= 64 {
		r := BitsAt(b.buf, b.pos, n)
		b.pos += int(n)
		return r
	}
	return b.readBitsLoop(n)
}

// readBitsLoop est la lecture bit a bit d origine : elle porte les conventions de bord que le
// chemin par mot ne couvre pas.
func (b *Bits) readBitsLoop(n uint) uint64 {
	var r uint64
	for i := uint(0); i < n; i++ {
		var bit uint64
		if idx := b.pos >> 3; idx < len(b.buf) {
			bit = uint64(b.buf[idx]>>(7-(uint(b.pos)&7))) & 1
		}
		r = r<<1 | bit
		b.pos++
	}
	return r
}

// BitsAt lit `n` bits (0..64) big-endian, MSB d abord, a la position bit `pos`, et les rend
// cales a droite. Les bits au-dela de la fin de `d` valent ZERO — c est le bourrage de queue du
// moteur.
//
// PRECONDITIONS (l appelant les garantit, la fonction ne les verifie pas) : `pos >= 0` et
// `n <= 64`. En dehors, les appelants retombent sur leur boucle d origine, qui seule connait
// leur convention (panique ou zero, troncature au-dela de 64 bits).
//
// Le corps est volontairement COURT : cette fonction doit rester INLINABLE (elle est appelee des
// dizaines de millions de fois par film, une par position de bit candidate, et le seul cout
// d appel se verrait au profil). Tout ce qui ne sert que la queue du tampon vit donc dans
// [bitsAtEdge], hors du budget d inlining.
func BitsAt(d []byte, pos int, n uint) uint64 {
	i := pos >> 3       // premier octet touche
	sh := uint(pos & 7) // bits a jeter en tete de cet octet
	// Cas dominant, et le seul qui doit tenir dans le budget d inlining : le champ tient dans le
	// mot de 64 bits qui commence a l octet i, et ce mot est entierement lisible. `w << sh`
	// amene le bit `pos` en tete du mot.
	if n+sh <= 64 && i+8 <= len(d) && n > 0 {
		return (binary.BigEndian.Uint64(d[i:i+8]) << sh) >> (64 - n)
	}
	return bitsAtEdge(d, i, sh, n)
}

// bitsAtEdge traite les trois cas que [BitsAt] laisse de cote : largeur nulle, champ a cheval
// sur un NEUVIEME octet (`n+sh > 64`, donc n proche de 64 et sh > 0), et queue du tampon (moins
// de huit octets lisibles a partir de l octet i, les manquants valant zero).
func bitsAtEdge(d []byte, i int, sh, n uint) uint64 {
	if n == 0 {
		return 0
	}
	var w uint64
	if i+8 <= len(d) {
		w = binary.BigEndian.Uint64(d[i : i+8])
	} else {
		for k := 0; k < 8 && i+k < len(d); k++ {
			w |= uint64(d[i+k]) << (56 - 8*uint(k))
		}
	}
	if n+sh <= 64 {
		return (w << sh) >> (64 - n)
	}
	extra := n + sh - 64 // 1..7
	var next uint64
	if i+8 < len(d) {
		next = uint64(d[i+8])
	}
	return ((w<<sh)>>sh)<<extra | next>>(8-extra)
}

// BitAt lit UN bit a la position `pos`, MSB d abord. Hors tampon — au-dela de la fin COMME avant
// le debut — il vaut ZERO.
//
// LA TOLERANCE A UNE POSITION NEGATIVE N EST PAS COSMETIQUE : les localisateurs de `killsource`
// testent le bit `s-1` d une position candidate, et la borne basse de leur balayage est une
// PROPRIETE de leur boucle, pas une garantie de cette fonction. [BitsAt], elle, panique sur une
// position negative — c est la difference entre les deux, et elle est voulue.
func BitAt(d []byte, pos int) int {
	if pos < 0 || pos>>3 >= len(d) {
		return 0
	}
	return int(d[pos>>3]>>uint(7-(pos&7))) & 1
}

// BitsTolerants lit `n` bits MSB d abord a la position `pos`, ZERO DES DEUX COTES du tampon —
// avant le premier bit comme apres le dernier.
//
// ELLE EXISTE POUR LES LECTEURS QUI RECULENT : `weaponv3.ResolveXuidToPI` relit les cinq bits
// qui PRECEDENT un motif trouve, et sur un motif au tout debut du chunk elle passe sous zero.
// [BitsAt] panique dans ce cas — c est sa convention, et elle est voulue.
func BitsTolerants(d []byte, pos, n int) uint64 {
	if pos >= 0 && n >= 0 && n <= 64 {
		return BitsAt(d, pos, uint(n))
	}
	var v uint64
	for i := 0; i < n; i++ {
		v = v<<1 | uint64(BitAt(d, pos+i))
	}
	return v
}

// BitsTronques lit AU PLUS `n` bits MSB d abord a la position `pos` et S ARRETE a la fin du
// tampon SANS BOURRER : la valeur rendue ne porte que les bits reellement lus, cales a droite.
//
// ELLE N EST PAS INTERCHANGEABLE AVEC [BitsAt], et c est tout l interet de la nommer : sur les
// derniers bits d un bloc, `BitsAt` decale la valeur lue vers la gauche des bits manquants
// (bourrage a zero du moteur) la ou celle-ci ne la decale pas. Les deux rendent alors des
// valeurs DIFFERENTES. C est la convention du lecteur du PIED DE FILM (`objectives`), et
// la fondre dans celle du moteur changerait des valeurs decodees — ce que D4 interdit.
func BitsTronques(d []byte, pos, n int) uint64 {
	if pos >= 0 && n >= 0 && n <= 64 {
		if reste := len(d)*8 - pos; reste < n {
			n = max(reste, 0)
		}
		return BitsAt(d, pos, uint(n))
	}
	return bitsTronquesBoucle(d, pos, n)
}

// bitsTronquesBoucle est la boucle d origine du lecteur du pied de film : elle seule porte les
// conventions de bord que le chemin par mot ne couvre pas (position negative, largeur > 64).
func bitsTronquesBoucle(d []byte, pos, n int) uint64 {
	var r uint64
	for i := 0; i < n; i++ {
		bi := (pos + i) / 8
		if bi >= len(d) {
			return r
		}
		off := 7 - ((pos + i) % 8)
		r = r<<1 | uint64((d[bi]>>uint(off))&1)
	}
	return r
}
