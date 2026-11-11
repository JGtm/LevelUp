package scheduler

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/auth/pool"
)

// RescanPool relit le magasin des jetons et ajoute au pool les comptes qu'il ne connaît pas
// encore, ou rafraîchit leur jeton. Rend le nombre de sources entrées dans le pool. Sans pool
// (tests), ne fait rien.
func (s *AutoSyncScheduler) RescanPool(ctx context.Context) int {
	if s.pool == nil {
		return 0
	}
	store := auth.NewMultiUserTokenStore(s.cfg.WatcherTokensDir())
	discovery := pool.NewDiscoveryWithStore(s.cfg, s.cfg.PathResolver(), title.DefaultSlug, store)
	sources, err := discovery.Scan(ctx)
	if err != nil {
		slog.WarnContext(ctx, "auto_sync: re-scan du magasin de jetons échoué", "err", err)
		return 0
	}
	added := 0
	for _, src := range sources {
		if err := s.pool.AddOrUpdateSource(ctx, src); err != nil {
			slog.WarnContext(ctx, "auto_sync: source non entrée dans le pool", "gamertag", src.Gamertag, "err", err)
			continue
		}
		added++
	}
	slog.InfoContext(ctx, "auto_sync: re-scan du magasin de jetons terminé",
		"sources", len(sources), "entrées", added, "pool_size", s.pool.Size())
	return added
}

// OnCredentialsWritten : un joueur vient de se connecter. Son jeton entre dans le pool tout
// de suite et un cycle est joué sans attendre le prochain tic, pour que sa première
// synchronisation parte dans la minute (et non après le re-scan périodique). Bloquant :
// l'appelant choisit son goroutine et son contexte.
func (s *AutoSyncScheduler) OnCredentialsWritten(ctx context.Context) {
	if s.RescanPool(ctx) == 0 {
		return
	}
	s.RunOnceTrigger(ctx, "login")
}
