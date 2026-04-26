package notifications

import "context"

// Emitter est l'interface réduite reçue par les hooks (sync engine, media handler...).
// Permet de déclencher une notification sans connaître l'impl ni avoir accès
// au reste de l'API (List/MarkRead/...).
type Emitter interface {
	// Emit insère une notification per-player après vérification de la pref
	// de catégorie. Synchrone, retourne nil si la catégorie est désactivée
	// (Emit silencieux, pas une erreur).
	Emit(ctx context.Context, in EmitInput) error
}

// NoopEmitter est une implémentation vide utile pour :
//   - les tests (engine sans dépendance notifications)
//   - le mode où les notifs sont globalement désactivées
//   - le bootstrap initial avant DI complète
type NoopEmitter struct{}

// Emit ne fait rien et renvoie nil.
func (NoopEmitter) Emit(_ context.Context, _ EmitInput) error { return nil }

// Compile-time check.
var _ Emitter = NoopEmitter{}
