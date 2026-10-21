package main

// cmd_backfill_titled.go — passes locales de `levelup backfill` qui s'appliquent à un
// (joueur, titre) : engagement, citations, LUSR v2, score de performance, modèle des
// assistances, citations composites, recalcul total des citations.
//
// `--title` choisit le titre de TOUTES ces passes. Elles étaient codées sur le titre par défaut
// (seule l'engagement acceptait `--title`) : sur Halo 5, un recalcul des dérivés n'avait aucune
// commande. Une seule boucle et une table de passes remplacent les six paires
// `runBackfillAllX` / `runBackfillXForPlayer` recopiées.
//
// Une passe porte éventuellement une CAPABILITY de porte : un titre qui ne la déclare pas n'est
// pas servi par la passe, qui répond « non supporté pour ce titre » et ne s'exécute pas (jamais
// de calcul à vide ni d'écriture dans la base du titre). Garde-rail :
// `cmd_backfill_titled_test.go` (aucun `DefaultSlug` codé en dur dans les passes).

import (
	"context"
	"fmt"
	"os"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/migration"
	settings_platform "levelup/go-api/internal/platform/settings"
	go_sync "levelup/go-api/internal/sync"
)

// backfillTarget : le titre visé et les joueurs de la passe (tous les profils du titre, ou un
// gamertag).
type backfillTarget struct {
	titleSlug string
	all       bool
	gamertag  string
}

// unitUpdated : l'unité d'une passe qui compte des matchs mis à jour.
const unitUpdated = "updated"

// titledPass décrit une passe locale (aucun appel API) sur un moteur de sync d'un titre.
type titledPass struct {
	name string // libellé du journal : "perf", "lusr"...
	unit string // ce que la passe compte : "updated", "n_modes"
	// gate : capability que le titre doit déclarer ; vide = la passe vaut pour tous les titres.
	gate games.CapabilityKey
	// migrate : cibles migrées (bases du TITRE) avant la passe, dans l'ordre.
	migrate []migration.TargetDB
	run     func(ctx context.Context, engine *go_sync.SyncEngine, force bool) (int, error)
}

// supported dit si le titre sert la passe (sa capability de porte, si elle en a une).
func (p titledPass) supported(cfg *config.AppConfig, titleSlug string) (bool, error) {
	if p.gate == "" {
		return true, nil
	}
	caps, err := games.LoadCapabilityMap(cfg.RepoRoot, titleSlug)
	if err != nil {
		return false, err
	}
	return caps.Has(p.gate), nil
}

// The passes of `levelup backfill`, one entry per flag. Order of the table = order of execution.
var (
	passEngagement = titledPass{
		name: "engagement", unit: unitUpdated,
		migrate: []migration.TargetDB{migration.TargetPlayer, migration.TargetShared},
		run: func(ctx context.Context, e *go_sync.SyncEngine, force bool) (int, error) {
			return e.RunBackfillEngagementScores(ctx, force)
		},
	}
	passCitations = titledPass{
		name: "citations", unit: unitUpdated, gate: games.CapCitationsEngine,
		migrate: []migration.TargetDB{migration.TargetPlayer, migration.TargetShared, migration.TargetMetadata},
		run: func(ctx context.Context, e *go_sync.SyncEngine, force bool) (int, error) {
			return e.RunBackfillCitations(ctx, force)
		},
	}
	passLUSR = titledPass{
		name: "lusr", unit: unitUpdated,
		migrate: []migration.TargetDB{migration.TargetPlayer},
		run: func(ctx context.Context, e *go_sync.SyncEngine, _ bool) (int, error) {
			return e.RecomputeLUSRCanonical(ctx)
		},
	}
	passPerf = titledPass{
		name: "perf", unit: unitUpdated,
		run: func(ctx context.Context, e *go_sync.SyncEngine, force bool) (int, error) {
			return e.RunBackfillPerf(ctx, force)
		},
	}
	passAssistsModel = titledPass{
		name: "assists-model", unit: "n_modes",
		migrate: []migration.TargetDB{migration.TargetPlayer},
		run: func(ctx context.Context, e *go_sync.SyncEngine, force bool) (int, error) {
			return e.RunBackfillAssistsModel(ctx, force)
		},
	}
	passCompositeOnly = titledPass{
		name: "composite-only", unit: unitUpdated, gate: games.CapCitationsEngine,
		migrate: []migration.TargetDB{migration.TargetPlayer, migration.TargetMetadata},
		run: func(ctx context.Context, e *go_sync.SyncEngine, _ bool) (int, error) {
			return e.RunBackfillCompositeOnlyCitations(ctx)
		},
	}
	passCitationsRecomputeAll = titledPass{
		name: "citations-recompute-all", unit: "matches_updated", gate: games.CapCitationsEngine,
		migrate: []migration.TargetDB{migration.TargetPlayer, migration.TargetShared, migration.TargetMetadata},
		run:     recomputeAllCitationsAndCheck,
	}
)

// recomputeAllCitationsAndCheck : recompute force=true puis invariants V1-V4. Une violation
// fait échouer le joueur.
func recomputeAllCitationsAndCheck(ctx context.Context, e *go_sync.SyncEngine, _ bool) (int, error) {
	updated, err := e.RunBackfillCitations(ctx, true)
	if err != nil {
		return 0, fmt.Errorf("recompute citations: %w", err)
	}
	violations, err := e.RunCitationPostComputeChecks(ctx)
	if err != nil {
		return updated, fmt.Errorf("post-compute checks: %w", err)
	}
	if len(violations) == 0 {
		return updated, nil
	}
	for _, v := range violations {
		fmt.Printf("citations-recompute-all VIOLATION [%s]: %s\n", v.Rule, v.Details)
	}
	return updated, fmt.Errorf("%d violation(s) detectee(s)", len(violations))
}

// backfillMigrationPath : la base du titre que vise une cible de migration.
func backfillMigrationPath(pr *titlePkg.PathResolver, titleSlug, gamertag string, t migration.TargetDB) (string, error) {
	switch t {
	case migration.TargetPlayer:
		return pr.PlayerDBPath(titleSlug, gamertag), nil
	case migration.TargetShared:
		return pr.SharedDBPath(titleSlug), nil
	case migration.TargetMetadata:
		return pr.MetadataDBPath(titleSlug), nil
	default:
		return "", fmt.Errorf("cible de migration inconnue pour une passe de backfill : %v", t)
	}
}

// backfillPlayers rend les joueurs de la passe. `--all` prend les profils DU TITRE (jamais
// ceux d'un autre titre ni les profils réservés à l'authentification, sans base joueur).
func backfillPlayers(cfg *config.AppConfig, t backfillTarget) ([]domain.PlayerSummary, error) {
	if !t.all {
		p, err := loadPlayerSummary(cfg, t.gamertag)
		if err != nil {
			return nil, err
		}
		return []domain.PlayerSummary{*p}, nil
	}
	profiles, err := cfg.LoadPlayers(t.titleSlug)
	if err != nil {
		return nil, fmt.Errorf("chargement db_profiles.json: %w", err)
	}
	players := make([]domain.PlayerSummary, 0, len(profiles))
	for _, p := range profiles {
		if !p.AuthOnly {
			players = append(players, p)
		}
	}
	if len(players) == 0 {
		return nil, fmt.Errorf("aucun joueur configure pour le titre %s", t.titleSlug)
	}
	return players, nil
}

// runTitledPass applique une passe aux joueurs du titre. Le titre sans la capability de la
// passe n'est pas un échec : la passe le dit et sort sans rien exécuter.
func runTitledPass(ctx context.Context, cfg *config.AppConfig, t backfillTarget, force bool, pass titledPass) error {
	if t.titleSlug == "" {
		t.titleSlug = titlePkg.DefaultSlug
	}
	ok, err := pass.supported(cfg, t.titleSlug)
	if err != nil {
		return fmt.Errorf("backfill %s: capabilities du titre %s: %w", pass.name, t.titleSlug, err)
	}
	if !ok {
		fmt.Printf("backfill %s SKIP: non supporte pour le titre %s (capability %s absente) : rien execute\n",
			pass.name, t.titleSlug, pass.gate)
		return nil
	}
	players, err := backfillPlayers(cfg, t)
	if err != nil {
		return err
	}
	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	var processed, skipped, failed, total int
	for _, p := range players {
		if _, statErr := os.Stat(pr.PlayerDBPath(t.titleSlug, p.Gamertag)); os.IsNotExist(statErr) {
			if !t.all {
				return fmt.Errorf("backfill %s: pas de base joueur de %s pour le titre %s", pass.name, p.Gamertag, t.titleSlug)
			}
			skipped++
			fmt.Printf("backfill %s SKIP: title=%s gamertag=%s reason=no_player_db\n", pass.name, t.titleSlug, p.Gamertag)
			continue
		}
		n, runErr := runTitledPassOne(ctx, cfg, t.titleSlug, p, force, pass)
		if runErr != nil {
			if !t.all {
				return runErr
			}
			failed++
			fmt.Printf("backfill %s FAIL: title=%s gamertag=%s err=%v\n", pass.name, t.titleSlug, p.Gamertag, runErr)
			continue
		}
		processed++
		total += n
		fmt.Printf("backfill %s OK: title=%s gamertag=%s %s=%d force=%t\n", pass.name, t.titleSlug, p.Gamertag, pass.unit, n, force)
	}
	if !t.all {
		return nil
	}
	fmt.Printf("backfill %s batch: title=%s total=%d processed=%d skipped=%d failed=%d total_%s=%d\n",
		pass.name, t.titleSlug, len(players), processed, skipped, failed, pass.unit, total)
	if failed > 0 {
		return fmt.Errorf("backfill %s: %d joueur(s) en echec", pass.name, failed)
	}
	return nil
}

// runTitledPassOne migre les bases du titre, construit le moteur DU TITRE (sans jetons : calcul
// local pur) et lance la passe pour un joueur.
func runTitledPassOne(ctx context.Context, cfg *config.AppConfig, titleSlug string, p domain.PlayerSummary, force bool, pass titledPass) (int, error) {
	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	for _, target := range pass.migrate {
		path, err := backfillMigrationPath(pr, titleSlug, p.Gamertag, target)
		if err != nil {
			return 0, err
		}
		if err := applyTitleMigrationsOnDB(path, titleSlug, target); err != nil {
			return 0, fmt.Errorf("migrations %s (%s): %w", target, p.Gamertag, err)
		}
	}
	engine := go_sync.NewSyncEngineForTitle(cfg.RepoRoot, titleSlug, p.Gamertag, p.XUID, nil, nil).
		WithSessionOptionsLoader(settings_platform.SessionOptionsLoaderForConfig(cfg, titleSlug))
	return pass.run(ctx, engine, force)
}
