package grammar

// components_biped_posture.go — `ti=35 i55 biped-posture-physics-component`, L UNION D ETAT
// PHYSIQUE DU BIPEDE, PORTEE EN ENTIER (lot 5.7, 2026-09-21).
//
// # CE QUE LE DEPOT LISAIT, ET CE QUE L ECRIVAIN DIT
//
// `consumeBipedPosturePhysics` lisait `R(2)` et s arretait la, sur la foi d un commentaire qui
// disait de `FUN_141fd997c` « resolution d etat, 0 bit lu ». **C ETAIT FAUX, et c est la
// decouverte D1 du lot 5.3** : `FUN_141fd997c` est LE REPARTITEUR D UNE UNION DISCRIMINEE. Il
// POSE UN OCTET DE GENRE (`dst+0x2c` = 1, 2, 3 pour les tags 1, 2, 3 ; le tag 0 prend une
// quatrieme voie qui ne le pose pas) et appelle un lecteur de charge DIFFERENT par tag — de
// quinze a plus de cent bits. Les quatre etaient sautes.
//
// LE COUT DU MANQUE N ETAIT PAS UNE DESYNC MAIS UNE TRONCATURE : `i55` est le 56e des 64
// composants du bipede, donc les bits non lus decalent `i56` a `i63`, le record ne ferme pas au
// bon bit, et la boucle de la vue s arrete au record SUIVANT dont l identifiant ne resout plus
// (c est le mecanisme mesure au 5.3.3-b : « 88,4 % des paquets sains gardent >= 24 bits non
// lus »). Le decodeur ne CORROMPAIT donc rien, il PERDAIT la queue de ces paquets.
//
// # LA CHAINE, RELUE A L OCTET (Ghidra lecture seule, 2026-09-21)
//
//	FUN_142f0293c   thunk : FUN_141015c90(obj+0x12b4)  [0 bit, init] ; param_3 = ctx[0x38] ;
//	                puis FUN_142f1f630(lecteur, ..., {lecteur, etat, param_3})
//	FUN_142f1f630   tag = R(FUN_1406d310c(4)) = R(2) ; FUN_141fd997c(tag, ctx)
//	FUN_141fd997c   tag 0 -> FUN_142f265dc  (aucun octet de genre)
//	                tag 1 -> FUN_142f25a3c  (octet de genre 1)
//	                tag 2 -> FUN_142f263ac  (octet de genre 2)
//	                tag 3 -> FUN_142f264f4  (octet de genre 3)
//
// Feuilles, toutes deja portees ailleurs dans ce paquet SAUF la premiere :
//
//	FUN_14080bd28   R(15)   (un handle court, masque `& 0x7fff`)  — NEUF ici
//	FUN_142af27f8   R(2)    (deja porte : components_object_state.go)
//	FUN_141015740   R(32)   (deja porte : consumeObjectLowFrequency)
//	FUN_14076e494   la queue quantifiee LEVEL=0x10 = [consumeSimStateHandleTail]
//	FUN_14076dc04   R(19)   — la largeur `R9D = 0x13` est LUE AU DESASSEMBLAGE des trois sites
//	                         (142f263e5 `LEA R9D,[RBX+0x13]` avec RBX=0 ; 142f2658b
//	                         `MOV R9D,0x13` ; 1431c357d `MOV R9D,0x13`)
//	FUN_1408f0ac4   porte + entier a largeur variable, CATEGORIE 0
//	FUN_141015cb0   0 bit (memset)
//
// # CE QUE LES QUATRE CHARGES PORTENT, ET CE QUI N EST PAS NOMME
//
// L image ne connait que TROIS classes d etat de bipede — `c_biped_ground_state`,
// `c_biped_airborne_state`, `c_biped_vehicle_state`, les seules chaines `c_biped_*` du binaire —
// et l octet de genre a exactement trois valeurs plus une quatrieme voie. La correspondance
// tag -> classe N EST PAS ETABLIE : aucune chaine n est attachee a l octet de genre, et le
// paragraphe 5.7.2 de la note dit ce que la mesure en fait. **Ce fichier porte des LARGEURS,
// pas des noms.**

// Largeurs de l union d `i55`, dans l ordre du flux.
const (
	// postureTagBits : `FUN_142f1f630` lit `FUN_1406d310c(4)` = `bitLen(4)` = 2.
	postureTagBits = 2
	// postureShortHandleBits : `FUN_14080bd28`, un handle court masque `& 0x7fff`.
	postureShortHandleBits = 15
	// posturePackedDirBits : `FUN_14076dc04(..., R9D = 0x13)`.
	posturePackedDirBits = 19
	// postureWord32Bits : `FUN_141015740`, et le R(32) plat de la branche non referencee du
	// tag 2.
	postureWord32Bits = 32
	// postureSubTagBits : le sous-genre du tag 0 (`FUN_1406d310c(4)` a nouveau) et les deux
	// champs courts du tag 1 (`FUN_1406d310c(3)` = `bitLen(3)` = 2 — la MEME largeur).
	postureSubTagBits = 2
	// postureSmallEnumBits : `FUN_142af27f8`.
	postureSmallEnumBits = 2
)

// consumeBipedPosturePhysics porte `FUN_142f1f630` PUIS le repartiteur `FUN_141fd997c` :
// le tag de 2 bits, et la charge de la branche qu il designe.
//
// LE TAG EST PUBLIE DEPUIS LE LOT 5.3.4, LA CHARGE DEPUIS LE LOT 5.7 : la forme de la tranche
// rendue est documentee sur [EtatPosture]. Le nombre de bits lus, lui, n est PLUS le meme
// qu avant le 2026-09-21 — c est une correction de grammaire, pas une publication.
func consumeBipedPosturePhysics(br *Lecteur) {
	tag := br.ReadBits(postureTagBits)
	var ch postureCharge
	switch tag {
	case 0:
		ch = consumePostureTag0(br) // FUN_142f265dc
	case 1:
		ch = consumePostureTag1(br) // FUN_142f25a3c
	case 2:
		ch = consumePostureTag2(br) // FUN_142f263ac
	default: // 3 — le `else` de FUN_141fd997c, et la seule valeur restante de 2 bits
		ch = consumePostureTag3(br) // FUN_142f264f4
	}
	br.publishEtatMouvement(EtatPosture, tag, ch.porte, ch.sous, ch.handle, ch.dir, ch.mot)
}

// postureCharge rassemble ce que les quatre branches ont de COMMUN a publier. Chaque branche
// remplit ce qu elle lit et laisse le reste a zero ; la forme par tag est documentee sur
// [EtatPosture].
type postureCharge struct {
	porte  uint64 // la porte de tete de la branche (presence de la charge)
	sous   uint64 // le sous-genre (tag 0) ou le premier champ court (tag 1)
	handle uint64 // le handle court R(15)
	dir    uint64 // la direction empaquetee R(19)
	mot    uint64 // le mot de 32 bits
}

// consumePostureTag0 porte FUN_142f265dc :
//
//	FUN_141015cb0                          0 bit (memset)
//	g = R(1)                               FUN_1406cf008 -> dst[0]
//	si g == 0 :
//	    k = R(2)                           FUN_1406d310c(4) -> dst[1]
//	    k == 1       : R(15) + R(1) + R(1) + R(2)
//	    k == 2 ou 3  : R(15) + FUN_1431c3538
//	    k == 0       : rien de plus
//
// La polarite est celle de l ecrivain : la charge est lue quand la porte vaut ZERO.
func consumePostureTag0(br *Lecteur) (ch postureCharge) {
	if br.ReadBit() { // FUN_1406cf008 : porte a 1 -> rien de plus
		ch.porte = 1
		return ch
	}
	ch.sous = br.ReadBits(postureSubTagBits) // FUN_1406d310c(4) = R(2)
	switch ch.sous {
	case 1:
		ch.handle = br.ReadBits(postureShortHandleBits) // FUN_14080bd28 = R(15)
		br.ReadBit()                                    // FUN_1406cf008 -> dst[4]
		br.ReadBit()                                    // FUN_1406cf008 -> dst[5]
		br.ReadBits(postureSmallEnumBits)               // FUN_142af27f8 = R(2)
	case 2, 3:
		ch.handle = br.ReadBits(postureShortHandleBits) // FUN_14080bd28 = R(15)
		ch.mot, ch.dir = consumePostureAnchorTail(br)   // FUN_1431c3538
	}
	return ch
}

// consumePostureAnchorTail porte FUN_1431c3538 (la queue commune des sous-genres 2 et 3 du
// tag 0), relu au desassemblage :
//
//	FUN_142af27f8   = R(2)
//	FUN_1406cf008   = R(1)   -> dst+2
//	FUN_141015740   = R(32)  -> dst+4
//	FUN_1406cf008   = R(1)   -> dst+1 ; si != 0 : FUN_14076dc04(..., R9D = 0x13) = R(19)
func consumePostureAnchorTail(br *Lecteur) (mot, dir uint64) {
	br.ReadBits(postureSmallEnumBits) // FUN_142af27f8 = R(2)
	br.ReadBit()                      // FUN_1406cf008
	mot = br.ReadBits(postureWord32Bits)
	if br.ReadBit() { // FUN_1406cf008 : la direction n est lue que si ce bit est pose
		dir = br.ReadBits(posturePackedDirBits) // FUN_14076dc04(..., 0x13)
	}
	return mot, dir
}

// consumePostureTag1 porte FUN_142f25a3c (octet de genre 1) :
//
//	v0 = R(2)                              FUN_1406d310c(3) -> dst[0]
//	si v0 != 0 :
//	    R(15)                              FUN_14080bd28      -> dst+0xc
//	    R(2)                               FUN_1406d310c(3)   -> dst[1]
//	    R(2)                               FUN_1406d310c(4)   -> dst[2]
//	    FUN_14076e494(..., 0x10)           consumeSimStateHandleTail -> dst+0x10
//	    FUN_14076dc04(..., 0x13) = R(19)   -> dst+0x1c
//
// C est la SEULE branche dont la charge entiere est derriere une valeur non nulle : un tag 1 a
// `v0 == 0` ne coute que deux bits.
func consumePostureTag1(br *Lecteur) (ch postureCharge) {
	ch.sous = br.ReadBits(postureSubTagBits) // FUN_1406d310c(3) = R(2)
	if ch.sous == 0 {
		return ch
	}
	ch.porte = 1
	ch.handle = br.ReadBits(postureShortHandleBits) // FUN_14080bd28 = R(15)
	br.ReadBits(postureSubTagBits)                  // FUN_1406d310c(3) = R(2) -> dst[1]
	br.ReadBits(postureSubTagBits)                  // FUN_1406d310c(4) = R(2) -> dst[2]
	consumeSimStateHandleTail(br)                   // FUN_14076e494(..., 0x10)
	ch.dir = br.ReadBits(posturePackedDirBits)      // FUN_14076dc04(..., 0x13)
	return ch
}

// consumePostureTag2 porte FUN_142f263ac (octet de genre 2) :
//
//	FUN_14076e494(..., 0x10)               consumeSimStateHandleTail -> dst+0
//	FUN_14076dc04(..., 0x13) = R(19)       -> dst+0xc
//	g = R(1)                               FUN_1406cf008 -> dst+0x1c
//	    g == 0 : R(32)                     -> dst+0x18
//	    g != 0 : FUN_1408f0ac4(..., 0)     une REFERENCE D ENTITE, categorie 0
//	R(15)                                  FUN_14080bd28 -> dst+0x28
//
// LES DEUX BRANCHES DE LA PORTE DESIGNENT LA MEME CHOSE, par deux chemins : un identifiant brut
// de 32 bits, ou une reference d entite resolue (`FUN_1408e04c8` la recopie vers `dst+0x18`,
// sans lire un bit). C est la signature d un OBJET PORTEUR.
func consumePostureTag2(br *Lecteur) (ch postureCharge) {
	consumeSimStateHandleTail(br)              // FUN_14076e494(..., 0x10)
	ch.dir = br.ReadBits(posturePackedDirBits) // FUN_14076dc04(..., 0x13)
	if br.ReadBit() {                          // FUN_1406cf008 : porte a 1 -> reference d entite
		ch.porte = 1
		_, ch.mot = consume1408f0ac4(br, 0) // FUN_1408f0ac4(..., categorie 0)
	} else {
		ch.mot = br.ReadBits(postureWord32Bits) // R(32) plat
	}
	ch.handle = br.ReadBits(postureShortHandleBits) // FUN_14080bd28 = R(15)
	return ch
}

// consumePostureTag3 porte FUN_142f264f4 (octet de genre 3) :
//
//	R(15)                                  FUN_14080bd28 -> dst+0x20  (AVANT la porte)
//	g = R(1)                               FUN_1406cf008 -> dst+0x1c
//	si g != 0 :
//	    FUN_14076e494(..., 0x10)           consumeSimStateHandleTail -> dst+0
//	    FUN_14076dc04(..., 0x13) = R(19)   -> dst+0xc
//	R(32)                                  FUN_141015740 -> dst+0x18
//	R(1) + R(1)                            FUN_1406cf008 x2 -> dst+0x1d, dst+0x1e
func consumePostureTag3(br *Lecteur) (ch postureCharge) {
	ch.handle = br.ReadBits(postureShortHandleBits) // FUN_14080bd28 = R(15)
	if br.ReadBit() {                               // FUN_1406cf008
		ch.porte = 1
		consumeSimStateHandleTail(br)              // FUN_14076e494(..., 0x10)
		ch.dir = br.ReadBits(posturePackedDirBits) // FUN_14076dc04(..., 0x13)
	}
	ch.mot = br.ReadBits(postureWord32Bits) // FUN_141015740 = R(32)
	br.ReadBit()                            // FUN_1406cf008 -> dst+0x1d
	br.ReadBit()                            // FUN_1406cf008 -> dst+0x1e
	return ch
}
