package main

// cmd_backfill_csr.go — passes CSR de `levelup backfill` (--csr, --shared-csr).
//
// Extrait de cmd_backfill.go (> 1 000 lignes, dette gelee : ne pas l accroitre) le 2026-09-16.
// DOCTRINE (D1) : le CSR est un endpoint PUBLIC, servi par n importe quel token du pool ; un
// joueur sans token propre est traite comme les autres. En --dry-run, AUCUN pool n est
// construit : construire le pool resout et fait tourner les refresh tokens de tout le parc
// (rotation persistee dans le store), ce qu un dry-run ne doit jamais faire (revue
// adversariale du 2026-09-16, P1).

import (
	"context"
	"fmt"
	"os"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
	auth_platform "levelup/go-api/internal/platform/auth"
	auth_pool "levelup/go-api/internal/platform/auth/pool"
	go_sync "levelup/go-api/internal/sync"
)

// newDryRunEngine : moteur SANS client Halo pour une passe --dry-run (aucun appel API, aucun
// pool, aucune rotation de token). RunBackfillSharedCSR n exige un client qu en non-dry-run.
func newDryRunEngine(cfg *config.AppConfig, player domain.PlayerSummary) *go_sync.SyncEngine {
	return go_sync.NewSyncEngine(cfg.RepoRoot, player.Gamertag, player.XUID, &domain.HaloTokens{},
		auth_platform.NewSISUProvider()).WithCSRSeasonID(cfg.CurrentCSRSeasonID)
}

// ── CSR backfill ───────────────────────────────────────────────────────────────
//
// Re-fetche GetMatchSkill pour chaque match classé en DB et persiste la ligne
// CSR dans match_skill_rank. Nécessite des tokens Halo valides (OAuth refresh
// via MSAL).

func runBackfillAllCSR(ctx context.Context, cfg *config.AppConfig, force bool) error {
	players, err := cfg.LoadPlayers()
	if err != nil {
		return fmt.Errorf("chargement db_profiles.json: %w", err)
	}
	if len(players) == 0 {
		return fmt.Errorf("aucun joueur configure")
	}
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	// Un seul pool pour toute la passe : le CSR est un endpoint public, servi par
	// n'importe quel token du parc (D1, plan 2026-09-16). Un joueur sans token propre
	// n'est plus saute.
	provider := auth_platform.NewSISUProvider()
	pool, poolErr := buildCLITokenPool(ctx, cfg, provider, players, 0, 0)
	if poolErr != nil {
		return poolErr
	}
	defer pool.Close()
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

		// Le joueur n'a pas besoin de son propre token : le CSR est un endpoint public,
		// servi par n'importe quel token du pool (D1, plan 2026-09-16).
		engine := newPooledEngine(cfg, provider, pool, player)
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
	// CSR = endpoint public : le pool sert le joueur meme sans token propre (D1).
	engine, closePool, err := newPooledEngineForPlayer(ctx, cfg, *player, 0, 0)
	if err != nil {
		return fmt.Errorf("backfill csr: pool de tokens indisponible pour %s: %w", player.Gamertag, err)
	}
	defer closePool()
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
	// Un seul pool pour toute la passe : le CSR est un endpoint public, servi par
	// n importe quel token du parc (D1, plan 2026-09-16). Un joueur sans token propre
	// n est plus saute. En --dry-run : AUCUN pool (aucun appel API, aucune rotation).
	var provider auth_platform.TokenProvider
	var pool auth_pool.Pool
	if !dryRun {
		provider = auth_platform.NewSISUProvider()
		p, poolErr := buildCLITokenPool(ctx, cfg, provider, players, 0, 0)
		if poolErr != nil {
			return poolErr
		}
		defer p.Close()
		pool = p
	}
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

		// CSR partage = endpoint public : le pool sert tout joueur suivi (D1). En dry-run,
		// moteur SANS client : rien ne sort sur le reseau, aucun token ne tourne.
		engine := newDryRunEngine(cfg, player)
		if !dryRun {
			engine = newPooledEngine(cfg, provider, pool, player)
		}
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

	// CSR partage = endpoint public : le pool sert le joueur meme sans token propre (D1).
	// En --dry-run : moteur SANS client, aucun pool construit, aucun token ne tourne.
	engine := newDryRunEngine(cfg, *player)
	if !dryRun {
		pooled, closePool, err := newPooledEngineForPlayer(ctx, cfg, *player, 0, 0)
		if err != nil {
			return fmt.Errorf("backfill shared-csr: pool de tokens indisponible pour %s: %w (--dry-run compte sans appel API ni pool)", player.Gamertag, err)
		}
		defer closePool()
		engine = pooled
	}
	res, err := engine.RunBackfillSharedCSR(ctx, go_sync.SharedCSRBackfillOpts{Force: force, DryRun: dryRun})
	if err != nil {
		return err
	}
	fmt.Printf("backfill shared-csr OK: gamertag=%s ranked=%d already_complete=%d need_backfill=%d fetched=%d inserted=%d no_recap=%d errors=%d force=%t dry_run=%t\n",
		player.Gamertag, res.RankedMatches, res.AlreadyComplete, res.NeedBackfill,
		res.Fetched, res.Inserted, res.SkippedNoRankRecap, res.SkillErrors+res.UpsertErrors, force, res.DryRun)
	return nil
}
