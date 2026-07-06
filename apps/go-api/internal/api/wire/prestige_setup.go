// Package api — câblage du module Prestige derrière le feature flag PRESTIGE_ENABLED.
//
// Au boot :
//  1. Charge tuning.toml (fallback DefaultTuning si absent)
//  2. Construit une PrestigeFactory qui résout un Service par player_slug
//
// Le catalogue (challenge_template, preset_arc) est seedé via la migration
// "seed_prestige_catalog_v1" — pas de chargement TOML au boot.
//
// Au runtime :
//  - Routes derrière le flag (handlers.PrestigeHandler avec service factory)
//  - Hook sync RunPostSyncHook appelable depuis le sync engine

package wire

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"sync"

	titlePkg "levelup/go-api/internal/domain/title"
	platform_duckdb "levelup/go-api/internal/platform/duckdb"
	prestigedb "levelup/go-api/internal/platform/duckdb/prestige"
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
	enabled        bool
	tuning         prestige.Tuning
	sharedSocialDB *platform_duckdb.DB
	metadataDB     *platform_duckdb.DB
	socialRepo     *prestigedb.PrestigeSocialRepo
	squadRepo      *prestigedb.PrestigeSquadRepo
	squadChallRepo *prestigedb.PrestigeSquadChallengeRepo
	templateRepo   *prestigedb.PrestigeTemplateRepo
	presetArcRepo  *prestigedb.PrestigePresetArcRepo
	resolve        PlayerResolver
	squadProfile   prestige.SquadProfileProvider
	mu             sync.Mutex
}

// NewPrestigeBundle initialise le bundle au boot.
//
// repoRoot = racine du repo (pour locate config/ et data/).
// resolve  = PlayerResolver pour ouvrir les player DB à la demande.
//
// Si une étape échoue (TOML absent, DB injoignable), retourne nil + error.
// L'appelant (server.go) doit décider de booter sans Prestige.
func NewPrestigeBundle(repoRoot string, resolve PlayerResolver, enabled bool) (*PrestigeBundle, error) {
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
		enabled:        enabled,
		tuning:         tuning,
		sharedSocialDB: sharedSocialDB,
		metadataDB:     metadataDB,
		socialRepo:     prestigedb.NewPrestigeSocialRepo(sharedSocialDB),
		squadRepo:      prestigedb.NewPrestigeSquadRepo(sharedSocialDB),
		squadChallRepo: prestigedb.NewPrestigeSquadChallengeRepo(sharedSocialDB),
		templateRepo:   prestigedb.NewPrestigeTemplateRepo(metadataDB),
		presetArcRepo:  prestigedb.NewPrestigePresetArcRepo(metadataDB),
		resolve:        resolve,
	}

	slog.Info("prestige_bundle_initialized",
		"shared_social_path", pr.SharedSocialDBPath(titleSlug),
		"metadata_path", pr.MetadataDBPath(titleSlug),
		"feature_flag_enabled", enabled,
	)
	return bundle, nil
}

// WithSquadProfile injecte le provider de profil d'escouade (axes de perf) pour
// le biais coach du pool. Optionnel ; sans lui RefreshSquadPool reste un shuffle.
// Chainable.
func (b *PrestigeBundle) WithSquadProfile(p prestige.SquadProfileProvider) *PrestigeBundle {
	if b != nil {
		b.squadProfile = p
	}
	return b
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

// TemplateRepoForCoach expose le PrestigeTemplateRepo du bundle pour le
// coach_advisor (ADR 0020 Phase 8). Le coach_advisor lit le catalogue de
// templates et peut UpsertOne un template synthétisé. Retourne nil si le
// bundle n'est pas initialisé.
func (b *PrestigeBundle) TemplateRepoForCoach() prestige.TemplateRepo {
	if b == nil {
		return nil
	}
	return b.templateRepo
}

// ServiceForPlayer construit un prestige.Service en injectant les repos
// par-joueur du PlayerDB résolu.
//
// Cette fonction est appelée à chaque requête Prestige nécessitant un
// player context (CRUD défis, arcs, EvaluateForUser). Les repos partagés
// (PrestigeRepo, SquadRepo, SquadChallengeRepo) sont réutilisés.
func (b *PrestigeBundle) ServiceForPlayer(ctx context.Context, playerSlug string) (prestige.Service, error) {
	_, svc, err := b.serviceAndPlayerDB(ctx, playerSlug)
	return svc, err
}

// serviceAndPlayerDB est la variante interne qui retourne aussi le *PlayerDB
// résolu — utilisé par LazyPrestigeService pour acquérir un *LeasedWriter
// avant les méthodes write (commit 2 du refactor leased-writer-enforcement).
func (b *PrestigeBundle) serviceAndPlayerDB(ctx context.Context, playerSlug string) (*platform_duckdb.PlayerDB, prestige.Service, error) {
	if b == nil {
		return nil, nil, errors.New("prestige: bundle not initialized")
	}
	pdb, err := b.resolve(ctx, playerSlug)
	if err != nil {
		return nil, nil, errors.New("prestige: cannot resolve player: " + err.Error())
	}
	if pdb == nil || pdb.Player == nil {
		return nil, nil, errors.New("prestige: player db not available")
	}

	deps := prestige.Deps{
		Tuning:           b.tuning,
		Challenges:       prestigedb.NewPrestigeChallengeRepo(pdb.Player),
		Arcs:             prestigedb.NewPrestigeArcRepo(pdb.Player),
		Moments:          prestigedb.NewPrestigeMomentCardRepo(pdb.Player),
		Prestige:         b.socialRepo,
		Telemetry:        prestigedb.NewPrestigeTelemetryRepo(pdb.Player),
		BaselineState:    prestigedb.NewPrestigeBaselineStateRepo(pdb.Player),
		Templates:        b.templateRepo,
		PresetArcs:       b.presetArcRepo,
		SquadChallenges:  b.squadChallRepo,
		Squads:           b.squadRepo,
		BaselineProvider: prestigedb.NewHaloBaselineProvider(pdb.SharedReadDB()),
		SquadMatches:     prestigedb.NewPrestigeSquadMatchProvider(pdb.SharedReadDB()),
		SquadProfile:     b.squadProfile,
	}
	return pdb, prestige.NewService(deps), nil
}

// RunPostSync est le point d'entrée du sync engine pour ré-évaluer
// les défis actifs après ingestion de matchs.
//
// No-op si PRESTIGE_ENABLED=false. Best-effort : log les erreurs
// sans propager (le sync ne doit pas échouer à cause de Prestige).
//
// ⚠️ Invariant deadlock-free (commit 7 db-concurrency) : RunPostSync est appelé
// par le sync engine **alors qu'il tient le lease player et shared_social**
// (cf. internal/sync/lease.go). Le service retourné par ServiceForPlayer ici
// est une instance **directe** (pas le wrapper LazyPrestigeService) — il
// n'acquiert pas de lease, ce qui évite le deadlock. Ne pas wrapper ce service
// avec un LazyPrestigeService configuré pour acquérir des leases sur
// EvaluateForUser tant qu'on n'a pas de propagation explicite du writer du
// sync engine au hook (commit futur).
func (b *PrestigeBundle) RunPostSync(ctx context.Context, playerSlug, titleSlug string) {
	if b == nil || !b.enabled {
		return
	}
	svc, err := b.ServiceForPlayer(ctx, playerSlug)
	if err != nil {
		slog.WarnContext(ctx, "prestige_post_sync_resolve_failed",
			"player_slug", playerSlug, "err", err)
		return
	}
	prestige.RunPostSyncHook(ctx, svc, playerSlug, titleSlug)
}
