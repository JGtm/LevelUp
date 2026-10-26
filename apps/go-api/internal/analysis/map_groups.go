package analysis

import (
	"context"
	"sort"

	"levelup/go-api/internal/domain"
)

// MapNameCount est ce que le registre des matchs sait d'une carte : son identifiant, son nom EN
// retenu (vrai nom, jamais NULL ni l'identifiant recopié ; "" quand aucun match n'en porte un), le
// nombre de matchs joués dessus et sa clé de module.
//
// Module est la clé de la carte INTÉGRÉE au jeu que l'identifiant désigne ("" quand il n'en désigne
// aucune : carte Forge, identifiant inconnu des catalogues). Elle est posée par le lecteur du
// registre (MapModuleResolver) : ce package ne lit aucun catalogue.
type MapNameCount struct {
	MapID   string
	Name    string
	Matches int
	Module  string
}

// MapModuleResolver rend la clé de module de chaque carte du registre, par identifiant. Les cartes
// sans clé sont absentes du résultat. Il est appelé une fois par lecture du registre.
type MapModuleResolver func(ctx context.Context, maps []MapNameCount) map[string]string

// MapGroupRules est ce qui fait de plusieurs identifiants de carte un même groupe : les suffixes de
// variante déclarés par le titre (map_variants.toml) et, facultatif, le résolveur de clé de module
// qui permet de réunir les republications d'une même carte intégrée. Valeur zéro : aucun groupe.
type MapGroupRules struct {
	Suffixes []string
	Modules  MapModuleResolver
}

// Active dit si les règles peuvent former un groupe.
func (r MapGroupRules) Active() bool { return len(r.Suffixes) > 0 || r.Modules != nil }

// WithModules rend les cartes avec leur clé de module posée (les cartes d'origine ne sont pas
// modifiées). Sans résolveur : les cartes telles quelles.
func (r MapGroupRules) WithModules(ctx context.Context, maps []MapNameCount) []MapNameCount {
	if r.Modules == nil || len(maps) == 0 {
		return maps
	}
	modules := r.Modules(ctx, maps)
	out := make([]MapNameCount, len(maps))
	for i, m := range maps {
		m.Module = modules[m.MapID]
		out[i] = m
	}
	return out
}

// MapGroupBase décrit la carte de base d'un groupe : l'identifiant représentatif (le plus joué parmi
// ceux qui portent le nom de base, à égalité le plus petit), le nom EN de la base et tous les
// identifiants qui portent ce nom, triés, représentant compris.
type MapGroupBase struct {
	ID   string
	Name string
	IDs  []string
}

// MapGroups est le résultat du regroupement : la table identifiant -> groupe et, par identifiant
// représentatif, la description de la base.
type MapGroups struct {
	Groups domain.GroupesDeCartes
	Bases  map[string]MapGroupBase
}

// maxVariantDepth borne la remontée variante -> base : « X Heavies - Ranked » a pour base « X
// Heavies », elle-même variante de « X ». Deux pas suffisent pour les suffixes déclarés ; la borne
// protège d'une déclaration qui bouclerait.
const maxVariantDepth = 4

// GroupMapsByBase regroupe les identifiants de carte d'un registre qui désignent le même terrain.
// Deux règles, et aucune autre :
//
//   - VARIANTE. Une carte est la variante d'une base quand `BaseMapName(nom, suffixes)` rend un nom
//     qui est LUI-MÊME le nom d'au moins une carte du registre : une variante sans carte de base reste
//     seule, rien n'est deviné. Le groupe d'une base réunit tous les identifiants qui portent son nom
//     et ceux de ses variantes.
//   - REPUBLICATION. Pour un nom sans variante au registre, les identifiants qui portent ce nom ET la
//     même clé de module (`Module`, la carte intégrée au jeu) forment un groupe : c'est la même carte
//     republiée sous un nouvel asset. Le nom seul ne suffit jamais : sans clé de module, un
//     identifiant reste seul, et deux clés différentes restent deux cartes. Deux versions d'une
//     carte Forge (clé de module vide) ne sont donc jamais réunies par cette règle.
//
// La clé du nom est le nom retenu du registre et non celui de chaque ligne : une carte dont certaines
// lignes sont sans nom ou portent l'identifiant recopié a un seul nom retenu, donc un seul groupe.
// Une carte sans nom retenu reste seule. Le représentant d'un groupe est son identifiant le plus
// joué, et son libellé le nom qu'il porte. Sans règle exploitable (ni suffixe, ni clé de module) :
// aucun groupe.
func GroupMapsByBase(maps []MapNameCount, suffixes []string) MapGroups {
	out := MapGroups{Bases: map[string]MapGroupBase{}}
	if len(maps) == 0 {
		return out
	}
	idsByName := make(map[string][]MapNameCount)
	for _, m := range maps {
		if m.MapID != "" && m.Name != "" {
			idsByName[m.Name] = append(idsByName[m.Name], m)
		}
	}

	// Noms de base ayant au moins une variante au registre : eux seuls forment un groupe de variantes.
	variantsOf := make(map[string][]string) // nom de base -> noms variantes
	isVariant := make(map[string]bool)
	if len(suffixes) > 0 {
		for name := range idsByName {
			if base := finalBaseName(name, suffixes, idsByName); base != name {
				variantsOf[base] = append(variantsOf[base], name)
				isVariant[name] = true
			}
		}
	}

	baseOf := make(map[string]string)
	for baseName, variantNames := range variantsOf {
		var variants []MapNameCount
		for _, v := range variantNames {
			variants = append(variants, idsByName[v]...)
		}
		out.addGroup(baseOf, baseName, idsByName[baseName], variants)
	}
	for name, members := range idsByName {
		if isVariant[name] || len(variantsOf[name]) > 0 {
			continue
		}
		for _, parModule := range republiees(members) {
			out.addGroup(baseOf, name, parModule, nil)
		}
	}
	out.Groups = domain.NewGroupesDeCartes(baseOf)
	return out
}

// addGroup inscrit le groupe d'une base : `base` sont les identifiants qui portent son nom (le plus
// joué est le représentant), `rattaches` ceux qui s'y rangent sans porter son nom.
func (g *MapGroups) addGroup(baseOf map[string]string, name string, base, rattaches []MapNameCount) {
	if len(base) == 0 {
		return
	}
	baseIDs := sortedByPlayed(append([]MapNameCount(nil), base...))
	rep := baseIDs[0].MapID
	ids := make([]string, 0, len(baseIDs))
	for _, m := range baseIDs {
		ids = append(ids, m.MapID)
		baseOf[m.MapID] = rep
	}
	sort.Strings(ids)
	for _, m := range rattaches {
		baseOf[m.MapID] = rep
	}
	g.Bases[rep] = MapGroupBase{ID: rep, Name: name, IDs: ids}
}

// republiees partitionne les identifiants d'un même nom par clé de module et rend les parts d'au
// moins deux identifiants. Un identifiant sans clé n'est dans aucune part.
func republiees(members []MapNameCount) [][]MapNameCount {
	byModule := make(map[string][]MapNameCount)
	for _, m := range members {
		if m.Module != "" {
			byModule[m.Module] = append(byModule[m.Module], m)
		}
	}
	var parts [][]MapNameCount
	for _, part := range byModule {
		if len(part) >= 2 {
			parts = append(parts, part)
		}
	}
	return parts
}

// finalBaseName remonte de `name` à la base la plus lointaine dont le nom existe au registre, ou
// rend `name` quand il n'est la variante de rien.
func finalBaseName(name string, suffixes []string, idsByName map[string][]MapNameCount) string {
	for i := 0; i < maxVariantDepth; i++ {
		base, ok := BaseMapName(name, suffixes)
		if !ok {
			return name
		}
		if _, exists := idsByName[base]; !exists {
			return name
		}
		name = base
	}
	return name
}

// sortedByPlayed trie les cartes par matchs décroissants puis identifiant croissant : le premier est
// le représentant.
func sortedByPlayed(maps []MapNameCount) []MapNameCount {
	sort.Slice(maps, func(i, j int) bool {
		if maps[i].Matches != maps[j].Matches {
			return maps[i].Matches > maps[j].Matches
		}
		return maps[i].MapID < maps[j].MapID
	})
	return maps
}
