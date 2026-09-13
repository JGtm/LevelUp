package coordination

// distances.go — LA FORME DE LA DISTANCE A L'EQUIPIER AU MOMENT DE MES MORTS.
//
// Le TAUX d'isolement (isolation.go) dit COMBIEN de morts sont survenues sans equipier a
// portee. Il ne dit pas de combien on etait loin : deux joueurs a 63 % d'isolement dont
// l'un meurt a 26 m et l'autre a 70 m ne jouent pas le meme jeu. Ce fichier rend la
// MEDIANE et la DISTRIBUTION de cette distance.
//
// # UNE DISTANCE ABSENTE N'ENTRE PAS
//
// `PlusProcheM` est nil quand aucun coequipier n'etait VISIBLE : il n'y a rien a mesurer.
// Ces morts sont COMPTEES A PART (`MortsSansDistance`) et n'entrent ni dans la mediane ni
// dans un intervalle — les verser dans « 50+ » inventerait une mesure (cf. la doc de
// domain.MortAExaminer.PlusProcheM : « une absence de mesure, jamais une distance
// infinie »).
//
// # LE MEME UNIVERS QUE LE TAUX
//
// Un match sans portee de radar connue est HORS de la lecture d'isolement (correction G2) :
// ses morts n'entrent pas plus ici. Et une mort « equipe a terre » (personne ne pouvait
// accompagner) ne dit rien du placement : elle sort du denominateur, ici comme la-bas.

import (
	"sort"

	"levelup/go-api/internal/domain"
)

// Distances rend la mediane et l'histogramme des distances mesurees, sur EXACTEMENT les
// morts que `Isolement` examine : celles d'un match dont la portee est connue, et dont au
// moins un coequipier pouvait accompagner.
func Distances(morts []domain.MortAExaminer, rayonParMatch map[string]float64) domain.TaCoordDistances {
	mesurees := make([]float64, 0, len(morts))
	sans := 0
	for _, m := range morts {
		rayon, connu := rayonParMatch[m.MatchID]
		if !connu || rayon <= 0 || m.EquipeATerre() {
			continue
		}
		if m.PlusProcheM == nil {
			sans++
			continue
		}
		mesurees = append(mesurees, *m.PlusProcheM)
	}
	sort.Float64s(mesurees)
	return domain.TaCoordDistances{
		Mediane:           medianeTriee(mesurees),
		Distribution:      histogramme(mesurees, domain.TacticalBornesDistanceM),
		N:                 len(mesurees),
		MortsSansDistance: sans,
	}
}

// medianeTriee rend la mediane d'une suite DEJA triee croissant, ou nil si elle est vide.
//
// MOYENNE DES DEUX CENTRALES SUR UN EFFECTIF PAIR : c'est la definition, et sur une
// grandeur continue comme une distance elle ne fabrique aucune valeur aberrante.
func medianeTriee(v []float64) *float64 {
	n := len(v)
	if n == 0 {
		return nil
	}
	var m float64
	if n%2 == 1 {
		m = v[n/2]
	} else {
		m = (v[n/2-1] + v[n/2]) / 2
	}
	return &m
}

// histogramme ventile `valeurs` (triees) dans les intervalles definis par `bornes`.
//
// LE DERNIER INTERVALLE EST OUVERT : tout ce qui depasse la derniere borne y tombe. Les
// intervalles VIDES SONT SERVIS (N = 0) — un histogramme dont les colonnes vides
// disparaissent change de forme d'un filtre a l'autre, et ne se compare plus.
func histogramme(valeurs []float64, bornes []float64) []domain.TacticalBinDistance {
	out := make([]domain.TacticalBinDistance, 0, len(bornes))
	for i, min := range bornes {
		bin := domain.TacticalBinDistance{MinM: min}
		if i+1 < len(bornes) {
			max := bornes[i+1]
			bin.MaxM = &max
		}
		out = append(out, bin)
	}
	if len(out) == 0 {
		return out
	}
	for _, v := range valeurs {
		out[indexDuBin(v, bornes)].N++
	}
	return out
}

// indexDuBin rend l'intervalle d'une valeur : borne basse INCLUSE, borne haute EXCLUE.
// Une valeur sous la premiere borne tombe dans le premier intervalle — une distance est
// positive par construction, et la premiere borne vaut zero.
func indexDuBin(v float64, bornes []float64) int {
	for i := len(bornes) - 1; i >= 0; i-- {
		if v >= bornes[i] {
			return i
		}
	}
	return 0
}
