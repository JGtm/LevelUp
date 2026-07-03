// Package validation — gate.go : automatisation de la checklist Gate Phase 4.
//
// La Gate Phase 4 conditionne le passage en Phase 5 (bascule et extinction Python).
// Ce module vérifie automatiquement les critères objectifs et produit un rapport.
//
// Utilisation :
//
//	report := RunGateCheck4(cfg)
//	fmt.Print(report.Format())
package validation

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"levelup/go-api/internal/config"

	titlePkg "levelup/go-api/internal/domain/title"
)

// tableMatchRegistry est le nom de la table centrale shared_matches_v2 listant
// les matchs uniques. Référencé par la checklist Gate et les tests de
// comparaison (compare_test.go) — centralisé ici pour éviter la duplication.
const tableMatchRegistry = "match_registry"

// ─────────────────────────────────────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────────────────────────────────────

// GateItem est un item de la checklist Gate Phase 4.
type GateItem struct {
	ID      string
	Label   string
	Passed  bool
	Message string // Détail en cas d'échec ou info complémentaire
}

// GateReport est le rapport complet de la Gate Phase 4.
type GateReport struct {
	GeneratedAt time.Time
	Items       []GateItem
	AllPassed   bool
}

// Format retourne le rapport formaté en texte.
func (r *GateReport) Format() string {
	var sb strings.Builder
	sb.WriteString("=== Gate Phase 4 — Checklist passage Phase 5 ===\n")
	fmt.Fprintf(&sb, "Généré le : %s\n\n", r.GeneratedAt.Format("2006-01-02 15:04:05"))

	passed, failed := 0, 0
	for _, item := range r.Items {
		icon := "✅"
		if !item.Passed {
			icon = "❌"
			failed++
		} else {
			passed++
		}
		fmt.Fprintf(&sb, "  %s [%s] %s\n", icon, item.ID, item.Label)
		if item.Message != "" {
			fmt.Fprintf(&sb, "       %s\n", item.Message)
		}
	}

	sb.WriteByte('\n')
	if r.AllPassed {
		fmt.Fprintf(&sb, "✅ GATE PHASE 4 VALIDÉE (%d/%d critères)\n", passed, len(r.Items))
		sb.WriteString("   → Passage en Phase 5 (Sprint 27 - Bascule progressive) autorisé.\n")
	} else {
		fmt.Fprintf(&sb, "❌ GATE PHASE 4 NON VALIDÉE (%d/%d critères — %d échec(s))\n",
			passed, len(r.Items), failed)
		sb.WriteString("   → Corriger les items ❌ avant de passer en Phase 5.\n")
	}
	return sb.String()
}

// GateCheckConfig contient les chemins nécessaires pour la Gate Phase 4.
type GateCheckConfig struct {
	RepoRoot       string
	DBProfilesPath string
	Gamertag       string // joueur de référence pour les vérifications
}

// ─────────────────────────────────────────────────────────────────────────────
// RunGateCheck4 — point d'entrée principal
// ─────────────────────────────────────────────────────────────────────────────

// RunGateCheck4 exécute la checklist automatisée de la Gate Phase 4.
// Sprint 44 : utilise PathResolver pour les chemins.
func RunGateCheck4(ctx context.Context, cfg GateCheckConfig) *GateReport {
	report := &GateReport{
		GeneratedAt: time.Now(),
	}

	pr := titlePkg.NewPathResolver(cfg.RepoRoot)

	type check struct {
		id    string
		label string
		fn    func() (bool, string)
	}

	// MT-10 : les contrôles de DB partagées sont répétés pour CHAQUE titre
	// enregistré (registry.All()). Mono-titre (halo_infinite seul) → une itération,
	// sortie inchangée ; l'id n'est suffixé du slug que si plusieurs titres coexistent.
	titles := titlePkg.DefaultRegistry().All()
	labelTitle := len(titles) > 1
	titledID := func(id, slug string) string {
		if labelTitle {
			return id + "[" + slug + "]"
		}
		return id
	}

	checks := []check{
		{
			"sync-binary",
			"Binaire levelup compilable et exécutable",
			func() (bool, string) {
				return checkBinary(cfg.RepoRoot)
			},
		},
	}

	for _, td := range titles {
		slug := td.Slug
		checks = append(checks,
			check{
				titledID("shared-db", slug),
				"shared_matches_v2.duckdb accessible en lecture",
				func() (bool, string) { return checkDBAccessible(ctx, pr.SharedDBPath(slug)) },
			},
			check{
				titledID("metadata-db", slug),
				"metadata.duckdb accessible en lecture",
				func() (bool, string) { return checkDBAccessible(ctx, pr.MetadataDBPath(slug)) },
			},
			check{
				titledID("shared-tables", slug),
				"Tables critiques présentes dans shared_matches_v2.duckdb",
				func() (bool, string) { return checkSharedTables(ctx, pr.SharedDBPath(slug)) },
			},
			check{
				titledID("shared-views", slug),
				"Vues V6 présentes (v_gamertag_lookup, v_match_full, v_weapon_kills)",
				func() (bool, string) { return checkSharedViews(ctx, pr.SharedDBPath(slug)) },
			},
		)
	}

	checks = append(checks,
		check{
			"migrations-applied",
			"Migrations DuckDB trackées dans schema_migrations",
			func() (bool, string) {
				if cfg.Gamertag == "" {
					return true, "ignoré (pas de gamertag configuré)"
				}
				return checkMigrationsApplied(ctx,
					cfg.RepoRoot, cfg.Gamertag,
				)
			},
		},
		check{
			"player-db",
			"stats.duckdb joueur accessible (player_match_enrichment non vide)",
			func() (bool, string) {
				if cfg.Gamertag == "" {
					return true, "ignoré (pas de gamertag configuré)"
				}
				return checkPlayerDB(ctx, titlePkg.NewPathResolver(cfg.RepoRoot).PlayerDBPath(titlePkg.DefaultSlug, cfg.Gamertag))
			},
		},
		check{
			"db-profiles",
			"db_profiles.json valide (au moins 1 profil configuré)",
			func() (bool, string) {
				return checkDBProfiles(cfg.DBProfilesPath)
			},
		},
		check{
			"discord-notify",
			"Notifications Discord configurées (DISCORD_WEBHOOK_URL ou app_settings.json)",
			func() (bool, string) {
				return checkDiscordNotify(cfg.RepoRoot)
			},
		},
	)

	allPassed := true
	for _, c := range checks {
		passed, msg := c.fn()
		if !passed {
			allPassed = false
		}
		report.Items = append(report.Items, GateItem{
			ID:      c.id,
			Label:   c.label,
			Passed:  passed,
			Message: msg,
		})
	}
	report.AllPassed = allPassed
	return report
}

// ─────────────────────────────────────────────────────────────────────────────
// Checks individuels
// ─────────────────────────────────────────────────────────────────────────────

func checkBinary(repoRoot string) (bool, string) {
	candidates := []string{
		filepath.Join(repoRoot, "apps", "go-api", "bin", "levelup"),
		filepath.Join(repoRoot, "apps", "go-api", "bin", "levelup.exe"),
		filepath.Join(repoRoot, "bin", "levelup"),
		filepath.Join(repoRoot, "bin", "levelup.exe"),
	}
	for _, p := range candidates {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return true, fmt.Sprintf("trouvé : %s (%d KB)", p, info.Size()/1024)
		}
	}
	return false, "binaire introuvable dans apps/go-api/bin/ ou bin/ — lancer 'make build'"
}

func checkDBAccessible(ctx context.Context, dbPath string) (bool, string) {
	if _, err := os.Stat(dbPath); err != nil {
		return false, fmt.Sprintf("fichier absent: %s", dbPath)
	}
	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")
	if err != nil {
		return false, fmt.Sprintf("open error: %v", err)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return false, fmt.Sprintf("ping error: %v", err)
	}
	return true, ""
}

func checkSharedTables(ctx context.Context, dbPath string) (bool, string) {
	required := []string{
		tableMatchRegistry,
		"match_participants",
		"medals_earned",
		"highlight_events",
		"xuid_aliases",
		"weapon_kills",
	}
	if _, err := os.Stat(dbPath); err != nil {
		return false, "shared DB absente"
	}
	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")
	if err != nil {
		return false, fmt.Sprintf("open: %v", err)
	}
	defer db.Close()

	missing := checkTablesExist(ctx, db, required)
	if len(missing) > 0 {
		return false, fmt.Sprintf("tables manquantes: %s", strings.Join(missing, ", "))
	}
	return true, fmt.Sprintf("%d tables critiques présentes", len(required))
}

func checkSharedViews(ctx context.Context, dbPath string) (bool, string) {
	requiredViews := []string{
		"v_gamertag_lookup",
		"v_match_full",
		"v_weapon_kills",
	}
	if _, err := os.Stat(dbPath); err != nil {
		return false, "shared DB absente"
	}
	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")
	if err != nil {
		return false, fmt.Sprintf("open: %v", err)
	}
	defer db.Close()

	missing := checkViewsExist(ctx, db, requiredViews)
	if len(missing) > 0 {
		return false, fmt.Sprintf("vues V6 manquantes: %s (relancer les migrations)", strings.Join(missing, ", "))
	}
	return true, "toutes les vues V6 présentes"
}

func checkMigrationsApplied(ctx context.Context, repoRoot, gamertag string) (bool, string) {
	pr := titlePkg.NewPathResolver(repoRoot)
	dbPath := pr.PlayerDBPath(titlePkg.DefaultSlug, gamertag)
	if _, err := os.Stat(dbPath); err != nil {
		return false, fmt.Sprintf("DB joueur absente: %s", dbPath)
	}
	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")
	if err != nil {
		return false, fmt.Sprintf("open: %v", err)
	}
	defer db.Close()

	var cnt int64
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&cnt); err != nil {
		return false, "table schema_migrations absente (migrations jamais exécutées)"
	}
	if cnt < 10 {
		return false, fmt.Sprintf("seulement %d migrations appliquées (minimum attendu: 10)", cnt)
	}
	return true, fmt.Sprintf("%d migrations trackées dans schema_migrations", cnt)
}

func checkPlayerDB(ctx context.Context, dbPath string) (bool, string) {
	if _, err := os.Stat(dbPath); err != nil {
		return false, fmt.Sprintf("DB joueur absente: %s", dbPath)
	}
	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")
	if err != nil {
		return false, fmt.Sprintf("open: %v", err)
	}
	defer db.Close()

	var cnt int64
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM player_match_enrichment").Scan(&cnt); err != nil {
		return false, "table player_match_enrichment absente"
	}
	if cnt == 0 {
		return false, "player_match_enrichment vide (aucun sync effectué ?)"
	}
	return true, fmt.Sprintf("player_match_enrichment: %d lignes", cnt)
}

func checkDBProfiles(profilesPath string) (bool, string) {
	if _, err := os.Stat(profilesPath); err != nil {
		return false, fmt.Sprintf("db_profiles.json absent: %s", profilesPath)
	}
	data, err := os.ReadFile(profilesPath)
	if err != nil {
		return false, fmt.Sprintf("lecture: %v", err)
	}
	content := strings.TrimSpace(string(data))
	if content == "{}" || content == "[]" || content == "" {
		return false, "db_profiles.json vide — configurer au moins un profil joueur"
	}
	return true, "profils joueur configurés"
}

func checkDiscordNotify(repoRoot string) (bool, string) {
	if url := config.DiscordWebhookURLFromEnv(); url != "" {
		if strings.HasPrefix(url, "https://discord.com/api/webhooks/") {
			return true, "webhook Discord configuré via env (LEVELUP_DISCORD_WEBHOOK_URL/DISCORD_WEBHOOK_URL)"
		}
		return false, "webhook Discord env invalide (doit commencer par https://discord.com/api/webhooks/)"
	}
	// Vérifier app_settings.json
	settingsPath := filepath.Join(repoRoot, "app_settings.json")
	if _, err := os.Stat(settingsPath); err != nil {
		return false, "DISCORD_WEBHOOK_URL non défini et app_settings.json absent"
	}
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return false, fmt.Sprintf("lecture app_settings.json: %v", err)
	}
	if strings.Contains(string(data), "discord_webhook_url") &&
		strings.Contains(string(data), "https://discord.com/api/webhooks/") {
		return true, "webhook Discord configuré dans app_settings.json"
	}
	return false, "webhook Discord non configuré (DISCORD_WEBHOOK_URL ou app_settings.json)"
}

// ─────────────────────────────────────────────────────────────────────────────
// Utilitaires DB
// ─────────────────────────────────────────────────────────────────────────────

func checkTablesExist(ctx context.Context, db *sql.DB, tables []string) []string {
	var missing []string
	for _, t := range tables {
		var cnt int
		err := db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='main' AND table_name=?", t,
		).Scan(&cnt)
		if err != nil || cnt == 0 {
			missing = append(missing, t)
		}
	}
	return missing
}

func checkViewsExist(ctx context.Context, db *sql.DB, views []string) []string {
	var missing []string
	for _, v := range views {
		var cnt int
		err := db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='main' AND table_name=? AND table_type='VIEW'", v,
		).Scan(&cnt)
		if err != nil || cnt == 0 {
			missing = append(missing, v)
		}
	}
	return missing
}
