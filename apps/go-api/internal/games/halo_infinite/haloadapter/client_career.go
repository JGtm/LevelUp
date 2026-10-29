package haloadapter

// client_career.go — rang carrière et apparence Spartan (Economy, à portée joueur) et résolution
// des images de personnalisation (GameCMS, pattern Grunt strict : descripteur JSON
// `/{jeu}/progression/file/{chemin}` puis image `/{jeu}/images/file/{media}` ; aucune URL
// inventée : une résolution en échec laisse le champ vide, l'interface affiche une image
// indisponible).
//
// Politique de portée joueur : un 401/403 de ces services n'est PAS une erreur — le jeton courant
// n'a pas la portée du joueur demandé —, l'appelant dégrade sur (nil, nil).

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/haloapi/infinite"
)

// SpartanCustomizationData : l'identité visuelle Spartan (tag de service, images bannière,
// emblème, fond) résolue ; le type canonique vit dans `domain`.
type SpartanCustomizationData = domain.SpartanCustomizationData

// GetCareerProgress récupère la progression du rang carrière (rang, XP courante, rang maximal) par
// `/{jeu}/careerranks/careerRank1?players=xuid(X)`. (nil, nil) si le jeton n'a pas la portée
// (401/403) ou si la réponse ne porte pas de progression.
func (c *Client) GetCareerProgress(ctx context.Context, xuid string) (*domain.CareerRankSnapshot, error) {
	dto, err := c.api(ctx, surfaceAPIGet, infinite.CleEconomy).GetCareerProgress(ctx, c.auth(), xuid)
	if err != nil {
		if strings.TrimSpace(xuid) != "" && IsAuthError(err) {
			return nil, nil
		}
		return nil, err
	}
	current := dto.Progress()
	if current == nil {
		slog.WarnContext(ctx, "halo_client: parseCareerProgressPayload — CurrentProgress introuvable", "xuid", xuid)
		return nil, nil
	}
	return &domain.CareerRankSnapshot{
		XUID:        xuid,
		CurrentRank: current.Rank,
		CurrentXP:   current.PartialProgress,
		IsMaxRank:   current.HasReachedMaxRank,
	}, nil
}

// GetSpartanCustomization récupère la personnalisation Spartan (tag de service, bannière, emblème,
// fond) et résout ses images. (nil, nil) si le jeton n'a pas la portée ou si la réponse est vide.
// La bannière manquante est dérivée de l'emblème ([ResolveNameplateURL]).
func (c *Client) GetSpartanCustomization(ctx context.Context, xuid string) (*SpartanCustomizationData, error) {
	appearance, err := c.fetchCustomizationAppearance(ctx, xuid)
	if err != nil {
		return nil, err
	}
	if appearance == nil {
		return nil, nil
	}
	out := &SpartanCustomizationData{SpartanID: appearance.ServiceTag}
	out.BannerImageURL = c.imageResolue(ctx, appearance.BannerImagePath, "spartan_id: banner image resolve failed")
	if out.BannerImageURL == "" && appearance.EmblemPath != "" {
		if in := infinite.AppearanceInputsDe(appearance); in != nil {
			if u := ResolveNameplateURL(ctx, appearance.EmblemPath, in.EmblemConfigID, c.spartanToken, c.clearanceToken); u != "" {
				out.BannerImageURL = u
			}
		}
	}
	out.EmblemImageURL = c.imageResolue(ctx, appearance.EmblemPath, "spartan_id: emblem image resolve failed")
	out.BackdropImageURL = c.imageResolue(ctx, appearance.BackdropImagePath, "spartan_id: backdrop image resolve failed")
	return out, nil
}

// imageResolue résout le chemin d'inventaire `path` en URL d'image ; vide si `path` l'est, ou si
// la résolution échoue (journalisé sous `message`).
func (c *Client) imageResolue(ctx context.Context, path, message string) string {
	if path == "" {
		return ""
	}
	resolved, err := c.resolveCustomizationImageURL(ctx, path)
	if err != nil {
		slog.WarnContext(ctx, message, "inventory_path", path, "err", err)
		return ""
	}
	return resolved
}

// fetchCustomizationAppearance lit le bloc Appearance de `/customization/appearance`, à portée
// joueur ; pour un joueur TIERS (403), la vue publique `/customization?view=public` expose le
// même bloc. (nil, nil) si aucune des deux n'est accessible.
func (c *Client) fetchCustomizationAppearance(ctx context.Context, xuid string) (*infinite.CustomizationAppearanceDTO, error) {
	api := c.api(ctx, surfaceAPIGet, infinite.CleEconomy)
	appearance, err := api.GetCustomizationAppearance(ctx, c.auth(), xuid)
	if err == nil {
		return appearance, nil
	}
	if strings.TrimSpace(xuid) == "" || !IsAuthError(err) {
		return nil, err
	}
	pub, pubErr := api.GetCustomizationPublic(ctx, c.auth(), xuid)
	if pubErr != nil {
		if IsAuthError(pubErr) {
			return nil, nil
		}
		return nil, fmt.Errorf("fetchCustomizationAppearance (public view): %w", pubErr)
	}
	slog.InfoContext(ctx, "spartan_id: customization via vue publique (joueur tiers)", "xuid", xuid)
	return pub, nil
}

// GetCareerRank récupère la progression du rang carrière combinée à la personnalisation Spartan
// (deux services en série) ; l'échec de la personnalisation est silencieux.
//
// Deprecated: appeler GetCareerProgress et GetSpartanCustomization séparément pour découpler les
// cadences de rafraîchissement. Conservé pour le client du pool et les tests.
func (c *Client) GetCareerRank(ctx context.Context, xuid string) (*domain.CareerRankSnapshot, error) {
	data, err := c.GetCareerProgress(ctx, xuid)
	if err != nil || data == nil {
		return data, err
	}
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

// resolveCustomizationImageURL résout un chemin d'inventaire en URL d'image GameCMS.
func (c *Client) resolveCustomizationImageURL(ctx context.Context, inventoryPath string) (string, error) {
	return c.api(ctx, surfaceAPIGet, infinite.CleGameCMS).ResolveCustomizationImageURL(ctx, c.auth(), inventoryPath)
}

// AppearanceInputs : les champs bruts de l'apparence que le diagnostic par composant lit (chemins
// d'inventaire, identifiant de configuration de l'emblème), pas des URL résolues.
type AppearanceInputs = infinite.AppearanceInputs

// FetchAppearanceInputs lit l'apparence et rend ses champs bruts, SANS résoudre aucune image (les
// primitives Diagnose* s'en chargent, avec leur verdict). (nil, nil) si les jetons n'ont pas la
// portée ou si la réponse est vide : le service en dérive un verdict dégradé, jamais un 500.
func (c *Client) FetchAppearanceInputs(ctx context.Context, xuid string) (*AppearanceInputs, error) {
	appearance, err := c.fetchCustomizationAppearance(ctx, xuid)
	if err != nil || appearance == nil {
		return nil, err
	}
	return infinite.AppearanceInputsDe(appearance), nil
}
