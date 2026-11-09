// Package native relaie les notifications de la cloche vers la coquille de bureau, qui les montre comme
// notifications Windows (ADR 0045, lot L21). Il branche le port notifications.ExternalForwarder au point
// d'émission in-app, comme le relais Discord (package external), et ne sait rien de Windows : il rend
// un [Notice] (titre et corps déjà composés dans la langue du joueur, type) à un [Sink] que le boot du
// profil desktop installe, et qui écrit l'événement stdout `notification`.
//
// Opt-in : réglage `native_notifications` (défaut NON), relu à chaque notification ; tant que le boot n'a
// pas installé de sink (serveur hébergé, mode sans coquille), rien n'est relayé.
package native

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/platform/settings"
)

// Compteurs expvar du relais.
const (
	// MetricRelayed : notifications remises au sink (donc écrites à la coquille).
	MetricRelayed = "notifications_native_relayed_total"
	// MetricSkipped : notifications d'une catégorie relayée que le réglage coupé a retenues.
	MetricSkipped = "notifications_native_skipped_total"
	// MetricForeign : notifications d'un joueur qui n'est pas le joueur local (un profil suivi, ami ou autre),
	// jamais relayées : la notification Windows s'adresse à l'utilisateur de la machine.
	MetricForeign = "notifications_native_foreign_total"
)

// Notice est la notification Windows à montrer : texte composé dans la langue du joueur.
type Notice struct {
	Title string
	Body  string
	Kind  Kind
}

// Sink reçoit une notice à montrer. Il ne bloque pas et ne renvoie pas d'erreur : un événement perdu est
// journalisé par l'écrivain d'événements.
type Sink func(Notice)

// SettingsReader : la lecture des réglages vivants (satisfaite par *settings.Store).
type SettingsReader interface {
	Load() (*settings.AppSettings, error)
}

// OwnerFilter dit si le joueur de ce xuid est un joueur LOCAL : un compte avec lequel l'utilisateur de la
// machine s'est connecté (il porte un jeton de rafraîchissement). Les autres profils suivis (un ami, un
// adversaire) n'ont pas de jeton : leurs changements de palier ne sont pas ceux de l'utilisateur.
type OwnerFilter func(xuid string) bool

// Relay implémente notifications.ExternalForwarder.
type Relay struct {
	mu       sync.RWMutex
	sink     Sink
	settings SettingsReader
	owner    OwnerFilter
}

var defaultRelay = &Relay{}

// Default rend le relais du processus : les services de notification sont construits par joueur, loin
// du boot, et se branchent tous sur ce même relais.
func Default() *Relay { return defaultRelay }

// Install branche le relais sur son sink et sur les réglages. Un second appel remplace le premier.
func (r *Relay) Install(sink Sink, store SettingsReader) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sink, r.settings = sink, store
}

// Uninstall débranche le relais (tests).
func (r *Relay) Uninstall() {
	r.Install(nil, nil)
	r.InstallOwnerFilter(nil)
}

// InstallOwnerFilter branche le filtre du joueur local utilisé par [Relay.ForPlayer]. Sans filtre, aucun
// joueur n'est local : rien n'est relayé (défaut sûr).
func (r *Relay) InstallOwnerFilter(f OwnerFilter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.owner = f
}

// ForPlayer rend le relais d'UN joueur : le service de notification de chaque joueur suivi s'y branche, et
// seules les notifications du joueur local sont relayées (voir [OwnerFilter]). Le filtre se consulte à chaque
// notification : une connexion faite après le démarrage prend effet sans redémarrage.
func (r *Relay) ForPlayer(xuid string) notifications.ExternalForwarder {
	return playerForwarder{relay: r, xuid: xuid}
}

// playerForwarder relaie les notifications d'un seul joueur, si ce joueur est local.
type playerForwarder struct {
	relay *Relay
	xuid  string
}

// Forward relaie la notification si son joueur est local.
func (p playerForwarder) Forward(ctx context.Context, n *notifications.Notification) {
	if p.relay == nil || n == nil {
		return
	}
	p.relay.mu.RLock()
	owner := p.relay.owner
	p.relay.mu.RUnlock()
	if owner == nil || !owner(p.xuid) {
		observability.IncCounter(MetricForeign)
		return
	}
	p.relay.Forward(ctx, n)
}

// Forward relaie la notification si le relais est branché, si sa catégorie est relayée et si le réglage
// `native_notifications` est actif. Best-effort strict : jamais bloquant, jamais paniquant, jamais une
// erreur pour le flux d'émission.
func (r *Relay) Forward(ctx context.Context, n *notifications.Notification) {
	if r == nil || n == nil {
		return
	}
	kind, ok := KindOf(n.Category)
	if !ok {
		return
	}
	r.mu.RLock()
	sink, store := r.sink, r.settings
	r.mu.RUnlock()
	if sink == nil || store == nil {
		return
	}
	defer func() {
		if rec := recover(); rec != nil {
			slog.ErrorContext(ctx, "notifications.native: panic du relais récupéré",
				"category", n.Category, "recover", fmt.Sprintf("%v", rec))
		}
	}()
	cfg, err := store.Load()
	if err != nil {
		slog.WarnContext(ctx, "notifications.native: réglages illisibles, notification non relayée",
			"category", n.Category, "err", err)
		return
	}
	if !cfg.NativeNotifications {
		observability.IncCounter(MetricSkipped)
		return
	}
	title, body := Compose(i18n.Parse(cfg.Lang), kind)
	observability.IncCounter(MetricRelayed)
	sink(Notice{Title: title, Body: body, Kind: kind})
}
