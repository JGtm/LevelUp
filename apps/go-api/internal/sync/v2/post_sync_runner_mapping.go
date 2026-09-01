// Package v2 — post_sync_runner_mapping.go : mapping V1 domain.PostSyncResult
// → V2 PlayerPostSyncResult.
//
// Fichier séparé pour isoler la dépendance à internal/domain (cf. note
// dans post_sync_runner.go). Init() installe le mapper dans la variable
// postSyncResultMapper consultée par mapV1PostSyncResult.
package v2

import (
	"fmt"

	"levelup/go-api/internal/domain"
)

func init() {
	postSyncResultMapper = func(slug string, v1 any) PlayerPostSyncResult {
		out := PlayerPostSyncResult{PlayerSlug: slug}
		res, ok := v1.(domain.PostSyncResult)
		if !ok {
			return out
		}
		out.CitationsComputed = res.CitationsComputed
		out.DominanceFlagsComputed = res.DominanceFlagsComputed
		if res.AchievementsSynced {
			out.AchievementsSynced = 1
		}
		// V1 ne distingue pas StatsHealed/SkillHealed/EventsHealed du
		// reste — ces compteurs restent à 0 en bridge V2 (les WARN sont
		// loggés par V1 directement, donc l'info reste accessible).
		for _, fatal := range res.FatalErrors {
			out.Warnings = append(out.Warnings, fmt.Sprintf("FATAL %s", fatal))
		}
		if len(res.FatalErrors) > 0 {
			out.Err = fmt.Errorf("post-sync fatal errors: %v", res.FatalErrors)
		}
		return out
	}
}
