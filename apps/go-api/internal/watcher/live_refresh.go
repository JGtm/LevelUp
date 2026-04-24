// Package watcher — live_refresh.go : rafraîchissement live Battle Pass / Challenges.
//
// Pendant la présence active d'un joueur, un ticker toutes les 5 minutes
// re-fetche et persiste les données Battle Pass et Challenges.
package watcher

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"levelup/go-api/internal/assets"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/halo"
	"levelup/go-api/internal/port"
)

// LiveRefreshTrigger est l'interface de rafraîchissement live.
// Appelé par PlayerWatcher lors des changements de présence.
type LiveRefreshTrigger interface {
	OnPresenceActive(ctx context.Context)
	OnPresenceInactive(ctx context.Context)
}

// liveRefreshInterval est la fréquence de rafraîchissement pendant la présence.
const liveRefreshInterval = 5 * time.Minute

// PlayerLiveRefresher implémente LiveRefreshTrigger pour un joueur.
// Démarre un ticker au moment de la présence active et arrête à la déconnexion.
type PlayerLiveRefresher struct {
	gamertag string
	xuid     string
	interval time.Duration
	provider *halo.HaloProvider
	sink     *duckdb.PersistSink
	notifier port.SessionNotifier // nil si non configuré

	notifierWarnOnce sync.Once // log Warn une seule fois si notifier nil
	cancelMu         sync.Mutex
	cancel           context.CancelFunc
}

// NewPlayerLiveRefresher crée un refresher pour un joueur.
// sink est le PersistSink fire-and-forget pour persister les résultats.
// resolver, si non nil, délègue le cache/fetch au resolver unifié.
func NewPlayerLiveRefresher(gamertag, xuid string, sink *duckdb.PersistSink, resolver assets.Resolver) *PlayerLiveRefresher {
	provider := halo.DefaultHaloProvider
	if resolver != nil {
		provider = provider.WithAssetResolver(resolver)
	}
	return &PlayerLiveRefresher{
		gamertag: gamertag,
		xuid:     xuid,
		interval: liveRefreshInterval,
		provider: provider,
		sink:     sink,
	}
}

// WithSessionNotifier configure le notifier de présence (implémente port.SessionNotifier).
// Retourne le refresher pour permettre le chaînage.
func (r *PlayerLiveRefresher) WithSessionNotifier(n port.SessionNotifier) *PlayerLiveRefresher {
	r.notifier = n
	return r
}

// OnPresenceActive démarre le ticker de rafraîchissement.
// Sans effet si le ticker tourne déjà.
func (r *PlayerLiveRefresher) OnPresenceActive(ctx context.Context) {
	// Notifier avant le ticker pour que le TTL réduit soit effectif dès le démarrage.
	if r.notifier != nil {
		r.notifier.SetSessionActive(true)
	} else {
		r.notifierWarnOnce.Do(func() {
			slog.WarnContext(ctx, "live_refresh: aucun SessionNotifier configuré — TTL dynamique inactif",
				"gamertag", r.gamertag)
		})
	}

	r.cancelMu.Lock()
	defer r.cancelMu.Unlock()

	if r.cancel != nil {
		return // déjà en cours
	}

	tickCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel

	go r.runTicker(tickCtx)

	slog.InfoContext(ctx, "live_refresh: ticker démarré",
		"gamertag", r.gamertag,
		"interval", r.interval,
		"notifier_configured", r.notifier != nil,
	)
}

// OnPresenceInactive arrête le ticker de rafraîchissement.
func (r *PlayerLiveRefresher) OnPresenceInactive(ctx context.Context) {
	r.cancelMu.Lock()
	defer r.cancelMu.Unlock()

	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
		slog.InfoContext(ctx, "live_refresh: ticker arrêté", "gamertag", r.gamertag)
	}

	// Notifier après l'arrêt du ticker pour garantir l'ordre (plus de refresh en cours).
	if r.notifier != nil {
		r.notifier.SetSessionActive(false)
	}
}

// runTicker boucle jusqu'à annulation du contexte.
func (r *PlayerLiveRefresher) runTicker(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.refresh(ctx)
		}
	}
}

// refresh re-fetche Battle Pass et Challenges si des tokens valides sont disponibles.
// Utilise halo.GetCachedPlayerTokens — tokens peuplés par le dernier accès via l'UI web.
// Si aucun token n'est disponible, la fonction est un no-op.
func (r *PlayerLiveRefresher) refresh(ctx context.Context) {
	tokens := halo.GetCachedPlayerTokens(r.xuid)
	if tokens == nil {
		slog.DebugContext(ctx, "live_refresh: tokens non disponibles, skip",
			"gamertag", r.gamertag)
		return
	}

	ctx = ctxkeys.WithHaloAuth(ctx, tokens, r.xuid)

	bpResp, bpRaw := r.provider.GetBattlePassWithRaw(ctx)
	if bpResp.Available && bpResp.RewardTrack != nil && len(bpRaw) > 0 {
		r.sink.PersistBattlePass(*bpResp.RewardTrack, bpRaw)
	}

	cResp, cRaw := r.provider.GetChallengesWithRaw(ctx)
	if cResp.Available && len(cRaw) > 0 {
		r.sink.PersistChallenges(cRaw)
	}

	slog.InfoContext(ctx, "live_refresh: données rafraîchies",
		"gamertag", r.gamertag,
		"bp_available", bpResp.Available,
		"challenges_available", cResp.Available,
	)
}
