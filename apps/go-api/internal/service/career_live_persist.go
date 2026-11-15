package service

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/domain"
)

// persistTracked écrit l'instantané de carrière sous le suivi d'arrêt du process
// (coalesce.Drain, phase « recomputes » de l'arrêt, avant la fermeture des bases).
//
// L'écriture attend l'écrivain de la base joueur au plus careerLivePersistWait : le temps qu'un
// recalcul des notes le rende. Elle est détachée de l'annulation du refresh (dont le budget est
// consommé par les appels à l'API) mais pas de celle de l'arrêt : une fois l'arrêt commencé, une
// écriture en attente ou en cours est annulée et une écriture pas encore lancée est refusée
// (coalesce.ErrDraining), si bien qu'aucun INSERT ne part après la fermeture des bases.
func (s *CareerLiveService) persistTracked(
	ctx context.Context,
	xuid string,
	progress *domain.CareerRankSnapshot,
	custom *domain.SpartanCustomizationData,
	status FetchStatus,
) {
	err := s.persistRuns.Do(context.WithoutCancel(ctx), xuid, func(runCtx context.Context) error {
		persistCtx, cancel := context.WithTimeout(runCtx, careerLivePersistWait)
		defer cancel()
		s.persistPartial(persistCtx, xuid, progress, custom, status)
		return nil
	})
	if err != nil {
		slog.WarnContext(ctx, careerLiveLogModule+": persist partial abandoned by shutdown",
			"xuid", xuid, "err", err)
	}
}

// persistPartial écrit dans career_progression UNIQUEMENT les champs
// effectivement rendus non-vides par l'API live (PartialFromLive). Les
// colonnes omises restent NULL dans la nouvelle ligne — la lecture via
// ARG_MAX FILTER WHERE NOT NULL conserve les valeurs historiques.
//
// Phase 6 PLAN_V2 : status trace l'issue du fetch (ok / api_empty /
// forbidden_403 / auth_missing / failed). Toujours écrit pour permettre
// le diag "pourquoi ce joueur n'a pas de bannière".
//
// Best-effort : une erreur est loggée mais non propagée à l'appelant.
func (s *CareerLiveService) persistPartial(
	ctx context.Context,
	xuid string,
	progress *domain.CareerRankSnapshot,
	custom *domain.SpartanCustomizationData,
	status FetchStatus,
) {
	if s.repo == nil {
		return
	}
	partial := PartialFromLive(progress, custom)
	statusStr := string(status)
	partial.LastFetchStatus = &statusStr

	inserted, err := s.repo.InsertCareerProgressionPartial(ctx, xuid, partial)
	if err != nil {
		slog.WarnContext(ctx, careerLiveLogModule+": persist partial failed",
			"xuid", xuid, "err", err)
		return
	}
	if inserted {
		careerLiveInsertChanged.Add(1)
		slog.InfoContext(ctx, careerLiveLogModule+": partial snapshot inserted",
			"xuid", xuid,
			"status", statusStr,
			"has_rank", partial.Rank != nil,
			"has_xp", partial.CurrentXP != nil,
			"has_banner", partial.BannerImageURL != nil,
			"has_emblem", partial.EmblemImageURL != nil,
			"has_spartan_id", partial.SpartanID != nil)
	} else {
		careerLiveInsertSkipped.Add(1)
	}
}
