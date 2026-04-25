// Phase C+ du plan multi-titres : endpoint preview player-scoped qui consomme
// le TitleDataAdapter avec un CareerSource player-scoped (capability career.
// progression réellement supportée).
//
// Différence avec MultiTitlePreviewHandler :
//   - le handler global utilise le DataAdapter registré au boot (sans Career
//     Source → capability not_exposed)
//   - ce handler player-scoped instancie un DataAdapter dédié par requête via
//     le ServiceRegistry.TitleDataAdapter, ce qui injecte le CareerRepo du
//     joueur courant
//
// Route : GET /api/v1/players/{player_slug}/preview/career-multi-title?locale=fr
// Derrière le flag MULTI_TITLE_API_ENABLED.
package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
)

// PlayerCareerAdapterFactory résout le slug joueur en TitleDataAdapter
// player-scoped (avec CareerRepo injecté). Implémenté par
// ServiceRegistry.TitleDataAdapter dans api/registry.go.
type PlayerCareerAdapterFactory func(ctx context.Context, slug string) (games.TitleDataAdapter, error)

// SemanticAdapterFactory résout le slug joueur en TitleSemanticAdapter via
// le resolver title-aware. Implémenté à partir de games.Resolver.Semantic().
type SemanticAdapterFactory func(ctx context.Context, titleSlug string) (games.TitleSemanticAdapter, error)

// MultiTitlePlayerPreviewHandler combine le DataAdapter player-scoped et le
// SemanticAdapter pour produire la même payload preview que
// MultiTitlePreviewHandler, mais avec la capability career.progression
// réellement résolue.
type MultiTitlePlayerPreviewHandler struct {
	dataFactory     PlayerCareerAdapterFactory
	semanticFactory SemanticAdapterFactory
	defaultSlug     string
	logger          *slog.Logger
}

// NewMultiTitlePlayerPreviewHandler injecte les factories.
func NewMultiTitlePlayerPreviewHandler(
	data PlayerCareerAdapterFactory,
	semantic SemanticAdapterFactory,
	defaultSlug string,
	logger *slog.Logger,
) *MultiTitlePlayerPreviewHandler {
	if logger == nil {
		logger = slog.Default()
	}
	if defaultSlug == "" {
		defaultSlug = "halo_infinite"
	}
	return &MultiTitlePlayerPreviewHandler{
		dataFactory:     data,
		semanticFactory: semantic,
		defaultSlug:     defaultSlug,
		logger:          logger,
	}
}

// GetCareerPreview gère la requête.
func (h *MultiTitlePlayerPreviewHandler) GetCareerPreview(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "missing_player_slug", "player_slug requis")
		return
	}
	locale := r.URL.Query().Get("locale")
	if locale == "" {
		locale = "fr"
	}

	data, err := h.dataFactory(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	titleSlug := data.TitleSlug()
	if titleSlug == "" {
		titleSlug = h.defaultSlug
	}

	semantic, err := h.semanticFactory(r.Context(), titleSlug)
	if err != nil {
		writeError(w, http.StatusNotFound, "title_semantic_not_found", err.Error())
		return
	}

	dto := careerPreviewDTO{
		TitleSlug:        titleSlug,
		Locale:           locale,
		XUID:             "", // sera renseigné après LoadCareerSnapshot
		CapabilityCareer: data.Capabilities()[games.CapCareerProgression],
	}

	snap, err := data.LoadCareerSnapshot(r.Context(), "", canonical.CareerOptions{})
	switch {
	case errors.Is(err, games.ErrCapabilityNotSupported):
		dto.NotSupportedReason = "career_progression_not_exposed_for_title"
		writeJSON(w, http.StatusOK, dto)
		return
	case err != nil:
		writeError(w, http.StatusInternalServerError, "load_failed", err.Error())
		return
	}

	if snap == nil {
		writeJSON(w, http.StatusOK, dto)
		return
	}

	dto.XUID = snap.Player.XUID

	if snap.CurrentRank != nil {
		dto.CurrentRank = &assetReferenceDTO{
			Kind:         snap.CurrentRank.Kind,
			ID:           snap.CurrentRank.ID,
			DefaultLabel: snap.CurrentRank.DefaultLabel,
		}
	}
	if snap.CurrentXP != nil {
		dto.CurrentXP = labeledIntFromCanonical(canonical.FieldCurrentXP, *snap.CurrentXP, locale, semantic)
	}
	if snap.XPForNextRank != nil {
		dto.XPForNextRank = labeledIntFromCanonical(canonical.FieldXPForNextRank, *snap.XPForNextRank, locale, semantic)
	}

	h.logger.Debug("multi_title_player_preview_served",
		"title_slug", titleSlug,
		"locale", locale,
		"player_slug", slug,
		"capability_career", string(dto.CapabilityCareer),
	)

	writeJSON(w, http.StatusOK, dto)
	_ = fmt.Sprintf // import préservé pour future extensibilité
}
