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
	recovered, err := Recover(root)
	if err != nil {
		slog.ErrorContext(ctx, "resources: reprise d'une bascule interrompue échouée", "err", err)
	}
	out.Recovered = recovered
	if recovered {
		slog.WarnContext(ctx, "resources: bascule interrompue défaite, ancien paquet rétabli")
	}
	out.Verdict = Check(root, expected)
	if out.Verdict.State != StateMatch && Staged(root, expected) {
		if err := Activate(root, expected); err != nil {
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
