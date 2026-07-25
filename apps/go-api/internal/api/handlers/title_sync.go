// Package handlers — title_sync.go : gestion par le joueur de ses titres.
//
// Endpoints (owner-gated, montés sous un groupe chi
// /profiles/{player_slug}/titles/{slug} avec TitleSlugFromPath + ownership, SANS
// RequireActiveTitle — il doit rester possible d'agir sur un titre coming_soon
// ou archivé) :
//
//	PATCH  /api/v1/profiles/{player_slug}/titles/{slug}/sync   body {enabled}
//	DELETE /api/v1/profiles/{player_slug}/titles/{slug}/data
//
// Sélection par titre (Pass B.5). player_slug == gamertag (cf. loadPlayersV3).
package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/platform/dbprofiles"
	"levelup/go-api/internal/service"
)

// TitleSyncHandler gère l'activation/pause et la purge d'un titre par joueur.
type TitleSyncHandler struct {
	profiles *service.ProfileService
}

// NewTitleSyncHandler crée un TitleSyncHandler.
func NewTitleSyncHandler(profiles *service.ProfileService) *TitleSyncHandler {
	return &TitleSyncHandler{profiles: profiles}
}

// Mount enregistre les routes via Huma sur le sous-routeur chi
// (préfixe /profiles/{player_slug}/titles/{slug} + ownership hérité).
func (h *TitleSyncHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Patch(api, "/sync", h.SetSync, humacore.Op("setTitleSync", "Activer ou mettre en pause un titre pour un joueur", "setup"))
	huma.Delete(api, "/data", h.Purge, humacore.Op("purgeTitleData", "Purger les données d'un titre pour un joueur", "setup"))
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

type titleSyncInput struct {
	PlayerSlug string `path:"player_slug"`
	Slug       string `path:"slug"`
	Body       struct {
		Enabled bool `json:"enabled"`
	}
}

type titleSyncOutput struct {
	Body struct {
		Gamertag    string `json:"gamertag"`
		TitleSlug   string `json:"title_slug"`
		SyncEnabled bool   `json:"sync_enabled"`
	}
}

type titlePurgeInput struct {
	PlayerSlug string `path:"player_slug"`
	Slug       string `path:"slug"`
}

type titlePurgeOutput struct {
	Body struct {
		Gamertag  string `json:"gamertag"`
		TitleSlug string `json:"title_slug"`
		// DataRemoved : false si les fichiers n'ont pas pu être supprimés malgré
		// le retrait du profil (verrou disque résiduel) — le titre est tout de
		// même désactivé.
		DataRemoved bool `json:"data_removed"`
	}
}

// ─── Endpoints ───────────────────────────────────────────────────────────────

// SetSync active (enabled=true) ou met en pause (false) un titre pour le joueur.
func (h *TitleSyncHandler) SetSync(ctx context.Context, in *titleSyncInput) (*titleSyncOutput, error) {
	if err := h.profiles.SetTitleSyncEnabled(in.Slug, in.PlayerSlug, in.Body.Enabled); err != nil {
		if e := mapTitleSyncError(ctx, err, "toggle", in.PlayerSlug, in.Slug); e != nil {
			return nil, e
		}
	}
	slog.InfoContext(ctx, "title sync toggled",
		"player_slug", in.PlayerSlug, "titleSlug", in.Slug, "enabled", in.Body.Enabled)
	out := &titleSyncOutput{}
	out.Body.Gamertag = in.PlayerSlug
	out.Body.TitleSlug = in.Slug
	out.Body.SyncEnabled = in.Body.Enabled
	return out, nil
}

// Purge retire le titre du profil et supprime ses données disque.
func (h *TitleSyncHandler) Purge(ctx context.Context, in *titlePurgeInput) (*titlePurgeOutput, error) {
	dataRemoved, err := h.profiles.PurgeTitleData(in.Slug, in.PlayerSlug)
	if err != nil {
		if e := mapTitleSyncError(ctx, err, "purge", in.PlayerSlug, in.Slug); e != nil {
			return nil, e
		}
	}
	if !dataRemoved {
		slog.WarnContext(ctx, "title purge: profil retiré mais fichiers non supprimés (verrou disque)",
			"player_slug", in.PlayerSlug, "titleSlug", in.Slug)
	}
	slog.InfoContext(ctx, "title purged",
		"player_slug", in.PlayerSlug, "titleSlug", in.Slug, "data_removed", dataRemoved)
	out := &titlePurgeOutput{}
	out.Body.Gamertag = in.PlayerSlug
	out.Body.TitleSlug = in.Slug
	out.Body.DataRemoved = dataRemoved
	return out, nil
}

// mapTitleSyncError mappe les erreurs sentinelles du store en erreurs Huma.
// Retourne nil si err est nil.
func mapTitleSyncError(ctx context.Context, err error, op, playerSlug, slug string) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, dbprofiles.ErrEntryNotFound):
		return humacore.NewError(http.StatusNotFound, "title_profile_not_found",
			"Aucun profil pour ce couple joueur/titre.")
	case errors.Is(err, dbprofiles.ErrLastActiveTitle):
		return humacore.NewError(http.StatusConflict, "last_active_title",
			"Au moins un titre doit rester actif pour ce joueur.")
	}
	slog.ErrorContext(ctx, "title sync op failed",
		"op", op, "player_slug", playerSlug, "titleSlug", slug, "err", err)
	return humacore.NewError(http.StatusInternalServerError, op+"_error", err.Error())
}
