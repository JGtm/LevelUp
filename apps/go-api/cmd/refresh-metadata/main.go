// cmd/refresh-metadata — CLI pour rafraîchir les métadonnées Waypoint.
//
// Sous-commandes :
//
//	refresh-metadata seasons     [--title-id halo_infinite] [--force]
//	refresh-metadata csr-seasons [--title-id halo_infinite] [--force]
//	refresh-metadata all         [--title-id halo_infinite] [--force]
//
// Variables d'environnement requises :
//
//	SPARTAN_TOKEN (ou lues depuis le cache MSAL via token exchange)
//
// Sprint 54 A.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/notify"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/halo"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config.Load: %v\n", err)
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "seasons":
		err = runSeasons(cfg, args, false)
	case "csr-seasons":
		err = runCSRSeasons(cfg, args, false)
	case "staging":
		err = runStaging(cfg, args)
	case "all":
		err = runAll(cfg, args)
	case "help", "--help", "-h":
		printUsage()
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "sous-commande inconnue : %q\n", cmd)
		printUsage()
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "erreur : %v\n", err)
		os.Exit(1)
	}
}

// ─── seasons ─────────────────────────────────────────────────────────────────

func runSeasons(cfg *config.AppConfig, args []string, _ bool) error {
	fs := flag.NewFlagSet("seasons", flag.ExitOnError)
	titleID := fs.String("title-id", "halo_infinite", "Title ID (ex: halo_infinite)")
	force := fs.Bool("force", false, "Forcer l'upsert même si content_hash identique")
	if err := fs.Parse(args); err != nil {
		return err
	}

	metaDB, err := openMetadataDB(cfg)
	if err != nil {
		return err
	}

	repo := duckdb.NewMetadataRepoFromDB(metaDB)
	ctx := context.Background()
	if err := repo.EnsureSeasonTables(ctx); err != nil {
		return fmt.Errorf("EnsureSeasonTables: %w", err)
	}

	provider := halo.DefaultHaloProvider
	seasons, raw, err := provider.FetchSeasonCalendar(ctx, *titleID)
	if err != nil {
		return fmt.Errorf("FetchSeasonCalendar: %w", err)
	}

	// Vérification ETag / content_hash pour éviter les upserts inutiles.
	newHash := halo.ContentHash(raw)
	if !*force {
		snap, _ := repo.GetSnapshot(ctx, *titleID, "season_calendar")
		if snap != nil && snap.ContentHash == newHash {
			fmt.Printf("✓ Saisons inchangées (hash %s) — skip\n", newHash[:8])
			return nil
		}
	}

	for _, s := range seasons {
		if err := repo.UpsertSeason(ctx, s); err != nil {
			return fmt.Errorf("UpsertSeason: %w", err)
		}
	}

	// Archiver le snapshot.
	_ = repo.UpsertSnapshot(ctx, buildSnapshot(*titleID, "season_calendar", newHash, "", raw))

	fmt.Printf("✅ %d saisons upsertées (hash %s)\n", len(seasons), newHash[:8])
	notifyHashChange(cfg, "season_calendar", newHash)
	return nil
}

// ─── csr-seasons ─────────────────────────────────────────────────────────────

func runCSRSeasons(cfg *config.AppConfig, args []string, _ bool) error {
	fs := flag.NewFlagSet("csr-seasons", flag.ExitOnError)
	titleID := fs.String("title-id", "halo_infinite", "Title ID")
	force := fs.Bool("force", false, "Forcer l'upsert même si content_hash identique")
	if err := fs.Parse(args); err != nil {
		return err
	}

	metaDB, err := openMetadataDB(cfg)
	if err != nil {
		return err
	}

	repo := duckdb.NewMetadataRepoFromDB(metaDB)
	ctx := context.Background()
	if err := repo.EnsureSeasonTables(ctx); err != nil {
		return fmt.Errorf("EnsureSeasonTables: %w", err)
	}

	provider := halo.DefaultHaloProvider
	seasons, raw, err := provider.FetchCSRSeasonCalendar(ctx, *titleID)
	if err != nil {
		return fmt.Errorf("FetchCSRSeasonCalendar: %w", err)
	}

	newHash := halo.ContentHash(raw)
	if !*force {
		snap, _ := repo.GetSnapshot(ctx, *titleID, "csr_season_calendar")
		if snap != nil && snap.ContentHash == newHash {
			fmt.Printf("✓ Saisons CSR inchangées (hash %s) — skip\n", newHash[:8])
			return nil
		}
	}

	for _, s := range seasons {
		if err := repo.UpsertCSRSeason(ctx, s); err != nil {
			return fmt.Errorf("UpsertCSRSeason: %w", err)
		}
	}

	_ = repo.UpsertSnapshot(ctx, buildSnapshot(*titleID, "csr_season_calendar", newHash, "", raw))

	fmt.Printf("✅ %d saisons CSR upsertées (hash %s)\n", len(seasons), newHash[:8])
	notifyHashChange(cfg, "csr_season_calendar", newHash)
	return nil
}

// ─── all ─────────────────────────────────────────────────────────────────────

func runAll(cfg *config.AppConfig, args []string) error {
	if err := runSeasons(cfg, args, false); err != nil {
		return fmt.Errorf("seasons: %w", err)
	}
	if err := runCSRSeasons(cfg, args, false); err != nil {
		return fmt.Errorf("csr-seasons: %w", err)
	}
	return runStaging(cfg, args)
}

// ─── staging ─────────────────────────────────────────────────────────────────

// runStaging crée les tables de staging waypoint_medals_raw et waypoint_assets_raw.
// Ne fait pas de fetch Waypoint : crée juste le schéma (prêt pour un futur backfill).
func runStaging(cfg *config.AppConfig, _ []string) error {
	metaDB, err := openMetadataDB(cfg)
	if err != nil {
		return err
	}
	repo := duckdb.NewMetadataRepoFromDB(metaDB)
	ctx := context.Background()
	if err := repo.EnsureStagingTables(ctx); err != nil {
		return fmt.Errorf("EnsureStagingTables: %w", err)
	}
	fmt.Println("✅ Tables staging waypoint_medals_raw + waypoint_assets_raw prêtes")
	return nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func openMetadataDB(cfg *config.AppConfig) (*duckdb.DB, error) {
	metaPath := filepath.Join(cfg.RepoRoot, "data", "warehouse", "metadata.duckdb")
	if cfg.RepoRoot == "" {
		return nil, fmt.Errorf("LEVELUP_REPO_ROOT non défini — ne peut pas localiser metadata.duckdb")
	}
	return duckdb.OpenReadWrite(metaPath)
}

func buildSnapshot(titleID, key, hash, etag string, payload []byte) domain.WaypointResourceSnapshot {
	return domain.WaypointResourceSnapshot{
		TitleID:     titleID,
		ResourceKey: key,
		Version:     time.Now().UTC().Format("20060102T150405"),
		FetchedAt:   time.Now().UTC(),
		ContentHash: hash,
		ETag:        etag,
		Payload:     string(payload),
	}
}

func notifyHashChange(cfg *config.AppConfig, resource, hash string) {
	notifyCfg := notify.LoadNotifyConfig(cfg.AppSettingsPath)
	if notifyCfg.WebhookURL == "" {
		return
	}
	embed := notify.Embed{
		Title:       "🔄 Métadonnées Waypoint mises à jour",
		Description: fmt.Sprintf("Ressource **%s** — nouveau hash : `%s`", resource, hash[:8]),
		Color:       0x5865F2, // Discord blurple
	}
	notify.SendWebhook(notifyCfg.WebhookURL, notify.WebhookPayload{Embeds: []notify.Embed{embed}})
}

func printUsage() {
	fmt.Println(`refresh-metadata — Rafraîchit les métadonnées Waypoint dans metadata.duckdb

Sous-commandes :
  seasons     [--title-id halo_infinite] [--force]   Saisons standards
  csr-seasons [--title-id halo_infinite] [--force]   Saisons CSR
  staging     Crée les tables staging medals/assets (schéma seulement)
  all         [--title-id halo_infinite] [--force]   Toutes les opérations ci-dessus

Variables d'environnement :
  SPARTAN_TOKEN       Token Spartan (fallback si cache MSAL absent)
  LEVELUP_REPO_ROOT   Racine du repo LevelUp (auto-détecté si absent)`)
}
