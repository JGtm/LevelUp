// Package analysis — perfect_kills.go : source unique des identifiants de
// médailles « frag parfait » (perfect kill) PAR TITRE.
//
// Pourquoi ici (et pas dans duckdb/service) : `internal/analysis` est un leaf
// déjà importé par `internal/platform/duckdb` ET `internal/service`, et il
// n'importe NI l'un NI l'autre → aucun cycle. Centraliser le set ici élimine le
// magic number `1512363953` qui était dupliqué dans ~10 requêtes/builders.
package analysis

import (
	"sort"
	"strconv"
	"strings"

	title "levelup/go-api/internal/domain/title"
)

// perfectKillMedalIDsByTitle mappe un slug de titre vers la liste des
// medal_name_id considérés comme « frag parfait » pour ce titre.
//
//   - halo_infinite : une seule médaille « Perfect » (1512363953).
//   - halo_5 : SIX médailles « Perfect Kill » (toutes nommées « Perfect Kill » /
//     « Frag parfait » dans le catalogue H5). Elles doivent être AGRÉGÉES car le
//     même concept produit est éclaté en plusieurs ids natifs.
//     NB : la médaille « Perfection » (3592822316, match sans-mort) est
//     VOLONTAIREMENT exclue — ce n'est pas un frag parfait mais une récompense
//     de fin de partie.
var perfectKillMedalIDsByTitle = map[string][]int64{
	"halo_infinite": {1512363953},
	"halo_5": {
		1080468863,
		3653057799,
		370413844,
		2279899989,
		3098362934,
		3992195104,
	},
}

// PerfectKillMedalIDs retourne les medal_name_id « frag parfait » pour le titre
// `titleSlug`. Un slug inconnu ou vide retombe sur le set du titre par défaut
// (title.DefaultSlug = halo_infinite) pour ne RIEN changer aux appels actuels.
//
// La slice retournée est triée de façon stable et est une copie (le caller peut
// la muter sans corrompre la source).
func PerfectKillMedalIDs(titleSlug string) []int64 {
	slug := strings.TrimSpace(titleSlug)
	ids, ok := perfectKillMedalIDsByTitle[slug]
	if !ok {
		ids = perfectKillMedalIDsByTitle[title.DefaultSlug]
	}
	out := make([]int64, len(ids))
	copy(out, ids)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// PerfectKillMedalInClause construit le fragment SQL `<col> IN (id1, id2, ...)`
// pour le titre `titleSlug`. Les ids sont des CONSTANTES int64 (jamais d'entrée
// utilisateur) → embedding direct sûr, pas de placeholders nécessaires.
//
// `col` doit être un littéral de confiance (nom de colonne, ex.
// "medal_name_id" ou "me.medal_name_id"). Pour HINF le résultat est
// byte-identique à un `<col> = 1512363953` (set d'un seul id → `IN (1512363953)`,
// sémantiquement et au plan d'exécution équivalent).
func PerfectKillMedalInClause(col, titleSlug string) string {
	ids := PerfectKillMedalIDs(titleSlug)
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.FormatInt(id, 10)
	}
	return col + " IN (" + strings.Join(parts, ", ") + ")"
}
