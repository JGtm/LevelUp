package analysis

import (
	"sort"

	"levelup/go-api/internal/domain"
)

// MapNameCount est ce que le registre des matchs sait d'une carte : son identifiant, son nom EN
// retenu (vrai nom, jamais NULL ni l'identifiant recopié ; "" quand aucun match n'en porte un) et
// le nombre de matchs joués dessus.
type MapNameCount struct {
	MapID   string
	Name    string
	Matches int
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

// GroupMapsByBase regroupe les cartes d'un registre sous leur carte de base.
//
// Une carte est la variante d'une base quand `BaseMapName(nom, suffixes)` rend un nom qui est LUI-
// MÊME le nom d'au moins une carte du registre : une variante sans carte de base reste seule, rien
// n'est deviné. Le groupe d'une base réunit tous les identifiants qui portent son nom (une carte
// republiée en a plusieurs) et ceux de ses variantes. Une carte sans variante n'est jamais
// regroupée, même si plusieurs identifiants portent son nom : seul le rattachement déclaré par
// map_variants.toml fait un groupe.
//
// La clé est le nom retenu du registre et non le nom de chaque ligne : une carte dont certaines
// lignes sont sans nom ou portent l'identifiant recopié a un seul nom retenu, donc un seul groupe.
// Une carte sans nom retenu reste seule. Sans suffixe : aucun groupe.
func GroupMapsByBase(maps []MapNameCount, suffixes []string) MapGroups {
	out := MapGroups{Bases: map[string]MapGroupBase{}}
	if len(suffixes) == 0 || len(maps) == 0 {
		return out
	}
	idsByName := make(map[string][]MapNameCount)
	for _, m := range maps {
		if m.MapID != "" && m.Name != "" {
			idsByName[m.Name] = append(idsByName[m.Name], m)
		}
	}

	// Noms de base ayant au moins une variante au registre : eux seuls forment un groupe.
	variantsOf := make(map[string][]string) // nom de base -> noms variantes
	for name := range idsByName {
		if base := finalBaseName(name, suffixes, idsByName); base != name {
			variantsOf[base] = append(variantsOf[base], name)
		}
	}

	baseOf := make(map[string]string)
	for baseName, variantNames := range variantsOf {
		members := append([]MapNameCount(nil), idsByName[baseName]...)
		baseIDs := sortedByPlayed(members)
		rep := baseIDs[0].MapID
		ids := make([]string, 0, len(baseIDs))
		for _, m := range baseIDs {
			ids = append(ids, m.MapID)
			baseOf[m.MapID] = rep
		}
		sort.Strings(ids)
		for _, v := range variantNames {
			for _, m := range idsByName[v] {
				baseOf[m.MapID] = rep
			}
		}
		out.Bases[rep] = MapGroupBase{ID: rep, Name: baseName, IDs: ids}
	}
	out.Groups = domain.NewGroupesDeCartes(baseOf)
	return out
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
