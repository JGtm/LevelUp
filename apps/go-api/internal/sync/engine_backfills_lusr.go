// Package sync - engine_backfills_lusr.go : simulations LUSR en lecture seule du SyncEngine
// (dry-run du recompute, simulation des variantes de formule) et pose du titre du moteur
// sur le contexte des lecteurs de sync.
package sync

import (
	"context"
	"fmt"

	"levelup/go-api/internal/ctxkeys"
)

// RunBackfillLUSRDryRun simule un recompute LUSR sans écrire en DB. Compare
// l'état persisté actuel avec celui qui serait produit par un recompute force,
// agrégé par playlist_group. Utile pour valider l'impact d'un rebuild ART
// (cf. Phase 1 plan stabilisation 2026-05-22) avant d'engager l'écriture.
//
// Retourne un LUSRDryRunReport. Lock writer non requis (lecture seule) ;
// le caller doit s'assurer qu'aucun sync ne tourne en parallèle pour avoir
// un snapshot cohérent.
func (e *SyncEngine) RunBackfillLUSRDryRun(ctx context.Context) (*LUSRDryRunReport, error) {
	ctx = e.pinTitle(ctx)
	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		return nil, fmt.Errorf("RunBackfillLUSRDryRun OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()

	// shared DB en read-only suffit pour le dry-run.
	sharedDB, releaseShared, err := e.acquireSharedWriter(ctxkeys.WithDBWriterLabel(ctx, "backfill_lusr_dryrun"))
	if err != nil {
		return nil, fmt.Errorf("RunBackfillLUSRDryRun: %w", err)
	}
	defer releaseShared()

	medalMap := e.loadMedalExploitMapBestEffort(ctx, sharedDB)
	return batchComputeLUSRPreview(ctx, playerHandle.SQLDb(), sharedDB, e.xuid, medalMap)
}

// RunFormulaSim simule les 5 variantes de formule LUSR sur les lastN derniers matchs.
// Lecture seule — aucune écriture DB. lastN=0 → tous les matchs.
func (e *SyncEngine) RunFormulaSim(ctx context.Context, lastN int) (*FormulaSimReport, error) {
	ctx = e.pinTitle(ctx)
	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		return nil, fmt.Errorf("RunFormulaSim OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()

	sharedDB, releaseShared, err := e.acquireSharedWriter(ctxkeys.WithDBWriterLabel(ctx, "backfill_formula_sim"))
	if err != nil {
		return nil, fmt.Errorf("RunFormulaSim: %w", err)
	}
	defer releaseShared()

	medalMap := e.loadMedalExploitMapBestEffort(ctx, sharedDB)
	return RunFormulaSim(ctx, playerHandle.SQLDb(), sharedDB, e.xuid, medalMap, lastN)
}

// pinTitle pose sur le ctx le titre du MOTEUR (celui de ses bases ouvertes), jamais celui d'un ctx
// entrant : les lecteurs de sync qui excluent la Campagne le lisent (analysis.SQLExcludeCampaign
// ByMatchIDCtx), et un ctx sans titre, ou portant un autre titre (onglet admin, header
// X-LevelUp-Title), ne doit pas décider de ce qui s'exclut d'une base d'un autre titre.
func (e *SyncEngine) pinTitle(ctx context.Context) context.Context {
	return ctxkeys.WithTitleSlug(ctx, e.titleSlug)
}
