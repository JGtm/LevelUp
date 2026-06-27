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
	"levelup/go-api/internal/domain"
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
	gamertag       string
	xuid           string
	interval       time.Duration
	provider       *halo.HaloProvider
	sink           *duckdb.PersistSink
	notifier       port.SessionNotifier // nil si non configuré
	tokenRefresher func(ctx context.Context, xuid string) (*domain.HaloTokens, error)

	cancelMu sync.Mutex
	cancel   context.CancelFunc
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

// WithTokenRefresher configure la fonction de refresh de tokens Halo.
// Appelée quand GetCachedPlayerTokens retourne nil (cache process expiré après ~50 min).
// Permet au watcher de se réauthentifier sans dépendre d'une requête HTTP de l'UI.
func (r *PlayerLiveRefresher) WithTokenRefresher(fn func(ctx context.Context, xuid string) (*domain.HaloTokens, error)) *PlayerLiveRefresher {
	r.tokenRefresher = fn
	return r
}

// OnPresenceActive démarre le ticker de rafraîchissement.
// Sans effet si le ticker tourne déjà.
func (r *PlayerLiveRefresher) OnPresenceActive(ctx context.Context) {
	// Notifier (si configuré) avant le ticker pour que le TTL réduit soit effectif
	// dès le démarrage. notifier nil est un état NORMAL — le HomeService n'est créé
	// qu'à l'ouverture de la page Home, et SetSessionActive est de toute façon un
	// no-op aujourd'hui : pas de WARN. L'état est tracé dans le log "ticker démarré"
	// ci-dessous via notifier_configured.
	if r.notifier != nil {
		r.notifier.SetSessionActive(true)
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
// Tente d'abord le cache process ; si expiré, utilise tokenRefresher (MSAL / OAuth v2 via DB).
// Si aucun token n'est disponible, la fonction est un no-op.
func (r *PlayerLiveRefresher) refresh(ctx context.Context) {
	tokens := halo.GetCachedPlayerTokens(r.xuid)
	if tokens == nil && r.tokenRefresher != nil {
		refreshed, err := r.tokenRefresher(ctx, r.xuid)
		if err != nil {
			slog.DebugContext(ctx, "live_refresh: refresh tokens échoué, skip",
				"gamertag", r.gamertag, "err", err)
			return
		}
		tokens = refreshed
	}
	if tokens == nil {
		slog.DebugContext(ctx, "live_refresh: tokens non disponibles, skip",
			"gamertag", r.gamertag)
		return
	}

	ctx = ctxkeys.WithHaloAuth(ctx, tokens, r.xuid)

	bpResp, bpRaw := r.provider.GetBattlePassWithRaw(ctx)
	if bpResp.Available && bpResp.RewardTrack != nil && len(bpRaw) > 0 {
		// W6 : variantes SYNC — l'écriture s'exécute DANS la goroutine du ticker
		// (liée à ctx, annulée à OnPresenceInactive / shutdown) au lieu d'un
		// goroutine détaché en context.Background() qui pouvait écrire après
		// duckdb.CloseAll().
		if err := r.sink.PersistBattlePassSync(ctx, *bpResp.RewardTrack, bpRaw); err != nil {
			slog.WarnContext(ctx, "live_refresh: battlepass persist échoué",
				"gamertag", r.gamertag, "err", err)
		}
	}

	cResp, cRaw := r.provider.GetChallengesWithRaw(ctx)
	if cResp.Available && len(cRaw) > 0 {
		if err := r.sink.PersistChallengesSync(ctx, cRaw, cResp.Items); err != nil {
			slog.WarnContext(ctx, "live_refresh: challenges persist échoué",
				"gamertag", r.gamertag, "err", err)
		}
	}

	slog.InfoContext(ctx, "live_refresh: données rafraîchies",
		"gamertag", r.gamertag,
		"bp_available", bpResp.Available,
		"challenges_available", cResp.Available,
	)
}
