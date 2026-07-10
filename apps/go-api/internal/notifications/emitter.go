package notifications

import (
	"context"
	"time"
)

// Emitter est l'interface réduite reçue par les hooks (sync engine, media handler...).
// Permet de déclencher une notification sans connaître l'impl ni avoir accès
// au reste de l'API (List/MarkRead/...).
type Emitter interface {
	// Emit insère une notification per-player après vérification de la pref
	// de catégorie. Synchrone, retourne nil si la catégorie est désactivée
	// (Emit silencieux, pas une erreur).
	Emit(ctx context.Context, in EmitInput) error

	// EmitCoalesced insère une notification en la COALESCANT avec une notif
	// candidate récente non lue de la même catégorie/acteur si elle existe dans
	// `window` : au lieu d'une nouvelle ligne, la candidate est réémise
	// (même ID, created_at rafraîchi, `count` sommé) — évite les bursts de
	// notifs quasi identiques (5 clips du même acteur en 5 min → 1 notif count=5,
	// N échecs de sync consécutifs → 1 notif count=N). Sans candidate, fallback
	// sur une émission normale. window <= 0 → équivaut à Emit.
	EmitCoalesced(ctx context.Context, in EmitInput, window time.Duration) error
}

// NoopEmitter est une implémentation vide utile pour :
//   - les tests (engine sans dépendance notifications)
//   - le mode où les notifs sont globalement désactivées
//   - le bootstrap initial avant DI complète
type NoopEmitter struct{}

// Emit ne fait rien et renvoie nil.
func (NoopEmitter) Emit(_ context.Context, _ EmitInput) error { return nil }

// EmitCoalesced ne fait rien et renvoie nil.
func (NoopEmitter) EmitCoalesced(_ context.Context, _ EmitInput, _ time.Duration) error {
	return nil
}

// Compile-time check.
var _ Emitter = NoopEmitter{}
