package resourcepack

import (
	"context"
	"log/slog"
)

// Outcome : ce que le démarrage a fait du paquet de ressources.
type Outcome struct {
	Verdict Verdict
	// Recovered : une bascule interrompue par un arrêt brutal a été défaite.
	Recovered bool
	// Activated : un paquet téléchargé pendant une exécution précédente est devenu le paquet actif.
	Activated bool
	// ActivateErr : la bascule d'un paquet prêt a échoué (l'ancien paquet reste actif).
	ActivateErr error
	// RecoverErr : la reprise d'une bascule interrompue n'a pas abouti ; rien n'est activé sur un état incertain.
	RecoverErr error
}

// Settle est l'étape de démarrage du paquet, avant que le serveur n'ouvre un seul fichier de
// ressources : défait une bascule interrompue, active le paquet prêt si c'est celui que le moteur
// attend, puis rend le verdict du contrôle d'empreinte. Elle ne décide de rien : refuser ou non
// de démarrer revient à l'appelant, selon le profil de déploiement.
func Settle(ctx context.Context, root, expected string) Outcome {
	var out Outcome
	if expected == "" {
		out.Verdict = Check(root, expected)
		return out
	}
	recovered, err := Recover(ctx, root)
	if err != nil {
		out.RecoverErr = err
		slog.ErrorContext(ctx, "resources: reprise d'une bascule interrompue échouée, rien n'est activé", "err", err)
	}
	switch recovered {
	case RecoveryUndone:
		out.Recovered = true
		slog.WarnContext(ctx, "resources: bascule interrompue défaite, ancien paquet rétabli")
	case RecoveryFinished:
		slog.InfoContext(ctx, "resources: bascule validée avant l'arrêt, ménage terminé")
	}
	out.Verdict = Check(root, expected)
	if out.RecoverErr == nil && out.Verdict.State != StateMatch && Staged(root, expected) {
		if err := Activate(ctx, root, expected); err != nil {
			out.ActivateErr = err
			slog.ErrorContext(ctx, "resources: activation du paquet prêt échouée, ancien paquet conservé",
				"err", err, "fingerprint", expected)
		} else {
			out.Activated = true
			slog.InfoContext(ctx, "resources: nouveau paquet activé", "fingerprint", expected,
				"ancien", out.Verdict.Active)
			out.Verdict = Check(root, expected)
		}
	}
	return out
}
