// Package handlers — setup.go : création de profil joueur + smoke test (Sprint 16).
//
// POST /setup/players    → crée un profil joueur dans db_profiles.json (201)
// POST /setup/smoke-test → lance une vérification basique de l'environnement (202)
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le routeur chi
// (mêmes points de montage /setup/players et /setup/smoke-test) et enregistre les
// 2 POST via huma.Post. Logique métier inchangée (ProfileService + jobStore), seul
// le wrapping HTTP change. Le corps de POST /setup/players est lu via RawBody +
// json.Unmarshal maison (et marqué OPTIONNEL) pour reproduire EXACTEMENT le contrat
// d'origine : un corps absent OU malformé renvoie 400 {invalid_body} (l'ancien
// json.NewDecoder(r.Body).Decode renvoyait io.EOF sur corps vide → 400 invalid_body),
// PAS le « request body is required » de Huma ni son 422 de validation.
package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/jobs"
	session_platform "levelup/go-api/internal/platform/session"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/port"
)

// SetupHandler gère les endpoints de mise en place du profil joueur.
type SetupHandler struct {
	cfg           *config.AppConfig
	sessionStore  *session_platform.Store
	settingsStore *settings_platform.Store
	jobStore      *jobs.Store
	profileSvc    port.ProfileService
}

// NewSetupHandler crée un SetupHandler.
func NewSetupHandler(
	cfg *config.AppConfig,
	sessionStore *session_platform.Store,
	settingsStore *settings_platform.Store,
	jobStore *jobs.Store,
	profileSvc port.ProfileService,
) *SetupHandler {
	return &SetupHandler{
		cfg:           cfg,
		sessionStore:  sessionStore,
		settingsStore: settingsStore,
		jobStore:      jobStore,
		profileSvc:    profileSvc,
	}
}

// Mount enregistre les 2 routes via Huma sur le routeur chi (mêmes points de
// montage que les routes chi d'origine). Le body POST /setup/players est marqué
// OPTIONNEL (MarkRequestBodyOptional) : le décodage maison rend un corps absent
// OU malformé en 400 {invalid_body}, contrat préservé.
func (h *SetupHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/setup/players", h.handleCreatePlayer)
	humacore.MarkRequestBodyOptional(api, http.MethodPost, "/setup/players")
	huma.Post(api, "/setup/smoke-test", h.handleSmokeTest)
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// setupCreatePlayerInput : corps brut décodé maison (RawBody + body marqué
// OPTIONNEL) → corps absent ou JSON malformé ⇒ 400 {invalid_body}.
type setupCreatePlayerInput struct {
	RawBody []byte
}

// setupCreatePlayerOutput : 201 CreatePlayerProfileResponse.
type setupCreatePlayerOutput struct {
	Status int
	Body   domain.CreatePlayerProfileResponse
}

// setupSmokeTestOutput : 202 Accepted, corps = snapshot du job créé.
type setupSmokeTestOutput struct {
	Status int
	Body   domain.AsyncJobStatus
}

// ─── Endpoints ───────────────────────────────────────────────────────────────

// handleCreatePlayer crée un profil joueur dans db_profiles.json.
// POST /setup/players → 201 CreatePlayerProfileResponse.
//
// Guards :
//   - 403 si can_self_provision=false dans app_settings.json
//   - 403 si instance verrouillée (env LEVELUP_INSTANCE_LOCKED ou app_settings)
//   - 409 si profile_mode="xbox" mais aucune identité Halo liée en session
//   - 409 si gamertag/XUID ne correspond pas à l'identité Halo liée
func (h *SetupHandler) handleCreatePlayer(ctx context.Context, in *setupCreatePlayerInput) (*setupCreatePlayerOutput, error) {
	// Guard : can_self_provision
	appCfg, err := h.settingsStore.Load()
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "settings_load_error", "Impossible de charger la configuration.")
	}
	if !appCfg.CanSelfProvision {
		return nil, humacore.NewError(http.StatusForbidden, "provisioning_disabled",
			"L'auto-provisioning est désactivé sur cette instance.")
	}

	// Guard : instance fermée (lockdown) — pas de nouvelle BDD joueur.
	// Verrou effectif = env (LEVELUP_INSTANCE_LOCKED) OU app_settings.instance_locked.
	if h.cfg.InstanceLocked || appCfg.InstanceLocked {
		slog.WarnContext(ctx, "setup: création profil refusée — instance verrouillée",
			"env_locked", h.cfg.InstanceLocked, "settings_locked", appCfg.InstanceLocked)
		return nil, humacore.NewError(http.StatusForbidden, "instance_locked",
			"Cette instance est fermée : la création de nouveaux profils est désactivée.")
	}

	var req domain.CreatePlayerProfileRequest
	if err := json.Unmarshal(in.RawBody, &req); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", "Corps de requête JSON invalide.")
	}

	req.Gamertag = strings.TrimSpace(req.Gamertag)
	if req.Gamertag == "" || len(req.Gamertag) > 50 {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_gamertag", "Le gamertag est vide ou trop long.")
	}
	if req.ProfileMode == "" {
		req.ProfileMode = authModeXbox
	}

	// Titre cible : priorité au body (onboarding multi-titre — le front crée un
	// profil par titre choisi, avec son initial_max_matches) sinon le titre du
	// contexte (header/session), sinon le titre par défaut.
	titleSlug := strings.TrimSpace(req.TitleSlug)
	if titleSlug == "" {
		titleSlug = ctxkeys.TitleSlug(ctx)
	}
	if titleSlug == "" {
		titleSlug = title.DefaultSlug
	}
	req.TitleSlug = titleSlug

	// Guard : identité Xbox liée (mode xbox uniquement)
	if req.ProfileMode == "xbox" {
		sess := middleware.GetSession(ctx)
		if sess != nil && sess.LinkedHaloIdentity != nil {
			linkedGT := strings.ToLower(sess.LinkedHaloIdentity.Gamertag)
			reqGT := strings.ToLower(req.Gamertag)
			if reqGT != linkedGT {
				return nil, humacore.NewError(http.StatusConflict, "identity_mismatch",
					"Le gamertag ne correspond pas à votre compte Xbox connecté.")
			}
			if req.XUID != "" && sess.LinkedHaloIdentity.XUID != "" && req.XUID != sess.LinkedHaloIdentity.XUID {
				return nil, humacore.NewError(http.StatusConflict, "identity_mismatch",
					"Le XUID ne correspond pas à votre compte Xbox connecté.")
			}
		} else if sess == nil || sess.LinkedHaloIdentity == nil {
			return nil, humacore.NewError(http.StatusConflict, "no_halo_identity",
				"Vous devez d'abord vous connecter à Xbox via le Device Code Flow.")
		}
	}

	// Créer le profil dans db_profiles.json
	playerKey, warnings, err := h.profileSvc.CreatePlayer(req)
	if err != nil {
		slog.ErrorContext(ctx, "setup.CreatePlayer: failed", "gamertag", req.Gamertag, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "profile_create_error",
			"Impossible de créer le profil joueur.")
	}

	// Mettre à jour la session avec le joueur courant
	if sess := middleware.GetSession(ctx); sess != nil {
		slug := playerKey
		sess.CurrentPlayerSlug = &slug
		_ = h.sessionStore.Touch(sess)
	}

	// Sprint 44 : chemin DB title-aware via PathResolver.
	pr := title.NewPathResolver(h.cfg.RepoRoot)
	dbPath := pr.PlayerDBPath(titleSlug, playerKey)
	dbCreated := fileExists(dbPath)

	player := domain.PlayerSummary{
		PlayerSlug:        playerKey,
		Gamertag:          req.Gamertag,
		XUID:              req.XUID,
		WaypointPlayer:    req.Gamertag,
		TitleSlug:         titleSlug,
		SyncEnabled:       true,
		InitialMaxMatches: req.InitialMaxMatches,
	}

	return &setupCreatePlayerOutput{
		Status: http.StatusCreated,
		Body: domain.CreatePlayerProfileResponse{
			Player:    player,
			DBCreated: dbCreated,
			Warnings:  warnings,
		},
	}, nil
}

// handleSmokeTest lance un job de vérification basique de l'environnement.
// POST /setup/smoke-test → 202 AsyncJobStatus.
func (h *SetupHandler) handleSmokeTest(_ context.Context, _ *struct{}) (*setupSmokeTestOutput, error) {
	job := h.jobStore.Create(domain.JobTypeSetupSmokeTest, "")
	// Snapshot avant le go func() : la goroutine modifie in-place le job dans le store.
	jobSnapshot := *job

	go func() {
		step := "Vérification de l'environnement"
		h.jobStore.SetStatus(job.JobID, domain.JobStatusRunning, &step)

		var warnings []string

		// Vérification 1 : db_profiles.json lisible
		if _, err := os.Stat(h.cfg.DBProfilesPath); err != nil {
			warnings = append(warnings, "db_profiles.json introuvable — aucun joueur configuré.")
		}

		// Vérification 2 : dossier sessions accessible
		if _, err := os.Stat(h.cfg.SessionDir); err != nil {
			warnings = append(warnings, "Dossier sessions inaccessible.")
		}

		// Vérification 3 : app_settings.json lisible
		if _, err := os.Stat(h.cfg.AppSettingsPath); err != nil {
			warnings = append(warnings, "app_settings.json introuvable.")
		}

		pct := 100
		done := "Terminé"
		h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
			j.Status = domain.JobStatusSucceeded
			j.ProgressPct = &pct
			j.CurrentStep = &done
			j.Warnings = warnings
			if len(warnings) == 0 {
				j.Result = map[string]any{jsonKeyStatus: "ok"}
			} else {
				j.Result = map[string]any{jsonKeyStatus: "ok_with_warnings"}
			}
		})
	}()

	return &setupSmokeTestOutput{Status: http.StatusAccepted, Body: jobSnapshot}, nil
}

// ---------------------------------------------------------------------------
// Helpers internes
// ---------------------------------------------------------------------------

// fileExists retourne vrai si le chemin existe.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
