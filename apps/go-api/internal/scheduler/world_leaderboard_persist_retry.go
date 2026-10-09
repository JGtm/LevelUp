// Package scheduler — world_leaderboard_persist_retry.go : nouvelle tentative bornée
// de l'acquisition du writer shared par le cron du classement mondial (lot B1,
// décision D-2 du plan `.ai/V7.5/PLAN_BACKLOG_2026-09-26.md`).
//
// CONSTAT : au boot, le scrape (~18 s) se terminait pendant que les lecteurs du
// démarrage tenaient encore le handle RO ; la vidange du provider expirait (5 s),
// `persist` rendait l'erreur et le scrape était PERDU jusqu'au tick suivant (24 h).
//
// POLITIQUE :
//   - seule `sharedprovider.ErrDrainTimeout` est retentée (errors.Is, jamais le texte
//     de l'erreur — ratchet DT-5) : c'est un encombrement transitoire des lecteurs ;
//   - trois tentatives au total, 30 s d'attente entre deux, annulable par le contexte ;
//   - le scrape reste en mémoire chez l'appelant : aucun re-scrape par tentative ;
//   - idempotent : INSERT pur en transaction, rien n'est écrit sur une tentative ratée ;
//   - WARN à chaque nouvelle tentative ; l'ERROR d'échec final reste celui de
//     l'appelant (un seul par échec).
package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

const (
	// worldLeaderboardPersistAttempts : nombre TOTAL d'acquisitions du writer tentées.
	worldLeaderboardPersistAttempts = 3
	// worldLeaderboardPersistRetryDelay : attente entre deux tentatives — six fois la
	// borne de vidange du provider, le temps que les lecteurs du démarrage rendent la main.
	worldLeaderboardPersistRetryDelay = 30 * time.Second
	// worldLeaderboardBootDelay : délai avant le premier cycle de Run (patron
	// assetSweepBootDelay) — laisse passer la rafale de lectures du démarrage.
	worldLeaderboardBootDelay = 2 * time.Minute
)

// worldLeaderboardSleep — seam de paquet pour l'attente entre deux tentatives (patron
// historyretry.Sleep). Les tests le remplacent par un enregistreur : aucune suite ne dort
// pour de vrai. Annulable : un arrêt du serveur n'attend pas la fin du sommeil.
var worldLeaderboardSleep = func(ctx context.Context, d time.Duration) error {
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// acquireWriterRetry acquiert le writer shared sous l'étiquette label, en retentant
// UNIQUEMENT une vidange expirée (cf. en-tête). Sert persist ET persistStats.
func (c *WorldLeaderboardCron) acquireWriterRetry(ctx context.Context, label string) (*sharedprovider.WriterHandle, error) {
	writerCtx := ctxkeys.WithDBWriterLabel(ctx, label)
	for attempt := 1; ; attempt++ {
		wh, err := c.provider.AcquireWriter(writerCtx)
		if err == nil {
			return wh, nil
		}
		if !errors.Is(err, sharedprovider.ErrDrainTimeout) {
			return nil, err
		}
		if attempt == worldLeaderboardPersistAttempts {
			return nil, fmt.Errorf("writer %s : %d tentatives épuisées : %w", label, attempt, err)
		}
		slog.WarnContext(ctx, "world_leaderboard_cron: vidange des lecteurs expirée — nouvelle tentative d'écriture",
			"module", logging.ModuleLeaderboard, "label", label, "attempt", attempt,
			"wait", worldLeaderboardPersistRetryDelay, "err", err)
		if serr := worldLeaderboardSleep(ctx, worldLeaderboardPersistRetryDelay); serr != nil {
			return nil, fmt.Errorf("writer %s : attente avant la tentative %d interrompue (%w) : %w",
				label, attempt+1, serr, err)
		}
	}
}
