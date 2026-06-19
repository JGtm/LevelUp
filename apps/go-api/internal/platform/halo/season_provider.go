// Package halo — season_provider.go : récupération des calendriers de saisons Waypoint.
//
// Sprint 54 A : SeasonCalendar.json + CsrSeasonCalendar.json
package halo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
)

const (
	defaultGameCMSHost = "https://gamecms-hacs.svc.halowaypoint.com"
	// Suffixes post-préfixe de jeu : le segment /hi|/h5 est injecté à l'usage via
	// p.gamePrefix(ctx) (MT-01), plus en dur dans le chemin.
	seasonCalendarPathSuffix    = "/Progression/file/SeasonCalendar.json"
	csrSeasonCalendarPathSuffix = "/Progression/file/CsrSeasonCalendar.json"
)

// seasonCalendarRaw est la structure brute du JSON SeasonCalendar.
type seasonCalendarRaw struct {
	Seasons []struct {
		SeasonMetadataPath string `json:"SeasonMetadataPath"`
		StartDate          string `json:"StartDate"`
		EndDate            string `json:"EndDate,omitempty"`
	} `json:"Seasons"`
}

// csrSeasonRaw est la structure brute d'une saison CSR.
type csrSeasonRaw struct {
	Seasons []struct {
		CsrSeasonFilePath string `json:"CsrSeasonFilePath"`
		StartDate         string `json:"StartDate"`
		EndDate           string `json:"EndDate,omitempty"`
	} `json:"Seasons"`
}

// FetchSeasonCalendar récupère le calendrier de saisons standard Halo Infinite.
// Les tokens sont lus depuis le contexte via ctxkeys.
func (p *HaloProvider) FetchSeasonCalendar(ctx context.Context, titleID string) ([]domain.SeasonCalendar, []byte, error) {
	tokens := ctxkeys.HaloTokens(ctx)
	if tokens == nil {
		return nil, nil, fmt.Errorf("FetchSeasonCalendar: tokens absents du contexte")
	}

	url := p.hostFor(ctx, games.EndpointGameCMS, p.gameCMSBaseURL, defaultGameCMSHost) + "/" + p.gamePrefix(ctx) + seasonCalendarPathSuffix
	body, err := p.doGet(ctx, url, tokens)
	if err != nil {
		return nil, nil, fmt.Errorf("FetchSeasonCalendar: %w", err)
	}

	var raw seasonCalendarRaw
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, body, fmt.Errorf("FetchSeasonCalendar parse: %w", err)
	}

	hash := ContentHash(body)
	fetchedAt := time.Now().UTC()
	seasons := make([]domain.SeasonCalendar, 0, len(raw.Seasons))
	for i, s := range raw.Seasons {
		sc := domain.SeasonCalendar{
			TitleID:     titleID,
			SeasonID:    fmt.Sprintf("s%d", i+1),
			Name:        s.SeasonMetadataPath,
			FetchedAt:   fetchedAt,
			ContentHash: hash,
			SourceURL:   url,
		}
		if t, err := time.Parse(time.RFC3339, s.StartDate); err == nil {
			sc.StartDate = t
		}
		if s.EndDate != "" {
			if t, err := time.Parse(time.RFC3339, s.EndDate); err == nil {
				sc.EndDate = &t
			}
		}
		seasons = append(seasons, sc)
	}
	return seasons, body, nil
}

// FetchCSRSeasonCalendar récupère le calendrier de saisons CSR.
func (p *HaloProvider) FetchCSRSeasonCalendar(ctx context.Context, titleID string) ([]domain.CSRSeasonCalendar, []byte, error) {
	tokens := ctxkeys.HaloTokens(ctx)
	if tokens == nil {
		return nil, nil, fmt.Errorf("FetchCSRSeasonCalendar: tokens absents du contexte")
	}

	url := p.hostFor(ctx, games.EndpointGameCMS, p.gameCMSBaseURL, defaultGameCMSHost) + "/" + p.gamePrefix(ctx) + csrSeasonCalendarPathSuffix
	body, err := p.doGet(ctx, url, tokens)
	if err != nil {
		return nil, nil, fmt.Errorf("FetchCSRSeasonCalendar: %w", err)
	}

	var raw csrSeasonRaw
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, body, fmt.Errorf("FetchCSRSeasonCalendar parse: %w", err)
	}

	hash := ContentHash(body)
	fetchedAt := time.Now().UTC()
	seasons := make([]domain.CSRSeasonCalendar, 0, len(raw.Seasons))
	for i, s := range raw.Seasons {
		cs := domain.CSRSeasonCalendar{
			TitleID:     titleID,
			SeasonID:    fmt.Sprintf("csr-s%d", i+1),
			Name:        s.CsrSeasonFilePath,
			FetchedAt:   fetchedAt,
			ContentHash: hash,
			SourceURL:   url,
		}
		if t, err := time.Parse(time.RFC3339, s.StartDate); err == nil {
			cs.StartDate = t
		}
		if s.EndDate != "" {
			if t, err := time.Parse(time.RFC3339, s.EndDate); err == nil {
				cs.EndDate = &t
			}
		}
		seasons = append(seasons, cs)
	}
	return seasons, body, nil
}

// ContentHash retourne le SHA-256 hex du contenu brut.
func ContentHash(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
