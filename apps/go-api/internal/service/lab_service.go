// Package service — lab_service.go : orchestration du Lab interne.
package service

import (
	"context"
	"errors"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

const (
	defaultLabLimit = 12
	maxLabLimit     = 50
)

// ErrLabForbidden est retournée quand l'instance ne permet pas la gestion interne.
var ErrLabForbidden = errors.New("lab access forbidden")

// LabService orchestre les trois panneaux du Lab.
type LabService struct {
	cfg      *config.AppConfig
	provider port.LabProvider
}

// NewLabService crée un LabService.
func NewLabService(cfg *config.AppConfig, provider port.LabProvider) *LabService {
	return &LabService{cfg: cfg, provider: provider}
}

// GetResources charge l'explorateur de ressources pour le titre courant.
func (s *LabService) GetResources(
	ctx context.Context,
	query domain.LabResourcesQuery,
) (*domain.LabResourcesResponse, error) {
	if err := s.requireAccess(); err != nil {
		return nil, err
	}
	return s.provider.GetResources(ctx, ctxkeys.TitleSlug(ctx), normalizeLabQuery(query))
}

// GetContracts charge le diff OpenAPI entre Go et la référence FastAPI.
func (s *LabService) GetContracts(ctx context.Context) (*domain.LabContractsResponse, error) {
	if err := s.requireAccess(); err != nil {
		return nil, err
	}
	return s.provider.GetContracts(ctx)
}

// GetDiagnostics charge les diagnostics d'instance pour le titre courant.
func (s *LabService) GetDiagnostics(ctx context.Context) (*domain.LabDiagnosticsResponse, error) {
	if err := s.requireAccess(); err != nil {
		return nil, err
	}
	return s.provider.GetDiagnostics(ctx, ctxkeys.TitleSlug(ctx))
}

func (s *LabService) requireAccess() error {
	settings, err := s.cfg.LoadAppSettings()
	if err != nil {
		// Fail-closed (2026-06-14, PMT-14 vol. C) : si app_settings.json existe
		// mais est illisible/corrompu, REFUSER l'accès au Lab (outil de gestion
		// d'instance) plutôt que de l'ouvrir par défaut. Un fichier ABSENT renvoie
		// un map vide SANS erreur (cf. config.LoadAppSettings) → le défaut
		// « autorisé quand non configuré » est préservé.
		return ErrLabForbidden
	}
	if raw, ok := settings["can_manage_instance"]; ok {
		if allowed, ok := raw.(bool); ok && !allowed {
			return ErrLabForbidden
		}
	}
	return nil
}

func normalizeLabQuery(query domain.LabResourcesQuery) domain.LabResourcesQuery {
	if query.Limit <= 0 {
		query.Limit = defaultLabLimit
	}
	if query.Limit > maxLabLimit {
		query.Limit = maxLabLimit
	}
	return query
}
