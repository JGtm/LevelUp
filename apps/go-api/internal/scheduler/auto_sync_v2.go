// Package scheduler — auto_sync_v2.go : le cycle des joueurs moteur par le pipeline V2 (ADR 0027).
package scheduler

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/notify"
	syncv2 "levelup/go-api/internal/sync/v2"
)

// WithCycleOrchestrator branche le pipeline V2 (ADR 0027), unique moteur de sync
// des joueurs moteur depuis la suppression du pipeline V1 (lot D1c). Câbler un
// orchestrator non-nil active V2 ; nil → filet structurel syncPlayer de boot.
//
// Doit être appelé depuis cmd/server/main.go au boot, avant Run.
func (s *AutoSyncScheduler) WithCycleOrchestrator(o syncv2.CycleOrchestrator) *AutoSyncScheduler {
	s.cycleOrchestrator = o
	return s
}

// shouldUseV2 retourne true si le pipeline V2 (ADR 0027) doit être tenté pour le
// prochain cycle. V2 est désormais le pipeline PAR DÉFAUT : il isole la fenêtre
// d'écriture (Discovery/Fetch en RO, Persist writer court) et supprime la
// contention qui gelait les lectures user-facing pendant un sync V1 (le writer RW
// shared était tenu pendant tout le fetch réseau).
//
// shouldUseV2 indique si le pipeline V2 (orchestrator) pilote le cycle. Depuis la
// suppression du pipeline V1 (lot D1c, ADR 0027), V2 est l'unique moteur : la seule
// condition est que l'orchestrator soit câblé (main.go, parity-complete). Le flag
// LEVELUP_SYNC_PIPELINE (échappatoire de rollback vers V1) et le fallback automatique
// ont été supprimés — un orchestrator non câblé bascule sur le filet syncPlayer de boot.
func (s *AutoSyncScheduler) shouldUseV2() bool {
	return s.cycleOrchestrator != nil
}

// runOnceV2 exécute un cycle complet via le pipeline V2 (ADR 0027).
// Convertit []PlayerSummary → []syncv2.PlayerProfile, appelle l'orchestrator,
// mappe le CycleResult → RunOnceResult.
//
// Retourne (nil, syncv2.ErrNotImplemented) si l'orchestrator est encore le
// stub D0 — le caller doit alors fallback vers V1.
func (s *AutoSyncScheduler) runOnceV2(ctx context.Context, players []domain.PlayerSummary) (*RunOnceResult, error) {
	profiles := make([]syncv2.PlayerProfile, 0, len(players))
	for _, p := range players {
		profiles = append(profiles, syncv2.PlayerProfile{
			Gamertag:   p.Gamertag,
			XUID:       p.XUID,
			PlayerSlug: p.PlayerSlug,
			TitleSlug:  resolveTitleSlug(p), // MT-11 / PMT-3 : porte le titre au pipeline V2
		})
	}

	start := time.Now()
	cycleRes, err := s.cycleOrchestrator.Run(ctx, profiles)
	if err != nil {
		return nil, err
	}
	s.recordV2Passes(players, cycleRes.PerPlayer)

	res := &RunOnceResult{
		Total:    len(players),
		Duration: time.Since(start),
		debt:     cycleDebt{history: s.pendingHistoryAfterV2(ctx, players, cycleRes.PerPlayer)},
	}
	for _, outcome := range cycleRes.PerPlayer {
		switch outcome.Status {
		case "ok", "partial":
			res.Synced++
		case "failed":
			res.Failed++
		default:
			res.Skipped++
		}
		// Collecter les joueurs moteur ayant reçu de nouveaux matchs pour la
		// notification Discord de fin de cycle (le pipeline V2 n'enregistre pas
		// dans playerOutcomes — seul syncPlayer le fait pour les joueurs live/filet).
		if outcome.MatchesInserted > 0 {
			res.newMatchPlayers = append(res.newMatchPlayers, notify.PlayerSyncResult{
				Gamertag:      outcome.Gamertag,
				MatchesSynced: outcome.MatchesInserted,
			})
		}
	}
	slog.InfoContext(ctx, "auto_sync: cycle V2 terminé",
		"total", res.Total,
		"synced", res.Synced,
		"failed", res.Failed,
		"unique_matches", cycleRes.UniqueMatches,
		"duration", res.Duration.Round(time.Millisecond),
	)
	return res, nil
}
