// Package handlers — session_page.go : handler HTTP pour la page détail de session.
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// (préfixe /players/{player_slug} + middleware ownership/title hérités, lit
// {player_slug} parent) et enregistre le POST via huma.Post. Logique métier
// inchangée (SessionPageService), seul le wrapping HTTP change.
//
// Le corps est pris en RawBody []byte (pas Body domain.SessionPageRequest) pour
// reproduire EXACTEMENT le contrat d'erreur d'origine : un JSON invalide renvoie
// 400 invalid_json (decode maison), PAS le 422 de validation Huma. Un corps
// absent (ContentLength 0 → RawBody vide) est toléré comme requête zéro-valeur.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// SessionPageHandler gère l'endpoint POST /pages/sessions/detail.
type SessionPageHandler struct {
	newSvc ServiceFactory[port.SessionPageService]
}

// NewSessionPageHandler crée un SessionPageHandler.
func NewSessionPageHandler(newSvc ServiceFactory[port.SessionPageService]) *SessionPageHandler {
	return &SessionPageHandler{newSvc: newSvc}
}

// Mount enregistre la route via Huma sur le sous-routeur chi (préfixe
// /players/{player_slug} + middleware ownership/title hérités).
func (h *SessionPageHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/pages/sessions/detail", h.GetPage, humacore.Op("postSessionDetailPage", "Détail d'une session avec suggestion de comparaison", "timeseries"))
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// sessionPageInput : {player_slug} parent + corps brut décodé à la main.
// RawBody (pas Body) → Huma ne valide PAS le JSON, le handler reproduit le
// contrat d'erreur d'origine (invalid_json / invalid_body).
type sessionPageInput struct {
	PlayerSlug string `path:"player_slug"`
	RawBody    []byte
}

type sessionPageOutput struct {
	Body domain.SessionPageResponse
}

// GetPage retourne la page de détail d'une session.
func (h *SessionPageHandler) GetPage(ctx context.Context, in *sessionPageInput) (*sessionPageOutput, error) {
	slug := in.PlayerSlug
	svc, err := h.newSvc(ctx, slug)
	if err != nil {
		slog.WarnContext(ctx, "session page: joueur introuvable", "player_slug", slug, "err", err)
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}

	var req domain.SessionPageRequest
	if len(in.RawBody) > 0 {
		if err := json.Unmarshal(in.RawBody, &req); err != nil {
			slog.WarnContext(ctx, "session page: corps invalide", "player_slug", slug, "err", err)
			return nil, humacore.NewError(http.StatusBadRequest, "invalid_json", "corps JSON invalide")
		}
	}
	if err := req.Validate(); err != nil {
		slog.WarnContext(ctx, "session page: requête invalide", "player_slug", slug, "err", err)
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
	}

	slog.DebugContext(ctx, "session page: calcul",
		"player_slug", slug,
		"session_label", derefReqString(req.SessionLabel),
		"compare_session_label", derefReqString(req.CompareSessionLabel),
		"enable_compare", req.EnableCompare,
	)

	page, err := svc.GetPage(ctx, req)
	if err != nil {
		var apiErr *domain.APIError
		if errors.As(err, &apiErr) && apiErr.Code == "session_not_found" {
			// Couche B (ADR 0029) : session demandée inexistante dans le périmètre.
			slog.InfoContext(ctx, "session page: session introuvable",
				"player_slug", slug, "session_label", derefReqString(req.SessionLabel))
			return nil, humacore.NewError(http.StatusNotFound, "session_not_found", apiErr.Message)
		}
		slog.ErrorContext(ctx, "session page: erreur service", "player_slug", slug, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "session_page_error", err.Error())
	}

	slog.InfoContext(ctx, "session page: générée",
		"player_slug", slug,
		"session_label", derefReqString(req.SessionLabel),
		"resolved_session", derefSessionLabel(page.CurrentSession),
		"available_sessions", len(page.AvailableSessions),
		"match_count", len(page.Matches),
		"compare_enabled", page.CompareEnabled,
		"compare_session", derefSessionLabel(page.CompareSession),
		"compare_match_count", len(page.CompareMatches),
		"previous_session_label", derefReqString(page.PreviousSessionLabel),
		"next_session_label", derefReqString(page.NextSessionLabel),
	)
	return &sessionPageOutput{Body: page}, nil
}

func derefReqString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func derefSessionLabel(entry *domain.SessionCompareEntry) string {
	if entry == nil {
		return ""
	}
	return entry.SessionLabel
}
