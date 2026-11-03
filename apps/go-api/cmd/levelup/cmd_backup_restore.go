package main

// cmd_backup_restore.go — `levelup backup restore` : alias de `levelup-server -mode restore`. La commande
// (drapeaux, verrou d'instance, restauration, bilan) est celle de parquetbackup.RunRestoreCommand ; ce fichier
// ne fait que lire la ligne de commande et lui passer les racines du processus.

import (
	"context"
	"io"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ops/parquetbackup"
)

const subcmdBackupRestore = "restore"

// nameBackupRestore : le nom de la commande dans l'usage et les messages.
const nameBackupRestore = "levelup backup restore"

// runBackupRestore restaure un jeu de sauvegarde dans une racine de données.
func runBackupRestore(cfg *config.AppConfig, args []string, out io.Writer) error {
	cmd, err := parquetbackup.ParseRestoreArgs(nameBackupRestore, args, out)
	if err != nil {
		return err
	}
	return parquetbackup.RunRestoreCommand(context.Background(), cmd, cfg.Roots(), out)
}
