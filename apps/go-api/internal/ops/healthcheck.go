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
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	titlePkg "levelup/go-api/internal/domain/title"

	_ "github.com/duckdb/duckdb-go/v2"
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
func RunHealthcheck(ctx context.Context, opts HealthcheckOptions) HealthReport {
	start := time.Now()
	var checks []HealthCheck //nolint:prealloc

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

	// 4. Répertoires de données
	for _, dir := range []string{
		pr.WarehouseDir(titlePkg.DefaultSlug),
		filepath.Join(pr.TitleDataDir(titlePkg.DefaultSlug), "players"),
	} {
		checks = append(checks, checkDirExists(filepath.Base(dir), dir))
	}

	// 5. Bases DuckDB critiques
	for _, db := range []struct{ name, path string }{
		{"shared_matches_v2", pr.SharedDBPath(titlePkg.DefaultSlug)},
		{"metadata", pr.MetadataDBPath(titlePkg.DefaultSlug)},
	} {
		checks = append(checks, checkDuckDB(ctx, db.name, db.path))
	}

	// 6. shared_pve.duckdb (optionnel)
	pvePath := pr.SharedPVEDBPath(titlePkg.DefaultSlug)
	if _, err := os.Stat(pvePath); err == nil {
		checks = append(checks, checkDuckDB(ctx, "shared_pve", pvePath))
	}

	// 7. Joueurs configurés
	playersDir := filepath.Join(pr.TitleDataDir(titlePkg.DefaultSlug), "players")
	if entries, err := os.ReadDir(playersDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			dbPath := pr.PlayerDBPath(titlePkg.DefaultSlug, e.Name())
			checks = append(checks, checkDuckDB(ctx, "player:"+e.Name(), dbPath))
		}
	}

	// 8. Outillage média : ffmpeg/ffprobe (transcoding HLS + miniatures).
	// Leur absence n'empêche pas le serveur de tourner mais désactive le
	// transcoding des MKV multipistes (rendu visible ici plutôt qu'en échec
	// silencieux au premier upload).
	checks = append(checks, checkBinary("ffmpeg"), checkBinary("ffprobe"))

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
		status := "[OK]"
		if !c.OK {
			status = "[KO]"
		}
		fmt.Fprintf(&sb, "%s  %-30s %s\n", status, c.Name, c.Message)
	}
	fmt.Fprintf(&sb, "\nDurée: %s\n", r.Duration.Round(time.Millisecond))
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

// checkBinary vérifie qu'un exécutable est présent dans le PATH. ffmpeg/ffprobe
// sont requis pour le transcoding média HLS et la génération des miniatures.
func checkBinary(name string) HealthCheck {
	path, err := exec.LookPath(name)
	if err != nil {
		return HealthCheck{Name: name, OK: false, Message: "introuvable dans le PATH — transcoding média désactivé"}
	}
	return HealthCheck{Name: name, OK: true, Message: path}
}

// checkDuckDB vérifie qu'une DB DuckDB s'ouvre et répond à COUNT(*).
func checkDuckDB(ctx context.Context, name, path string) HealthCheck {
	if _, err := os.Stat(path); err != nil {
		return HealthCheck{Name: name, OK: false, Message: "fichier absent"}
	}
	db, err := sql.Open("duckdb", path+"?access_mode=read_only")
	if err != nil {
		return HealthCheck{Name: name, OK: false, Message: fmt.Sprintf("ouverture: %v", err)}
	}
	defer db.Close()

	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_type = 'BASE TABLE'").Scan(&count); err != nil {
		return HealthCheck{Name: name, OK: false, Message: fmt.Sprintf("requête: %v", err)}
	}
	return HealthCheck{Name: name, OK: true, Message: fmt.Sprintf("%d tables", count)}
}
