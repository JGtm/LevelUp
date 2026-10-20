package grammar

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
// global de configuration `DAT_145121140` (déjà modélisé par `Lecteur.fullPrecision`,
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
// LE CHEMIN `mode == 1` (FUN_142e29bac) EST PORTE, et il est EMPRUNTE. Il n est atteignable
// que si `param >= 2` ; ce composant a `level = 2` dans le registre du film sur les quatre
// archetypes qui le portent (ti=38/39/40/43), donc le bit de porte C EST lu et le mode 1 est
// choisi quand il est pose. `consumeFwdUpDynPrecConfig` le consomme : R(1) ; si 0 -> R(30) ;
// puis R(30) inconditionnel — 31 ou 61 bits.
//
// LE COMMENTAIRE QUI TENAIT ICI DISAIT LE CONTRAIRE (« paramForComponent rend 1 pour ce
// composant, donc le bit C n est jamais lu ») : il decrivait l etat d avant le 2026-09-03, ou la
// table ne portait pas ce composant. Corrige avec la source de `param_4` au lot 5.1.7.
func consumeObjectForwardAndUpDynPrec(br *Lecteur, param uint32) bool {
	_, ok := decodeObjectForwardAndUpDynPrec(br, param)
	return ok
}

// FwdUpDynPrec porte ce qu'un i2 dynamic-precision livre : la direction packée — le vecteur
// HAUT — et l ANGLE DE ROULIS qui, avec elle, reconstruit l AVANT ([ForwardFromUpRoll]).
// HasDir est faux sur le chemin « keep » (mode 2, deux vec3 bruts, JAMAIS emprunte) et sur les
// quartets du chemin « delta », qui n'écrivent pas de direction absolue dans le flux.
//
// LARGEURS PAR MODE, et c est le mode qui les decide : mode 0 -> direction R(19) et roulis
// R(8) ; mode 1 (chemin « config », DOMINANT sur les builds recents) -> R(30) et R(30). Les
// deux paires passent par la MEME reconstruction.
type FwdUpDynPrec struct {
	HasDir bool
	DirRaw uint32 // direction cubemap du vecteur HAUT, a la largeur du mode
	Mode   uint8  // 0 = incrémental/absolu, 1 = config, 2 = keep (2 vec3 bruts)
	Delta  bool   // bit B : charge utile FUN_14076e744 au lieu de FUN_140c5fa84
	// HasRoll / RollRaw : l angle de roulis ABSOLU, quand le chemin l ecrit. Le chemin
	// « delta » ne l ecrit PAS (sa queue R(1)[+R(4)] est un increment sur l etat precedent),
	// et c est la seule population ou l avant reste hors d atteinte sans suivi d etat.
	HasRoll bool
	RollRaw uint32
	// DirDefault : la porte de direction est posee, donc le moteur prend `(0, 0, 1)` comme
	// HAUT (`DAT_143b8f860` / `0x14472a654`). Le chassis est a plat : l avant vaut alors
	// exactement l angle de roulis.
	DirDefault bool
}

// FwdUpModeConfig est le mode 1, le chemin « config » (`FUN_142e29bac`) : direction sur 30 bits,
// roulis sur 30 bits.
//
// C EST LE SEUL MODE DONT L AVANT RECONSTRUIT EST PROUVE (lot 5.4.2, deux films, oracle du
// deplacement et temoin par permutation) :
//
//	`4f77afc1` mode 1, 35 350 echantillons : mediane 11,0 deg, temoin 88,8
//	`a349fea8` mode 1,    877 echantillons : mediane 23,4 deg (7,7 en AVANCE), temoin 81,5
//
// Le mode 0 est REFUTE sur `a349fea8` (mediane 95,5 contre un temoin a 94,5 : indiscernable) et
// faible sur `4f77afc1` (50,8 sur 538 echantillons). Sa direction lue n y est pas verticale —
// |z| median 0,585 contre 0,979 pour le mode 1 — donc ce n est meme pas le vecteur HAUT qui en
// sort. La publication du cap s appuie sur CE mode et sur lui seul ; ailleurs, la velocite reste
// la source. Cf. D5 (5.4).
const FwdUpModeConfig uint8 = 1

// FwdUpDirBits / FwdUpRollBits rendent les largeurs de la direction et du roulis pour un mode.
// Elles vivent ici, avec la grammaire qui les impose, pour qu aucun lecteur ne les redevine.
func FwdUpDirBits(mode uint8) uint {
	if mode == 1 {
		return 30
	}
	return 19
}

func FwdUpRollBits(mode uint8) uint {
	if mode == 1 {
		return 30
	}
	return 8
}

// Haut rend le vecteur HAUT du composant : la direction lue, ou le defaut `(0, 0, 1)` quand la
// porte est posee. ok est faux quand le chemin n ecrit aucune direction absolue.
func (v FwdUpDynPrec) Haut() ([3]float32, bool) {
	if v.DirDefault {
		return [3]float32{0, 0, 1}, true
	}
	if !v.HasDir {
		return [3]float32{}, false
	}
	return DecodeAimVectorChecked(v.DirRaw, FwdUpDirBits(v.Mode))
}

// Avant rend l AVANT DU CHASSIS — la perpendiculaire que le moteur construit en `base+0x18`.
// ok est faux des qu une des deux moities manque.
func (v FwdUpDynPrec) Avant() ([3]float32, bool) {
	up, ok := v.Haut()
	if !ok || !v.HasRoll {
		return [3]float32{}, false
	}
	return ForwardFromUpRoll(up, RollAngleFromRaw(v.RollRaw, FwdUpRollBits(v.Mode))), true
}

// decodeObjectForwardAndUpDynPrec est le SEUL détenteur de la grammaire d'i2 dyn.-préc. ;
// les deux sauteurs de bits (dispatch et balayage offline) l'appellent. Rend ok=false
// quand le chemin config-dépendant non porté est atteint (cf. la limite ci-dessus).
func decodeObjectForwardAndUpDynPrec(br *Lecteur, param uint32) (FwdUpDynPrec, bool) {
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
		out.DirRaw, out.HasDir, out.RollRaw = decodeFwdUpDynPrecConfig(br)
		out.HasRoll, out.DirDefault = true, !out.HasDir
	case out.Delta:
		dir, has := decodeFwdUpDynPrecDelta(br)
		out.DirRaw, out.HasDir = dir, has
	default:
		out.DirRaw, out.HasDir, out.RollRaw = decodeObjectForwardAndUp(br) // FUN_140c5fa84
		out.HasRoll, out.DirDefault = true, !out.HasDir
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
func decodeFwdUpDynPrecDelta(br *Lecteur) (dir uint32, has bool) {
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
//
// LE « SCALAIRE DEQUANTIFIE » DE QUEUE EST L ANGLE DE ROULIS, et il n est plus jete (lot 5.4) :
// `FUN_1406d84b4(n = 0x1e, min = -pi, max = +pi)` puis `FUN_1406d8678`, exactement la
// reconstruction du chemin de 8 bits a une autre largeur. Quand g == 1, la direction vaut le
// defaut `(0, 0, 1)` (`0x14472a654`) : le chassis est a plat.
func decodeFwdUpDynPrecConfig(br *Lecteur) (dir uint32, hasDir bool, roll uint32) {
	if !br.ReadBit() { // g
		dir, hasDir = uint32(br.ReadBits(30)), true // 0x1e : direction packée du HAUT
	}
	return dir, hasDir, uint32(br.ReadBits(30)) // 0x1e : FUN_1406d84b4, angle de roulis
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
func consumeObjectAngularVelocityDynPrec(br *Lecteur) {
	consumeObjectAngularVelocity(br)
}
