// Package sync — career.go : synchronisation du rang carrière.
//
// Portage de src/data/sync/_career.py.
// Récupère la progression de rang via l'API economy player-gated
// et insère un snapshot dans career_progression.
package sync

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// CareerRankData contient les données d'un snapshot de rang.
// Portage de src/data/sync/models.py CareerRankData.
type CareerRankData struct {
	XUID            string
	CurrentRank     int
	CurrentRankName string
	CurrentRankTier string
	CurrentXP       int
	XPForNextRank   int
	XPTotal         int
	IsMaxRank       bool
	AdornmentPath   string
	SpartanID       string
}

// syncCareerRank récupère la progression du rang carrière via l'API.
// Si le token joueur est absent, la sync est sautée proprement (nil, nil).
func syncCareerRank(
	ctx context.Context,
	client *HaloAPIClient,
	xuid string,
) (*CareerRankData, error) {
	url := fmt.Sprintf("https://economy.svc.halowaypoint.com/hi/players/xuid(%s)/customization", xuid)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("syncCareerRank: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-343-authorization-spartan", client.spartanToken)
	req.Header.Set("343-clearance", client.clearanceToken)

	resp, err := client.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("syncCareerRank HTTP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
		// Token joueur absent ou insuffisant — skip gracieusement.
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("syncCareerRank: status %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("syncCareerRank decode: %w", err)
	}

	return parseCareerRank(body, xuid), nil
}

// parseCareerRank extrait les données de rang depuis la réponse API.
func parseCareerRank(body map[string]interface{}, xuid string) *CareerRankData {
	data := &CareerRankData{XUID: xuid}

	if rank, ok := body["Rank"]; ok {
		if r, ok := rank.(map[string]interface{}); ok {
			if v, ok := r["Value"].(float64); ok {
				data.CurrentRank = int(v)
			}
			if v, ok := r["Name"].(string); ok {
				data.CurrentRankName = v
			}
			if v, ok := r["Tier"].(string); ok {
				data.CurrentRankTier = v
			}
		}
	}

	if xp, ok := body["RewardTrack"]; ok {
		if r, ok := xp.(map[string]interface{}); ok {
			if v, ok := r["CurrentProgress"].(float64); ok {
				data.CurrentXP = int(v)
			}
			if v, ok := r["NextLevelRequirement"].(float64); ok {
				data.XPForNextRank = int(v)
			}
			if v, ok := r["TotalEarned"].(float64); ok {
				data.XPTotal = int(v)
			}
			if v, ok := r["IsMaxRank"].(bool); ok {
				data.IsMaxRank = v
			}
		}
	}

	if v, ok := body["AdornmentPath"].(string); ok {
		data.AdornmentPath = v
	}
	if v, ok := body["SpartanId"].(string); ok {
		data.SpartanID = v
	}

	return data
}

// saveCareerRank insère un snapshot de progression dans la player DB.
func saveCareerRank(db *sql.DB, data *CareerRankData) error {
	now := time.Now().UTC()
	_, err := db.Exec(`
		INSERT INTO career_progression (
			xuid, rank, rank_name, rank_tier,
			current_xp, xp_for_next_rank, xp_total,
			is_max_rank, adornment_path, spartan_id, recorded_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		data.XUID, data.CurrentRank, data.CurrentRankName, data.CurrentRankTier,
		data.CurrentXP, data.XPForNextRank, data.XPTotal,
		data.IsMaxRank, data.AdornmentPath, data.SpartanID, now,
	)
	if err != nil {
		return fmt.Errorf("saveCareerRank: %w", err)
	}

	// Mettre à jour sync_meta
	_ = SetSyncMeta(db, "last_career_sync_at", now.Format(time.RFC3339))
	_ = SetSyncMeta(db, "current_rank", fmt.Sprintf("%d", data.CurrentRank))

	return nil
}
