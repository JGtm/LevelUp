package himap

// points_apparition.go — LA RECETTE : reconnaitre un point d'apparition d'objet ramassable
// dans le catalogue Forge, sans connaitre la carte.
//
// POURQUOI UNE RECETTE ET PAS UNE LISTE. Le catalogue des socles reposait sur trois `type_id`
// isoles a la main par appariement contre des artefacts cuits. Sur une carte jamais vue, une
// liste de trois identifiants ne dit pas si elle est complete. Cette fonction repond a la
// question pour n'importe quel `type_id`, en interrogeant les fichiers du jeu.
//
// L'ENONCE, en deux cribles :
//
//	1. le `type_id` resout, dans le catalogue Forge, vers un tag `food` qui reference au moins
//	   un tag du groupe `foki` ;
//	2. aucun de ces `foki` ne reference de `bloc` (geometrie solide) ni de `hsc*` (script).
//
// CE QUE CHAQUE CRIBLE A COUTE, parce que sans ca on les croit arbitraires :
//
//	Le crible 1 seul SUR-RETIENT. Mesure sur 15 cartes : il classait 61,5 % des objets de
//	Highpower et 60,5 % de Scarr comme points d'apparition — invraisemblable. Deux types
//	portaient l'anomalie (`0x8413E9BA` jusqu'a 178 objets sur une carte, `0xA4EE54ED` 83).
//	Le simple comptage ne pouvait PAS les ecarter : le socle `rack`, PROUVE, monte lui-meme a
//	52 objets sur Fragmentation Heavies.
//
//	Le crible 2 les separe net. Les deux aberrants menent a `bloc:4 hsc*:4` ; les cinq etalons
//	(trois socles prouves, deux points mesures) ne menent qu'a `fosp:4 foki:1`. Un point
//	d'apparition NU fait naitre un objet — un objet Forge scripte et solide n'en est pas un.
//	Le discriminant va dans le sens INVERSE de l'hypothese de depart, qui pariait sur la
//	cardinalite.
//
// SELECTIVITE MESUREE : 16 types retenus sur les 4 235 tags `food` du catalogue (0,38 %),
// 5 etalons sur 5 gardes, 0 des 3 types de decor retenu. Sur 15 cartes, les 12 natives tiennent
// dans 61 a 118 points (le crible 1 seul donnait 61 a 322).
//
// CE QUE LA RECETTE NE FAIT PAS : elle dit OU un objet ramassable peut naitre, jamais LEQUEL.
// La chaine de tags ne descend pas jusqu'a l'objet engendre — elle s'arrete au `fosp`, dont les
// references ne resolvent dans aucun module indexe. Le typage passe par la mesure, cote
// `replay/mapvar`.

// GroupeRamassable est le groupe de tag qu'un point d'apparition doit referencer.
const GroupeRamassable = "foki"

// groupesDisqualifiants — les groupes dont la presence SOUS le `foki` refute le point
// d'apparition : de la geometrie solide, ou un script. Voir le crible 2 en tete de fichier.
var groupesDisqualifiants = map[string]bool{"bloc": true, "hsc*": true}

// EstPointDApparition dit si un `type_id` de variante de carte designe un point d'apparition
// d'objet ramassable, et rend le nombre de `foki` distincts qu'il porte.
//
// `idxForge` doit indexer le catalogue Forge (`any/globals/forge/forge_objects-rtx-new.module`)
// et `idxRef` un index ou les tags references se resolvent. LE MODULE COMPTE, et c'est une
// erreur payee : sonder les memes `type_id` contre le chemin de geometrie (56 766 entrees) en
// resout ZERO sur huit, pas meme les trois socles prouves.
func EstPointDApparition(idxForge, idxRef *ModuleIndex, typeID uint32) (bool, int) {
	if idxForge == nil || idxRef == nil {
		return false, 0
	}
	tag, err := idxForge.Extract(typeID)
	if err != nil {
		return false, 0
	}
	// CRIBLE 1 — le type reference au moins un `foki`.
	var fokis []uint32
	vus := make(map[uint32]bool, 4)
	RefsInline(tag, func(h uint32) bool {
		if g, _, ok := idxRef.Lookup(h); ok && g == GroupeRamassable && !vus[h] {
			vus[h] = true
			fokis = append(fokis, h)
		}
		return false
	})
	if len(fokis) == 0 {
		return false, 0
	}
	// CRIBLE 2 — aucun de ces `foki` ne porte de geometrie solide ni de script.
	for _, f := range fokis {
		sous, err := idxRef.Extract(f)
		if err != nil {
			// Un `foki` inextractible ne peut pas etre disculpe : on refuse plutot que de
			// promouvoir un type sur une lecture manquante.
			return false, 0
		}
		disqualifie := false
		RefsInline(sous, func(h uint32) bool {
			if g, _, ok := idxRef.Lookup(h); ok && groupesDisqualifiants[g] {
				disqualifie = true
				return true
			}
			return false
		})
		if disqualifie {
			return false, 0
		}
	}
	return true, len(fokis)
}
