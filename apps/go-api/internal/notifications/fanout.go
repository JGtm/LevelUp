package notifications

import "context"

// fanout relaie une notification à plusieurs [ExternalForwarder], dans l'ordre.
type fanout []ExternalForwarder

// Forward appelle chaque relais ; chacun garde son contrat best-effort (jamais bloquant, jamais
// paniquant), la boucle n'ajoute donc aucune garde.
func (f fanout) Forward(ctx context.Context, n *Notification) {
	for _, fw := range f {
		fw.Forward(ctx, n)
	}
}

// Fanout combine des relais externes en un seul, que le service appelle après chaque insertion. Les
// relais nil sont ignorés ; sans relais, le résultat est nil (le service n'appelle alors rien).
func Fanout(forwarders ...ExternalForwarder) ExternalForwarder {
	kept := make(fanout, 0, len(forwarders))
	for _, fw := range forwarders {
		if fw != nil {
			kept = append(kept, fw)
		}
	}
	switch len(kept) {
	case 0:
		return nil
	case 1:
		return kept[0]
	}
	return kept
}
