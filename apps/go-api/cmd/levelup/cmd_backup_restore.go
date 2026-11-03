package main

// cmd_backup_restore.go — `levelup backup restore` : la restauration d'un jeu de sauvegarde en Parquet
// (lot L22, ADR 0045) dans une racine de données. Le serveur de cette racine est ARRÊTÉ : la commande
// prend le verrou d'instance de la racine cible et refuse tant qu'il est tenu.
//
// Usage :
//
//	levelup backup restore --from D:\Sauvegardes\levelup-20261103-0930 --to D:\NouvellesDonnees [--dry-run] [--force]
//
// Cas « disque mort, nouveau PC » : installer l'application, la fermer, restaurer le dernier jeu dans la
// racine de données de l'application (`%LOCALAPPDATA%\LevelUp`), puis relancer : les migrations du
// démarrage mettent les bases à niveau. Les jetons d'un autre PC ne se déchiffrent pas (coffre du système) :
// se reconnecter depuis l'assistant. Les films archivés ne font pas partie du jeu.

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/ops/parquetbackup"
	"levelup/go-api/internal/platform/instancelock"
)

const subcmdBackupRestore = "restore"

const usageBackupRestore = "usage: levelup backup restore --from <dossier du jeu> --to <racine de données> [--dry-run] [--force]"

// backupRestoreOptions : les arguments de `backup restore`.
type backupRestoreOptions struct {
	from, to      string
	dryRun, force bool
}

// runBackupRestore restaure un jeu de sauvegarde dans une racine de données.
func runBackupRestore(cfg *config.AppConfig, args []string, out io.Writer) error {
	var o backupRestoreOptions
	fs := flag.NewFlagSet("backup restore", flag.ContinueOnError)
	fs.SetOutput(out)
	fs.StringVar(&o.from, "from", "", "dossier du jeu de sauvegarde (levelup-AAAAMMJJ-HHMM)")
	fs.StringVar(&o.to, "to", "", "racine de DONNEES cible (le dossier qui contient titles/, auth/…)")
	fs.BoolVar(&o.dryRun, "dry-run", false, "verifie le jeu et dit ce qui serait restaure, n'ecrit rien")
	fs.BoolVar(&o.force, "force", false, "remplace les bases et fichiers deja presents dans la racine cible")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if o.from == "" || o.to == "" {
		return errors.New(usageBackupRestore)
	}
	to, err := filepath.Abs(o.to)
	if err != nil {
		return fmt.Errorf("racine cible : %w", err)
	}
	roots := title.Roots{Resources: cfg.Roots().Resources, Data: to, Logs: cfg.Roots().Logs, DataSet: true}
	pr := title.NewPathResolverRoots(roots)
	if !o.dryRun {
		lock, err := instancelock.Acquire(pr.InstanceLockPath())
		if errors.Is(err, instancelock.ErrInstanceDejaLancee) {
			return fmt.Errorf("le serveur de %s tourne : l'arreter avant de restaurer", to)
		}
		if err != nil {
			return err
		}
		defer func() { _ = lock.Release() }()
	}
	res, err := parquetbackup.Restore(context.Background(), parquetbackup.RestoreOptions{
		From: o.from, Paths: pr, AuthDir: filepath.Join(to, "auth"), Force: o.force, DryRun: o.dryRun})
	if err != nil {
		return err
	}
	printRestore(out, res, to, o.dryRun)
	return nil
}

// printRestore écrit le bilan d'une restauration.
func printRestore(out io.Writer, res *parquetbackup.RestoreResult, to string, dryRun bool) {
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
