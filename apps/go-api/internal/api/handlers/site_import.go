// Package handlers — site_import.go : l'import des données d'un joueur depuis une instance hébergée (lot L22.b,
// ADR 0045), côté application de bureau : POST /api/v1/import/site démarre l'import, GET /api/v1/import/site en
// rend l'état. Montées sous la capacité ImportFromSite seulement.
package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/service"
	"levelup/go-api/internal/transfer"
)

// Codes des refus de la demande d'import (avant le démarrage).
const (
	codeImportRunning   = "import_already_running"
	codeImportNoPlayer  = "import_no_tracked_player"
	codeImportAmbiguous = "import_player_ambiguous"
	codeImportNoXuid    = "import_player_without_xuid"
)

// SiteImportHandler sert les routes d'import depuis le site.
type SiteImportHandler struct {
	svc *service.SiteImport
}

// NewSiteImportHandler crée le handler sur le service de l'import.
func NewSiteImportHandler(svc *service.SiteImport) *SiteImportHandler {
	return &SiteImportHandler{svc: svc}
}

// Mount enregistre les deux routes au point de montage de l'appelant.
func (h *SiteImportHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/import/site", h.handleStart,
		humacore.Op("postSiteImport", "Import the player data from a hosted instance", "import"),
		humacore.DefaultStatus(http.StatusAccepted))
	huma.Get(api, "/import/site", h.handleStatus,
		humacore.Op("getSiteImportStatus", "State of the import from a hosted instance", "import"))
}

type siteImportStartInput struct {
	Body domain.SiteImportRequest
}

type siteImportStatusOutput struct {
	Body *domain.SiteImportStatus
}

// handleStart valide la demande et lance l'import en arrière-plan (202) ; l'état se lit par GET /import/site.
func (h *SiteImportHandler) handleStart(ctx context.Context, in *siteImportStartInput) (*siteImportStatusOutput, error) {
	err := h.svc.Start(service.SiteImportRequest{URL: in.Body.URL, Token: in.Body.Token, Gamertag: in.Body.Gamertag})
	switch {
	case err == nil:
		return &siteImportStatusOutput{Body: statusBody(h.svc.Status())}, nil
	case errors.Is(err, transfer.ErrImportRunning):
		return nil, humacore.NewError(http.StatusConflict, codeImportRunning, "import running")
	case errors.Is(err, service.ErrSiteImportNoPlayer):
		return nil, humacore.NewError(http.StatusConflict, codeImportNoPlayer, "no tracked player")
	case errors.Is(err, service.ErrSiteImportAmbiguous):
		return nil, humacore.NewError(http.StatusBadRequest, codeImportAmbiguous, "player ambiguous")
	case errors.Is(err, service.ErrSiteImportPlayerNoXID):
		return nil, humacore.NewError(http.StatusConflict, codeImportNoXuid, "player without xuid")
	default:
		return nil, humacore.NewErrorFrom(ctx, http.StatusServiceUnavailable, "import_unavailable", err)
	}
}

// handleStatus rend l'état de l'import courant ou du dernier.
func (h *SiteImportHandler) handleStatus(_ context.Context, _ *struct{}) (*siteImportStatusOutput, error) {
	return &siteImportStatusOutput{Body: statusBody(h.svc.Status())}, nil
}

// statusBody convertit l'état de l'exécuteur en réponse d'API ; Matches est le nombre de matchs du registre ajoutés.
func statusBody(st transfer.Status) *domain.SiteImportStatus {
	return &domain.SiteImportStatus{Phase: string(st.Phase), ErrorCode: st.ErrorCode, Inserted: st.Inserted,
		Matches: st.Inserted["shared/match_registry"]}
}
