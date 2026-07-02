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

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// MatchRecomputer recalcule performance_score + LUSR pour un joueur après une
// (dé)exclusion de match. Détient les paths nécessaires à l'acquisition des
// leases d'écriture sur les deux DBs.
//
// sharedProvider (sprint B1 commit 13b) : si non-nil, route les écritures
// shared via Provider.AcquireWriter (coordonne avec le pool joueur et les
// readers HTTP). Si nil, fallback legacy (dblease + OpenSharedDB).
type MatchRecomputer struct {
	playerDBPath   string
	sharedDBPath   string
	metadataDBPath string
	xuid           string
	gamertag       string
	sharedProvider sharedprovider.Provider
}

// NewMatchRecomputer construit un MatchRecomputer pour un joueur donné.
// metadataDBPath peut être vide → recompute fonctionne sans bonus médailles.
// provider peut être nil → mode legacy (cf. AcquireSharedWriterStandalone).
func NewMatchRecomputer(
	playerDBPath, sharedDBPath, metadataDBPath, xuid, gamertag string,
	provider sharedprovider.Provider,
) *MatchRecomputer {
	return &MatchRecomputer{
		playerDBPath:   playerDBPath,
		sharedDBPath:   sharedDBPath,
		metadataDBPath: metadataDBPath,
		xuid:           xuid,
		gamertag:       gamertag,
		sharedProvider: provider,
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

	playerHandle, err := OpenPlayerDB(r.playerDBPath)
	if err != nil {
		return fmt.Errorf("MatchRecomputer.RecomputeAfterExclusion OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()

	// Sprint B1 commit 13b : helper standalone (Provider en B-swap, legacy sinon).
	sharedSQL, releaseShared, err := AcquireSharedWriterStandalone(ctxkeys.WithDBWriterLabel(ctx, "match_exclusion_recompute"), r.sharedProvider, r.sharedDBPath)
	if err != nil {
		return fmt.Errorf("MatchRecomputer.RecomputeAfterExclusion: %w", err)
	}
	defer releaseShared()

	playerSQL := playerHandle.SQLDb()

	medalMap := loadMedalExploitMap(ctx, r.metadataDBPath, sharedSQL, r.xuid)

	updatedPerf, err := batchComputePerformanceScores(ctx, playerSQL, sharedSQL, r.xuid, medalMap, true)
	if err != nil {
		return fmt.Errorf("MatchRecomputer.RecomputeAfterExclusion perf: %w", err)
	}

	updatedLUSR, err := batchComputeLUSR(ctx, playerSQL, sharedSQL, r.xuid, medalMap, true)
	if err != nil {
		return fmt.Errorf("MatchRecomputer.RecomputeAfterExclusion lusr: %w", err)
	}

	slog.InfoContext(ctx, "MatchRecomputer: terminé",
		"match_id", matchID, "gamertag", r.gamertag,
		"updated_perf", updatedPerf, "updated_lusr", updatedLUSR,
		"duration_ms", time.Since(start).Milliseconds())
	return nil
}
