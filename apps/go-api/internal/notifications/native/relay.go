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

// Relay implémente notifications.ExternalForwarder.
type Relay struct {
	mu       sync.RWMutex
	sink     Sink
	settings SettingsReader
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
func (r *Relay) Uninstall() { r.Install(nil, nil) }

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
