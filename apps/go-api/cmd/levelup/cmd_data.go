// cmd_data.go — sous-commandes de gestion des données joueur :
// backup, restore, archive, index-media, seed.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/ops"
)

// ─────────────────────────────────────────────────────────────────────────────
// backup
// ─────────────────────────────────────────────────────────────────────────────

func runBackup(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("backup", flag.ExitOnError)
	gamertag := fs.String("gamertag", "", "Gamertag du joueur (obligatoire)")
	titleSlug := fs.String("title", title.DefaultSlug, "Slug du titre (ex: halo_infinite)")
	outputDir := fs.String("output-dir", "", "Répertoire de sortie (défaut: data/backups/<gamertag>)")
	level := fs.Int("compression-level", 9, "Niveau compression Zstd (1-22)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *gamertag == "" {
		return fmt.Errorf("--gamertag est obligatoire")
	}
	pr := title.NewPathResolver(cfg.RepoRoot)
	dbPath := pr.PlayerDBPath(*titleSlug, *gamertag)
	outDir := *outputDir
	if outDir == "" {
		outDir = pr.BackupDir(*titleSlug, *gamertag)
	}
	result, err := ops.BackupPlayer(ops.BackupOptions{
		Gamertag:         *gamertag,
		PlayerDBPath:     dbPath,
		OutputDir:        outDir,
		CompressionLevel: *level,
		IncludeMetadata:  true,
	})
	if err != nil {
		return err
	}
	fmt.Printf("✅ %s\n", result.Message)
	fmt.Printf("   Répertoire: %s\n", result.OutputDir)
	for t, info := range result.Tables {
		fmt.Printf("   %-35s %d lignes  (%.1f KB)\n", t, info.Rows, float64(info.FileSizeBytes)/1024)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// restore
// ─────────────────────────────────────────────────────────────────────────────

func runRestore(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("restore", flag.ExitOnError)
	gamertag := fs.String("gamertag", "", "Gamertag du joueur (obligatoire)")
	titleSlug := fs.String("title", title.DefaultSlug, "Slug du titre (ex: halo_infinite)")
	backupDir := fs.String("backup-dir", "", "Répertoire du backup (obligatoire)")
	tables := fs.String("tables", "", "Tables à restaurer (CSV, vide = toutes)")
	replace := fs.Bool("replace", false, "DROP TABLE avant restauration")
	dryRun := fs.Bool("dry-run", false, "Lister sans modifier")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *gamertag == "" || *backupDir == "" {
		return fmt.Errorf("--gamertag et --backup-dir sont obligatoires")
	}
	pr := title.NewPathResolver(cfg.RepoRoot)
	dbPath := pr.PlayerDBPath(*titleSlug, *gamertag)
	var tableList []string
	if *tables != "" {
		tableList = strings.Split(*tables, ",")
	}
	result, err := ops.RestorePlayer(ops.RestoreOptions{
		Gamertag:     *gamertag,
		PlayerDBPath: dbPath,
		BackupDir:    *backupDir,
		Tables:       tableList,
		Replace:      *replace,
		DryRun:       *dryRun,
	})
	if err != nil {
		return err
	}
	fmt.Printf("✅ %s\n", result.Message)
	fmt.Printf("   Tables: %s\n", strings.Join(result.TablesLoaded, ", "))
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// archive
// ─────────────────────────────────────────────────────────────────────────────

func runArchive(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("archive", flag.ExitOnError)
	gamertag := fs.String("gamertag", "", "Gamertag du joueur (obligatoire)")
	xuid := fs.String("xuid", "", "XUID du joueur (obligatoire)")
	titleSlug := fs.String("title", title.DefaultSlug, "Slug du titre (ex: halo_infinite)")
	cutoffStr := fs.String("cutoff", "", "Date limite YYYY-MM-DD (obligatoire)")
	deleteAfter := fs.Bool("delete-after", false, "Supprimer les matchs après archivage")
	dryRun := fs.Bool("dry-run", false, "Lister sans archiver")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *gamertag == "" || *xuid == "" || *cutoffStr == "" {
		return fmt.Errorf("--gamertag, --xuid et --cutoff sont obligatoires")
	}
	cutoff, err := time.Parse("2006-01-02", *cutoffStr)
	if err != nil {
		return fmt.Errorf("format --cutoff invalide (attendu: YYYY-MM-DD): %w", err)
	}
	pr := title.NewPathResolver(cfg.RepoRoot)
	result, err := ops.ArchiveMatches(ops.ArchiveOptions{
		Gamertag:     *gamertag,
		XUID:         *xuid,
		PlayerDBPath: pr.PlayerDBPath(*titleSlug, *gamertag),
		SharedDBPath: pr.SharedDBPath(*titleSlug),
		ArchiveDir:   pr.PlayerArchiveDir(*titleSlug, *gamertag),
		CutoffDate:   cutoff,
		DeleteAfter:  *deleteAfter,
		DryRun:       *dryRun,
		ByYear:       true,
	})
	if err != nil {
		return err
	}
	fmt.Printf("✅ %s\n", result.Message)
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// index-media
// ─────────────────────────────────────────────────────────────────────────────

func runIndexMedia(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("index-media", flag.ExitOnError)
	gamertag := fs.String("gamertag", "", "Gamertag du joueur (obligatoire)")
	titleSlug := fs.String("title", title.DefaultSlug, "Slug du titre (ex: halo_infinite)")
	force := fs.Bool("force-rescan", false, "Réindexer tous les fichiers")
	tolMin := fs.Int("tolerance-min", 5, "Tolérance association match (minutes)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *gamertag == "" {
		return fmt.Errorf("--gamertag est obligatoire")
	}
	pr := title.NewPathResolver(cfg.RepoRoot)
	result, err := ops.IndexMedia(ops.MediaIndexOptions{
		Gamertag:            *gamertag,
		PlayerDBPath:        pr.PlayerDBPath(*titleSlug, *gamertag),
		SharedSocialDBPath:  pr.SharedSocialDBPath(*titleSlug),
		SharedMatchesDBPath: pr.SharedDBPath(*titleSlug),
		CapturesDir:         pr.PlayerCapturesDir(*titleSlug, *gamertag),
		ForceRescan:         *force,
		ToleranceMin:        *tolMin,
	})
	if err != nil {
		return err
	}
	fmt.Printf("✅ Indexation terminée\n")
	fmt.Printf("   Scannés:    %d\n", result.Scanned)
	fmt.Printf("   Nouveaux:   %d\n", result.NewFiles)
	fmt.Printf("   Associés:   %d\n", result.Associated)
	for _, e := range result.Errors {
		fmt.Fprintf(os.Stderr, "   ⚠️  %s\n", e)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// seed
// ─────────────────────────────────────────────────────────────────────────────

func runSeed(cfg *config.AppConfig, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("composant requis: career-ranks | citation-mappings | medals")
	}
	// seed utilise toujours metadata.duckdb du titre par défaut
	pr := title.NewPathResolver(cfg.RepoRoot)
	opts := ops.SeedOptions{
		MetaDBPath: pr.MetadataDBPath(title.DefaultSlug),
		DataDir:    filepath.Join(cfg.RepoRoot, "data"),
	}
	var result ops.SeedResult
	var err error
	switch args[0] {
	case "career-ranks":
		result, err = ops.SeedCareerRanks(opts)
	case "citation-mappings":
		result, err = ops.SeedCitationMappings(opts)
	case "medals":
		result, err = ops.SeedMedalDefinitions(opts)
	default:
		return fmt.Errorf("composant inconnu: %q (career-ranks | citation-mappings | medals)", args[0])
	}
	if err != nil {
		return err
	}
	fmt.Printf("✅ %s: %s\n", result.Component, result.Message)
	return nil
}
