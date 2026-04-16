// cmd_ops.go — sous-commandes d'opérations / diagnostic :
// healthcheck, diagnose, check-env, compare-db, gate-check, surface-status.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/validation"
)

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
// surface-status
// ─────────────────────────────────────────────────────────────────────────────

func runSurfaceStatus(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("surface-status", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "Sortie JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ff := &cfg.FeatureFlags

	type surfaceEntry struct {
		Surface string `json:"surface"`
		Backend string `json:"backend"`
	}

	entries := make([]surfaceEntry, 0, len(config.AllSurfaces))
	for _, s := range config.AllSurfaces {
		entries = append(entries, surfaceEntry{
			Surface: string(s),
			Backend: string(ff.BackendFor(s)),
		})
	}

	if *asJSON {
		return json.NewEncoder(os.Stdout).Encode(entries)
	}

	allGo := ff.AllOnGo()
	status := "✅ Migration complète"
	if !allGo {
		status = "⚠️  Bascule partielle — certaines surfaces sur Python"
	}
	fmt.Printf("LevelUp — Statut des surfaces\n%s\n\n", status)
	for _, e := range entries {
		indicator := "✅"
		if e.Backend == "python" {
			indicator = "⚠️ "
		}
		fmt.Printf("  %s %-14s → %s\n", indicator, e.Surface, e.Backend)
	}
	fmt.Printf("\nVar env de rollback : LEVELUP_FF_<SURFACE>=python\n")
	return nil
}
