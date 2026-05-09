// cmd_backfill.go : sous-commande CLI pour les backfills locaux.
//
// Le backfill est principalement expose via HTTP (POST /backfill/start).
// Cette sous-commande offre une voie locale pour les backfills purement Go
// (pas d'appel API Halo requis), utile pour bootstrap ou re-run en masse.
//
// Backfills supportes :
//   - --engagement-scores [--force]
//   - --citations         [--force]
//   - --lusr              [--force]  (recalcule LUSR TrueSkill 2 + poids médailles)
//   - --perf              [--force]  (recalcule performance score relatif v5)
//
// Usage :
//
//	levelup backfill --gamertag X --lusr  [--force]
//	levelup backfill --all          --perf  [--force]
//	levelup backfill --all          --lusr --perf --force
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/auth"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	go_sync "levelup/go-api/internal/sync"
)

func runBackfill(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("backfill", flag.ExitOnError)
	gamertag := fs.String("gamertag", "", "Gamertag du joueur (mutuellement exclusif avec --all)")
	allPlayers := fs.Bool("all", false, "Applique le backfill a tous les joueurs configures")
	engagementScores := fs.Bool("engagement-scores", false, "Backfill du score d'engagement (Phase 6 plan engagement)")
	citations := fs.Bool("citations", false, "Backfill des citations (match_citations) depuis citation_mappings + medals + stats + awards")
	lusr := fs.Bool("lusr", false, "Backfill LUSR TrueSkill 2 avec poids medailles v5")
	perf := fs.Bool("perf", false, "Backfill performance score relatif v5 (off_conv + def_res + medal_exploit)")
	assistsModel := fs.Bool("assists-model", false, "Calcule le modèle OLS expected_assists par mode (player_assists_model dans stats.duckdb)")
	weapons := fs.Bool("weapons", false, "Backfill weapon_kills depuis film CDN (tous les participants par match)")
	compositeOnly := fs.Bool("composite-only", false, "Backfill citations composites uniquement (additive, sans recalcul depuis shared_matches)")
	force := fs.Bool("force", false, "Force le recalcul meme si deja persiste")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *allPlayers && strings.TrimSpace(*gamertag) != "" {
		return fmt.Errorf("--gamertag et --all sont mutuellement exclusifs")
	}
	if !*allPlayers && strings.TrimSpace(*gamertag) == "" {
		return fmt.Errorf("--gamertag est obligatoire sauf avec --all")
	}
	if !*engagementScores && !*citations && !*lusr && !*perf && !*assistsModel && !*weapons && !*compositeOnly {
		return fmt.Errorf("aucun backfill selectionne (utiliser --engagement-scores, --citations, --lusr, --perf, --assists-model, --weapons ou --composite-only)")
	}

	ctx := context.Background()
	if *engagementScores {
		if *allPlayers {
			if err := runBackfillAllEngagement(ctx, cfg, *force); err != nil {
				return err
			}
		} else {
			player, err := loadPlayerSummary(cfg, *gamertag)
			if err != nil {
				return err
			}
			if err := runBackfillEngagementForPlayer(ctx, cfg, player.Gamertag, player.XUID, *force); err != nil {
				return err
			}
		}
	}
	if *citations {
		var err error
		if *allPlayers {
			err = runBackfillAllCitations(ctx, cfg, *force)
		} else {
			player, err2 := loadPlayerSummary(cfg, *gamertag)
			if err2 != nil {
				return err2
			}
			err = runBackfillCitationsForPlayer(ctx, cfg, player.Gamertag, player.XUID, *force)
		}
		if err != nil {
			return err
		}
	}
	if *lusr {
		if *allPlayers {
			if err := runBackfillAllLUSR(ctx, cfg, *force); err != nil {
				return err
			}
		} else {
			player, err := loadPlayerSummary(cfg, *gamertag)
			if err != nil {
				return err
			}
			if err := runBackfillLUSRForPlayer(ctx, cfg, player, *force); err != nil {
				return err
			}
		}
	}
	if *perf {
		if *allPlayers {
			return runBackfillAllPerf(ctx, cfg, *force)
		}
		player, err := loadPlayerSummary(cfg, *gamertag)
		if err != nil {
			return err
		}
		return runBackfillPerfForPlayer(ctx, cfg, player, *force)
	}
	if *assistsModel {
		if *allPlayers {
			return runBackfillAllAssistsModel(ctx, cfg, *force)
		}
		player, err := loadPlayerSummary(cfg, *gamertag)
		if err != nil {
			return err
		}
		return runBackfillAssistsModelForPlayer(ctx, cfg, player.Gamertag, player.XUID, *force)
	}
	if *weapons {
		return runBackfillAllWeapons(ctx, cfg, *force)
	}
	if *compositeOnly {
		if *allPlayers {
			return runBackfillAllCompositeOnly(ctx, cfg)
		}
		player, err := loadPlayerSummary(cfg, *gamertag)
		if err != nil {
			return err
		}
		return runBackfillCompositeOnlyForPlayer(ctx, cfg, player.Gamertag, player.XUID)
	}
	return nil
}

func runBackfillAllEngagement(ctx context.Context, cfg *config.AppConfig, force bool) error {
	players, err := cfg.LoadPlayers()
	if err != nil {
		return fmt.Errorf("chargement db_profiles.json: %w", err)
	}
	if len(players) == 0 {
		return fmt.Errorf("aucun joueur configure")
	}

	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	total := len(players)
	processed := 0
	skipped := 0
	failed := 0
	totalUpdated := 0

	for _, player := range players {
		dbPath := resolver.PlayerDBPath(titlePkg.DefaultSlug, player.Gamertag)
		if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
			skipped++
			fmt.Printf("backfill engagement SKIP: gamertag=%s reason=no_player_db\n", player.Gamertag)
			continue
		}

		updated, runErr := runBackfillEngagementOne(ctx, cfg, player.Gamertag, player.XUID, force)
		if runErr != nil {
			failed++
			fmt.Printf("backfill engagement FAIL: gamertag=%s err=%v\n", player.Gamertag, runErr)
			continue
		}
		processed++
		totalUpdated += updated
		fmt.Printf("backfill engagement OK: gamertag=%s updated=%d\n", player.Gamertag, updated)
	}

	fmt.Printf("backfill engagement batch: total=%d processed=%d skipped=%d failed=%d total_updated=%d\n",
		total, processed, skipped, failed, totalUpdated)
	if failed > 0 {
		return fmt.Errorf("backfill engagement: %d joueur(s) en echec", failed)
	}
	return nil
}

func runBackfillEngagementForPlayer(ctx context.Context, cfg *config.AppConfig, gamertag, xuid string, force bool) error {
	updated, err := runBackfillEngagementOne(ctx, cfg, gamertag, xuid, force)
	if err != nil {
		return err
	}
	fmt.Printf("backfill engagement OK: gamertag=%s updated=%d force=%t\n", gamertag, updated, force)
	return nil
}

// runBackfillEngagementOne instancie un SyncEngine sans tokens (calcul local
// pur) et appelle RunBackfillEngagementScores. Aucune requete API requise.
//
// Applique les migrations Phase 2 engagement (colonnes player + match_intensity
// shared) avant le backfill, car sync.OpenPlayerDB/OpenSharedDB ne lance pas
// migration.RunForDB (contrairement au pool DuckDB / boot serveur).
func runBackfillEngagementOne(ctx context.Context, cfg *config.AppConfig, gamertag, xuid string, force bool) (int, error) {
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	playerDBPath := resolver.PlayerDBPath(titlePkg.DefaultSlug, gamertag)
	sharedDBPath := resolver.SharedDBPath(titlePkg.DefaultSlug)

	if err := applyMigrationsOnDB(playerDBPath, migration.TargetPlayer); err != nil {
		return 0, fmt.Errorf("migrations player %s: %w", gamertag, err)
	}
	if err := applyMigrationsOnDB(sharedDBPath, migration.TargetShared); err != nil {
		return 0, fmt.Errorf("migrations shared: %w", err)
	}

	engine := go_sync.NewSyncEngine(cfg.RepoRoot, gamertag, xuid, nil, nil)
	return engine.RunBackfillEngagementScores(ctx, force)
}

// runBackfillAllCitations applique le backfill citations sur tous les joueurs
// configures dans db_profiles.json.
func runBackfillAllCitations(ctx context.Context, cfg *config.AppConfig, force bool) error {
	players, err := cfg.LoadPlayers()
	if err != nil {
		return fmt.Errorf("chargement db_profiles.json: %w", err)
	}
	if len(players) == 0 {
		return fmt.Errorf("aucun joueur configure")
	}

	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	processed := 0
	skipped := 0
	failed := 0
	totalUpdated := 0

	for _, player := range players {
		dbPath := resolver.PlayerDBPath(titlePkg.DefaultSlug, player.Gamertag)
		if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
			skipped++
			fmt.Printf("backfill citations SKIP: gamertag=%s reason=no_player_db\n", player.Gamertag)
			continue
		}

		updated, runErr := runBackfillCitationsOne(ctx, cfg, player.Gamertag, player.XUID, force)
		if runErr != nil {
			failed++
			fmt.Printf("backfill citations FAIL: gamertag=%s err=%v\n", player.Gamertag, runErr)
			continue
		}
		processed++
		totalUpdated += updated
		fmt.Printf("backfill citations OK: gamertag=%s updated=%d\n", player.Gamertag, updated)
	}

	fmt.Printf("backfill citations batch: total=%d processed=%d skipped=%d failed=%d total_updated=%d\n",
		len(players), processed, skipped, failed, totalUpdated)
	if failed > 0 {
		return fmt.Errorf("backfill citations: %d joueur(s) en echec", failed)
	}
	return nil
}

func runBackfillCitationsForPlayer(ctx context.Context, cfg *config.AppConfig, gamertag, xuid string, force bool) error {
	updated, err := runBackfillCitationsOne(ctx, cfg, gamertag, xuid, force)
	if err != nil {
		return err
	}
	fmt.Printf("backfill citations OK: gamertag=%s updated=%d force=%t\n", gamertag, updated, force)
	return nil
}

// runBackfillCitationsOne applique les migrations puis appelle
// SyncEngine.RunBackfillCitations. Aucun appel API requis.
func runBackfillCitationsOne(ctx context.Context, cfg *config.AppConfig, gamertag, xuid string, force bool) (int, error) {
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	playerDBPath := resolver.PlayerDBPath(titlePkg.DefaultSlug, gamertag)
	sharedDBPath := resolver.SharedDBPath(titlePkg.DefaultSlug)
	metaDBPath := resolver.MetadataDBPath(titlePkg.DefaultSlug)

	if err := applyMigrationsOnDB(playerDBPath, migration.TargetPlayer); err != nil {
		return 0, fmt.Errorf("migrations player %s: %w", gamertag, err)
	}
	if err := applyMigrationsOnDB(sharedDBPath, migration.TargetShared); err != nil {
		return 0, fmt.Errorf("migrations shared: %w", err)
	}
	if err := applyMigrationsOnDB(metaDBPath, migration.TargetMetadata); err != nil {
		return 0, fmt.Errorf("migrations metadata: %w", err)
	}

	engine := go_sync.NewSyncEngine(cfg.RepoRoot, gamertag, xuid, nil, nil)
	return engine.RunBackfillCitations(ctx, force)
}

// applyMigrationsOnDB ouvre une DB en RW et applique les migrations enregistrees
// pour la cible. Idempotent — DuckDB tolere une migration deja appliquee via
// schema_migrations.
func applyMigrationsOnDB(path string, target migration.TargetDB) error {
	_ = migration.All()
	db, err := duckdbpkg.OpenReadWrite(path)
	if err != nil {
		return fmt.Errorf("open rw %s: %w", path, err)
	}
	defer db.Close()
	return migration.RunForDB(db.SQLDb(), target)
}

// ── LUSR backfill ─────────────────────────────────────────────────────────────

func runBackfillAllLUSR(ctx context.Context, cfg *config.AppConfig, force bool) error {
	players, err := cfg.LoadPlayers()
	if err != nil {
		return fmt.Errorf("chargement db_profiles.json: %w", err)
	}
	if len(players) == 0 {
		return fmt.Errorf("aucun joueur configure")
	}
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	total, processed, skipped, failed, totalUpdated := len(players), 0, 0, 0, 0
	for _, player := range players {
		dbPath := resolver.PlayerDBPath(titlePkg.DefaultSlug, player.Gamertag)
		if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
			skipped++
			fmt.Printf("backfill lusr SKIP: gamertag=%s reason=no_player_db\n", player.Gamertag)
			continue
		}
		engine := go_sync.NewSyncEngine(cfg.RepoRoot, player.Gamertag, player.XUID, nil, nil)
		updated, runErr := engine.RunBackfillLUSR(ctx, force)
		if runErr != nil {
			failed++
			fmt.Printf("backfill lusr FAIL: gamertag=%s err=%v\n", player.Gamertag, runErr)
			continue
		}
		processed++
		totalUpdated += updated
		fmt.Printf("backfill lusr OK: gamertag=%s updated=%d\n", player.Gamertag, updated)
	}
	fmt.Printf("backfill lusr batch: total=%d processed=%d skipped=%d failed=%d total_updated=%d\n",
		total, processed, skipped, failed, totalUpdated)
	if failed > 0 {
		return fmt.Errorf("backfill lusr: %d joueur(s) en echec", failed)
	}
	return nil
}

func runBackfillLUSRForPlayer(ctx context.Context, cfg *config.AppConfig, player *domain.PlayerSummary, force bool) error {
	engine := go_sync.NewSyncEngine(cfg.RepoRoot, player.Gamertag, player.XUID, nil, nil)
	updated, err := engine.RunBackfillLUSR(ctx, force)
	if err != nil {
		return err
	}
	fmt.Printf("backfill lusr OK: gamertag=%s updated=%d force=%t\n", player.Gamertag, updated, force)
	return nil
}

// ── Performance score backfill ─────────────────────────────────────────────────

func runBackfillAllPerf(ctx context.Context, cfg *config.AppConfig, force bool) error {
	players, err := cfg.LoadPlayers()
	if err != nil {
		return fmt.Errorf("chargement db_profiles.json: %w", err)
	}
	if len(players) == 0 {
		return fmt.Errorf("aucun joueur configure")
	}
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	total, processed, skipped, failed, totalUpdated := len(players), 0, 0, 0, 0
	for _, player := range players {
		dbPath := resolver.PlayerDBPath(titlePkg.DefaultSlug, player.Gamertag)
		if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
			skipped++
			fmt.Printf("backfill perf SKIP: gamertag=%s reason=no_player_db\n", player.Gamertag)
			continue
		}
		engine := go_sync.NewSyncEngine(cfg.RepoRoot, player.Gamertag, player.XUID, nil, nil)
		updated, runErr := engine.RunBackfillPerf(ctx, force)
		if runErr != nil {
			failed++
			fmt.Printf("backfill perf FAIL: gamertag=%s err=%v\n", player.Gamertag, runErr)
			continue
		}
		processed++
		totalUpdated += updated
		fmt.Printf("backfill perf OK: gamertag=%s updated=%d\n", player.Gamertag, updated)
	}
	fmt.Printf("backfill perf batch: total=%d processed=%d skipped=%d failed=%d total_updated=%d\n",
		total, processed, skipped, failed, totalUpdated)
	if failed > 0 {
		return fmt.Errorf("backfill perf: %d joueur(s) en echec", failed)
	}
	return nil
}

func runBackfillPerfForPlayer(ctx context.Context, cfg *config.AppConfig, player *domain.PlayerSummary, force bool) error {
	engine := go_sync.NewSyncEngine(cfg.RepoRoot, player.Gamertag, player.XUID, nil, nil)
	updated, err := engine.RunBackfillPerf(ctx, force)
	if err != nil {
		return err
	}
	fmt.Printf("backfill perf OK: gamertag=%s updated=%d force=%t\n", player.Gamertag, updated, force)
	return nil
}

// ── Assists model backfill ─────────────────────────────────────────────────────

func runBackfillAllAssistsModel(ctx context.Context, cfg *config.AppConfig, force bool) error {
	players, err := cfg.LoadPlayers()
	if err != nil {
		return fmt.Errorf("chargement db_profiles.json: %w", err)
	}
	if len(players) == 0 {
		return fmt.Errorf("aucun joueur configure")
	}
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	total, processed, skipped, failed, totalUpdated := len(players), 0, 0, 0, 0
	for _, player := range players {
		dbPath := resolver.PlayerDBPath(titlePkg.DefaultSlug, player.Gamertag)
		if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
			skipped++
			fmt.Printf("backfill assists-model SKIP: gamertag=%s reason=no_player_db\n", player.Gamertag)
			continue
		}
		updated, runErr := runBackfillAssistsModelOne(ctx, cfg, player.Gamertag, player.XUID, force)
		if runErr != nil {
			failed++
			fmt.Printf("backfill assists-model FAIL: gamertag=%s err=%v\n", player.Gamertag, runErr)
			continue
		}
		processed++
		totalUpdated += updated
		fmt.Printf("backfill assists-model OK: gamertag=%s n_modes=%d\n", player.Gamertag, updated)
	}
	fmt.Printf("backfill assists-model batch: total=%d processed=%d skipped=%d failed=%d total_modes=%d\n",
		total, processed, skipped, failed, totalUpdated)
	if failed > 0 {
		return fmt.Errorf("backfill assists-model: %d joueur(s) en echec", failed)
	}
	return nil
}

func runBackfillAssistsModelForPlayer(ctx context.Context, cfg *config.AppConfig, gamertag, xuid string, force bool) error {
	n, err := runBackfillAssistsModelOne(ctx, cfg, gamertag, xuid, force)
	if err != nil {
		return err
	}
	fmt.Printf("backfill assists-model OK: gamertag=%s n_modes=%d force=%t\n", gamertag, n, force)
	return nil
}

func runBackfillAssistsModelOne(ctx context.Context, cfg *config.AppConfig, gamertag, xuid string, force bool) (int, error) {
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	playerDBPath := resolver.PlayerDBPath(titlePkg.DefaultSlug, gamertag)

	if err := applyMigrationsOnDB(playerDBPath, migration.TargetPlayer); err != nil {
		return 0, fmt.Errorf("migrations player %s: %w", gamertag, err)
	}

	engine := go_sync.NewSyncEngine(cfg.RepoRoot, gamertag, xuid, nil, nil)
	return engine.RunBackfillAssistsModel(ctx, force)
}

// ── Weapon kills backfill (film parsing) ───────────────────────────────────

func runBackfillAllWeapons(ctx context.Context, cfg *config.AppConfig, force bool) error {
	players, err := cfg.LoadPlayers()
	if err != nil {
		return fmt.Errorf("LoadPlayers: %w", err)
	}
	if len(players) == 0 {
		return fmt.Errorf("aucun joueur configure")
	}

	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	total := len(players)
	processed := 0
	skipped := 0
	failed := 0

	for _, player := range players {
		dbPath := resolver.PlayerDBPath(titlePkg.DefaultSlug, player.Gamertag)
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			fmt.Printf("backfill weapons SKIP: gamertag=%s reason=no_player_db\n", player.Gamertag)
			skipped++
			continue
		}

		// Load Halo API tokens via OAuth refresh token (same pattern as cmd_sync.go)
		refreshToken := oauthRefreshTokenForPlayer(player.Gamertag)
		if refreshToken == "" {
			fmt.Printf("backfill weapons SKIP: gamertag=%s reason=no_refresh_token (%s)\n",
				player.Gamertag, oauthRefreshEnvKey(player.Gamertag))
			skipped++
			continue
		}

		provider := auth.NewMSALProvider()
		accessToken, err := provider.TryOAuthRefresh(ctx, refreshToken)
		if err != nil || accessToken == "" {
			fmt.Printf("backfill weapons SKIP: gamertag=%s reason=oauth_refresh_failed err=%v\n", player.Gamertag, err)
			skipped++
			continue
		}

		result, err := provider.Exchange(ctx, accessToken)
		if err != nil {
			fmt.Printf("backfill weapons SKIP: gamertag=%s reason=exchange_failed err=%v\n", player.Gamertag, err)
			skipped++
			continue
		}
		tokens := result.Tokens

		engine := go_sync.NewSyncEngine(cfg.RepoRoot, player.Gamertag, player.XUID, tokens, nil)

		sharedDBPath := resolver.SharedDBPath(titlePkg.DefaultSlug)
		matchIDs, err := findMissingWeaponMatches(ctx, sharedDBPath, player.XUID, force)
		if err != nil {
			fmt.Printf("backfill weapons FAIL: gamertag=%s err=%v\n", player.Gamertag, err)
			failed++
			continue
		}

		if len(matchIDs) == 0 {
			fmt.Printf("backfill weapons OK: gamertag=%s matches=0\n", player.Gamertag)
			processed++
			continue
		}

		fmt.Printf("backfill weapons: gamertag=%s matches=%d\n", player.Gamertag, len(matchIDs))
		done, noFilm, err := engine.BackfillWeaponKillsForMatches(ctx, matchIDs)
		if err != nil {
			fmt.Printf("backfill weapons FAIL: gamertag=%s err=%v\n", player.Gamertag, err)
			failed++
			continue
		}

		fmt.Printf("backfill weapons OK: gamertag=%s done=%d nofilm=%d\n", player.Gamertag, done, noFilm)
		processed++
	}

	fmt.Printf("backfill weapons batch: total=%d processed=%d skipped=%d failed=%d\n", total, processed, skipped, failed)
	if failed > 0 {
		return fmt.Errorf("backfill weapons: %d joueur(s) en echec", failed)
	}
	return nil
}

func findMissingWeaponMatches(ctx context.Context, sharedDBPath, xuid string, force bool) ([]string, error) {
	db, err := duckdbpkg.OpenReadOnly(sharedDBPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	const mBitWeaponKills = 1 << 21
	const mBitWeaponKillsNoFilm = 1 << 22

	var query string
	var args []any
	if force {
		// Force: retourne tous les matchs non-firefight, indépendamment du bitmask.
		query = `
			SELECT DISTINCT mp.match_id
			FROM match_participants mp
			JOIN match_registry mr ON mr.match_id = mp.match_id
			WHERE mp.xuid = ?
			  AND COALESCE(mr.is_firefight, FALSE) = FALSE
			ORDER BY mr.start_time DESC
			LIMIT 30
		`
		args = []any{xuid}
	} else {
		query = `
			SELECT DISTINCT mp.match_id
			FROM match_participants mp
			JOIN match_registry mr ON mr.match_id = mp.match_id
			WHERE mp.xuid = ?
			  AND (COALESCE(mr.backfill_completed, 0) & ?) = 0
			  AND (COALESCE(mr.backfill_completed, 0) & ?) = 0
			  AND COALESCE(mr.is_firefight, FALSE) = FALSE
			ORDER BY mr.start_time DESC
			LIMIT 30
		`
		args = []any{xuid, mBitWeaponKills, mBitWeaponKillsNoFilm}
	}

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matchIDs []string
	for rows.Next() {
		var mid string
		if err := rows.Scan(&mid); err != nil {
			continue
		}
		matchIDs = append(matchIDs, mid)
	}
	return matchIDs, rows.Err()
}

// ── Composite-only citations backfill ──────────────────────────────────────────

func runBackfillAllCompositeOnly(ctx context.Context, cfg *config.AppConfig) error {
	players, err := cfg.LoadPlayers()
	if err != nil {
		return fmt.Errorf("chargement db_profiles.json: %w", err)
	}
	if len(players) == 0 {
		return fmt.Errorf("aucun joueur configure")
	}

	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	total, processed, skipped, failed, totalUpdated := len(players), 0, 0, 0, 0
	for _, player := range players {
		dbPath := resolver.PlayerDBPath(titlePkg.DefaultSlug, player.Gamertag)
		if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
			skipped++
			fmt.Printf("backfill composite-only SKIP: gamertag=%s reason=no_player_db\n", player.Gamertag)
			continue
		}
		updated, runErr := runBackfillCompositeOnlyOne(ctx, cfg, player.Gamertag, player.XUID)
		if runErr != nil {
			failed++
			fmt.Printf("backfill composite-only FAIL: gamertag=%s err=%v\n", player.Gamertag, runErr)
			continue
		}
		processed++
		totalUpdated += updated
		fmt.Printf("backfill composite-only OK: gamertag=%s updated=%d\n", player.Gamertag, updated)
	}
	fmt.Printf("backfill composite-only batch: total=%d processed=%d skipped=%d failed=%d total_updated=%d\n",
		total, processed, skipped, failed, totalUpdated)
	if failed > 0 {
		return fmt.Errorf("backfill composite-only: %d joueur(s) en echec", failed)
	}
	return nil
}

func runBackfillCompositeOnlyForPlayer(ctx context.Context, cfg *config.AppConfig, gamertag, xuid string) error {
	updated, err := runBackfillCompositeOnlyOne(ctx, cfg, gamertag, xuid)
	if err != nil {
		return err
	}
	fmt.Printf("backfill composite-only OK: gamertag=%s updated=%d\n", gamertag, updated)
	return nil
}

// runBackfillCompositeOnlyOne applique les migrations puis appelle
// SyncEngine.RunBackfillCompositeOnlyCitations. Aucun appel API requis.
func runBackfillCompositeOnlyOne(ctx context.Context, cfg *config.AppConfig, gamertag, xuid string) (int, error) {
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	playerDBPath := resolver.PlayerDBPath(titlePkg.DefaultSlug, gamertag)
	metaDBPath := resolver.MetadataDBPath(titlePkg.DefaultSlug)

	if err := applyMigrationsOnDB(playerDBPath, migration.TargetPlayer); err != nil {
		return 0, fmt.Errorf("migrations player %s: %w", gamertag, err)
	}
	if err := applyMigrationsOnDB(metaDBPath, migration.TargetMetadata); err != nil {
		return 0, fmt.Errorf("migrations metadata: %w", err)
	}

	engine := go_sync.NewSyncEngine(cfg.RepoRoot, gamertag, xuid, nil, nil)
	return engine.RunBackfillCompositeOnlyCitations(ctx)
}
