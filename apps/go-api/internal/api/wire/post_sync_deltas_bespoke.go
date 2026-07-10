// Package api — post_sync_deltas_bespoke.go : émetteurs bespoke des deltas
// post-sync (career_rank, skill_tier). Extraits de post_sync_deltas.go
// (seuil fichier 500 L / fonction 80 L).
package wire

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/notifications"
)

// emitCareerRankDelta émet career_rank sur montée du rang Halo lifetime
// (career_progression). Remplace l'ancien câblage CategorySeasonPassLevel —
// déprécié depuis 2026-05-16. Extrait d'EmitPostSyncDeltas (seuil 80 L).
//
// A4 : ne jamais émettre depuis un rang « inconnu » (before=0). L'incident
// cold-start portait previous:0 — un rang non initialisé n'est pas une montée.
func emitCareerRankDelta(
	ctx context.Context, emitter notifications.Emitter, slug string,
	before, after *PlayerSnapshot,
) {
	if after.CurrentRank <= before.CurrentRank || after.CurrentRank <= 0 {
		return
	}
	if before.CurrentRank == 0 {
		slog.DebugContext(ctx, "post_sync: career_rank previous=0 — émission supprimée",
			"slug", slug, "rank", after.CurrentRank)
		return
	}
	if err := emitter.Emit(ctx, notifications.EmitInput{
		Category: notifications.CategoryCareerRank,
		Severity: notifications.SeveritySuccess,
		TitleKey: "notif.career_rank.title",
		BodyKey:  "notif.career_rank.body",
		Params: map[string]any{
			"rank":      after.CurrentRank,
			"rank_name": rankSubRoman(after.CurrentRankName),
			"previous":  before.CurrentRank,
		},
		TargetRoute: fmt.Sprintf("/players/%s/career", slug),
		Source:      postSyncSource,
	}); err != nil {
		slog.WarnContext(ctx, "post_sync: career_rank", "err", err)
	}
}

// emitSkillTierDeltas émet skill_tier (CSR / LUSR unifié) : une notif par
// playlist_group dont le tier|sub_tier|rating_type a changé entre les 2
// snapshots. Extrait d'EmitPostSyncDeltas (seuil 80 L).
//
// B9/DP4 : montées uniquement entre deux rangs CONNUS (pas de démotion ni de
// mouvement latéral — flapping Or IV↔V). Rang inconnu d'un côté (tier
// exotique/multi-titre) → fail-open. Playlist nouvelle (oldVal == "", hors
// cold-start déjà écarté) → toujours émettre.
// B10/DP4 : dédup 24 h par (playlist_group, valeur cible).
func emitSkillTierDeltas(
	ctx context.Context, emitter notifications.Emitter, slug string,
	before, after *PlayerSnapshot, o PostSyncDeltaOptions,
) {
	for _, playlist := range sortedPlaylistKeys(after.SkillTierByPlaylist) {
		newVal := after.SkillTierByPlaylist[playlist]
		oldVal := before.SkillTierByPlaylist[playlist]
		if newVal == oldVal {
			continue
		}
		ratingType, tier, subTier := splitSkillTier(newVal)
		oldRT, oldTier, oldSub := splitSkillTier(oldVal)
		if oldVal != "" {
			newRank := skillTierRank(tier, subTier)
			oldRank := skillTierRank(oldTier, oldSub)
			if newRank >= 0 && oldRank >= 0 && newRank <= oldRank {
				slog.DebugContext(ctx, "post_sync: skill_tier démotion/latéral — émission supprimée",
					"playlist", playlist, "from", oldVal, "to", newVal)
				continue
			}
		}
		if skillTierAlreadyNotified(o.RecentSkillTiers, playlist, ratingType, tier, subTier, o.Now) {
			slog.DebugContext(ctx, "post_sync: skill_tier déjà notifié < 24 h — supprimé",
				"playlist", playlist, "value", newVal)
			continue
		}
		if err := emitter.Emit(ctx, notifications.EmitInput{
			Category: notifications.CategorySkillTier,
			Severity: notifications.SeveritySuccess,
			TitleKey: "notif.skill_tier.title",
			BodyKey:  "notif.skill_tier.body",
			Params: map[string]any{
				"playlist_group":    playlist,
				"rating_type":       ratingType,
				"tier":              tier,
				"sub_tier":          subTier,
				"previous_type":     oldRT,
				"previous_tier":     oldTier,
				"previous_sub_tier": oldSub,
			},
			TargetRoute: fmt.Sprintf("/players/%s/synthesis", slug),
			Source:      postSyncSource,
		}); err != nil {
			slog.WarnContext(ctx, "post_sync: skill_tier", "playlist", playlist, "err", err)
		}
	}
}
