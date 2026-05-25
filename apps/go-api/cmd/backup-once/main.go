// backup-once exécute un cycle de backup DuckDB → restic unique et affiche le résultat.
// Usage : go run ./cmd/backup-once (ou compiler avec CGO_ENABLED=1 go build)
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/ops"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	pr := title.NewPathResolver(cfg.RepoRoot)
	scheduler := ops.NewLevelUpBackupScheduler(cfg.Backup, pr)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	slog.InfoContext(ctx, "backup: démarrage cycle manuel",
		"repo", cfg.Backup.ResticRepo,
		"dir", cfg.Backup.BackupDir,
		"enabled", cfg.Backup.Enabled)

	result, err := scheduler.RunOnce(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "backup échoué: %v\n", err)
		os.Exit(1)
	}

	if result.Skipped {
		fmt.Println("Backup ignoré: aucune modification détectée depuis le dernier cycle.")
		return
	}

	fmt.Printf("Backup terminé\n")
	fmt.Printf("  snapshot_id : %s\n", result.SnapshotID)
	fmt.Printf("  durée       : %dms\n", result.DurationMs)
	fmt.Printf("  DBs exportées (%d):\n", len(result.Exported))
	for _, key := range result.Exported {
		fmt.Printf("    - %s\n", key)
	}
}
