package parquetbackup

// command.go — LA commande de restauration d'un jeu de sauvegarde, écrite UNE fois et appelée par deux portes :
// `levelup-server -mode restore` (le serveur livré avec l'application : l'installeur ne livre pas levelup.exe) et
// l'alias `levelup backup restore` (cmd/levelup). Les drapeaux, le verrou d'instance, la restauration et le
// bilan sont ceux de ce fichier ; aucune porte n'en recopie une ligne.
//
// Usage :
//
//	levelup-server.exe -mode restore --from D:\Sauvegardes\levelup-20261103-0930 --to D:\NouvellesDonnees [--dry-run] [--force]
//
// Cas « disque mort, nouveau PC » : installer l'application, la fermer, restaurer le dernier jeu dans la
// racine de données de l'application (`%LOCALAPPDATA%\LevelUp`), puis relancer : les migrations du
// démarrage mettent les bases à niveau. Les jetons d'un autre PC ne se déchiffrent pas (coffre du système) :
// se reconnecter depuis l'assistant. Les films archivés ne font pas partie du jeu.
//
// LE VERROU D'INSTANCE DE LA RACINE CIBLE est pris avant d'écrire et refuse la restauration tant qu'il est tenu :
// tout serveur, quel que soit son profil, le tient pendant qu'il tourne. `--force` autorise le remplacement de
// bases ou de fichiers EXISTANTS, jamais le passage outre un serveur en marche ; seul `--dry-run`, qui n'écrit
// rien, ignore le verrou.

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/instancelock"
)

// RestoreCommand : les arguments de la commande de restauration.
type RestoreCommand struct {
	// From : le dossier du jeu (levelup-AAAAMMJJ-HHMM).
	From string
	// To : la racine de DONNÉES cible (le dossier qui contient titles/, auth/…).
	To string
	// DryRun : vérifier le jeu et dire ce qui serait restauré, sans rien écrire.
	DryRun bool
	// Force : remplacer les bases et fichiers déjà présents (jamais outrepasser le verrou d'instance).
	Force bool
}

// RestoreUsage rend le rappel d'usage de la commande appelée `name`.
func RestoreUsage(name string) string {
	return "usage : " + name + " --from <dossier du jeu> --to <racine de données> [--dry-run] [--force]"
}

// RegisterRestoreFlags déclare les drapeaux de la commande sur fs et rend la structure qu'ils remplissent
// (lue une fois fs.Parse fait).
func RegisterRestoreFlags(fs *flag.FlagSet) *RestoreCommand {
	var c RestoreCommand
	fs.StringVar(&c.From, "from", "", "dossier du jeu de sauvegarde (levelup-AAAAMMJJ-HHMM)")
	fs.StringVar(&c.To, "to", "", "racine de DONNEES cible (le dossier qui contient titles/, auth/…)")
	fs.BoolVar(&c.DryRun, "dry-run", false, "verifie le jeu et dit ce qui serait restaure, n'ecrit rien")
	fs.BoolVar(&c.Force, "force", false, "remplace les bases et fichiers deja presents dans la racine cible (jamais un serveur en marche)")
	return &c
}

// ParseRestoreArgs lit les arguments de la commande appelée `name` (alias `levelup backup restore`).
func ParseRestoreArgs(name string, args []string, out io.Writer) (RestoreCommand, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(out)
	c := RegisterRestoreFlags(fs)
	if err := fs.Parse(args); err != nil {
		return RestoreCommand{}, err
	}
	return *c, c.Validate(name)
}

// Validate refuse la commande qui n'a pas ses deux dossiers ; name est le nom de la commande dans l'usage.
func (c RestoreCommand) Validate(name string) error {
	if c.From == "" || c.To == "" {
		return errors.New(RestoreUsage(name))
	}
	return nil
}

// RunRestoreCommand restaure le jeu c.From dans la racine c.To. base porte les racines de ressources et de
// journaux du processus (seule la racine de données change). Le bilan est écrit sur out.
func RunRestoreCommand(ctx context.Context, c RestoreCommand, base title.Roots, out io.Writer) error {
	to, err := filepath.Abs(c.To)
	if err != nil {
		return fmt.Errorf("racine cible : %w", err)
	}
	pr := title.NewPathResolverRoots(title.Roots{Resources: base.Resources, Data: to, Logs: base.Logs, DataSet: true})
	if !c.DryRun {
		lock, err := instancelock.Acquire(pr.InstanceLockPath())
		if errors.Is(err, instancelock.ErrInstanceDejaLancee) {
			return fmt.Errorf("le serveur de %s tourne : l'arreter avant de restaurer (--force ne passe pas outre)", to)
		}
		if err != nil {
			return err
		}
		defer func() { _ = lock.Release() }()
	}
	res, err := Restore(ctx, RestoreOptions{From: c.From, Paths: pr, AuthDir: filepath.Join(to, "auth"), Force: c.Force, DryRun: c.DryRun})
	if err != nil {
		return err
	}
	printRestore(out, res, to, c.DryRun)
	return nil
}

// printRestore écrit le bilan d'une restauration.
func printRestore(out io.Writer, res *RestoreResult, to string, dryRun bool) {
	verbe := "restaurees"
	if dryRun {
		verbe = "qui seraient restaurees"
	}
	m := res.Manifest
	_, _ = fmt.Fprintf(out, "Jeu %s (application %s, %d bases, %d lignes)%s\n", m.Name, m.AppVersion, len(m.Databases), m.TotalRows,
		map[bool]string{true: " : PARTIEL (voir le manifeste)", false: ""}[m.Partial])
	_, _ = fmt.Fprintf(out, "Racine cible : %s\nBases %s (%d) :\n", to, verbe, len(res.Import.Restored))
	for _, k := range res.Import.Restored {
		_, _ = fmt.Fprintf(out, "  + %s\n", k)
	}
	for _, k := range res.Import.Skipped {
		_, _ = fmt.Fprintf(out, "  - ignoree (cle inconnue) : %s\n", k)
	}
	for _, t := range res.Import.FallbackTables {
		_, _ = fmt.Fprintf(out, "  ! table sans cle ni defaut (schema absent du jeu) : %s\n", t)
	}
	for _, s := range res.Import.SchemaFailures {
		_, _ = fmt.Fprintf(out, "  ! objet du schema non recree : %s\n", s)
	}
	if !dryRun {
		_, _ = fmt.Fprintf(out, "Fichiers de configuration remis en place : %d\n", len(res.ConfigRestored))
		_, _ = fmt.Fprintln(out, "Relancer l'application : les migrations mettent les bases a niveau. Les jetons d'un autre poste ne se dechiffrent pas : se reconnecter.")
	}
}
