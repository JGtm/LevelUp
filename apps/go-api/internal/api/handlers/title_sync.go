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
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dbprofiles"
	"levelup/go-api/internal/service"
)

// TitleWatcher est ce dont ce handler a besoin du daemon watcher pour que le
// suivi live suive les profils : retirer un couple (joueur, titre) mis en pause
// ou purgé, remettre un couple réactivé. Implémenté par *watcher.Daemon ; défini
// ici pour ne pas coupler handlers → watcher.
//
// POURQUOI (revue adversariale du 2026-09-16, P1) : sans ce retrait, le poller
// d'un titre mis en pause survivait jusqu'au redémarrage ; depuis la porte
// « profil suivi » (ADR 0035 D3), chacun de ses matchs faisait grimper
// `sync_refused_no_profile` — le compteur qui signale une identité inconnue —
// et l'annuaire affichait `watched_without_profile` en warning. Une action
// d'administration légitime déclenchait l'alarme d'intrusion.
type TitleWatcher interface {
	IsRunning() bool
	RemovePlayerTitle(ctx context.Context, xuid, titleSlug string) bool
	AddPlayer(ctx context.Context, p domain.PlayerSummary) error
}

// PlayerLookup résout le profil (titre, gamertag) — nécessaire pour remettre un
// titre réactivé au watcher (xuid, initial_max_matches). Câblé sur
// config.AppConfig.LoadPlayers.
type PlayerLookup func(titleSlug, playerSlug string) (domain.PlayerSummary, bool)

// TitleSyncHandler gère l'activation/pause et la purge d'un titre par joueur.
type TitleSyncHandler struct {
	profiles *service.ProfileService
	// watcher résout le daemon au moment de l'appel (créé après le montage des
	// routes, cf. daemonGetter du SSO). nil, ou daemon nil/arrêté ⇒ rien à faire :
	// le daemon reprend les profils au prochain démarrage.
	watcher func() TitleWatcher
	lookup  PlayerLookup
}

// NewTitleSyncHandler crée un TitleSyncHandler.
func NewTitleSyncHandler(profiles *service.ProfileService) *TitleSyncHandler {
	return &TitleSyncHandler{profiles: profiles}
}

// WithWatcher injecte le résolveur du daemon watcher (lazy).
func (h *TitleSyncHandler) WithWatcher(getter func() TitleWatcher) *TitleSyncHandler {
	h.watcher = getter
	return h
}

// WithPlayerLookup injecte la résolution de profil pour la réactivation.
func (h *TitleSyncHandler) WithPlayerLookup(lookup PlayerLookup) *TitleSyncHandler {
	h.lookup = lookup
	return h
}

// syncWatcher aligne le suivi live sur le profil après une pause, une
// réactivation ou une purge. Best-effort : journalisé, jamais bloquant — le
// profil fait foi, le daemon le relit au boot.
//
// knownXUID : xuid résolu AVANT une purge (le profil n'existe plus après) ;
// vide sinon, auquel cas le profil est résolu ici.
func (h *TitleSyncHandler) syncWatcher(ctx context.Context, titleSlug, playerSlug string, enabled bool, knownXUID string) {
	if h.watcher == nil {
		return
	}
	w := h.watcher()
	if w == nil || !w.IsRunning() {
		return
	}
	if h.lookup == nil {
		slog.WarnContext(ctx, "title sync: suivi live non aligné — résolution de profil non câblée",
			"player_slug", playerSlug, "titleSlug", titleSlug)
		return
	}
	p, ok := h.lookup(titleSlug, playerSlug)
	if !enabled {
		xuid := knownXUID
		if xuid == "" && ok {
			xuid = p.XUID
		}
		if xuid == "" {
			return
		}
		if w.RemovePlayerTitle(ctx, xuid, titleSlug) {
			slog.InfoContext(ctx, "title sync: suivi live retiré",
				"player_slug", playerSlug, "titleSlug", titleSlug, "xuid", xuid)
		}
		return
	}
	if !ok {
		slog.WarnContext(ctx, "title sync: profil introuvable après réactivation, suivi live non repris",
			"player_slug", playerSlug, "titleSlug", titleSlug)
		return
	}
	p.SyncEnabled = true
	if err := w.AddPlayer(ctx, p); err != nil {
		slog.WarnContext(ctx, "title sync: suivi live non repris (non bloquant)",
			"player_slug", playerSlug, "titleSlug", titleSlug, "xuid", p.XUID, "err", err)
		return
	}
	slog.InfoContext(ctx, "title sync: suivi live repris",
		"player_slug", playerSlug, "titleSlug", titleSlug, "xuid", p.XUID)
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
	h.syncWatcher(ctx, in.Slug, in.PlayerSlug, in.Body.Enabled, "")
	out := &titleSyncOutput{}
	out.Body.Gamertag = in.PlayerSlug
	out.Body.TitleSlug = in.Slug
	out.Body.SyncEnabled = in.Body.Enabled
	return out, nil
}

// Purge retire le titre du profil et supprime ses données disque.
func (h *TitleSyncHandler) Purge(ctx context.Context, in *titlePurgeInput) (*titlePurgeOutput, error) {
	// Le xuid se résout AVANT le retrait : après, le profil n'existe plus.
	var xuid string
	if h.lookup != nil {
		if p, ok := h.lookup(in.Slug, in.PlayerSlug); ok {
			xuid = p.XUID
		}
	}
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
	h.syncWatcher(ctx, in.Slug, in.PlayerSlug, false, xuid)
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
