// Package sync — match_recomputer.go : recompute perf_score + LUSR après une
// exclusion/réactivation manuelle de match.
//
// Sémantique : lorsque l'utilisateur (dé)exclut un match via PATCH
// /matches/{id}/exclusion, le flag `is_excluded` est mis à jour dans
// player_match_enrichment. Les batches `batchComputePerformanceScores` et
// `batchComputeLUSR` filtrent les matchs exclus en amont via
// `loadExcludedMatchIDs`. Il faut donc relancer ces deux batches en mode
// force=true pour propager le changement aux matchs ultérieurs de la même
// chaîne / playlist_group.
//
// Choix d'implémentation : recompute global force=true plutôt que ciblé par
// chaîne+fromTime. Justification :
//   - La cascade TrueSkill LUSR doit de toute façon repartir du premier match
//     du playlist_group concerné pour reconstituer l'état mu/sigma (pas de
//     reprise en milieu de chaîne).
//   - Les chaînes/groupes non impactés produisent des scores identiques sous
//     force=true (idempotent par construction) → coût marginal en lecture
//     seulement, pas de divergence de valeurs.
//   - Implémentation drastiquement plus simple, sans dupliquer la logique des
//     batches.
//
// Volumétrie typique : 5000-10000 matchs total → ~2-10s en synchrone.
// Si la latence pose problème, ré-introduire un mode ciblé via paramètre.
package sync

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/platform/dblease"
)

// MatchRecomputer recalcule performance_score + LUSR pour un joueur après une
// (dé)exclusion de match. Détient les paths nécessaires à l'acquisition des
// leases d'écriture sur les deux DBs.
type MatchRecomputer struct {
	playerDBPath   string
	sharedDBPath   string
	metadataDBPath string
	xuid           string
	gamertag       string
}

// NewMatchRecomputer construit un MatchRecomputer pour un joueur donné.
// metadataDBPath peut être vide → recompute fonctionne sans bonus médailles.
func NewMatchRecomputer(playerDBPath, sharedDBPath, metadataDBPath, xuid, gamertag string) *MatchRecomputer {
	return &MatchRecomputer{
		playerDBPath:   playerDBPath,
		sharedDBPath:   sharedDBPath,
		metadataDBPath: metadataDBPath,
		xuid:           xuid,
		gamertag:       gamertag,
	}
}

// RecomputeAfterExclusion relance les batches perf_score + LUSR en force=true
// pour propager l'effet d'une (dé)exclusion. matchID est utilisé pour les logs
// uniquement (le filtrage se fait via le flag `is_excluded` déjà persisté).
func (r *MatchRecomputer) RecomputeAfterExclusion(ctx context.Context, matchID string) error {
	start := time.Now()
	slog.InfoContext(ctx, "MatchRecomputer: démarrage",
		"match_id", matchID, "gamertag", r.gamertag, "xuid", r.xuid)

	writerPlayer, err := dblease.AcquireWriterCtx(ctx, nil, r.playerDBPath, dblease.KindPlayer)
	if err != nil {
		return fmt.Errorf("MatchRecomputer.RecomputeAfterExclusion lease player: %w", err)
	}
	defer writerPlayer.Release()

	writerShared, err := dblease.AcquireWriterCtx(ctx, nil, r.sharedDBPath, dblease.KindSharedMatches)
	if err != nil {
		return fmt.Errorf("MatchRecomputer.RecomputeAfterExclusion lease shared: %w", err)
	}
	defer writerShared.Release()

	playerHandle, err := OpenPlayerDB(r.playerDBPath)
	if err != nil {
		return fmt.Errorf("MatchRecomputer.RecomputeAfterExclusion OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()

	sharedHandle, err := OpenSharedDB(r.sharedDBPath)
	if err != nil {
		return fmt.Errorf("MatchRecomputer.RecomputeAfterExclusion OpenSharedDB: %w", err)
	}
	defer sharedHandle.Close()

	playerSQL := playerHandle.SQLDb()
	sharedSQL := sharedHandle.SQLDb()

	medalMap := loadMedalExploitMap(ctx, r.metadataDBPath, sharedSQL, r.xuid)

	updatedPerf, err := batchComputePerformanceScores(playerSQL, sharedSQL, r.xuid, medalMap, true)
	if err != nil {
		return fmt.Errorf("MatchRecomputer.RecomputeAfterExclusion perf: %w", err)
	}

	updatedLUSR, err := batchComputeLUSR(playerSQL, sharedSQL, r.xuid, medalMap, true)
	if err != nil {
		return fmt.Errorf("MatchRecomputer.RecomputeAfterExclusion lusr: %w", err)
	}

	slog.InfoContext(ctx, "MatchRecomputer: terminé",
		"match_id", matchID, "gamertag", r.gamertag,
		"updated_perf", updatedPerf, "updated_lusr", updatedLUSR,
		"duration_ms", time.Since(start).Milliseconds())
	return nil
}
