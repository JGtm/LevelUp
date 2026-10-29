package infinite

// gamecms.go — fichiers de progression GameCMS : métadonnées des médailles, calendrier des saisons,
// calendrier des saisons CSR (GET et DTO) ; les entrées de domaine (identifiants de saison, empreinte, horodatage) sont
// construites par l'appelant depuis le DTO et le corps brut rendu.

import (
	"context"
	"encoding/json"
	"fmt"
)

// Chemins GameCMS, après le segment de jeu.
const (
	medalsMetadataPathSuffix    = "/Progression/file/medals/metadata.json"
	seasonCalendarPathSuffix    = "/Progression/file/SeasonCalendar.json"
	csrSeasonCalendarPathSuffix = "/Progression/file/CsrSeasonCalendar.json"
)

// MedalMetadataDTO : une médaille de `medals/metadata.json`.
type MedalMetadataDTO struct {
	NameID        string `json:"NameId"`
	DescriptionID string `json:"DescriptionId"`
	SpriteIndex   int    `json:"SpriteIndex"`
	Difficulty    string `json:"Difficulty"`
	Type          string `json:"Type"`
	PersonalScore int    `json:"PersonalScore"`
	MedalID       int64  `json:"MedalId"`
}

// MedalsMetadataDTO : la réponse brute de `GET /{jeu}/Progression/file/medals/metadata.json`.
type MedalsMetadataDTO struct {
	Medals []MedalMetadataDTO `json:"Medals"`
}

// FetchMedalsMetadata récupère les métadonnées des médailles ([CleGameCMS]). Rend le DTO et le
// corps brut ; sur un corps illisible, le corps est rendu avec l'erreur.
func (c *Client) FetchMedalsMetadata(ctx context.Context, auth Auth) (*MedalsMetadataDTO, []byte, error) {
	body, err := c.get(ctx, requete{url: MedalsMetadataURL(c.endpoints), auth: auth})
	if err != nil {
		return nil, nil, fmt.Errorf("FetchMedalsMetadata: %w", err)
	}
	var dto MedalsMetadataDTO
	if err := json.Unmarshal(body, &dto); err != nil {
		return nil, body, fmt.Errorf("FetchMedalsMetadata parse: %w", err)
	}
	return &dto, body, nil
}

// MedalsMetadataURL rend l'adresse des métadonnées des médailles.
func MedalsMetadataURL(e Endpoints) string { return e.base(CleGameCMS) + medalsMetadataPathSuffix }

// SeasonCalendarDTO : la réponse brute de `SeasonCalendar.json`.
type SeasonCalendarDTO struct {
	Seasons []struct {
		SeasonMetadataPath string `json:"SeasonMetadataPath"`
		StartDate          string `json:"StartDate"`
		EndDate            string `json:"EndDate,omitempty"`
	} `json:"Seasons"`
}

// SeasonCalendarURL rend l'adresse du calendrier des saisons (l'appelant la garde comme source).
func SeasonCalendarURL(e Endpoints) string { return e.base(CleGameCMS) + seasonCalendarPathSuffix }

// FetchSeasonCalendar récupère le calendrier des saisons ([CleGameCMS]). Rend le DTO et le corps
// brut ; sur un corps illisible, le corps est rendu avec l'erreur.
func (c *Client) FetchSeasonCalendar(ctx context.Context, auth Auth) (*SeasonCalendarDTO, []byte, error) {
	body, err := c.get(ctx, requete{url: SeasonCalendarURL(c.endpoints), auth: auth})
	if err != nil {
		return nil, nil, fmt.Errorf("FetchSeasonCalendar: %w", err)
	}
	var dto SeasonCalendarDTO
	if err := json.Unmarshal(body, &dto); err != nil {
		return nil, body, fmt.Errorf("FetchSeasonCalendar parse: %w", err)
	}
	return &dto, body, nil
}

// CSRSeasonCalendarDTO : la réponse brute de `CsrSeasonCalendar.json`.
type CSRSeasonCalendarDTO struct {
	Seasons []struct {
		CsrSeasonFilePath string `json:"CsrSeasonFilePath"`
		StartDate         string `json:"StartDate"`
		EndDate           string `json:"EndDate,omitempty"`
	} `json:"Seasons"`
}

// CSRSeasonCalendarURL rend l'adresse du calendrier des saisons CSR.
func CSRSeasonCalendarURL(e Endpoints) string {
	return e.base(CleGameCMS) + csrSeasonCalendarPathSuffix
}

// FetchCSRSeasonCalendar récupère le calendrier des saisons CSR ([CleGameCMS]). Rend le DTO et le
// corps brut ; sur un corps illisible, le corps est rendu avec l'erreur.
func (c *Client) FetchCSRSeasonCalendar(ctx context.Context, auth Auth) (*CSRSeasonCalendarDTO, []byte, error) {
	body, err := c.get(ctx, requete{url: CSRSeasonCalendarURL(c.endpoints), auth: auth})
	if err != nil {
		return nil, nil, fmt.Errorf("FetchCSRSeasonCalendar: %w", err)
	}
	var dto CSRSeasonCalendarDTO
	if err := json.Unmarshal(body, &dto); err != nil {
		return nil, body, fmt.Errorf("FetchCSRSeasonCalendar parse: %w", err)
	}
	return &dto, body, nil
}
