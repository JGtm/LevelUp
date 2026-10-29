package infinite

// skill.go — skill d'un match (MMR, attendus, RankRecap), CSR d'un joueur, CSR d'une playlist.
// Copié de `sync/haloclient/halo_skill.go` et `halo_skill_csr.go` (GET et DTO) ; la transformation
// en valeurs de domaine (MMR connu ou non, MMR adverse moyen, choix de l'entrée d'un joueur) reste
// chez l'appelant. Une absence (404/410) est rendue typée ([IsNotFound]) : l'appelant en fait un
// résultat vide.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// FlexFloat accepte un nombre JSON OU une chaîne contenant un nombre : le service skill rend
// parfois Expected/StdDev en chaîne (notation scientifique des très petits nombres). null ou ""
// laissent la valeur à zéro.
type FlexFloat float64

// UnmarshalJSON décode un nombre ou une chaîne numérique.
func (f *FlexFloat) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	var n float64
	if err := json.Unmarshal(b, &n); err == nil {
		*f = FlexFloat(n)
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s == "" {
		return nil
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("flexFloat: %q: %w", s, err)
	}
	*f = FlexFloat(n)
	return nil
}

// CSRRankDTO : un instantané de classement tel que le service le rend.
type CSRRankDTO struct {
	Value                       float64 `json:"Value"`
	MeasurementMatchesRemaining int     `json:"MeasurementMatchesRemaining"`
	Tier                        string  `json:"Tier"`
	SubTier                     int     `json:"SubTier"`
	InitialMeasurementMatches   int     `json:"InitialMeasurementMatches"`
}

// MatchSkillDTO : la réponse brute de `GET /{jeu}/matches/{id}/skill?players=xuid(X)...`.
type MatchSkillDTO struct {
	Value []struct {
		ID         string `json:"Id"`
		ResultCode int    `json:"ResultCode"`
		Result     struct {
			TeamMMR          FlexFloat            `json:"TeamMmr"`
			TeamID           int                  `json:"TeamId"`
			TeamMMRs         map[string]FlexFloat `json:"TeamMmrs"`
			StatPerformances *struct {
				Kills *struct {
					Count    int       `json:"Count"`
					Expected FlexFloat `json:"Expected"`
					StdDev   FlexFloat `json:"StdDev"`
				} `json:"Kills"`
				Deaths *struct {
					Count    int       `json:"Count"`
					Expected FlexFloat `json:"Expected"`
					StdDev   FlexFloat `json:"StdDev"`
				} `json:"Deaths"`
			} `json:"StatPerformances"`
			RankRecap *struct {
				PreMatchCsr  *CSRRankDTO `json:"PreMatchCsr"`
				PostMatchCsr *CSRRankDTO `json:"PostMatchCsr"`
			} `json:"RankRecap"`
		} `json:"Result"`
	} `json:"Value"`
}

// GetMatchSkill récupère le skill d'un match pour les xuid humains de `xuids` ([CleSkill]). Les
// bots (`bid(...)`) et les vides sont retirés AVANT l'appel (le service ne répond rien pour eux) ;
// sans xuid humain, aucun appel : DTO vide.
func (c *Client) GetMatchSkill(ctx context.Context, auth Auth, matchID string, xuids []string) (*MatchSkillDTO, error) {
	if !rexUUID.MatchString(matchID) {
		return nil, fmt.Errorf("GetMatchSkill: matchID invalide %q", matchID)
	}
	humains := FilterHumanXUIDs(xuids)
	if len(humains) == 0 {
		return &MatchSkillDTO{}, nil
	}
	params := url.Values{}
	for _, x := range humains {
		params.Add("players", "xuid("+x+")")
	}
	endpoint := fmt.Sprintf("%s/matches/%s/skill?%s", c.endpoints.base(CleSkill), url.PathEscape(matchID), params.Encode())
	body, err := c.get(ctx, requete{url: endpoint, auth: auth})
	if err != nil {
		return nil, fmt.Errorf("GetMatchSkill(%s): %w", matchID, err)
	}
	dto, err := DecodeMatchSkill(body)
	if err != nil {
		return nil, fmt.Errorf("GetMatchSkill decode(%s): %w", matchID, err)
	}
	return dto, nil
}

// DecodeMatchSkill décode un corps brut de réponse skill (la même forme est stockée hors ligne par
// OpenSpartan). Un corps vide n'est pas une réponse skill : erreur de décodage JSON ; c'est à
// l'appelant hors ligne de décider qu'un corps stocké vide vaut « aucun skill ».
func DecodeMatchSkill(body []byte) (*MatchSkillDTO, error) {
	var dto MatchSkillDTO
	if err := json.Unmarshal(body, &dto); err != nil {
		return nil, err
	}
	return &dto, nil
}

// PlayerCSRsDTO : la réponse brute de `GET /{jeu}/players/xuid({xuid})/csrs?Season={id}`.
type PlayerCSRsDTO struct {
	Value []struct {
		ID         string `json:"Id"`
		ResultCode int    `json:"ResultCode"`
		Result     struct {
			Current      CSRRankDTO `json:"Current"`
			SeasonMax    CSRRankDTO `json:"SeasonMax"`
			AllTimeMax   CSRRankDTO `json:"AllTimeMax"`
			PlaylistID   string     `json:"PlaylistId"`
			PlaylistName string     `json:"PlaylistName"`
			Queue        string     `json:"Queue"`
			Input        string     `json:"Input"`
		} `json:"Result"`
	} `json:"Value"`
}

// GetPlayerCSRs récupère les classements CSR d'un joueur pour les playlists classées qu'il a
// jouées dans une saison ([CleSkill]).
func (c *Client) GetPlayerCSRs(ctx context.Context, auth Auth, xuid, seasonID string) (*PlayerCSRsDTO, error) {
	if strings.TrimSpace(xuid) == "" {
		return nil, fmt.Errorf("GetPlayerCSRs: xuid vide")
	}
	if strings.TrimSpace(seasonID) == "" {
		return nil, fmt.Errorf("GetPlayerCSRs: seasonID vide")
	}
	endpoint := fmt.Sprintf("%s/players/xuid(%s)/csrs?Season=%s", c.endpoints.base(CleSkill),
		url.PathEscape(xuid), url.QueryEscape(seasonID))
	body, err := c.get(ctx, requete{url: endpoint, auth: auth})
	if err != nil {
		return nil, fmt.Errorf("GetPlayerCSRs(%s): %w", xuid, err)
	}
	var dto PlayerCSRsDTO
	if err := json.Unmarshal(body, &dto); err != nil {
		return nil, fmt.Errorf("GetPlayerCSRs decode(%s): %w", xuid, err)
	}
	return &dto, nil
}

// PlaylistCSRDTO : la réponse brute de `GET /{jeu}/playlist/{id}/csrs?players=xuid(X)&season=...`.
// La playlist est dans l'URL : Result ne porte ni PlaylistId, ni Queue, ni Input.
type PlaylistCSRDTO struct {
	Value []struct {
		ID         string `json:"Id"`
		ResultCode int    `json:"ResultCode"`
		Result     struct {
			Current    CSRRankDTO `json:"Current"`
			SeasonMax  CSRRankDTO `json:"SeasonMax"`
			AllTimeMax CSRRankDTO `json:"AllTimeMax"`
		} `json:"Result"`
	} `json:"Value"`
}

// GetPlaylistCsr récupère le CSR d'un joueur pour UNE playlist classée ([CleSkill]) ; le service
// répond pour toute playlist, y compris jamais jouée. `seasonID` vide : saison courante.
func (c *Client) GetPlaylistCsr(ctx context.Context, auth Auth, playlistID, xuid, seasonID string) (*PlaylistCSRDTO, error) {
	if strings.TrimSpace(playlistID) == "" {
		return nil, fmt.Errorf("GetPlaylistCsr: playlistID vide")
	}
	if strings.TrimSpace(xuid) == "" {
		return nil, fmt.Errorf("GetPlaylistCsr: xuid vide")
	}
	endpoint := fmt.Sprintf("%s/playlist/%s/csrs?players=xuid(%s)", c.endpoints.base(CleSkill),
		url.PathEscape(playlistID), url.PathEscape(xuid))
	if s := strings.TrimSpace(seasonID); s != "" {
		endpoint += "&season=" + url.QueryEscape(s)
	}
	body, err := c.get(ctx, requete{url: endpoint, auth: auth})
	if err != nil {
		return nil, fmt.Errorf("GetPlaylistCsr(%s): %w", playlistID, err)
	}
	var dto PlaylistCSRDTO
	if err := json.Unmarshal(body, &dto); err != nil {
		return nil, fmt.Errorf("GetPlaylistCsr decode(%s): %w", playlistID, err)
	}
	return &dto, nil
}

// FilterHumanXUIDs retire les bots (`bid(N.0)`) et les xuid vides.
func FilterHumanXUIDs(xuids []string) []string {
	out := make([]string, 0, len(xuids))
	for _, x := range xuids {
		x = strings.TrimSpace(x)
		if x == "" || strings.HasPrefix(x, "bid(") {
			continue
		}
		out = append(out, x)
	}
	return out
}

// UnwrapXUID retire l'enveloppe `xuid(...)` : « xuid(2533274858283686) » -> « 2533274858283686 ».
func UnwrapXUID(wrapped string) string {
	wrapped = strings.TrimSpace(wrapped)
	if strings.HasPrefix(wrapped, "xuid(") && strings.HasSuffix(wrapped, ")") {
		return wrapped[5 : len(wrapped)-1]
	}
	return wrapped
}
