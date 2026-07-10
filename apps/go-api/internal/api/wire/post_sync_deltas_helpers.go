// Package api — post_sync_deltas_helpers.go : helpers purs (signature skill-tier,
// tri stable des clés playlist). Extrait de post_sync_deltas.go (refactor god-file).
package wire

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/notifications"
)

// skillTierBaseRank ordonne les paliers de skill du plus bas au plus haut,
// insensible à la casse. Bronze=1 … Onyx=6 (Halo Infinite), Champion=7 (Halo 5).
// Tier inconnu (LUSR exotique, autre titre) → absent de la map. Ordre confirmé :
// internal/games/halo_5/livesync/csr_mapper.go (Platinum < Diamond < Onyx < Champion).
var skillTierBaseRank = map[string]int{
	"bronze":   1,
	"silver":   2,
	"gold":     3,
	"platinum": 4,
	"diamond":  5,
	"onyx":     6,
	"champion": 7,
}

// skillTierRank convertit un couple (tier, sub_tier) en rang total comparable.
// Rang = base×10 + sub_tier (dans un tier, un sous-palier supérieur est meilleur).
// Retourne -1 si le tier est inconnu → l'appelant fait alors du fail-open
// (émettre sur changement), politique DP4 pour la robustesse multi-titre.
func skillTierRank(tier string, subTier int) int {
	base, ok := skillTierBaseRank[strings.ToLower(strings.TrimSpace(tier))]
	if !ok {
		return -1
	}
	return base*10 + subTier
}

// skillTierAlreadyNotified retourne vrai si une notif skill_tier a déjà été
// émise dans les 24 h pour le même playlist_group ET la même valeur cible
// (rating_type|tier|sub_tier). Dédup B10/DP4 : le flapping Or IV↔V ne notifie
// qu'une fois par jour et par cible.
func skillTierAlreadyNotified(
	recent []notifications.Notification,
	playlist, ratingType, tier string, subTier int,
	now time.Time,
) bool {
	cutoff := now.Add(-24 * time.Hour)
	for _, n := range recent {
		if n.Category != notifications.CategorySkillTier || n.CreatedAt.Before(cutoff) {
			continue
		}
		var p struct {
			PlaylistGroup string `json:"playlist_group"`
			RatingType    string `json:"rating_type"`
			Tier          string `json:"tier"`
			SubTier       int    `json:"sub_tier"`
		}
		if len(n.Params) == 0 || json.Unmarshal(n.Params, &p) != nil {
			continue
		}
		if p.PlaylistGroup == playlist && p.RatingType == ratingType &&
			p.Tier == tier && p.SubTier == subTier {
			return true
		}
	}
	return false
}

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
