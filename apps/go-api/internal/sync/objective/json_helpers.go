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

// extractPlayerXUID extrait le XUID depuis une map joueur (Xuid/PlayerId = "xuid(1234)"),
// avec fallback PlayerProperties[0].PlayerId. Retourne "" si absent.
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

// cleanXUID retire le préfixe/suffixe "xuid(…)".
func cleanXUID(raw string) string {
	raw = strings.TrimPrefix(raw, "xuid(")
	raw = strings.TrimSuffix(raw, ")")
	return raw
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
