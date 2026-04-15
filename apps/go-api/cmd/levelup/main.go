// cmd/levelup — CLI d'exploitation LevelUp.
//
// Sous-commandes disponibles :
//
//	levelup backup         --gamertag X [--output-dir D] [--compression-level N]
//	levelup restore        --gamertag X --backup-dir D [--replace] [--dry-run] [--tables T1,T2]
//	levelup healthcheck    [--verbose]
//	levelup diagnose       --db PATH [--verbose]
//	levelup check-env
//	levelup archive        --gamertag X --xuid U --cutoff YYYY-MM-DD [--delete-after] [--dry-run]
//	levelup index-media    --gamertag X [--force-rescan] [--tolerance-min N]
//	levelup seed           career-ranks | citation-mappings | medals
//	levelup notify-version --version v1.2.3
//	levelup notify-sync    --gamertag X --op sync_delta --duration 120s [--matches N]
//	levelup compare-db     --go-db PATH --python-db PATH [--json]
//	levelup gate-check     [--gamertag X] [--json]
//
// Variables d'environnement : LEVELUP_REPO_ROOT (auto-détecté si absent).
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"encoding/json"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/notify"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/validation"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "erreur config: %v\n", err)
		os.Exit(1)
	}

	subcmd := os.Args[1]
	args := os.Args[2:]

	var exitErr error
	switch subcmd {
	case "backup":
		exitErr = runBackup(cfg, args)
	case "restore":
		exitErr = runRestore(cfg, args)
	case "healthcheck":
		exitErr = runHealthcheck(cfg, args)
	case "diagnose":
		exitErr = runDiagnose(args)
	case "check-env":
		exitErr = runCheckEnv(cfg)
	case "archive":
		exitErr = runArchive(cfg, args)
	case "index-media":
		exitErr = runIndexMedia(cfg, args)
	case "seed":
		exitErr = runSeed(cfg, args)
	case "notify-version":
		exitErr = runNotifyVersion(cfg, args)
	case "notify-sync":
		exitErr = runNotifySync(cfg, args)
	case "compare-db":
		exitErr = runCompareDB(cfg, args)
	case "gate-check":
		exitErr = runGateCheck(cfg, args)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "sous-commande inconnue: %q\n", subcmd)
		printUsage()
		os.Exit(1)
	}

	if exitErr != nil {
		fmt.Fprintf(os.Stderr, "erreur: %v\n", exitErr)
		os.Exit(1)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// backup
// ─────────────────────────────────────────────────────────────────────────────

func runBackup(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("backup", flag.ExitOnError)
	gamertag := fs.String("gamertag", "", "Gamertag du joueur (obligatoire)")
	outputDir := fs.String("output-dir", "", "Répertoire de sortie (défaut: data/backups/<gamertag>)")
	level := fs.Int("compression-level", 9, "Niveau compression Zstd (1-22)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *gamertag == "" {
		return fmt.Errorf("--gamertag est obligatoire")
	}
	dbPath := filepath.Join(cfg.RepoRoot, "data", "players", *gamertag, "stats.duckdb")
	outDir := *outputDir
	if outDir == "" {
		outDir = filepath.Join(cfg.RepoRoot, "data", "backups", *gamertag)
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
	dbPath := filepath.Join(cfg.RepoRoot, "data", "players", *gamertag, "stats.duckdb")
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
// healthcheck
// ─────────────────────────────────────────────────────────────────────────────

func runHealthcheck(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("healthcheck", flag.ExitOnError)
	verbose := fs.Bool("verbose", false, "Affichage détaillé")
	if err := fs.Parse(args); err != nil {
		return err
	}
	report := ops.RunHealthcheck(ops.HealthcheckOptions{
		RepoRoot: cfg.RepoRoot,
		Verbose:  *verbose,
	})
	fmt.Print(report.Summary())
	if !report.OK {
		return fmt.Errorf("healthcheck échoué")
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// diagnose
// ─────────────────────────────────────────────────────────────────────────────

func runDiagnose(args []string) error {
	fs := flag.NewFlagSet("diagnose", flag.ExitOnError)
	dbPath := fs.String("db", "", "Chemin vers la DB DuckDB (obligatoire)")
	verbose := fs.Bool("verbose", false, "Afficher le nb de lignes par table")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dbPath == "" {
		return fmt.Errorf("--db est obligatoire")
	}
	report, err := ops.DiagnoseDB(ops.DiagnoseOptions{DBPath: *dbPath, Verbose: *verbose})
	if err != nil {
		return err
	}
	fmt.Print(ops.FormatDiagnoseReport(report))
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// check-env
// ─────────────────────────────────────────────────────────────────────────────

func runCheckEnv(cfg *config.AppConfig) error {
	fmt.Printf("LevelUp Go — vérification environnement\n\n")
	fmt.Printf("LEVELUP_REPO_ROOT : %s\n", cfg.RepoRoot)
	fmt.Printf("db_profiles.json  : %s\n", cfg.DBProfilesPath)
	fmt.Printf("app_settings.json : %s\n", cfg.AppSettingsPath)
	fmt.Println()
	// Healthcheck rapide
	report := ops.RunHealthcheck(ops.HealthcheckOptions{RepoRoot: cfg.RepoRoot})
	fmt.Print(report.Summary())
	if !report.OK {
		return fmt.Errorf("environnement incomplet")
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// archive
// ─────────────────────────────────────────────────────────────────────────────

func runArchive(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("archive", flag.ExitOnError)
	gamertag := fs.String("gamertag", "", "Gamertag du joueur (obligatoire)")
	xuid := fs.String("xuid", "", "XUID du joueur (obligatoire)")
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
	result, err := ops.ArchiveMatches(ops.ArchiveOptions{
		Gamertag:     *gamertag,
		XUID:         *xuid,
		PlayerDBPath: filepath.Join(cfg.RepoRoot, "data", "players", *gamertag, "stats.duckdb"),
		SharedDBPath: filepath.Join(cfg.RepoRoot, "data", "warehouse", "shared_matches_v2.duckdb"),
		ArchiveDir:   filepath.Join(cfg.RepoRoot, "data", "players", *gamertag, "archive"),
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
	force := fs.Bool("force-rescan", false, "Réindexer tous les fichiers")
	tolMin := fs.Int("tolerance-min", 5, "Tolérance association match (minutes)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *gamertag == "" {
		return fmt.Errorf("--gamertag est obligatoire")
	}
	result, err := ops.IndexMedia(ops.MediaIndexOptions{
		Gamertag:     *gamertag,
		PlayerDBPath: filepath.Join(cfg.RepoRoot, "data", "players", *gamertag, "stats.duckdb"),
		CapturesDir:  filepath.Join(cfg.RepoRoot, "data", "players", *gamertag, "captures"),
		ForceRescan:  *force,
		ToleranceMin: *tolMin,
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
	opts := ops.SeedOptions{
		MetaDBPath: filepath.Join(cfg.RepoRoot, "data", "warehouse", "metadata.duckdb"),
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

// ─────────────────────────────────────────────────────────────────────────────
// notify-version
// ─────────────────────────────────────────────────────────────────────────────

func runNotifyVersion(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("notify-version", flag.ExitOnError)
	version := fs.String("version", "", "Version à notifier ex: v6.5.0 (obligatoire)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *version == "" {
		return fmt.Errorf("--version est obligatoire")
	}
	notifyCfg := notify.LoadNotifyConfig(cfg.AppSettingsPath)
	sent := notify.NotifyNewVersion(notifyCfg, *version)
	if sent {
		fmt.Printf("✅ Notification version %s envoyée\n", *version)
	} else {
		fmt.Printf("ℹ️  Notification version %s ignorée (déjà notifiée, patch seul, ou désactivée)\n", *version)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// notify-sync  (test / debug)
// ─────────────────────────────────────────────────────────────────────────────

func runNotifySync(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("notify-sync", flag.ExitOnError)
	gamertag := fs.String("gamertag", "", "Gamertag du joueur (obligatoire)")
	op := fs.String("op", "sync_delta", "Opération : sync_delta | sync_full | backfill")
	durationSec := fs.Int("duration", 0, "Durée de l'opération en secondes")
	matches := fs.Int("matches", 0, "Nombre de matchs synchronisés")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *gamertag == "" {
		return fmt.Errorf("--gamertag est obligatoire")
	}
	notifyCfg := notify.LoadNotifyConfig(cfg.AppSettingsPath)
	finishedAt := time.Now()
	startedAt := finishedAt.Add(-time.Duration(*durationSec) * time.Second)
	players := []notify.PlayerSyncResult{
		{Gamertag: *gamertag, MatchesSynced: *matches},
	}
	notify.NotifySync(notifyCfg, *op, startedAt, finishedAt, players, true, false)
	fmt.Printf("✅ Notification sync envoyée pour %s\n", *gamertag)
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Usage
// ─────────────────────────────────────────────────────────────────────────────

// ─────────────────────────────────────────────────────────────────────────────
// compare-db
// ─────────────────────────────────────────────────────────────────────────────

func runCompareDB(_ *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("compare-db", flag.ExitOnError)
	goPath := fs.String("go-db", "", "Chemin vers la DB Go stats.duckdb (obligatoire)")
	pyPath := fs.String("python-db", "", "Chemin vers la DB Python stats.duckdb (obligatoire)")
	asJSON := fs.Bool("json", false, "Sortie JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *goPath == "" || *pyPath == "" {
		return fmt.Errorf("--go-db et --python-db sont obligatoires")
	}
	report, err := validation.ComparePlayerDBs(*goPath, *pyPath)
	if err != nil {
		return err
	}
	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}
	fmt.Print(report.Summary)
	if !report.OverallOK {
		return fmt.Errorf("divergences détectées")
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// gate-check
// ─────────────────────────────────────────────────────────────────────────────

func runGateCheck(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("gate-check", flag.ExitOnError)
	gamertag := fs.String("gamertag", "", "Gamertag du joueur de référence (optionnel)")
	asJSON := fs.Bool("json", false, "Sortie JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	gateCfg := validation.GateCheckConfig{
		RepoRoot:       cfg.RepoRoot,
		DBProfilesPath: cfg.DBProfilesPath,
		Gamertag:       *gamertag,
	}
	report := validation.RunGateCheck4(gateCfg)
	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}
	fmt.Print(report.Format())
	if !report.AllPassed {
		return fmt.Errorf("gate phase 4 non validée")
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Usage
// ─────────────────────────────────────────────────────────────────────────────

func printUsage() {
	fmt.Print(`levelup — Outil d'exploitation LevelUp (backend Go)

Usage:
  levelup <commande> [options]

Commandes:
  backup          Sauvegarder une DB joueur en Parquet Zstd
  restore         Restaurer une DB joueur depuis un backup Parquet
  healthcheck     Diagnostic d'intégrité des bases DuckDB
  diagnose        Inspecter le schéma d'une DB DuckDB
  check-env       Vérifier l'environnement et la configuration
  archive         Archiver les matchs anciens en Parquet
  index-media     Indexer et associer les médias au joueur
  seed            Peupler les référentiels metadata.duckdb
  notify-version  Envoyer une notification Discord de nouvelle version
  notify-sync     Envoyer une notification Discord de fin de sync (test/debug)
  compare-db      Comparer la parité Go vs Python (DB joueur)
  gate-check      Vérifier la checklist Gate Phase 4

Options globales:
  LEVELUP_REPO_ROOT        Racine du repo (auto-détecté si absent)
  LEVELUP_NOTIFY_VERSIONS  Mettre à '1' pour activer les notifs de version en prod
  DISCORD_WEBHOOK_URL      URL webhook Discord (prévaut sur app_settings.json)

Aide par commande:
  levelup <commande> --help
`)
}
