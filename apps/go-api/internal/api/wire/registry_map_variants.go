package wire

import "levelup/go-api/internal/platform/duckdb"

// WithMapVariantSuffixes injecte la table PAR TITRE des suffixes de nom qui font d'une carte la
// variante d'une autre (slug -> suffixes), chargée depuis map_variants.toml au boot. Titre sans
// fichier -> absent de la map -> aucune variante. Retourne le registry pour chaînage.
func (r *ServiceRegistry) WithMapVariantSuffixes(byTitle map[string][]string) *ServiceRegistry {
	r.mapVariantSuffixes = byTitle
	return r
}

// mapVariantSuffixesFor rend les suffixes de variante de carte du titre du joueur, nil si le titre
// n'en déclare pas. SOURCE UNIQUE des repos de filtres et d'historique : le filtre de cartes de
// l'Explorateur et celui du pipeline de filtres regroupent les mêmes variantes.
func (r *ServiceRegistry) mapVariantSuffixesFor(pdb *duckdb.PlayerDB) []string {
	return r.mapVariantSuffixes[pdb.TitleSlug]
}
