// Package snapshot — capabilities.go : prédicat de capability local (K3c). slugHasLUSR est
// aussi défini côté sync-root (skill_v2_capability.go) pour engine_postsync/skill : 2 copies
// triviales tolérées (règle ≤2). slugHasFirefightCap/slugProducesWeaponKills vivent déjà ici
// (snapshot_readiness_eval.go).
package snapshot

import "levelup/go-api/internal/domain/title"

func slugHasLUSR(slug string) bool {
	if slug == "" {
		slug = title.DefaultSlug
	}
	desc := title.DefaultRegistry().Get(slug)
	return desc != nil && desc.HasCapability(title.CapLUSR)
}
