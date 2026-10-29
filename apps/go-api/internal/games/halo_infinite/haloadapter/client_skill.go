package haloadapter

// client_skill.go — skill d'un match (MMR d'équipe, MMR adverse dérivé, attendus, CSR avant/après
// d'un match classé) et classements CSR d'un joueur. Une absence côté service (404/410 : match
// personnalisé, local, bots seuls ; joueur non classé) est un résultat VIDE, pas une erreur.

import (
	"context"
	"errors"
	"fmt"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/haloapi/infinite"
)

// MatchSkillData regroupe le skill d'un joueur dans un match ; un champ nil n'est pas fourni par
// le service.
type MatchSkillData struct {
	XUID           string
	TeamMMR        *float64
	EnemyMMR       *float64
	KillsExpected  *float64
	KillsStdDev    *float64
	DeathsExpected *float64
	DeathsStdDev   *float64
	// PreMatchCSR / PostMatchCSR : CSR avant/après le match, présents seulement pour un match
	// classé (RankRecap) ; nil pour les matchs sociaux, firefight, personnalisés.
	PreMatchCSR  *CSRRankSnapshot
	PostMatchCSR *CSRRankSnapshot
}

// CSRRankSnapshot est un instantané de classement (courant, saison, historique).
type CSRRankSnapshot struct {
	Value                       float64
	Tier                        string
	SubTier                     int
	MeasurementMatchesRemaining int
	// InitialMeasurementMatches : matchs de placement de la saison pour la file ; > 0 sur tout
	// match classé, 0 sur un match social (critère unique de classement d'un match, D-10).
	InitialMeasurementMatches int
}

// PlayerPlaylistCSR regroupe les classements d'un joueur pour une playlist classée.
type PlayerPlaylistCSR struct {
	PlaylistID   string
	PlaylistName string
	Queue        string // "SoloAndDuo" | "Open" | ""
	Input        string // "Crossplay" | "Keyboard" | "Controller" | ""
	Current      CSRRankSnapshot
	Season       CSRRankSnapshot
	AllTime      CSRRankSnapshot
}

// ErrSkillStatsUnavailable : le jeton n'a pas accès au service skill. Non bloquant : la
// synchronisation continue sans skill.
var ErrSkillStatsUnavailable = errors.New("skill stats unavailable")

// GetMatchSkill récupère le skill d'un match pour les xuid donnés : xuid -> données ; un xuid
// absent de la réponse (bot, joueur non classé) n'est pas dans la map. 404/410 : map vide, nil ;
// 401/403 : erreur.
func (c *Client) GetMatchSkill(ctx context.Context, matchID string, xuids []string) (map[string]*MatchSkillData, error) {
	dto, err := c.api(ctx, surfaceAPIGet, infinite.CleSkill).GetMatchSkill(ctx, c.auth(), matchID, xuids)
	if err != nil {
		if infinite.IsNotFound(err) {
			return map[string]*MatchSkillData{}, nil
		}
		return nil, err
	}
	return transformMatchSkillResponse(*dto), nil
}

// ParseMatchSkillResponseJSON décode un corps brut de réponse skill (OpenSpartan stocke ce corps
// tel quel) avec exactement le décodage du chemin live ; corps vide : map vide.
func ParseMatchSkillResponseJSON(body []byte) (map[string]*MatchSkillData, error) {
	dto, err := infinite.DecodeMatchSkill(body)
	if err != nil {
		return nil, fmt.Errorf("ParseMatchSkillResponseJSON: %w", err)
	}
	return transformMatchSkillResponse(*dto), nil
}

// transformMatchSkillResponse convertit la réponse skill en map xuid -> données : MMR d'équipe
// connu seulement (`analysis.MMRKnown`), MMR adverse moyen, attendus, CSR avant/après.
func transformMatchSkillResponse(resp infinite.MatchSkillDTO) map[string]*MatchSkillData {
	out := make(map[string]*MatchSkillData, len(resp.Value))
	for _, v := range resp.Value {
		if v.ResultCode != 0 {
			continue // pas de skill (joueur invité, bot, etc.)
		}
		xuid := infinite.UnwrapXUID(v.ID)
		if xuid == "" {
			continue
		}
		data := &MatchSkillData{XUID: xuid}
		if tm := float64(v.Result.TeamMMR); analysis.MMRKnown(tm) {
			data.TeamMMR = &tm
		}
		teamMMRs := make(map[string]float64, len(v.Result.TeamMMRs))
		for k, val := range v.Result.TeamMMRs {
			teamMMRs[k] = float64(val)
		}
		data.EnemyMMR = computeEnemyMMR(v.Result.TeamID, teamMMRs)
		if sp := v.Result.StatPerformances; sp != nil {
			if sp.Kills != nil {
				ke, sd := float64(sp.Kills.Expected), float64(sp.Kills.StdDev)
				data.KillsExpected, data.KillsStdDev = &ke, &sd
			}
			if sp.Deaths != nil {
				de, sd := float64(sp.Deaths.Expected), float64(sp.Deaths.StdDev)
				data.DeathsExpected, data.DeathsStdDev = &de, &sd
			}
		}
		if rr := v.Result.RankRecap; rr != nil {
			data.PreMatchCSR = snapshotOuNil(rr.PreMatchCsr)
			data.PostMatchCSR = snapshotOuNil(rr.PostMatchCsr)
		}
		out[xuid] = data
	}
	return out
}

// snapshotOuNil convertit un CSR brut présent ; nil reste nil.
func snapshotOuNil(r *infinite.CSRRankDTO) *CSRRankSnapshot {
	if r == nil {
		return nil
	}
	snap := rawToCSRSnapshot(*r)
	return &snap
}

// computeEnemyMMR moyenne le MMR des équipes autres que selfTeamID. nil si seule l'équipe du
// joueur est présente, ou si aucune équipe adverse n'a de MMR réel (une équipe à 0 est une valeur
// absente, `analysis.MMRKnown` : elle ne tire pas la moyenne vers le bas).
func computeEnemyMMR(selfTeamID int, teamMMRs map[string]float64) *float64 {
	if len(teamMMRs) < 2 {
		return nil
	}
	selfKey := fmt.Sprintf("%d", selfTeamID)
	var sum float64
	var count int
	for k, v := range teamMMRs {
		if k == selfKey || !analysis.MMRKnown(v) {
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

func rawToCSRSnapshot(r infinite.CSRRankDTO) CSRRankSnapshot {
	return CSRRankSnapshot{
		Value:                       r.Value,
		Tier:                        r.Tier,
		SubTier:                     r.SubTier,
		MeasurementMatchesRemaining: r.MeasurementMatchesRemaining,
		InitialMeasurementMatches:   r.InitialMeasurementMatches,
	}
}

// GetPlayerCSRs récupère les classements CSR du joueur pour les playlists classées qu'il a jouées
// dans une saison ; 404/410 : tranche vide, nil.
func (c *Client) GetPlayerCSRs(ctx context.Context, xuid, seasonID string) ([]PlayerPlaylistCSR, error) {
	dto, err := c.api(ctx, surfaceAPIGet, infinite.CleSkill).GetPlayerCSRs(ctx, c.auth(), xuid, seasonID)
	if err != nil {
		if infinite.IsNotFound(err) {
			return []PlayerPlaylistCSR{}, nil
		}
		return nil, err
	}
	out := make([]PlayerPlaylistCSR, 0, len(dto.Value))
	for _, v := range dto.Value {
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

// GetPlaylistCsr récupère le CSR d'un joueur pour UNE playlist classée et une saison (vide :
// courante) ; le service répond pour toute playlist, y compris jamais jouée (« Non classé »).
// (nil, nil) sur 404/410 ou sans entrée pour le xuid. PlaylistName, Queue et Input restent vides :
// l'appelant les renseigne depuis la référence des playlists classées.
func (c *Client) GetPlaylistCsr(ctx context.Context, playlistID, xuid, seasonID string) (*PlayerPlaylistCSR, error) {
	dto, err := c.api(ctx, surfaceAPIGet, infinite.CleSkill).GetPlaylistCsr(ctx, c.auth(), playlistID, xuid, seasonID)
	if err != nil {
		if infinite.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return playlistCSRDepuisDTO(dto, playlistID, xuid), nil
}

// playlistCSRDepuisDTO rend le CSR du xuid demandé dans la réponse ; nil sans entrée valide pour
// lui (non classé sans donnée).
func playlistCSRDepuisDTO(dto *infinite.PlaylistCSRDTO, playlistID, xuid string) *PlayerPlaylistCSR {
	for _, v := range dto.Value {
		if v.ResultCode != 0 || infinite.UnwrapXUID(v.ID) != xuid {
			continue
		}
		return &PlayerPlaylistCSR{
			PlaylistID: playlistID,
			Current:    rawToCSRSnapshot(v.Result.Current),
			Season:     rawToCSRSnapshot(v.Result.SeasonMax),
			AllTime:    rawToCSRSnapshot(v.Result.AllTimeMax),
		}
	}
	return nil
}
