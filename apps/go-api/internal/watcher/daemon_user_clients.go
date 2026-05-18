// Package watcher — daemon_user_clients.go : multi-user RTA (stratégie A, PR 2.5c).
//
// Extrait de daemon.go pour respecter la limite 500L/fichier (CLAUDE.md règle 14).
// Tout ce qui touche aux userClient (1 RTAClient + 1 ReconnectManager par user
// SSO Xbox) vit ici. Les champs `userClients`, `userClientsMu` et
// `perUserAuthRefresh` du struct Daemon restent dans daemon.go (couplés au type).
//
// Cycle de vie :
//  1. main.go.startWatcherDaemon : daemon.Start(...) puis pour chaque
//     UserTokens du MultiUserTokenStore → daemon.AddUserClient(tokens).
//  2. À chaque login Xbox SSO réussi : XboxSSOLinkStrategy.notifyWatcher
//     appelle daemon.AddUserClient via WatcherDaemon interface.
//  3. Sur status=3 (XSTS expiré), ReconnectManager invoque le callback
//     perUserAuthRefresh → auth.RefreshUserXSTS reconstruit le XSTS via le
//     cache MSAL persisté + UpdateAuth sur le rtaClient.
//  4. daemon.Stop() ferme tous les userClient + attend les goroutines.
package watcher

import (
	"context"
	"fmt"
	"log/slog"

	auth_platform "levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/presence"
)

// userClient encapsule un RTAClient dédié à un utilisateur SSO Xbox (PR 2.5c).
// Chaque user qui se logge via SSO obtient sa propre connexion RTA + reconnect
// manager, indépendamment du graphe social Xbox (pas besoin d'être ami du tracker).
type userClient struct {
	xuid         string
	gamertag     string
	rtaClient    *presence.RTAClient
	reconnectMgr *presence.ReconnectManager
	cancel       context.CancelFunc // annule le RunWithReconnect dédié
}

// PerUserAuthRefresher est appelé quand le ReconnectManager d'un user reçoit
// status=3 (XSTS expiré). Doit retourner un nouvel auth header XBL3.0 ou une
// erreur si refresh impossible (user doit re-login Xbox SSO). Typiquement
// implémenté par auth.RefreshUserXSTS.
type PerUserAuthRefresher func(ctx context.Context, xuid string) (string, error)

// WithPerUserAuthRefresh injecte un callback de refresh XSTS pour les userClients
// (PR 2.5c). Si fourni, chaque userClient.reconnectMgr l'appelle avec son XUID
// quand un subscribe est refusé avec status=3.
func (d *Daemon) WithPerUserAuthRefresh(refresher PerUserAuthRefresher) *Daemon {
	d.perUserAuthRefresh = refresher
	return d
}

// AddUserClient ajoute un user SSO Xbox avec sa propre connexion RTA dédiée (PR 2.5c).
// Le user subscribe son propre XUID avec son propre auth header XBL3.0 — pas
// besoin que le tracker historique soit ami Xbox de cet user. Résiste au social
// graph Xbox.
//
// No-op si le user (XUID) est déjà présent dans le map. Erreur si auth header vide
// ou XUID/Gamertag vides. Lance une goroutine RunWithReconnect dédiée pour la durée
// de vie du daemon (rootCtx capturé dans Start).
//
// Note : le daemon doit avoir été démarré (Start appelé) avant AddUserClient,
// sinon rootCtx est nil. En pratique, main.go appelle Start puis AddUserClient
// pour chaque user du MultiUserTokenStore.
func (d *Daemon) AddUserClient(ctx context.Context, userTokens *auth_platform.UserTokens) error {
	if userTokens == nil || userTokens.XUID == "" || userTokens.Gamertag == "" {
		return fmt.Errorf("watcher_daemon: AddUserClient requiert xuid+gamertag non vides")
	}
	authHeader := userTokens.AuthHeader()
	if authHeader == "" {
		return fmt.Errorf("watcher_daemon: AddUserClient requiert XSTS+UserHash non vides (xuid=%s)", userTokens.XUID)
	}
	if d.rootCtx == nil {
		return fmt.Errorf("watcher_daemon: AddUserClient appelé avant Start (xuid=%s)", userTokens.XUID)
	}

	d.userClientsMu.Lock()
	if _, exists := d.userClients[userTokens.XUID]; exists {
		d.userClientsMu.Unlock()
		slog.DebugContext(ctx, "watcher_daemon: AddUserClient no-op, déjà présent",
			"xuid", userTokens.XUID, "gamertag", userTokens.Gamertag)
		return nil
	}

	// Créer le PlayerWatcher si pas déjà présent (sinon réutiliser).
	d.playersMu.Lock()
	pw, ok := d.players[userTokens.Gamertag]
	if !ok {
		pw = NewPlayerWatcher(userTokens.Gamertag, userTokens.XUID, nil, &queueSyncTrigger{
			queue:    d.queue,
			gamertag: userTokens.Gamertag,
			xuid:     userTokens.XUID,
		})
		if d.cfg.LiveRefreshFactory != nil {
			pw = pw.WithLiveRefresh(d.cfg.LiveRefreshFactory(userTokens.Gamertag, userTokens.XUID))
		}
		d.players[userTokens.Gamertag] = pw
	}
	d.playersMu.Unlock()

	// Créer le RTAClient + ReconnectManager dédiés.
	rtaClient := presence.NewRTAClient(authHeader)
	connectFunc := d.makeUserConnectFunc(rtaClient, pw, userTokens.XUID, userTokens.Gamertag)
	reconnectMgr := presence.NewReconnectManager(rtaClient, presence.DefaultReconnectPolicy(), connectFunc)

	// Refresh on-demand si callback fourni.
	if d.perUserAuthRefresh != nil {
		xuid := userTokens.XUID // capture
		reconnectMgr.OnAuthExpired = func(refreshCtx context.Context) error {
			newHeader, err := d.perUserAuthRefresh(refreshCtx, xuid)
			if err != nil {
				return err
			}
			rtaClient.UpdateAuth(newHeader)
			return nil
		}
	}

	// Contexte dédié pour pouvoir arrêter ce userClient indépendamment.
	clientCtx, cancel := context.WithCancel(d.rootCtx)
	uc := &userClient{
		xuid:         userTokens.XUID,
		gamertag:     userTokens.Gamertag,
		rtaClient:    rtaClient,
		reconnectMgr: reconnectMgr,
		cancel:       cancel,
	}
	d.userClients[userTokens.XUID] = uc
	d.userClientsMu.Unlock()

	slog.InfoContext(ctx, "watcher_daemon: userClient ajouté (RTA dédié)",
		"xuid", userTokens.XUID, "gamertag", userTokens.Gamertag)

	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		reconnectMgr.RunWithReconnect(clientCtx)
	}()
	return nil
}

// makeUserConnectFunc construit la closure connect+subscribe pour un userClient.
// Subscribe uniquement le XUID de cet user (pas tous les players du daemon).
func (d *Daemon) makeUserConnectFunc(rtaClient *presence.RTAClient, pw *PlayerWatcher, xuid, gamertag string) func(context.Context) error {
	return func(connectCtx context.Context) error {
		if err := rtaClient.Connect(connectCtx); err != nil {
			return err
		}
		handler := d.makePresenceHandler(connectCtx, pw)
		var lastErr error
		for _, td := range d.titleReg.All() {
			if td.XboxTitleID == "" {
				continue
			}
			if err := rtaClient.Subscribe(connectCtx, xuid, td.XboxTitleID, handler); err != nil {
				slog.WarnContext(connectCtx, "watcher_daemon: échec subscribe userClient",
					"xuid", xuid, "gamertag", gamertag, "title", td.Name, "err", err)
				lastErr = err
			}
		}
		pw.SetSubscribeError(lastErr)
		return nil
	}
}
