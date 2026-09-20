package grammar

// components_navpoint_filters.go — LE BLOC « JEU DE FILTRES » DU MOTEUR, `FUN_140dbe400`.
//
// # CE QU'IL EST, ET POURQUOI IL A SON FICHIER
//
// Cinq composants de l'archetype `managed-navpoint` (ti=12 `i2`..`i6`) commencent par le MEME
// bloc, et `ti=11 i4 managed-objective-interaction-filter-component` le partage. Le releve
// integral — structure, table des quinze tags, largeur de chaque charge — est
// `.ai/V7.5/film_re/NOTE_3_6_TI12_GRAMMAIRES_A_2026-09-17.md` § 3, lu au desassemblage de
// `FUN_140dbe400` et de ses trois rappels (`FUN_141e98e10` tags 0..5, `FUN_141e98c70` tags
// 6..11, `FUN_141e98f90` tags 12..14).
//
// UN SEUL LECTEUR POUR LES SIX COMPOSANTS (note § 17 pt 2) : la charge d'un filtre ne depend
// que de son tag, jamais du composant qui le porte. Le dupliquer par composant serait la
// troisieme copie d'un meme motif, que la regle 6 de `CLAUDE.md` interdit.
//
// # LE BIT DE VERSION `v`, ET CE QU'IL DECIDE
//
// Le deserialiseur de chaque composant calcule `v` depuis son `param_4` — `2 < param_4` pour
// `i2`, `1 < param_4` pour `i3`..`i6` — et le passe au bloc. `v` ne change QU'UNE largeur ici :
// le drapeau qui suit le masque vaut UN bit quand `v` est pose, TRENTE-DEUX sinon
// (`iVar5 = (-(v != 0) & 0xffffffe1) + 0x20`). La mise en page `v == 0` est celle d'anciennes
// versions du composant, que l'executable courant ne PRODUIT plus (contre-epreuve par le
// serialiseur, note § 2) — elle est portee quand meme : une largeur qu'on lit doit venir du
// lecteur du jeu, pas de ce que le jeu ecrit aujourd'hui.
//
// # LE TAG 15 N'EST PAS LU, IL ARRETE LA MARCHE
//
// Chez le jeu, un tag hors [0, 14] tombe sur `FUN_1411c8f80`, qui NE REVIENT PAS. Un film sain
// n'en porte pas. Ce lecteur rend donc `false` — la traversee s'arrete proprement sur le
// composant (`DesyncAt`) au lieu de deviner une largeur.

// Largeurs du bloc de filtres, telles que le desassemblage les donne. Une constante par role :
// c'est la table de largeurs du bloc, et l'unique endroit ou elle est ecrite.
const (
	// navpointFilterSlots : quatre filtres de 0x40 octets, masque sur quatre bits
	// (`FUN_140dbe598`).
	navpointFilterSlots    = 4
	navpointFilterMaskBits = 4
	// navpointFilterTagBits : le tag de filtre, `R(4)` sur le chemin lent `FUN_1406d6c7c`.
	navpointFilterTagBits = 4
	// navpointFilterFlagBitsRecent / navpointFilterFlagBitsLegacy : le drapeau qui suit le
	// masque, `v ? 1 : 32`.
	navpointFilterFlagBitsRecent = 1
	navpointFilterFlagBitsLegacy = 32
	// navpointFilterOrderBitsRecent / navpointFilterOrderBitsLegacy : une entree d'ordre,
	// `v ? 3 : 2` (`EBP = 2 + (v != 0)`).
	navpointFilterOrderBitsRecent = 3
	navpointFilterOrderBitsLegacy = 2
	// navpointFilterLegacyByteBits : l'octet legacy par filtre present, lu SEULEMENT quand
	// `v == 0` (`TEST R12D,R12D ; JNZ`).
	navpointFilterLegacyByteBits = 4
)

// Largeurs des charges de filtre, par tag (note § 3). Elles ne sont ecrites qu'ici.
const (
	navpointFilterTagMaskBits  = 32 // tag 2 : 32 x R(1), bit i du masque
	navpointFilterTagEnumBits  = 4  // tags 3, 7, 10 : R(4), valeur - 1
	navpointFilterTagRangeBits = 9  // tag 4
	navpointFilterTagListBits  = 3  // tag 5 : R(3) de compte
	navpointFilterTagStrBits   = 13 // tag 5 : charge derriere une porte INVERSEE
	navpointFilterTagRefsBits  = 4  // tag 6 : R(4) de compte
	navpointFilterTagByteBits  = 8  // tag 8
	navpointFilterTagWordBits  = 32 // tags 9, 10, 12, 13, 14
	navpointFilterTagIdxBits   = 5  // tag 11 : charge derriere une porte INVERSEE
)

// Les quinze tags de filtre, nommes. Un litteral de tag dans un `switch` de grammaire est
// exactement le « magic number » que la grille de revue interdit.
const (
	navpointFilterTagVide     = 0  // objet + 0x38 = 0, rien de lu
	navpointFilterTagBool     = 1  // FUN_141e9d6d0
	navpointFilterTagMask     = 2  // FUN_141e9d120
	navpointFilterTagEnumA    = 3  // FUN_1407ef804, valeur - 1
	navpointFilterTagRange    = 4  // FUN_141e9d670
	navpointFilterTagStrList  = 5  // FUN_141e9a9c0
	navpointFilterTagRefList  = 6  // FUN_141e9aa60
	navpointFilterTagEnumB    = 7  // FUN_140968284, valeur - 1
	navpointFilterTagByte     = 8  // FUN_14109414c
	navpointFilterTagWord     = 9  // FUN_141e9d0c0
	navpointFilterTagTriple   = 10 // FUN_141e9d5e0
	navpointFilterTagOptIndex = 11 // FUN_141e9d1a0
	navpointFilterTagRefWord  = 12 // FUN_141e9d440
	navpointFilterTagMarker   = 13 // FUN_141e9cf50 (etiquette « watched-marker »)
	navpointFilterTagRefFlag  = 14 // FUN_141e9d260
)

// navpointFilterFlagBits rend la largeur du drapeau qui suit le masque.
func navpointFilterFlagBits(v bool) uint {
	if v {
		return navpointFilterFlagBitsRecent
	}
	return navpointFilterFlagBitsLegacy
}

// navpointFilterOrderBits rend la largeur d'une entree d'ordre.
func navpointFilterOrderBits(v bool) uint {
	if v {
		return navpointFilterOrderBitsRecent
	}
	return navpointFilterOrderBitsLegacy
}

// navpointFilterCount rend le nombre de bits poses dans un masque de filtres — le `K` des
// boucles d'ordre de `i2` et `i3`/`i4`.
func navpointFilterCount(mask uint64) int {
	n := 0
	for i := 0; i < navpointFilterSlots; i++ {
		if mask>>uint(i)&1 != 0 {
			n++
		}
	}
	return n
}

// consumeFilterSet lit le bloc `FUN_140dbe400(dest, flux, v)` et rend le masque des filtres
// presents. `ok` est faux si un tag hors [0, 14] apparait : chez le jeu il ne revient pas, ici
// la traversee s'arrete proprement.
func consumeFilterSet(br *Lecteur, v bool) (mask uint64, ok bool) {
	mask = br.ReadBits(navpointFilterMaskBits)
	br.ReadBits(navpointFilterFlagBits(v))
	for i := 0; i < navpointFilterSlots; i++ {
		if mask>>uint(i)&1 == 0 {
			continue
		}
		tag := br.ReadBits(navpointFilterTagBits)
		if tag == navpointFilterTagVide {
			continue // objet + 0x38 = 0 : le rappel ne construit rien et ne lit rien
		}
		br.ReadBit() // le R(1) commun, ecrit a `objet + 0x08`, avant l'appel virtuel +0x10
		if !consumeFilterPayload(br, tag) {
			return mask, false
		}
	}
	return mask, true
}

// consumeFilterPayload lit la charge d'un filtre selon son tag (table de la note § 3).
func consumeFilterPayload(br *Lecteur, tag uint64) bool { //nolint:gocyclo // une branche par tag du moteur
	switch tag {
	case navpointFilterTagBool:
		br.ReadBit()
	case navpointFilterTagMask:
		br.ReadBits(navpointFilterTagMaskBits)
	case navpointFilterTagEnumA, navpointFilterTagEnumB:
		br.ReadBits(navpointFilterTagEnumBits)
	case navpointFilterTagRange:
		br.ReadBits(navpointFilterTagRangeBits)
	case navpointFilterTagStrList:
		// `R(3)` de compte, puis par entree la porte INVERSEE de `FUN_142b67f08` : le `R(13)`
		// n'est lu que si le bit vaut ZERO (a un, le champ prend -1).
		for n := br.ReadBits(navpointFilterTagListBits); n > 0; n-- {
			if !br.ReadBit() {
				br.ReadBits(navpointFilterTagStrBits)
			}
		}
	case navpointFilterTagRefList:
		// `R(4)` de compte, puis par entree `FUN_141e9c9c0` : porte DROITE, et la reference
		// d'entite n'est lue que si le bit vaut UN.
		for n := br.ReadBits(navpointFilterTagRefsBits); n > 0; n-- {
			if br.ReadBit() {
				consumeFilterEntityRef(br)
			}
		}
	case navpointFilterTagByte:
		br.ReadBits(navpointFilterTagByteBits)
	case navpointFilterTagWord:
		br.ReadBits(navpointFilterTagWordBits)
	case navpointFilterTagTriple:
		br.ReadBits(navpointFilterTagEnumBits) // FUN_1407ef804
		br.ReadBits(navpointFilterTagEnumBits) // FUN_140968284
		br.ReadBits(navpointFilterTagWordBits)
	case navpointFilterTagOptIndex:
		if !br.ReadBit() { // FUN_1407f2058 : porte INVERSEE
			br.ReadBits(navpointFilterTagIdxBits)
		}
	case navpointFilterTagRefWord, navpointFilterTagMarker, navpointFilterTagRefFlag:
		if br.ReadBit() {
			consumeFilterEntityRef(br)
		}
		br.ReadBits(navpointFilterTagWordBits)
		if tag == navpointFilterTagRefFlag {
			br.ReadBit() // la queue propre au tag 14
		}
	default: // tag 15 : FUN_1411c8f80, qui ne revient pas
		return false
	}
	return true
}

// consumeFilterEntityRef lit une reference d'entite du DOMAINE 0 — `FUN_1406d3140(_, flux, 0,
// out)`, avec `XOR R8D,R8D` sur les quatre sites d'appel (tags 6, 12, 13, 14).
//
// LA LARGEUR DE L'INDEX EST UNE VALEUR DE RUNTIME, pas une constante du format : le jeu la
// calcule par `FUN_1406d310c(cardinal du domaine)`, sur une table peuplee au chargement de
// carte. Le depot la porte DEJA, en un seul endroit — `refDomWidth` (`event_list.go`), la table
// des domaines qui a remplace les trois copies du lot E — et le lecteur de la paire
// index + generation est `readRecordID`, celui des en-tetes de record. Aucune des deux n'est
// recodee ici (note § 17 pt 3).
func consumeFilterEntityRef(br *Lecteur) {
	readRecordID(br, int(refDomWidth(navpointFilterRefDomain)), 0)
}

// navpointFilterRefDomain : le domaine des references de filtre, lu sur les quatre sites
// d'appel.
const navpointFilterRefDomain = 0
