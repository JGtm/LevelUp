// Package presence — reconnect.go : politique de reconnexion WebSocket avec backoff exponentiel.
//
// Utilisé par le watcher pour reconnecter automatiquement le client RTA
// après une déconnexion (timeout, erreur réseau, token expiré).
package presence

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"time"
)

// authRefreshRetryDelay est le délai appliqué avant reconnexion quand le refresh
// XSTS échoue ou qu'aucun callback OnAuthExpired n'est fourni.
const authRefreshRetryDelay = 30 * time.Second

// ReconnectPolicy définit les paramètres du backoff exponentiel.
type ReconnectPolicy struct {
	InitialDelay time.Duration // délai initial (défaut 1s)
	MaxDelay     time.Duration // délai max (défaut 5min)
	Multiplier   float64       // facteur multiplicatif (défaut 2.0)
}

// DefaultReconnectPolicy retourne la politique de reconnexion par défaut.
func DefaultReconnectPolicy() ReconnectPolicy {
	return ReconnectPolicy{
		InitialDelay: 1 * time.Second,
		MaxDelay:     5 * time.Minute,
		Multiplier:   2.0,
	}
}

// ReconnectManager gère la reconnexion automatique du client RTA.
type ReconnectManager struct {
	client      *RTAClient
	policy      ReconnectPolicy
	connectFunc func(ctx context.Context) error // Fonction de connexion + re-subscribe

	// OnAuthExpired est appelé quand tous les subscribes ont été refusés (status=3).
	// Doit acquérir un token XSTS frais et appeler client.UpdateAuth.
	// Si nil ou si l'appel échoue, un délai de 30s est appliqué avant reconnexion.
	OnAuthExpired func(ctx context.Context) error

	// waitFn injecte une temporisation annulable. Utilisé en test pour éviter les vrais sleeps.
	// Si nil, utilise time.After. Retourne false si ctx est annulé avant la fin du délai.
	waitFn func(ctx context.Context, d time.Duration) bool
}

// NewReconnectManager crée un gestionnaire de reconnexion.
// connectFunc est appelé à chaque tentative : il doit connecter ET re-souscrire.
func NewReconnectManager(client *RTAClient, policy ReconnectPolicy, connectFunc func(ctx context.Context) error) *ReconnectManager {
	return &ReconnectManager{
		client:      client,
		policy:      policy,
		connectFunc: connectFunc,
	}
}

// RunWithReconnect exécute ReadLoop en boucle avec reconnexion automatique.
// Bloquant — à lancer dans une goroutine. S'arrête quand ctx est annulé.
func (r *ReconnectManager) RunWithReconnect(ctx context.Context) {
	attempt := 0

	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "reconnect: arrêt demandé")
			return
		default:
		}

		// Si le client a signalé un token XSTS expiré (status=3), tenter un refresh
		// avant de reconnecter pour éviter la boucle infinie reconnect→status=3.
		if err := r.refreshAuthIfNeeded(ctx); err != nil {
			// Refresh échoué ou absent — attendre avant de réessayer.
			if !r.wait(ctx, authRefreshRetryDelay) {
				return
			}
			continue
		}

		// Connexion (ou reconnexion)
		if err := r.connectFunc(ctx); err != nil {
			delay := r.backoffDelay(attempt)
			slog.WarnContext(ctx, "reconnect: échec connexion, retry après backoff",
				"attempt", attempt+1,
				"delay", delay,
				"err", err,
			)
			attempt++
			if !r.wait(ctx, delay) {
				return
			}
			continue
		}

		// Connexion réussie → reset le compteur
		attempt = 0
		slog.InfoContext(ctx, "reconnect: connecté, démarrage read loop")

		// ReadLoop bloque jusqu'à déconnexion
		if err := r.client.ReadLoop(ctx); err != nil {
			slog.WarnContext(ctx, "reconnect: read loop terminé", "err", err)
		}

		// ReadLoop terminé → reconnecter
		slog.InfoContext(ctx, "reconnect: préparation reconnexion...")
	}
}

// refreshAuthIfNeeded vérifie si le client a signalé un token expiré (status=3)
// et, si oui, exécute OnAuthExpired pour obtenir un token XSTS frais.
// Retourne nil si aucune action n'était nécessaire ou si le refresh a réussi.
// Retourne une erreur si le callback est absent ou a échoué — dans ce cas
// la boucle appellera wait(authRefreshRetryDelay) avant de retenter.
func (r *ReconnectManager) refreshAuthIfNeeded(ctx context.Context) error {
	if !r.client.IsAuthExpired() {
		return nil
	}
	slog.WarnContext(ctx, "reconnect: token XSTS expiré (status=3), refresh on-demand...")
	if r.OnAuthExpired == nil {
		slog.WarnContext(ctx, "reconnect: aucun callback OnAuthExpired fourni, attente avant retry",
			"delay", authRefreshRetryDelay)
		return errors.New("no OnAuthExpired callback")
	}
	if err := r.OnAuthExpired(ctx); err != nil {
		slog.ErrorContext(ctx, "reconnect: refresh XSTS échoué, attente avant retry",
			"err", err, "delay", authRefreshRetryDelay)
		return err
	}
	r.client.ResetAuthExpired()
	slog.InfoContext(ctx, "reconnect: refresh XSTS réussi, reprise connexion")
	return nil
}

// wait attend la durée d ou l'annulation du contexte.
// Retourne false si le contexte a été annulé (arrêt demandé).
func (r *ReconnectManager) wait(ctx context.Context, d time.Duration) bool {
	if r.waitFn != nil {
		return r.waitFn(ctx, d)
	}
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

// backoffDelay calcule le délai pour la tentative n.
func (r *ReconnectManager) backoffDelay(attempt int) time.Duration {
	delay := float64(r.policy.InitialDelay) * math.Pow(r.policy.Multiplier, float64(attempt))
	d := time.Duration(delay)
	if d > r.policy.MaxDelay {
		d = r.policy.MaxDelay
	}
	return d
}
