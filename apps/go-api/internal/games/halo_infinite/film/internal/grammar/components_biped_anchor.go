package grammar

// components_biped_anchor.go — le CORPS tag==3 d'i59 `biped-spartan-ability-non-predicted-
// state` : `FUN_142f25e90`, l'ANCRE DU GRAPPIN.
//
// POURQUOI CE CORPS VAUT UN PORT : la mesure du 2026-08-16 (phase E de
// PLAN_ETAT_ACTIF_EQUIPEMENT) attribue 115 des 117 lectures tag==3 à porteur identifié à
// des vies rang 20 (grappin), par paires à ~0,15 s — c'est l'événement du tir de grappin,
// et son corps porte le POINT D'ACCROCHE (`grappleLines[]` du document).
//
// LA GRAMMAIRE EST CELLE DU LECTEUR DU JEU (relue le 2026-10-08, lot des arrêts de la vue B,
// suite ; `HaloInfinite.exe` HI_1_13_0, lecture seule). Elle remplace la grammaire MESURÉE du
// 2026-08-16, qui ne lisait que deux étiquettes et rendait `ported=false` sur tout le reste :
//
//	FUN_142f21c0c       étiquette = R(3) + 1 (de 1 à 8 ; la branche « étiquette 0 » est morte)
//	FUN_142f26e40       le PRÉFIXE : FUN_1408f0ac4(catégorie 1) -> référence (porte + entier),
//	                    puis FUN_142f04664(c = référence présente) : c nul -> FUN_14076e494 au
//	                    niveau 0x10 (la position de l'ancre) ; sinon R(2) + 3 x R(13) + R(1)[R(16)]
//	FUN_14297ea84       R(6) de drapeaux
//	étiquette 1         FUN_1407f08bc : R(1) [R(8)]
//	étiquette 2         FUN_1408f0ac4(5), FUN_1407f08bc                        (le TIR)
//	étiquette 3         FUN_1408f0ac4(0), FUN_1408f0ac4(5), 3 x FUN_142f26e9c,
//	                    FUN_14076dc04(0x18) = R(24), R(9)                       (l'ACCROCHE)
//	étiquettes 4 et 5   FUN_1408f0ac4(5), FUN_142f26e9c, FUN_14076e494(0x10) (CALL 142f2605d),
//	                    FUN_14076dc04(0x18), R(9)
//	étiquette 6         FUN_1407f08bc ; s'il est absent, FUN_1408f0ac4(5) ; FUN_1408f0ac4(0),
//	                    2 x FUN_142f26e9c, R(1), FUN_14076dc04(0x18)
//	étiquettes 7 et 8   rien : le `switch` du jeu rend la main
//
// L'écrivain (`FUN_142f05660` -> `FUN_142f27930` -> `FUN_142f272ac`) écrit les mêmes champs dans
// le même ordre : l'étiquette moins un, `FUN_142f27da4` (le préfixe), `FUN_1429980a0` (les six
// drapeaux), puis la même branche par étiquette. Aucune étiquette ne fait échouer la lecture.
//
// CE QUE LA GRAMMAIRE MESURÉE LISAIT, EN TERMES DU JEU : ses « drapeaux Zero3 » étaient la porte
// de la référence, la porte de la position et l'index de plage ; sa « position aux largeurs de la
// carte », les trois axes de `FUN_14076e524` à l'index de la plage ; son « Mid7 », les six
// drapeaux et la porte de la première référence de l'étiquette. Sur les corps où ces portes sont
// fermées — les seuls qu'elle lisait — les deux lectures consomment les mêmes bits.
//
// La queue R(3) du composant (FUN_140fc147c, param_4>1) est lue par le DÉSER, pas ici.

// Largeurs du corps tag==3, lues dans `FUN_142f25e90` et ses appelés.
const (
	// anchorInnerBits : `FUN_142f21c0c`, `+0x2c += 3`.
	anchorInnerBits = 3
	// anchorDrapeauxBits : `FUN_14297ea84`, `+0x2c += 6`.
	anchorDrapeauxBits = 6
	// anchorGate8Width : `FUN_1407f08bc` -> `FUN_1407f08f8`, R(8) derrière sa porte.
	anchorGate8Width = 8
	// anchorVecDirBits / anchorVecMagBits : FUN_142f26e9c -> FUN_14076d528, largeurs empilées par
	// l'appelant (0x18 lue en premier, puis 0xc).
	anchorVecDirBits = 24
	anchorVecMagBits = 12
	// anchorPackedBits : FUN_14076dc04(..., 0x18) = R(24) plat. anchorTailBits : le R(9) en ligne
	// qui suit (`+0x2c += 9`).
	anchorPackedBits = 24
	anchorTailBits   = 9
)

// Les catégories que `FUN_142f25e90` pousse dans R8D avant `FUN_1408f0ac4`.
const (
	categorieAncreReference = 1 // FUN_142f26e40 : `MOV R8D, 1`
	categorieAncreCible     = 0 // étiquettes 3 et 6 : `XOR R8D, R8D`
	categorieAncreSource    = 5 // étiquettes 2, 4, 5, 6 : `LEA R8D, [RBX+5]` / `[RBP+6]` (RBP = -1)
)

// anchorInnerLight / anchorInnerHeavy : les valeurs BRUTES des deux corps du grappin, le tir
// (étiquette 2) et l'accroche (étiquette 3). `Inner` porte le brut : étiquette = brut + 1.
const (
	anchorInnerLight = 1
	anchorInnerHeavy = 2
)

// Les étiquettes rangées par `FUN_142f21c0c` (brut + 1), dans l'ordre du `switch` du jeu.
const (
	anchorEtiquettePorte    = 1
	anchorEtiquetteTir      = 2
	anchorEtiquetteAccroche = 3
	anchorEtiquette4        = 4
	anchorEtiquette5        = 5
	anchorEtiquette6        = 6
)

// C'ÉTAIT LA VARIABLE DE PAQUET `abilityAnchorBodyPorted` JUSQU'AU LOT 2.3 : la bascule A/B du
// corps tag==3 d'i59 vit dans [GrammaireBalayage.CorpsAncrageCapacite].

// AbilityAnchorVec est UNE lecture de FUN_142f26e9c : porte à 1 = vecteur constant (0 bit
// de charge) ; porte à 0 = direction cubemap (anchorVecDirBits) + magnitude log/exp
// (anchorVecMagBits), quanta bruts.
type AbilityAnchorVec struct {
	Present bool
	DirQ    uint32
	MagQ    uint32
}

// AbilityNonPredictedState est la lecture complète d'i59 par le déser de production,
// publiée par observateur.AbilityNonPredictedHook (cf. ability_state_hooks.go).
type AbilityNonPredictedState struct {
	// Tag est le tag externe R(2) (FUN_142f2679c). Le corps n'existe que pour Tag==3.
	Tag uint32
	// BodyWalked : le corps a été parcouru (Tag==3 et port actif).
	// BodyOK : le parcours est allé au bout.
	BodyWalked, BodyOK bool
	// Inner est la valeur BRUTE R(3) de tête (étiquette = Inner + 1) : anchorInnerLight (tir) ou
	// anchorInnerHeavy (accroche) pour les deux corps du grappin. -1 quand le corps n'est pas lu.
	Inner int
	// Reference : la porte de la référence du préfixe (`FUN_1408f0ac4`, catégorie 1). Posée, le
	// préfixe ne lit pas de position (branche `c != 0` de `FUN_142f04664`).
	Reference bool
	// PosCarte : la position du préfixe est lue par `FUN_14076e524` à un index de plage, donc aux
	// largeurs d'axe de la carte — la seule forme dont PosQ porte les quanta.
	PosCarte bool
	// PosQ : les trois QUANTA de la position du préfixe quand PosCarte. La déquantification exige
	// les bornes de la carte : pas de bornes, pas de coordonnée monde (règle map_bounds.go).
	PosQ [3]uint32
	// Drapeaux : le R(6) de `FUN_14297ea84`.
	Drapeaux uint32
	// HasR8/R8 : la charge de la porte `FUN_1407f08bc` (étiquettes 1, 2 et 6).
	HasR8 bool
	R8    uint32
	// Vec : les vecteurs de `FUN_142f26e9c`, dans l'ordre du flux (trois à l'étiquette 3).
	Vec [3]AbilityAnchorVec
	// Packed24 : le R(24) de `FUN_14076dc04`. Tail9 : le R(9) final (étiquettes 3 à 5).
	Packed24, Tail9 uint32
}

// consumeAbilityAnchorBody porte `FUN_142f25e90`. Aucune étiquette n échoue : le jeu n en a pas
// d'invalide (au-delà de 6, il ne lit plus rien).
func consumeAbilityAnchorBody(br *Lecteur, st *AbilityNonPredictedState) {
	st.Inner = int(br.ReadBits(anchorInnerBits))          // FUN_142f21c0c
	consumeAbilityAnchorPrefix(br, st)                    // FUN_142f26e40
	st.Drapeaux = uint32(br.ReadBits(anchorDrapeauxBits)) // FUN_14297ea84
	switch st.Inner + 1 {
	case anchorEtiquettePorte:
		consumeAbilityAnchorGate8(br, st)
	case anchorEtiquetteTir:
		consume1408f0ac4(br, categorieAncreSource)
		consumeAbilityAnchorGate8(br, st)
	case anchorEtiquetteAccroche:
		consume1408f0ac4(br, categorieAncreCible)
		consume1408f0ac4(br, categorieAncreSource)
		for i := range st.Vec {
			st.Vec[i] = consumeAbilityAnchorVec(br) // FUN_142f26e9c x3
		}
		st.Packed24 = uint32(br.ReadBits(anchorPackedBits)) // FUN_14076dc04(..., 0x18)
		st.Tail9 = uint32(br.ReadBits(anchorTailBits))
	case anchorEtiquette4, anchorEtiquette5:
		consumeAbilityAnchorDeplacement(br, st)
	case anchorEtiquette6:
		consumeAbilityAnchorEtiquette6(br, st)
	}
}

// consumeAbilityAnchorPrefix porte `FUN_142f26e40` : la référence de catégorie 1, puis
// `FUN_142f04664` ([consume142f04664]) sur la porte de cette référence.
//
// `c` EST LA PORTE : `FUN_142f26e40` teste le handle rangé (`CMP [RBX+4], -1`), que
// `FUN_1408f0ac4` pose à -1 quand sa porte est fermée et à `(queue << 30) | (base + valeur)`
// sinon — une base de catégorie (0x200 à 0x400) plus treize bits au plus, jamais -1.
func consumeAbilityAnchorPrefix(br *Lecteur, st *AbilityNonPredictedState) {
	st.Reference, _ = consume1408f0ac4(br, categorieAncreReference)
	pos, lue := consume142f04664(br, st.Reference)
	if !lue || pos.brute || pos.precHaut || pos.idx < 0 {
		return
	}
	st.PosCarte = true
	for axe := range st.PosQ {
		st.PosQ[axe] = uint32(pos.q[axe]) //nolint:gosec // un axe tient sur 32 bits au plus
	}
}

// consumeAbilityAnchorDeplacement porte les étiquettes 4 et 5 de `FUN_142f25e90` : la référence
// de catégorie 5, un vecteur, la position de `FUN_14076e494` au niveau 0x10 (CALL 142f2605d,
// `LEA R8D, [RBP+0x11]` avec RBP = -1), R(24), R(9).
func consumeAbilityAnchorDeplacement(br *Lecteur, st *AbilityNonPredictedState) {
	consume1408f0ac4(br, categorieAncreSource)
	st.Vec[0] = consumeAbilityAnchorVec(br)
	lireE494(br, niveauPosition)
	st.Packed24 = uint32(br.ReadBits(anchorPackedBits)) // FUN_14076dc04 (`LEA R9D, [RBP+0x19]`)
	st.Tail9 = uint32(br.ReadBits(anchorTailBits))
}

// consumeAbilityAnchorEtiquette6 porte l'étiquette 6 de `FUN_142f25e90` : la porte
// `FUN_1407f08bc` ; fermée (0xffff rangé, `CMP WORD [RSI+0x78], CX`), la référence de catégorie 5 ;
// puis la référence de catégorie 0, deux vecteurs, R(1), R(24) — sans R(9).
func consumeAbilityAnchorEtiquette6(br *Lecteur, st *AbilityNonPredictedState) {
	if !consumeAbilityAnchorGate8(br, st) {
		consume1408f0ac4(br, categorieAncreSource)
	}
	consume1408f0ac4(br, categorieAncreCible)
	st.Vec[0] = consumeAbilityAnchorVec(br)
	st.Vec[1] = consumeAbilityAnchorVec(br)
	br.ReadBit()                                        // FUN_1406cf008 -> +0x4f
	st.Packed24 = uint32(br.ReadBits(anchorPackedBits)) // FUN_14076dc04(..., 0x18)
}

// consumeAbilityAnchorGate8 porte `FUN_1407f08bc` : une porte R(1), puis R(8) (`FUN_1407f08f8`).
// Rend la porte : fermée, le jeu range 0xffff, que l'étiquette 6 teste.
func consumeAbilityAnchorGate8(br *Lecteur, st *AbilityNonPredictedState) bool {
	if !br.ReadBit() {
		return false
	}
	st.HasR8, st.R8 = true, uint32(br.ReadBits(anchorGate8Width))
	return true
}

// consumeAbilityAnchorVec porte FUN_142f26e9c -> FUN_14076d528(mag=0xc, dir=0x18) : même
// porte (charge sur bit==0) et même ordre de flux (direction puis magnitude) que
// consume14076d528 — seules les largeurs diffèrent, et les quanta sont publiés.
func consumeAbilityAnchorVec(br *Lecteur) (v AbilityAnchorVec) {
	if br.ReadBit() { // porte==1 -> vecteur constant, 0 bit de charge
		return v
	}
	v.Present = true
	v.DirQ = uint32(br.ReadBits(anchorVecDirBits))
	v.MagQ = uint32(br.ReadBits(anchorVecMagBits))
	return v
}
