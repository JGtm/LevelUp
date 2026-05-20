// Package sync — career.go : synchronisation du rang carrière.
//
// Portage de src/data/sync/_career.py.
// Récupère la progression de rang via l'API economy player-gated
// et insère un snapshot dans career_progression.
package sync

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// CareerRankData contient les données d'un snapshot de rang.
// Portage de src/data/sync/models.py CareerRankData.
type CareerRankData struct {
	XUID             string
	CurrentRank      int
	CurrentRankName  string
	CurrentRankTier  string
	CurrentXP        int
	XPForNextRank    int
	XPTotal          int
	IsMaxRank        bool
	AdornmentPath    string
	SpartanID        string
	BannerImageURL   string
	EmblemImageURL   string
	BackdropImageURL string
}

// parseCareerRank extrait les données de rang depuis la réponse API.
// Utilisée par career_test.go et career_integration_test.go (tag integration).
//
//nolint:unused // référencée uniquement depuis les tests sous tag integration
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

func enrichCareerRankFromMetadata(db *sql.DB, data *CareerRankData) error {
	if db == nil || data == nil {
		return nil
	}

	var (
		titleEN       string
		tierType      sql.NullString
		grade         sql.NullInt64
		xpRequired    int
		adornmentPath sql.NullString
	)
	err := db.QueryRow(
		`SELECT title_en, tier_type, grade, xp_required, adornment_icon_path
		 FROM career_ranks
		 WHERE rank_id = ?`,
		data.CurrentRank,
	).Scan(&titleEN, &tierType, &grade, &xpRequired, &adornmentPath)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("enrichCareerRankFromMetadata row: %w", err)
	}

	data.XPForNextRank = xpRequired
	if tierType.Valid {
		data.CurrentRankTier = strings.TrimSpace(tierType.String)
	}
	data.CurrentRankName = buildCareerRankName(titleEN, data.CurrentRankTier, grade)
	if adornmentPath.Valid {
		data.AdornmentPath = strings.TrimSpace(adornmentPath.String)
	}

	var completedXP int
	if err := db.QueryRow(
		`SELECT COALESCE(SUM(xp_required), 0)
		 FROM career_ranks
		 WHERE rank_id < ?`,
		data.CurrentRank,
	).Scan(&completedXP); err != nil {
		return fmt.Errorf("enrichCareerRankFromMetadata sum: %w", err)
	}
	data.XPTotal = completedXP + data.CurrentXP
	return nil
}

// syncPlayerCSRs récupère les classements CSR du joueur via l'API et les persiste.
// Retourne (0, nil) si la saison est vide ou si l'API ne renvoie rien.
func syncPlayerCSRs(
	ctx context.Context,
	client HaloClient,
	db *sql.DB,
	xuid, seasonID string,
) (int, error) {
	if strings.TrimSpace(seasonID) == "" {
		return 0, nil
	}
	csrs, err := client.GetPlayerCSRs(ctx, xuid, seasonID)
	if err != nil {
		return 0, fmt.Errorf("syncPlayerCSRs fetch: %w", err)
	}
	if len(csrs) == 0 {
		return 0, nil
	}
	return saveCSRSnapshots(db, csrs, seasonID)
}

// saveCSRSnapshots insère ou remplace les snapshots CSR dans player_csr_snapshots.
func saveCSRSnapshots(db *sql.DB, csrs []PlayerPlaylistCSR, seasonID string) (int, error) {
	now := time.Now().UTC()
	var inserted int
	for _, c := range csrs {
		if strings.TrimSpace(c.PlaylistID) == "" {
			continue
		}
		_, err := db.Exec(`
			INSERT OR REPLACE INTO player_csr_snapshots (
				playlist_id, playlist_name, queue, input, season_id,
				current_value, current_tier, current_sub_tier, current_measurement_remaining,
				season_value, season_tier, season_sub_tier,
				alltime_value, alltime_tier, alltime_sub_tier,
				fetched_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			c.PlaylistID, c.PlaylistName, c.Queue, c.Input, seasonID,
			c.Current.Value, c.Current.Tier, c.Current.SubTier, c.Current.MeasurementMatchesRemaining,
			c.Season.Value, c.Season.Tier, c.Season.SubTier,
			c.AllTime.Value, c.AllTime.Tier, c.AllTime.SubTier,
			now,
		)
		if err != nil {
			return inserted, fmt.Errorf("saveCSRSnapshots upsert %s: %w", c.PlaylistID, err)
		}
		inserted++
	}
	return inserted, nil
}

func buildCareerRankName(titleEN, tierType string, grade sql.NullInt64) string {
	titleEN = strings.TrimSpace(titleEN)
	tierType = strings.TrimSpace(tierType)
	if titleEN == "" {
		return ""
	}
	parts := []string{titleEN}
	if tierType != "" {
		parts = append(parts, tierType)
	}
	if grade.Valid && grade.Int64 > 0 {
		parts = append(parts, fmt.Sprintf("%d", grade.Int64))
	}
	return strings.Join(parts, " ")
}
