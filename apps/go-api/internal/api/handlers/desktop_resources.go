// Package handlers — desktop_resources.go : l'état du paquet de ressources de l'application de bureau
// (ADR 0045, amendement L24) : GET /api/v1/desktop/resources rend l'étape de la préparation (téléchargement,
// vérification, paquet prêt, échec avec son code), POST .../retry relance le téléchargement du paquet du moteur.
//
// Les routes ne sont montées que sous la capacité UpdateApp et exigent le cookie de la coquille (garde racine).
package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/resourcepack"
)

// Codes des refus.
const (
	codeResourcesUnavailable = "resources_unavailable"
	codeResourcesBusy        = "resources_busy"
)

// ResourcesService : ce que les routes demandent au gestionnaire du paquet (satisfait par *resourcepack.Manager).
type ResourcesService interface {
	Status() resourcepack.Status
	Retry() (resourcepack.Status, error)
}

// DesktopResourcesHandler sert les routes du paquet de ressources.
type DesktopResourcesHandler struct {
	// service rend le gestionnaire du processus, nil tant qu'il n'est pas posé.
	service func() ResourcesService
}

// NewDesktopResourcesHandler crée le handler ; service est relu à chaque requête (le gestionnaire naît au démarrage).
func NewDesktopResourcesHandler(service func() ResourcesService) *DesktopResourcesHandler {
	return &DesktopResourcesHandler{service: service}
}

// Mount enregistre les deux routes au point de montage de l'appelant.
func (h *DesktopResourcesHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/desktop/resources", h.handleStatus,
		humacore.Op("getDesktopResources", "State of the resource pack (desktop app)", "desktop"))
	huma.Post(api, "/desktop/resources/retry", h.handleRetry,
		humacore.Op("postDesktopResourcesRetry", "Download the resource pack of this engine again (desktop app)", "desktop"),
		humacore.DefaultStatus(http.StatusAccepted))
}

type resourcesOutput struct {
	Body *resourcepack.Status
}

func (h *DesktopResourcesHandler) handleStatus(_ context.Context, _ *struct{}) (*resourcesOutput, error) {
	svc := h.service()
	if svc == nil {
		return nil, humacore.NewError(http.StatusServiceUnavailable, codeResourcesUnavailable, "resource pack not managed")
	}
	st := svc.Status()
	return &resourcesOutput{Body: &st}, nil
}

func (h *DesktopResourcesHandler) handleRetry(ctx context.Context, _ *struct{}) (*resourcesOutput, error) {
	svc := h.service()
	if svc == nil {
		return nil, humacore.NewError(http.StatusServiceUnavailable, codeResourcesUnavailable, "resource pack not managed")
	}
	st, err := svc.Retry()
	switch {
	case err == nil:
		return &resourcesOutput{Body: &st}, nil
	case errors.Is(err, resourcepack.ErrBusy):
		return nil, humacore.NewError(http.StatusConflict, codeResourcesBusy, "a preparation is already running")
	default:
		return nil, humacore.NewErrorFrom(ctx, http.StatusServiceUnavailable, codeResourcesUnavailable, err)
	}
}
