// Package objective — json_helpers.go : petits helpers de navigation JSON locaux.
//
// Copies triviales (règle ≤2) de leurs jumeaux du package sync racine
// (extractPlayerXUID/pve.go, asString+intPtrFrom/transforms_helpers.go). Locales ICI
// parce que le package sync racine importe ce sous-package (objective.ExtractObjectiveStats)
// → l'importer en retour créerait un cycle. Comportement identique aux originaux.
package objective

import "strings"

const (
	jsonKeyPlayerID = "PlayerId"
	jsonKeyXuid     = "Xuid"
)

// extractPlayerXUID extrait la CLÉ joueur depuis une map joueur (Xuid/PlayerId =
// "xuid(1234)" pour un humain, "bid(3.0)" pour un bot), avec fallback
// PlayerProperties[0].PlayerId. Retourne "" si absent ou non reconnu.
func extractPlayerXUID(playerMap map[string]any) string {
	for _, key := range []string{jsonKeyXuid, jsonKeyPlayerID} {
		if raw, ok := playerMap[key].(string); ok && raw != "" {
			return cleanXUID(raw)
		}
	}
	props, _ := playerMap["PlayerProperties"].([]any)
	if len(props) > 0 {
		if first, ok := props[0].(map[string]any); ok {
			if id, ok := first[jsonKeyPlayerID].(string); ok {
				return cleanXUID(id)
			}
		}
	}
	return ""
}

// botKeyPrefix : préfixe des identifiants de bot Halo Infinite ("bid(3.0)"). Le
// prédicat SQL canonique côté lecture est analysis.SQLIs[Not]BotCol → `xuid LIKE
// 'bid(%'` : la forme STOCKÉE d'un bot conserve donc le préfixe ET la parenthèse
// fermante.
const botKeyPrefix = "bid("

// cleanXUID normalise un PlayerId Halo Infinite en la CLÉ écrite dans le shared.
// Parité STRICTE avec sync.extractXUID (transforms_helpers.go), qui produit les
// clés de match_participants :
//   - humain "xuid(1234)" → "1234" ;
//   - bot "bid(3.0)" → "bid(3.0)" (forme intégrale), "bid(3.0" (API sans paren)
//     → "bid(3.0)" ;
//   - toute autre forme → "" (le joueur est sauté, pas de ligne orpheline).
//
// BUG CORRIGÉ (contre-revue V7.2, 2026-07-25) : l'implémentation précédente était
// un simple TrimPrefix("xuid(") + TrimSuffix(")"), qui laissait « bid(3.0 » pour
// un bot — une clé qui ne correspond à AUCUN participant et que le prédicat bot
// `LIKE 'bid(%'` ne rattrape pas non plus côté agrégat. Résultat : des lignes
// définitivement orphelines dans match_objective_stats à chaque match avec bots.
//
// Copie locale assumée (règle ≤2) : le package sync racine importe ce
// sous-package (objective.ExtractObjectiveStats) — l'importer en retour créerait
// un cycle. Garde-rail de parité : TestCleanXUID_ParityWithSyncExtractXUID.
func cleanXUID(raw string) string {
	if strings.HasPrefix(raw, "xuid(") {
		if inner := strings.TrimSuffix(strings.TrimPrefix(raw, "xuid("), ")"); inner != "" {
			return inner
		}
		return ""
	}
	if strings.HasPrefix(raw, botKeyPrefix) {
		if !strings.HasSuffix(raw, ")") {
			return raw + ")"
		}
		return raw
	}
	return ""
}

// asString renvoie v en string, ou "" si nil / pas une string.
func asString(v any) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

// intPtrFrom lit une clé numérique JSON (float64) en *int, nil si absente/non numérique.
func intPtrFrom(m map[string]any, key string) *int {
	v, ok := m[key].(float64)
	if !ok {
		return nil
	}
	n := int(v)
	return &n
}
