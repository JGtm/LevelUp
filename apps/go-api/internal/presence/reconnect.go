// Package presence — reconnect.go : politique de reconnexion WebSocket avec backoff exponentiel.
//
// Utilisé par le watcher pour reconnecter automatiquement le client RTA
// après une déconnexion (timeout, erreur réseau, token expiré).
package presence

import (
	"context"
	"log/slog"
	"math"
	"time"
)

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

		// Connexion (ou reconnexion)
		if err := r.connectFunc(ctx); err != nil {
			delay := r.backoffDelay(attempt)
			slog.WarnContext(ctx, "reconnect: échec connexion, retry après backoff",
				"attempt", attempt+1,
				"delay", delay,
				"err", err,
			)
			attempt++
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
				continue
			}
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

// backoffDelay calcule le délai pour la tentative n.
func (r *ReconnectManager) backoffDelay(attempt int) time.Duration {
	delay := float64(r.policy.InitialDelay) * math.Pow(r.policy.Multiplier, float64(attempt))
	d := time.Duration(delay)
	if d > r.policy.MaxDelay {
		d = r.policy.MaxDelay
	}
	return d
}
