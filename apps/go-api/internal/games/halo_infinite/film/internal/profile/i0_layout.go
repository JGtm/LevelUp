package profile

// i0_layout.go — LE DECOUPAGE BINAIRE DU COMPOSANT i0 (object-position-dynamic-precision),
// COMME VALEUR.
//
// EXTRAIT DE `grammar/i0_layout.go` AU LOT 2.5.b, ET LA COUPE SUIT LA COUCHE : ce fichier porte
// le TYPE et les trois largeurs d en-tete, c est-a-dire la DONNEE ; la DETECTION
// (`DetectI0LayoutOf`, le profil de bascule, ses seuils et son rapport) reste en `grammar`,
// parce qu elle BALAYE le film — et elle rend desormais un [I0Layout] de ce paquet.
//
// Les largeurs d'axe NE SONT PAS une constante du décodeur : ce sont des CONSTANTES PAR
// CARTE. Le moteur (FUN_140be9a14, appelé en fin de chargement de carte) parcourt le bloc
// structure-BSP du tag scenario et précalcule, pour chaque région de compression r et chaque
// niveau de précision L :
//
//	W[r][L][axe] = min(26, ceilLog2(ceil(extent_axe / (2*q(L)))))   avec q(L) = 2^(16-L)/120
//
// La position d'objet utilise L = 16 câblé au site d'appel (MOV R9D,0x10 en 1406d008a), donc
// 2*q = 1/60 et W = min(26, ceilLog2(ceil(60*extent))). Ni les largeurs ni les bornes ne
// transitent par le bitstream, et le REGISTRE chunk_00 n'y contribue pas : il est
// BIT-À-BIT IDENTIQUE d'un film à l'autre DANS UN BUILD.
//
// C'est pourquoi le découpage est une valeur de PROFIL : il se pose par carte, il ne se lit
// jamais dans le flux. Ce que la grammaire sait en faire — le LIRE sur le profil de bascule
// d'un film quand le catalogue ne le donne pas — est une MESURE, pas une largeur du format.

import "fmt"

// Découpage de l'en-tête d'i0, seule partie NON dérivée du film (source : Ghidra).
//
//	I0SpineBits      : les 3 bits de contrôle lus par FUN_1406cfe44 avant l'aiguillage.
//	I0UseDefaultBits : le bit useDefault de FUN_14076e524 (1 => table par défaut, boîte monde).
//	i0RegionIndexBits: DAT_144632be0 = ceilLog2(nb de BSP valides), 1 quand la carte en a 2.
//
// La valeur 1 de l'index est CONFIRMÉE DANS LE FILM sur les deux cartes de référence : le bit
// 4 d'i0 ne bascule JAMAIS (0 bascule sur 170 518 / 291 288 paires consécutives) mais prend la
// valeur 1 sur une poignée d'enregistrements (3 sur Cliffhanger, 2 sur Catalyst) — signature
// d'un champ d'index d'un bit, et non d'un bit de données.
//
// LES DEUX PREMIERES SONT EXPORTEES DEPUIS LE LOT 2.5.b : six fichiers de `grammar`
// (`i0_layout.go`, `offline_biped.go`, `equipment_recovery.go`, `vehicle_creation.go`,
// `profil_balayage.go`, plus le catalogue de cartes qui vit ici) les lisent pour composer ou
// verifier une longueur d en-tete. Les DUPLIQUER de l autre cote de la frontiere aurait cree
// deux verites pour une largeur du jeu (CLAUDE.md regle 6) ; la troisieme reste privee, son
// seul lecteur etant [DefaultI0GateBits] juste dessous.
const (
	I0SpineBits       = 3
	I0UseDefaultBits  = 1
	i0RegionIndexBits = 1
)

// I0Layout est le découpage binaire du vec3 absolu d'i0 pour UNE carte.
type I0Layout struct {
	// GateBits est le nombre de bits qui précèdent l'axe X (spine + useDefault + index).
	GateBits int
	// AxisW sont les largeurs de quantification de X, Y, Z, en bits.
	AxisW [3]uint
	// Region est la VALEUR d'index de région attendue sur les records décodés (l'index
	// occupe GateBits-4 bits). Zéro partout sauf sur les cartes dont la région jouée n'est
	// pas la première du bloc structure-BSP (Live Fire : région 1 sur 2 bits — lot C
	// catalogues, 2026-08-27). Un record d'une AUTRE région est écarté : ses quanta sont
	// exprimés dans une autre AABB, les déquantifier avec ces bornes produirait une
	// coordonnée fausse silencieuse.
	Region uint32
}

// DefaultI0GateBits est la longueur d'en-tête d'i0 sur le chemin dominant (région explicite).
const DefaultI0GateBits = I0SpineBits + I0UseDefaultBits + i0RegionIndexBits

// TotalBits est la longueur du composant i0 jusqu'à la fin du vec3 (queue exclue).
func (l I0Layout) TotalBits() int {
	return l.GateBits + int(l.AxisW[0]+l.AxisW[1]+l.AxisW[2])
}

// AxisOffset est l'offset bit de l'axe ax depuis le début d'i0.
func (l I0Layout) AxisOffset(ax int) int {
	off := l.GateBits
	for i := 0; i < ax; i++ {
		off += int(l.AxisW[i])
	}
	return off
}

// Valid signale un découpage exploitable (largeurs dans la plage physique du moteur).
func (l I0Layout) Valid() bool {
	if l.GateBits < I0SpineBits+I0UseDefaultBits {
		return false
	}
	for _, w := range l.AxisW {
		if w < 8 || w > 26 { // cap moteur = 26 ; sous 8 bits aucune carte ne descend
			return false
		}
	}
	return true
}

func (l I0Layout) String() string {
	if l.Region != 0 {
		return fmt.Sprintf("gate=%d region=%d %d/%d/%d (i0=%d bits)",
			l.GateBits, l.Region, l.AxisW[0], l.AxisW[1], l.AxisW[2], l.TotalBits())
	}
	return fmt.Sprintf("gate=%d %d/%d/%d (i0=%d bits)", l.GateBits, l.AxisW[0], l.AxisW[1], l.AxisW[2], l.TotalBits())
}
