package killicon

// splatter.go — LE PICTOGRAMME DE L ECRASEMENT.
//
// Un joueur ecrase par un vehicule ne meurt d aucune arme : le decodeur publie pour cette mort
// le tag d effet de COLLISION global du jeu (`0000005f`, `[effet matg 000024a6]` dans
// labels.tsv, classe INCONNU) et la categorie de degat `CollisionDamage`. Le tag ne nomme ni le
// vehicule ni l arme, il ne peut donc pas passer par [Lookup] ; c est la CATEGORIE qui dit
// « ecrase », et l atlas KILL FEED du jeu porte le pictogramme dedie (`killfeed-60`, nomme
// `splatter` a l extraction, `static/weapons-assets/halo_infinite/jeu/index.json`).
//
// Pas de weapon_key : un ecrasement n est pas une arme du registre, il ne produit aucune ligne
// de statistiques d arme — seule l icone est servie, comme pour la melee.

// splatterSprite : stem du pictogramme d ecrasement dans l atlas KILL FEED.
const splatterSprite = "killfeed-60"

// Splatter rend la vignette d une mort par ecrasement. L appelant ne l utilise QUE sur une
// mort dont la categorie de degat est `CollisionDamage`.
func Splatter() Icon {
	return Icon{Sprite: splatterSprite, Genre: GenreClasse}
}
