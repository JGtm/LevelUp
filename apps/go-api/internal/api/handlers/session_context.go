// Package handlers — session_context.go : handler HTTP pour la gestion du contexte session.
//
// Endpoints :
//
//	POST /session/context  → SessionContextResponse
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) au même point de
// montage racine (pas de préfixe /players/{player_slug}) et enregistre le POST
// via huma.Post. Logique métier inchangée (session store + registry titres),
// seul le wrapping HTTP change.
//
// Le corps est lu via RawBody (pas de Body typé) pour reproduire EXACTEMENT le
// contrat de décodage d'origine : un JSON invalide renvoie 400 {invalid_body}
// (parse maison) et non le 422 de validation Huma qu'un Body typé produirait.
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/session"
	"levelup/go-api/internal/service"
)

// SessionHandler gère les endpoints de session.
type SessionHandler struct {
	store *session.Store
}

// NewSessionHandler crée un SessionHandler.
func NewSessionHandler(store *session.Store) *SessionHandler {
	return &SessionHandler{store: store}
}

// Mount enregistre la route via Huma au même point de montage que le routeur
// chi fourni (racine : POST /session/context, pas de path param parent).
func (h *SessionHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/session/context", h.handlePostContext, humacore.Op("updateSessionContext", "Mise à jour du contexte de session", "bootstrap"))
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// sessionContextInput : corps brut décodé maison. RawBody (pas Body typé) →
// préserve le contrat 400 {invalid_body} sur JSON invalide (un Body typé
// renverrait le 422 de validation Huma).
type sessionContextInput struct {
	RawBody []byte
}

type sessionContextOutput struct{ Body domain.SessionContextResponse }

// ─── Endpoint ────────────────────────────────────────────────────────────────

// handlePostContext met à jour le contexte de session (player_slug, title_slug,
// locale).
//
// POST /session/context
func (h *SessionHandler) handlePostContext(ctx context.Context, in *sessionContextInput) (*sessionContextOutput, error) {
	sess := middleware.GetSession(ctx)
	if sess == nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "no_session", "session non initialisée")
	}

	var req domain.SessionContextRequest
	if err := json.NewDecoder(bytes.NewReader(in.RawBody)).Decode(&req); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", "corps JSON invalide")
	}

	if req.PlayerSlug != nil {
		sess.CurrentPlayerSlug = req.PlayerSlug
	}
	if req.TitleSlug != nil && *req.TitleSlug != "" {
		// Sprint 44 : switch titre — valider que le slug est connu. MT-16 : le
		// registre PARTAGÉ (DefaultRegistry) connaît les titres découverts en
		// config → on peut switcher vers un 2e titre déclaré.
		reg := titlePkg.DefaultRegistry()
		if reg.Exists(*req.TitleSlug) {
			sess.CurrentTitleSlug = *req.TitleSlug
			// Invalider le joueur courant lors d'un switch de titre
			// (le frontend devra re-sélectionner).
			sess.CurrentPlayerSlug = nil
		}
	}
	if req.Locale != nil {
		sess.Locale = *req.Locale
	}

	if err := h.store.Save(sess); err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "session_save_error", err.Error())
	}

	return &sessionContextOutput{Body: domain.SessionContextResponse{
		CurrentPlayerSlug: sess.CurrentPlayerSlug,
		CurrentTitleSlug:  sess.CurrentTitleSlug,
		AvailableTitles:   service.BuildAvailableTitles(),
		Locale:            sess.Locale,
		HintsVisible:      sess.HintsVisible,
		AuthReady:         sess.AuthReady,
	}}, nil
}
