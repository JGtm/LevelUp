package infinite

// privacy.go — confidentialité des matchs d'un compte (GET et DTO) ; le cache (succès 30 min, échec 5 min) et la lecture du verdict sont à l'appelant.

import (
	"context"
	"encoding/json"
	"fmt"
)

// PrivacyDTO : la réponse brute de `GET /{jeu}/players/xuid(X)/matches-privacy`.
type PrivacyDTO struct {
	AllMatchesPrivacy    string `json:"AllMatchesPrivacy"`
	PublicMatchesPrivacy string `json:"PublicMatchesPrivacy"`
	RankedMatchesPrivacy string `json:"RankedMatchesPrivacy"`
	CustomMatchesPrivacy string `json:"CustomMatchesPrivacy"`
}

// GetMatchPrivacy récupère la confidentialité des matchs d'un joueur ([CleStatsSansPort]).
func (c *Client) GetMatchPrivacy(ctx context.Context, auth Auth, xuid string) (*PrivacyDTO, error) {
	endpoint := fmt.Sprintf("%s/players/xuid(%s)/matches-privacy", c.endpoints.base(CleStatsSansPort), xuid)
	body, err := c.get(ctx, requete{url: endpoint, auth: auth})
	if err != nil {
		return nil, err
	}
	var dto PrivacyDTO
	if err := json.Unmarshal(body, &dto); err != nil {
		return nil, fmt.Errorf("privacy parse: %w", err)
	}
	return &dto, nil
}
