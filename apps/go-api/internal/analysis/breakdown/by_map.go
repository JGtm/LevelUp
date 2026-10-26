package breakdown

import (
	"sort"

	"levelup/go-api/internal/domain"
)

// MapAggregate est l'agregation par carte.
//
// `MapID` est la cle de regroupement : l'identifiant de la carte de BASE quand la carte appartient a un
// groupe (domain.GroupesDeCartes), son propre identifiant sinon.
//
// `AvgPerformanceScore` est la moyenne globale toutes chaines confondues
// (preservee pour compat ascendante). Sa semantique devient floue depuis que
// le score de performance est relatif a la chaine de chaque match (ex. score
// BTB et score Arena Slayer ne sont pas sur la meme echelle de reference).
//
// `PerfByChain` decoupe la moyenne par chaine (`arena_slayer`, `btb`, ...) :
// c'est la lecture la plus precise pour comparer les performances sur une
// meme carte entre differents contextes de jeu. Vide si aucune row n'a a la
// fois un PerformanceScore et un PerformanceChain non vides.
type MapAggregate struct {
	MapID    string
	MapLabel string
	Counts
	AvgPerformanceScore *float64
	PerfByChain         map[string]*float64
}

// ByMap groupe les rows par carte et calcule les compteurs. `groups` rattache une variante a sa carte
// de base : les rows d'un meme groupe forment UNE ligne. Le parametre est obligatoire pour que chaque
// appelant decide du regroupement ; la valeur zero (aucun groupe) laisse chaque identifiant seul.
//
// Regles :
//   - Les rows avec MapID == "" sont ignorees.
//   - Une carte de groupe porte le libelle de la carte de BASE dans `locale`
//     (groups.Libelle) : jamais celui d'une variante.
//   - Sinon le label retenu est celui de la premiere row rencontree avec un label
//     non vide pour ce MapID. (Convention : si le service envoie des rows
//     coherentes, il n'y aura qu'un seul label par ID. En cas de divergence,
//     premiere wins.)
//   - Tri : WinRate desc puis MapID asc.
func ByMap(rows []Row, groups domain.GroupesDeCartes, locale string) []MapAggregate {
	grouped := make(map[string][]Row)
	labels := make(map[string]string)
	order := make([]string, 0)
	for _, r := range rows {
		if r.MapID == "" {
			continue
		}
		key := groups.Groupe(r.MapID)
		if _, seen := grouped[key]; !seen {
			order = append(order, key)
		}
		grouped[key] = append(grouped[key], r)
		if labels[key] != "" {
			continue
		}
		if base, ok := groups.Libelle(r.MapID, locale); ok {
			labels[key] = base
		} else if r.MapLabel != "" && !groups.Regroupe(r.MapID) {
			labels[key] = r.MapLabel
		}
	}
	out := make([]MapAggregate, 0, len(grouped))
	for _, id := range order {
		items := grouped[id]
		out = append(out, MapAggregate{
			MapID:               id,
			MapLabel:            labelOfGroup(labels[id], items),
			Counts:              computeCounts(items),
			AvgPerformanceScore: avgPerformanceScore(items),
			PerfByChain:         avgPerformanceScoreByChain(items),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].WinRate != out[j].WinRate {
			return out[i].WinRate > out[j].WinRate
		}
		return out[i].MapID < out[j].MapID
	})
	return out
}

// labelOfGroup rend le libelle retenu d'un groupe. Un groupe dont la base n'a aucun libelle (table sans
// libelles) retombe sur le libelle de la premiere row qui en porte un : mieux qu'une ligne sans nom.
func labelOfGroup(label string, items []Row) string {
	if label != "" {
		return label
	}
	for _, r := range items {
		if r.MapLabel != "" {
			return r.MapLabel
		}
	}
	return ""
}
