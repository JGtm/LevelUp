// Package handlers — assets.go : handler unifié pour tous les assets visuels Waypoint.
//
// Toute la logique local-first → API-fallback est dans internal/assets/.
// Ce fichier contient uniquement les trampolines HTTP : parse les params chi,
// construit assets.Ref, appelle resolver.Get, puis redirige ou sert.
package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/assets"
	"levelup/go-api/internal/ctxkeys"
)

// AssetHandler gère le cache-aside unifié des assets visuels Halo.
type AssetHandler struct {
	resolver assets.Resolver
}

// NewAssetHandler crée un AssetHandler depuis un Resolver.
func NewAssetHandler(resolver assets.Resolver) *AssetHandler {
	return &AssetHandler{resolver: resolver}
}

// GetMedalImage sert l'image d'une médaille depuis le resolver.
// GET /api/v1/assets/medals/{title_id}/{medal_id}/image
func (h *AssetHandler) GetMedalImage(w http.ResponseWriter, r *http.Request) {
	titleID := chi.URLParam(r, "title_id")
	medalIDStr := chi.URLParam(r, "medal_id")

	medalID, err := strconv.ParseInt(medalIDStr, 10, 64)
	if err != nil {
		httpError(r.Context(), w, "medal_id invalide", http.StatusBadRequest)
		return
	}

	ref := assets.Ref{
		Kind:    assets.KindMedalImage,
		TitleID: titleID,
		ID:      strconv.FormatInt(medalID, 10),
	}

	resolved, err := h.resolver.Get(r.Context(), ref)
	if err != nil {
		handleResolverError(r.Context(), w, err, "GetMedalImage", ref)
		return
	}
	serveResolved(w, r, resolved)
}

// GetMapImage sert l'image d'une map depuis le resolver.
// GET /api/v1/assets/maps/{title_id}/{map_id}/image
func (h *AssetHandler) GetMapImage(w http.ResponseWriter, r *http.Request) {
	titleID := chi.URLParam(r, "title_id")
	mapID := chi.URLParam(r, "map_id")

	if mapID == "" {
		httpError(r.Context(), w, "map_id requis", http.StatusBadRequest)
		return
	}

	// version_id (optionnel) : porté par ?v=. Requis pour le fallback DiscoveryUGC
	// (404 sans), et inclus dans la clé de cache (Variant) → cache par (map, version).
	ref := assets.Ref{
		Kind:    assets.KindMapImage,
		TitleID: titleID,
		ID:      mapID,
		Variant: strings.TrimSpace(r.URL.Query().Get("v")),
	}

	resolved, err := h.resolver.Get(r.Context(), ref)
	if err != nil {
		handleResolverError(r.Context(), w, err, "GetMapImage", ref)
		return
	}
	serveResolved(w, r, resolved)
}

// GetChallengeBadge sert l'image d'un badge de défi depuis le resolver.
// GET /api/v1/assets/challenge-badge/{title_id}/{badge_id}
func (h *AssetHandler) GetChallengeBadge(w http.ResponseWriter, r *http.Request) {
	titleID := chi.URLParam(r, "title_id")
	badgeID := chi.URLParam(r, "badge_id")

	if badgeID == "" || strings.ContainsAny(badgeID, "/\\") {
		httpError(r.Context(), w, "badge_id invalide", http.StatusBadRequest)
		return
	}

	ref := assets.Ref{
		Kind:    assets.KindChallengeBadge,
		TitleID: titleID,
		ID:      badgeID,
	}

	resolved, err := h.resolver.Get(r.Context(), ref)
	if err != nil {
		handleResolverError(r.Context(), w, err, "GetChallengeBadge", ref)
		return
	}
	serveResolved(w, r, resolved)
}

// GetBattlePassImage sert les images Battle Pass depuis le resolver.
// GET /api/v1/assets/battlepass/{subdir}/*
func (h *AssetHandler) GetBattlePassImage(w http.ResponseWriter, r *http.Request) {
	subDir := chi.URLParam(r, "subdir")
	gamecmsPath := chi.URLParam(r, "*")

	if subDir == "" || strings.ContainsAny(subDir, "/\\") {
		httpError(r.Context(), w, "subdir invalide", http.StatusBadRequest)
		return
	}
	if gamecmsPath == "" {
		httpError(r.Context(), w, "chemin image manquant", http.StatusBadRequest)
		return
	}
	cleaned := path.Clean(gamecmsPath)
	if strings.Contains(cleaned, "..") {
		httpError(r.Context(), w, "chemin invalide", http.StatusBadRequest)
		return
	}

	kind := assets.KindBPTrackImage
	if subDir == "background" {
		kind = assets.KindBPBackground
	}

	ref := assets.Ref{
		Kind:    kind,
		TitleID: ctxkeys.TitleSlug(r.Context()),
		ID:      cleaned,
		Variant: subDir,
	}

	resolved, err := h.resolver.Get(r.Context(), ref)
	if err != nil {
		handleResolverError(r.Context(), w, err, "GetBattlePassImage", ref)
		return
	}
	serveResolved(w, r, resolved)
}

// GetSpartanImage sert les visuels du bloc identitaire Spartan via le resolver.
// GET /api/v1/assets/spartan/{image_type}/{title_id}/*
func (h *AssetHandler) GetSpartanImage(w http.ResponseWriter, r *http.Request) {
	imageType := chi.URLParam(r, "image_type")
	titleID := chi.URLParam(r, "title_id")
	gamecmsPath := chi.URLParam(r, "*")

	kind, ok := spartanImageKind(imageType)
	if !ok {
		httpError(r.Context(), w, "image_type invalide", http.StatusBadRequest)
		return
	}
	if titleID == "" {
		httpError(r.Context(), w, "title_id requis", http.StatusBadRequest)
		return
	}
	if gamecmsPath == "" {
		httpError(r.Context(), w, "chemin image manquant", http.StatusBadRequest)
		return
	}

	cleaned := path.Clean(gamecmsPath)
	if strings.Contains(cleaned, "..") {
		httpError(r.Context(), w, "chemin invalide", http.StatusBadRequest)
		return
	}

	ref := assets.Ref{
		Kind:    kind,
		TitleID: titleID,
		ID:      cleaned,
	}

	resolved, err := h.resolver.Get(r.Context(), ref)
	if err != nil {
		handleResolverError(r.Context(), w, err, "GetSpartanImage", ref)
		return
	}
	serveResolved(w, r, resolved)
}

func spartanImageKind(imageType string) (assets.Kind, bool) {
	switch strings.TrimSpace(imageType) {
	case "emblem":
		return assets.KindSpartanEmblem, true
	case "banner":
		return assets.KindSpartanBanner, true
	case "backdrop":
		return assets.KindSpartanBackdrop, true
	case "career-rank":
		return assets.KindCareerRankImage, true
	default:
		return "", false
	}
}

// serveResolved écrit la réponse HTTP depuis un Resolved.
// URLPayload → 302, BinaryPayload → 200 + bytes.
func serveResolved(w http.ResponseWriter, r *http.Request, res assets.Resolved) {
	switch p := res.Payload.(type) {
	case assets.URLPayload:
		http.Redirect(w, r, p.URL, http.StatusFound)
	case assets.BinaryPayload:
		w.Header().Set("Content-Type", p.ContentType)
		w.Header().Set("Cache-Control", "public, max-age=86400")
		if p.ETag != "" {
			w.Header().Set("ETag", `"`+p.ETag+`"`)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(p.Bytes)
	default:
		httpError(r.Context(), w, "payload inattendu", http.StatusInternalServerError)
	}
}

// handleResolverError traduit les erreurs du resolver en réponses HTTP.
func handleResolverError(ctx context.Context, w http.ResponseWriter, err error, op string, ref assets.Ref) {
	switch {
	case errors.Is(err, assets.ErrNotFound):
		slog.Debug(op+": asset not found", ref.LogAttrs()...)
		httpError(ctx, w, "asset non trouvé", http.StatusNotFound)
	case errors.Is(err, assets.ErrUpstreamUnavailable):
		slog.Warn(op+": upstream unavailable", append(ref.LogAttrs(), "err", err)...)
		httpError(ctx, w, "source distante indisponible", http.StatusBadGateway)
	case errors.Is(err, assets.ErrUnsupportedKind):
		slog.ErrorContext(ctx, op+": unsupported kind", append(ref.LogAttrs(), "err", err)...)
		httpError(ctx, w, "type d'asset non supporté", http.StatusInternalServerError)
	default:
		slog.ErrorContext(ctx, op+": resolver error", append(ref.LogAttrs(), "err", err)...)
		httpError(ctx, w, "erreur interne", http.StatusInternalServerError)
	}
}
