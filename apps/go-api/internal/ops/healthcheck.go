// Package ops — healthcheck.go : diagnostic d'intégrité des bases DuckDB LevelUp.
//
// Portage de scripts/check_env.py et logique de healthcheck Python.
//
// Usage :
//
//	report := RunHealthcheck(HealthcheckOptions{RepoRoot: "/path/to/levelup"})
//	if !report.OK { fmt.Println(report.Summary()) }
package ops

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
	titlePkg "levelup/go-api/internal/domain/title"
)

// ─────────────────────────────────────────────────────────────────────────────
// Healthcheck principal
// ─────────────────────────────────────────────────────────────────────────────

// HealthcheckOptions configure le diagnostic.
type HealthcheckOptions struct {
	RepoRoot string
	Verbose  bool
}

// HealthReport résume le résultat du diagnostic.
type HealthReport struct {
	OK       bool
	Checks   []HealthCheck
	Duration time.Duration
}

// HealthCheck représente un contrôle individuel.
type HealthCheck struct {
	Name    string
	OK      bool
	Message string
}

// RunHealthcheck effectue tous les contrôles d'intégrité.
// Portage de check_env.py + logique Python healthcheck.
// Utilise PathResolver pour les chemins title-aware (Sprint 44).
func RunHealthcheck(opts HealthcheckOptions) HealthReport {
	start := time.Now()
	var checks []HealthCheck

	pr := titlePkg.NewPathResolver(opts.RepoRoot)

	// 1. Système d'exploitation
	checks = append(checks, HealthCheck{
		Name:    "os",
		OK:      true,
		Message: fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	})

	// 2. Répertoire racine
	checks = append(checks, checkDirExists("repo_root", opts.RepoRoot))

	// 3. Fichiers de configuration
	for _, cfg := range []string{pr.DBProfilesPath(), pr.AppSettingsPath()} {
		checks = append(checks, checkFileExists(filepath.Base(cfg), cfg))
	}

	// 4. Répertoires de données (legacy + title-aware)
	for _, dir := range []string{
		pr.LegacyWarehouseDir(),
		filepath.Join(opts.RepoRoot, "data", "players"),
	} {
		checks = append(checks, checkDirExists(filepath.Base(dir), dir))
	}

	// 5. Bases DuckDB critiques (legacy paths — rétrocompatibilité)
	for _, db := range []struct{ name, path string }{
		{"shared_matches_v2", pr.LegacySharedDBPath()},
		{"metadata", pr.LegacyMetadataDBPath()},
	} {
		checks = append(checks, checkDuckDB(db.name, db.path))
	}

	// 6. shared_pve.duckdb (optionnel, legacy path)
	pvePath := filepath.Join(pr.LegacyWarehouseDir(), "shared_pve.duckdb")
	if _, err := os.Stat(pvePath); err == nil {
		checks = append(checks, checkDuckDB("shared_pve", pvePath))
	}

	// 7. Joueurs configurés
	playersDir := filepath.Join(opts.RepoRoot, "data", "players")
	if entries, err := os.ReadDir(playersDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			dbPath := filepath.Join(playersDir, e.Name(), "stats.duckdb")
			checks = append(checks, checkDuckDB("player:"+e.Name(), dbPath))
		}
	}

	ok := true
	for _, c := range checks {
		if !c.OK {
			ok = false
			break
		}
	}
	return HealthReport{
		OK:       ok,
		Checks:   checks,
		Duration: time.Since(start),
	}
}

// Summary retourne un résumé lisible du rapport.
func (r HealthReport) Summary() string {
	var sb strings.Builder
	for _, c := range r.Checks {
		status := "✅"
		if !c.OK {
			status = "❌"
		}
		sb.WriteString(fmt.Sprintf("%s  %-30s %s\n", status, c.Name, c.Message))
	}
	sb.WriteString(fmt.Sprintf("\nDurée: %s\n", r.Duration.Round(time.Millisecond)))
	if r.OK {
		sb.WriteString("Résultat: TOUT OK\n")
	} else {
		sb.WriteString("Résultat: ERREURS DÉTECTÉES\n")
	}
	return sb.String()
}

// ─────────────────────────────────────────────────────────────────────────────
// Contrôles élémentaires
// ─────────────────────────────────────────────────────────────────────────────

func checkDirExists(name, path string) HealthCheck {
	if _, err := os.Stat(path); err != nil {
		return HealthCheck{Name: name, OK: false, Message: fmt.Sprintf("introuvable: %s", path)}
	}
	return HealthCheck{Name: name, OK: true, Message: path}
}

func checkFileExists(name, path string) HealthCheck {
	if _, err := os.Stat(path); err != nil {
		return HealthCheck{Name: name, OK: false, Message: fmt.Sprintf("introuvable: %s", path)}
	}
	return HealthCheck{Name: name, OK: true, Message: "présent"}
}

// checkDuckDB vérifie qu'une DB DuckDB s'ouvre et répond à COUNT(*).
func checkDuckDB(name, path string) HealthCheck {
	if _, err := os.Stat(path); err != nil {
		return HealthCheck{Name: name, OK: false, Message: "fichier absent"}
	}
	db, err := sql.Open("duckdb", path+"?access_mode=read_only")
	if err != nil {
		return HealthCheck{Name: name, OK: false, Message: fmt.Sprintf("ouverture: %v", err)}
	}
	defer db.Close()

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_type = 'BASE TABLE'").Scan(&count); err != nil {
		return HealthCheck{Name: name, OK: false, Message: fmt.Sprintf("requête: %v", err)}
	}
	return HealthCheck{Name: name, OK: true, Message: fmt.Sprintf("%d tables", count)}
}
