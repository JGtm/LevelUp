// Package sync — halo_skill.go : endpoint skill (team_mmr, kills_expected, deaths_expected).
//
// Endpoint :
//
//	GET https://skill.svc.halowaypoint.com:443/hi/matches/{match_id}/skill?players=xuid(X)&players=xuid(Y)
//
// Portage du SkillService Python (spnkr/services/skill.py::get_match_skill).
//
// Le payload de réponse a la forme :
//
//	{
//	  "Value": [
//	    {
//	      "Id": "xuid(2533274858283686)",
//	      "ResultCode": 0,
//	      "Result": {
//	        "TeamMmr":  1500.0,
//	        "TeamId":   0,
//	        "TeamMmrs": { "0": 1500.0, "1": 1450.0 },
//	        "StatPerformances": {
//	          "Kills":  { "Count": 12, "Expected": 10.5, "StdDev": 2.3 },
//	          "Deaths": { "Count":  8, "Expected":  9.1, "StdDev": 1.5 }
//	        }
//	      }
//	    }, ...
//	  ]
//	}
//
// `enemy_mmr` est dérivé : TeamMmrs[other_team_id]. En 2 équipes c'est trivial,
// en FFA (multiples teams) on prend la moyenne des autres équipes.
//
// Les bots (xuid `bid(N.0)`) sont filtrés en amont — l'API skill ne renvoie
// rien pour eux.
package haloclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"levelup/go-api/internal/games"
)

const haloSkillHost = "https://skill.svc.halowaypoint.com:443"

// MatchSkillData regroupe les champs skill extraits pour un joueur.
// Champs *float64 : nil si non fournis par l'API.
type MatchSkillData struct {
	XUID           string
	TeamMMR        *float64
	EnemyMMR       *float64
	KillsExpected  *float64
	KillsStdDev    *float64
	DeathsExpected *float64
	DeathsStdDev   *float64
	// PreMatchCSR / PostMatchCSR : snapshot CSR avant/après le match, présent
	// uniquement pour les matchs classés (champ RankRecap du payload skill).
	// nil pour matchs sociaux / firefight / custom.
	PreMatchCSR  *CSRRankSnapshot
	PostMatchCSR *CSRRankSnapshot
}

// flexFloat accepte un float64 OU une string JSON contenant un nombre.
// L'API Halo skill renvoie parfois Expected/StdDev en string (notation
// scientifique pour très petits nombres) ; sans ce parser on a un decode
// error et le heal saute le match.
type flexFloat float64

func (f *flexFloat) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	// Cas 1 : nombre brut.
	var n float64
	if err := json.Unmarshal(b, &n); err == nil {
		*f = flexFloat(n)
		return nil
	}
	// Cas 2 : string contenant un nombre.
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
	*f = flexFloat(n)
	return nil
}

// matchSkillResponse est la déserialisation directe du JSON Halo.
type matchSkillResponse struct {
	Value []struct {
		ID         string `json:"Id"`
		ResultCode int    `json:"ResultCode"`
		Result     struct {
			TeamMMR          flexFloat            `json:"TeamMmr"`
			TeamID           int                  `json:"TeamId"`
			TeamMMRs         map[string]flexFloat `json:"TeamMmrs"`
			StatPerformances *struct {
				Kills *struct {
					Count    int       `json:"Count"`
					Expected flexFloat `json:"Expected"`
					StdDev   flexFloat `json:"StdDev"`
				} `json:"Kills"`
				Deaths *struct {
					Count    int       `json:"Count"`
					Expected flexFloat `json:"Expected"`
					StdDev   flexFloat `json:"StdDev"`
				} `json:"Deaths"`
			} `json:"StatPerformances"`
			RankRecap *struct {
				PreMatchCsr  *csrRankRaw `json:"PreMatchCsr"`
				PostMatchCsr *csrRankRaw `json:"PostMatchCsr"`
			} `json:"RankRecap"`
		} `json:"Result"`
	} `json:"Value"`
}

// GetMatchSkill récupère les stats skill (MMR, expected) d'un match pour les
// XUIDs donnés. Retourne une map xuid → MatchSkillData ; les XUIDs absents de
// la réponse (bots, joueurs non classés) ne sont pas dans la map.
//
// Erreurs 404/410 (skill absent — match custom/local) → (map vide, nil).
// Erreurs 401/403 (token sans droits) → (nil, error).
func (c *HaloAPIClient) GetMatchSkill(
	ctx context.Context,
	matchID string,
	xuids []string,
) (map[string]*MatchSkillData, error) {
	if !rexUUID.MatchString(matchID) {
		return nil, fmt.Errorf("GetMatchSkill: matchID invalide %q", matchID)
	}
	humans := filterHumanXUIDs(xuids)
	if len(humans) == 0 {
		return map[string]*MatchSkillData{}, nil
	}

	endpoint := fmt.Sprintf("%s/%s/matches/%s/skill", c.hostFor(ctx, games.EndpointSkill, haloSkillHost), c.gamePrefix(ctx), url.PathEscape(matchID))
	params := url.Values{}
	for _, x := range humans {
		params.Add("players", "xuid("+x+")")
	}

	body, err := c.doGet(ctx, endpoint+"?"+params.Encode())
	if err != nil {
		// Skill absent (custom/local/bots-only) → comportement gracieux.
		if isNotFoundErr(err) {
			return map[string]*MatchSkillData{}, nil
		}
		return nil, fmt.Errorf("GetMatchSkill(%s): %w", matchID, err)
	}

	var resp matchSkillResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("GetMatchSkill decode(%s): %w", matchID, err)
	}
	return transformMatchSkillResponse(resp), nil
}

// ParseMatchSkillResponseJSON décode un corps brut de réponse skill Halo en map
// xuid → MatchSkillData (TeamMmr, StatPerformances, et snapshots CSR RankRecap).
//
// OpenSpartan stocke ce payload verbatim dans sa table PlayerMatchStats
// (ResponseBody) — même forme que la réponse live /hi/matches/{id}/skill. L'import
// OpenSpartan réutilise donc cette fonction pour récupérer le CSR par-match hors
// ligne, avec exactement le même décodage (et le même traitement RankRecap) que
// le chemin live GetMatchSkill. Corps vide → map vide (pas d'erreur).
func ParseMatchSkillResponseJSON(body []byte) (map[string]*MatchSkillData, error) {
	if len(body) == 0 {
		return map[string]*MatchSkillData{}, nil
	}
	var resp matchSkillResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("ParseMatchSkillResponseJSON: %w", err)
	}
	return transformMatchSkillResponse(resp), nil
}

// transformMatchSkillResponse convertit le payload skill Halo décodé en map
// xuid → MatchSkillData. Extrait en helper pur (sans IO) pour faciliter les
// tests unitaires du parser, notamment du champ RankRecap (CSR pré/post-match).
func transformMatchSkillResponse(resp matchSkillResponse) map[string]*MatchSkillData {
	out := make(map[string]*MatchSkillData, len(resp.Value))
	for _, v := range resp.Value {
		if v.ResultCode != 0 {
			continue // pas de skill (joueur invité, bot, etc.)
		}
		xuid := unwrapXUID(v.ID)
		if xuid == "" {
			continue
		}
		data := &MatchSkillData{XUID: xuid}
		if tm := float64(v.Result.TeamMMR); tm > 0 {
			data.TeamMMR = &tm
		}
		// Convertir TeamMMRs en map[string]float64 pour computeEnemyMMR.
		teamMMRs := make(map[string]float64, len(v.Result.TeamMMRs))
		for k, val := range v.Result.TeamMMRs {
			teamMMRs[k] = float64(val)
		}
		if em := computeEnemyMMR(v.Result.TeamID, teamMMRs); em != nil {
			data.EnemyMMR = em
		}
		if sp := v.Result.StatPerformances; sp != nil {
			if sp.Kills != nil {
				ke, sd := float64(sp.Kills.Expected), float64(sp.Kills.StdDev)
				data.KillsExpected = &ke
				data.KillsStdDev = &sd
			}
			if sp.Deaths != nil {
				de, sd := float64(sp.Deaths.Expected), float64(sp.Deaths.StdDev)
				data.DeathsExpected = &de
				data.DeathsStdDev = &sd
			}
		}
		if rr := v.Result.RankRecap; rr != nil {
			if rr.PreMatchCsr != nil {
				snap := rawToCSRSnapshot(*rr.PreMatchCsr)
				data.PreMatchCSR = &snap
			}
			if rr.PostMatchCsr != nil {
				snap := rawToCSRSnapshot(*rr.PostMatchCsr)
				data.PostMatchCSR = &snap
			}
		}
		out[xuid] = data
	}
	return out
}

// computeEnemyMMR moyenne le MMR des équipes autres que selfTeamID.
// Retourne nil si seule l'équipe du joueur est présente (FFA 1v0, edge case).
func computeEnemyMMR(selfTeamID int, teamMMRs map[string]float64) *float64 {
	if len(teamMMRs) < 2 {
		return nil
	}
	selfKey := fmt.Sprintf("%d", selfTeamID)
	var sum float64
	var count int
	for k, v := range teamMMRs {
		if k == selfKey {
			continue
		}
		sum += v
		count++
	}
	if count == 0 {
		return nil
	}
	avg := sum / float64(count)
	return &avg
}

// filterHumanXUIDs retire les bots ("bid(N.0)") et XUIDs vides.
func filterHumanXUIDs(xuids []string) []string {
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

// unwrapXUID retire le wrapper "xuid(...)" pour ne garder que les chiffres.
// "xuid(2533274858283686)" → "2533274858283686"
func unwrapXUID(wrapped string) string {
	wrapped = strings.TrimSpace(wrapped)
	if strings.HasPrefix(wrapped, "xuid(") && strings.HasSuffix(wrapped, ")") {
		return wrapped[5 : len(wrapped)-1]
	}
	return wrapped
}

// ErrSkillStatsUnavailable est retournée par GetMatchSkill quand le token n'a
// pas accès au skill endpoint. Non-bloquant : le sync continue sans skill data.
var ErrSkillStatsUnavailable = errors.New("skill stats unavailable")

// ---------------------------------------------------------------------------
// Player CSR (classement compétitif par playlist)
// ---------------------------------------------------------------------------

// CSRRankSnapshot est un instantané de classement (current/season/alltime).
type CSRRankSnapshot struct {
	Value                       float64
	Tier                        string
	SubTier                     int
	MeasurementMatchesRemaining int
}

// PlayerPlaylistCSR regroupe les classements d'un joueur pour une playlist ranked.
type PlayerPlaylistCSR struct {
	PlaylistID   string
	PlaylistName string
	Queue        string // "SoloAndDuo" | "Open" | ""
	Input        string // "Crossplay" | "Keyboard" | "Controller" | ""
	Current      CSRRankSnapshot
	Season       CSRRankSnapshot
	AllTime      CSRRankSnapshot
}

// playerCSRResponse est la désérialisation brute du JSON Waypoint
// GET https://skill.svc.halowaypoint.com/hi/players/xuid({xuid})/csrs?Season={id}
type playerCSRResponse struct {
	Value []struct {
		ID         string `json:"Id"`
		ResultCode int    `json:"ResultCode"`
		Result     struct {
			Current      csrRankRaw `json:"Current"`
			SeasonMax    csrRankRaw `json:"SeasonMax"`
			AllTimeMax   csrRankRaw `json:"AllTimeMax"`
			PlaylistID   string     `json:"PlaylistId"`
			PlaylistName string     `json:"PlaylistName"`
			Queue        string     `json:"Queue"`
			Input        string     `json:"Input"`
		} `json:"Result"`
	} `json:"Value"`
}

type csrRankRaw struct {
	Value                       float64 `json:"Value"`
	MeasurementMatchesRemaining int     `json:"MeasurementMatchesRemaining"`
	Tier                        string  `json:"Tier"`
	SubTier                     int     `json:"SubTier"`
}

func rawToCSRSnapshot(r csrRankRaw) CSRRankSnapshot {
	return CSRRankSnapshot{
		Value:                       r.Value,
		Tier:                        r.Tier,
		SubTier:                     r.SubTier,
		MeasurementMatchesRemaining: r.MeasurementMatchesRemaining,
	}
}

// GetPlayerCSRs récupère les classements CSR du joueur pour toutes les playlists
// ranked d'une saison donnée. Utilise le service token (endpoint public).
func (c *HaloAPIClient) GetPlayerCSRs(ctx context.Context, xuid, seasonID string) ([]PlayerPlaylistCSR, error) {
	if strings.TrimSpace(xuid) == "" {
		return nil, fmt.Errorf("GetPlayerCSRs: xuid vide")
	}
	if strings.TrimSpace(seasonID) == "" {
		return nil, fmt.Errorf("GetPlayerCSRs: seasonID vide")
	}

	endpoint := fmt.Sprintf(
		"%s/%s/players/xuid(%s)/csrs?Season=%s",
		c.hostFor(ctx, games.EndpointSkill, haloSkillHost),
		c.gamePrefix(ctx),
		url.PathEscape(xuid),
		url.QueryEscape(seasonID),
	)

	body, err := c.doGet(ctx, endpoint)
	if err != nil {
		if isNotFoundErr(err) {
			return []PlayerPlaylistCSR{}, nil
		}
		return nil, fmt.Errorf("GetPlayerCSRs(%s): %w", xuid, err)
	}

	var resp playerCSRResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("GetPlayerCSRs decode(%s): %w", xuid, err)
	}

	out := make([]PlayerPlaylistCSR, 0, len(resp.Value))
	for _, v := range resp.Value {
		if v.ResultCode != 0 {
			continue
		}
		out = append(out, PlayerPlaylistCSR{
			PlaylistID:   v.Result.PlaylistID,
			PlaylistName: v.Result.PlaylistName,
			Queue:        v.Result.Queue,
			Input:        v.Result.Input,
			Current:      rawToCSRSnapshot(v.Result.Current),
			Season:       rawToCSRSnapshot(v.Result.SeasonMax),
			AllTime:      rawToCSRSnapshot(v.Result.AllTimeMax),
		})
	}
	return out, nil
}

// GetPlaylistCsr (CSR par playlist, endpoint /hi/playlist/{id}/csrs) vit dans
// halo_skill_csr.go (extrait pour la règle 500 lignes/fichier).
