package infinite

// appearance.go — apparence Spartan (Economy) et résolution d'une image de personnalisation
// (GameCMS) : fetch et parse de l'apparence, résolution de l'adresse d'image. Le repli « vue
// publique » d'un joueur tiers (403 sur `/customization/appearance`) est une politique : l'appelant enchaîne
// [Client.GetCustomizationAppearance] puis [Client.GetCustomizationPublic] sur [IsAuthError].

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// CustomizationAppearanceDTO : les champs du bloc `Appearance` d'une réponse de personnalisation,
// lus par leurs chemins JSON connus (le bloc a plusieurs formes selon l'objet équipé).
type CustomizationAppearanceDTO struct {
	ServiceTag            string
	BannerImagePath       string
	BackdropImagePath     string
	EmblemPath            string
	EmblemConfigurationID string
}

// GetCustomizationAppearance récupère l'apparence d'un joueur par
// `/players/xuid(X)/customization/appearance` ([CleEconomy], à portée joueur : 403 pour un tiers).
func (c *Client) GetCustomizationAppearance(ctx context.Context, auth Auth, xuid string) (*CustomizationAppearanceDTO, error) {
	return c.customization(ctx, auth, xuid, "/customization/appearance")
}

// GetCustomizationPublic récupère l'apparence d'un joueur par la vue publique
// `/players/xuid(X)/customization?view=public` (même bloc `Appearance`, pour tout joueur).
func (c *Client) GetCustomizationPublic(ctx context.Context, auth Auth, xuid string) (*CustomizationAppearanceDTO, error) {
	return c.customization(ctx, auth, xuid, "/customization?view=public")
}

func (c *Client) customization(ctx context.Context, auth Auth, xuid, suffixe string) (*CustomizationAppearanceDTO, error) {
	if strings.TrimSpace(xuid) == "" {
		return nil, errors.New("fetchCustomizationAppearance: xuid vide")
	}
	endpoint := fmt.Sprintf("%s/players/xuid(%s)%s", c.endpoints.base(CleEconomy), url.PathEscape(xuid), suffixe)
	body, err := c.get(ctx, requete{url: endpoint, auth: auth})
	if err != nil {
		return nil, fmt.Errorf("fetchCustomizationAppearance: %w", err)
	}
	dto, err := DecodeCustomizationAppearance(body)
	if err != nil {
		return nil, fmt.Errorf("fetchCustomizationAppearance decode: %w", err)
	}
	return dto, nil
}

// Clés JSON du bloc de personnalisation.
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

// DecodeCustomizationAppearance lit le bloc `Appearance` d'un corps de personnalisation.
func DecodeCustomizationAppearance(body []byte) (*CustomizationAppearanceDTO, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return &CustomizationAppearanceDTO{
		ServiceTag: firstNonEmptyPayloadString(payload, []string{jsonKeyAppearance, "ServiceTag"}),
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
		BackdropImagePath: firstNonEmptyPayloadString(payload, []string{jsonKeyAppearance, "BackdropImagePath"}),
		EmblemPath: firstNonEmptyPayloadString(payload,
			[]string{jsonKeyAppearance, jsonKeyEmblem, jsonKeyEmblemPath},
			[]string{jsonKeyAppearance, jsonKeyEmblemPath},
		),
		EmblemConfigurationID: stringifyConfigurationID(firstNonEmptyPayloadValue(payload,
			[]string{jsonKeyAppearance, jsonKeyEmblem, "ConfigurationId"},
			[]string{jsonKeyAppearance, jsonKeyEmblem, "ConfigurationID"},
		)),
	}, nil
}

// AppearanceInputs : les champs bruts de l'apparence qu'un diagnostic par composant lit (chemins
// d'inventaire et identifiant de configuration, pas d'URL résolue).
type AppearanceInputs struct {
	ServiceTag        string
	BannerImagePath   string
	BackdropImagePath string
	EmblemPath        string
	// EmblemConfigID : ConfigurationId de l'emblème (0 si absent ou illisible).
	EmblemConfigID int64
}

// AppearanceInputsDe rend les entrées de diagnostic d'une apparence décodée ; nil rend nil.
func AppearanceInputsDe(a *CustomizationAppearanceDTO) *AppearanceInputs {
	if a == nil {
		return nil
	}
	cfg, _ := strconv.ParseInt(strings.TrimSpace(a.EmblemConfigurationID), 10, 64) // illisible = 0, comme absent
	return &AppearanceInputs{
		ServiceTag:        a.ServiceTag,
		BannerImagePath:   a.BannerImagePath,
		BackdropImagePath: a.BackdropImagePath,
		EmblemPath:        a.EmblemPath,
		EmblemConfigID:    cfg,
	}
}

// ResolveCustomizationImageURL résout le chemin d'inventaire d'une image de personnalisation en URL
// d'image GameCMS ([CleGameCMS]) : `GET /{jeu}/progression/file/{chemin}` puis le chemin média du
// descripteur, préfixé de `/{jeu}/images/file/`.
func (c *Client) ResolveCustomizationImageURL(ctx context.Context, auth Auth, inventoryPath string) (string, error) {
	trimmed := strings.TrimSpace(strings.TrimLeft(inventoryPath, "/"))
	if trimmed == "" {
		return "", fmt.Errorf("resolveCustomizationImageURL: inventory path vide")
	}
	body, err := c.get(ctx, requete{url: c.endpoints.base(CleGameCMS) + "/progression/file/" + trimmed, auth: auth})
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
	return buildCustomizationImageURL(c.endpoints.Hote(CleGameCMS), c.endpoints.PrefixeJeu(), mediaPath), nil
}

func extractCustomizationMediaPath(payload map[string]any) string {
	return firstNonEmptyPayloadString(payload,
		[]string{jsonKeyCommonData, jsonKeyDisplayPath, jsonKeyMedia, jsonKeyMediaURL, jsonKeyPath},
		[]string{jsonKeyDisplayPath, jsonKeyMedia, jsonKeyMediaURL, jsonKeyPath},
		[]string{jsonKeyImagePath, jsonKeyMedia, jsonKeyMediaURL, jsonKeyPath},
		[]string{jsonKeyCommonData, jsonKeyImagePath, jsonKeyMedia, jsonKeyMediaURL, jsonKeyPath},
	)
}

func buildCustomizationImageURL(baseURL, gamePrefix, mediaPath string) string {
	trimmedBase := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	trimmedPath := strings.TrimSpace(mediaPath)
	if trimmedBase == "" || trimmedPath == "" {
		return ""
	}
	lower := strings.ToLower(trimmedPath)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return trimmedPath
	}
	trimmedPath = strings.TrimLeft(trimmedPath, "/")
	lower = strings.ToLower(trimmedPath)
	prefix := strings.ToLower(gamePrefix)
	if strings.HasPrefix(lower, prefix+"/images/file/") {
		return trimmedBase + "/" + trimmedPath
	}
	if strings.HasPrefix(lower, "images/file/") {
		return trimmedBase + "/" + gamePrefix + "/" + trimmedPath
	}
	return trimmedBase + "/" + gamePrefix + "/images/file/" + trimmedPath
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

func stringifyConfigurationID(value any) string {
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
