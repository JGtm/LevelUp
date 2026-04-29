// Package analysis — identity.go : helpers pour identifier les joueurs.
//
// Centralise la convention IsBot (revue 2026-04-29 axe 6 BLOQUANT) :
// auparavant 1 helper Go + 8 fragments SQL `xuid LIKE 'bid(%'` répétés. Le
// commit du jour (engagement B2) a corrigé 4 bugs causés exactement par cette
// duplication.
//
// Pour le filtrage SQL, utiliser `analysis.SQLIsBot` (cf. sql_fragments.go).
package analysis

import "strings"

// botXUIDPrefix est le préfixe d'identifiant utilisé par Halo Infinite pour
// les bots de matchmaking. Documenté empiriquement (cf. xuid_aliases en DB).
const botXUIDPrefix = "bid("

// IsBot retourne true si l'identifiant xuid correspond à un bot de match
// (préfixe `bid(`). Convention propre au sync Halo Infinite mais
// suffisamment universelle pour rester en analysis/ (les services qui
// agrègent doivent exclure les bots).
func IsBot(xuid string) bool {
	return strings.HasPrefix(xuid, botXUIDPrefix)
}
