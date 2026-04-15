// Package handlers — setup.go : création de profil joueur + smoke test (Sprint 16).
//
// POST /setup/players    → crée un profil joueur dans db_profiles.json (201)
// POST /setup/smoke-test → lance une vérification basique de l'environnement (202)
package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
	settings_platform "levelup/go-api/internal/platform/settings"
	session_platform "levelup/go-api/internal/platform/session"
)

// SetupHandler gère les endpoints de mise en place du profil joueur.
type SetupHandler struct {
	cfg          *config.AppConfig
	sessionStore *session_platform.Store
	settingsStore *settings_platform.Store
	jobStore     *jobs.Store
}

// NewSetupHandler crée un SetupHandler.
func NewSetupHandler(
	cfg *config.AppConfig,
	sessionStore *session_platform.Store,
	settingsStore *settings_platform.Store,
	jobStore *jobs.Store,
) *SetupHandler {
	return &SetupHandler{
		cfg:          cfg,
		sessionStore: sessionStore,
		settingsStore: settingsStore,
		jobStore:     jobStore,
	}
}

// CreatePlayer crée un profil joueur dans db_profiles.json.
// POST /setup/players → 201 CreatePlayerProfileResponse.
//
// Guards :
//   - 403 si can_self_provision=false dans app_settings.json
//   - 409 si profile_mode="xbox" mais aucune identité Halo liée en session
//   - 409 si gamertag ne correspond pas à l'identité Halo liée
func (h *SetupHandler) CreatePlayer(w http.ResponseWriter, r *http.Request) {
	// Guard : can_self_provision
	appCfg, err := h.settingsStore.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "settings_load_error", "Impossible de charger la configuration.")
		return
	}
	if !appCfg.CanSelfProvision {
		writeError(w, http.StatusForbidden, "provisioning_disabled",
			"L'auto-provisioning est désactivé sur cette instance.")
		return
	}

	var req domain.CreatePlayerProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Corps de requête JSON invalide.")
		return
	}

	req.Gamertag = strings.TrimSpace(req.Gamertag)
	if req.Gamertag == "" || len(req.Gamertag) > 50 {
		writeError(w, http.StatusBadRequest, "invalid_gamertag", "Le gamertag est vide ou trop long.")
		return
	}
	if req.ProfileMode == "" {
		req.ProfileMode = "xbox"
	}

	// Guard : identité Xbox liée (mode xbox uniquement)
	if req.ProfileMode == "xbox" {
		sess := middleware.GetSession(r.Context())
		if sess != nil && sess.LinkedHaloIdentity != nil {
			linkedGT := strings.ToLower(sess.LinkedHaloIdentity.Gamertag)
			reqGT := strings.ToLower(req.Gamertag)
			if reqGT != linkedGT {
				writeError(w, http.StatusConflict, "identity_mismatch",
					"Le gamertag ne correspond pas à votre compte Xbox connecté.")
				return
			}
			if req.XUID != "" && sess.LinkedHaloIdentity.XUID != "" && req.XUID != sess.LinkedHaloIdentity.XUID {
				writeError(w, http.StatusConflict, "identity_mismatch",
					"Le XUID ne correspond pas à votre compte Xbox connecté.")
				return
			}
		} else if sess == nil || sess.LinkedHaloIdentity == nil {
			writeError(w, http.StatusConflict, "no_halo_identity",
				"Vous devez d'abord vous connecter à Xbox via le Device Code Flow.")
			return
		}
	}

	// Créer le profil dans db_profiles.json
	playerKey, warnings, err := createPlayerInProfiles(h.cfg, req)
	if err != nil {
		slog.Error("setup.CreatePlayer: failed", "gamertag", req.Gamertag, "err", err)
		writeError(w, http.StatusInternalServerError, "profile_create_error",
			"Impossible de créer le profil joueur.")
		return
	}

	// Mettre à jour la session avec le joueur courant
	if sess := middleware.GetSession(r.Context()); sess != nil {
		slug := playerKey
		sess.CurrentPlayerSlug = &slug
		_ = h.sessionStore.Touch(sess)
	}

	dbPath := filepath.Join(h.cfg.RepoRoot, "data", "players", playerKey, "stats.duckdb")
	dbCreated := fileExists(dbPath)

	player := domain.PlayerSummary{
		PlayerSlug:     playerKey,
		Gamertag:       req.Gamertag,
		XUID:           req.XUID,
		WaypointPlayer: req.Gamertag,
	}

	writeJSON(w, http.StatusCreated, domain.CreatePlayerProfileResponse{
		Player:    player,
		DBCreated: dbCreated,
		Warnings:  warnings,
	})
}

// SmokeTest lance un job de vérification basique de l'environnement.
// POST /setup/smoke-test → 202 AsyncJobStatus.
func (h *SetupHandler) SmokeTest(w http.ResponseWriter, r *http.Request) {
	job := h.jobStore.Create(domain.JobTypeSetupSmokeTest, "")

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
				j.Result = map[string]any{"status": "ok"}
			} else {
				j.Result = map[string]any{"status": "ok_with_warnings"}
			}
		})
	}()

	writeJSON(w, http.StatusAccepted, job)
}

// ---------------------------------------------------------------------------
// Helpers internes
// ---------------------------------------------------------------------------

// dbProfilesFile représente le format de db_profiles.json v2.1.
type dbProfilesFile struct {
	Version  string                       `json:"version"`
	Profiles map[string]dbProfileEntry    `json:"profiles"`
}

type dbProfileEntry struct {
	DBPath         string `json:"db_path"`
	WaypointPlayer string `json:"waypoint_player,omitempty"`
	XUID           string `json:"xuid,omitempty"`
}

// createPlayerInProfiles crée ou met à jour un profil dans db_profiles.json.
// Retourne la clé du profil et les warnings éventuels.
func createPlayerInProfiles(cfg *config.AppConfig, req domain.CreatePlayerProfileRequest) (string, []string, error) {
	var profiles dbProfilesFile

	data, err := os.ReadFile(cfg.DBProfilesPath)
	if err != nil && !os.IsNotExist(err) {
		return "", nil, err
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &profiles); err != nil {
			return "", nil, err
		}
	}
	if profiles.Version == "" {
		profiles.Version = "2.1"
	}
	if profiles.Profiles == nil {
		profiles.Profiles = make(map[string]dbProfileEntry)
	}

	// Recherche insensible à la casse
	finalKey := req.Gamertag
	for k := range profiles.Profiles {
		if strings.EqualFold(k, req.Gamertag) {
			finalKey = k
			break
		}
	}

	dbPath := filepath.Join("data", "players", finalKey, "stats.duckdb")
	entry := dbProfileEntry{
		DBPath:         dbPath,
		WaypointPlayer: req.Gamertag,
	}
	if req.XUID != "" {
		entry.XUID = req.XUID
	}

	// Merge avec l'existant
	if existing, ok := profiles.Profiles[finalKey]; ok {
		if req.XUID == "" && existing.XUID != "" {
			entry.XUID = existing.XUID
		}
	}
	profiles.Profiles[finalKey] = entry

	// Écriture
	out, err := json.MarshalIndent(profiles, "", "  ")
	if err != nil {
		return "", nil, err
	}
	if err := os.WriteFile(cfg.DBProfilesPath, out, 0o644); err != nil {
		return "", nil, err
	}

	// Créer le dossier joueur
	playerDir := filepath.Join(cfg.RepoRoot, "data", "players", finalKey)
	if err := os.MkdirAll(playerDir, 0o755); err != nil {
		return finalKey, []string{"Dossier joueur non créé : " + err.Error()}, nil
	}

	return finalKey, nil, nil
}

// fileExists retourne vrai si le chemin existe.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
