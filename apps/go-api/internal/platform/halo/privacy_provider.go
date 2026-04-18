// Package halo — privacy_provider.go : interrogation de la privacy d'un compte Halo.
//
// Sprint 54 B : GET /hi/players/{xuid}/matches-privacy
package halo

import (
	"context"
	"encoding/json"
	"fmt"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

// privacyResponse est la réponse brute de l'endpoint matches-privacy.
type privacyResponse struct {
	AllMatchesPrivacy    string `json:"AllMatchesPrivacy"`
	PublicMatchesPrivacy string `json:"PublicMatchesPrivacy"`
	RankedMatchesPrivacy string `json:"RankedMatchesPrivacy"`
	CustomMatchesPrivacy string `json:"CustomMatchesPrivacy"`
}

const defaultStatsHost = "https://halostats.svc.halowaypoint.com"

// GetMatchPrivacy interroge l'API Waypoint pour connaître la privacy du compte.
// Les tokens sont lus depuis le contexte via ctxkeys.
// Retourne un MatchPrivacyInfo avec Hint="auth_required" si l'auth est absente.
func (p *HaloProvider) GetMatchPrivacy(ctx context.Context, xuid string) (*domain.MatchPrivacyInfo, error) {
	if xuid == "" {
		return &domain.MatchPrivacyInfo{Hint: "auth_required"}, nil
	}

	tokens := ctxkeys.HaloTokens(ctx)
	if tokens == nil {
		return &domain.MatchPrivacyInfo{Hint: "auth_required"}, nil
	}

	url := fmt.Sprintf("%s/hi/players/xuid(%s)/matches-privacy", defaultStatsHost, xuid)
	body, err := p.doGet(ctx, url, tokens)
	if err != nil {
		// Privacy non critique — on ne bloque pas le bootstrap.
		return &domain.MatchPrivacyInfo{IsPartial: true, Hint: "fetch_error"}, nil
	}

	var resp privacyResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return &domain.MatchPrivacyInfo{IsPartial: true, Hint: "parse_error"}, nil
	}

	info := &domain.MatchPrivacyInfo{}
	if resp.AllMatchesPrivacy == "Private" {
		info.IsPrivate = true
		info.Hint = "full_private"
	} else if resp.RankedMatchesPrivacy == "Private" || resp.PublicMatchesPrivacy == "Private" {
		info.IsPartial = true
		info.Hint = "partial_private"
	}
	return info, nil
}
