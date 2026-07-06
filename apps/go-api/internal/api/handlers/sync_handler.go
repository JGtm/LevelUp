// Package handlers — sync_handler.go : lancement de la sync initiale (Sprint 17-18).
//
// POST /sync/initial → crée un job "initial_sync" et retourne 202 immédiatement.
// Règles :
//   - 403 si can_start_initial_sync=false dans app_settings.json
//   - 409 si un job initial_sync actif existe déjà pour ce player_slug
//   - Exclusivité stricte : 1 seule sync active par joueur
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	gosync "sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_5/livesync"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/observability/logging"
	auth_platform "levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/platform/jobs"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/service"
	go_sync "levelup/go-api/internal/sync"
)

// syncJobTimeout borne la durée d'un job de sync HTTP (D3-04, revue 2026-06-01).
// Sans borne, le bgCtx context.Background() faisait attendre le lease KindPlayer
// indéfiniment si un autre writer (auto-sync, watcher) le tenait. 30 min couvre
// largement un sync initial volumineux + post-sync.
const syncJobTimeout = 30 * time.Minute

// syncErrorMessage (D3-05) produit un message de job lisible — distingue la
// contention de lease (un autre sync du même joueur est en cours, réessayable)
// d'un échec réel.
func syncErrorMessage(err error) string {
	if errors.Is(err, dblease.ErrDBLocked) {
		return "db_busy: un autre sync de ce joueur est en cours (réessayer) — " + err.Error()
	}
	return err.Error()
}

// NotificationsEmitterFactory construit un emitter de notifications pour un slug
// joueur donné. Optionnel : si le SyncHandler n'a pas reçu de factory, aucun
// hook de notification n'est émis (pratique pour les tests et le bootstrap initial).
type NotificationsEmitterFactory func(ctx context.Context, slug string) (notifications.Emitter, error)

// PostSyncDeltaHook est un hook closure-based pour la détection delta autour
// d'une exécution de sync. L'appelant invoque le hook avant la sync ; il reçoit
// une fonction "after" à invoquer en cas de succès. Le tout reste opaque côté
// handler (les types PlayerSnapshot vivent dans internal/api/, pas ici).
type PostSyncDeltaHook func(ctx context.Context, slug string) (after func(ctx context.Context))

// SyncHandler gère les endpoints de synchronisation des données Halo.
type SyncHandler struct {
	cfg           *config.AppConfig
	settingsStore *settings_platform.Store
	jobStore      *jobs.Store
	provider      auth_platform.TokenProvider
	notifierFor   NotificationsEmitterFactory // optionnel
	postSync      PostSyncDeltaHook           // optionnel : season_pass_level / objective_completed / challenge_completed
	// prestigeHook (optionnel) ré-évalue les défis Prestige actifs après ingestion.
	// Injecté par server.go = PrestigeBundle.RunPostSync (no-op si flag off ou bundle
	// nil). Câblé sur le SyncEngine construit par le handler (newEngineFor) via
	// SyncEngine.WithPrestigeHook → fire à engine.run() après le post-sync, pendant
	// que le lease est encore tenu (instance directe non-lease : cf. prestige_setup.go
	// invariant deadlock-free). Le chemin delta/auto-sync passe par BuildEngine, câblé
	// séparément côté scheduler (parité runtime).
	prestigeHook PrestigeHook
	// syncGate déduplique les syncs cross-source (cf. go_sync.SyncGate). Un sync
	// HTTP cède si le joueur est déjà en vol (watcher ou auto-sync) : pré-check
	// IsInFlight → 409 à la requête, et TryClaim dans la goroutine pour la garantie
	// race-safe. Par défaut NopSyncGate ; main.go injecte le Coordinator partagé.
	syncGate go_sync.SyncGate
	// serverCtx est le ctx de vie du serveur (annulé au shutdown). Les goroutines
	// de sync HTTP en dérivent leur bgCtx au lieu de context.Background() : à
	// l'arrêt, le RunDelta en cours est annulé, libère le lease + le claim, et
	// WaitInFlight() peut rendre la main avant duckdb.CloseAll() (sinon write-after
	// -close #7659). Par défaut context.Background() (comportement legacy).
	serverCtx context.Context
	// engineBuilder construit le moteur du sync manuel delta (StartSyncAll /
	// StartDeltaSync) via le MÊME wiring que l'auto-sync (AutoSyncScheduler.
	// BuildEngine → PooledHaloClient du pool partagé). Injecté par server.go.
	// Nil → fallback legacy newEngineFor (session tokens). Cf. EngineBuilder.
	engineBuilder EngineBuilder
	// cooldown : fenêtre anti-spam du sync manuel delta. 0 = désactivé (tests).
	// lastManualAt mémorise l'instant du dernier déclenchement par clé ("delta:"+
	// slug pour /players/{slug}/sync, "all" pour /sync/all), protégé par cooldownMu.
	cooldown     time.Duration
	cooldownMu   gosync.Mutex
	lastManualAt map[string]time.Time
}

// NewSyncHandler crée un SyncHandler.
func NewSyncHandler(
	cfg *config.AppConfig,
	settingsStore *settings_platform.Store,
	jobStore *jobs.Store,
	provider auth_platform.TokenProvider,
) *SyncHandler {
	return &SyncHandler{
		cfg:           cfg,
		settingsStore: settingsStore,
		jobStore:      jobStore,
		provider:      provider,
		syncGate:      go_sync.NopSyncGate{}, // défaut no-op (pas de dédup tant que non injecté)
		serverCtx:     context.Background(),  // défaut : pas d'annulation au shutdown
		cooldown:      defaultManualSyncCooldown,
		lastManualAt:  make(map[string]time.Time),
	}
}

// WithSyncGate branche le gate de déduplication cross-source (le Coordinator
// partagé du watcher). À appeler depuis server.go quand le watcher est actif.
// Sans appel : NopSyncGate (le lease reste le seul rempart cross-source).
func (h *SyncHandler) WithSyncGate(g go_sync.SyncGate) *SyncHandler {
	if g != nil {
		h.syncGate = g
	}
	return h
}

// WithServerContext branche le ctx de vie du serveur (annulé au shutdown) dont
// les syncs HTTP dérivent leur bgCtx. Permet d'annuler proprement les RunDelta en
// cours à l'arrêt. Sans appel : context.Background() (legacy).
func (h *SyncHandler) WithServerContext(ctx context.Context) *SyncHandler {
	if ctx != nil {
		h.serverCtx = ctx
	}
	return h
}

// WithNotificationsEmitterFactory branche la factory d'émetteurs de notifications.
// À appeler depuis server.go après création du ServiceRegistry.
//
// L'émission est best-effort : toute erreur est loguée et ne propage pas. Le
// hook est invoqué post-RunDelta (succès → match_synced, erreur → sync_error).
func (h *SyncHandler) WithNotificationsEmitterFactory(f NotificationsEmitterFactory) *SyncHandler {
	h.notifierFor = f
	return h
}

// WithPostSyncDeltaHook branche le hook delta-detection (season_pass_level,
// objective_completed, challenge_completed). Best-effort.
func (h *SyncHandler) WithPostSyncDeltaHook(hook PostSyncDeltaHook) *SyncHandler {
	h.postSync = hook
	return h
}

// PrestigeHook est la signature du hook post-sync injecté pour le module Prestige.
// La logique métier vit dans internal/api/wire/prestige_setup.go (RunPostSync).
// L'identifiant attendu est le playerSlug (= user_id des défis Prestige) ; pour
// les joueurs réels PlayerSlug == Gamertag (config_players.go).
type PrestigeHook func(ctx context.Context, playerSlug, titleSlug string)

// newEngineFor instancie un SyncEngine pré-câblé avec le loader friends
// (settings.FriendGamertags), pour que le hook auto-recompute is_with_friends
// post-sync delta soit toujours actif sur les syncs déclenchés par cet handler.
func (h *SyncHandler) newEngineFor(titleSlug, gamertag, xuid string, tokens *domain.HaloTokens) *go_sync.SyncEngine {
	loader := func() ([]string, error) {
		s, err := h.settingsStore.Load()
		if err != nil {
			return nil, err
		}
		return s.FriendGamertags, nil
	}
	// Title-aware (MT-11 / PMT-3) : le moteur écrit dans data/titles/{titleSlug}/...
	// au lieu de halo_infinite systématique (corrige le sync initial multi-titre).
	engine := go_sync.NewSyncEngineForTitle(h.cfg.RepoRoot, titleSlug, gamertag, xuid, tokens, h.provider).
		WithFriendsLoader(loader).
		WithCSRSeasonID(h.cfg.CurrentCSRSeasonID)
	// Sprint B1 commit 11b : aligner le sync HTTP-triggered sur auto_sync — sans
	// ce câblage, un user qui clique "Sync now" déclenche un sync en mode legacy
	// (OpenSharedDB direct RW) qui court-circuite la coordination Provider ↔ pool
	// joueur. Résultat : Catalog Error / "different configuration" pour les
	// readers HTTP pendant cette fenêtre. Wire identique à scheduler/auto_sync.go.
	if h.cfg.SharedProvider != nil {
		engine = engine.WithSharedProvider(h.cfg.SharedProvider)
	}
	// Media scan post-sync : cohérent avec AutoSyncScheduler.defaultRunnerFactory.
	// timezone REQUISE pour parseCaptureTimeFromFilename (cf. media_index_service).
	if h.settingsStore != nil {
		engine = engine.WithMediaScanHook(service.BuildMediaScanHook(h.cfg.RepoRoot, gamertag,
			func() string {
				s, _ := h.settingsStore.Load()
				if s != nil {
					return s.MediaCapturesBaseDir
				}
				return ""
			},
			func() string {
				s, _ := h.settingsStore.Load()
				if s != nil {
					return s.UserTimezone
				}
				return ""
			},
		))
	}
	// Hook Prestige post-sync (best-effort). Fire à engine.run() après le pipeline
	// post-sync, pendant que le lease player/shared est encore tenu (instance directe
	// non-lease : invariant deadlock-free, cf. wire/prestige_setup.go). L'engine fournit
	// le gamertag ; pour les joueurs réels PlayerSlug == Gamertag → identifiant correct
	// pour RunPostSync/EvaluateForUser (filtre UserID des défis).
	if h.prestigeHook != nil {
		engine = engine.WithPrestigeHook(func(hookCtx context.Context, playerSlug, ts string) {
			h.prestigeHook(hookCtx, playerSlug, ts)
		})
	}
	return engine
}

// deltaRunner = surface commune *go_sync.SyncEngine / *livesync.Runner (RunDelta).
// Permet de brancher un runner live-sync DÉDIÉ (Halo 5) à l'entonnoir HTTP de sync
// sans toucher le chemin Infinite (SyncEngine reste byte-identique).
type deltaRunner interface {
	RunDelta(ctx context.Context, opts domain.SyncOptions) (domain.SyncResult, error)
}

// runnerFor sélectionne (REGISTRY-DRIVEN, jamais slug==) le runner d'un sync + le
// ctx d'exécution. Titre live-only (Halo 5) → runner dédié livesync + ctx portant le
// SpartanToken (l'adapter h5 le lit du ctx, là où Infinite reçoit les tokens en
// argument explicite). Sinon → SyncEngine Infinite + ctx inchangé.
func (h *SyncHandler) runnerFor(ctx context.Context, titleSlug, gamertag, xuid string, tokens *domain.HaloTokens) (deltaRunner, context.Context) {
	if r := livesync.RunnerForTitle(titleSlug, h.cfg, gamertag, xuid); r != nil {
		return r, ctxkeys.WithHaloAuth(ctx, tokens, xuid)
	}
	return h.newEngineFor(titleSlug, gamertag, xuid, tokens), ctx
}

// WithPrestigeHook stocke le hook Prestige post-sync. Le handler le câble sur
// chaque SyncEngine qu'il construit via newEngineFor (StartInitialSync). Le chemin
// delta/auto-sync (BuildEngine) est câblé côté scheduler (parité runtime). No-op si
// le hook est nil (bundle absent ou flag Prestige off côté RunPostSync).
func (h *SyncHandler) WithPrestigeHook(hook PrestigeHook) *SyncHandler {
	h.prestigeHook = hook
	return h
}

// emitMatchSynced émet une notification agrégée match_synced si > 0 matchs insérés.
func (h *SyncHandler) emitMatchSynced(ctx context.Context, slug string, inserted int) {
	if h.notifierFor == nil || inserted <= 0 {
		return
	}
	em, err := h.notifierFor(ctx, slug)
	if err != nil || em == nil {
		slog.WarnContext(ctx, "notifications: emitter factory failed", "slug", slug, "err", err)
		return
	}
	if err := em.Emit(ctx, notifications.EmitInput{
		Category: notifications.CategoryMatchSynced,
		Severity: notifications.SeveritySuccess,
		TitleKey: "notif.match_synced.title",
		BodyKey:  "notif.match_synced.body",
		Params:   map[string]any{jsonKeyCount: inserted},
		Source:   "sync_handler",
	}); err != nil {
		slog.WarnContext(ctx, "notifications: match_synced emit", "err", err)
	}
}

// emitSyncError émet une notification sync_error sur échec d'une sync.
func (h *SyncHandler) emitSyncError(ctx context.Context, slug, jobID, message string) {
	if h.notifierFor == nil {
		return
	}
	em, err := h.notifierFor(ctx, slug)
	if err != nil || em == nil {
		return
	}
	if err := em.Emit(ctx, notifications.EmitInput{
		Category:    notifications.CategorySyncError,
		Severity:    notifications.SeverityError,
		TitleKey:    "notif.sync_error.title",
		BodyKey:     "notif.sync_error.body",
		Params:      map[string]any{"message": truncate(message, 200), "job_id": jobID},
		TargetRoute: fmt.Sprintf("/players/%s/sync", slug),
		Source:      "sync_handler",
	}); err != nil {
		slog.WarnContext(ctx, "notifications: sync_error emit", "err", err)
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// ─── Montage Huma + types Input/Output ───────────────────────────────────────

// syncJobOutput : 202 Accepted, corps = snapshot du job de sync créé (les 3
// routes partagent ce contrat). Status override la valeur par défaut Huma.
type syncJobOutput struct {
	Status int
	Body   *domain.AsyncJobStatus
}

// syncInitialInput : corps brut décodé maison. RawBody (pas Body typé) → préserve
// le contrat 400 {invalid_body} sur JSON invalide (un Body typé renverrait le 422
// de validation Huma).
type syncInitialInput struct {
	RawBody []byte
}

// syncDeltaInput : path param {player_slug} (l'original lisait r.PathValue).
type syncDeltaInput struct {
	PlayerSlug string `path:"player_slug"`
}

// MountInitialAndAll enregistre les routes admin /sync/initial et /sync/all via
// Huma sur le routeur chi fourni (middleware RequireAuth/RequireAdmin hérités du
// groupe admin où server.go le monte).
func (h *SyncHandler) MountInitialAndAll(r chi.Router) {
	api := humacore.NewAPI(r)
	huma.Post(api, "/sync/initial", h.StartInitialSync)
	huma.Post(api, "/sync/all", h.StartSyncAll)
}

// MountDelta enregistre la route /sync via Huma sur le sous-routeur chi (préfixe
// /players/{player_slug} + middleware hérités — lit {player_slug} parent).
func (h *SyncHandler) MountDelta(r chi.Router) {
	api := humacore.NewAPI(r)
	huma.Post(api, "/sync", h.StartDeltaSync)
}

// StartInitialSync lance la sync initiale pour un joueur.
// POST /sync/initial -> 202 AsyncJobStatus.
func (h *SyncHandler) StartInitialSync(ctx context.Context, in *syncInitialInput) (*syncJobOutput, error) {
	appCfg, err := h.settingsStore.Load()
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "settings_load_error", "Impossible de charger la configuration.")
	}
	if !appCfg.CanStartInitialSync {
		return nil, humacore.NewError(http.StatusForbidden, "initial_sync_disabled",
			"Le lancement d'une sync initiale est désactivé sur cette instance.")
	}

	var req domain.InitialSyncStartRequest
	if err := json.Unmarshal(in.RawBody, &req); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", "Corps de requete JSON invalide.")
	}

	if req.PlayerSlug == "" || len(req.PlayerSlug) > 50 {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_player_slug", "player_slug vide ou trop long.")
	}

	// Titre cible : explicite (req.TitleSlug) sinon titre du contexte (header/session).
	titleSlug := req.TitleSlug
	if titleSlug == "" {
		titleSlug = ctxkeys.TitleSlug(ctx)
	}
	bodyMaxOmitted := req.MaxMatches == 0

	// Sprint B1 commit 17 : event_id pour tracer le sync initial (breadcrumb logs HTTP).
	_, evID := logging.WithEvent(ctx, "http.sync.initial:"+req.PlayerSlug)
	slog.InfoContext(ctx, "sync_handler: StartInitialSync démarré",
		"player_slug", req.PlayerSlug, "title_slug", titleSlug, "max_matches", req.MaxMatches, "event", evID)

	// Bornage : un body explicite hors bornes est rejeté d'emblée. Le défaut du
	// profil (initial_max_matches) est appliqué APRÈS résolution du profil.
	if req.MaxMatches == 0 {
		req.MaxMatches = 200
	}
	if req.MaxMatches < 1 || req.MaxMatches > 2000 {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_max_matches", "max_matches doit etre entre 1 et 2000.")
	}

	if active := h.jobStore.FindActiveInitialSync(req.PlayerSlug); active != nil {
		return nil, humacore.NewError(http.StatusConflict, "sync_already_active",
			"Une sync initiale est deja en cours pour ce joueur.")
	}

	sess := middleware.GetSession(ctx)
	if sess == nil || sess.HaloTokens == nil {
		return nil, humacore.NewError(http.StatusUnauthorized, "auth_required",
			"Tokens Halo absents.")
	}
	tokens := sess.HaloTokens

	// Résoudre le couple (gamertag, xuid) + nb de matchs initiaux du profil POUR CE
	// TITRE — lève l'ambiguïté quand un gamertag existe sous plusieurs titres.
	players, err := h.cfg.LoadPlayers(titleSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "profiles_load_error", "Impossible de charger db_profiles.json.")
	}
	var gamertag, xuid string
	var profileMaxMatches int
	for _, p := range players {
		if p.PlayerSlug == req.PlayerSlug {
			gamertag = p.Gamertag
			xuid = p.XUID
			profileMaxMatches = p.InitialMaxMatches
			break
		}
	}
	if gamertag == "" {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found",
			fmt.Sprintf("Joueur %q introuvable pour le titre %q dans db_profiles.json.", req.PlayerSlug, titleSlug))
	}

	// Défaut profil : body omis + initial_max_matches du profil → l'utiliser
	// (clampé 1..2000) plutôt que le 200 générique.
	if bodyMaxOmitted && profileMaxMatches > 0 {
		req.MaxMatches = profileMaxMatches
		if req.MaxMatches > 2000 {
			req.MaxMatches = 2000
		}
	}

	// Dédup cross-source : claim SYNCHRONE avant de créer le job. Si un sync de ce
	// joueur est déjà en vol (watcher ou auto-sync), 409 sans créer de job —
	// cohérent avec le 409 FindActiveInitialSync (même-source). Le claim étant posé
	// AVANT le retour du handler, son gateWG.Add a un happens-before avec le retour
	// → il est garanti vu par WaitInFlight au shutdown (pas de goroutine détachée
	// qui claim tardivement). release est passé à la goroutine (defer release()).
	release, claimed := h.syncGate.TryClaimT(titleSlug, gamertag)
	if !claimed {
		return nil, humacore.NewError(http.StatusConflict, "sync_already_active",
			"Une synchronisation de ce joueur est déjà en cours (watcher ou auto-sync).")
	}

	job := h.jobStore.Create(domain.JobTypeInitialSync, req.PlayerSlug)
	// Snapshot avant le go func() : la goroutine modifie in-place le job dans le store.
	jobSnapshot := *job

	go func() {
		defer release()
		bgCtx, cancel := context.WithTimeout(h.serverCtx, syncJobTimeout)
		defer cancel()
		var after func(ctx context.Context)
		if h.postSync != nil {
			after = h.postSync(bgCtx, req.PlayerSlug)
		}
		opts := domain.DefaultSyncOptions()
		opts.MaxMatches = req.MaxMatches

		// Sélection registry-driven : Halo 5 (live-only) → runner dédié, sinon Infinite.
		runner, runCtx := h.runnerFor(bgCtx, titleSlug, gamertag, xuid, tokens)
		result, err := runner.RunDelta(runCtx, opts)
		if err != nil {
			errMsg := syncErrorMessage(err)
			h.jobStore.SetStatus(job.JobID, domain.JobStatusFailed, &errMsg)
			h.emitSyncError(bgCtx, req.PlayerSlug, job.JobID, errMsg)
			return
		}
		summary := fmt.Sprintf("inserted=%d skipped=%d medals=%d duration=%.1fs status=%s",
			result.MatchesInserted, result.MatchesSkipped, result.MedalsInserted,
			result.DurationSeconds, result.Status())
		h.jobStore.SetStatus(job.JobID, domain.JobStatusSucceeded, &summary)
		h.emitMatchSynced(bgCtx, req.PlayerSlug, result.MatchesInserted)
		if after != nil {
			after(bgCtx)
		}
	}()

	return &syncJobOutput{Status: http.StatusAccepted, Body: &jobSnapshot}, nil
}

// StartDeltaSync lance une synchronisation delta pour un joueur donné.
// POST /api/v1/players/{player_slug}/sync → 202 AsyncJobStatus.
// Contrairement à StartInitialSync, cette route n'est pas protégée par can_start_initial_sync.
func (h *SyncHandler) StartDeltaSync(ctx context.Context, in *syncDeltaInput) (*syncJobOutput, error) {
	playerSlug := in.PlayerSlug
	if playerSlug == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_player_slug", "player_slug manquant.")
	}

	// Session requise + cooldown anti-spam (clé = slug). Tokens Halo via le pool.
	if err := h.guardManualDeltaSync(ctx, "delta:"+playerSlug); err != nil {
		return nil, err
	}

	if active := h.jobStore.FindActiveInitialSync(playerSlug); active != nil {
		return nil, humacore.NewError(http.StatusConflict, "sync_already_active",
			"Une synchronisation est déjà en cours pour ce joueur.")
	}

	// Sprint B1 commit 17 : event_id pour tracer le sync delta HTTP-triggered.
	_, evID := logging.WithEvent(ctx, "http.sync.delta:"+playerSlug)
	slog.InfoContext(ctx, "sync_handler: StartDeltaSync démarré",
		"player_slug", playerSlug, "event", evID)

	players, err := h.cfg.LoadPlayers()
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "profiles_load_error", "Impossible de charger db_profiles.json.")
	}
	var gamertag, xuid string
	for _, p := range players {
		if p.PlayerSlug == playerSlug {
			gamertag = p.Gamertag
			xuid = p.XUID
			break
		}
	}
	if gamertag == "" {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found",
			fmt.Sprintf("Joueur %q introuvable dans db_profiles.json.", playerSlug))
	}

	// Dédup cross-source (cf. StartInitialSync) : claim SYNCHRONE, 409 si déjà en vol.
	release, claimed := h.syncGate.TryClaimT(ctxkeys.TitleSlug(ctx), gamertag)
	if !claimed {
		return nil, humacore.NewError(http.StatusConflict, "sync_already_active",
			"Une synchronisation de ce joueur est déjà en cours (watcher ou auto-sync).")
	}

	job := h.jobStore.Create(domain.JobTypeInitialSync, playerSlug)
	// Snapshot avant le go func() : la goroutine modifie in-place le job dans le store.
	jobSnapshot2 := *job

	go func() {
		defer release()
		bgCtx, cancel := context.WithTimeout(h.serverCtx, syncJobTimeout)
		defer cancel()
		var after func(ctx context.Context)
		if h.postSync != nil {
			after = h.postSync(bgCtx, playerSlug)
		}
		engine := h.newPooledEngine(bgCtx, gamertag, xuid)
		opts := domain.DefaultSyncOptions()

		result, err := engine.RunDelta(bgCtx, opts)
		if err != nil {
			errMsg := syncErrorMessage(err)
			h.jobStore.SetStatus(job.JobID, domain.JobStatusFailed, &errMsg)
			h.emitSyncError(bgCtx, playerSlug, job.JobID, errMsg)
			return
		}
		summary := fmt.Sprintf("inserted=%d skipped=%d medals=%d duration=%.1fs status=%s",
			result.MatchesInserted, result.MatchesSkipped, result.MedalsInserted,
			result.DurationSeconds, result.Status())
		h.jobStore.SetStatus(job.JobID, domain.JobStatusSucceeded, &summary)
		h.emitMatchSynced(bgCtx, playerSlug, result.MatchesInserted)
		if after != nil {
			after(bgCtx)
		}
	}()

	return &syncJobOutput{Status: http.StatusAccepted, Body: &jobSnapshot2}, nil
}

// StartSyncAll lance une synchronisation delta pour tous les joueurs configurés.
// POST /api/v1/sync/all → 202 AsyncJobStatus.
func (h *SyncHandler) StartSyncAll(ctx context.Context, _ *struct{}) (*syncJobOutput, error) {
	// Session requise + cooldown anti-spam (clé "all"). Tokens Halo via le pool
	// (même mécanique que l'auto-sync, ADR 0023), pas la session.
	if err := h.guardManualDeltaSync(ctx, "all"); err != nil {
		return nil, err
	}

	players, err := h.cfg.LoadPlayers()
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "profiles_load_error", "Impossible de charger db_profiles.json.")
	}
	if len(players) == 0 {
		return nil, humacore.NewError(http.StatusNotFound, "no_players", "Aucun joueur configuré dans db_profiles.json.")
	}

	// Sprint B1 commit 17 : event_id pour tracer le sync de tous les joueurs.
	_, evID := logging.WithEvent(ctx, "http.sync.all")
	slog.InfoContext(ctx, "sync_handler: StartSyncAll démarré",
		"player_count", len(players), "event", evID)

	job := h.jobStore.Create(domain.JobTypeDeltaSyncAll, "all")
	// Snapshot avant le go func() : la goroutine modifie in-place le job dans le store.
	jobSnapshot3 := *job

	go func() {
		total := len(players)
		var succeeded, failed, coalesced int
		for i, p := range players {
			// Shutdown : schedulerCtx annulé (cancelScheduler) → cesser de lancer de
			// nouveaux RunDelta. Le claim étant posé/relâché par joueur, gateWG
			// retombe à 0 entre joueurs ; sans ce break, WaitInFlight pourrait rendre
			// la main puis CloseAll pendant qu'on démarre le joueur suivant
			// (write-after-close). closing + ce break ferment la fenêtre.
			if h.serverCtx.Err() != nil {
				break
			}
			step := fmt.Sprintf("%s (%d/%d)", p.Gamertag, i+1, total)
			h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
				j.CurrentStep = &step
				pct := (i * 100) / total
				j.ProgressPct = &pct
			})

			// Dédup cross-source par joueur : si déjà en vol (watcher/auto), on
			// saute ce joueur (il est synchronisé par l'autre source) sans le
			// compter en échec. IIFE + defer release() → libération garantie même
			// sur panic de RunDelta.
			func() {
				release, claimed := h.syncGate.TryClaimT(p.TitleSlug, p.Gamertag)
				if !claimed {
					coalesced++
					slog.DebugContext(h.serverCtx, "sync_handler: joueur coalescé dans sync/all (déjà en vol via une autre source)",
						"gamertag", p.Gamertag)
					return
				}
				defer release()
				engine := h.newPooledEngine(h.serverCtx, p.Gamertag, p.XUID)
				opts := domain.DefaultSyncOptions()
				// D3-04 : timeout par joueur (borne l'attente du lease KindPlayer).
				pCtx, cancel := context.WithTimeout(h.serverCtx, syncJobTimeout)
				_, perr := engine.RunDelta(pCtx, opts)
				cancel()
				if perr != nil {
					failed++
				} else {
					succeeded++
				}
			}()
		}

		summary := fmt.Sprintf("players=%d succeeded=%d failed=%d coalesced=%d", total, succeeded, failed, coalesced)
		if failed > 0 {
			h.jobStore.SetStatus(job.JobID, domain.JobStatusFailed, &summary)
		} else {
			h.jobStore.SetStatus(job.JobID, domain.JobStatusSucceeded, &summary)
		}
	}()

	return &syncJobOutput{Status: http.StatusAccepted, Body: &jobSnapshot3}, nil
}
