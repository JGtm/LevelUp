// Package sync — xbox_client.go : client HTTP pour l'API Xbox Achievements v2.
//
// Endpoint : GET https://achievements.xboxlive.com/users/xuid({xuid})/achievements
// Header obligatoire : x-xbl-contract-version: 2 (sinon retourne le schéma Xbox 360 v1).
// Pagination via pagingInfo.continuationToken / skipItems.
package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/netguard"
)

const xboxAchievementsBaseURL = "https://achievements.xboxlive.com"

// XboxAchievementsClient abstrait les appels à l'API Xbox Achievements.
// L'interface permet l'injection de mock dans les tests.
type XboxAchievementsClient interface {
	GetPlayerAchievements(ctx context.Context, xuid, lang string) ([]PlayerAchievementRaw, error)
}

// PlayerAchievementRaw contient les données brutes d'un achievement retourné par l'API Xbox.
type PlayerAchievementRaw struct {
	ID             string
	Name           string
	Description    string
	LockedDesc     string
	Gamerscore     int
	ImageURL       string
	IsSecret       bool
	RarityCategory string
	RarityPercent  float64
	Unlocked       bool
	UnlockedAt     time.Time
	// Progression (base game achievements)
	CurrentProgress int
	TargetProgress  int
	// XboxTitleID est le premier TitleAssociation.ID renvoyé par l'API (numérique → string).
	XboxTitleID string
	// ServiceConfigID identifie le jeu de manière unique (SCID Xbox — plus fiable que TitleID).
	ServiceConfigID string
}

// xboxHTTPClient implémente XboxAchievementsClient via l'API Xbox Live.
type xboxHTTPClient struct {
	authHeader  string
	httpClient  *http.Client
	xboxTitleID string // ex: "1144039928" pour Halo Infinite
}

// NewXboxHTTPClient crée un xboxHTTPClient pour un titre Xbox donné.
// xboxTitleID est l'identifiant numérique Xbox du titre (ex: "1144039928" pour Halo Infinite) —
// utiliser titlePkg.XboxTitleIDFor(slug) pour le résoudre depuis un slug LevelUp.
func NewXboxHTTPClient(xstsResult *auth.XSTSResult, xboxTitleID string) XboxAchievementsClient {
	return &xboxHTTPClient{
		authHeader:  xstsResult.AuthHeader(),
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		xboxTitleID: xboxTitleID,
	}
}

// GetPlayerAchievements récupère tous les achievements Halo Infinite pour un joueur.
// lang doit être un tag BCP-47 (ex: "en-US", "fr-FR").
// La pagination est gérée automatiquement via pagingInfo.continuationToken.
func (c *xboxHTTPClient) GetPlayerAchievements(ctx context.Context, xuid, lang string) ([]PlayerAchievementRaw, error) {
	var all []PlayerAchievementRaw
	skipItems := 0

	slog.DebugContext(ctx, "xbox_client: début récupération achievements", "xuid", xuid, "lang", lang)
	for {
		batch, continuationToken, err := c.fetchPage(ctx, xuid, lang, skipItems)
		if err != nil {
			slog.WarnContext(ctx, "xbox_client: erreur fetchPage",
				"xuid", xuid, "lang", lang, "skip", skipItems, "err", err)
			return nil, err
		}
		all = append(all, batch...)
		slog.DebugContext(ctx, "xbox_client: page achievements récupérée",
			"xuid", xuid, "lang", lang,
			"batch", len(batch), "total", len(all),
			"continuation", continuationToken != "",
		)
		if continuationToken == "" {
			break
		}
		skipItems += len(batch)
	}

	return all, nil
}

// --- structures de désérialisation de la réponse Xbox Achievements v2 ---

type xboxAchievementsResponse struct {
	Achievements []xboxAchievementItem `json:"achievements"`
	PagingInfo   struct {
		TotalRecords      int    `json:"totalRecords"`
		ContinuationToken string `json:"continuationToken"`
	} `json:"pagingInfo"`
}

type xboxAchievementItem struct {
	ID                string `json:"id"`
	ServiceConfigID   string `json:"serviceConfigId"`
	Name              string `json:"name"`
	TitleAssociations []struct {
		Name string `json:"name"`
		ID   int    `json:"id"`
	} `json:"titleAssociations"`
	ProgressState string `json:"progressState"` // "Achieved", "InProgress", "NotStarted"
	Progression   struct {
		Requirements []struct {
			ID                    string `json:"id"`
			Current               string `json:"current"`
			Target                string `json:"target"`
			OperationType         string `json:"operationType"`
			ValueType             string `json:"valueType"`
			RuleParticipationType string `json:"ruleParticipationType"`
		} `json:"requirements"`
		TimeUnlocked string `json:"timeUnlocked"`
	} `json:"progression"`
	MediaAssets []struct {
		Name string `json:"name"`
		Type string `json:"type"`
		URL  string `json:"url"`
	} `json:"mediaAssets"`
	Platforms         []string `json:"platforms"`
	IsSecret          bool     `json:"isSecret"`
	Description       string   `json:"description"`
	LockedDescription string   `json:"lockedDescription"`
	ProductID         string   `json:"productId"`
	ActivityID        string   `json:"activityId"`
	Rewards           []struct {
		Name        interface{} `json:"name"`
		Description interface{} `json:"description"`
		Value       string      `json:"value"`
		Type        string      `json:"type"`
		MediaAsset  interface{} `json:"mediaAsset"`
	} `json:"rewards"`
	EstimatedTime string `json:"estimatedTime"`
	Deeplink      string `json:"deeplink"`
	IsRevoked     bool   `json:"isRevoked"`
	Rarity        struct {
		CurrentCategory   string  `json:"currentCategory"`
		CurrentPercentage float64 `json:"currentPercentage"`
	} `json:"rarity"`
}

// fetchPage récupère une page d'achievements (avec skipItems pour la pagination).
func (c *xboxHTTPClient) fetchPage(ctx context.Context, xuid, lang string, skipItems int) ([]PlayerAchievementRaw, string, error) {
	u, err := url.Parse(fmt.Sprintf("%s/users/xuid(%s)/achievements", xboxAchievementsBaseURL, xuid))
	if err != nil {
		return nil, "", fmt.Errorf("xbox_client: parse URL: %w", err)
	}
	q := u.Query()
	if c.xboxTitleID != "" {
		q.Set("titleId", c.xboxTitleID)
	}
	if skipItems > 0 {
		q.Set("skipItems", strconv.Itoa(skipItems))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, "", fmt.Errorf("xbox_client: new request: %w", err)
	}
	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("x-xbl-contract-version", "2")
	req.Header.Set("Accept-Language", lang)
	req.Header.Set("Accept", "application/json")

	// Mode démo : aucune sortie tierce (cf. internal/platform/netguard).
	if gErr := netguard.Check(ctx, "xbox_achievements.get"); gErr != nil {
		return nil, "", gErr
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("xbox_client: HTTP do: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("xbox_client: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("xbox_client: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var apiResp xboxAchievementsResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, "", fmt.Errorf("xbox_client: JSON decode: %w", err)
	}

	items := make([]PlayerAchievementRaw, 0, len(apiResp.Achievements))
	for _, a := range apiResp.Achievements {
		raw := parseAchievementItem(a)
		items = append(items, raw)
	}

	return items, apiResp.PagingInfo.ContinuationToken, nil
}

// parseAchievementItem convertit un item API en PlayerAchievementRaw.
func parseAchievementItem(a xboxAchievementItem) PlayerAchievementRaw {
	raw := PlayerAchievementRaw{
		ID:              a.ID,
		Name:            a.Name,
		Description:     a.Description,
		LockedDesc:      a.LockedDescription,
		IsSecret:        a.IsSecret,
		RarityCategory:  a.Rarity.CurrentCategory,
		RarityPercent:   a.Rarity.CurrentPercentage,
		Unlocked:        a.ProgressState == "Achieved",
		ServiceConfigID: a.ServiceConfigID,
	}

	// Gamerscore depuis Rewards
	for _, r := range a.Rewards {
		if r.Type == "Gamerscore" {
			if v, err := strconv.Atoi(r.Value); err == nil {
				raw.Gamerscore = v
			}
			break
		}
	}

	// Image principale
	for _, m := range a.MediaAssets {
		if m.Type == "Icon" {
			raw.ImageURL = m.URL
			break
		}
	}

	// Xbox title ID depuis le premier TitleAssociation
	if len(a.TitleAssociations) > 0 {
		raw.XboxTitleID = strconv.Itoa(a.TitleAssociations[0].ID)
	}

	// Date de déverrouillage
	if a.Progression.TimeUnlocked != "" && a.Progression.TimeUnlocked != "0001-01-01T00:00:00Z" {
		if t, err := time.Parse(time.RFC3339, a.Progression.TimeUnlocked); err == nil {
			raw.UnlockedAt = t
		}
	}

	// Progression (premier requirement disponible)
	if len(a.Progression.Requirements) > 0 {
		req := a.Progression.Requirements[0]
		if v, err := strconv.Atoi(req.Current); err == nil {
			raw.CurrentProgress = v
		}
		if v, err := strconv.Atoi(req.Target); err == nil {
			raw.TargetProgress = v
		}
	}

	return raw
}
