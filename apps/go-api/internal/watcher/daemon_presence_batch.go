// Package watcher — daemon_presence_batch.go : présence Xbox de tiers (amis).
//
// Le daemon détient le SEUL header XBL3.0 maintenu à jour du process (celui du
// tracker, rafraîchi par UpdateAuth à chaque cycle XSTS). Toute question de
// présence posée hors du poll des joueurs suivis — « combien d'amis sont en
// jeu ? » — passe donc par lui, plutôt que de re-fabriquer un client et une
// chaîne de refresh en parallèle.
//
// Fichier séparé de daemon.go, déjà au-delà du seuil de 500 lignes (dette gelée).
package watcher

import (
	"context"
	"errors"

	"levelup/go-api/internal/presence"
)

// ErrPresenceClientUnavailable : le daemon n'a pas (encore) de client REST —
// watcher jamais démarré. L'appelant dégrade (compteur d'amis à zéro), il ne
// s'agit pas d'une panne.
var ErrPresenceClientUnavailable = errors.New("watcher_daemon: client de présence indisponible")

// PresenceBatch interroge Xbox pour la présence de plusieurs users en un appel,
// avec l'authentification du tracker. Sert le compteur « amis en jeu » ; les
// joueurs suivis, eux, sont déjà pollés individuellement.
func (d *Daemon) PresenceBatch(ctx context.Context, xuids []string) ([]presence.PresenceEvent, error) {
	// Même verrou de lecture que GetStatus pour le champ trackerRestClient.
	d.playersMu.RLock()
	client := d.trackerRestClient
	d.playersMu.RUnlock()

	if client == nil {
		return nil, ErrPresenceClientUnavailable
	}
	return client.GetPresenceBatch(ctx, xuids)
}
