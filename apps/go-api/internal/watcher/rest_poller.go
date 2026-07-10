// Package watcher — rest_poller.go : poller REST de la présence Xbox d'un user.
//
// Pourquoi REST polling plutôt que RTA WebSocket :
//   - RTA `/titles/<TID>` ne push pas après le snapshot (Xbox limitation pour
//     s'auto-observer son propre compte). Confirmé 22 min sans push 2026-05-25.
//   - RTA `/richpresence` → status=3 avec notre RelyingParty XSTS standard.
//   - REST poll est la stratégie utilisée par tous les projets équivalents :
//     OpenXbox/xbox-webapi-python, misiektoja/xbox_monitor, MrCoolAndroid/
//     Xbox-Rich-Presence-Discord, epicmanmoo/xbox-discord-rich-presence.
//
// Intervalle fixe 10s : pas de quota free vs payant côté Microsoft (on hit
// userpresence.xboxlive.com directement avec notre XSTS personnel). 4 joueurs
// × 1 req/10s = 24 req/min total, soit ~0.8 % de la limite documentée Xbox
// (~50 req/s/token). Le choix de 10s donne une bonne réactivité UI sur
// transition Active↔Inactive, tout en restant un comportement de bon citoyen.
//
// Note : la latence de détection "fin de match" reste dominée par l'API Halo
// (GetMatchHistory met ~30-60s à exposer un match terminé) et la grâce
// post-extinction (90s) — un poll plus rapide n'aide pas sur ce point.
//
// Refresh XSTS : si Xbox retourne 401, le poller invoque `onAuthExpired` qui
// doit refresh le token + appeler `client.UpdateAuth(newHeader)` + retourner
// nil. Le poller retente immédiatement après.
package watcher

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/presence"
)

const (
	// restPollInterval : fréquence de poll en fonctionnement nominal.
	restPollInterval = 10 * time.Second

	// restPollBackoffRateLimit : pause après un 429 (rate-limit Xbox). 5 min
	// est un délai conservateur pour ne pas aggraver.
	restPollBackoffRateLimit = 5 * time.Minute

	// restPollBackoffTransient : pause après un 5xx Xbox. Court pour reprendre
	// vite quand Xbox redémarre.
	restPollBackoffTransient = 30 * time.Second

	// restPollBackoffNetwork : pause après une erreur réseau (timeout, DNS, etc.).
	restPollBackoffNetwork = 30 * time.Second
)

// PresenceFetcher est l'interface minimale dont dépend RESTPoller. Permet
// d'injecter un mock dans les tests (sans réseau réel).
type PresenceFetcher interface {
	GetPresence(ctx context.Context, xuid string) (presence.PresenceEvent, error)
	UpdateAuth(authHeader string)
}

// AuthRefresher est appelé quand le poller reçoit un 401 Xbox. Doit refresh
// le XSTS du user concerné et retourner le nouveau header XBL3.0. Le poller
// appelle ensuite client.UpdateAuth(newHeader). En cas d'erreur (refresh
// impossible), le poller backoff comme sur une erreur réseau.
type AuthRefresher func(ctx context.Context) (string, error)

// RESTPoller boucle sur PresenceFetcher.GetPresence à un intervalle fixe et
// dispatche chaque résultat via le handler fourni (typiquement le même que
// celui du RTA pour réutiliser la chaîne FSM → MatchPoller → Coordinator).
type RESTPoller struct {
	xuid         string
	gamertag     string
	client       PresenceFetcher
	handler      presence.EventHandler
	refresh      AuthRefresher // nil = pas de refresh, juste backoff sur 401
	interval     time.Duration // override possible pour tests
	backoffRate  time.Duration
	backoffTrans time.Duration
	backoffNet   time.Duration
}

// NewRESTPoller construit un poller pour un (xuid, gamertag). L'interval
// utilise la valeur par défaut prod ; pour les tests utiliser WithInterval.
func NewRESTPoller(xuid, gamertag string, client PresenceFetcher, handler presence.EventHandler) *RESTPoller {
	return &RESTPoller{
		xuid:         xuid,
		gamertag:     gamertag,
		client:       client,
		handler:      handler,
		interval:     restPollInterval,
		backoffRate:  restPollBackoffRateLimit,
		backoffTrans: restPollBackoffTransient,
		backoffNet:   restPollBackoffNetwork,
	}
}

// WithAuthRefresher injecte un callback de refresh XSTS appelé sur 401.
func (p *RESTPoller) WithAuthRefresher(r AuthRefresher) *RESTPoller {
	p.refresh = r
	return p
}

// WithInterval override l'interval (pour tests).
func (p *RESTPoller) WithInterval(d time.Duration) *RESTPoller {
	p.interval = d
	return p
}

// WithBackoffs override les backoffs sur erreur (pour tests).
func (p *RESTPoller) WithBackoffs(rateLimit, transient, network time.Duration) *RESTPoller {
	p.backoffRate = rateLimit
	p.backoffTrans = transient
	p.backoffNet = network
	return p
}

// Run lance la boucle de polling. Bloquant jusqu'à ctx.Done().
//
// Comportement :
//  1. Tick immédiat au démarrage (snapshot initial).
//  2. Attend `interval` avant le prochain tick.
//  3. Sur erreur : log + backoff selon le type d'erreur.
//  4. Sur 401 : appelle p.refresh si non-nil + retry immédiat.
func (p *RESTPoller) Run(ctx context.Context) {
	slog.InfoContext(ctx, "rest_poller: démarré",
		"xuid", p.xuid, "gamertag", p.gamertag, "interval", p.interval,
	)
	defer slog.InfoContext(ctx, "rest_poller: arrêté", "xuid", p.xuid, "gamertag", p.gamertag)

	// 1er tick immédiat pour avoir un état dès le démarrage.
	nextDelay := time.Duration(0)
	for {
		if nextDelay > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(nextDelay):
			}
		} else {
			// Yield rapide pour respecter ctx.Done() même au 1er tour.
			select {
			case <-ctx.Done():
				return
			default:
			}
		}

		nextDelay = p.tickOnce(ctx)
	}
}

// tickOnce effectue un appel et retourne le délai avant le prochain.
// Extraite pour testabilité (peut être appelée seule dans les tests).
func (p *RESTPoller) tickOnce(ctx context.Context) time.Duration {
	event, err := p.client.GetPresence(ctx, p.xuid)
	if err != nil {
		return p.handleError(ctx, err)
	}

	// Dispatch vers le handler (même type que RTA, réutilise la chaîne FSM).
	p.handler(event)

	return p.interval
}

// handleError choisit le bon backoff selon le type d'erreur + tente un
// refresh XSTS si 401 + refresher disponible.
func (p *RESTPoller) handleError(ctx context.Context, err error) time.Duration {
	var httpErr *presence.HTTPError
	if errors.As(err, &httpErr) {
		switch {
		case httpErr.IsAuthExpired():
			return p.handleAuthExpired(ctx)
		case httpErr.IsRateLimited():
			slog.WarnContext(ctx, "rest_poller: rate-limited par Xbox",
				"xuid", p.xuid, "gamertag", p.gamertag, "backoff", p.backoffRate)
			return p.backoffRate
		case httpErr.IsTransient():
			// Anti-flood B6.4 : un incident Xbox/infra frappe tous les pollers en
			// boucle → clé globale par cause, 1 log / fenêtre, compteur expvar exact.
			if allow, since := observability.AllowThrottledLog(
				"log_throttle_rest_poller_transient", observability.NetworkFloodWindow); allow {
				slog.WarnContext(ctx, "rest_poller: erreur transiente Xbox",
					"xuid", p.xuid, "gamertag", p.gamertag,
					"status", httpErr.StatusCode, "backoff", p.backoffTrans,
					"throttled_since_last", since)
			}
			return p.backoffTrans
		default:
			slog.ErrorContext(ctx, "rest_poller: HTTP non géré",
				"xuid", p.xuid, "gamertag", p.gamertag,
				"status", httpErr.StatusCode, "err", err)
			return p.backoffNet
		}
	}

	// Erreur non-HTTP (réseau, timeout, parse). Anti-flood B6.4 : pendant une
	// panne DNS/réseau, chaque poller échoue à chaque cycle → clé globale.
	if allow, since := observability.AllowThrottledLog(
		"log_throttle_rest_poller_network", observability.NetworkFloodWindow); allow {
		slog.WarnContext(ctx, "rest_poller: erreur réseau/parse",
			"xuid", p.xuid, "gamertag", p.gamertag,
			"err", err, "backoff", p.backoffNet, "throttled_since_last", since)
	}
	return p.backoffNet
}

// handleAuthExpired tente un refresh XSTS. Si OK, retry immédiat (delay=0).
// Si KO, backoff réseau pour éviter le hammering sur Xbox auth.
func (p *RESTPoller) handleAuthExpired(ctx context.Context) time.Duration {
	if p.refresh == nil {
		slog.ErrorContext(ctx, "rest_poller: XSTS expiré mais pas de refresher câblé",
			"xuid", p.xuid, "gamertag", p.gamertag, "backoff", p.backoffNet)
		return p.backoffNet
	}

	newHeader, err := p.refresh(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "rest_poller: refresh XSTS échoué",
			"xuid", p.xuid, "gamertag", p.gamertag, "err", err, "backoff", p.backoffNet)
		return p.backoffNet
	}

	p.client.UpdateAuth(newHeader)
	slog.InfoContext(ctx, "rest_poller: XSTS refresh OK, retry immédiat",
		"xuid", p.xuid, "gamertag", p.gamertag)
	return 0 // retry immédiat avec le nouveau header
}
