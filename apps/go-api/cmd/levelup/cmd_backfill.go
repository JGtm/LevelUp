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
//   - --lusr                       [--force]  (rejoue le LUSR v2 complet + les 8 composantes)
//   - --perf                       [--force]  (recalcule performance score relatif v5)
//   - --assists-model              [--force]  (modele OLS des assistances attendues)
//
// `--title` (defaut : le titre par defaut) choisit le titre de toutes les passes locales ci-dessus
// et de --compare-formulas ; --csr et --shared-csr (appel API, base du titre par defaut) le refusent
// pour un autre titre. Une passe dont le titre ne declare pas la capability (citations sur Halo 5)
// repond « non supporte » sans rien executer. Les passes selectionnees tournent toutes, dans l'ordre
// de la table de cmd_backfill_titled.go.
//
// Usage :
//
//	levelup backfill --gamertag X --lusr  [--force]
//	levelup backfill --all          --perf  [--force]
//	levelup backfill --all          --lusr --perf --force
//	levelup backfill --all --title halo_5 --perf --assists-model --lusr --force
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
	settings_platform "levelup/go-api/internal/platform/settings"
	go_sync "levelup/go-api/internal/sync"
)

func runBackfill(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("backfill", flag.ExitOnError)
	gamertag := fs.String("gamertag", "", "Gamertag du joueur (mutuellement exclusif avec --all)")
	allPlayers := fs.Bool("all", false, "Applique le backfill a tous les joueurs configures")
	engagementScores := fs.Bool("engagement-scores", false, "Backfill du score d'engagement (Phase 6 plan engagement)")
	titleSlug := fs.String("title", titlePkg.DefaultSlug, "Slug du titre des passes locales (halo_infinite | halo_5)")
	citations := fs.Bool("citations", false, "Backfill des citations (match_citations) depuis citation_mappings + medals + stats + awards")
	lusr := fs.Bool("lusr", false, "Rejoue le LUSR v2 complet du joueur (note + 8 composantes de performance)")
	csr := fs.Bool("csr", false, "Backfill CSR par-match via GetMatchSkill (RankRecap). Idempotent ; --force re-fetche tous les matchs ranked")
	sharedCSR := fs.Bool("shared-csr", false, "Backfill shared.match_csrs (CSR de TOUS les participants des matchs ranked). --dry-run pour compter sans écrire.")
	perf := fs.Bool("perf", false, "Backfill performance score relatif v5 (off_conv + def_res + medal_exploit)")
	assistsModel := fs.Bool("assists-model", false, "Calcule le modèle OLS expected_assists par mode (player_assists_model dans stats.duckdb)")
	compositeOnly := fs.Bool("composite-only", false, "Backfill citations composites uniquement (additive, sans recalcul depuis shared_matches)")
	citationsRecomputeAll := fs.Bool("citations-recompute-all", false, "Recompute total des citations (force=true) + vérifications invariants V1-V4")
	force := fs.Bool("force", false, "Force le recalcul meme si deja persiste")
	dryRun := fs.Bool("dry-run", false, "Mode dry-run (--shared-csr) : compte les matchs sans appel API ni écriture")
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
	if !*engagementScores && !*citations && !*citationsRecomputeAll && !*lusr && !*csr && !*sharedCSR && !*perf && !*assistsModel && !*compositeOnly && !*compareFormulas {
		return fmt.Errorf("aucun backfill selectionne (utiliser --engagement-scores, --citations, --citations-recompute-all, --lusr, --csr, --shared-csr, --perf, --assists-model, --composite-only ou --compare-formulas)")
	}
	if *dryRun && !*sharedCSR {
		return fmt.Errorf("--dry-run n'est supporté qu'avec --shared-csr")
	}
	if *csr || *sharedCSR {
		if err := requireCSRCapability(cfg, *titleSlug); err != nil {
			return err
		}
	}

	ctx := context.Background()
	target := backfillTarget{titleSlug: *titleSlug, all: *allPlayers, gamertag: strings.TrimSpace(*gamertag)}
	selected := []struct {
		on   bool
		pass titledPass
	}{
		{*engagementScores, passEngagement},
		{*citations, passCitations},
		{*lusr, passLUSR},
	}
	for _, s := range selected {
		if s.on {
			if err := runTitledPass(ctx, cfg, target, *force, s.pass); err != nil {
				return err
			}
		}
	}
	if err := runBackfillCSRPasses(ctx, cfg, target, *csr, *sharedCSR, *force, *dryRun); err != nil {
		return err
	}
	selected = []struct {
		on   bool
		pass titledPass
	}{
		{*perf, passPerf},
		{*assistsModel, passAssistsModel},
		{*compositeOnly, passCompositeOnly},
		{*citationsRecomputeAll, passCitationsRecomputeAll},
	}
	for _, s := range selected {
		if s.on {
			if err := runTitledPass(ctx, cfg, target, *force, s.pass); err != nil {
				return err
			}
		}
	}
	if *compareFormulas {
		return runFormulaSims(ctx, cfg, target, *lastN)
	}
	return nil
}

// runBackfillCSRPasses : --csr puis --shared-csr (titre par defaut, appel API).
func runBackfillCSRPasses(ctx context.Context, cfg *config.AppConfig, t backfillTarget, csr, sharedCSR, force, dryRun bool) error {
	if !csr && !sharedCSR {
		return nil
	}
	var player *domain.PlayerSummary
	if !t.all {
		p, err := loadPlayerSummary(cfg, t.gamertag)
		if err != nil {
			return err
		}
		player = p
	}
	if csr {
		if t.all {
			if err := runBackfillAllCSR(ctx, cfg, force); err != nil {
				return err
			}
		} else if err := runBackfillCSRForPlayer(ctx, cfg, player, force); err != nil {
			return err
		}
	}
	if sharedCSR {
		if t.all {
			return runBackfillAllSharedCSR(ctx, cfg, force, dryRun)
		}
		return runBackfillSharedCSRForPlayer(ctx, cfg, player, force, dryRun)
	}
	return nil
}

// runFormulaSims : --compare-formulas sur les joueurs du titre.
func runFormulaSims(ctx context.Context, cfg *config.AppConfig, t backfillTarget, lastN int) error {
	players, err := backfillPlayers(cfg, t)
	if err != nil {
		return err
	}
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	for _, player := range players {
		if _, statErr := os.Stat(resolver.PlayerDBPath(t.titleSlug, player.Gamertag)); os.IsNotExist(statErr) {
			fmt.Printf("sim-formula SKIP: title=%s gamertag=%s reason=no_player_db\n", t.titleSlug, player.Gamertag)
			continue
		}
		if err := runFormulaSimForPlayer(ctx, cfg, t.titleSlug, &player, lastN); err != nil {
			if !t.all {
				return err
			}
			fmt.Printf("sim-formula FAIL: gamertag=%s err=%v\n", player.Gamertag, err)
		}
	}
	return nil
}

// ── Simulation de formules LUSR ──────────────────────────────────────────────
//
// Compare 5 variantes (baseline / piste-A / B / C / A+C) sur les N derniers
// matchs, partant de InitialMU=1500. Aide à choisir la formule optimale.

func runFormulaSimForPlayer(ctx context.Context, cfg *config.AppConfig, titleSlug string, player *domain.PlayerSummary, lastN int) error {
	engine := go_sync.NewSyncEngineForTitle(cfg.RepoRoot, titleSlug, player.Gamertag, player.XUID, nil, nil).
		WithSessionOptionsLoader(settings_platform.SessionOptionsLoaderForConfig(cfg, titleSlug))
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
