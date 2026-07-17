package notifications

import "context"

// ExternalForwarder est le port OPTIONNEL de relais externe d'une notification
// déjà émise in-app (canal Discord webhook, opt-in — cf. package
// internal/notifications/external).
//
// Contrat best-effort STRICT : l'implémentation ne DOIT jamais bloquer le flux
// d'émission ni faire échouer l'insertion in-app. Elle gère elle-même son
// asynchronisme (goroutine + recover), son timeout et son filtrage (catégories
// forwardées, flag d'activation). Forward ne retourne donc pas d'erreur : le
// relais externe est un effet de bord silencieux du point de vue de l'émission.
//
// Le port vit dans le package notifications (stdlib-only) pour rester découplé
// de toute impl HTTP ; l'adapter concret (Dispatcher) vit dans le sous-package
// external et n'est injecté que par le câblage boot (WithExternalForwarder).
type ExternalForwarder interface {
	// Forward reçoit une notification qui vient d'être persistée in-app. À charge
	// de l'impl de décider si elle relaie (catégorie forwardée + relais actif) et
	// de le faire sans jamais paniquer ni bloquer l'appelant.
	Forward(ctx context.Context, n *Notification)
}
