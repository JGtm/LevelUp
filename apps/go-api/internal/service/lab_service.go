// Package service — lab_service.go : diagnostic d'instance (ex-Lab).
//
// A3.5 (DC-9, 2026-07-10) : le Lab est retiré de l'app — ne reste que
// GetDiagnostics (panneau parité + garde-fous médailles de l'onglet admin
// Données). Le gate d'instance can_manage_instance subsiste (kill-switch).
package service

import (
	"context"
	"errors"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// ErrLabForbidden est retournée quand l'instance ne permet pas la gestion interne.
var ErrLabForbidden = errors.New("lab access forbidden")

// LabService orchestre le diagnostic d'instance.
type LabService struct {
	cfg      *config.AppConfig
	provider port.LabProvider
}

// NewLabService crée un LabService.
func NewLabService(cfg *config.AppConfig, provider port.LabProvider) *LabService {
	return &LabService{cfg: cfg, provider: provider}
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
		// mais est illisible/corrompu, REFUSER l'accès (outil de gestion
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
