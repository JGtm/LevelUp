// Package service — campaign_service.go : les campagnes d'amélioration d'un joueur pour la couche HTTP.
package service

import (
	"context"
	"time"

	"levelup/go-api/internal/campaign"
	"levelup/go-api/internal/port"
)

// CampaignService implémente port.CampaignService : il résout le joueur puis délègue à
// `campaign.Service`, qui porte les règles de campagne.
type CampaignService struct {
	resolve port.CampaignScopeResolver
}

var _ port.CampaignService = (*CampaignService)(nil)

// NewCampaignService construit le service ; `resolve` donne les dépôts de campagne d'un joueur.
func NewCampaignService(resolve port.CampaignScopeResolver) *CampaignService {
	return &CampaignService{resolve: resolve}
}

// scope résout le joueur ou rend un *port.ProgressionError de nature ProgErrPlayerNotFound.
func (s *CampaignService) scope(ctx context.Context, playerSlug string) (port.CampaignScope, *campaign.Service, error) {
	sc, err := s.resolve(ctx, playerSlug)
	if err != nil {
		return port.CampaignScope{}, nil, &port.ProgressionError{Kind: port.ProgErrPlayerNotFound, Err: err}
	}
	return sc, campaign.NewService(sc.Repo, sc.Samples), nil
}

// Start crée une campagne pour le joueur résolu.
func (s *CampaignService) Start(ctx context.Context, playerSlug string, p campaign.StartParams, now time.Time) (campaign.ImprovementCampaign, error) {
	sc, svc, err := s.scope(ctx, playerSlug)
	if err != nil {
		return campaign.ImprovementCampaign{}, err
	}
	p.UserID = sc.XUID
	return svc.StartCampaign(ctx, p, now)
}

// Active rend la campagne active du joueur pour le titre.
func (s *CampaignService) Active(ctx context.Context, playerSlug, titleSlug string) (campaign.ImprovementCampaign, error) {
	sc, svc, err := s.scope(ctx, playerSlug)
	if err != nil {
		return campaign.ImprovementCampaign{}, err
	}
	return svc.GetActive(ctx, sc.XUID, titleSlug)
}

// Ended rend les campagnes closes du joueur pour le titre.
func (s *CampaignService) Ended(ctx context.Context, playerSlug, titleSlug string) ([]campaign.ImprovementCampaign, error) {
	sc, svc, err := s.scope(ctx, playerSlug)
	if err != nil {
		return nil, err
	}
	return svc.ListEnded(ctx, sc.XUID, titleSlug)
}

// ByID rend une campagne et ses défis liés.
func (s *CampaignService) ByID(ctx context.Context, playerSlug, id string) (campaign.ImprovementCampaign, error) {
	_, svc, err := s.scope(ctx, playerSlug)
	if err != nil {
		return campaign.ImprovementCampaign{}, err
	}
	return svc.GetByID(ctx, id)
}

// Pause met une campagne active en pause.
func (s *CampaignService) Pause(ctx context.Context, playerSlug, id string) error {
	_, svc, err := s.scope(ctx, playerSlug)
	if err != nil {
		return err
	}
	return svc.PauseCampaign(ctx, id)
}

// Resume reprend une campagne en pause.
func (s *CampaignService) Resume(ctx context.Context, playerSlug, id string) error {
	_, svc, err := s.scope(ctx, playerSlug)
	if err != nil {
		return err
	}
	return svc.ResumeCampaign(ctx, id)
}

// Close clôture une campagne.
func (s *CampaignService) Close(ctx context.Context, playerSlug, id string, now time.Time) error {
	_, svc, err := s.scope(ctx, playerSlug)
	if err != nil {
		return err
	}
	return svc.CloseCampaign(ctx, id, now)
}

// Abandon abandonne une campagne.
func (s *CampaignService) Abandon(ctx context.Context, playerSlug, id string, now time.Time) error {
	_, svc, err := s.scope(ctx, playerSlug)
	if err != nil {
		return err
	}
	return svc.AbandonCampaign(ctx, id, now)
}
