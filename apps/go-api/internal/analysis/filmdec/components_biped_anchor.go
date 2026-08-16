package filmdec

// components_biped_anchor.go — le CORPS tag==3 d'i59 `biped-spartan-ability-non-predicted-
// state` : FUN_142f25e90, l'ANCRE DU GRAPPIN. Porté le 2026-08-16 (plan
// .ai/V7.5/replay2d/PLAN_GRAPPIN_LIGNE.md, phase 0).
//
// POURQUOI CE CORPS VAUT UN PORT : la mesure du 2026-08-16 (phase E de
// PLAN_ETAT_ACTIF_EQUIPEMENT) attribue 115 des 117 lectures tag==3 à porteur identifié à
// des vies rang 20 (grappin), par paires à ~0,15 s — c'est l'événement du tir de grappin,
// et son corps est le canal candidat du POINT D'ACCROCHE.
//
// LA GRAMMAIRE CI-DESSOUS EST MESURÉE, PAS CONJECTURÉE. Le décompilé consigné
// (.ai/V7.5/killweapon/WALK_PORT_NOTES.md §i59) donnait un switch à tags dont les
// LARGEURS DE FEUILLES sont confirmées ici (vec 0x18/0xc, packed 0x18, refill +9, gate8),
// mais dont l'enrobage ne correspondait pas au flux. La grammaire a été LUE dans les
// films (instruments TestI59AnchorBodyDump / TestI59AnchorTemplate, i59_anchor_test.go :
// champs complets bornés par le record suivant, empilés par classe de longueur, consensus
// par position de bit, paires alignées membre à membre), sur DEUX cartes aux largeurs
// d'axe différentes — c'est ce qui a exposé la POSITION À LARGEURS DE CARTE :
//
//	film 000d5950 (Cliffhanger, axes 13+13+14 = 40) : corps 62 (léger) / 162 (lourd)
//	film 00502e52 (Bazaar,      axes 17+17+16 = 50) : corps 72 (léger) / 172 (lourd)
//	                                                  +10 = +10 d'axes, les deux classes
//
// GRAMMAIRE (après le tag externe R(2)==3 du déser) :
//
//	[R(3) interne]   1 = corps LÉGER (1er membre de la paire : le tir) ;
//	                 2 = corps LOURD (2e membre, ~0,15 s après : l'accroche)
//	[R(3) drapeaux]  000 sur 202 des 210 records du corpus ; les 8 restants (drapeaux
//	                 001/100/110) changent la forme de la suite -> désync propre
//	[R(Wx)][R(Wy)][R(Wz)]  POSITION ABSOLUE quantifiée aux largeurs d'axe de la CARTE
//	                 (WorldObjectPrecision.AxisW — l'entrée MapQuantEntry du match,
//	                 JAMAIS le défaut Cliffhanger ; les bits hauts de Z sont ~constants
//	                 sur un film, la signature d'une coordonnée verticale bornée)
//	[R(7)]           petit champ (observé 0000xy0)
//	[R(1) porte gate8 ; si 1 -> R(8)]        FUN_1407f08bc — ouvert sur les corps légers
//	                 observés, fermé sur les lourds
//	si interne == 2 (corps lourd), et LÀ SEULEMENT :
//	  3 x [R(1) porte ; si 0 -> R(24) dir + R(12) mag]   FUN_142f26e9c (polarité
//	                 FUN_14076d528 : charge sur 0) — observé : deux présents, le 3e absent
//	  [R(24)]        FUN_14076dc04(..., 0x18)
//	  [R(9)]         le « REFILL (+9) » du décompilé
//
// La queue R(3) du composant (FUN_140fc147c, param_4>1) est lue par le DÉSER, pas ici.
// Comptes pleins vérifiés par cohérence AVAL stricte (TestI59AnchorWalkProof) : chaque
// record se termine AU BIT EXACT où commence le record biped suivant (écarts
// min=p10=p50=p90=0), comme le témoin tag!=3.
//
// LES VALEURS INTERNES != 1 et != 2 (4 occurrences résiduelles sur 210 dans le corpus)
// rendent ported=false — désync propre plutôt qu'une largeur devinée (règle du plan).
//
// La MAGNITUDE (R(12)) des vecteurs du corps lourd dépend d'une plage (min, max) que ni
// le décompilé ni le flux seul ne donnent : les QUANTA BRUTS sont publiés ; la direction
// se déquantifie sans plage (DecodeAimVectorChecked) ; la position, elle, se déquantifie
// avec les BORNES de la carte (MapQuantEntry.Range) — côté lecteur, jamais ici.

// Largeurs fixes du corps tag==3 (mesurées ; feuilles confirmées contre le décompilé).
const (
	anchorInnerBits  = 3
	anchorZeroBits   = 3
	anchorMidBits    = 7
	anchorGate8Width = 8
	// anchorVecDirBits / anchorVecMagBits : FUN_142f26e9c -> FUN_14076d528(mag=0xc, dir=0x18).
	anchorVecDirBits = 24
	anchorVecMagBits = 12
	// anchorPackedBits : FUN_14076dc04(..., 0x18) = R(24) plat. anchorTailBits : R(9) final.
	anchorPackedBits = 24
	anchorTailBits   = 9
)

// anchorInnerLight / anchorInnerHeavy : les deux seules valeurs internes observées en
// nombre (fire / accroche).
const (
	anchorInnerLight = 1
	anchorInnerHeavy = 2
)

// abilityAnchorBodyPorted : le corps tag==3 est-il décodé ? Bascule A/B (patron
// MobilityActionBodyPorted) pour rejouer la ligne de base d'avant le port (tag lu, corps
// sauté, marche continuée sans lui).
var abilityAnchorBodyPorted = true

// SetAbilityAnchorBodyPorted bascule le portage du corps d'i59 (instruments uniquement).
func SetAbilityAnchorBodyPorted(b bool) { abilityAnchorBodyPorted = b }

// AbilityAnchorVec est UNE lecture de FUN_142f26e9c : porte à 1 = vecteur constant (0 bit
// de charge) ; porte à 0 = direction cubemap (anchorVecDirBits) + magnitude log/exp
// (anchorVecMagBits), quanta bruts.
type AbilityAnchorVec struct {
	Present bool
	DirQ    uint32
	MagQ    uint32
}

// AbilityNonPredictedState est la lecture complète d'i59 par le déser de production,
// publiée par abilityNonPredictedHook (cf. ability_state_hooks.go).
type AbilityNonPredictedState struct {
	// Tag est le tag externe R(2) (FUN_142f2679c). Le corps n'existe que pour Tag==3.
	Tag uint32
	// BodyWalked : le corps a été parcouru (Tag==3 et port actif).
	// BodyOK : le parcours est allé au bout (valeur interne connue).
	BodyWalked, BodyOK bool
	// Inner est la valeur R(3) de tête : anchorInnerLight (tir) ou anchorInnerHeavy
	// (accroche). -1 quand le corps n'est pas lu.
	Inner int
	// Zero3 : les trois drapeaux après la valeur interne (000 sur la quasi-totalité des
	// records ; autre valeur = forme non établie, BodyOK passe à false).
	Zero3 uint32
	// PosQ : les trois QUANTA de la position absolue, aux largeurs d'axe de la carte
	// (WorldObjectPrecision.AxisW au moment de la lecture). La déquantification exige les
	// bornes de la carte : pas de bornes, pas de coordonnée monde (règle map_bounds.go).
	PosQ [3]uint32
	// Mid7 : le champ R(7) entre la position et la porte gate8.
	Mid7 uint32
	// HasR8/R8 : la charge de la porte gate8 (FUN_1407f08bc).
	HasR8 bool
	R8    uint32
	// Vec : les trois vecteurs quantifiés du corps lourd, dans l'ordre du flux.
	// Observé : les deux premiers présents, le troisième absent.
	Vec [3]AbilityAnchorVec
	// Packed24 : le R(24) plat de queue du corps lourd. Tail9 : le R(9) final.
	Packed24, Tail9 uint32
}

// consumeAbilityAnchorBody porte le corps tag==3 selon la grammaire mesurée. Rend false
// sur les valeurs internes jamais observées : au-delà, le curseur ne serait plus digne de
// confiance et la marche doit s'arrêter proprement.
func consumeAbilityAnchorBody(br *BitReader, st *AbilityNonPredictedState) bool {
	st.Inner = int(br.ReadBits(anchorInnerBits))
	st.Zero3 = uint32(br.ReadBits(anchorZeroBits))
	if st.Zero3 != 0 {
		// Les trois bits sont des DRAPEAUX : quand ils ne valent pas 000, la suite change
		// de forme (8 records sur 210 dans le corpus famille B, tailles singleton 37, 92,
		// 114, 142, 189-247 bits, dont deux indépendantes de la carte). Leur grammaire
		// n'est pas établie : désync propre plutôt qu'une largeur devinée.
		return false
	}
	for ax := 0; ax < 3; ax++ {
		st.PosQ[ax] = uint32(br.ReadBits(WorldObjectPrecision.AxisW[ax])) // largeurs de CARTE
	}
	st.Mid7 = uint32(br.ReadBits(anchorMidBits))
	if br.ReadBit() { // FUN_1407f08bc : porte à 1 -> R(8)
		st.HasR8, st.R8 = true, uint32(br.ReadBits(anchorGate8Width))
	}
	switch st.Inner {
	case anchorInnerLight:
		return true
	case anchorInnerHeavy:
		for i := range st.Vec {
			st.Vec[i] = consumeAbilityAnchorVec(br) // FUN_142f26e9c x3
		}
		st.Packed24 = uint32(br.ReadBits(anchorPackedBits)) // FUN_14076dc04(..., 0x18)
		st.Tail9 = uint32(br.ReadBits(anchorTailBits))      // refill R(9)
		return true
	default: // jamais observé en nombre : désync propre plutôt qu'une largeur devinée
		return false
	}
}

// consumeAbilityAnchorVec porte FUN_142f26e9c -> FUN_14076d528(mag=0xc, dir=0x18) : même
// porte (charge sur bit==0) et même ordre de flux (direction puis magnitude) que
// consume14076d528 — seules les largeurs diffèrent, et les quanta sont publiés.
func consumeAbilityAnchorVec(br *BitReader) (v AbilityAnchorVec) {
	if br.ReadBit() { // porte==1 -> vecteur constant, 0 bit de charge
		return v
	}
	v.Present = true
	v.DirQ = uint32(br.ReadBits(anchorVecDirBits))
	v.MagQ = uint32(br.ReadBits(anchorVecMagBits))
	return v
}
