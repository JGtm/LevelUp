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

// ErrLabWaypointUnavailable : l'explorateur d'API live n'est pas câblé (aucune
// source de token Spartan disponible au démarrage).
var ErrLabWaypointUnavailable = errors.New("lab waypoint explorer unavailable")

// ErrLabWaypointInvalid : segment / asset_id / version_id manquant ou invalide.
var ErrLabWaypointInvalid = errors.New("lab waypoint query invalid")

// WaypointExplorerFunc exécute un appel live à l'API Discovery UGC (résolution
// d'un token Spartan + FetchAsset). Injectée depuis server.go (où le pool de
// tokens et le client halo sont disponibles) pour garder ce service découplé de
// halo/auth et testable.
type WaypointExplorerFunc func(ctx context.Context, q domain.LabWaypointQuery) (*domain.LabWaypointResponse, error)

// LabService orchestre les panneaux du Lab interne.
type LabService struct {
	cfg      *config.AppConfig
	provider port.LabProvider
	explore  WaypointExplorerFunc
}

// NewLabService crée un LabService.
func NewLabService(cfg *config.AppConfig, provider port.LabProvider) *LabService {
	return &LabService{cfg: cfg, provider: provider}
}

// WithWaypointExplorer câble l'explorateur d'API live (Lab). Sans lui,
// ExploreWaypoint renvoie ErrLabWaypointUnavailable.
func (s *LabService) WithWaypointExplorer(fn WaypointExplorerFunc) *LabService {
	s.explore = fn
	return s
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

// validLabSegments liste les AssetType acceptés par l'explorateur d'API (miroir
// de halo.AssetTypeToEndpoint, sans dépendre du package halo dans ce service).
var validLabSegments = map[string]bool{
	domain.LabSegmentMap:         true,
	domain.LabSegmentPlaylist:    true,
	domain.LabSegmentPair:        true,
	domain.LabSegmentGameVariant: true,
}

// ExploreWaypoint exécute un appel live Discovery UGC pour un asset donné
// (Lab). Valide la requête, puis délègue à l'explorateur injecté.
func (s *LabService) ExploreWaypoint(ctx context.Context, q domain.LabWaypointQuery) (*domain.LabWaypointResponse, error) {
	if err := s.requireAccess(); err != nil {
		return nil, err
	}
	if !validLabSegments[q.Segment] || q.AssetID == "" || q.VersionID == "" {
		return nil, ErrLabWaypointInvalid
	}
	if s.explore == nil {
		return nil, ErrLabWaypointUnavailable
	}
	return s.explore(ctx, q)
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
