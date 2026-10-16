package main

// cmd_pool_sharing.go — sous-commandes `levelup pool-sharing list` et `grandfather`.
//
// Le jeton d'un joueur n'entre dans le pool de synchro partagé que s'il l'a décidé
// (auth.UserTokens.PoolSharing). `list` montre où en est chaque joueur ; `grandfather`
// enregistre l'accord de ceux qui ont donné le leur AVANT l'existence du réglage, hors de
// l'application. C'est un geste d'opérateur, à ne faire que pour des accords réellement
// recueillis : il écrit un consentement au nom des joueurs.
//
// `grandfather` exige la désignation des joueurs (`--players`, ou `--all`) et reste une
// SIMULATION tant qu'on n'a pas dit `--yes`. Il n'écrase jamais une décision existante
// (accord ou refus) et ne marque que les joueurs qui ont un jeton.
//
// Usage :
//
//	levelup pool-sharing list
//	levelup pool-sharing grandfather --players GT1,GT2        # simulation
//	levelup pool-sharing grandfather --players GT1,GT2 --yes  # enregistre
//	levelup pool-sharing grandfather --all [--yes]            # tous les joueurs sans décision

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

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
		return errors.New("usage: levelup pool-sharing list | levelup pool-sharing grandfather --players GT1,GT2 | --all [--yes]")
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
	players := fs.String("players", "", "joueurs dont l'accord a été recueilli : gamertags ou xuids, séparés par des virgules")
	all := fs.Bool("all", false, "tous les joueurs sans décision (à n'utiliser que si TOUS ont donné leur accord)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if (*players == "") == !*all {
		return errors.New("grandfather : désigner les joueurs avec --players GT1,GT2 (ou xuids), ou --all ; jamais les deux")
	}

	rows, err := poolSharingRows(store)
	if err != nil {
		return err
	}
	chosen, err := selectGrandfathered(rows, *players, *all)
	if err != nil {
		return err
	}

	xuids := make([]string, 0, len(chosen))
	for _, t := range chosen {
		state := store.PoolSharingState(t.XUID)
		if state != domain.PoolSharingUndecided {
			fmt.Fprintf(out, "ignoré (déjà %s) : %s (%s)\n", state, t.Gamertag, t.XUID)
			continue
		}
		fmt.Fprintf(out, "accord à enregistrer : %s (%s)\n", t.Gamertag, t.XUID)
		xuids = append(xuids, t.XUID)
	}
	if !*yes {
		fmt.Fprintf(out, "SIMULATION : %d joueur(s) seraient marqués. Relancer avec --yes pour enregistrer.\n", len(xuids))
		return nil
	}
	n, err := store.GrandfatherPoolSharing(xuids, auth_platform.PoolSharingBasisGrandfathered)
	fmt.Fprintf(out, "%d accord(s) enregistré(s).\n", n)
	return err
}

// selectGrandfathered résout la désignation de l'opérateur. Un nom inconnu fait échouer la
// commande AVANT toute écriture : une faute de frappe ne doit pas laisser un ami hors du
// pool sans que personne ne le voie.
func selectGrandfathered(rows []*auth_platform.UserTokens, players string, all bool) ([]*auth_platform.UserTokens, error) {
	if all {
		return rows, nil
	}
	byKey := make(map[string]*auth_platform.UserTokens, 2*len(rows))
	for _, t := range rows {
		byKey[strings.ToLower(t.Gamertag)] = t
		byKey[t.XUID] = t
	}
	var chosen []*auth_platform.UserTokens
	var unknown []string
	for _, name := range strings.Split(players, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if t, ok := byKey[strings.ToLower(name)]; ok {
			chosen = append(chosen, t)
			continue
		}
		unknown = append(unknown, name)
	}
	if len(unknown) > 0 {
		return nil, fmt.Errorf("grandfather : joueur(s) sans jeton enregistré : %s (voir pool-sharing list)", strings.Join(unknown, ", "))
	}
	return chosen, nil
}
