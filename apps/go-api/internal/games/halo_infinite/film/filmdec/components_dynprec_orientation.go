package filmdec

// GRAMMAIRE DES COMPOSANTS D'ORIENTATION « DYNAMIC-PRECISION » (i2 / i3 de ti=40).
//
// RÉSOLUTION STATIQUE DU DÉSÉRIALISEUR (2026-09-03, Ghidra HTTP, LECTURE SEULE).
// Jusqu'ici le dépôt tenait le mapping composant -> déser pour une constante de
// RUNTIME, lisible seulement par Cheat Engine sur un jeu lancé (`DAT_144e61d88`,
// cf. RECETTE_DECODAGE_FILM_CHUNKS.md § 5). C'est FAUX pour la partie qui compte :
// le descripteur de chaque TYPE de composant est un objet STATIQUE, et sa vtable
// aussi. La chaîne, purement statique :
//
//	1. la chaîne ASCII du nom du composant (ex. « object-forward-and-up-dynamic-
//	   precision-component » @0x143ca7380) ;
//	2. son UNIQUE xref DATA = un thunk `LEA RAX,[chaîne] ; RET` — la méthode
//	   virtuelle name() du descripteur (ex. FUN_140fc3ec0) ;
//	3. le SLOT qui stocke ce thunk EST la vtable du descripteur, à +0x08
//	   (ex. 0x143e2ba80) ;
//	4. déser = vtable[0x28] = slot+0x20 ; si cette entrée vaut le thunk
//	   `FUN_14076ce9c`, déser = vtable[0x30] = slot+0x28 (la règle exacte de la
//	   recette, transposée en statique).
//
// La chaîne est VALIDÉE 6/6 contre la table extraite en LIVE du bipède
// (RECETTE_DECODAGE_FILM_CHUNKS.md § 6) : object-body-vitality -> FUN_140fb8978,
// object-shield-vitality -> FUN_140d50cbc, object-position-dynamic-precision ->
// FUN_1406cfe44, object-translational-velocity-dynamic-precision -> FUN_14076d45c,
// object-angular-velocity -> FUN_140d70998, object-forward-and-up -> FUN_14076e278.
//
// CE QU'ELLE RÉVÈLE, ET C'ÉTAIT LE BLOCAGE DE ti=40 :
//
//	object-forward-and-up-component                      -> FUN_14076e278  (bipède i2)
//	object-forward-and-up-dynamic-precision-component    -> FUN_140c5f7ec  (ti=40 i2)
//	object-angular-velocity-component                    -> FUN_140d70998  (bipède i3)
//	object-angular-velocity-dynamic-precision-component  -> FUN_140d87740  (ti=40 i3)
//
// Ce sont QUATRE fonctions distinctes. Le dispatch les fusionnait deux à deux
// (une seule branche par famille), donc pour ti=40 il lisait i2 et i3 avec la
// grammaire du BIPÈDE — plus courte — et désynchronisait le curseur AVANT i4
// (object-body-vitality). C'est la cause racine mesurée du lot V2b : 1247/1249
// des records i4 de ti=40 portent i2 et/ou i3 avant i4.
//
// TOUS LES AUTRES composants de ti=40 (i4..i29) résolvent vers EXACTEMENT le même
// déser que le bipède (vérifié un par un le 2026-09-03) : i4 et i5 n'avaient donc
// jamais de problème de grammaire propre, seulement un curseur mal placé.

// fwdUpDynPrecMode2Bits est la largeur du chemin « keep » d'i2 dyn.-préc. :
// FUN_140c5f938 appelle DEUX fois FUN_1406d676c avec R9D = 0x60
// (`@0x140c5f98a` MOV R9D,0x60 ; CALL 0x1406d676c ; `@0x140c5f998` idem), soit deux
// vec3 float32 bruts (avant + haut) = 192 bits.
const fwdUpDynPrecMode2Bits = 2 * rawVec3Bits // 0xc0 = 192

// consumeObjectForwardAndUpDynPrec consomme i2
// `object-forward-and-up-dynamic-precision-component` (ti=38/39/40/43) en portant
// FUN_140c5f7ec, relu au désassemblage `@0x140c5f7ec..0x140c5f8a4` :
//
//	A = R(1)                              CALL 0x1406cf008 @0x140c5f811
//	si A == 0 : B = R(1) ; mode = 0       CALL 0x1406cf008 @0x140c5f81d ; XOR EBX,EBX
//	si A == 1 : B = 0    ; mode = 2       XOR R14B,R14B ; MOV EBX,0x2   @0x140c5f829
//	si param >= 2 : C = R(1) ; si C : mode = 1
//	                                      CMP [RSP+0x70],0x2 ; JC @0x140c5f831
//	                                      CALL 0x1406cf008 @0x140c5f83b ; CMOVNZ EBX,1
//	si B == 0 : FUN_140c5f938(mode)       @0x140c5f877
//	sinon     : FUN_140c5f8a8(mode)       @0x140c5f867
//
// Les deux corps ont la MÊME sélection de charge utile par `mode` (désassemblage
// `@0x140c5f947..0x140c5f9b1` et `@0x140c5f8b7..0x140c5f920`), gouvernée par le
// global de configuration `DAT_145121140` (déjà modélisé par PositionFullPrecision,
// components_movement.go — faux en retail) :
//
//	mode == 2                      : R(96) + R(96)                    (192 bits)
//	mode == 1, ou (DAT==1 et mode<1) : FUN_142e29bac                  (chemin config)
//	mode == 0 et B == 0            : FUN_140c5fa84                    (9 ou 28 bits)
//	mode == 0 et B == 1            : FUN_14076e744                    (2 à 26 bits)
//
// FUN_140c5f9c8, appelé après FUN_140c5fa84 / FUN_14076e744, est de la déquantification
// PURE : aucun bit de flux (aucun `+0x2c +=` dans son désassemblage).
//
// LIMITE ASSUMÉE, écrite avant la mesure : le chemin `mode == 1` (FUN_142e29bac)
// n'est atteignable que si `param >= 2` (ou si la haute précision de process est
// active, ce qu'elle n'est pas en retail). Il n'est PAS porté : paramForComponent
// rend 1 pour ce composant, donc le bit C n'est jamais lu et le mode 1 jamais choisi.
// S'il l'était, la fonction rend `false` (non porté) plutôt que de consommer une
// largeur inventée — FUN_142e29bac vaut R(1) ; si 0 -> R(30) ; puis FUN_1406d84b4
// dont la largeur n'est PAS figée au call-site.
func consumeObjectForwardAndUpDynPrec(br *BitReader, param uint32) bool {
	_, ok := decodeObjectForwardAndUpDynPrec(br, param)
	return ok
}

// FwdUpDynPrec porte ce qu'un i2 dynamic-precision livre : la direction packée sur 19 bits
// quand le chemin qui la porte est emprunté. HasDir est faux sur les chemins « keep »
// (mode 2, deux vec3 bruts) et « delta » (quartets), qui n'écrivent pas de direction
// absolue dans le flux.
type FwdUpDynPrec struct {
	HasDir bool
	DirRaw uint32 // R(19) : même encodage cubemap que l'i2 du bipède
	Mode   uint8  // 0 = incrémental/absolu, 2 = keep (2 vec3 bruts)
	Delta  bool   // bit B : charge utile FUN_14076e744 au lieu de FUN_140c5fa84
}

// decodeObjectForwardAndUpDynPrec est le SEUL détenteur de la grammaire d'i2 dyn.-préc. ;
// les deux sauteurs de bits (dispatch et balayage offline) l'appellent. Rend ok=false
// quand le chemin config-dépendant non porté est atteint (cf. la limite ci-dessus).
func decodeObjectForwardAndUpDynPrec(br *BitReader, param uint32) (FwdUpDynPrec, bool) {
	var out FwdUpDynPrec
	if br.ReadBit() { // A
		out.Mode = 2
	} else {
		out.Delta = br.ReadBit() // B
	}
	if param >= 2 {
		if br.ReadBit() { // C
			out.Mode = 1
		}
	}
	switch {
	case out.Mode == 2:
		br.ReadBits(fwdUpDynPrecMode2Bits)
	case out.Mode == 1:
		consumeFwdUpDynPrecConfig(br)
	case out.Delta:
		dir, has := decodeFwdUpDynPrecDelta(br)
		out.DirRaw, out.HasDir = dir, has
	default:
		out.DirRaw, out.HasDir = decodeObjectForwardAndUp(br) // FUN_140c5fa84, inchangé
	}
	return out, true
}

// consumeFwdUpDynPrecDelta porte FUN_14076e744 (charge utile d'i2 dyn.-préc. quand
// le bit B est posé), relu au désassemblage `@0x14076e744..0x14076e932` :
//
//	g1 = R(1)                                     CALL 0x1406cf008 @0x14076e75f
//	si g1 == 0 :
//	    g2 = R(1)                                 CALL 0x1406cf008 @0x14076e779
//	    si g2 == 0 : R(19)                        (bloc froid 0x14230d170, largeur 0x13)
//	    sinon      : R(4) ; R(4)                  ADD [RBX+0x2c],0x4 @0x14076e79d et @0x14076e7de
//	t = R(1)                                      INC [RBX+0x2c] @0x14076e835
//	si t : R(4)                                   ADD [RBX+0x2c],0x4 @0x14076e865
//
// Les trois chemins convergent sur `JMP 0x14076e828` (g1 == 1 par `@0x14076e8d2`,
// g2 == 0 par LAB_14076e8cd) : la queue R(1)[+R(4)] est INCONDITIONNELLE.
// Coût : 2 (g1=1, t=0) à 26 bits (g1=0, g2=0, t=1).
func decodeFwdUpDynPrecDelta(br *BitReader) (dir uint32, has bool) {
	if !br.ReadBit() { // g1
		if !br.ReadBit() { // g2
			dir = uint32(br.ReadBits(19)) // 0x13 : direction packée absolue
			has = true
		} else {
			br.ReadBits(4) // delta axe 1
			br.ReadBits(4) // delta axe 2
		}
	}
	if br.ReadBit() { // t
		br.ReadBits(4) // delta axe 3
	}
	return dir, has
}

// consumeFwdUpDynPrecConfig porte FUN_142e29bac, la charge utile du mode 1 (bit de porte C
// posé). Relu au désassemblage `@0x142e29bac..0x142e29cf3` :
//
//	g = R(1)                                     CALL 0x1406cf008 @0x142e29bc4
//	si g == 0 : R(30)                            ADD [RBX+0x2c],0x1e @0x142e29bfc / @0x142e29c6c
//	                                             puis FUN_1406d8288(...,0x1e) @0x142e29cac : dépaquetage, 0 bit
//	R(30) INCONDITIONNEL                         MOV [RSP+0x20],0x1e ; CALL 0x1406d84b4 @0x142e29cd6
//	                                             (arg5 = largeur ; même signature que la déquant
//	                                             endpoint d'i5, cf. vitality.go)
//	FUN_1406d8678 : math pure, 0 bit             JMP 0x1406d8678 @0x142e29cf3
//
// Coût : 31 bits (g == 1) ou 61 bits (g == 0).
func consumeFwdUpDynPrecConfig(br *BitReader) {
	if !br.ReadBit() { // g
		br.ReadBits(30) // 0x1e : direction packée
	}
	br.ReadBits(30) // 0x1e : FUN_1406d84b4, scalaire déquantifié
}

// consumeObjectAngularVelocityDynPrec consomme i3
// `object-angular-velocity-dynamic-precision-component` (ti=40 uniquement dans ce
// registre) en portant FUN_140d87740 :
//
//	cVar2 = R(1)                                       CALL 0x1406cf008
//	FUN_14076e1c8(reader, dst, cVar2 ? 2 : 0)
//	    mode 2 : FUN_1406d676c(...,0x60) = R(96)
//	    mode 0 : FUN_14076d528(...,8,0x13) = R(1) présent ; si 0 -> R(19) + R(8)
//
// C'est EXACTEMENT `consumeObjectAngularVelocity` (components_movement.go), le
// porteur déjà écrit de FUN_140d87740 — qui avait été DÉBRANCHÉ en 2026-07 sous un
// drapeau `useLegacyAngularVel` (supprimé le 2026-09-05, lot E : il n'avait plus
// d'installateur) parce qu'il était faux pour le BIPÈDE. Il l'était :
// le bipède porte `object-angular-velocity-component` -> FUN_140d70998, sans le
// gate externe. La correction de 2026-07 était juste pour ti=35 et a cassé ti=40,
// qui est le SEUL archétype du registre à porter la variante dyn.-préc.
func consumeObjectAngularVelocityDynPrec(br *BitReader) {
	consumeObjectAngularVelocity(br)
}
