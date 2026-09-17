package profile

// loi_largeurs.go — LA LOI DES LARGEURS D AXE ET DE LA LARGEUR D INDEX DE PLAGE, EN PRODUCTION.
//
// # CE QUE CE FICHIER EST
//
// La transcription de `FUN_140be9b88` / `FUN_140be9c78` — le REMPLISSEUR que le moteur execute
// en fin de chargement de carte (`FUN_140be9a14`) — ouverte dans Ghidra au lot 3.4 preparation
// (`.ai/V7.5/film_re/NOTE_3_4_REMPLISSEUR_LARGEURS_2026-09-16.md`). Les largeurs de
// quantification d une position ne sont NI une constante de code NI une donnee recopiee de la
// carte : elles sont CALCULEES, par une loi fermee, a partir des seules BORNES.
//
//	pas(L)     = 2^(16-L) * C          si L <= 16          C = 1/120 (DAT_143cd9758)
//	pas(L)     = C / 2^(L-16)          si L >  16
//	W[axe]     = min(26, ceilLog2(min(ceil(etendue / (2*pas(L))), 2^22)))
//	W[axe]     = 26 pour les trois axes si pas(L) < 1e-4   (donc des L >= 23)
//	indexW     = 1 si la carte declare UNE plage, sinon ceilLog2(compte BRUT de plages)
//
// # LES DEUX TABLES QUE LE LECTEUR CHOISIT, ET LA SEULE CHOSE QUI LES SEPARE
//
// `FUN_14076e524(dst, lecteur, indexLu, LEVEL)` lit un bit de porte puis, si la porte est
// OUVERTE, un index de plage sur `indexW` bits. L index decide de la table :
//
//	index == -1 (porte posee)  ->  table DEFAUT     : bornes `+/-20000` de `.rdata`
//	                               (DAT_143b8c6b8), donc [LargeursAxeParDefautDuBuild]
//	index >= 0                 ->  table PAR INDEX  : les bornes de la plage `index` de la
//	                               CARTE, donc les largeurs de son entree de catalogue
//
// Le NIVEAU est un IMMEDIAT du site d appel — `MOV R9D,0x10` sur les NEUF sites du composant de
// position (`1406d008a`, `140f04dd5`, `140f04f32`, `140f04f80`, `140f04fe5`, `140f05018`,
// `140fb8b33`, `140ee7288`, `14226a6b8`) : cf. [NiveauPositionDObjet]. Il ne se lit ni dans le
// flux, ni dans le record, ni au registre du build.
//
// # POURQUOI LA LOI VIT ICI, DANS LA COUCHE `profile`
//
// Parce que c est la couche qui DIT COMMENT LIRE (ADR 0034 D-1) et qu elle est une FEUILLE : le
// chemin absolu d i0 a besoin des largeurs de la table DEFAUT a chaque record dont la porte
// d index est posee, et il ne peut les demander a personne d autre sans faire remonter `profile`
// vers une autre couche.
//
// # CE QUE CETTE LOI NE REMPLACE PAS, ET C EST VOULU
//
// Le champ `axisWidths` de `map_quant_bounds.json` reste LA valeur des largeurs par carte. Il
// est produit HORS LIGNE par `cmd/mapquant-build` depuis les `.module` du jeu, sous un gate
// `gamefiles` qui verifie que l outil rend le fichier commis a l octet — c est-a-dire par une
// chaine qui LIT les bornes du jeu, la ou une derivation a l execution ne ferait que recalculer
// en `float32` ce que le producteur a deja calcule. La loi est donc, pour les cartes, un
// CONTROLE : `TestLaLoiRendLesLargeursDuCatalogue` exige qu elle rende exactement les 79 entrees
// commises (l instrument de recherche du lot 3.4 preparation, porte en production).
//
// # PROVENANCE, adresses collees
//
//	FUN_140be9a14   le remplisseur (fin de chargement de carte)      FUN_140be9b88  la loi
//	FUN_140be9c78   le pas du niveau                                 FUN_1406d310c  ceilLog2
//	FUN_14076e524   le lecteur, qui choisit sa table selon l index
//	DAT_143cd9758 = 0x3c088889 = 1/120      DAT_143cd975c = 0x4a800000 = 2^22
//	DAT_143cd837c = 0x38d1b717 = 1e-4       DAT_143b8c6b8 = +/-20000 sur les trois axes
//	plafond de largeur 0x1a = 26            (`LEA EBP,[RSI+0x17]` RSI=3, `CMOVG` en 140be9c34)
//
// `HaloInfinite.exe`, base d image `0x140000000`, build `hi_1_13_0`, projet Ghidra du
// 2026-06-04, lu le 2026-09-16.

import "math"

// NiveauPositionDObjet est le niveau de precision CABLE au site d appel du composant de
// position d objet repliquee (`MOV R9D,0x10` en `1406d008a`, et de meme sur les huit autres
// sites de position). Les autres niveaux releves aux sites d appel du meme lecteur — `0xc`,
// `0xf` — appartiennent a d AUTRES composants.
const NiveauPositionDObjet = 16

// LargeurAxeMax est le plafond moteur d une largeur d axe quantifie (`0x1a`).
const LargeurAxeMax = 26

// pasAuNiveau16 est `DAT_143cd9758` = 1/120 : le pas de quantification au niveau 16.
const pasAuNiveau16 = float32(1) / 120

// plafondDeComptage est `DAT_143cd975c` = 2^22 : le GARDE DE DEBORDEMENT. Au-dela de ce nombre
// de casiers le moteur cesse de compter et fige la valeur (`MOV ECX,0x400000` en `140be9c16`).
// Au niveau 16 il se declenche a une etendue de 69 905,1 unites monde ; la plus grande etendue
// du catalogue est 2 707,4 (`recharge`), donc aucune carte actuelle ne l atteint — un canevas
// Forge plus grand, si.
const plafondDeComptage = float32(1 << 22)

// epsilonDePas est `DAT_143cd837c` = 1e-4 : sous ce pas, les trois largeurs valent d emblee 26
// SANS que les bornes soient regardees (`MOV RAX,0x1a0000001a` en `140be9c62`). A cette loi cela
// vaut pour tout niveau >= 23 — donc jamais pour le composant de position, qui lit au 16.
const epsilonDePas = float32(1e-4)

// borneParDefautDuBuild est la demi-etendue des bornes `DAT_143b8c6b8` de `.rdata`, recopiees en
// `DAT_1445cc9c8` par le remplisseur puis passees 32 fois a la loi pour remplir la table DEFAUT.
// Ce sont des octets du BUILD, pas une donnee de carte.
const borneParDefautDuBuild = float32(20000)

// BornesParDefautDuBuild rend les bornes de la table DEFAUT, axe par axe.
func BornesParDefautDuBuild() [3][2]float32 {
	return [3][2]float32{
		{-borneParDefautDuBuild, borneParDefautDuBuild},
		{-borneParDefautDuBuild, borneParDefautDuBuild},
		{-borneParDefautDuBuild, borneParDefautDuBuild},
	}
}

// PasDuNiveau rend `FUN_140be9c78(niveau)` : le pas de quantification d un niveau.
func PasDuNiveau(niveau int) float32 {
	if niveau <= NiveauPositionDObjet {
		return float32(uint32(1)<<uint(NiveauPositionDObjet-niveau)) * pasAuNiveau16
	}
	return pasAuNiveau16 / float32(uint32(1)<<uint(niveau-NiveauPositionDObjet))
}

// LargeursAxeDuNiveau rend `FUN_140be9b88(niveau, bornes)` : les trois largeurs d axe qu une
// plage de bornes donne a un niveau. `bornes[axe]` est le couple {min, max} de l axe, dans
// l ordre ou `FUN_140be9d1c` les valide (minX maxX minY maxY minZ maxZ).
//
// L arithmetique est celle du desassemblage, en SIMPLE PRECISION : `SUBSS` pour l etendue,
// `DIVSS` par deux fois le pas, `ceilf`, `CVTTSS2SI`, puis `ceilLog2` et le plafond 26.
func LargeursAxeDuNiveau(bornes [3][2]float32, niveau int) [3]uint {
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
		out[axe] = plafonnerLargeur(ceilLog2Moteur(casiers))
	}
	return out
}

// LargeursAxeParDefautDuBuild rend la ligne `niveau` de la TABLE DEFAUT (`DAT_1445cc9e0`) :
// la loi appliquee aux bornes `+/-20000` du build. Au niveau du composant de position elle vaut
// `22/22/22`, et c est la largeur que le chemin absolu d i0 lit quand la porte d index est posee
// (`index == -1`), JAMAIS les largeurs de la carte.
//
// Elle reproduit, aux niveaux 0, 1 et 2, le releve Cheat Engine de `DAT_1445cc9e0` consigne au
// journal le 2026-06-11 (`ce_prec_widths_1445cc9e0.bin`) : `6/6/6`, `7/7/7`, `8/8/8`.
func LargeursAxeParDefautDuBuild(niveau int) [3]uint {
	return LargeursAxeDuNiveau(BornesParDefautDuBuild(), niveau)
}

// LargeurIndexDePlage rend `DAT_144632be0` tel que le remplisseur le pose en sortie de boucle
// (`CMP ECX,0x1` en `140be9b16`) : 1 quand la carte ne declare qu UNE plage, sinon
// `ceilLog2(compte)`. Le compte retenu est le compte BRUT relu en `140be9afb`
// (`MOV ECX,dword ptr [RSI + 0x7bc]`), PAS le nombre de plages dont l AABB est valide — le champ
// de bits des valides (`DAT_1445ccb60`) n alimente pas cette largeur.
func LargeurIndexDePlage(nbPlages int) uint {
	if nbPlages == 1 {
		return 1
	}
	if nbPlages < 0 {
		nbPlages = 0
	}
	return plafonnerLargeur(ceilLog2Moteur(uint32(nbPlages)))
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

// plafonnerLargeur applique le plafond moteur de 26 bits.
func plafonnerLargeur(w int) uint {
	if w > LargeurAxeMax {
		return LargeurAxeMax
	}
	if w < 0 {
		return 0
	}
	return uint(w)
}
