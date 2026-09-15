// Package handlers — setup.go : création de profil joueur + smoke test (Sprint 16).
//
// POST /setup/players    → crée un profil joueur via PlayerDirectory.Onboard (201)
// POST /setup/smoke-test → lance une vérification basique de l'environnement (202)
//
// ADR 0035 D4 : la création du profil ne se fait plus ici. Le handler garde ce
// qui est HTTP (gardes d'ouverture, validation, identité Xbox de la session,
// réponse) et délègue la mise en place — profil de suivi PUIS suivi live — à
// l'annuaire des joueurs. Un garde-rail interdit tout autre appelant de
// `CreatePlayer(` (internal/archlint/no_direct_profile_create_test.go).
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
	// directory est le SEUL chemin de création de profil (ADR 0035 D4) : il crée
	// le profil de suivi puis, si le watcher tourne, l'y ajoute — dans cet ordre.
	// Le handler ne connaît plus ProfileService : un garde-rail interdit tout
	// appel direct à CreatePlayer hors de l'annuaire
	// (internal/archlint/no_direct_profile_create_test.go). nil ⇒ 503 typé.
	directory port.PlayerDirectory
	// instanceLocked résout le verrou « instance fermée ». Injecté depuis le
	// point de décision unique (authz.InstanceLocked, ADR 0035 D5) — ce handler
	// ne lit JAMAIS la clé lui-même (garde-rail archlint). nil = jamais
	// verrouillé : même convention que XboxSSOLinkStrategy/UserAuthHandler, et
	// seam des tests unitaires.
	instanceLocked func() bool
	// userLookup résout l'utilisateur courant derrière la session pour l'exemption
	// admin (ajouter le profil d'un ami est un acte d'administration). nil ⇒ repli
	// sur le rôle porté par la session, comme middleware.RequireAdmin.
	userLookup authz.UserLookup
}

// NewSetupHandler crée un SetupHandler.
func NewSetupHandler(
	cfg *config.AppConfig,
	sessionStore *session_platform.Store,
	settingsStore *settings_platform.Store,
	jobStore *jobs.Store,
) *SetupHandler {
	return &SetupHandler{
		cfg:           cfg,
		sessionStore:  sessionStore,
		settingsStore: settingsStore,
		jobStore:      jobStore,
	}
}

// WithDirectory injecte l'annuaire des joueurs, seul chemin de création de
// profil (ADR 0035 D4). Sans lui, POST /setup/players répond 503.
func (h *SetupHandler) WithDirectory(directory port.PlayerDirectory) *SetupHandler {
	h.directory = directory
	return h
}

// WithInstanceLock injecte le résolveur du verrou « instance fermée », construit
// une seule fois au câblage depuis authz.InstanceLocked (ADR 0035 D5).
func (h *SetupHandler) WithInstanceLock(fn func() bool) *SetupHandler {
	h.instanceLocked = fn
	return h
}

// WithUserLookup injecte la résolution de l'utilisateur courant (exemption admin
// sur POST /setup/players).
func (h *SetupHandler) WithUserLookup(lookup authz.UserLookup) *SetupHandler {
	h.userLookup = lookup
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
// Guards (les deux premières sont levées pour un ADMIN, cf. guardProvisioning) :
//   - 403 si can_self_provision=false dans app_settings.json
//   - 403 si instance verrouillée (env LEVELUP_INSTANCE_LOCKED ou app_settings)
//   - 409 si profile_mode="xbox" mais aucune identité Halo liée en session
//   - 409 si gamertag/XUID ne correspond pas à l'identité Halo liée
func (h *SetupHandler) handleCreatePlayer(ctx context.Context, in *setupCreatePlayerInput) (*setupCreatePlayerOutput, error) {
	appCfg, err := h.settingsStore.Load()
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "settings_load_error", "Impossible de charger la configuration.")
	}
	// Gardes d'ouverture. Un ADMIN en est exempté des DEUX : ajouter le profil
	// d'un ami est un acte d'administration, pas de l'auto-provisioning (ADR 0035 D5).
	actorIsAdmin := h.actorIsAdmin(ctx)
	if err := h.guardProvisioning(ctx, appCfg.CanSelfProvision, actorIsAdmin); err != nil {
		return nil, err
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
	if actorIsAdmin {
		slog.InfoContext(ctx, "setup: création profil par admin", "gamertag", req.Gamertag)
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

	if err := guardLinkedXboxIdentity(ctx, req); err != nil {
		return nil, err
	}

	// Mise en place du joueur : profil de suivi PUIS suivi live, par l'annuaire.
	// C'est le seul chemin de création de profil du dépôt (ADR 0035 D4) — le
	// handler ne décide plus de l'ordre, qui est ce que le 2026-07-23 a cassé.
	if h.directory == nil {
		slog.ErrorContext(ctx, "setup: annuaire des joueurs non câblé — création impossible")
		return nil, humacore.NewError(http.StatusServiceUnavailable, "directory_unavailable",
			"Annuaire des joueurs indisponible.")
	}
	res, err := h.directory.Onboard(ctx, domain.OnboardRequest{
		TitleSlug:         titleSlug,
		Gamertag:          req.Gamertag,
		XUID:              req.XUID,
		InitialMaxMatches: req.InitialMaxMatches,
		ActorUsername:     actorUsername(ctx),
	})
	if err != nil {
		slog.ErrorContext(ctx, "setup.Onboard: failed", "gamertag", req.Gamertag, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "profile_create_error",
			"Impossible de créer le profil joueur.")
	}

	// Mettre à jour la session avec le joueur courant
	if sess := middleware.GetSession(ctx); sess != nil {
		slug := res.PlayerKey
		sess.CurrentPlayerSlug = &slug
		_ = h.sessionStore.Touch(sess)
	}

	player := domain.PlayerSummary{
		PlayerSlug:        res.PlayerKey,
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
			DBCreated: res.DBCreated,
			Warnings:  res.Warnings,
		},
	}, nil
}

// guardLinkedXboxIdentity vérifie qu'un profil en mode "xbox" porte bien
// l'identité Halo liée à la session : on ne déclare pas le profil d'un AUTRE
// joueur sous couvert de son propre compte Xbox. Sans effet dans les autres
// modes (profil manuel). Rend nil quand la création peut continuer.
func guardLinkedXboxIdentity(ctx context.Context, req domain.CreatePlayerProfileRequest) error {
	if req.ProfileMode != "xbox" {
		return nil
	}
	sess := middleware.GetSession(ctx)
	if sess == nil || sess.LinkedHaloIdentity == nil {
		return humacore.NewError(http.StatusConflict, "no_halo_identity",
			"Vous devez d'abord vous connecter à Xbox via le Device Code Flow.")
	}
	if !strings.EqualFold(req.Gamertag, sess.LinkedHaloIdentity.Gamertag) {
		return humacore.NewError(http.StatusConflict, "identity_mismatch",
			"Le gamertag ne correspond pas à votre compte Xbox connecté.")
	}
	if req.XUID != "" && sess.LinkedHaloIdentity.XUID != "" && req.XUID != sess.LinkedHaloIdentity.XUID {
		return humacore.NewError(http.StatusConflict, "identity_mismatch",
			"Le XUID ne correspond pas à votre compte Xbox connecté.")
	}
	return nil
}

// actorUsername rend le nom du compte à l'origine de la demande, pour le journal
// de l'annuaire. Vide si la requête n'a pas de session (démo / auth non activée).
func actorUsername(ctx context.Context) string {
	sess := middleware.GetSession(ctx)
	if sess == nil || sess.Username == nil {
		return ""
	}
	return *sess.Username
}

// actorIsAdmin dit si l'appelant est administrateur de l'instance. Priorité au
// store (authz.CurrentUser : un rôle rétrogradé après l'ouverture de session y
// est visible) ; sans lookup câblé, repli sur le rôle porté par la session, même
// source que middleware.RequireAdmin.
func (h *SetupHandler) actorIsAdmin(ctx context.Context) bool {
	sess := middleware.GetSession(ctx)
	if sess == nil {
		return false
	}
	if h.userLookup != nil {
		if u := authz.CurrentUser(sess, h.userLookup); u != nil {
			return u.Role == domain.RoleAdmin
		}
		return false
	}
	return sess.Role != nil && *sess.Role == string(domain.RoleAdmin)
}

// guardProvisioning applique les deux gardes d'ouverture de l'instance :
// auto-provisioning autorisé, puis verrou « instance fermée ». Un admin en est
// exempté (ADR 0035 D5). Retourne nil quand la création peut continuer.
func (h *SetupHandler) guardProvisioning(ctx context.Context, canSelfProvision, actorIsAdmin bool) error {
	if actorIsAdmin {
		return nil
	}
	if !canSelfProvision {
		slog.WarnContext(ctx, "setup: création profil refusée — auto-provisioning désactivé")
		return humacore.NewError(http.StatusForbidden, "provisioning_disabled",
			"L'auto-provisioning est désactivé sur cette instance.")
	}
	// Verrou effectif = env (LEVELUP_INSTANCE_LOCKED) OU app_settings.instance_locked,
	// résolu par le point de décision unique injecté au câblage.
	if h.instanceLocked != nil && h.instanceLocked() {
		slog.WarnContext(ctx, "setup: création profil refusée — instance verrouillée")
		return humacore.NewError(http.StatusForbidden, "instance_locked",
			"Cette instance est fermée : la création de nouveaux profils est désactivée.")
	}
	return nil
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
