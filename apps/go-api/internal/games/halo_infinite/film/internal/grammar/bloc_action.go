package grammar

// bloc_action.go — LE BLOC D ACTION `FUN_1406d025c`, LU EN ENTIER (lot M4b de la campagne « retours
// rejeu », 2026-09-24).
//
// # OU IL VIT, ET POURQUOI IL EST UN FICHIER
//
// Le meme deserialiseur sert DEUX lecteurs : le composant `i19 unit-actor-control` de la vue B
// (`consumeUnitActorControl`, `FUN_1408f0778`) et l entree de controle `kind 0` de la vue C
// (`FUN_1406cd860` +0x18, `frame_vue_controle.go`). Il vivait dans `unit_control.go` sous le nom
// `consume1406d025c`, qui CONSOMMAIT ses bits et jetait les valeurs. La vue C en a besoin : ses
// gachettes sont le TIR CONTINU (sonde P1-S3, `.ai/V7.5/retours_rejeu_2026-09-23/
// SONDE_P1S3_tir_continu_vue_controle.md`). Il est donc sorti ici, et il REND ce qu il lit.
//
// # CE QUI CHANGE DANS LA LECTURE, RELU CHEZ L ECRIVAIN (HaloInfinite.exe, lecture seule)
//
// Deux sous-lecteurs etaient faux, et la sonde P1-S3 les a nommes ; ils ont ete relus pour ce lot :
//
//	la QUEUE `FUN_142f26740` -> `FUN_140c9e4d8` -> `FUN_140c9e990` : le genre 1 lit l entier a
//	  largeur variable de `FUN_1406d3140` en CATEGORIE 1 (`MOV R8D,EBP` avec `EBP = 1`, @140c9e9c7,
//	  sonde comprise), le genre 2 en CATEGORIE 2 (`MOV R8D,0x2`, @1423e5a69). Le port lisait la
//	  categorie 0 dans les deux cas : 15 bits au lieu de 12, 16 ou 10.
//	le VECTEUR `FUN_1431a0cbc` : `FUN_142af27f8` = R(2) mode ; 1 -> `FUN_14076dc04` a
//	  `R9D = 0x13` (@1431a0cdf) = R(19) ; 0 -> `FUN_14076e494(br, dst, 0x10, 0, 0, 0)`
//	  (@1431a0d0d) = la position quantifiee de niveau 16, exactement le lecteur que le depot porte
//	  deja sous le nom [consumeSimStateHandleTail] ; 2 et 3 -> une constante, ZERO bit. Le port
//	  lisait `R(1)[R(1)[R(1)]]`.
//
// Le reste etait juste et ne bouge pas : la garde de tete, les six bits de gachettes
// (`FUN_1431ab1ec` : bit `entree` du mot de la main), les quatre bits de barillets
// (`FUN_1431ab1cc` : bit `barillet` de l octet de la main), le bloc de visee (`FUN_1431a0bbc`
// R(1)[R(8)], `FUN_1431a0abc` R(1)[R(10)]), `FUN_1406d0f20` R(3), et les deux `FUN_1406d00ec`
// gardes par les gachettes et barillets de leur main.

// ArmeSentinelle est la valeur de `FUN_1406d00ec` quand son bit de tete vaut 1 : le jeu y pose
// `0xffffffff`, et aucun R(2) ne suit.
const ArmeSentinelle = -1

// ArmeAbsente dit que le champ `FUN_1406d00ec` d une main n est PAS dans le flux : sa garde (une
// gachette ou un barillet de la main) est fermee. Distinct de [ArmeSentinelle], qui est une valeur
// ecrite.
const ArmeAbsente = -2

// BlocDAction est ce que `FUN_1406d025c` lit : les GACHETTES tenues et les BARILLETS en tir de
// chaque main, et l index d arme que la commande en tire.
type BlocDAction struct {
	// Present : la garde de tete. A faux, le bloc est VIDE — aucune gachette, aucun barillet.
	Present bool
	// Gachettes : par MAIN (0, 1), le masque des ENTREES tenues, dans la forme que le jeu ecrit
	// (`FUN_1431ab1ec` : bit `entree` du mot de la main). Trois entrees ; le bit 0 est la gachette
	// principale (`FUN_1406dc5b8` : bit 0 -> drapeau de commande 31, bits 1 | 2 -> 32).
	Gachettes [2]uint8
	// Barillets : par main, le masque des BARILLETS en tir (`FUN_1431ab1cc` : bit `barillet` de
	// l octet de la main). Deux barillets.
	Barillets [2]uint8
	// Arme : par main, `FUN_1406d00ec` (+7 et +8 du bloc) : 0..3 lu, [ArmeSentinelle] ou
	// [ArmeAbsente].
	Arme [2]int
}

// Tire dit si le bloc porte au moins une gachette tenue ou un barillet en tir.
func (b BlocDAction) Tire() bool {
	return b.Gachettes[0]|b.Gachettes[1]|b.Barillets[0]|b.Barillets[1] != 0
}

// Largeurs du bloc, lues chez l ecrivain.
const (
	// entreesDeGachetteParMain : `FUN_1431ab1ec(p, main, {0,1,2}, bit)` — trois entrees.
	entreesDeGachetteParMain = 3
	// barilletsParMain : `FUN_1431ab1cc(p, main, {0,1}, bit)` — deux barillets.
	barilletsParMain = 2
	// largeurVecteurDirection : `FUN_14076dc04` du mode 1 de `FUN_1431a0cbc` (`R9D = 0x13`).
	largeurVecteurDirection = 19
	// largeurIndexArme : le R(2) de `FUN_1406d00ec`.
	largeurIndexArme = 2
	// largeurQueue1406d0f20 : `FUN_1406d0f20` = R(3) (-> +6 du bloc).
	largeurQueue1406d0f20 = 3
)

// Les modes du vecteur `FUN_1431a0cbc` (`FUN_142af27f8` = R(2)).
const (
	modeVecteurQuantifie = 0 // FUN_14076e494(..., 0x10, 0, 0, 0)
	modeVecteurDirection = 1 // FUN_14076dc04, R(19)
)

// lireBlocDAction lit `FUN_1406d025c` et rend ce qu il porte. Memes bits que le jeu, dans son
// ordre ; aucune valeur n est devinee.
func lireBlocDAction(br *Lecteur) BlocDAction {
	b := BlocDAction{Arme: [2]int{ArmeAbsente, ArmeAbsente}}
	if b.Present = br.ReadBit(); !b.Present { // garde (FUN_1406cf008) ; 0 -> bloc vide
		return b
	}
	if br.ReadBit() { // gachettes : main 0 entrees 0..2, puis main 1 entrees 0..2
		for main := 0; main < 2; main++ {
			for entree := 0; entree < entreesDeGachetteParMain; entree++ {
				if br.ReadBit() {
					b.Gachettes[main] |= 1 << entree
				}
			}
		}
	}
	if br.ReadBit() { // barillets : main 0 barillets 0..1, puis main 1
		for main := 0; main < 2; main++ {
			for barillet := 0; barillet < barilletsParMain; barillet++ {
				if br.ReadBit() {
					b.Barillets[main] |= 1 << barillet
				}
			}
		}
	}
	if br.ReadBit() { // bloc de visee -> +0x10 bit 1
		br.ReadBits(2)          // deux R(1) -> +0x10 bits 3 et 2
		consumeOpt1431a0bbc(br) // FUN_1431a0bbc = R(1)[R(8)]
		consumeOpt1431a0abc(br) // FUN_1431a0abc = R(1)[R(10)]
		lireVecteur1431a0cbc(br)
	}
	br.ReadBits(largeurQueue1406d0f20) // FUN_1406d0f20 -> +6
	for main := 0; main < 2; main++ {
		if b.Gachettes[main]|b.Barillets[main] != 0 { // garde du champ de la main (+7, +8)
			b.Arme[main] = lireIndexArme(br)
		}
	}
	consume142f26740(br) // FUN_142f26740 -> FUN_140c9e4d8(bloc+0x28, br, 0)
	return b
}

// lireIndexArme lit `FUN_1406d00ec` : R(1) ; si 0, R(2), sinon la sentinelle.
func lireIndexArme(br *Lecteur) int {
	if br.ReadBit() {
		return ArmeSentinelle
	}
	return int(br.ReadBits(largeurIndexArme)) //nolint:gosec // R(2)
}

// lireVecteur1431a0cbc lit `FUN_1431a0cbc` : R(2) mode ; 1 -> R(19) ; 0 -> la position
// quantifiee de niveau 16 (`FUN_14076e494(..., 0x10, 0, 0, 0)`, le lecteur de
// [consumeSimStateHandleTail]) ; 2 et 3 -> une constante, zero bit.
func lireVecteur1431a0cbc(br *Lecteur) {
	switch br.ReadBits(2) { // FUN_142af27f8
	case modeVecteurDirection:
		br.ReadBits(largeurVecteurDirection)
	case modeVecteurQuantifie:
		consumeSimStateHandleTail(br)
	}
}

// consumeOpt1431a0bbc mirrors FUN_1431a0bbc: R(1) gate; if set R(8).
// CONFIRMED by decompile (refill test "0x40-x < 8", shift 0x38, *(p+0x2c)+=8).
func consumeOpt1431a0bbc(br *Lecteur) { consumeGateR(br, 8) }

// consumeOpt1431a0abc mirrors FUN_1431a0abc: R(1) gate; if set R(10).
// CONFIRMED by decompile (refill test "0x40-x < 10", shift 0x36, *(p+0x2c)+=10).
func consumeOpt1431a0abc(br *Lecteur) { consumeGateR(br, 10) }

// consume142f26740 mirrors the action-block tail FUN_142f26740, a one-line thunk that
// tail-calls FUN_140c9e4d8(struct+0x28, br, /*param_3=*/0):
//
//	gate = FUN_1406cf008: R(1); if bit==0 -> return (no body).
//	if gate:
//	  FUN_140c9e990(sub)            (R(2) genre + entier a largeur variable, cf. consume140c9e990)
//	  R(1) f0  -> [0x18] bit0.
//	  if f0==0:
//	    2x FUN_1406d84b4 dequant (width 4 here; the +0x1c/+0x20 floats)
//	    R(1) f1 -> [0x18] bit1.
//	    if f1==0: return (FUN_140c9e738 NOT called).
//	  FUN_140c9e738(...)            (compressed dir; param_3==0 -> R(1)+[R(15)+R(7)])
func consume142f26740(br *Lecteur) {
	if !br.ReadBit() { // FUN_1406cf008 gate; bit==0 -> no body
		return
	}
	consume140c9e990(br) // FUN_140c9e990 sub-block
	f0 := br.ReadBit()   // [0x18] bit0
	if !f0 {
		br.ReadBits(dequant140c9e4d8Width) // FUN_1406d84b4 -> +0x1c
		br.ReadBits(dequant140c9e4d8Width) // FUN_1406d84b4 -> +0x20
		if !br.ReadBit() {                 // f1 -> [0x18] bit1; if 0 -> return (skip 738)
			return
		}
	}
	consume140c9e738(br, false) // FUN_140c9e738 (param_3==0 -> R(1)+[R(15)+R(7)])
}

// dequant140c9e4d8Width: the FUN_1406d84b4 dequant width inside FUN_140c9e4d8.
// FUN_1406d84b4 takes its width as a stack arg (in_stack_00000028); at this call
// site it is 4 (CONFIRMED by disasm: "MOV EBX,0x4 ; MOV [RSP+0x20],EBX" before both
// CALL 1406d84b4 @140c9e5a4 / @140c9e5bf). Modeled as a named constant.
const dequant140c9e4d8Width = 4

// Les genres de `FUN_140c9e990` (`FUN_1407f0278` = R(2)) et la CATEGORIE de `FUN_1406d3140`
// que chacun pousse dans R8D.
const (
	genreCibleCategorie1 = 1 // MOV R8D,EBP (EBP = 1) @140c9e9c7 : categorie 1, sonde comprise
	genreCibleCategorie2 = 2 // MOV R8D,0x2 @1423e5a69 : categorie 2
)

// consume140c9e990 lit FUN_140c9e990 : R(2) genre ; 1 -> l entier a largeur variable en
// CATEGORIE 1 (sonde comprise), puis R(1)[R(6)] ; 2 -> le meme entier en CATEGORIE 2 ; 0 et 3 ->
// rien. Les deux categories sont relues au site d appel (cf. l en-tete du fichier).
func consume140c9e990(br *Lecteur) {
	switch br.ReadBits(2) { // FUN_1407f0278 = R(2)
	case genreCibleCategorie1:
		readVarWidthInt(br, varWidthProbeCategory) // FUN_1406d3140, categorie 1
		if br.ReadBit() {                          // FUN_1406cf008 gate
			br.ReadBits(6) // R(6)
		}
	case genreCibleCategorie2:
		readVarWidthInt(br, genreCibleCategorie2) // FUN_1406d3140, categorie 2
	}
}

// consume140c9e738 mirrors FUN_140c9e738 -> FUN_14076d528 (compressed direction):
//
//	R(1) gate; if bit==1 -> constant direction (0 further bits).
//	if bit==0: R(widthDir) packed dir + R(widthMag) magnitude.
//	  param_3==1 -> widthDir=0x14(20), widthMag=0xe(14).
//	  else (our case, param_3==0) -> widthDir=0xf(15), widthMag=7.
func consume140c9e738(br *Lecteur, recordStateIsOne bool) {
	if br.ReadBit() { // gate==1 -> constant, no read
		return
	}
	widthDir, widthMag := uint(15), uint(7)
	if recordStateIsOne {
		widthDir, widthMag = 20, 14
	}
	br.ReadBits(widthDir) // FUN_14076dc04/FUN_1406d8288 packed dir
	br.ReadBits(widthMag) // FUN_14076d6dc magnitude
}
