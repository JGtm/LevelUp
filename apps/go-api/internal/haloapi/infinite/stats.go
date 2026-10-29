package infinite

// stats.go — historique et stats des matchs, service record. Copié de `sync/haloclient/halo_client.go`
// (`GetMatchHistory`, `GetMatchStats`), de `platform/halo/discovery_client.go` (`FetchMatchStats`)
// et de `platform/halo/compare_provider.go` (GET du service record, `buildSeasonServiceRecordURL`) ;
// les calculs de domaine (ratios, FDA, médailles) restent chez l'appelant.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// MatchCountMax : nombre maximal de matchs par page d'historique.
const MatchCountMax = 25

// validMatchType dit si `t` est un type de match accepté par l'historique.
func validMatchType(t string) bool {
	switch t {
	case "all", "matchmaking", "custom", "local":
		return true
	}
	return false
}

// rexUUID valide un identifiant de match (UUID).
var rexUUID = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// EstUUID dit si `id` a la forme d'un identifiant de match (UUID), celle que tout service de la brique
// exige avant d'émettre. Un appelant qui consulte un cache avant la brique valide d'abord par elle.
func EstUUID(id string) bool { return rexUUID.MatchString(id) }

// MatchHistoryDTO : la réponse brute de `GET /{jeu}/players/{joueur}/matches`.
type MatchHistoryDTO struct {
	Results []MatchHistoryResultDTO `json:"Results"`
}

// MatchHistoryResultDTO : une entrée de l'historique.
type MatchHistoryResultDTO struct {
	MatchID   string `json:"MatchId"`
	MatchInfo struct {
		StartTime string `json:"StartTime"`
	} `json:"MatchInfo"`
}

// PageHistorique désigne une page de l'historique des matchs d'un joueur.
type PageHistorique struct {
	// Joueur : au format `xuid(NNN)` (un gamertag textuel rend un historique figé).
	Joueur string
	// MatchType : "all", "matchmaking", "custom", "local".
	MatchType string
	// Start : décalage (0 = plus récent).
	Start int
	// Count : taille de la page, 1..[MatchCountMax].
	Count int
}

// GetMatchHistory récupère une page de l'historique des matchs d'un joueur ([CleStats]).
func (c *Client) GetMatchHistory(ctx context.Context, auth Auth, p PageHistorique) (*MatchHistoryDTO, error) {
	joueur, matchType, start, count := p.Joueur, p.MatchType, p.Start, p.Count
	if strings.TrimSpace(joueur) == "" {
		return nil, errors.New("GetMatchHistory: gamertag vide")
	}
	if !validMatchType(matchType) {
		return nil, fmt.Errorf("GetMatchHistory: matchType invalide %q (attendu : all|matchmaking|custom|local)", matchType)
	}
	if start < 0 {
		return nil, fmt.Errorf("GetMatchHistory: start doit être ≥ 0 (reçu %d)", start)
	}
	if count < 1 || count > MatchCountMax {
		return nil, fmt.Errorf("GetMatchHistory: count doit être entre 1 et %d (reçu %d)", MatchCountMax, count)
	}
	params := url.Values{"start": {strconv.Itoa(start)}, "count": {strconv.Itoa(count)}, "type": {matchType}}
	endpoint := fmt.Sprintf("%s/players/%s/matches?%s", c.endpoints.base(CleStats), url.PathEscape(joueur), params.Encode())
	body, err := c.get(ctx, requete{url: endpoint, auth: auth})
	if err != nil {
		return nil, fmt.Errorf("GetMatchHistory: %w", err)
	}
	var resp MatchHistoryDTO
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("GetMatchHistory decode: %w", err)
	}
	return &resp, nil
}

// GetMatchStats récupère les stats détaillées d'un match ([CleStats], avec `:443`) : le JSON brut.
func (c *Client) GetMatchStats(ctx context.Context, auth Auth, matchID string) (map[string]any, error) {
	if !rexUUID.MatchString(matchID) {
		return nil, fmt.Errorf("GetMatchStats: matchID n'est pas un UUID valide %q", matchID)
	}
	endpoint := fmt.Sprintf("%s/matches/%s/stats", c.endpoints.base(CleStats), url.PathEscape(matchID))
	body, err := c.get(ctx, requete{url: endpoint, auth: auth})
	if err != nil {
		return nil, fmt.Errorf("GetMatchStats(%s): %w", matchID, err)
	}
	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("GetMatchStats decode(%s): %w", matchID, err)
	}
	return result, nil
}

// xuidArbitraire : `FetchMatchStats` interroge les stats par un joueur quelconque ; `MatchInfo` est
// global au match.
const xuidArbitraire = "0"

// FetchMatchStats récupère les stats complètes d'un match par le chemin joueur
// (`/players/xuid(0)/matches/{id}/stats`, [CleStatsSansPort]) : le JSON brut. C'est l'hôte SANS
// `:443` du fournisseur, distinct de [Client.GetMatchStats].
func (c *Client) FetchMatchStats(ctx context.Context, auth Auth, matchID string) (map[string]any, error) {
	endpoint := fmt.Sprintf("%s/players/xuid(%s)/matches/%s/stats", c.endpoints.base(CleStatsSansPort), xuidArbitraire, matchID)
	body, err := c.get(ctx, requete{url: endpoint, auth: auth})
	if err != nil {
		return nil, fmt.Errorf("fetch match stats %s: %w", matchID, err)
	}
	var stats map[string]any
	if err := json.Unmarshal(body, &stats); err != nil {
		return nil, fmt.Errorf("decode match stats %s: %w", matchID, err)
	}
	return stats, nil
}

// ServiceRecordDTO : la réponse brute de `GET /{jeu}/players/{joueur}/Matchmade/servicerecord`.
// Wins/Losses/MatchesCompleted et TimePlayed (ISO-8601) au premier niveau, CoreStats (+ Medals) en
// sous-objet, Subqueries (saisons jouées, classé, playlists) pour le service record cumulatif.
type ServiceRecordDTO struct {
	MatchesCompleted int    `json:"MatchesCompleted"`
	Wins             int    `json:"Wins"`
	Losses           int    `json:"Losses"`
	TimePlayed       string `json:"TimePlayed"`
	CoreStats        struct {
		Kills       int     `json:"Kills"`
		Deaths      int     `json:"Deaths"`
		Assists     int     `json:"Assists"`
		ShotsFired  float64 `json:"ShotsFired"`
		ShotsHit    float64 `json:"ShotsHit"`
		DamageDealt float64 `json:"DamageDealt"`
		DamageTaken float64 `json:"DamageTaken"`
		Medals      []struct {
			NameID int64 `json:"NameId"`
			Count  int   `json:"Count"`
		} `json:"Medals"`
	} `json:"CoreStats"`
	Subqueries struct {
		SeasonIDs        []string `json:"SeasonIds"`
		IsRanked         []bool   `json:"IsRanked"`
		PlaylistAssetIDs []string `json:"PlaylistAssetIds"`
	} `json:"Subqueries"`
}

// GetServiceRecord récupère le service record matchmade CUMULATIF d'un joueur
// ([CleStatsSansPort]). `joueur` est inséré tel quel dans le chemin (gamertag ou `xuid(NNN)`).
func (c *Client) GetServiceRecord(ctx context.Context, auth Auth, joueur string) (*ServiceRecordDTO, error) {
	endpoint := fmt.Sprintf("%s/players/%s/Matchmade/servicerecord", c.endpoints.base(CleStatsSansPort), joueur)
	return c.serviceRecord(ctx, auth, endpoint, "FetchServiceRecord("+joueur+")")
}

// GetSeasonServiceRecord récupère le service record d'UNE saison (`seasonID` = chemin CMS, ex.
// « Seasons/Season7.json »), filtré classé / non classé si `isRanked` est donné.
func (c *Client) GetSeasonServiceRecord(ctx context.Context, auth Auth, joueur, seasonID string, isRanked *bool) (*ServiceRecordDTO, error) {
	if seasonID == "" {
		return nil, errors.New("FetchSeasonServiceRecord: seasonID vide")
	}
	endpoint := BuildSeasonServiceRecordURL(c.endpoints, joueur, seasonID, isRanked)
	return c.serviceRecord(ctx, auth, endpoint, "FetchSeasonServiceRecord("+joueur+", "+seasonID+")")
}

func (c *Client) serviceRecord(ctx context.Context, auth Auth, endpoint, op string) (*ServiceRecordDTO, error) {
	body, err := c.get(ctx, requete{url: endpoint, auth: auth})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var resp ServiceRecordDTO
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("%s parse: %w", op, err)
	}
	return &resp, nil
}

// BuildSeasonServiceRecordURL construit l'URL du service record filtré par saison (et classé si
// donné). Fonction pure ; `seasonId` (avec ses barres) et `isRanked` sont encodés en requête.
func BuildSeasonServiceRecordURL(e Endpoints, joueur, seasonID string, isRanked *bool) string {
	q := url.Values{}
	q.Set("seasonId", seasonID)
	if isRanked != nil {
		q.Set("isRanked", strconv.FormatBool(*isRanked))
	}
	return fmt.Sprintf("%s/players/%s/Matchmade/servicerecord?%s", e.base(CleStatsSansPort), joueur, q.Encode())
}
