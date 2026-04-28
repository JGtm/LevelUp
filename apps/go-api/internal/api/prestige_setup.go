// Package api — câblage du module Prestige derrière le feature flag PRESTIGE_ENABLED.
//
// Au boot :
//  1. Charge tuning.toml (fallback DefaultTuning si absent)
//  2. Charge templates + preset arcs Halo dans metadata.duckdb
//  3. Construit une PrestigeFactory qui résout un Service par player_slug
//
// Au runtime :
//  - Routes derrière le flag (handlers.PrestigeHandler avec service factory)
//  - Hook sync RunPostSyncHook appelable depuis le sync engine

package api

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"sync"

	titlePkg "levelup/go-api/internal/domain/title"
	platform_duckdb "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/prestige"
)

// PrestigeBundle regroupe les ressources globales du module Prestige.
//
// Bases ouvertes :
//   - sharedSocialDB : pool partagé sur shared_social.duckdb (events, user_prestige, squad)
//   - metadataDB     : pool partagé sur metadata.duckdb (templates, preset_arcs)
//
// Le PlayerDB (par-joueur) est résolu à la demande via PlayerResolver.
type PrestigeBundle struct {
	tuning         prestige.Tuning
	sharedSocialDB *platform_duckdb.DB
	metadataDB     *platform_duckdb.DB
	socialRepo     *platform_duckdb.PrestigeSocialRepo
	squadRepo      *platform_duckdb.PrestigeSquadRepo
	squadChallRepo *platform_duckdb.PrestigeSquadChallengeRepo
	templateRepo   *platform_duckdb.PrestigeTemplateRepo
	presetArcRepo  *platform_duckdb.PrestigePresetArcRepo
	resolve        PlayerResolver
	mu             sync.Mutex
}

// NewPrestigeBundle initialise le bundle au boot.
//
// repoRoot = racine du repo (pour locate config/ et data/).
// resolve  = PlayerResolver pour ouvrir les player DB à la demande.
//
// Si une étape échoue (TOML absent, DB injoignable), retourne nil + error.
// L'appelant (server.go) doit décider de booter sans Prestige.
func NewPrestigeBundle(repoRoot string, resolve PlayerResolver) (*PrestigeBundle, error) {
	pr := titlePkg.NewPathResolver(repoRoot)
	titleSlug := titlePkg.DefaultSlug

	// 1. Charger tuning.toml (fallback géré dans LoadTuning)
	tuning := prestige.LoadTuning(filepath.Join(repoRoot, "config", "prestige", "tuning.toml"))

	// 2. Ouvrir shared_social.duckdb (partagé)
	sharedSocialDB, err := platform_duckdb.OpenReadWriteShared(pr.SharedSocialDBPath(titleSlug))
	if err != nil {
		return nil, errors.New("prestige: cannot open shared_social.duckdb: " + err.Error())
	}

	// 3. Ouvrir metadata.duckdb (partagé)
	metadataDB, err := platform_duckdb.OpenReadWriteShared(pr.MetadataDBPath(titleSlug))
	if err != nil {
		sharedSocialDB.Close()
		return nil, errors.New("prestige: cannot open metadata.duckdb: " + err.Error())
	}

	bundle := &PrestigeBundle{
		tuning:         tuning,
		sharedSocialDB: sharedSocialDB,
		metadataDB:     metadataDB,
		socialRepo:     platform_duckdb.NewPrestigeSocialRepo(sharedSocialDB),
		squadRepo:      platform_duckdb.NewPrestigeSquadRepo(sharedSocialDB),
		squadChallRepo: platform_duckdb.NewPrestigeSquadChallengeRepo(sharedSocialDB),
		templateRepo:   platform_duckdb.NewPrestigeTemplateRepo(metadataDB),
		presetArcRepo:  platform_duckdb.NewPrestigePresetArcRepo(metadataDB),
		resolve:        resolve,
	}

	// 4. Charger le catalogue Halo Infinite depuis TOML.
	// Best-effort : si le fichier est absent ou invalide, on log warn mais
	// on continue — le boot ne doit pas échouer pour absence de catalogue.
	bundle.loadHaloCatalog(repoRoot, titleSlug)

	slog.Info("prestige_bundle_initialized",
		"shared_social_path", pr.SharedSocialDBPath(titleSlug),
		"metadata_path", pr.MetadataDBPath(titleSlug),
		"feature_flag_enabled", prestige.IsEnabled(),
	)
	return bundle, nil
}

// loadHaloCatalog charge templates + preset arcs Halo dans la DB metadata.
func (b *PrestigeBundle) loadHaloCatalog(repoRoot, titleSlug string) {
	ctx := context.Background()

	templatesPath := filepath.Join(repoRoot, "config", "titles", titleSlug, "challenges", "templates.toml")
	if n, err := prestige.LoadTemplatesFromTOML(ctx, b.templateRepo, templatesPath); err != nil {
		slog.Warn("prestige_templates_load_failed", "title_slug", titleSlug, "err", err.Error())
	} else if n > 0 {
		slog.Info("prestige_templates_loaded", "title_slug", titleSlug, "count", n)
	}

	presetsPath := filepath.Join(repoRoot, "config", "titles", titleSlug, "arcs", "presets.toml")
	if n, err := prestige.LoadPresetArcsFromTOML(ctx, b.presetArcRepo, presetsPath); err != nil {
		slog.Warn("prestige_presets_load_failed", "title_slug", titleSlug, "err", err.Error())
	} else if n > 0 {
		slog.Info("prestige_presets_loaded", "title_slug", titleSlug, "count", n)
	}
}

// Close libère les pools de connexions.
func (b *PrestigeBundle) Close() {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.sharedSocialDB != nil {
		_ = b.sharedSocialDB.Close()
		b.sharedSocialDB = nil
	}
	if b.metadataDB != nil {
		_ = b.metadataDB.Close()
		b.metadataDB = nil
	}
}

// ServiceForPlayer construit un prestige.Service en injectant les repos
// par-joueur du PlayerDB résolu.
//
// Cette fonction est appelée à chaque requête Prestige nécessitant un
// player context (CRUD défis, arcs, EvaluateForUser). Les repos partagés
// (PrestigeRepo, SquadRepo, SquadChallengeRepo) sont réutilisés.
func (b *PrestigeBundle) ServiceForPlayer(ctx context.Context, playerSlug string) (prestige.Service, error) {
	if b == nil {
		return nil, errors.New("prestige: bundle not initialized")
	}
	pdb, err := b.resolve(ctx, playerSlug)
	if err != nil {
		return nil, errors.New("prestige: cannot resolve player: " + err.Error())
	}
	if pdb == nil || pdb.Player == nil {
		return nil, errors.New("prestige: player db not available")
	}

	deps := prestige.Deps{
		Tuning:           b.tuning,
		Challenges:       platform_duckdb.NewPrestigeChallengeRepo(pdb.Player),
		Arcs:             platform_duckdb.NewPrestigeArcRepo(pdb.Player),
		Moments:          platform_duckdb.NewPrestigeMomentCardRepo(pdb.Player),
		Prestige:         b.socialRepo,
		Telemetry:        platform_duckdb.NewPrestigeTelemetryRepo(pdb.Player),
		BaselineState:    platform_duckdb.NewPrestigeBaselineStateRepo(pdb.Player),
		Templates:        b.templateRepo,
		PresetArcs:       b.presetArcRepo,
		SquadChallenges:  b.squadChallRepo,
		Squads:           b.squadRepo,
		BaselineProvider: platform_duckdb.NewHaloBaselineProvider(pdb.Shared),
	}
	return prestige.NewService(deps), nil
}

// RunPostSync est le point d'entrée du sync engine pour ré-évaluer
// les défis actifs après ingestion de matchs.
//
// No-op si PRESTIGE_ENABLED=false. Best-effort : log les erreurs
// sans propager (le sync ne doit pas échouer à cause de Prestige).
func (b *PrestigeBundle) RunPostSync(ctx context.Context, playerSlug, titleSlug string) {
	if b == nil || !prestige.IsEnabled() {
		return
	}
	svc, err := b.ServiceForPlayer(ctx, playerSlug)
	if err != nil {
		slog.WarnContext(ctx, "prestige_post_sync_resolve_failed",
			"player_slug", playerSlug, "err", err.Error())
		return
	}
	prestige.RunPostSyncHook(ctx, svc, playerSlug, titleSlug)
}
