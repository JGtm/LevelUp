// Package analysis â€” home_recent.go : projection des matchs rÃ©cents pour la
// timeline home (BuildRecentMatches*) + helpers nullable et URL d'image map.
package analysis

import (
	"strings"
)

// mapImageURLFromRegistry retourne *string si la home_repo a résolu une URL
// depuis map_images_registry, nil sinon. Pas de fallback name-based : un
// map_id absent du registry signale un asset à indexer via cmd/migrate-static-maps,
// pas une URL à fabriquer côté analyse (le name peut être un UUID brut ou un
// label localisé qui ne correspond à aucun fichier sur disque).
func mapImageURLFromRegistry(localPath string) *string {
	if strings.TrimSpace(localPath) == "" {
		return nil
	}
	return &localPath
}

// mmrDelta calcule team_mmr - enemy_mmr ; retourne nil si l'un ou l'autre est absent.
func mmrDelta(team, enemy *float64) *float64 {
	if team == nil || enemy == nil {
		return nil
	}
	v := *team - *enemy
	return &v
}

// float64PtrVal retourne la valeur pointÃ©e ou 0 si nil.
func float64PtrVal(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

// intPtrIfPos retourne un pointeur vers v si v > 0, nil sinon.
func intPtrIfPos(v int) *int {
	if v > 0 {
		return &v
	}
	return nil
}
