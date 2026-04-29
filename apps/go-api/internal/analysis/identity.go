// Package analysis — identity.go : helpers pour identifier les joueurs.
//
// Centralise la convention IsBot (revue 2026-04-29 axe 6 BLOQUANT) :
// auparavant 1 helper Go + 8 fragments SQL `xuid LIKE 'bid(%'` répétés. Le
// commit du jour (engagement B2) a corrigé 4 bugs causés exactement par cette
// duplication.
//
// Pour le filtrage SQL, utiliser `analysis.SQLIsBot` (cf. sql_fragments.go).
package analysis

import (
	"regexp"
	"strings"
)

// botXUIDPrefix est le préfixe d'identifiant utilisé par Halo Infinite pour
// les bots de matchmaking. Documenté empiriquement (cf. xuid_aliases en DB).
const botXUIDPrefix = "bid("

// botDisplayRE extrait le numéro entier depuis un xuid bot ("bid(3.0)" → "3").
var botDisplayRE = regexp.MustCompile(`^bid\((\d+)`)

// IsBot retourne true si l'identifiant xuid correspond à un bot de match
// (préfixe `bid(`). Convention propre au sync Halo Infinite mais
// suffisamment universelle pour rester en analysis/ (les services qui
// agrègent doivent exclure les bots).
func IsBot(xuid string) bool {
	return strings.HasPrefix(xuid, botXUIDPrefix)
}

// BotDisplayName retourne le nom d'affichage lisible d'un bot depuis son xuid.
// "bid(3.0)" → "343 Bot 3". Retourne le xuid tel quel si le format est inconnu.
func BotDisplayName(xuid string) string {
	if m := botDisplayRE.FindStringSubmatch(xuid); len(m) == 2 {
		return "343 Bot " + m[1]
	}
	return xuid
}
