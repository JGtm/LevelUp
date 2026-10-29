package infinite

// operations.go — passe de combat (opérations, Economy) et défis (decks, stats) : GET et DTO,
// decks compris. Le corps brut est rendu avec le DTO : l'appelant le
// persiste tel quel. La résolution des définitions (pistes, objets, défis) passe par les actifs de
// LevelUp, pas par la brique.

import (
	"context"
	"encoding/json"
	"fmt"
)

// BattlePassTrackDTO : une opération de la réponse `rewardtracks/operations`.
type BattlePassTrackDTO struct {
	RewardTrackPath string `json:"RewardTrackPath"`
	CurrentProgress struct {
		Rank            int `json:"Rank"`
		PartialProgress int `json:"PartialProgress"`
	} `json:"CurrentProgress"`
}

// OperationsDTO : la réponse brute de `GET /{jeu}/players/xuid(X)/rewardtracks/operations`.
type OperationsDTO struct {
	ActiveOperationRewardTrackPath string               `json:"ActiveOperationRewardTrackPath"`
	OperationRewardTracks          []BattlePassTrackDTO `json:"OperationRewardTracks"`
}

// GetOperations récupère les opérations (passe de combat) d'un joueur ([CleEconomy]). Rend le DTO
// et le corps brut ; sur un corps illisible, le corps est rendu avec l'erreur.
func (c *Client) GetOperations(ctx context.Context, auth Auth, xuid string) (*OperationsDTO, []byte, error) {
	endpoint := fmt.Sprintf("%s/players/xuid(%s)/rewardtracks/operations", c.endpoints.base(CleEconomy), xuid)
	body, err := c.get(ctx, requete{url: endpoint, auth: auth})
	if err != nil {
		return nil, nil, err
	}
	var dto OperationsDTO
	if err := json.Unmarshal(body, &dto); err != nil {
		return nil, body, fmt.Errorf("battle_pass decode: %w", err)
	}
	return &dto, body, nil
}

// ChallengeDeckDTO : un deck de défis assigné.
type ChallengeDeckDTO struct {
	Expiration struct {
		ISO8601Date string `json:"ISO8601Date"`
	} `json:"Expiration"`
	ActiveChallenges    []ChallengeDeckItemDTO `json:"ActiveChallenges"`
	CompletedChallenges []json.RawMessage      `json:"CompletedChallenges"`
}

// ChallengeDeckItemDTO : un défi actif d'un deck.
type ChallengeDeckItemDTO struct {
	Path            string `json:"Path"`
	TrackingID      string `json:"TrackingId"`
	XPReward        *int   `json:"XPReward"`
	Threshold       *int   `json:"Threshold"`
	Progress        *int   `json:"Progress"`
	CurrentProgress *int   `json:"CurrentProgress"`
	CanReroll       *bool  `json:"CanReroll"`
	Expiration      struct {
		ISO8601Date string `json:"ISO8601Date"`
	} `json:"Expiration"`
}

// DecksDTO : la réponse brute de `GET /{jeu}/players/xuid(X)/decks`.
type DecksDTO struct {
	AssignedDecks []ChallengeDeckDTO `json:"AssignedDecks"`
}

// GetDecks récupère les decks de défis d'un joueur ([CleChallenges]). Rend le DTO et le corps
// brut ; sur un corps illisible, le corps est rendu avec l'erreur.
func (c *Client) GetDecks(ctx context.Context, auth Auth, xuid string) (*DecksDTO, []byte, error) {
	endpoint := fmt.Sprintf("%s/players/xuid(%s)/decks", c.endpoints.base(CleChallenges), xuid)
	body, err := c.get(ctx, requete{url: endpoint, auth: auth})
	if err != nil {
		return nil, nil, err
	}
	var dto DecksDTO
	if err := json.Unmarshal(body, &dto); err != nil {
		return nil, body, fmt.Errorf("challenges decode: %w", err)
	}
	return &dto, body, nil
}
