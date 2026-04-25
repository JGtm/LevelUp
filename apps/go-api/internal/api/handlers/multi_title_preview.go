// Phase C du plan multi-titres : endpoint preview démontrant le pipeline
// complet TitleDataAdapter + TitleSemanticAdapter sur le canonique services.
//
// Cet endpoint est uniquement disponible derrière MULTI_TITLE_API_ENABLED.
// Il sert de proof-of-concept end-to-end avant la bascule effective des
// endpoints produit (Phase C suite + Phase D côté frontend).
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/mappings"
)

// MultiTitlePreviewHandler expose un preview de la couche canonique pour
// déboguer / valider le pipeline. Pas un endpoint produit.
type MultiTitlePreviewHandler struct {
	resolver games.Resolver
	logger   *slog.Logger
	recorder *mappings.LookupRecorder
}

// NewMultiTitlePreviewHandler injecte le resolver pour résoudre l'adapter
// du titre courant. Un LookupRecorder partagé est créé pour rate-limiter les
// `field_lookup_missing` quand un caller demande un FieldKey absent du TOML.
func NewMultiTitlePreviewHandler(r games.Resolver, logger *slog.Logger) *MultiTitlePreviewHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &MultiTitlePreviewHandler{
		resolver: r,
		logger:   logger,
		recorder: mappings.NewLookupRecorder(logger),
	}
}

// careerPreviewDTO combine la lecture canonique et les libellés sémantiques
// dans une payload locale-aware.
type careerPreviewDTO struct {
	TitleSlug          string                 `json:"title_slug"`
	Locale             string                 `json:"locale"`
	XUID               string                 `json:"xuid"`
	CapabilityCareer   games.CapabilityStatus `json:"capability_career_progression"`
	CurrentRank        *assetReferenceDTO     `json:"current_rank,omitempty"`
	CurrentXP          *labeledIntDTO         `json:"current_xp,omitempty"`
	XPForNextRank      *labeledIntDTO         `json:"xp_for_next_rank,omitempty"`
	NotSupportedReason string                 `json:"not_supported_reason,omitempty"`
}

type assetReferenceDTO struct {
	Kind         string `json:"kind"`
	ID           string `json:"id"`
	DefaultLabel string `json:"default_label,omitempty"`
}

type labeledIntDTO struct {
	Label    string `json:"label"`
	Value    int    `json:"value"`
	Format   string `json:"format,omitempty"`
	Fallback bool   `json:"fallback,omitempty"`
}

// GetCareerPreview retourne un payload preview pour le titre demandé.
//
// Route : GET /api/v1/titles/{slug}/preview/career?xuid=...&locale=fr
func (h *MultiTitlePreviewHandler) GetCareerPreview(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	xuid := r.URL.Query().Get("xuid")
	locale := r.URL.Query().Get("locale")
	if locale == "" {
		locale = "fr"
	}

	data, err := h.resolver.Data(slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "title_not_found", err.Error())
		return
	}
	semantic, err := h.resolver.Semantic(slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "title_semantic_not_found", err.Error())
		return
	}

	dto := careerPreviewDTO{
		TitleSlug:        slug,
		Locale:           locale,
		XUID:             xuid,
		CapabilityCareer: data.Capabilities()[games.CapCareerProgression],
	}

	snap, err := data.LoadCareerSnapshot(r.Context(), xuid, canonical.CareerOptions{})
	switch {
	case errors.Is(err, games.ErrCapabilityNotSupported):
		dto.NotSupportedReason = "career_progression_not_exposed_for_title"
		h.writeJSON(w, http.StatusOK, dto)
		return
	case err != nil:
		writeError(w, http.StatusInternalServerError, "load_failed", err.Error())
		return
	}

	if snap == nil {
		h.writeJSON(w, http.StatusOK, dto)
		return
	}

	if snap.CurrentRank != nil {
		dto.CurrentRank = &assetReferenceDTO{
			Kind:         snap.CurrentRank.Kind,
			ID:           snap.CurrentRank.ID,
			DefaultLabel: snap.CurrentRank.DefaultLabel,
		}
	}
	if snap.CurrentXP != nil {
		dto.CurrentXP = labeledIntFromCanonical(canonical.FieldCurrentXP, *snap.CurrentXP, locale, semantic, h.recorder)
	}
	if snap.XPForNextRank != nil {
		dto.XPForNextRank = labeledIntFromCanonical(canonical.FieldXPForNextRank, *snap.XPForNextRank, locale, semantic, h.recorder)
	}

	h.writeJSON(w, http.StatusOK, dto)
	_ = context.Background // import préservé
}

// labeledIntFromCanonical résout le libellé d'un FieldKey via le SemanticAdapter
// et empaquette la valeur avec ce libellé.
//
// Si recorder est fourni et que le FieldKey est absent du TOML (cas de bug ou
// d'oubli au moment d'enrichir le canonique sans mettre à jour le TOML), un
// log Warn `field_lookup_missing` rate-limité est émis.
func labeledIntFromCanonical(
	key canonical.FieldKey, value int, locale string, semantic games.TitleSemanticAdapter,
	recorder *mappings.LookupRecorder,
) *labeledIntDTO {
	out := &labeledIntDTO{Value: value}
	if semantic == nil {
		out.Label = string(key)
		out.Fallback = true
		return out
	}
	mapping, ok := semantic.Fields().Get(key)
	if !ok {
		if recorder != nil {
			recorder.Record(semantic.TitleSlug(), string(key), locale)
		}
		out.Label = string(key)
		out.Fallback = true
		return out
	}
	label, fellback := mapping.Label(locale)
	out.Label = label
	out.Fallback = fellback
	out.Format = string(mapping.Format)
	return out
}

func (h *MultiTitlePreviewHandler) writeJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}
