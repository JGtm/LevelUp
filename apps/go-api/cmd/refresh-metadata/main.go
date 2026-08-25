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
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/metadata"
	"levelup/go-api/internal/notify"
	authpkg "levelup/go-api/internal/platform/auth"
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
	case "medals":
		err = runMedals(cfg, args)
	case "assets":
		err = runAssets(cfg, args)
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

	metaDB, err := openMetadataDB(cfg, *titleID)
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

	metaDB, err := openMetadataDB(cfg, *titleID)
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
	if err := runMedals(cfg, args); err != nil {
		return fmt.Errorf("medals: %w", err)
	}
	return runStaging(cfg, args)
}

// ─── medals ──────────────────────────────────────────────────────────────────

// runMedals fetch les métadonnées médailles Waypoint, applique les garde-fous
// et upserte dans waypoint_medals_raw (staging uniquement, pas de promotion auto).
// Avec --promote : copie difficulty + medal_type vers medal_definitions après l'upsert.
func runMedals(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("medals", flag.ExitOnError)
	titleID := fs.String("title-id", "halo_infinite", "Title ID")
	force := fs.Bool("force", false, "Ignorer le garde-fou de cardinalité")
	promote := fs.Bool("promote", false, "Promouvoir difficulty+medal_type vers medal_definitions après l'upsert")
	player := fs.String("player", "", "Gamertag pour la résolution des tokens OAuth (si SPARTAN_TOKEN absent)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	metaDB, err := openMetadataDB(cfg, *titleID)
	if err != nil {
		return err
	}

	repo := duckdb.NewMetadataRepoFromDB(metaDB)
	ctx := context.Background()

	tokens, err := resolveTokens(ctx, cfg, *player)
	if err != nil {
		return fmt.Errorf("résolution tokens: %w", err)
	}
	ctx = ctxkeys.WithHaloAuth(ctx, tokens, "")

	if err := repo.EnsureStagingTables(ctx); err != nil {
		return fmt.Errorf("EnsureStagingTables: %w", err)
	}

	provider := halo.DefaultHaloProvider
	entries, raw, err := provider.FetchMedalsMetadata(ctx, *titleID)
	if err != nil {
		return fmt.Errorf("FetchMedalsMetadata: %w", err)
	}

	newHash := halo.ContentHash(raw)

	// Garde-fous (D1.3).
	if !*force {
		localCount, _ := repo.CountMedalsRaw(ctx, *titleID)
		result := metadata.RunAllGuards(entries, localCount)
		if !result.Passed {
			fmt.Fprintf(os.Stderr, "❌ Garde-fou échoué : %s\n", result.Reason)
			for _, d := range result.Details {
				fmt.Fprintf(os.Stderr, "   %s\n", d)
			}
			return fmt.Errorf("import bloqué par garde-fous")
		}
		fmt.Printf("✓ Garde-fous OK : %s\n", result.Reason)
	}

	if err := repo.UpsertMedalsRaw(ctx, entries, newHash); err != nil {
		return fmt.Errorf("UpsertMedalsRaw: %w", err)
	}

	fmt.Printf("✅ %d médailles upsertées dans waypoint_medals_raw (hash %s)\n", len(entries), newHash[:8])

	if *promote {
		n, err := repo.PromoteMedalDifficultyType(ctx, *titleID)
		if err != nil {
			return fmt.Errorf("PromoteMedalDifficultyType: %w", err)
		}
		fmt.Printf("✅ %d lignes medal_definitions enrichies (difficulty + medal_type)\n", n)
	} else {
		fmt.Println("ℹ️  Promotion vers medal_definitions : relancer avec --promote pour enrichir difficulty+medal_type")
	}
	return nil
}

// ─── assets ──────────────────────────────────────────────────────────────────

// runAssets génère un rapport diff des assets Waypoint vs DB locale (D2.3).
// Pas d'écriture automatique en production sans validation humaine.
func runAssets(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("assets", flag.ExitOnError)
	titleID := fs.String("title-id", "halo_infinite", "Title ID")
	write := fs.Bool("write", false, "Écrire les assets nouveaux/modifiés en DB (désactivé par défaut)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	metaDB, err := openMetadataDB(cfg, *titleID)
	if err != nil {
		return err
	}

	repo := duckdb.NewMetadataRepoFromDB(metaDB)
	ctx := context.Background()

	if err := repo.EnsureStagingTables(ctx); err != nil {
		return fmt.Errorf("EnsureStagingTables: %w", err)
	}

	// Charger la liste locale.
	local, err := repo.ListAssets(ctx, *titleID)
	if err != nil {
		return fmt.Errorf("ListAssets: %w", err)
	}

	// Pas de fetch Waypoint réel ici — placeholder pour l'intégration future.
	// Le fetch Waypoint sera ajouté quand l'endpoint Waypoint assets sera documenté.
	incoming := local // diff vide = 0 nouveaux, 0 modifiés, len(local) inchangés

	report := duckdb.ComputeAssetDiff(local, incoming)

	fmt.Printf("📊 Rapport diff assets (title_id=%s)\n", *titleID)
	fmt.Printf("   Nouveaux   : %d\n", len(report.New))
	fmt.Printf("   Modifiés   : %d\n", len(report.Modified))
	fmt.Printf("   Inchangés  : %d\n", report.Unchanged)

	if !*write {
		fmt.Println("ℹ️  Mode lecture seule — relancer avec --write pour persister les changements")
		return nil
	}

	for _, e := range append(report.New, report.Modified...) {
		if err := repo.UpsertAsset(ctx, e); err != nil {
			return fmt.Errorf("UpsertAsset: %w", err)
		}
	}
	fmt.Printf("✅ %d assets écrits en DB\n", len(report.New)+len(report.Modified))
	return nil
}

// ─── staging ─────────────────────────────────────────────────────────────────

// runStaging crée les tables de staging waypoint_medals_raw et waypoint_assets_raw.
// Ne fait pas de fetch Waypoint : crée juste le schéma (prêt pour un futur backfill).
func runStaging(cfg *config.AppConfig, _ []string) error {
	metaDB, err := openMetadataDB(cfg, titlePkg.DefaultSlug)
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

// openMetadataDB ouvre la metadata.duckdb DU TITRE `slug` (MT-16 / day-one 2e
// titre : le flag --title-id était threadé au fetch Waypoint mais IGNORÉ pour le
// chemin DB → écriture dans la metadata de Halo. Corrigé : chemin résolu pour slug.
// Les commandes passent leur *titleID ; runStaging (sans flag) passe DefaultSlug.
func openMetadataDB(cfg *config.AppConfig, slug string) (*duckdb.DB, error) {
	if cfg.RepoRoot == "" {
		return nil, fmt.Errorf("LEVELUP_REPO_ROOT non défini — ne peut pas localiser metadata.duckdb")
	}
	metaPath := titlePkg.NewPathResolver(cfg.RepoRoot).MetadataDBPath(slug)
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

// resolveTokens obtient les tokens Halo depuis SPARTAN_TOKEN (env)
// ou en rejouant la chaîne OAuth depuis la DB du joueur (--player).
func resolveTokens(ctx context.Context, cfg *config.AppConfig, playerSlug string) (*domain.HaloTokens, error) {
	if envToken := os.Getenv("SPARTAN_TOKEN"); envToken != "" {
		return &domain.HaloTokens{
			SpartanToken:   envToken,
			ClearanceToken: os.Getenv("CLEARANCE_TOKEN"),
		}, nil
	}
	if playerSlug == "" {
		return nil, fmt.Errorf("SPARTAN_TOKEN absent ET --player non fourni")
	}

	provider := authpkg.NewSISUProvider()

	pdb, err := config.ResolvePlayer(ctx, cfg, playerSlug, titlePkg.DefaultSlug)
	if err != nil {
		return nil, fmt.Errorf("résoudre player %q: %w", playerSlug, err)
	}

	// ADR 0023 — pipeline canonique via MultiUserTokenStore (source unique).
	store := authpkg.NewMultiUserTokenStore(titlePkg.NewPathResolver(cfg.RepoRoot).WatcherTokensDir())

	result, err := authpkg.RefreshHaloTokensViaStoreFirst(ctx, store, provider, pdb.XUID, pdb.Gamertag)
	if err != nil {
		return nil, err
	}
	if tokens := authpkg.HaloTokensFromExchange(result); tokens != nil {
		return tokens, nil
	}
	return nil, fmt.Errorf("aucun token disponible pour player %q (aucun refresh token dans le store watcher_tokens)", playerSlug)
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
  medals      [--title-id halo_infinite] [--force]   Médailles (staging + garde-fous)
  assets      [--title-id halo_infinite] [--write]   Assets diff (rapport sans écriture par défaut)
  staging     Crée les tables staging medals/assets (schéma seulement)
  all         [--title-id halo_infinite] [--force]   Toutes les opérations ci-dessus

Variables d'environnement :
  SPARTAN_TOKEN       Token Spartan (fallback si cache MSAL absent)
  LEVELUP_REPO_ROOT   Racine du repo LevelUp (auto-détecté si absent)`)
}
