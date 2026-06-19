// Package halo — medal_provider.go : fetch des métadonnées médailles Waypoint.
//
// Sprint 54 D1.2 : commande `medals` du CLI refresh-metadata.
// Endpoint : GET /hi/Progression/file/medals/metadata.json
package halo

import (
	"context"
	"encoding/json"
	"fmt"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/metadata"
)

// medalsMetadataPathSuffix : partie post-préfixe de jeu (segment /hi|/h5 injecté
// à l'usage via p.gamePrefix(ctx), MT-01).
const medalsMetadataPathSuffix = "/Progression/file/medals/metadata.json"

// medalMetadataRaw est la structure brute du JSON Waypoint.
type medalMetadataRaw struct {
	Medals []struct {
		NameID        string `json:"NameId"`
		DescriptionID string `json:"DescriptionId"`
		SpriteIndex   int    `json:"SpriteIndex"`
		Difficulty    string `json:"Difficulty"`
		Type          string `json:"Type"`
		PersonalScore int    `json:"PersonalScore"`
		MedalID       int64  `json:"MedalId"`
	} `json:"Medals"`
}

// FetchMedalsMetadata récupère les métadonnées de médailles depuis Waypoint.
// Retourne les entrées parsées, le corps brut pour le hash, et une erreur éventuelle.
func (p *HaloProvider) FetchMedalsMetadata(ctx context.Context, titleID string) ([]metadata.MedalEntry, []byte, error) {
	tokens := ctxkeys.HaloTokens(ctx)
	if tokens == nil {
		return nil, nil, fmt.Errorf("FetchMedalsMetadata: tokens absents du contexte")
	}

	url := p.hostFor(ctx, games.EndpointGameCMS, p.gameCMSBaseURL, defaultGameCMSHost) + "/" + p.gamePrefix(ctx) + medalsMetadataPathSuffix
	body, err := p.doGet(ctx, url, tokens)
	if err != nil {
		return nil, nil, fmt.Errorf("FetchMedalsMetadata: %w", err)
	}

	var raw medalMetadataRaw
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, body, fmt.Errorf("FetchMedalsMetadata parse: %w", err)
	}

	entries := make([]metadata.MedalEntry, 0, len(raw.Medals))
	for _, m := range raw.Medals {
		rowJSON, _ := json.Marshal(m)
		entries = append(entries, metadata.MedalEntry{
			TitleID:   titleID,
			MedalID:   m.MedalID,
			Label:     m.NameID,
			Category:  m.Type,
			Rarity:    m.Difficulty,
			SpriteIdx: m.SpriteIndex,
			RawJSON:   string(rowJSON),
		})
	}

	return entries, body, nil
}
