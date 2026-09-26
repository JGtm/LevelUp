package grammar

// type1_datums.go — LE BLOC DE TYPE 1 : LA TABLE DE DATUMS DU CHUNK, LUE A L OCTET
// (lot 5.21.1, decouverte D1 du lot 5.20).
//
// # CE QUE CE BLOC EST, ET POURQUOI LE DEPOT NE LE LISAIT PAS
//
// Chaque chunk porte, JUSTE AVANT son image-cle, un bloc de type 1 de 343 019 octets —
// deux fois l image-cle, 27 chunks sur 27 sur `bfecd02b`, 5 sur 5 sur `dad793c7`. Le 5.16
// §2.2 (c) l avait ecarte sur `FUN_142989418`, le handler de la POMPE DE LECTURE COURANTE,
// qui lit seize octets et avance un compteur. C est vrai de cette pompe-la ; c est faux du
// chemin de CHARGEMENT D ETAT, ou `FUN_1428e2a9c` le lit AVANT l image-cle :
//
//	FUN_1428e2a9c(session, _, enTete, ctx, tableau) :
//	    FUN_1429883ec(session+0x130, enTete, *(ctx + 0x40))   ; LE BLOC DE TYPE 1
//	    FUN_142988338(session+0x130, hdr2, 0x10, 0)           ; l en-tete de l image-cle
//	    FUN_142988338(session+0x130, session[0x240], hdr2.taille, 0)
//	    FUN_1424c7b4c(lecteur, session[0x240], taille)
//	    FUN_142e2bfd0(lecteur, tableau)                       ; L IMAGE-CLE (lot 5.20)
//
// `*(ctx + 0x40)` est le tableau que `FUN_1408f1618` dimensionne : `monde+0x120` / `+0x128`
// (les bornes de la table de datums, pas 0x18), `monde+0x140` / `+0x148` (un bitmap de pas
// 0x20), et cinq mots de 32 bits a `monde+0x160`. Un bloc que la pompe saute n est pas un
// bloc que le jeu ignore.
//
// # LA GRAMMAIRE, LUE CHEZ L ECRIVAIN (`FUN_1429883ec`), BIT PAR BIT
//
//	FUN_1424c7b4c(lecteur, payload, taille) ; FUN_1406d5cc0(lecteur, 3)  ; 0 bit consomme
//	pour slot de 0 a min(cardinal de la table, DAT_144706100) - 1 :
//	    FUN_14297ea84(lecteur, _, e+0x00)   ; R(6)  -> octet de DRAPEAUX
//	    R(8)                     -> e+0x01  ; la GENERATION
//	    R(32)                    -> e+0x04  ; le COMPTEUR DE GENERATION (pas l archetype)
//	    FUN_140e74e6c(lecteur, _, e+0x08)   ; 33 x R(1), LSB d abord -> MASQUE PAR VUE
//	pour b de 0 a min(cardinal du bitmap, DAT_144706100) - 1 :
//	    0x100 x R(1), LSB d abord dans huit mots de 32 bits  ; 256 bits PAR SLOT
//	5 x R(32)                    -> monde+0x160 .. +0x170
//
// Les deux largeurs qui ne se lisent pas dans `FUN_1429883ec` se lisent dans les deux
// lecteurs qu il appelle : `FUN_14297ea84` prend les SIX bits de tete de sa fenetre
// (`octet de tete >> 2`) et les ecrit en un octet ; `FUN_140e74e6c` boucle `0x21` fois sur
// `FUN_1406cf008` (un bit) et pose `1 << i` dans un mot de 64 bits — 33, comme les 33 vues
// que `FUN_140e74b74` parcourt (`i < 0x21`).
//
// # LA VERIFICATION QUI NE COUTE RIEN : L ARITHMETIQUE REND LA CONSTANTE MESUREE
//
//	par slot        6 + 8 + 32 + 33 = 79 bits, plus 256 bits de bitmap = 335
//	8 191 slots     8 191 x 335                      = 2 743 985 bits
//	queue           5 x 32                           =       160 bits
//	TOTAL                                            = 2 744 145 bits = 343 018,125 octets
//	                                                 -> 343 019 octets, 7 bits de bourrage
//
// 343 019 est EXACTEMENT la taille mesuree du bloc sur les deux films (5.20.3 (d)), et
// 8 191 est `DAT_144706100` (`0x1fff`, D4 du 5.20). La grammaire n est donc pas seulement
// lue : elle est FERMEE par la mesure, sans qu une seule largeur ait ete supposee. Le
// cardinal se DERIVE de la taille (voir [LireBlocDeDatums]) : 2 743 985 = 5 x 67 x 8 191
// n admet aucun autre decoupage en `n x (E + 256) + 160` avec une entree plausible.
//
// # CE QUE PORTE UNE ENTREE — ET CE QU ELLE NE PORTE PAS
//
//	+0x00  DRAPEAUX, 6 bits. `FUN_1408f1618` pose 1 a l allocation, `FUN_1408f12c4` leve 2
//	       a la liberation, et `FUN_1408f1730` ne retient une entite que si
//	       `(f & 1) && (f & 4) && !(f & 2) && !(f & 0x20)` — la seule lecture COMPLETE de ce
//	       champ que le binaire expose. Mesure : trois classes sur les deux films — `0x5`
//	       (vivante), `0x0` avec generation (liberee), `0x0` sans (jamais allouee).
//	+0x01  GENERATION. `FUN_1408f1618` y ecrit `param_2 >> 0x1e` (les deux bits de tete de
//	       l identifiant complet) et `FUN_142f2f73c` la relit pour reconstituer un eid.
//	+0x04  COMPTEUR DE GENERATION, 32 bits — PAS L ARCHETYPE, et c est la correction que ce
//	       lot apporte a D1 (5.20). `FUN_142e2aab4` construit le conteneur AVANT la lecture
//	       et met ce mot a 1 pour chacune de ses entrees ; `FUN_1408f1618` fait de meme a
//	       l allocation. Mesure sur 221 157 entrees de `bfecd02b` : TROIS valeurs, 1, 2 et
//	       3, et elles valent TOUJOURS `+0x01 + 1` — la generation d un slot, dont `+0x01`
//	       ne garde que les deux bits de tete de l eid. Un archetype en prendrait des
//	       dizaines de valeurs sans rapport avec la generation.
//	+0x08  MASQUE PAR VUE, 33 bits. `FUN_1408f1358` teste `(masque >> vue) & 1` pour
//	       decider entre ses deux compteurs — le bit dit ce que la vue `vue` fait de
//	       l entite, pas si elle existe.
//	+0x10  HUIT OCTETS QUE LE BLOC NE PORTE PAS : `FUN_1408f1618` les met a zero,
//	       `FUN_1429883ec` ne les ecrit jamais. Ils ne sont pas dans le film.
//
// # LE BITMAP DE 256 BITS EST LE MASQUE DE PRESENCE DES COMPOSANTS
//
// `FUN_1408f1618` redimensionne `monde+0x140` EN MEME TEMPS que la table de datums et au
// MEME cardinal : le bitmap est indexe par SLOT. Ce que ses 256 bits disent est MESURE, pas
// suppose : sous l archetype que l image-cle donne au slot, chaque bit leve tombe DANS les
// bornes de la liste de composants de cet archetype ([Archetype.Components]) et le nomme —
// `ti=34` leve `tacmap-cameraheading`, `tacmap-fasttravelstate`, `tacmap-waypointstate`,
// `tacmap-queuedreplaymission` ; `ti=2` leve treize composants `game-engine-*` ; les dix
// entites `ti=6` d un meme chunk levent TOUTES `{3, 5, 7, 13, 15, 57}` sur 58 composants.
// ZERO bit hors bornes sur les temoins. C est le meme masque que celui d un record, et il
// est CONSTANT par archetype : 184 masques distincts sur `bfecd02b`, dont 2 ambigus.
//
// HORS LIGNE — jamais depuis un chemin de requete.

import (
	"errors"
	"fmt"
	"math/bits"
)

// PacketTypeDatums est le type du bloc qui precede chaque image-cle : la table de datums du
// chunk, lue par `FUN_1429883ec` sur le chemin de chargement d etat.
const PacketTypeDatums uint16 = 1

// Les largeurs du bloc, toutes lues chez l ecrivain (cf. l en-tete de fichier).
const (
	datumDrapeauxBits = 6     // FUN_14297ea84
	datumGenBits      = 8     // R(8) inline
	datumEtatBits     = 32    // R(32) inline (compteur de generation)
	datumMasqueBits   = 33    // FUN_140e74e6c, 0x21 tours
	datumBitmapBits   = 0x100 // 256 bits par slot
	datumQueueMots    = 5     // monde+0x160 .. +0x170
	datumQueueBits    = datumQueueMots * 32
	datumEntreeBits   = datumDrapeauxBits + datumGenBits + datumEtatBits + datumMasqueBits
	datumPasParSlot   = datumEntreeBits + datumBitmapBits
	// datumCapSlots est `DAT_144706100` en rejeu : `0x1fff`. Il n est pas litteral dans le
	// jeu (D4 du 5.20 : `slotMax + 1`, reecrit quand la table GRANDIT), mais la table est
	// pre-dimensionnee au chargement et il vaut 8 191 sur les deux films temoins.
	datumCapSlots = 0x1fff
	// datumBourrageMax : le bloc se clot sur l octet, donc au plus sept bits de bourrage.
	datumBourrageMax = 7
)

// Les drapeaux de `+0x00`, tels que les ecrivains les posent et les lisent.
const (
	// DatumAlloue est pose par `FUN_1408f1618` quand le slot est alloue.
	DatumAlloue uint8 = 1
	// DatumLibere est leve par `FUN_1408f12c4` quand le slot est rendu.
	DatumLibere uint8 = 2
	// DatumPublie est exige par `FUN_1408f1730` pour qu une entite entre dans sa liste.
	DatumPublie uint8 = 4
	// DatumEcarte est refuse par `FUN_1408f1730`.
	DatumEcarte uint8 = 0x20
)

// ErrBlocDeDatums signale un bloc dont la taille ne se referme pas sur la grammaire lue.
var ErrBlocDeDatums = errors.New("bloc de datums")

// DatumEntry est UNE entree de la table de datums, telle que le bloc de type 1 la porte.
type DatumEntry struct {
	// Drapeaux : les 6 bits de `+0x00` (cf. [DatumAlloue] et ses voisins).
	Drapeaux uint8
	// Gen : la generation de `+0x01`, les deux bits de tete de l identifiant complet.
	Gen uint8
	// Generation : le compteur de 32 bits de `+0x04`. Mesure : il vaut TOUJOURS `Gen + 1`
	// (1 pour une entree jamais allouee, 2 a la premiere generation, 3 a la seconde).
	Generation uint32
	// MasqueVue : les 33 bits de `+0x08`, un bit par vue.
	MasqueVue uint64
	// Composants : le bitmap de 256 bits du slot — la PRESENCE des composants de
	// l archetype de l entite, indexee comme [Archetype.Components].
	Composants [4]uint64
}

// DatumGenerationVierge est la valeur que `FUN_142e2aab4` pre-inscrit dans tout le conteneur
// avant la lecture, et celle que le bloc porte pour un slot jamais alloue.
const DatumGenerationVierge uint32 = 1

// Composant dit si le composant d index `i` de l archetype est present sur cette entite.
func (e DatumEntry) Composant(i int) bool {
	if i < 0 || i >= datumBitmapBits {
		return false
	}
	return e.Composants[i/64]>>uint(i%64)&1 == 1
}

// Vivante applique le predicat de `FUN_1408f1730`, la seule lecture COMPLETE des drapeaux que
// le binaire expose : allouee, publiee, ni liberee ni ecartee.
func (e DatumEntry) Vivante() bool {
	return e.Drapeaux&DatumAlloue != 0 && e.Drapeaux&DatumPublie != 0 &&
		e.Drapeaux&DatumLibere == 0 && e.Drapeaux&DatumEcarte == 0
}

// BlocDeDatums est le contenu d un bloc de type 1, indexe par SLOT.
type BlocDeDatums struct {
	// Entrees[slot] est l entree du slot : la table est dense, un slot vide y figure.
	Entrees []DatumEntry
	// Bitmaps est le nombre d elements de 256 bits consommes, BitmapBitsLeves le nombre de
	// bits a 1 qu ils portent. La matiere n est pas gardee : aucun lecteur ne la demande.
	Bitmaps, BitmapBitsLeves int
	// Queue : les cinq mots de 32 bits de `monde+0x160`.
	Queue [datumQueueMots]uint32
	// BitsLus et Bourrage : ce que la grammaire consomme, et ce qui reste jusqu a l octet.
	BitsLus, Bourrage int
}

// LireBlocDeDatums lit un bloc de type 1 EN ENTIER et rend la table de datums du chunk.
//
// LE CARDINAL N EST PAS SUPPOSE, IL EST DERIVE DE LA TAILLE. Le jeu boucle sur
// `min(cardinal de la table, DAT_144706100)`, deux quantites que le film ne porte pas ; la
// taille du bloc, elle, les determine : `n = (bits - queue) / 335`, a sept bits de bourrage
// pres. Si le compte ainsi derive ne referme pas le bloc a l octet, ce n est pas un bloc de
// datums et la fonction le DIT — elle ne devine pas un cardinal.
func LireBlocDeDatums(pay []byte) (BlocDeDatums, error) {
	total := len(pay) * 8
	n := (total - datumQueueBits + datumBourrageMax) / datumPasParSlot
	if total < datumQueueBits || n <= 0 {
		return BlocDeDatums{}, fmt.Errorf("%w : %d octets ne portent aucune entree",
			ErrBlocDeDatums, len(pay))
	}
	if n > datumCapSlots {
		// D4 du 5.20 : `DAT_144706100` suit la table quand elle GRANDIT. Un film qui
		// depasserait le cap de rejeu changerait aussi la largeur d identifiant du flux de
		// trame — ce lot ne l invente pas, il s arrete.
		return BlocDeDatums{}, fmt.Errorf("%w : %d entrees derivees au-dela du cap %d",
			ErrBlocDeDatums, n, datumCapSlots)
	}
	lus := n*datumPasParSlot + datumQueueBits
	if reste := total - lus; reste < 0 || reste > datumBourrageMax {
		return BlocDeDatums{}, fmt.Errorf("%w : %d bits pour %d entrees, reste %d",
			ErrBlocDeDatums, total, n, reste)
	}
	b := BlocDeDatums{Entrees: make([]DatumEntry, n), BitsLus: lus, Bourrage: total - lus}
	pos := lireEntreesDeDatum(pay, b.Entrees)
	pos = lireBitmapsDeDatum(pay, pos, n, &b)
	for i := range b.Queue {
		b.Queue[i] = uint32(kfReadBits(pay, pos, 32)) //nolint:gosec // R(32)
		pos += 32
	}
	return b, nil
}

// lireEntreesDeDatum consomme la boucle d entrees et rend la position de bit atteinte.
func lireEntreesDeDatum(pay []byte, out []DatumEntry) int {
	pos := 0
	for i := range out {
		e := DatumEntry{
			Drapeaux:   uint8(kfReadBits(pay, pos, datumDrapeauxBits)),                             //nolint:gosec // R(6)
			Gen:        uint8(kfReadBits(pay, pos+datumDrapeauxBits, datumGenBits)),                //nolint:gosec // R(8)
			Generation: uint32(kfReadBits(pay, pos+datumDrapeauxBits+datumGenBits, datumEtatBits)), //nolint:gosec // R(32)
		}
		pos += datumDrapeauxBits + datumGenBits + datumEtatBits
		// LE MASQUE PAR VUE EST ECRIT LSB D ABORD (`1L << i` dans `FUN_140e74e6c`) : c est
		// l inverse de l ordre des trois champs precedents, et le confondre decale les 33
		// vues bout a bout.
		for k := 0; k < datumMasqueBits; k++ {
			e.MasqueVue |= kfBitAt(pay, pos+k) << uint(k)
		}
		pos += datumMasqueBits
		out[i] = e
	}
	return pos
}

// lireBitmapsDeDatum consomme les `n` masques de composants et les pose dans les entrees.
//
// L ORDRE EST CELUI DE L ECRIVAIN, ET IL EST INVERSE DE CELUI DES TROIS PREMIERS CHAMPS : le
// jeu pose le bit de rang `i` du flux dans `*(uint *)(base + (i >> 5) * 4) |= 1 << (i &
// 0x1f)`, c est-a-dire LSB d abord dans chaque mot de 32 bits. Lire 64 bits MSB d abord puis
// INVERSER LES 64 BITS rend exactement cette disposition (le bit de rang `t` du flux passe
// du rang `63 - t` de la lecture MSB au rang `t`), et evite 2,1 millions d appels par chunk
// (`TestBlocDeDatumsAllerRetour` oppose la lecture par mots a l ecriture bit a bit).
//
// `math/bits`, pas `encoding/binary` : le ratchet de `filmdec` refuse le second.
func lireBitmapsDeDatum(pay []byte, pos, n int, b *BlocDeDatums) int {
	for i := 0; i < n; i++ {
		for m := range b.Entrees[i].Composants {
			mot := kfReadBits(pay, pos, 64)
			b.Entrees[i].Composants[m] = bits.Reverse64(mot)
			b.BitmapBitsLeves += bits.OnesCount64(mot)
			pos += 64
		}
	}
	b.Bitmaps = n
	return pos
}
