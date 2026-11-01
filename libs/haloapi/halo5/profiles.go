package halo5

// profiles.go — les services de l'hôte profils : apparence et rendus PNG (Spartan, emblème).

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// GetAppearance recupere l'appearance identitaire d'un joueur (service tag + emblem
// IDs/couleurs + customization armure). Host PROFILS (haloplayer). GAMERTAG BRUT.
// Path confirme sonde : /h5/profiles/{gt}/appearance → 200 JSON. On ne projette que
// les champs identitaires (cf. H5Appearance).
func (c *Client) GetAppearance(ctx context.Context, auth Auth, gamertag string) (*H5Appearance, error) {
	if strings.TrimSpace(gamertag) == "" {
		return nil, errors.New("GetAppearance: gamertag vide")
	}
	endpoint := fmt.Sprintf("%s/h5/profiles/%s/appearance", c.endpoints.hoteProfils(), url.PathEscape(gamertag))
	body, err := c.getJSON(ctx, auth, endpoint)
	if err != nil {
		return nil, fmt.Errorf("GetAppearance(%s): %w", gamertag, err)
	}
	return decoder[H5Appearance]("GetAppearance", body)
}

// GetSpartanRenderPNG recupere le RENDU PNG du Spartan (corps/armure) d'un joueur.
// L'endpoint /h5/profiles/{gt}/spartan renvoie un 302 vers image.halocdn.com (URL
// signee, non reproductible cote client) ; le Doer binaire suit la redirection et
// telecharge les octets PNG. Renvoie (bytes, contentType, err). Host PROFILS (haloplayer).
func (c *Client) GetSpartanRenderPNG(ctx context.Context, auth Auth, gamertag string) ([]byte, string, error) {
	if strings.TrimSpace(gamertag) == "" {
		return nil, "", errors.New("GetSpartanRenderPNG: gamertag vide")
	}
	endpoint := fmt.Sprintf("%s/h5/profiles/%s/spartan", c.endpoints.hoteProfils(), url.PathEscape(gamertag))
	b, ct, err := c.getBinaire(ctx, auth, endpoint)
	if err != nil {
		return nil, "", fmt.Errorf("GetSpartanRenderPNG(%s): %w", gamertag, err)
	}
	return b, ct, nil
}

// GetEmblemPNG recupere le rendu PNG de l'emblème d'un joueur. L'endpoint
// /h5/profiles/{gt}/emblem renvoie un 302 vers image.halocdn.com (path
// emblems/{EmblemId}_{ColorPrimary}_{ColorSecondary}_{ColorTertiary}, hash signe) ;
// le Doer binaire suit la redirection et telecharge le PNG. Renvoie (bytes,
// contentType, err). Host PROFILS (haloplayer).
func (c *Client) GetEmblemPNG(ctx context.Context, auth Auth, gamertag string) ([]byte, string, error) {
	if strings.TrimSpace(gamertag) == "" {
		return nil, "", errors.New("GetEmblemPNG: gamertag vide")
	}
	endpoint := fmt.Sprintf("%s/h5/profiles/%s/emblem", c.endpoints.hoteProfils(), url.PathEscape(gamertag))
	b, ct, err := c.getBinaire(ctx, auth, endpoint)
	if err != nil {
		return nil, "", fmt.Errorf("GetEmblemPNG(%s): %w", gamertag, err)
	}
	return b, ct, nil
}
