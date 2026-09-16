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
	"levelup/go-api/internal/authz"
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
	// users + grantClearer : droit à usage unique de créer SON profil sur
	// instance verrouillée (D3). nil → aucun invité ne passe le verrou
	// (comportement d'origine).
	users        authz.UserLookup
	grantClearer ProvisionGrantClearer
}

// ProvisionGrantClearer efface le droit de provisioning d'un compte après usage.
// Satisfait par *userstore.Store.SetProvisionGrant(username, "").
type ProvisionGrantClearer interface {
	SetProvisionGrant(username, code string) error
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

// WithProvisionGrant branche la résolution du compte courant et l'effacement du
// droit de provisioning. Sans ce câblage, le verrou d'instance reste absolu.
func (h *SetupHandler) WithProvisionGrant(users authz.UserLookup, clearer ProvisionGrantClearer) *SetupHandler {
	h.users = users
	h.grantClearer = clearer
	return h
}

// Mount enregistre les 2 routes via Huma sur le routeur chi (mêmes points de
// montage que les routes chi d'origine). Le body POST /setup/players est marqué
// OPTIONNEL (MarkRequestBodyOptional) : le décodage maison rend un corps absent
// OU malformé en 400 {invalid_body}, contrat préservé.
func (h *SetupHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/setup/players", h.handleCreatePlayer, humacore.Op("postSetupPlayers", "Créer un profil joueur", "setup"),
		humacore.DefaultStatus(http.StatusCreated))
	humacore.MarkRequestBodyOptional(api, http.MethodPost, "/setup/players")
	huma.Post(api, "/setup/smoke-test", h.handleSmokeTest, humacore.Op("postSetupSmokeTest", "Lancer un smoke test (Go-only — à valider Sprint 31)", "setup"),
		humacore.DefaultStatus(http.StatusAccepted))
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// setupCreatePlayerInput : corps brut décodé maison (RawBody + body marqué
// OPTIONNEL) → corps absent ou JSON malformé ⇒ 400 {invalid_body}.
type setupCreatePlayerInput struct {
	RawBody []byte
}

// setupCreatePlayerOutput : 201 CreatePlayerProfileResponse (statut posé UNE
// fois par humacore.DefaultStatus au montage — cf. Mount).
type setupCreatePlayerOutput struct {
	Body domain.CreatePlayerProfileResponse
}

// setupSmokeTestOutput : 202 Accepted, corps = snapshot du job créé (statut posé
// par humacore.DefaultStatus au montage).
type setupSmokeTestOutput struct {
	Body domain.AsyncJobStatus
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

	// Guard : instance fermée (lockdown) — pas de nouvelle BDD joueur, SAUF pour
	// un invité qui porte un droit de provisioning non consommé et n'a pas encore
	// de profil (D3). La vraie barrière reste le contrôle « xuid = identité liée »
	// plus bas : le droit ne dispense pas d'être soi.
	// Verrou effectif = env (LEVELUP_INSTANCE_LOCKED) OU app_settings.instance_locked.
	grantHolder := (*domain.User)(nil)
	if h.cfg.InstanceLocked || appCfg.InstanceLocked {
		grantHolder = h.provisionGrantHolder(ctx)
		if grantHolder == nil {
			slog.WarnContext(ctx, "setup: création profil refusée — instance verrouillée",
				"env_locked", h.cfg.InstanceLocked, "settings_locked", appCfg.InstanceLocked)
			return nil, humacore.NewError(http.StatusForbidden, "instance_locked",
				"Cette instance est fermée : la création de nouveaux profils est désactivée.")
		}
		slog.InfoContext(ctx, "setup: verrou levé par un droit de provisioning",
			"username", grantHolder.Username)
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

	// Verrou levé par un droit de provisioning : la requête est ÉPINGLÉE à
	// l identité du porteur, quel que soit profile_mode. Sans cela un invité
	// pouvait envoyer profile_mode "azure_manual" (ou un xuid vide) et sauter le
	// bloc d identité ci-dessous, donc écrire dans db_profiles.json un profil pour
	// un gamertag et un xuid étrangers (constat P0 de revue, 2026-09-16). Le droit
	// ne vaut que pour SON premier profil : gamertag et xuid du compte, mode xbox.
	if grantHolder != nil {
		if !strings.EqualFold(req.Gamertag, grantHolder.Gamertag) ||
			(req.XUID != "" && req.XUID != grantHolder.XUID) {
			slog.WarnContext(ctx, "setup: droit de provisioning refusé — identité étrangère",
				"username", grantHolder.Username, "gamertag", req.Gamertag, "xuid", req.XUID)
			return nil, humacore.NewError(http.StatusConflict, "identity_mismatch",
				"Le droit de provisioning ne vaut que pour votre propre compte Xbox.")
		}
		req.XUID = grantHolder.XUID
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

	// Le droit de provisioning a servi : l'effacer pour qu'il ne soit pas
	// rejouable. Échec journalisé — la création, elle, reste acquise.
	if grantHolder != nil && h.grantClearer != nil {
		if err := h.grantClearer.SetProvisionGrant(grantHolder.Username, ""); err != nil {
			slog.ErrorContext(ctx, "setup: effacement du droit de provisioning échoué",
				"username", grantHolder.Username, "err", err)
		}
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
		Body: domain.CreatePlayerProfileResponse{
			Player:    player,
			DBCreated: dbCreated,
			Warnings:  warnings,
		},
	}, nil
}

// provisionGrantHolder retourne l'utilisateur de la session S'IL porte un droit
// de provisioning non consommé ET qu'aucun profil de db_profiles.json ne porte
// déjà son xuid. Sinon nil : le verrou d'instance s'applique.
//
// La double condition est le point : le droit sert UNE fois, pour SON premier
// profil. Un invité qui a déjà un profil n'a plus rien à provisionner.
func (h *SetupHandler) provisionGrantHolder(ctx context.Context) *domain.User {
	if h.users == nil {
		return nil
	}
	user := authz.CurrentUser(middleware.GetSession(ctx), h.users)
	if user == nil || user.ProvisionGrant == "" || user.XUID == "" {
		return nil
	}
	players, err := h.cfg.LoadPlayers()
	if err != nil {
		// Ne pas lever le verrou sur une lecture ratée : on ne peut pas prouver
		// que l'invité n'a pas déjà un profil.
		slog.ErrorContext(ctx, "setup: lecture des profils impossible — droit de provisioning ignoré",
			"username", user.Username, "err", err)
		return nil
	}
	for i := range players {
		if players[i].XUID == user.XUID {
			slog.WarnContext(ctx, "setup: droit de provisioning ignoré — le compte a déjà un profil",
				"username", user.Username, "player_slug", players[i].PlayerSlug)
			return nil
		}
	}
	return user
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

	return &setupSmokeTestOutput{Body: jobSnapshot}, nil
}

// ---------------------------------------------------------------------------
// Helpers internes
// ---------------------------------------------------------------------------

// fileExists retourne vrai si le chemin existe.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
