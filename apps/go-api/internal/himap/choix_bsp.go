// Package himap — choix_bsp.go : QUEL tag sbsp d'un module porte l'arene qu'on doit dessiner.
//
// Ce fichier ne contient qu'une decision, mais c'est celle qui decide de TOUT le fond : un
// module declare plusieurs bsp, l'un est l'aire de jeu, les autres sont des decors lointains,
// et retenir le mauvais rend une image ou l'arene n'apparait pas.
//
// A NE PAS CONFONDRE avec `sbsp_region.go` : celui-la designe le bsp contre lequel le MOTEUR
// quantifie les positions repliquees, en lisant l'ordre des regions dans le tag de scenario.
// Ici on cherche le bsp a DESSINER, pour une carte dont on connait les ancres d'objectif. Les
// deux questions ont la meme allure et des reponses independantes.
package himap

import "math"

// ChoisitBSP retient le bsp qui contient le PLUS D'ANCRES ; a nombre d'ancres EGAL, celui dont
// l'emprise au sol est la PLUS PETITE ; et a defaut d'ancre, celui qui porte le plus
// d'instances.
//
// POURQUOI UN DEPARTAGE, ET POURQUOI CELUI-LA (mesure du 2026-08-27,
// `TestSondeBSPCartesNatives` et `TestSondeBSPCommunLiveFire`).
//
// L'egalite n'est pas un cas de bord : elle est la NORME. Sur toutes les cartes temoins qui
// declarent plusieurs bsp, TOUS leurs bsp contiennent la totalite des ancres — un decor
// lointain est une boite qui englobe l'arene, il la contient donc par construction :
//
//	carte                  arene                        horizon                      rapport
//	ctf_forbidden          82,5 x 62,8 m,  13 830 inst  2 468 x 2 260 m,  1 588 inst    x 1 077
//	catalyst_map          297,5 x 408,4 m, 11 468 inst  6 466 x 7 169 m,  1 135 inst    x   381
//	cliffhanger_ridgeline 113,2 x 113,8 m, 10 357 inst  6 619 x 10 472 m, 3 971 inst    x 5 378
//	streets_sgh_streets    51,7 x  52,9 m, 10 910 inst    501 x    501 m,     91 inst    x    92
//	common-rtx-new (bsp0)  63,2 x  63,8 m, 12 556 inst  7 890 x  7 776 m,  4 863 inst    x15 211
//
// Jusqu'au 2026-08-27 cette egalite se tranchait par l'ORDRE DE LECTURE :
// `ReadModuleInstances` trie les tags sbsp par taille decompressee decroissante et la
// comparaison etait STRICTE, donc le plus GROS TAG gagnait. Il se trouve que l'arene pese plus
// d'octets que l'horizon sur les six cas ci-dessus : la chaine tombait juste PAR HASARD — le
// defaut exact que `sbsp_region.go` a deja documente sur les six canevas Forge, ou le plus gros
// tag est le decor lointain.
//
// LE CRITERE RETENU EST L'EMPRISE AU SOL. Parmi les boites qui contiennent toutes les ancres,
// la plus petite est la plus SPECIFIQUE : elle contient l'aire de jeu et rien d'autre. Le
// rapport le plus serre du tableau vaut 92 — il n'y a aucune ambiguite numerique a trancher.
//
// LE CRITERE D'ALTITUDE A ETE ESSAYE, ET IL EST REFUTE. « Retenir le bsp dont la geometrie est
// la plus proche du niveau de jeu » designe le BON bsp sur quatre temoins — ecart entre la
// mediane des z d'instances et la mediane des ancres : 1,1 m contre 10,4 sur forbidden ; 1,1
// contre 76,4 sur catalyst ; 0,2 contre 147,6 sur cliffhanger ; 1,9 contre 27,6 sur streets —
// mais le MAUVAIS sur le seul cas a corriger : sur `common-rtx-new`, l'horizon est a 0,5 m de la
// mediane des ancres de Live Fire quand l'arene est a 1,7 m. Son terrain est au niveau du sol,
// et la mediane d'un decor de 7 890 m de cote ne dit rien de l'endroit ou l'on joue.
//
// UN BSP SANS INSTANCE NE GAGNE JAMAIS UNE EGALITE : `ReadModuleInstances` conserve les tags
// dont le bloc d'instances est vide, et une boite vide plus serree que l'arene rendrait un fond
// blanc — une regression que le critere d'emprise seul rendrait possible.
//
// Le repli sur le bsp le plus peuple ne sert que si AUCUNE ancre ne tombe dans aucune boite.
func ChoisitBSP(bsps []BSPInstances, ancres [][3]float64) BSPInstances {
	if retenu, ok := bspParLesAncres(bsps, ancres); ok {
		return retenu
	}
	var meilleur BSPInstances
	for _, b := range bsps {
		if len(b.Instances) > len(meilleur.Instances) {
			meilleur = b
		}
	}
	return meilleur
}

// bspParLesAncres rend le bsp designe par les ancres, et dit si les ancres ont tranche.
func bspParLesAncres(bsps []BSPInstances, ancres [][3]float64) (BSPInstances, bool) {
	if len(ancres) == 0 {
		return BSPInstances{}, false
	}
	var retenu BSPInstances
	mieux := 0
	for _, b := range bsps {
		n := CompteAncresDansBoite(b.Bounds, ancres)
		if n == 0 || n < mieux {
			continue
		}
		if n > mieux || plusSpecifique(b, retenu) {
			mieux, retenu = n, b
		}
	}
	return retenu, mieux > 0
}

// CompteAncresDansBoite compte les ancres a l'interieur de la boite monde d'un bsp.
func CompteAncresDansBoite(b Bounds, ancres [][3]float64) int {
	n := 0
	for _, a := range ancres {
		dedans := true
		for k := 0; k < 3; k++ {
			if a[k] < b.Min[k] || a[k] > b.Max[k] {
				dedans = false
				break
			}
		}
		if dedans {
			n++
		}
	}
	return n
}

// plusSpecifique dit si `c` doit l'emporter sur `tenant` A NOMBRE D'ANCRES EGAL : la geometrie
// prime sur le vide, puis la boite la plus serree prime sur la plus large.
func plusSpecifique(c, tenant BSPInstances) bool {
	if (len(c.Instances) > 0) != (len(tenant.Instances) > 0) {
		return len(c.Instances) > 0
	}
	return EmpriseAuSol(c.Bounds) < EmpriseAuSol(tenant.Bounds)
}

// EmpriseAuSol est l'aire, en metres carres, de la boite monde d'un bsp vue de dessus. Une
// emprise non mesurable vaut +Inf : elle ne peut alors jamais gagner un departage.
//
// Les DEUX etendues sont controlees separement : une boite inversee sur les deux axes rendrait
// un produit POSITIF et se ferait passer pour la plus serree de toutes.
func EmpriseAuSol(b Bounds) float64 {
	x, y := b.Extent(0), b.Extent(1)
	if !(x > 0) || !(y > 0) || math.IsInf(x, 0) || math.IsInf(y, 0) {
		return math.Inf(1)
	}
	return x * y
}
