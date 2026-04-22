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
	"unicode"
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

// syncCareerRank récupère la progression du rang carrière via le client Halo.
// Si le token joueur est absent, la sync est sautée proprement (nil, nil).
func syncCareerRank(
	ctx context.Context,
	client HaloClient,
	xuid string,
) (*CareerRankData, error) {
	// B4 : validation xuid — non vide, numérique, longueur typique 16 chiffres.
	if strings.TrimSpace(xuid) == "" {
		return nil, fmt.Errorf("syncCareerRank: xuid vide")
	}
	for _, r := range xuid {
		if !unicode.IsDigit(r) {
			return nil, fmt.Errorf("syncCareerRank: xuid doit être numérique (reçu %q)", xuid)
		}
	}
	if len(xuid) < 12 || len(xuid) > 20 {
		return nil, fmt.Errorf("syncCareerRank: longueur xuid inattendue %d (attendu 12-20 chiffres)", len(xuid))
	}
	return client.GetCareerRank(ctx, xuid)
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

func openCareerMetadataDB(path string) (*sql.DB, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("openCareerMetadataDB: path vide")
	}
	db, err := sql.Open("duckdb", path+"?access_mode=read_only")
	if err != nil {
		return nil, fmt.Errorf("openCareerMetadataDB: %w", err)
	}
	db.SetMaxOpenConns(1)
	return db, nil
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
