//go:build research

package filmre

// loi_largeurs_axe.go — LA LOI DES LARGEURS D AXE ET DE LA LARGEUR D INDEX DE PLAGE,
// RECOPIEE DU REMPLISSEUR `FUN_140be9a14` (lot 3.4, preparation Ghidra du 2026-09-16).
//
// POURQUOI CE FICHIER EXISTE. Le depot portait deja la loi, en DEUX endroits et sous forme
// abregee : `internal/himap/sbsp.go` (`Bounds.AxisWidths`, la forme fermee au seul niveau 16)
// et l en-tete de `film/profile/i0_layout.go`. Ni l un ni l autre n avait ete confronte au
// remplisseur lui-meme, jamais ouvert jusqu a ce lot (note M3 §3.2, §6 Q7). Ce fichier est la
// transcription COMPLETE du desassemblage — les 32 niveaux, les deux tables, le garde
// d epsilon, le plafond de debordement et la largeur d index — pour que le test de recherche
// qui l accompagne puisse la confronter au catalogue commis et aux deux relevés memoire
// historiques. Il n est PAS de la production : `//go:build research`, personne ne l importe.
//
// PROVENANCE. `HaloInfinite.exe`, base d image `0x140000000`, build `hi_1_13_0`, projet Ghidra
// du 2026-06-04, lecture seule par HTTP `127.0.0.1:8089` le 2026-09-16. Fonctions et adresses :
//
//	FUN_140be9a14                le remplisseur, appele en fin de chargement de carte par
//	                             FUN_140be9890 (140be9b?? : `CALL 0x140be9a14`)
//	FUN_140be9b88                le calcul des trois largeurs d une plage a un niveau
//	FUN_140be9c78                le pas de quantification du niveau
//	FUN_140be9d1c                le test de validite d une AABB (6 float32 strictement croissants)
//	FUN_1406d310c                ceilLog2 (0 pour 0 et pour 1)
//	FUN_14076e524                le LECTEUR, qui choisit sa table selon l index lu au flux
//	DAT_143cd9758 = 0x3c088889   le pas au niveau 16 : 1/120
//	DAT_143cd975c = 0x4a800000   le plafond de comptage : 2^22 = 4194304
//	DAT_143cd837c = 0x38d1b717   l epsilon de pas : 1e-4
//	DAT_143b8c6b8                les bornes PAR DEFAUT du build : +/-20000 sur les trois axes
//	DAT_1445cc9c8                leur copie en .data (24 octets), juste avant la table defaut
//	DAT_1445cc9e0                table DEFAUT : 32 niveaux x 3 u32, pas 0xc
//	DAT_1445ccbe0                table PAR INDEX DE PLAGE : (index*0x20 + niveau)*0xc
//	DAT_14462cbe0                les bornes par index de plage (24 octets, pas 0x18)
//	DAT_1445ccb60                le champ de bits des index de plage valides (1 bit par plage)
//	DAT_144632be0                la largeur d index de plage, cf. [LargeurIndexDePlage]

import "math"

// NiveauPosition est le niveau de precision cable au site d appel du composant de position
// d objet : `MOV R9D,0x10` en `1406d008a` (dans `FUN_1406cfe44`), et de meme en `140f04dd5`,
// `140f04f32`, `140f04f80`, `140f04fe5`, `140f05018`, `140fb8b33`, `140ee7288`, `14226a6b8`.
// C est un IMMEDIAT, jamais un champ de record ni une valeur du registre : les autres niveaux
// releves aux sites d appel du meme lecteur sont `0xc` (`1410f044d`, `14112137e`) et `0xf`
// (`140809775`), chacun pour un AUTRE composant.
const NiveauPosition = 16

// NiveauxDeLaTable est le nombre de niveaux de chaque table (`CMP R14D,0x20` en `140be9ad4`).
const NiveauxDeLaTable = 32

// LargeurAxeMax est le plafond d une largeur d axe (`LEA EBP,[RSI + 0x17]` avec RSI=3, soit
// 0x1a, puis `CMOVG EAX,EBP` en `140be9c34`).
const LargeurAxeMax = 26

// pasNiveau16 est `DAT_143cd9758` = 0x3c088889 = 1/120, le pas de quantification au niveau 16.
const pasNiveau16 = float32(1) / 120

// plafondDeComptage est `DAT_143cd975c` = 0x4a800000 = 2^22 : au-dela, le nombre de casiers
// n est plus calcule, il est fige (`MOV ECX,0x400000` en `140be9c16`).
const plafondDeComptage = float32(1 << 22)

// epsilonDePas est `DAT_143cd837c` = 0x38d1b717 = 1e-4. En dessous de ce pas, les trois
// largeurs valent d emblee 26 (`MOV RAX,0x1a0000001a` en `140be9c62`), sans regarder les
// bornes. A la loi ci-dessous cela vaut pour tout niveau >= 23.
const epsilonDePas = float32(1e-4)

// BornesParDefautDuBuild sont les bornes de `DAT_143b8c6b8`, copiees en `DAT_1445cc9c8` par
// `FUN_140be9a14` puis passees 32 fois a `FUN_140be9b88` pour remplir la table DEFAUT. Ce
// sont des octets de `.rdata` : une donnee du BUILD, pas de la carte.
var BornesParDefautDuBuild = [3][2]float32{{-20000, 20000}, {-20000, 20000}, {-20000, 20000}}

// PasDuNiveau rend `FUN_140be9c78(niveau)` : le pas de quantification du niveau.
//
//	niveau <= 16 : pas = 2^(16-niveau) * (1/120)
//	niveau >  16 : pas = (1/120) / 2^(niveau-16)
func PasDuNiveau(niveau int) float32 {
	if niveau <= NiveauPosition {
		return float32(uint32(1)<<uint(NiveauPosition-niveau)) * pasNiveau16
	}
	return pasNiveau16 / float32(uint32(1)<<uint(niveau-NiveauPosition))
}

// LargeursAxe rend `FUN_140be9b88(niveau, bornes)` : les trois largeurs d axe qu une plage de
// bornes donne a un niveau. `bornes[axe]` est le couple {min, max} de l axe, dans l ordre ou
// `FUN_140be9d1c` les valide (6 float32 : minX, maxX, minY, maxY, minZ, maxZ).
//
// L arithmetique est celle du desassemblage, en simple precision : `SUBSS` pour l etendue,
// `DIVSS` par deux fois le pas, `ceilf`, `CVTTSS2SI`, puis `FUN_1406d310c` et le plafond 26.
func LargeursAxe(bornes [3][2]float32, niveau int) [3]uint {
	pas := PasDuNiveau(niveau)
	if pas < epsilonDePas {
		return [3]uint{LargeurAxeMax, LargeurAxeMax, LargeurAxeMax}
	}
	deuxPas := pas + pas
	limite := deuxPas * plafondDeComptage
	var out [3]uint
	for axe := 0; axe < 3; axe++ {
		etendue := bornes[axe][1] - bornes[axe][0]
		casiers := uint32(plafondDeComptage)
		if etendue < limite {
			casiers = uint32(float32(math.Ceil(float64(etendue / deuxPas))))
		}
		out[axe] = capLargeur(ceilLog2Moteur(casiers))
	}
	return out
}

// LargeurIndexDePlage rend `DAT_144632be0` tel que `FUN_140be9a14` le pose en sortie de
// boucle (`CMP ECX,0x1` en `140be9b16`) : 1 quand la carte ne declare qu UNE plage, sinon
// `ceilLog2(nombre de plages)`. Le nombre retenu est le compte BRUT relu en `140be9afb`
// (`MOV ECX,dword ptr [RSI + 0x7bc]`), pas le nombre de plages dont l AABB est valide — le
// champ de bits des valides (`DAT_1445ccb60`) ne l alimente pas.
func LargeurIndexDePlage(nbPlages int) uint {
	if nbPlages == 1 {
		return 1
	}
	if nbPlages < 0 {
		nbPlages = 0
	}
	return uint(ceilLog2Moteur(uint32(nbPlages)))
}

// ceilLog2Moteur est `FUN_1406d310c` : 0 pour 0 comme pour 1, sinon le nombre de bits
// necessaires pour numeroter `v` valeurs.
func ceilLog2Moteur(v uint32) int {
	if v == 0 {
		return 0
	}
	haut := 31
	for v>>uint(haut) == 0 {
		haut--
	}
	n := haut
	if v&((1<<uint(haut))-1) != 0 {
		n++
	}
	return n
}

func capLargeur(w int) uint {
	if w > LargeurAxeMax {
		return LargeurAxeMax
	}
	if w < 0 {
		return 0
	}
	return uint(w)
}
