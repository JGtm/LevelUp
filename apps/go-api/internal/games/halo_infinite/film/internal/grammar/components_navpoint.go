package grammar

// components_navpoint.go — L'ARCHETYPE `managed-navpoint` (ti=12), DE `i1` AU MINUTEUR MANUEL.
//
// # POURQUOI CE LOT PORTE CET ARCHETYPE (lot 5.1.1, 2026-09-18)
//
// Le compte a rebours que le HUD affiche sous un objectif — « le drapeau revient dans 0:12 » —
// est ecrit par le film en DEUX composants de cet archetype : `i11` la duree INITIALE, `i12` la
// duree COURANTE, `R(17)` chacun, quantifiees au pas de 50 ms, sans indirection ni index a
// resoudre. C'est le chemin le moins cher vers un compte a rebours d'objectif, et le lot 3.7 a
// MESURE que l'autre — le bassin de minuteurs du moteur, designe par `ti=11 i0` — ne porte PAS
// le minuteur du drapeau : `(-1, -1)`, « aucun minuteur », sur 446 records d'objectif sur 446
// (`.ai/V7.5/film_re/NOTE_3_7_REAPPARITION_2026-09-17.md` § 6 bis.3).
//
// LE BLOQUANT ETAIT `i1`, DIX COMPOSANTS AVANT. Une image-cle est un etat COMPLET : son record
// cite tous les composants declares, donc la marche de `ti=12` s'arretait sur `i1` avant
// d'atteindre quoi que ce soit (ratchet 0.A.3 : `i1 managed-navpoint-flags-component` sur les
// sept bobines par build). Ce fichier porte `i1` a `i12` d'un bloc : le bloquant avance a `i13`.
//
// # LA SOURCE DE CHAQUE LARGEUR
//
// `.ai/V7.5/film_re/NOTE_3_6_TI12_GRAMMAIRES_A_2026-09-17.md`, une section par composant (§ 5 a
// § 16), chaque largeur etant un immediat lu au desassemblage du deserialiseur `+0x40` du
// descripteur, recoupe par son serialiseur `+0x28`. AUCUNE largeur n'est ici « parce qu'elle
// marche » : le bloc de filtres de `i2`..`i6` vit dans `components_navpoint_filters.go`, la
// largeur de reference d'entite vient de la table des domaines du depot, et le `param_4` des
// cinq filtres vient du slot `+0x10` de leur descripteur (`component_param4.go`).

// Largeurs de l'archetype, telles que le desassemblage les donne. Une constante par role.
const (
	navpointSubTypeBits      = 32 // i0  FUN_14080dec4
	navpointFlagsBits        = 8  // i1  FUN_14109414c
	navpointDistanceBits     = 16 // i2  FUN_1411b4e6c, quantifie sur [-1, 1000]
	navpointDockingOrderBits = 8  // i7  FUN_142ed5050, lecture en ligne
	navpointGroupNameBits    = 32 // i8  FUN_14080dec4
	navpointTextEntriesBits  = 8  // i9  le compte d'entrees
	navpointTextIDBits       = 32 // i9  `textStringId` par entree
	navpointManualTimerBits  = 17 // i11 / i12 FUN_1406d84b4, 0x11
)

// Bornes de dequantification du minuteur manuel, lues aux adresses que les deux deserialiseurs
// chargent (`DAT_143d13418` et `DAT_143e418a8`, notes § 15 et § 16). Le pas qui en sort vaut
// EXACTEMENT 50 ms — `6553,6002 / 131072 = 0,0500000015` — et c'est la seule chose qui rende
// cette lecture confrontable a une duree affichee.
const (
	navpointManualTimerMin float32 = -0.025    // DAT_143d13418 = 0xbccccccd
	navpointManualTimerMax float32 = 6553.5752 // DAT_143e418a8 = 0x45cccc9a
	// navpointManualTimerDeadZone : `i12` SEUL applique une zone morte apres la
	// dequantification — `|v| <= DAT_143cd837c` rend zero. C'est un traitement de la VALEUR,
	// pas une lecture : la largeur de `i12` vaut celle de `i11`.
	navpointManualTimerDeadZone float32 = 1.0e-4
)

// consumeNavpointFlags (ti=12 i1) — `FUN_141094130` -> `FUN_14109414c` : `R(8)` plat.
// C'ETAIT LE BLOQUANT NOMME DE L'ARCHETYPE sur les sept bobines du ratchet 0.A.3.
func consumeNavpointFlags(br *Lecteur) { br.ReadBits(navpointFlagsBits) }

// consumeNavpointVisibilityDistanceFilters (ti=12 i2) — `FUN_140dbde1c` ->
// `FUN_140dbde44(dest, flux, 0, v = 2 < param_4)` (note § 6).
//
//	bloc de filtres (§ 3)
//	R(16) distance A ; R(16) distance B
//	par filtre present : deux R(16) de plus
//	par filtre present, SEULEMENT si v == 0 : l'octet legacy R(4)
//	K entrees d'ordre de `v ? 3 : 2` bits, K = nombre de filtres presents
func consumeNavpointVisibilityDistanceFilters(br *Lecteur, v bool) bool {
	mask, ok := consumeFilterSet(br, v)
	if !ok {
		return false
	}
	br.ReadBits(navpointDistanceBits)
	br.ReadBits(navpointDistanceBits)
	k := navpointFilterCount(mask)
	for i := 0; i < k; i++ {
		br.ReadBits(navpointDistanceBits)
		br.ReadBits(navpointDistanceBits)
	}
	if !v {
		for i := 0; i < k; i++ {
			br.ReadBits(navpointFilterLegacyByteBits)
		}
	}
	consumeNavpointFilterOrder(br, k, v)
	return true
}

// consumeNavpointBoolFilters (ti=12 i3 et i4) — `FUN_140dbdf80` / `FUN_140dbdfac` ->
// `FUN_140dbdfd8(dest, flux, record, v = 1 < param_4)` (notes § 7 et § 8 : grammaire
// IDENTIQUE, seule la destination change).
//
//	bloc de filtres (§ 3) ; R(1) drapeau ; par filtre present R(1)
//	par filtre present, SEULEMENT si v == 0 : l'octet legacy R(4)
//	K entrees d'ordre de `v ? 3 : 2` bits
func consumeNavpointBoolFilters(br *Lecteur, v bool) bool {
	mask, ok := consumeFilterSet(br, v)
	if !ok {
		return false
	}
	br.ReadBit()
	k := navpointFilterCount(mask)
	for i := 0; i < k; i++ {
		br.ReadBit()
	}
	if !v {
		for i := 0; i < k; i++ {
			br.ReadBits(navpointFilterLegacyByteBits)
		}
	}
	consumeNavpointFilterOrder(br, k, v)
	return true
}

// consumeNavpointFilterOrder lit les `K` entrees d'ordre qui ferment `i2`, `i3` et `i4`. La
// sentinelle `0xff` que le jeu pose ensuite est une ECRITURE MEMOIRE : elle ne consomme rien.
func consumeNavpointFilterOrder(br *Lecteur, k int, v bool) {
	w := navpointFilterOrderBits(v)
	for i := 0; i < k; i++ {
		br.ReadBits(w)
	}
}

// consumeNavpointFilterOnly (ti=12 i5 et i6) — `FUN_140dbe194` / `FUN_140dbdf34` ->
// `FUN_140dbe400(dest, flux, v = 1 < param_4)` : le bloc de filtres SEUL, sans queue (notes
// § 9 et § 10).
func consumeNavpointFilterOnly(br *Lecteur, v bool) bool {
	_, ok := consumeFilterSet(br, v)
	return ok
}

// consumeNavpointDockingOrder (ti=12 i7) — `FUN_142ed5050` : `R(8)` plat, lu en ligne.
func consumeNavpointDockingOrder(br *Lecteur) { br.ReadBits(navpointDockingOrderBits) }

// consumeNavpointDockingGroupName (ti=12 i8) — `FUN_142ed5028` -> `FUN_14080dec4` : `R(32)`,
// un identifiant de chaine.
func consumeNavpointDockingGroupName(br *Lecteur) { br.ReadBits(navpointGroupNameBits) }

// consumeNavpointFormattedText (ti=12 i9) — `FUN_1410e7b90` (note § 13).
//
// `R(8)` de compte, puis par entree un `R(32)` de `textStringId` SUIVI DU MEME SAC TEXTE que
// `ti=11 i2` : presence, identifiant, compte d'arguments, arguments tagges. Le corps n'est donc
// pas recopie — c'est `consumeObjectiveFormattedText` (`components_batch3.go`), dont la note
// etablit la concordance tag par tag avec le `switch` de `FUN_1407f0ebc`.
//
// LA BORNE DU COMPTE N'EST PAS DANS LE LECTEUR : la destination reserve quatre slots
// d'argument par entree et le jeu complete de zero ceux qui manquent, mais `N` est lu tel
// qu'ecrit. Un film sain en porte peu ; aucun plafond n'est ajoute ici, qui ferait diverger de
// la marche du jeu.
func consumeNavpointFormattedText(br *Lecteur) {
	for n := br.ReadBits(navpointTextEntriesBits); n > 0; n-- {
		br.ReadBits(navpointTextIDBits)
		consumeObjectiveFormattedText(br)
	}
}

// consumeNavpointTimers (ti=12 i10) — `FUN_1410d9040` : boucle de `dest + 0x6e8` a
// `dest + 0x6f0` par pas de quatre, donc DEUX `FUN_1410d9088`, `R(7)` rendant `valeur - 1`.
//
// LARGEUR ET SEMANTIQUE IDENTIQUES A `ti=11 i0` : ce sont deux INDEX de minuteur, pas deux
// durees, et zero signifie « aucun minuteur » (`ObjectiveTimerValue`). Le lecteur de l'objectif
// est donc reutilise tel quel (note § 14, note § 17 pt 4) — il publie sur le canal des
// objectifs, ce qui est exact : c'est le meme champ, lu par le meme deserialiseur du moteur.
func consumeNavpointTimers(br *Lecteur) { consumeObjectiveTimers(br) }

// consumeNavpointManualTimerInitial (ti=12 i11) — `FUN_142ed5194` : `R(17)` quantifie, LA DUREE
// INITIALE du minuteur manuel du point de navigation.
func consumeNavpointManualTimerInitial(br *Lecteur) {
	br.obs.publishNavpoint(NavpointManualTimerInitial, br.ReadBits(navpointManualTimerBits))
}

// consumeNavpointManualTimerCurrent (ti=12 i12) — `FUN_142ed512c` : meme `R(17)`, LA DUREE
// COURANTE. La zone morte du jeu porte sur la valeur rendue, pas sur les bits lus : elle est
// appliquee par `NavpointManualTimerValue`, jamais ici.
func consumeNavpointManualTimerCurrent(br *Lecteur) {
	br.obs.publishNavpoint(NavpointManualTimerCurrent, br.ReadBits(navpointManualTimerBits))
}

// NavpointManualTimerValue dequantifie une duree de minuteur manuel (`i11` ou `i12`), EN
// SECONDES. Drapeaux `f6 = 0`, `f7 = 0` : la formule est celle du point milieu de
// `FUN_1406d84b4`, et elle se simplifie en `q * 0,05` sur ces bornes.
//
// LA ZONE MORTE DE `i12` EST APPLIQUEE ICI, pour les deux champs : elle ne change rien a `i11`
// (dont le jeu ne l'applique pas) tant que la valeur depasse 0,1 ms, et elle evite un second
// convertisseur pour deux champs qui lisent le meme enregistrement.
func NavpointManualTimerValue(q uint64) float32 {
	v := DequantEndpoint(q, navpointManualTimerMin, navpointManualTimerMax,
		navpointManualTimerBits, false, false)
	if v <= navpointManualTimerDeadZone && v >= -navpointManualTimerDeadZone {
		return 0
	}
	return v
}
