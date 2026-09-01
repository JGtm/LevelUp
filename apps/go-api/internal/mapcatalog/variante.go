package mapcatalog

// variante.go — CHOISIR LE BON `.mvar` DANS UN ASSET, et c'est une fonction PURE parce que
// personne ne doit en refaire une copie.
//
// UN ASSET UGC DE CARTE SERT SOUVENT DEUX `.mvar` : la carte de BASE, nommee d'apres le niveau
// (`btb_highpower.mvar`, `ctf_aquarius.mvar`), et la VARIANTE jouee, nommee `map.mvar`. Les deux
// parsent, les deux rendent des socles plausibles, et leurs socles n'ont RIEN A VOIR.
//
// PREUVE (2026-09-01), par les comptes d'objets que le catalogue enregistre lui-meme :
//
//	Highpower Sentry Defense  catalogue 421 objets · map.mvar 421 · btb_highpower.mvar 524
//	Aquarius - Ranked         catalogue 236 objets · map.mvar 236 · ctf_aquarius.mvar 349
//
// Une passe de re-validation qui avait pris « le plus gros fichier » a produit des socles
// deplaces de 22 a 80 METRES sur neuf cartes — des chiffres qui ne decrivaient aucune mise a
// jour du jeu, mais la carte de BASE plaquee sur la variante. Avec `map.mvar`, sept de ces neuf
// ecarts disparaissent et six cartes retombent socle pour socle.
//
// LA FONCTION EST ICI, ET UNE SEULE FOIS. Le client Halo et la CLI de generation choisissent le
// meme fichier ou ils ne choisissent rien : deux implementations de cette regle divergeraient,
// et la divergence est precisement ce qui a coute les 80 metres.

import "path"

// NomDeLaVariante est le nom que porte, dans un asset UGC, le fichier de la VARIANTE JOUEE.
const NomDeLaVariante = "map.mvar"

// ChoisirFichierVariante rend, parmi les chemins d'un asset, celui de la variante jouee.
//
// ORDRE DE PREFERENCE, et il n'est pas negociable :
//
//  1. `map.mvar` — la VARIANTE, cf. l'en-tete ;
//  2. le fichier DECLARE par le catalogue d'objectifs ;
//  3. le premier de la liste, faute de mieux.
//
// Le fichier declare vient APRES parce qu'il peut nommer la carte de BASE : le catalogue
// d'objectifs enregistre, pour plusieurs cartes, le nom du niveau et non celui de la variante.
//
// Rend la chaine vide sur une liste vide — l'appelant doit le traiter, il n'y a pas de defaut
// raisonnable a inventer.
func ChoisirFichierVariante(chemins []string, declare string) string {
	if len(chemins) == 0 {
		return ""
	}
	choisi := chemins[0]
	for _, p := range chemins {
		if path.Base(p) == declare {
			choisi = p
		}
	}
	for _, p := range chemins {
		if path.Base(p) == NomDeLaVariante {
			return p
		}
	}
	return choisi
}

// EstFichierDeVariante dit si un nom de fichier est celui de la variante jouee. Utile aux
// chemins qui travaillent sur des noms deja aplatis plutot que sur les chemins d'un asset.
func EstFichierDeVariante(nom string) bool { return path.Base(nom) == NomDeLaVariante }
