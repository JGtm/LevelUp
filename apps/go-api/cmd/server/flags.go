// flags.go — parseur UNIQUE des arguments du serveur.
//
// Les racines (données, ressources, journaux) se règlent par variables d'environnement lues
// dans internal/config ; les drapeaux ne portent que le mode de lancement. Tout nouveau
// drapeau s'ajoute ICI (jamais un os.Args lu à la main ailleurs).
//
// Le mode enfant de cuisson (replaychild.IsChild, -film-child) est intercepté par main AVANT
// ce parseur, sur la ligne de commande brute : il ne passe pas par ici.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"levelup/go-api/internal/config"
)

// Valeurs du drapeau -mode.
const (
	// serverModeDefault : serveur ordinaire (hébergé ou poste de développement).
	serverModeDefault = config.LaunchModeDefault
	// serverModeDesktop : serveur lancé par la coquille desktop.
	serverModeDesktop = config.LaunchModeDesktop
	// serverModeRefreshOnly : serveur de rafraîchissement seul.
	serverModeRefreshOnly = "refresh-only"
)

// flagsExitCode : code de sortie d'une ligne de commande refusée (convention du paquet flag).
const flagsExitCode = 2

// serverUsage : rappel affiché avec toute erreur de ligne de commande.
const serverUsage = "usage : server [-health-check] [-arret-sur-stdin] [-mode desktop|refresh-only]"

// errModeUnavailable : le mode est reconnu mais son code n'est pas encore livré (refresh-only).
var errModeUnavailable = errors.New("mode non disponible dans cette version")

// serverFlags : les arguments du serveur, validés.
type serverFlags struct {
	// HealthCheck : interroger /health du serveur local puis sortir (HEALTHCHECK Docker).
	HealthCheck bool
	// StopOnStdin : arrêt propre à la fermeture de stdin (lanceur parent qui meurt ou ferme).
	StopOnStdin bool
	// Mode : serverModeDefault, serverModeDesktop ou serverModeRefreshOnly.
	Mode string
}

// parseServerFlags lit args (os.Args[1:]) et valide les valeurs. Erreurs : drapeau inconnu,
// argument positionnel, mode inconnu, mode reconnu mais pas encore disponible
// (errModeUnavailable : refresh-only le reste jusqu'à sa livraison). Une demande d'aide rend
// flag.ErrHelp. Le parseur n'écrit rien : main affiche l'erreur.
func parseServerFlags(args []string) (serverFlags, error) {
	var f serverFlags
	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.BoolVar(&f.HealthCheck, "health-check", false, "interroge /health puis sort")
	fs.BoolVar(&f.StopOnStdin, "arret-sur-stdin", false, "arrêt propre à la fermeture de stdin")
	fs.StringVar(&f.Mode, "mode", serverModeDefault, "mode de lancement : desktop ou refresh-only")
	if err := fs.Parse(args); err != nil {
		return serverFlags{}, err
	}
	if fs.NArg() > 0 {
		return serverFlags{}, fmt.Errorf("argument inattendu %q", fs.Arg(0))
	}
	switch f.Mode {
	case serverModeDefault, serverModeDesktop:
	case serverModeRefreshOnly:
		return serverFlags{}, fmt.Errorf("-mode %s : %w", f.Mode, errModeUnavailable)
	default:
		return serverFlags{}, fmt.Errorf("-mode %q inconnu (valeurs admises : %s, %s)",
			f.Mode, serverModeDesktop, serverModeRefreshOnly)
	}
	return f, nil
}

// parseServerFlagsOrExit lit os.Args[1:] et sort du processus sur une ligne de commande
// refusée (code flagsExitCode, message et rappel d'usage sur stderr) ; -h sort en 0.
func parseServerFlagsOrExit() serverFlags {
	f, err := parseServerFlags(os.Args[1:])
	switch {
	case err == nil:
		return f
	case errors.Is(err, flag.ErrHelp):
		fmt.Fprintln(os.Stderr, serverUsage)
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "server : %v\n%s\n", err, serverUsage)
		os.Exit(flagsExitCode)
	}
	return serverFlags{}
}
