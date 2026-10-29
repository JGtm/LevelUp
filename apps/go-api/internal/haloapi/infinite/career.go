package infinite

// career.go — progression du rang carrière (Economy) ; un 401/403 est rendu typé ([IsAuthError]) :
// la lecture « jeton sans portée joueur = absent » est une politique de l'appelant.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// CareerProgressDTO : `CurrentProgress` d'une piste de récompense.
type CareerProgressDTO struct {
	Rank              int  `json:"Rank"`
	PartialProgress   int  `json:"PartialProgress"`
	HasReachedMaxRank bool `json:"HasReachedMaxRank"`
}

// CareerRankDTO : la réponse brute de `GET /{jeu}/careerranks/careerRank1?players=xuid(X)`. Deux
// formes : `CurrentProgress` au premier niveau, ou `RewardTracks[].Result.CurrentProgress`.
type CareerRankDTO struct {
	CurrentProgress *CareerProgressDTO `json:"CurrentProgress"`
	RewardTracks    []struct {
		Result struct {
			CurrentProgress *CareerProgressDTO `json:"CurrentProgress"`
		} `json:"Result"`
	} `json:"RewardTracks"`
}

// Progress rend la progression portée par la réponse, quelle que soit sa forme (premier niveau,
// sinon première piste qui en porte une) ; nil si aucune.
func (d *CareerRankDTO) Progress() *CareerProgressDTO {
	if d.CurrentProgress != nil {
		return d.CurrentProgress
	}
	for _, track := range d.RewardTracks {
		if track.Result.CurrentProgress != nil {
			return track.Result.CurrentProgress
		}
	}
	return nil
}

// GetCareerProgress récupère la progression du rang carrière d'un joueur ([CleEconomy], service
// à portée joueur).
func (c *Client) GetCareerProgress(ctx context.Context, auth Auth, xuid string) (*CareerRankDTO, error) {
	if strings.TrimSpace(xuid) == "" {
		return nil, errors.New("GetCareerProgress: xuid vide")
	}
	endpoint := fmt.Sprintf("%s/careerranks/careerRank1?players=xuid(%s)", c.endpoints.base(CleEconomy), url.PathEscape(xuid))
	body, err := c.get(ctx, requete{url: endpoint, auth: auth})
	if err != nil {
		return nil, fmt.Errorf("GetCareerProgress: %w", err)
	}
	var dto CareerRankDTO
	if err := json.Unmarshal(body, &dto); err != nil {
		return nil, fmt.Errorf("GetCareerProgress decode: %w", err)
	}
	return &dto, nil
}
