package main

// cmd_pool_sharing.go — sous-commandes `levelup pool-sharing list` et `grandfather`.
//
// Le jeton d'un joueur n'entre dans le pool de synchro partagé que s'il l'a décidé
// (auth.UserTokens.PoolSharing). `list` montre où en est chaque joueur ; `grandfather`
// enregistre l'accord de ceux qui ont donné le leur AVANT l'existence du réglage, hors de
// l'application. C'est un geste d'opérateur, à ne faire que pour des accords réellement
// recueillis : il écrit un consentement au nom des joueurs.
//
// `grandfather` est une SIMULATION tant qu'on n'a pas dit `--yes`. Il n'écrase jamais une
// décision existante (accord ou refus) et ne marque que les joueurs qui ont un jeton.
//
// Usage :
//
//	levelup pool-sharing list
//	levelup pool-sharing grandfather          # simulation
//	levelup pool-sharing grandfather --yes    # enregistre

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	auth_platform "levelup/go-api/internal/platform/auth"
)

// runPoolSharing route `pool-sharing <sous-commande>`.
func runPoolSharing(cfg *config.AppConfig, args []string) error {
	return runPoolSharingTo(cfg, args, os.Stdout)
}

func runPoolSharingTo(cfg *config.AppConfig, args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: levelup pool-sharing list | levelup pool-sharing grandfather [--yes]")
	}
	store := auth_platform.NewMultiUserTokenStore(cfg.WatcherTokensDir())
	switch args[0] {
	case "list":
		return poolSharingList(store, out)
	case "grandfather":
		return poolSharingGrandfather(store, args[1:], out)
	default:
		return fmt.Errorf("sous-commande pool-sharing inconnue: %q (list | grandfather)", args[0])
	}
}

// poolSharingRows lit le store et trie par gamertag pour un affichage stable.
func poolSharingRows(store *auth_platform.MultiUserTokenStore) ([]*auth_platform.UserTokens, error) {
	all, err := store.LoadAll()
	if err != nil {
		return nil, err
	}
	rows := make([]*auth_platform.UserTokens, 0, len(all))
	for _, t := range all {
		rows = append(rows, t)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Gamertag < rows[j].Gamertag })
	return rows, nil
}

func poolSharingList(store *auth_platform.MultiUserTokenStore, out io.Writer) error {
	rows, err := poolSharingRows(store)
	if err != nil {
		return err
	}
	for _, t := range rows {
		state := store.PoolSharingState(t.XUID)
		basis := ""
		if t.PoolSharing != nil {
			basis = fmt.Sprintf("  base=%s  le=%s", t.PoolSharing.Basis, t.PoolSharing.DecidedAt.Format("2006-01-02"))
		}
		fmt.Fprintf(out, "%-24s %-18s %s%s\n", t.Gamertag, t.XUID, state, basis)
	}
	return nil
}

func poolSharingGrandfather(store *auth_platform.MultiUserTokenStore, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("pool-sharing grandfather", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	yes := fs.Bool("yes", false, "enregistre l'accord (sinon simulation)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	rows, err := poolSharingRows(store)
	if err != nil {
		return err
	}
	pending := 0
	for _, t := range rows {
		if store.PoolSharingState(t.XUID) == domain.PoolSharingUndecided {
			fmt.Fprintf(out, "accord à enregistrer : %s (%s)\n", t.Gamertag, t.XUID)
			pending++
		}
	}
	if !*yes {
		fmt.Fprintf(out, "SIMULATION : %d joueur(s) seraient marqués. Relancer avec --yes pour enregistrer.\n", pending)
		return nil
	}
	n, err := store.GrandfatherPoolSharing(auth_platform.PoolSharingBasisGrandfathered)
	fmt.Fprintf(out, "%d accord(s) enregistré(s).\n", n)
	return err
}
