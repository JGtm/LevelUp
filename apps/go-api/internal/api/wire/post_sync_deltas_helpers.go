// Package api — post_sync_deltas_helpers.go : helpers purs (signature skill-tier,
// tri stable des clés playlist). Extrait de post_sync_deltas.go (refactor god-file).
package wire

import (
	"fmt"
	"sort"
)

func sortedPlaylistKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// splitSkillTier décompose une signature "rating_type|tier|sub_tier" en
// composants. Renvoie ("", "", 0) sur entrée vide ou mal formée — ce qui
// laisse les params previous_* à zéro pour une 1re apparition de playlist.
func splitSkillTier(sig string) (ratingType, tier string, subTier int) {
	if sig == "" {
		return "", "", 0
	}
	var sub int
	parts := [3]string{}
	idx := 0
	start := 0
	for i := 0; i < len(sig) && idx < 3; i++ {
		if sig[i] == '|' {
			parts[idx] = sig[start:i]
			idx++
			start = i + 1
		}
	}
	if idx < 3 {
		parts[idx] = sig[start:]
	}
	_, _ = fmt.Sscanf(parts[2], "%d", &sub)
	return parts[0], parts[1], sub
}
