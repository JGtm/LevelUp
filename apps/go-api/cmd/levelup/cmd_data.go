// cmd_data.go — sous-commandes de gestion des données joueur :
// backup, restore, archive, index-media, seed.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
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
	result, err := ops.BackupPlayer(context.Background(), ops.BackupOptions{
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
	result, err := ops.RestorePlayer(context.Background(), ops.RestoreOptions{
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
	result, err := ops.ArchiveMatches(context.Background(), ops.ArchiveOptions{
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
	bufMin := fs.Int("buffer-min", 2, "Buffer autour de la fenêtre match pour l'association (minutes)")
	capturesDir := fs.String("captures-dir", "", "Dossier contenant les captures du joueur (optionnel, surcharge app_settings.json:media_captures_base_dir)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *gamertag == "" {
		return fmt.Errorf("--gamertag est obligatoire")
	}
	pr := title.NewPathResolver(cfg.RepoRoot)
	// Priorité : flag CLI > app_settings.json:media_captures_base_dir > fallback
	// interne PlayerCapturesDir. La centralisation par ResolveCapturesDir
	// garantit qu'on partage le même comportement que les handlers HTTP et
	// que le scanner d'index.
	var resolvedCapturesDir string
	switch {
	case *capturesDir != "":
		resolvedCapturesDir = *capturesDir
	default:
		resolvedCapturesDir = pr.ResolveCapturesDir(*titleSlug, *gamertag, cfg.MediaCapturesBaseDir)
	}
	result, err := ops.IndexMedia(context.Background(), ops.MediaIndexOptions{
		Gamertag:            *gamertag,
		PlayerDBPath:        pr.PlayerDBPath(*titleSlug, *gamertag),
		SharedSocialDBPath:  pr.SharedSocialDBPath(*titleSlug),
		SharedMatchesDBPath: pr.SharedDBPath(*titleSlug),
		CapturesDir:         resolvedCapturesDir,
		CapturesBase:        cfg.MediaCapturesBaseDir,
		ForceRescan:         *force,
		BufferMin:           *bufMin,
		Timezone:            cfg.UserTimezone,
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

// runSeedDemo génère le jeu de données démo (data/demo/) depuis un joueur source.
// Portage de scripts/prepare_demo_data.py (supprimé au commit c03707aa lors du
// nettoyage Python legacy).
func runSeedDemo(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("seed-demo", flag.ExitOnError)
	gamertag := fs.String("gamertag", "JGtm", "Gamertag source (lu depuis db_profiles.json)")
	maxMatches := fs.Int("max-matches", ops.DefaultMaxMatches, "Nombre de matchs à extraire")
	outDir := fs.String("out", "data/demo", "Répertoire de sortie (relatif au repo root)")
	serviceTag := fs.String("service-tag", "SPTA", "Spartan ID affiché sous le gamertag DEMO")
	maxMedia := fs.Int("max-media", ops.DefaultMaxMedia, "Nombre max de fichiers média à extraire")
	noMedia := fs.Bool("no-media", false, "Désactiver l'extraction média")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Résoudre le xuid source depuis db_profiles.json.
	profilesPath := cfg.DBProfilesPath
	if profilesPath == "" {
		profilesPath = filepath.Join(cfg.RepoRoot, "db_profiles.json")
	}
	sourceXUID, srcPlayerRel, err := ops.ResolveSourceXUIDFromProfiles(profilesPath, *gamertag)
	if err != nil {
		return fmt.Errorf("seed-demo: resolve xuid: %w", err)
	}

	pr := title.NewPathResolver(cfg.RepoRoot)
	sourcePlayerDB := filepath.Join(cfg.RepoRoot, srcPlayerRel)
	opts := ops.SeedDemoOptions{
		SourcePlayerDB: sourcePlayerDB,
		SourceSharedDB: pr.SharedDBPath(title.DefaultSlug),
		SourceMetaDB:   pr.MetadataDBPath(title.DefaultSlug),
		SourceXUID:     sourceXUID,
		OutDir:         filepath.Join(cfg.RepoRoot, *outDir),
		MaxMatches:     *maxMatches,
		SourceLabel:    *gamertag,
		ServiceTag:     *serviceTag,
		IncludeMedia:   !*noMedia,
		MaxMedia:       *maxMedia,
		// Racine des player DBs (…/players) pour emprunter une identité Spartan.
		SourcePlayersDir: filepath.Dir(filepath.Dir(sourcePlayerDB)),
		// db_profiles + repo root : pour seeder les player DB des coéquipiers
		// principaux (DemoPlayer2/3) résolus par xuid.
		ProfilesPath: profilesPath,
		RepoRoot:     cfg.RepoRoot,
	}

	ctx := context.Background()
	res, err := ops.SeedDemo(ctx, opts)
	if err != nil {
		return err
	}

	fmt.Printf("✅ Données démo générées dans %s\n", res.OutDir)
	fmt.Printf("   - %d matchs extraits\n", len(res.MatchIDs))
	fmt.Printf("   - metadata.duckdb: copiée\n")
	fmt.Printf("   - shared_matches_v2.duckdb: %v\n", res.SharedRows)
	fmt.Printf("   - stats.duckdb: %v\n", res.PlayerRows)
	if res.MediaCopied > 0 {
		fmt.Printf("   - %d fichiers média copiés\n", res.MediaCopied)
	}
	fmt.Printf("   - db_profiles.json + app_settings.json: écrits\n")
	fmt.Printf("   Durée: %s\n", res.Duration.Round(time.Millisecond))
	return nil
}

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
	// Garantir que le schéma est à jour avant tout seed (idempotent — DuckDB
	// passe les migrations déjà appliquées via schema_migrations).
	if err := applyMigrationsOnDB(opts.MetaDBPath, migration.TargetMetadata); err != nil {
		return fmt.Errorf("migrations metadata avant seed: %w", err)
	}
	var result ops.SeedResult
	var err error
	ctx := context.Background()
	switch args[0] {
	case "career-ranks":
		result, err = ops.SeedCareerRanks(ctx, opts)
	case "citation-mappings":
		result, err = ops.SeedCitationMappings(ctx, opts)
	case "medals":
		result, err = ops.SeedMedalDefinitions(ctx, opts)
	default:
		return fmt.Errorf("composant inconnu: %q (career-ranks | citation-mappings | medals)", args[0])
	}
	if err != nil {
		return err
	}
	fmt.Printf("✅ %s: %s\n", result.Component, result.Message)
	return nil
}
