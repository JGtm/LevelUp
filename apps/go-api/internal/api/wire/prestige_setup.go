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
// Bases ouvertes, TOUTES DEUX sur le titre par défaut (cf. le bloc « ÉPINGLAGE AU
// TITRE PAR DÉFAUT » de NewPrestigeBundle — contrainte, pas oubli) :
//   - sharedSocialDB : pool partagé sur shared_social.duckdb (events, user_prestige, squad)
//   - metadataDB     : pool partagé sur metadata.duckdb (templates, preset_arcs)
//
// Le PlayerDB (par-joueur) est résolu à la demande via PlayerResolver — lui EST
// per-titre (ServiceRegistry.resolve lit ctxkeys.TitleSlug) : arcs, défis, moments
// et baselines d'un joueur sont donc bien isolés par titre.
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
//
// ─── ÉPINGLAGE AU TITRE PAR DÉFAUT — constat daté 2026-07-25 (V721-14a / D-06) ───
//
// Les DEUX bases ouvertes ici le sont sur `titlePkg.DefaultSlug`, quel que soit le
// titre de la requête. Ce n'est PAS un oubli de câblage : c'est la conséquence de
// deux propriétés du modèle Prestige, qui rendent un bundle par titre impossible
// sans décision produit préalable.
//
//  1. metadata.duckdb — le CATALOGUE Prestige (`challenge_template`, `preset_arc`)
//     est un référentiel possédé par Halo Infinite : il est créé et seedé par
//     `internal/games/halo_infinite/migrations` depuis `config/titles/halo_infinite/
//     {challenges,arcs}/*.toml`. Le set de migrations d'un titre additionnel qui
//     possède son target metadata (cf. `TitleMigrationSet.OwnsTarget`) ne crée PAS
//     ces tables — c'est un invariant TESTÉ (anti-pollution, cf.
//     internal/games/halo_5/migrations/metadata_test.go). Ouvrir la metadata du
//     titre courant ferait donc échouer toute lecture de catalogue pour un titre
//     qui n'en a pas.
//
//  2. shared_social.duckdb — les entités cross-joueurs `squad` et `squad_member`
//     ne portent AUCUNE colonne `title_slug` (cf. steps_shared_social.go) : une
//     escouade n'appartient à aucun titre. Isoler shared_social par titre
//     scinderait les escouades elles-mêmes et orphelinerait les `squad_challenge`
//     (qui, eux, portent un title_slug).
//
// Portée du problème aujourd'hui : les lignes écrites ici portent le VRAI
// title_slug de la requête et toutes les lectures per-titre filtrent dessus — il
// n'y a donc pas de fusion sémantique entre titres. Ce qui est violé, c'est
// l'isolation PAR CHEMIN (ADR 0008) : les points de progression gagnés sur un
// titre additionnel atterrissent physiquement dans le shared_social d'Halo
// Infinite. Les données déjà écrites y resteraient après tout correctif de
// câblage — leur déplacement demanderait une migration dédiée.
//
// Ne pas « corriger » ce point en n'isolant qu'une des deux bases : l'analyse
// complète et les options chiffrées sont dans .ai/V7.2.1/PLAN_V721_NOTION_BATCH.md,
// lot V721-14a, découverte D-06.
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
		// Indice escouade : modes servis en FR canonique (mode_name_tr) prêts à
		// afficher, comme home/match-view — le sous-titre « surtout … » était en EN.
		// Playlists : résolution FR par playlist_id (asset_translations, même
		// résolveur que la page Carrière) — comble le trou "Quick Play"/"Big Team
		// Battle" dont playlist_name_fr est vide dans match_registry (V72-10 suite).
		//
		// LIMITATION ASSUMÉE (contre-revue V7.2, 2026-07-25) : ces traducteurs sont
		// câblés en dur sur le FR, sans tenir compte de la locale de la requête —
		// un client EN reçoit donc l'indice en français. C'est le comportement
		// FR-first DÉJÀ retenu sur les autres surfaces serveur qui résolvent des
		// libellés d'assets (home, match-view, historique : LoadModeTranslationsFR
		// / LoadAssetTranslationsFR n'ont pas de variante locale-aware). Rendre
		// l'indice locale-aware isolément le désalignerait du reste de l'app ; le
		// jour où la résolution d'assets devient locale-aware, ce point de câblage
		// suit le même mouvement — il n'a pas à le précéder.
		SquadMatches: prestigedb.NewPrestigeSquadMatchProvider(pdb.SharedReadDB()).
			WithModeTranslatorFR(platform_duckdb.NewSquadRepo(pdb).LoadModeTranslationsFR).
			WithPlaylistTranslatorFR(func(ctx context.Context, ids []string) (map[string]string, error) {
				return platform_duckdb.NewSquadRepo(pdb).LoadAssetTranslationsFR(ctx, "playlist", ids)
			}),
		SquadProfile: b.squadProfile,
	}
	return pdb, prestige.NewService(deps), nil
}

// RunPostSync est le point d'entrée des chemins de sync (V1 via
// SyncEngine.WithPrestigeHook câblé par scheduler.BuildEngine + handler
// newEngineFor ; V2 via CycleOrchestratorImpl.WithPrestigeHook) pour ré-évaluer
// les défis actifs après ingestion de matchs. Câblage réel depuis VF-1 (2026-07-06).
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
