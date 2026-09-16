package main

// cmd_identity.go — sous-commandes `levelup identity list` et `levelup identity purge`.
//
// ELLES REGARDENT ET RETIRENT UNE IDENTITE, PAS UN PROFIL. Quatre registres decrivent
// un joueur — le compte (data/auth/users.json), les credentials
// (data/auth/watcher_tokens/{xuid}.json), le profil de suivi (db_profiles.json) et le
// suivi live du daemon — et la seule cle qui les relie est le XUID (ADR 0035 D1).
// `identity list` les lit ENSEMBLE et signale ce qui ne colle pas ; `identity purge`
// retire une identite de tous, dans l'ordre de l'ADR 0035 D6.
//
// # ELLE NE TOUCHE JAMAIS LA BASE PARTAGEE
//
// Les matchs deja persistes dans shared_matches_v2.duckdb portent aussi les donnees des
// adversaires et des coequipiers du joueur purge, et l'entrepot est append-only par
// construction (ADR 0026). La purge ne l'ouvre meme pas : le paquet qui l'implemente
// n'importe aucun paquet DuckDB (garde-rail archlint).
//
// # ELLE EST UNE SIMULATION TANT QU'ON N'A PAS DIT `--yes`
//
// Sans `--yes`, la commande imprime le rapport COMPLET de ce qu'elle ferait et sort 0,
// sans rien supprimer. C'est la forme qu'on lit avant de decider.
//
// # PRECONDITION : LE SERVEUR NE DOIT PAS TENIR LA PLAYER DB DU JOUEUR
//
// La purge supprime le dossier du joueur avec sa player DB. Elle evince les handles
// DuckDB du PROCESSUS COURANT, pas ceux du serveur : si celui-ci tient le fichier,
// la suppression echoue (verrou Windows) et l'etape est rendue en echec dans le
// rapport. Arreter le serveur, ou purger un joueur qui n'est pas suivi.
//
// Usage :
//
//	levelup identity list                          # l'annuaire, anomalies comprises
//	levelup identity purge 2533274796795729        # SIMULATION, rien n'est supprime
//	levelup identity purge 2533274796795729 --yes  # execute

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	auth_platform "levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/groupstore"
	"levelup/go-api/internal/platform/userstore"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
	"levelup/go-api/internal/service/playerdirectory"
)

// runIdentity route `identity <sous-commande>`.
func runIdentity(cfg *config.AppConfig, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: levelup identity list | levelup identity purge <xuid|gamertag> [--yes]")
	}
	switch args[0] {
	case "list":
		return runIdentityList(cfg, os.Stdout)
	case "purge":
		return runIdentityPurge(cfg, args[1:], os.Stdout)
	default:
		return fmt.Errorf("sous-commande identity inconnue: %q (list | purge)", args[0])
	}
}

// buildCLIDirectory assemble l'annuaire en mode CLI : lecteurs FICHIERS seulement.
// Aucun daemon dans ce processus — donc aucun suivi live a lire ni a retirer, et
// l'annuaire n'a pas de cas particulier a traiter pour autant (tous ses lecteurs
// sont nil-safe).
func buildCLIDirectory(cfg *config.AppConfig) port.PlayerDirectory {
	paths := titlePkg.NewPathResolver(cfg.RepoRoot)
	users := userstore.NewStore(cfg.UsersFilePath())
	groups := groupstore.NewGroupStore(filepath.Join(cfg.AuthDir, "groups.json"))
	tokens := auth_platform.NewMultiUserTokenStore(paths.WatcherTokensDir())
	profiles := service.NewProfileService(cfg.DBProfilesPath, cfg.RepoRoot)

	return playerdirectory.New(playerdirectory.Deps{
		Profiles: cfg,
		Accounts: users,
		Tokens:   tokens,
		FS:       playerdirectory.NewPathFS(cfg.RepoRoot),
		Creator:  profiles,
		Purge: playerdirectory.PurgeDeps{
			Profiles: profiles,
			Tokens:   tokens,
			Groups:   groups,
			Accounts: users,
		},
	})
}

// sortie ecrit le rapport en retenant la PREMIERE erreur d'ecriture. Un rapport
// tronque (tube ferme, disque plein) ne doit pas passer pour un rapport complet :
// l'erreur remonte a l'appelant, elle n'est pas avalee.
type sortie struct {
	w   io.Writer
	err error
}

func (s *sortie) printf(format string, a ...any) {
	if s.err != nil {
		return
	}
	_, s.err = fmt.Fprintf(s.w, format, a...)
}

// runIdentityList imprime l'annuaire : une ligne par identite, ses registres et
// ses anomalies.
func runIdentityList(cfg *config.AppConfig, out io.Writer) error {
	resp, err := buildCLIDirectory(cfg).List(context.Background())
	if err != nil {
		return fmt.Errorf("lecture de l'annuaire: %w", err)
	}
	s := &sortie{w: out}
	if len(resp.Identities) == 0 {
		s.printf("aucune identite connue des registres\n")
		return s.err
	}
	s.printf("%-20s %-18s %-8s %-28s %-7s %s\n",
		"XUID", "GAMERTAG", "COMPTE", "PROFILS", "JETON", "ANOMALIES")
	for _, rec := range resp.Identities {
		s.printf("%-20s %-18s %-8s %-28s %-7s %s\n",
			orTiret(rec.XUID), orTiret(rec.Gamertag), accountCell(rec),
			profilesCell(rec), tokenCell(rec), anomaliesCell(rec))
	}
	s.printf("\n%d identite(s), %d anomalie(s) a regarder, %d information(s)\n",
		resp.Counts["identities"], resp.Counts["warnings"], resp.Counts["infos"])
	return s.err
}

// runIdentityPurge retire une identite. Sans `--yes`, simulation : le rapport est
// imprime et la commande sort 0 sans rien supprimer.
func runIdentityPurge(cfg *config.AppConfig, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("identity purge", flag.ExitOnError)
	yes := fs.Bool("yes", false, "executer reellement (sans ce drapeau, simulation)")
	// `flag` s'arrete au PREMIER argument non-flag : sans ce tri prealable,
	// `identity purge <xuid> --yes` — l'ordre qu'un humain ecrit, et celui que
	// documente l'aide — laisserait --yes non lu, donc une simulation silencieuse
	// la ou l'on croyait executer.
	xuid, rest := splitPositional(args)
	if err := fs.Parse(rest); err != nil {
		return err
	}
	if xuid == "" {
		return errors.New("usage: levelup identity purge <xuid|gamertag> [--yes]")
	}

	report, err := buildCLIDirectory(cfg).Purge(context.Background(), xuid,
		domain.PurgeOptions{DryRun: !*yes})
	// Le rapport s'imprime AVANT tout aiguillage d'erreur : une purge partielle
	// doit se lire — c'est justement quand elle echoue qu'on en a besoin.
	printErr := printPurgeReport(out, report, *yes)
	if errors.Is(err, port.ErrIdentityNotFound) {
		return fmt.Errorf("aucune identite pour la cle %q — un xuid, ou le gamertag d'une identite sans xuid (voir `levelup identity list`)", xuid)
	}
	if err != nil {
		return err
	}
	return printErr
}

// splitPositional sort le PREMIER argument non-flag et rend le reste, pour que
// les drapeaux soient parsables quelle que soit leur place dans la ligne.
func splitPositional(args []string) (string, []string) {
	var positional string
	rest := make([]string, 0, len(args))
	for _, a := range args {
		if positional == "" && !strings.HasPrefix(a, "-") {
			positional = strings.TrimSpace(a)
			continue
		}
		rest = append(rest, a)
	}
	return positional, rest
}

// printPurgeReport imprime le rapport, simulation comprise. Une etape simulee
// n'est ni faite ni en echec : elle dit ce qui SERAIT fait.
func printPurgeReport(out io.Writer, report domain.PurgeReport, executed bool) error {
	if len(report.Steps) == 0 {
		return nil
	}
	entete := "SIMULATION (ajouter --yes pour executer)"
	if executed {
		entete = "EXECUTION"
	}
	s := &sortie{w: out}
	s.printf("purge de l'identite %s (%s) — %s\n",
		orTiret(report.XUID), orTiret(report.Gamertag), entete)
	for _, step := range report.Steps {
		s.printf("  %-12s %-40s %s\n", step.Kind, step.Target, stepIssue(step, executed))
	}
	s.printf("la base partagee des matchs n'est jamais touchee (ADR 0035)\n")
	return s.err
}

func stepIssue(s domain.PurgeStep, executed bool) string {
	switch {
	case !executed:
		return "a faire"
	case s.Err != "":
		return "ECHEC: " + s.Err
	case s.Done:
		return "fait"
	default:
		return "non fait"
	}
}

func orTiret(v string) string {
	if v == "" {
		return "-"
	}
	return v
}

func accountCell(rec domain.IdentityRecord) string {
	if rec.Account == nil {
		return "-"
	}
	return string(rec.Account.Role)
}

func tokenCell(rec domain.IdentityRecord) string {
	switch {
	case rec.Token == nil:
		return "-"
	case rec.Token.ReauthRequired:
		return "reauth"
	case rec.Token.HasRefreshToken:
		return "oui"
	default:
		return "vide"
	}
}

func profilesCell(rec domain.IdentityRecord) string {
	if len(rec.Profiles) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(rec.Profiles))
	for _, p := range rec.Profiles {
		label := p.TitleSlug
		switch {
		case p.AuthOnly:
			label += " (auth)"
		case !p.SyncEnabled:
			label += " (pause)"
		}
		parts = append(parts, label)
	}
	return strings.Join(parts, ", ")
}

func anomaliesCell(rec domain.IdentityRecord) string {
	if len(rec.Anomalies) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(rec.Anomalies))
	for _, a := range rec.Anomalies {
		parts = append(parts, a.Code)
	}
	return strings.Join(parts, " ")
}
