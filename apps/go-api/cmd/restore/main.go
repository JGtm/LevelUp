// restore liste les snapshots de backup ou restaure un snapshot DuckDB depuis restic.
//
// Usage:
//
//	restore --list                   liste les snapshots disponibles (date, ID, hôte)
//	restore                          restaure le snapshot le plus récent
//	restore --date 2026-05-25        restaure le snapshot le plus récent de ce jour
//	restore --snapshot 6ba84d2b      restaure un snapshot par ID court
//	restore --output /tmp/restore/   répertoire cible (défaut: data/restore/YYYY-MM-DD/)
//	restore --live                   restaure directement sur les DBs de production
//	restore --dry-run                simule sans modifier aucun fichier
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/pkg/duckdbbackup"
)

func main() {
	listFlag := flag.Bool("list", false, "liste les snapshots disponibles")
	dateFlag := flag.String("date", "", "date du snapshot à restaurer (YYYY-MM-DD)")
	snapFlag := flag.String("snapshot", "", "ID court ou complet du snapshot à restaurer")
	outFlag := flag.String("output", "", "répertoire de sortie (défaut: data/restore/YYYY-MM-DD/)")
	liveFlag := flag.Bool("live", false, "restaure sur les DBs de production (ATTENTION: arrêter le serveur)")
	dryRun := flag.Bool("dry-run", false, "simule sans modifier aucun fichier")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fatalf("config: %v", err)
	}

	restorer := duckdbbackup.NewRestorer(duckdbbackup.Config{
		ResticBin:  "restic",
		ResticRepo: cfg.Backup.ResticRepo,
		BackupDir:  cfg.Backup.BackupDir,
	})
	ctx := context.Background()

	if *listFlag {
		runList(ctx, restorer)
		return
	}

	snap, date, err := resolveSnapshot(ctx, restorer, *snapFlag, *dateFlag)
	if err != nil {
		fatalf("%v", err)
	}
	fmt.Printf("Snapshot sélectionné : %s (%s)\n", snap, date)

	if *liveFlag && *outFlag != "" {
		fatalf("--output et --live sont incompatibles")
	}

	outDir := *outFlag
	if outDir == "" && !*liveFlag {
		outDir = filepath.Join(cfg.RepoRoot, "data", "restore", date)
	}

	pr := title.NewPathResolver(cfg.RepoRoot)

	if *dryRun {
		fmt.Println("[dry-run] aucune modification ne sera effectuée.")
	}
	if *liveFlag {
		fmt.Println("ATTENTION: les DBs de production vont être remplacées.")
		fmt.Println("Assurez-vous que le serveur Go est arrêté.")
		fmt.Print("Continuer ? [y/N] ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(answer)) != "y" {
			fmt.Println("Annulé.")
			return
		}
	}

	tmpDir, err := os.MkdirTemp("", "levelup-restore-*")
	if err != nil {
		fatalf("tmpdir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	stagingDir, err := restorer.ExtractSnapshot(ctx, snap, tmpDir)
	if err != nil {
		fatalf("extraction snapshot: %v", err)
	}

	var keyToPath func(string) string
	if *liveFlag {
		keyToPath = livePathResolver(pr)
	} else {
		fmt.Printf("Restauration vers : %s\n", outDir)
		keyToPath = outputPathResolver(outDir)
	}

	result, err := duckdbbackup.ImportFromStaging(ctx, stagingDir, keyToPath, *dryRun)
	if err != nil {
		fatalf("import: %v", err)
	}

	printResult(result)
}

// runList affiche les snapshots disponibles triés du plus récent au plus ancien.
func runList(ctx context.Context, restorer *duckdbbackup.Restorer) {
	snaps, err := restorer.ListSnapshots(ctx)
	if err != nil {
		fatalf("liste snapshots: %v", err)
	}
	if len(snaps) == 0 {
		fmt.Println("Aucun snapshot disponible.")
		return
	}
	fmt.Printf("%-10s  %-22s  %s\n", "ID", "Date", "Hôte")
	fmt.Println(strings.Repeat("-", 52))
	for _, s := range snaps {
		fmt.Printf("%-10s  %-22s  %s\n",
			s.ShortID,
			s.Time.Format("2006-01-02 15:04:05"),
			s.Hostname)
	}
}

// resolveSnapshot retourne l'ID et la date du snapshot à restaurer.
func resolveSnapshot(ctx context.Context, restorer *duckdbbackup.Restorer, snapFlag, dateFlag string) (id, date string, err error) {
	if snapFlag != "" && dateFlag != "" {
		return "", "", fmt.Errorf("--snapshot et --date sont incompatibles")
	}

	snaps, listErr := restorer.ListSnapshots(ctx)
	if listErr != nil {
		if snapFlag != "" {
			// Listing failed but we have an explicit ID — proceed with current date.
			slog.Warn("restore: impossible de lister les snapshots", "err", listErr)
			return snapFlag, time.Now().Format("2006-01-02"), nil
		}
		return "", "", listErr
	}
	if len(snaps) == 0 {
		return "", "", fmt.Errorf("aucun snapshot disponible")
	}

	if snapFlag != "" {
		for _, s := range snaps {
			if s.ShortID == snapFlag || s.ID == snapFlag {
				return s.ShortID, s.Time.Format("2006-01-02"), nil
			}
		}
		// Not found in list — try anyway (restic will error if invalid).
		return snapFlag, time.Now().Format("2006-01-02"), nil
	}

	if dateFlag != "" {
		for _, s := range snaps {
			if s.Time.Format("2006-01-02") == dateFlag {
				return s.ShortID, dateFlag, nil
			}
		}
		return "", "", fmt.Errorf("aucun snapshot trouvé pour la date %s", dateFlag)
	}

	// Default: most recent snapshot.
	s := snaps[0]
	slog.Info("restore: sélection du snapshot le plus récent", "id", s.ShortID, "date", s.Time.Format("2006-01-02"))
	return s.ShortID, s.Time.Format("2006-01-02"), nil
}

// livePathResolver mappe les clés de backup sur les chemins DuckDB de production.
func livePathResolver(pr *title.PathResolver) func(string) string {
	return func(key string) string {
		parts := strings.Split(key, ":")
		switch {
		case key == "xbox_aliases":
			return pr.GlobalXuidAliasesDBPath()
		case len(parts) == 2:
			slug, db := parts[0], parts[1]
			switch db {
			case "shared_matches_v2":
				return pr.SharedDBPath(slug)
			case "metadata":
				return pr.MetadataDBPath(slug)
			case "shared_pve":
				return pr.SharedPVEDBPath(slug)
			case "shared_social":
				return pr.SharedSocialDBPath(slug)
			}
		case len(parts) == 3 && parts[1] == "player":
			return pr.PlayerDBPath(parts[0], parts[2])
		}
		return ""
	}
}

// outputPathResolver mappe les clés de backup sur des chemins dans outputDir.
// Structure miroir du staging : outputDir/{slug}/{db}.duckdb, outputDir/{slug}/players/{gt}/stats.duckdb
func outputPathResolver(outputDir string) func(string) string {
	return func(key string) string {
		parts := strings.Split(key, ":")
		switch {
		case key == "xbox_aliases":
			return filepath.Join(outputDir, "xbox_aliases.duckdb")
		case len(parts) == 2:
			return filepath.Join(outputDir, parts[0], parts[1]+".duckdb")
		case len(parts) == 3 && parts[1] == "player":
			return filepath.Join(outputDir, parts[0], "players", parts[2], "stats.duckdb")
		}
		return ""
	}
}

func printResult(result *duckdbbackup.ImportResult) {
	fmt.Printf("\nRestauration terminée en %dms\n", result.DurationMs)
	fmt.Printf("DBs restaurées (%d) :\n", len(result.Restored))
	for _, key := range result.Restored {
		fmt.Printf("  + %s\n", key)
	}
	if len(result.Skipped) > 0 {
		fmt.Printf("Ignorées — clé inconnue (%d) :\n", len(result.Skipped))
		for _, key := range result.Skipped {
			fmt.Printf("  - %s\n", key)
		}
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "restore: "+format+"\n", args...)
	os.Exit(1)
}
