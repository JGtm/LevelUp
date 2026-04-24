// Package sync — trigger.go : déclencheur de sync in-process.
//
// Adapte le SyncEngine (conçu pour être jetable, 1 instance = 1 sync)
// pour être utilisé par le Coordinator du watcher.
//
// Implémente l'interface SyncRunner du coordinator.
package sync

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/domain"
)

// TokenProvider fournit les tokens Halo nécessaires au sync.
type TokenProvider interface {
	GetTokens(ctx context.Context) (*domain.HaloTokens, error)
}

// Trigger déclenche des syncs in-process via SyncEngine.
// Il crée un SyncEngine jetable à chaque appel.
type Trigger struct {
	repoRoot      string
	tokenProvider TokenProvider
	defaultOpts   domain.SyncOptions
}

// NewTrigger crée un trigger de sync in-process.
func NewTrigger(repoRoot string, tp TokenProvider, defaultOpts domain.SyncOptions) *Trigger {
	return &Trigger{
		repoRoot:      repoRoot,
		tokenProvider: tp,
		defaultOpts:   defaultOpts,
	}
}

// RunSync implémente SyncRunner pour le Coordinator.
// Crée un SyncEngine jetable et lance un RunDelta.
func (t *Trigger) RunSync(ctx context.Context, gamertag, xuid string, matchIDs []string) error {
	slog.InfoContext(ctx, "trigger: démarrage sync",
		"gamertag", gamertag,
		"xuid", xuid,
		"match_ids_hint", len(matchIDs),
	)

	tokens, err := t.tokenProvider.GetTokens(ctx)
	if err != nil {
		return fmt.Errorf("trigger: get tokens: %w", err)
	}

	engine := NewSyncEngine(t.repoRoot, gamertag, xuid, tokens, nil)

	opts := t.defaultOpts
	// Le watcher détecte les matchs mais le RunDelta va re-fetch l'historique API
	// pour récupérer les stats complètes. On limite au nombre de matchs détectés + marge.
	if len(matchIDs) > 0 && opts.MaxMatches == 0 {
		opts.MaxMatches = len(matchIDs) + 5
	}

	result, err := engine.RunDelta(ctx, opts)
	if err != nil {
		slog.ErrorContext(ctx, "trigger: sync échoué",
			"gamertag", gamertag,
			"err", err,
		)
		return fmt.Errorf("trigger: sync %s: %w", gamertag, err)
	}

	slog.InfoContext(ctx, "trigger: sync terminé",
		"gamertag", gamertag,
		"matches_inserted", result.MatchesInserted,
		"participants_done", result.ParticipantsDone,
		"medals_inserted", result.MedalsInserted,
	)

	return nil
}
