// cmd_backfill.go : sous-commande CLI pour les backfills locaux.
//
// Le backfill est principalement expose via HTTP (POST /backfill/start).
// Cette sous-commande offre une voie locale pour les backfills purement Go
// (pas d'appel API Halo requis), utile pour bootstrap ou re-run en masse.
//
// Backfills supportes :
//   - --engagement-scores          [--force]
//   - --citations                  [--force]
//   - --citations-recompute-all              (recompute total + checks V1-V4)
//   - --lusr                       [--force]  (recalcule LUSR TrueSkill 2 + poids médailles)
//   - --perf                       [--force]  (recalcule performance score relatif v5)
//
// Usage :
//
//	levelup backfill --gamertag X --lusr  [--force]
//	levelup backfill --all          --perf  [--force]
//	levelup backfill --all          --lusr --perf --force
//	levelup backfill --gamertag X --citations-recompute-all
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
	engagementTitle := fs.String("title", titlePkg.DefaultSlug, "Slug du titre pour --engagement-scores (halo_infinite | halo_5)")
	citations := fs.Bool("citations", false, "Backfill des citations (match_citations) depuis citation_mappings + medals + stats + awards")
	lusr := fs.Bool("lusr", false, "Backfill LUSR TrueSkill 2 avec poids medailles v5")
	csr := fs.Bool("csr", false, "Backfill CSR par-match via GetMatchSkill (RankRecap). Idempotent ; --force re-fetche tous les matchs ranked")
	sharedCSR := fs.Bool("shared-csr", false, "Backfill shared.match_csrs (CSR de TOUS les participants des matchs ranked). --dry-run pour compter sans écrire.")
	perf := fs.Bool("perf", false, "Backfill performance score relatif v5 (off_conv + def_res + medal_exploit)")
	assistsModel := fs.Bool("assists-model", false, "Calcule le modèle OLS expected_assists par mode (player_assists_model dans stats.duckdb)")
	weapons := fs.Bool("weapons", false, "Backfill weapon_kills depuis film CDN (tous les participants par match)")
	compositeOnly := fs.Bool("composite-only", false, "Backfill citations composites uniquement (additive, sans recalcul depuis shared_matches)")
	citationsRecomputeAll := fs.Bool("citations-recompute-all", false, "Recompute total des citations (force=true) + vérifications invariants V1-V4")
	force := fs.Bool("force", false, "Force le recalcul meme si deja persiste")
	dryRun := fs.Bool("dry-run", false, "Mode dry-run (--shared-csr ou --lusr) : --shared-csr compte les matchs sans appel API ni écriture ; --lusr compute LUSR sans écrire et diff vs état persisté par playlist_group")
	compareFormulas := fs.Bool("compare-formulas", false, "Simulation des 5 variantes de formule LUSR (baseline/piste-A/B/C/A+C) sur --last-n matchs")
	lastN := fs.Int("last-n", 20, "Nombre de derniers matchs pour --compare-formulas (0 = tous)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *allPlayers && strings.TrimSpace(*gamertag) != "" {
		return fmt.Errorf("--gamertag et --all sont mutuellement exclusifs")
	}
	if !*allPlayers && strings.TrimSpace(*gamertag) == "" {
		return fmt.Errorf("--gamertag est obligatoire sauf avec --all")
	}
	if !*engagementScores && !*citations && !*citationsRecomputeAll && !*lusr && !*csr && !*sharedCSR && !*perf && !*assistsModel && !*weapons && !*compositeOnly && !*compareFormulas {
		return fmt.Errorf("aucun backfill selectionne (utiliser --engagement-scores, --citations, --citations-recompute-all, --lusr, --csr, --shared-csr, --perf, --assists-model, --weapons, --composite-only ou --compare-formulas)")
	}
	if *dryRun && !*sharedCSR && !*lusr {
		return fmt.Errorf("--dry-run n'est supporté qu'avec --shared-csr ou --lusr")
	}

	ctx := context.Background()
	if *engagementScores {
		if *allPlayers {
			if err := runBackfillAllEngagement(ctx, cfg, *engagementTitle, *force); err != nil {
				return err
			}
		} else {
			player, err := loadPlayerSummary(cfg, *gamertag)
			if err != nil {
				return err
			}
			if err := runBackfillEngagementForPlayer(ctx, cfg, *engagementTitle, player.Gamertag, player.XUID, *force); err != nil {
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
		if *dryRun {
			if *allPlayers {
				if err := runBackfillAllLUSRDryRun(ctx, cfg); err != nil {
					return err
				}
			} else {
				player, err := loadPlayerSummary(cfg, *gamertag)
				if err != nil {
					return err
				}
				if err := runBackfillLUSRDryRunForPlayer(ctx, cfg, player); err != nil {
					return err
				}
			}
		} else if *allPlayers {
			if err := runBackfillAllLUSR(ctx, cfg); err != nil {
				return err
			}
		} else {
			player, err := loadPlayerSummary(cfg, *gamertag)
			if err != nil {
				return err
			}
			if err := runBackfillLUSRForPlayer(ctx, cfg, player); err != nil {
				return err
			}
		}
	}
	if *csr {
		if *allPlayers {
			if err := runBackfillAllCSR(ctx, cfg, *force); err != nil {
				return err
			}
		} else {
			player, err := loadPlayerSummary(cfg, *gamertag)
			if err != nil {
				return err
			}
			if err := runBackfillCSRForPlayer(ctx, cfg, player, *force); err != nil {
				return err
			}
		}
	}
	if *sharedCSR {
		if *allPlayers {
			if err := runBackfillAllSharedCSR(ctx, cfg, *force, *dryRun); err != nil {
				return err
			}
		} else {
			player, err := loadPlayerSummary(cfg, *gamertag)
			if err != nil {
				return err
			}
			if err := runBackfillSharedCSRForPlayer(ctx, cfg, player, *force, *dryRun); err != nil {
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
	if *citationsRecomputeAll {
		if *allPlayers {
			return runRecomputeAllCitationsAll(ctx, cfg)
		}
		player, err := loadPlayerSummary(cfg, *gamertag)
		if err != nil {
			return err
		}
		return runRecomputeAllCitationsForPlayer(ctx, cfg, player.Gamertag, player.XUID)
	}
	if *compareFormulas {
		if *allPlayers {
			if err := runFormulaSimAll(ctx, cfg, *lastN); err != nil {
				return err
			}
		} else {
			player, err := loadPlayerSummary(cfg, *gamertag)
			if err != nil {
				return err
			}
			if err := runFormulaSimForPlayer(ctx, cfg, player, *lastN); err != nil {
				return err
			}
		}
	}
	return nil
}

func runBackfillAllEngagement(ctx context.Context, cfg *config.AppConfig, titleSlug string, force bool) error {
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
		dbPath := resolver.PlayerDBPath(titleSlug, player.Gamertag)
		if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
			skipped++
			fmt.Printf("backfill engagement SKIP: title=%s gamertag=%s reason=no_player_db\n", titleSlug, player.Gamertag)
			continue
		}

		updated, runErr := runBackfillEngagementOne(ctx, cfg, titleSlug, player.Gamertag, player.XUID, force)
		if runErr != nil {
			failed++
			fmt.Printf("backfill engagement FAIL: title=%s gamertag=%s err=%v\n", titleSlug, player.Gamertag, runErr)
			continue
		}
		processed++
		totalUpdated += updated
		fmt.Printf("backfill engagement OK: title=%s gamertag=%s updated=%d\n", titleSlug, player.Gamertag, updated)
	}

	fmt.Printf("backfill engagement batch: title=%s total=%d processed=%d skipped=%d failed=%d total_updated=%d\n",
		titleSlug, total, processed, skipped, failed, totalUpdated)
	if failed > 0 {
		return fmt.Errorf("backfill engagement: %d joueur(s) en echec", failed)
	}
	return nil
}

func runBackfillEngagementForPlayer(ctx context.Context, cfg *config.AppConfig, titleSlug, gamertag, xuid string, force bool) error {
	updated, err := runBackfillEngagementOne(ctx, cfg, titleSlug, gamertag, xuid, force)
	if err != nil {
		return err
	}
	fmt.Printf("backfill engagement OK: title=%s gamertag=%s updated=%d force=%t\n", titleSlug, gamertag, updated, force)
	return nil
}

// runBackfillEngagementOne instancie un SyncEngine sans tokens (calcul local
// pur) et appelle RunBackfillEngagementScores. Aucune requete API requise.
//
// Applique les migrations Phase 2 engagement (colonnes player + match_intensity
// shared) avant le backfill, car sync.OpenPlayerDB/OpenSharedDB ne lance pas
// migration.RunForDB (contrairement au pool DuckDB / boot serveur).
func runBackfillEngagementOne(ctx context.Context, cfg *config.AppConfig, titleSlug, gamertag, xuid string, force bool) (int, error) {
	if titleSlug == "" {
		titleSlug = titlePkg.DefaultSlug
	}
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	playerDBPath := resolver.PlayerDBPath(titleSlug, gamertag)
	sharedDBPath := resolver.SharedDBPath(titleSlug)

	if err := applyMigrationsOnDB(playerDBPath, migration.TargetPlayer); err != nil {
		return 0, fmt.Errorf("migrations player %s: %w", gamertag, err)
	}
	if err := applyMigrationsOnDB(sharedDBPath, migration.TargetShared); err != nil {
		return 0, fmt.Errorf("migrations shared: %w", err)
	}

	engine := go_sync.NewSyncEngineForTitle(cfg.RepoRoot, titleSlug, gamertag, xuid, nil, nil)
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

func runBackfillAllLUSR(ctx context.Context, cfg *config.AppConfig) error {
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
		if err := applyMigrationsOnDB(dbPath, migration.TargetPlayer); err != nil {
			failed++
			fmt.Printf("backfill lusr FAIL: gamertag=%s err=migrations: %v\n", player.Gamertag, err)
			continue
		}
		engine := go_sync.NewSyncEngine(cfg.RepoRoot, player.Gamertag, player.XUID, nil, nil)
		updated, runErr := engine.RecomputeLUSRCanonical(ctx)
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

func runBackfillLUSRForPlayer(ctx context.Context, cfg *config.AppConfig, player *domain.PlayerSummary) error {
	engine := go_sync.NewSyncEngine(cfg.RepoRoot, player.Gamertag, player.XUID, nil, nil)
	updated, err := engine.RecomputeLUSRCanonical(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("backfill lusr OK: gamertag=%s updated=%d\n", player.Gamertag, updated)
	return nil
}

// ── LUSR dry-run (preview, sans écriture) ─────────────────────────────────────
//
// Simule un recompute force=true et imprime un diff per-playlist_group :
// (OldMU, NewMU, delta). Utile pour valider l'impact d'un rebuild ART
// (cf. Phase 1 plan stabilisation 2026-05-22) avant d'engager l'écriture.

func runBackfillAllLUSRDryRun(ctx context.Context, cfg *config.AppConfig) error {
	players, err := cfg.LoadPlayers()
	if err != nil {
		return fmt.Errorf("chargement db_profiles.json: %w", err)
	}
	if len(players) == 0 {
		return fmt.Errorf("aucun joueur configure")
	}
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	total, processed, skipped, failed := len(players), 0, 0, 0
	for _, player := range players {
		dbPath := resolver.PlayerDBPath(titlePkg.DefaultSlug, player.Gamertag)
		if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
			skipped++
			fmt.Printf("dry-run lusr SKIP: gamertag=%s reason=no_player_db\n", player.Gamertag)
			continue
		}
		if err := runBackfillLUSRDryRunForPlayer(ctx, cfg, &player); err != nil {
			failed++
			fmt.Printf("dry-run lusr FAIL: gamertag=%s err=%v\n", player.Gamertag, err)
			continue
		}
		processed++
	}
	fmt.Printf("dry-run lusr batch: total=%d processed=%d skipped=%d failed=%d (aucune écriture)\n",
		total, processed, skipped, failed)
	if failed > 0 {
		return fmt.Errorf("dry-run lusr: %d joueur(s) en echec", failed)
	}
	return nil
}

func runBackfillLUSRDryRunForPlayer(ctx context.Context, cfg *config.AppConfig, player *domain.PlayerSummary) error {
	engine := go_sync.NewSyncEngine(cfg.RepoRoot, player.Gamertag, player.XUID, nil, nil)
	report, err := engine.RunBackfillLUSRDryRun(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("\n══ dry-run lusr %s (xuid=%s) ══\n", player.Gamertag, player.XUID)
	fmt.Printf("  matches_processed: %d\n", report.MatchesProcessed)
	if len(report.Playlists) == 0 {
		fmt.Printf("  (aucun playlist_group calculé — pas de matchs éligibles)\n")
		return nil
	}
	// Ordre canonique des composantes pour l'affichage reproductible.
	compOrder := []struct{ key, short string }{
		{go_sync.MetricKeyKillsVsExpected, "KvE"},
		{go_sync.MetricKeyDeathsVsExpected, "DvE"},
		{go_sync.MetricKeyWinFactor, "win"},
		{go_sync.MetricKeyDamageEfficiency, "dmg"},
		{go_sync.MetricKeyAccuracyDelta, "acc"},
		{go_sync.MetricKeyMedalExploit, "med"},
		{go_sync.MetricKeyOffensiveConv, "off"},
		{go_sync.MetricKeyDefensiveResist, "def"},
	}
	fmt.Printf("  %-30s %-15s %-15s %-12s %s\n", "playlist_group", "OLD μ/σ", "NEW μ/σ", "Δ μ", "matchs")
	fmt.Printf("  %s\n", strings.Repeat("─", 90))
	for _, p := range report.Playlists {
		fmt.Printf("  %-30s %-15s %-15s %+8.1f    %d\n",
			truncate(p.PlaylistGroup, 30),
			formatMuSigma(p.OldMU, p.OldSigma),
			formatMuSigma(p.NewMU, p.NewSigma),
			p.DeltaMU(),
			p.MatchCount,
		)
		if len(p.ComponentAvgs) > 0 {
			fmt.Printf("    avgs:")
			for _, c := range compOrder {
				if v, ok := p.ComponentAvgs[c.key]; ok {
					marker := " "
					if v < 0.48 {
						marker = "↓"
					} else if v > 0.52 {
						marker = "↑"
					}
					fmt.Printf("  %s=%.3f%s", c.short, v, marker)
				}
			}
			fmt.Println()
		}
	}
	if !report.HasChanges() {
		fmt.Printf("  → AUCUN CHANGEMENT significatif (tous deltas < 1.0 μ)\n")
	}
	return nil
}

// ── Simulation de formules LUSR ──────────────────────────────────────────────
//
// Compare 5 variantes (baseline / piste-A / B / C / A+C) sur les N derniers
// matchs, partant de InitialMU=1500. Aide à choisir la formule optimale.

func runFormulaSimAll(ctx context.Context, cfg *config.AppConfig, lastN int) error {
	players, err := cfg.LoadPlayers()
	if err != nil {
		return fmt.Errorf("chargement db_profiles.json: %w", err)
	}
	if len(players) == 0 {
		return fmt.Errorf("aucun joueur configure")
	}
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	for _, player := range players {
		dbPath := resolver.PlayerDBPath(titlePkg.DefaultSlug, player.Gamertag)
		if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
			fmt.Printf("sim-formula SKIP: gamertag=%s reason=no_player_db\n", player.Gamertag)
			continue
		}
		if err := runFormulaSimForPlayer(ctx, cfg, &player, lastN); err != nil {
			fmt.Printf("sim-formula FAIL: gamertag=%s err=%v\n", player.Gamertag, err)
		}
	}
	return nil
}

func runFormulaSimForPlayer(ctx context.Context, cfg *config.AppConfig, player *domain.PlayerSummary, lastN int) error {
	engine := go_sync.NewSyncEngine(cfg.RepoRoot, player.Gamertag, player.XUID, nil, nil)
	report, err := engine.RunFormulaSim(ctx, lastN)
	if err != nil {
		return err
	}
	labelN := "tous"
	if lastN > 0 {
		labelN = fmt.Sprintf("%d", lastN)
	}
	fmt.Printf("\n══ sim-formula %s (last %s matchs, depuis μ=1500) ══\n", player.Gamertag, labelN)
	if len(report.Results) == 0 {
		fmt.Printf("  (aucun match éligible LUSR)\n")
		return nil
	}
	// En-tête
	fmt.Printf("  %-20s %6s", "chain", "matchs")
	for _, v := range go_sync.SimulationVariants {
		fmt.Printf("  %-18s", v.Name)
	}
	fmt.Println()
	fmt.Printf("  %s\n", strings.Repeat("─", 20+7+len(go_sync.SimulationVariants)*20))

	for _, r := range report.Results {
		fmt.Printf("  %-20s %6d", r.Chain, r.MatchCount)
		for _, v := range go_sync.SimulationVariants {
			mu := r.MUByVariant[v.Name]
			tier := go_sync.FormatTierLabel(mu)
			fmt.Printf("  %6.0f %-11s", mu, tier)
		}
		fmt.Println()
	}
	// Sigma baseline par chaîne — mesure la convergence TrueSkill.
	fmt.Printf("  sigma (baseline):")
	for _, r := range report.Results {
		fmt.Printf("  %s=%.1f", r.Chain, r.SigmaByVariant["baseline"])
	}
	fmt.Println()
	return nil
}

func formatMuSigma(mu, sigma float64) string {
	if mu == 0 && sigma == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.1f/%.1f", mu, sigma)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

// ── CSR backfill ───────────────────────────────────────────────────────────────
//
// Re-fetche GetMatchSkill pour chaque match classé en DB et persiste la ligne
// CSR dans match_skill_rank. Nécessite des tokens Halo valides (OAuth refresh
// via MSAL) — pattern identique à --weapons.

func runBackfillAllCSR(ctx context.Context, cfg *config.AppConfig, force bool) error {
	players, err := cfg.LoadPlayers()
	if err != nil {
		return fmt.Errorf("chargement db_profiles.json: %w", err)
	}
	if len(players) == 0 {
		return fmt.Errorf("aucun joueur configure")
	}
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	total, processed, skipped, failed, totalInserted := len(players), 0, 0, 0, 0
	for _, player := range players {
		dbPath := resolver.PlayerDBPath(titlePkg.DefaultSlug, player.Gamertag)
		if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
			skipped++
			fmt.Printf("backfill csr SKIP: gamertag=%s reason=no_player_db\n", player.Gamertag)
			continue
		}
		if err := applyMigrationsOnDB(dbPath, migration.TargetPlayer); err != nil {
			failed++
			fmt.Printf("backfill csr FAIL: gamertag=%s err=migrations: %v\n", player.Gamertag, err)
			continue
		}

		tokens, tokErr := refreshHaloTokensForPlayer(ctx, player.Gamertag)
		if tokErr != nil {
			skipped++
			fmt.Printf("backfill csr SKIP: gamertag=%s reason=%v\n", player.Gamertag, tokErr)
			continue
		}

		engine := go_sync.NewSyncEngine(cfg.RepoRoot, player.Gamertag, player.XUID, tokens, nil)
		res, runErr := engine.RunBackfillCSR(ctx, force)
		if runErr != nil {
			failed++
			fmt.Printf("backfill csr FAIL: gamertag=%s err=%v\n", player.Gamertag, runErr)
			continue
		}
		processed++
		totalInserted += res.Inserted
		fmt.Printf("backfill csr OK: gamertag=%s inserted=%d already=%d no_recap=%d errors=%d\n",
			player.Gamertag, res.Inserted, res.AlreadyHadCSR, res.SkippedNoRankRecap, res.SkillErrors)
	}
	fmt.Printf("backfill csr batch: total=%d processed=%d skipped=%d failed=%d total_inserted=%d\n",
		total, processed, skipped, failed, totalInserted)
	if failed > 0 {
		return fmt.Errorf("backfill csr: %d joueur(s) en echec", failed)
	}
	return nil
}

func runBackfillCSRForPlayer(ctx context.Context, cfg *config.AppConfig, player *domain.PlayerSummary, force bool) error {
	tokens, err := refreshHaloTokensForPlayer(ctx, player.Gamertag)
	if err != nil {
		return fmt.Errorf("backfill csr: tokens Halo indisponibles pour %s: %w", player.Gamertag, err)
	}
	engine := go_sync.NewSyncEngine(cfg.RepoRoot, player.Gamertag, player.XUID, tokens, nil)
	res, err := engine.RunBackfillCSR(ctx, force)
	if err != nil {
		return err
	}
	fmt.Printf("backfill csr OK: gamertag=%s inserted=%d already=%d no_recap=%d errors=%d force=%t\n",
		player.Gamertag, res.Inserted, res.AlreadyHadCSR, res.SkippedNoRankRecap, res.SkillErrors, force)
	return nil
}

// ── Shared CSR backfill (Option A — all participants per match) ────────────
//
// Persiste le CSR de TOUS les joueurs d'un match ranked dans shared.match_csrs
// (vs. legacy --csr qui n'écrit que le CSR du joueur sync dans sa player DB).
// Mode --dry-run : compte les matchs nécessitant un backfill sans appel API
// ni écriture — idéal pour valider l'ampleur avant exécution réelle.

func runBackfillAllSharedCSR(ctx context.Context, cfg *config.AppConfig, force, dryRun bool) error {
	players, err := cfg.LoadPlayers()
	if err != nil {
		return fmt.Errorf("chargement db_profiles.json: %w", err)
	}
	if len(players) == 0 {
		return fmt.Errorf("aucun joueur configure")
	}
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	total, processed, skipped, failed, totalInserted := len(players), 0, 0, 0, 0
	for _, player := range players {
		dbPath := resolver.PlayerDBPath(titlePkg.DefaultSlug, player.Gamertag)
		if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
			skipped++
			fmt.Printf("backfill shared-csr SKIP: gamertag=%s reason=no_player_db\n", player.Gamertag)
			continue
		}
		sharedDBPath := resolver.SharedDBPath(titlePkg.DefaultSlug)
		if err := applyMigrationsOnDB(sharedDBPath, migration.TargetShared); err != nil {
			failed++
			fmt.Printf("backfill shared-csr FAIL: gamertag=%s err=migrations shared: %v\n", player.Gamertag, err)
			continue
		}

		var tokens *domain.HaloTokens
		if !dryRun {
			t, tokErr := refreshHaloTokensForPlayer(ctx, player.Gamertag)
			if tokErr != nil {
				skipped++
				fmt.Printf("backfill shared-csr SKIP: gamertag=%s reason=%v (try --dry-run)\n", player.Gamertag, tokErr)
				continue
			}
			tokens = t
		}

		engine := go_sync.NewSyncEngine(cfg.RepoRoot, player.Gamertag, player.XUID, tokens, nil)
		res, runErr := engine.RunBackfillSharedCSR(ctx, go_sync.SharedCSRBackfillOpts{Force: force, DryRun: dryRun})
		if runErr != nil {
			failed++
			fmt.Printf("backfill shared-csr FAIL: gamertag=%s err=%v\n", player.Gamertag, runErr)
			continue
		}
		processed++
		totalInserted += res.Inserted
		fmt.Printf("backfill shared-csr OK: gamertag=%s ranked=%d already_complete=%d need_backfill=%d fetched=%d inserted=%d no_recap=%d errors=%d dry_run=%t\n",
			player.Gamertag, res.RankedMatches, res.AlreadyComplete, res.NeedBackfill,
			res.Fetched, res.Inserted, res.SkippedNoRankRecap, res.SkillErrors+res.UpsertErrors, res.DryRun)
	}
	fmt.Printf("backfill shared-csr batch: total=%d processed=%d skipped=%d failed=%d total_inserted=%d dry_run=%t\n",
		total, processed, skipped, failed, totalInserted, dryRun)
	if failed > 0 {
		return fmt.Errorf("backfill shared-csr: %d joueur(s) en echec", failed)
	}
	return nil
}

func runBackfillSharedCSRForPlayer(ctx context.Context, cfg *config.AppConfig, player *domain.PlayerSummary, force, dryRun bool) error {
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	sharedDBPath := resolver.SharedDBPath(titlePkg.DefaultSlug)
	if err := applyMigrationsOnDB(sharedDBPath, migration.TargetShared); err != nil {
		return fmt.Errorf("backfill shared-csr: migrations shared: %w", err)
	}

	var tokens *domain.HaloTokens
	if !dryRun {
		t, err := refreshHaloTokensForPlayer(ctx, player.Gamertag)
		if err != nil {
			return fmt.Errorf("backfill shared-csr: tokens Halo indisponibles pour %s: %w (utiliser --dry-run pour compter sans appel API)", player.Gamertag, err)
		}
		tokens = t
	}

	engine := go_sync.NewSyncEngine(cfg.RepoRoot, player.Gamertag, player.XUID, tokens, nil)
	res, err := engine.RunBackfillSharedCSR(ctx, go_sync.SharedCSRBackfillOpts{Force: force, DryRun: dryRun})
	if err != nil {
		return err
	}
	fmt.Printf("backfill shared-csr OK: gamertag=%s ranked=%d already_complete=%d need_backfill=%d fetched=%d inserted=%d no_recap=%d errors=%d force=%t dry_run=%t\n",
		player.Gamertag, res.RankedMatches, res.AlreadyComplete, res.NeedBackfill,
		res.Fetched, res.Inserted, res.SkippedNoRankRecap, res.SkillErrors+res.UpsertErrors, force, res.DryRun)
	return nil
}

// refreshHaloTokensForPlayer charge le refresh token OAuth du joueur et le
// rafraîchit via MSAL pour obtenir des tokens Halo (Spartan + Clearance)
// utilisables par les backfills qui appellent l'API. Retourne une erreur
// descriptive si le refresh_token est absent ou si l'échange MSAL/Halo échoue.
func refreshHaloTokensForPlayer(ctx context.Context, gamertag string) (*domain.HaloTokens, error) {
	refreshToken := oauthRefreshTokenForPlayer(gamertag)
	if refreshToken == "" {
		return nil, fmt.Errorf("no_refresh_token (%s)", oauthRefreshEnvKey(gamertag))
	}
	provider := auth.NewMSALProvider()
	accessToken, err := provider.TryOAuthRefresh(ctx, refreshToken)
	if err != nil || accessToken == "" {
		return nil, fmt.Errorf("oauth_refresh_failed: %w", err)
	}
	result, err := provider.Exchange(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("halo_exchange_failed: %w", err)
	}
	return result.Tokens, nil
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

// ── Citations recompute-all (force + vérifications V1-V4) ─────────────────────

func runRecomputeAllCitationsAll(ctx context.Context, cfg *config.AppConfig) error {
	players, err := cfg.LoadPlayers()
	if err != nil {
		return fmt.Errorf("chargement db_profiles.json: %w", err)
	}
	if len(players) == 0 {
		return fmt.Errorf("aucun joueur configure")
	}

	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	total, processed, skipped, failed := len(players), 0, 0, 0

	for _, player := range players {
		dbPath := resolver.PlayerDBPath(titlePkg.DefaultSlug, player.Gamertag)
		if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
			skipped++
			fmt.Printf("citations-recompute-all SKIP: gamertag=%s reason=no_player_db\n", player.Gamertag)
			continue
		}
		if runErr := runRecomputeAllCitationsOne(ctx, cfg, player.Gamertag, player.XUID); runErr != nil {
			failed++
			fmt.Printf("citations-recompute-all FAIL: gamertag=%s err=%v\n", player.Gamertag, runErr)
			continue
		}
		processed++
	}

	fmt.Printf("citations-recompute-all batch: total=%d processed=%d skipped=%d failed=%d\n",
		total, processed, skipped, failed)
	if failed > 0 {
		return fmt.Errorf("citations-recompute-all: %d joueur(s) en echec", failed)
	}
	return nil
}

func runRecomputeAllCitationsForPlayer(ctx context.Context, cfg *config.AppConfig, gamertag, xuid string) error {
	return runRecomputeAllCitationsOne(ctx, cfg, gamertag, xuid)
}

// runRecomputeAllCitationsOne : recompute force=true puis invariants V1-V4.
func runRecomputeAllCitationsOne(ctx context.Context, cfg *config.AppConfig, gamertag, xuid string) error {
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	playerDBPath := resolver.PlayerDBPath(titlePkg.DefaultSlug, gamertag)
	sharedDBPath := resolver.SharedDBPath(titlePkg.DefaultSlug)
	metaDBPath := resolver.MetadataDBPath(titlePkg.DefaultSlug)

	for _, p := range []struct {
		path   string
		target migration.TargetDB
	}{
		{playerDBPath, migration.TargetPlayer},
		{sharedDBPath, migration.TargetShared},
		{metaDBPath, migration.TargetMetadata},
	} {
		if err := applyMigrationsOnDB(p.path, p.target); err != nil {
			return fmt.Errorf("migrations %s: %w", p.path, err)
		}
	}

	engine := go_sync.NewSyncEngine(cfg.RepoRoot, gamertag, xuid, nil, nil)

	updated, err := engine.RunBackfillCitations(ctx, true)
	if err != nil {
		return fmt.Errorf("recompute citations: %w", err)
	}
	fmt.Printf("citations-recompute-all OK: gamertag=%s matches_updated=%d\n", gamertag, updated)

	violations, err := engine.RunCitationPostComputeChecks(ctx)
	if err != nil {
		return fmt.Errorf("post-compute checks: %w", err)
	}
	if len(violations) == 0 {
		fmt.Printf("citations-recompute-all checks OK: gamertag=%s invariants=V1-V4\n", gamertag)
		return nil
	}
	for _, v := range violations {
		fmt.Printf("citations-recompute-all VIOLATION [%s]: gamertag=%s %s\n", v.Rule, gamertag, v.Details)
	}
	return fmt.Errorf("citations-recompute-all: %d violation(s) détectée(s) pour %s", len(violations), gamertag)
}
