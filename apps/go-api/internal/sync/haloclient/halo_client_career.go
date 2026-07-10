// Package sync - halo_client_career.go : endpoints career progression +
// spartan customization + helpers de parsing. Decoupe de halo_client.go
// (god-file split, refactor 2026-05-27).
package haloclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"

	"levelup/go-api/internal/games"
)

// defaultEconomyHostSync est l'host economy Halo par défaut (fallback legacy MT-01).
const defaultEconomyHostSync = "https://economy.svc.halowaypoint.com"

func (c *HaloAPIClient) GetCareerProgress(ctx context.Context, xuid string) (*CareerRankData, error) {
	if strings.TrimSpace(xuid) == "" {
		return nil, errors.New("GetCareerProgress: xuid vide")
	}
	progressURL := fmt.Sprintf(
		"%s/%s/careerranks/careerRank1?players=xuid(%s)",
		c.economyHost(ctx),
		c.gamePrefix(ctx),
		url.PathEscape(xuid),
	)
	progressBody, ok, err := c.doPlayerGatedGet(ctx, progressURL)
	if err != nil {
		return nil, fmt.Errorf("GetCareerProgress: %w", err)
	}
	if !ok {
		return nil, nil
	}
	data, err := parseCareerProgressPayload(progressBody, xuid)
	if err != nil {
		return nil, fmt.Errorf("GetCareerProgress decode: %w", err)
	}
	return data, nil
}

// GetSpartanCustomization récupère uniquement la customisation Spartan
// (ServiceTag, banner, emblem, backdrop) via l'API Economy player-gated.
// Retourne (nil, nil) si le token est absent/insuffisant (401/403) ou si la
// réponse est vide.
//
// 2026-05-30 — fallback `/customization?view=public` ajouté pour les joueurs
// TIERS (cas Explorer : adversaire non suivi). `/customization/appearance` est
// player-gated → 403 pour un xuid non propriétaire ; la vue publique expose le
// MÊME bloc Appearance pour n'importe quel joueur (vérifié empiriquement avec nos
// tokens : 403 vs 200). Ce n'était donc pas une dépréciation mais le gating
// par défaut de l'endpoint /appearance. Les deux réponses se décodent via le
// même parseCustomizationAppearance.
//
// 2026-05-14 — endpoint primaire `/customization/appearance` (le `view=public`
// timeoutait alors comme appel primaire ; on le réserve désormais au fallback
// tiers). Source : projet Grunt API (github.com/dend/grunt —
// EconomyModule.PlayerAppearanceCustomization). Réponse de la forme
// {Status, Appearance:{ServiceTag, BackdropImagePath, Emblem:{EmblemPath},
// PlayerTitlePath, ...}}.
//
// 2026-05-08 — pattern Grunt strict : pas d'invention d'URL en cas d'échec
// resolve (ex-fallbackCustomization* retiraient la résolution canonique au
// profit d'URLs /Waypoint/file/images/... qui n'existent pas sur Microsoft
// GameCMS → 403). Si le resolve échoue, on log warn et on laisse vide.
func (c *HaloAPIClient) GetSpartanCustomization(ctx context.Context, xuid string) (*SpartanCustomizationData, error) {
	if strings.TrimSpace(xuid) == "" {
		return nil, errors.New("GetSpartanCustomization: xuid vide")
	}
	customizationURL := fmt.Sprintf(
		"%s/%s/players/xuid(%s)/customization/appearance",
		c.economyHost(ctx),
		c.gamePrefix(ctx),
		url.PathEscape(xuid),
	)
	customizationBody, ok, err := c.doPlayerGatedGet(ctx, customizationURL)
	if err != nil {
		return nil, fmt.Errorf("GetSpartanCustomization: %w", err)
	}
	if !ok {
		// `/customization/appearance` est player-gated → 403 pour un joueur TIERS
		// (cas Explorer : adversaire non suivi). Fallback sur la vue publique
		// `/customization?view=public`, qui expose le MÊME bloc Appearance
		// (ServiceTag/Emblem/BackdropImagePath) pour n'importe quel joueur.
		// Décodée par le même parseCustomizationAppearance (navigation Appearance.*).
		publicURL := fmt.Sprintf(
			"%s/%s/players/xuid(%s)/customization?view=public",
			c.economyHost(ctx),
			c.gamePrefix(ctx),
			url.PathEscape(xuid),
		)
		pubBody, pubOK, pubErr := c.doPlayerGatedGet(ctx, publicURL)
		if pubErr != nil {
			return nil, fmt.Errorf("GetSpartanCustomization (public view): %w", pubErr)
		}
		if !pubOK {
			return nil, nil
		}
		slog.InfoContext(ctx, "spartan_id: customization via vue publique (joueur tiers)",
			"xuid", xuid, "bytes", len(pubBody))
		customizationBody = pubBody
	}
	appearance, err := parseCustomizationAppearance(customizationBody)
	if err != nil {
		return nil, fmt.Errorf("GetSpartanCustomization decode: %w", err)
	}
	if appearance == nil {
		return nil, nil
	}

	out := &SpartanCustomizationData{SpartanID: appearance.ServiceTag}
	if appearance.BannerImagePath != "" {
		if resolved, resolveErr := c.resolveCustomizationImageURL(ctx, appearance.BannerImagePath); resolveErr == nil {
			out.BannerImageURL = resolved
		} else {
			slog.WarnContext(ctx, "spartan_id: banner image resolve failed",
				"inventory_path", appearance.BannerImagePath, "err", resolveErr)
		}
	}
	// Fallback nameplate : Halo /customization/appearance ne retourne pas
	// toujours BannerImagePath. Port du flow Python `resolve_positive_emblem_cfg`
	// (ResolveNameplateURL dans spartan_nameplate_resolver.go) — fetch JSON
	// emblem CMS, parse AvailableConfigurations[], prend le 1er cfg > 0,
	// construit URL `/hi/Waypoint/file/images/nameplates/<stem>_<cfg>.png`.
	if out.BannerImageURL == "" && appearance.EmblemPath != "" {
		cfg, _ := strconv.ParseInt(strings.TrimSpace(appearance.EmblemConfigurationID), 10, 64)
		if url := ResolveNameplateURL(ctx, appearance.EmblemPath, cfg, c.spartanToken, c.clearanceToken); url != "" {
			out.BannerImageURL = url
		}
	}
	if appearance.EmblemPath != "" {
		if resolved, resolveErr := c.resolveCustomizationImageURL(ctx, appearance.EmblemPath); resolveErr == nil {
			out.EmblemImageURL = resolved
		} else {
			slog.WarnContext(ctx, "spartan_id: emblem image resolve failed",
				"inventory_path", appearance.EmblemPath, "err", resolveErr)
		}
	}
	if appearance.BackdropImagePath != "" {
		if resolved, resolveErr := c.resolveCustomizationImageURL(ctx, appearance.BackdropImagePath); resolveErr == nil {
			out.BackdropImageURL = resolved
		} else {
			slog.WarnContext(ctx, "spartan_id: backdrop image resolve failed",
				"inventory_path", appearance.BackdropImagePath, "err", resolveErr)
		}
	}
	return out, nil
}

// GetCareerRank récupère la progression du rang carrière combinée à la
// customisation Spartan (1 appel d'orchestration = 2 endpoints en série).
//
// Deprecated: appeler GetCareerProgress et GetSpartanCustomization séparément
// pour découpler les cadences de refresh (XP live throttled vs customization
// 6h TTL). Conservé pour compat avec PooledHaloClient et tests legacy.
func (c *HaloAPIClient) GetCareerRank(ctx context.Context, xuid string) (*CareerRankData, error) {
	data, err := c.GetCareerProgress(ctx, xuid)
	if err != nil || data == nil {
		return data, err
	}
	// Customization: échec silencieux (préserve la sémantique antérieure).
	if custom, cerr := c.GetSpartanCustomization(ctx, xuid); cerr == nil && custom != nil {
		if custom.SpartanID != "" {
			data.SpartanID = custom.SpartanID
		}
		data.BannerImageURL = custom.BannerImageURL
		data.EmblemImageURL = custom.EmblemImageURL
		data.BackdropImageURL = custom.BackdropImageURL
	}
	return data, nil
}

// economyHost résout l'host economy (career/customization). Précédence (MT-01) :
// override d'instance `economyBaseURL` (champ test, non vide) → resolver
// title-aware (ctx slug) → const Halo legacy. Byte-identique pour halo_infinite.
func (c *HaloAPIClient) economyHost(ctx context.Context) string {
	if s := strings.TrimSpace(c.economyBaseURL); s != "" {
		return strings.TrimRight(s, "/")
	}
	return c.hostFor(ctx, games.EndpointEconomy, defaultEconomyHostSync)
}

// gameCMSHost résout l'host gamecms (customization media). Même précédence.
func (c *HaloAPIClient) gameCMSHost(ctx context.Context) string {
	if s := strings.TrimSpace(c.gameCMSBaseURL); s != "" {
		return strings.TrimRight(s, "/")
	}
	return c.hostFor(ctx, games.EndpointGameCMS, haloGameCMSHost)
}

func (c *HaloAPIClient) doPlayerGatedGet(ctx context.Context, rawURL string) ([]byte, bool, error) {
	body, err := c.doGet(ctx, rawURL)
	if err != nil {
		if isAuthErr(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return body, true, nil
}

func isAuthErr(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return contains(s, "HTTP 401") || contains(s, "HTTP 403")
}

func parseCareerProgressPayload(body []byte, xuid string) (*CareerRankData, error) {
	type progress struct {
		Rank              int  `json:"Rank"`
		PartialProgress   int  `json:"PartialProgress"`
		HasReachedMaxRank bool `json:"HasReachedMaxRank"`
	}
	type alternateTrack struct {
		Result struct {
			CurrentProgress *progress `json:"CurrentProgress"`
		} `json:"Result"`
	}
	var payload struct {
		CurrentProgress *progress        `json:"CurrentProgress"`
		RewardTracks    []alternateTrack `json:"RewardTracks"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	current := payload.CurrentProgress
	if current == nil {
		for _, track := range payload.RewardTracks {
			if track.Result.CurrentProgress != nil {
				current = track.Result.CurrentProgress
				break
			}
		}
	}
	if current == nil {
		preview := body
		if len(preview) > 300 {
			preview = preview[:300]
		}
		slog.Warn("halo_client: parseCareerProgressPayload — CurrentProgress introuvable",
			"xuid", xuid, "body_preview", string(preview))
		return nil, nil
	}

	return &CareerRankData{
		XUID:        xuid,
		CurrentRank: current.Rank,
		CurrentXP:   current.PartialProgress,
		IsMaxRank:   current.HasReachedMaxRank,
	}, nil
}

type customizationAppearance struct {
	ServiceTag            string
	BannerImagePath       string
	BackdropImagePath     string
	EmblemPath            string
	EmblemConfigurationID string
}

// Clés JSON Halo API customization payload — utilisées par
// parseCustomizationAppearance / extractCustomizationMediaPath pour
// naviguer dans la réponse imbriquée (Appearance.Banner.ImagePath, etc.).
const (
	jsonKeyAppearance  = "Appearance"
	jsonKeyEmblem      = "Emblem"
	jsonKeyEmblemPath  = "EmblemPath"
	jsonKeyCommonData  = "CommonData"
	jsonKeyImagePath   = "ImagePath"
	jsonKeyPath        = "Path"
	jsonKeyDisplayPath = "DisplayPath"
	jsonKeyMedia       = "Media"
	jsonKeyMediaURL    = "MediaUrl"
)

func parseCustomizationAppearance(body []byte) (*customizationAppearance, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	return &customizationAppearance{
		ServiceTag: firstNonEmptyPayloadString(payload,
			[]string{jsonKeyAppearance, "ServiceTag"},
		),
		BannerImagePath: firstNonEmptyPayloadString(payload,
			[]string{jsonKeyAppearance, "BannerImagePath"},
			[]string{jsonKeyAppearance, "NameplateImagePath"},
			[]string{jsonKeyAppearance, "PlayerTitlePath"},
			[]string{jsonKeyAppearance, "Nameplate", "NameplateImagePath"},
			[]string{jsonKeyAppearance, "Nameplate", jsonKeyImagePath},
			[]string{jsonKeyAppearance, "Nameplate", jsonKeyPath},
			[]string{jsonKeyAppearance, "Banner", "BannerImagePath"},
			[]string{jsonKeyAppearance, "Banner", jsonKeyImagePath},
			[]string{jsonKeyAppearance, "Banner", jsonKeyPath},
		),
		BackdropImagePath: firstNonEmptyPayloadString(payload,
			[]string{jsonKeyAppearance, "BackdropImagePath"},
		),
		EmblemPath: firstNonEmptyPayloadString(payload,
			[]string{jsonKeyAppearance, jsonKeyEmblem, jsonKeyEmblemPath},
			[]string{jsonKeyAppearance, jsonKeyEmblemPath},
		),
		EmblemConfigurationID: stringifyCustomizationConfigurationID(firstNonEmptyPayloadValue(payload,
			[]string{jsonKeyAppearance, jsonKeyEmblem, "ConfigurationId"},
			[]string{jsonKeyAppearance, jsonKeyEmblem, "ConfigurationID"},
		)),
	}, nil
}

func firstNonEmptyPayloadString(payload map[string]any, keySets ...[]string) string {
	for _, keys := range keySets {
		if value := nestedPayloadString(payload, keys...); value != "" {
			return value
		}
	}
	return ""
}

func firstNonEmptyPayloadValue(payload map[string]any, keySets ...[]string) any {
	for _, keys := range keySets {
		if value := nestedPayloadValue(payload, keys...); value != nil {
			return value
		}
	}
	return nil
}

func stringifyCustomizationConfigurationID(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case json.Number:
		return v.String()
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func (c *HaloAPIClient) resolveCustomizationImageURL(ctx context.Context, inventoryPath string) (string, error) {
	trimmed := strings.TrimSpace(strings.TrimLeft(inventoryPath, "/"))
	if trimmed == "" {
		return "", fmt.Errorf("resolveCustomizationImageURL: inventory path vide")
	}

	endpoint := fmt.Sprintf("%s/%s/progression/file/%s", c.gameCMSHost(ctx), c.gamePrefix(ctx), trimmed)
	body, err := c.doGet(ctx, endpoint)
	if err != nil {
		return "", err
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}

	mediaPath := extractCustomizationMediaPath(payload)
	if mediaPath == "" {
		return "", fmt.Errorf("resolveCustomizationImageURL: media path absent")
	}
	return buildCustomizationImageURL(c.gameCMSHost(ctx), c.gamePrefix(ctx), mediaPath), nil
}

func extractCustomizationMediaPath(payload map[string]any) string {
	paths := [][]string{
		{jsonKeyCommonData, jsonKeyDisplayPath, jsonKeyMedia, jsonKeyMediaURL, jsonKeyPath},
		{jsonKeyDisplayPath, jsonKeyMedia, jsonKeyMediaURL, jsonKeyPath},
		{jsonKeyImagePath, jsonKeyMedia, jsonKeyMediaURL, jsonKeyPath},
		{jsonKeyCommonData, jsonKeyImagePath, jsonKeyMedia, jsonKeyMediaURL, jsonKeyPath},
	}
	for _, keys := range paths {
		if value := nestedPayloadString(payload, keys...); value != "" {
			return value
		}
	}
	return ""
}

func nestedPayloadString(payload map[string]any, keys ...string) string {
	value, ok := nestedPayloadValue(payload, keys...).(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func nestedPayloadValue(payload map[string]any, keys ...string) any {
	current := any(payload)
	for _, key := range keys {
		asMap, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		next, ok := asMap[key]
		if !ok {
			return nil
		}
		current = next
	}
	return current
}

func buildCustomizationImageURL(baseURL, gamePrefix, mediaPath string) string {
	trimmedBase := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	trimmedPath := strings.TrimSpace(mediaPath)
	if trimmedBase == "" || trimmedPath == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(trimmedPath), "http://") || strings.HasPrefix(strings.ToLower(trimmedPath), "https://") {
		return trimmedPath
	}
	trimmedPath = strings.TrimLeft(trimmedPath, "/")
	prefix := strings.ToLower(gamePrefix)
	if strings.HasPrefix(strings.ToLower(trimmedPath), prefix+"/images/file/") {
		return trimmedBase + "/" + trimmedPath
	}
	if strings.HasPrefix(strings.ToLower(trimmedPath), "images/file/") {
		return trimmedBase + "/" + gamePrefix + "/" + trimmedPath
	}
	return trimmedBase + "/" + gamePrefix + "/images/file/" + trimmedPath
}

// (2026-05-08) Les fonctions `fallbackCustomization{Emblem,Backdrop,Banner}URL`
// + `fallbackCustomizationBannerFromEmblem` + `customizationInventoryStem` ont
// été SUPPRIMÉES. Elles inventaient des URLs au format
// `/hi/Waypoint/file/images/{kind}/{stem}.png` qui n'existent pas côté
// Microsoft GameCMS — l'API retourne 403 systématiquement. Le seul pattern
// correct, aligné sur Grunt API
// (https://github.com/dend/grunt — endpoints `GameCms_GetProgressionFile` +
// `GameCms_GetProgressionImage`), est :
//
//  1. Fetch JSON descriptor : `GET /hi/progression/file/{InventoryPath}` →
//     champ `CommonData.DisplayPath.Media.MediaUrl.Path` (ou variantes).
//  2. Fetch image : `GET /hi/images/file/{MediaPath}`.
//
// Implémenté par `resolveCustomizationImageURL` ci-dessus (ligne ~733). Quand
// le resolve échoue (auth absente, JSON malformé, asset Microsoft retiré),
// on stocke chaîne vide → le frontend affiche un placeholder "image
// indisponible" au lieu d'une URL inventée qui finirait en 403/502.
