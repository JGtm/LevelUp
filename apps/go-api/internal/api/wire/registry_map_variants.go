package wire

import (
	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

// WithMapVariantSuffixes injecte la table PAR TITRE des suffixes de nom qui font d'une carte la
// variante d'une autre (slug -> suffixes), chargée depuis map_variants.toml au boot. Titre sans
// fichier -> absent de la map -> aucune variante. Retourne le registry pour chaînage.
func (r *ServiceRegistry) WithMapVariantSuffixes(byTitle map[string][]string) *ServiceRegistry {
	r.mapVariantSuffixes = byTitle
	return r
}

// mapGroupRulesFor rend les règles qui réunissent plusieurs identifiants de carte du titre du joueur :
// ses suffixes de variante (nil si le titre n'en déclare pas) et, quand le titre a un catalogue de
// cartes intégrées, le résolveur de clé de module qui réunit les republications d'une même carte.
// SOURCE UNIQUE des filtres, de l'historique, des motifs et des statistiques par carte : ils
// regroupent les mêmes identifiants.
func (r *ServiceRegistry) mapGroupRulesFor(pdb *duckdb.PlayerDB) analysis.MapGroupRules {
	rules := analysis.MapGroupRules{Suffixes: r.mapVariantSuffixes[pdb.TitleSlug]}
	if r.cfg != nil {
		if modules, ok := service.MapModuleResolverFor(r.cfg.RepoRoot, pdb.TitleSlug); ok {
			rules.Modules = modules
		}
	}
	return rules
}

// mapGroupsFor rend le lecteur des groupes de cartes du joueur : une carte de base, ses variantes et ses
// republications forment UNE ligne de toute statistique par carte. MÊMES règles que les filtres
// (mapGroupRulesFor) : le regroupement des statistiques et celui des filtres ne peuvent pas diverger.
func (r *ServiceRegistry) mapGroupsFor(pdb *duckdb.PlayerDB) port.MapGroupReader {
	return duckdb.NewMapGroupsRepo(pdb).WithMapGroupRules(r.mapGroupRulesFor(pdb))
}

// PatternsRepo construit le lecteur du Pattern Engine du joueur : ses motifs « par carte » comptent une
// variante ou une republication sous sa carte de base (mêmes règles que les filtres, mapGroupRulesFor).
func (r *ServiceRegistry) PatternsRepo(pdb *duckdb.PlayerDB) port.PatternsRepository {
	return duckdb.NewPatternsRepo(pdb).WithMapGroupRules(r.mapGroupRulesFor(pdb))
}
