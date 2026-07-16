// Package duckdb - season_pass_repo.go : acces DB pour la page Season Pass.
//
// Le code est decoupe en fichiers thematiques pour respecter la limite des
// 500 lignes par fichier (CLAUDE.md). Ce fichier contient le type repo, le
// constructeur, localBPImageURL et loadTrackSnapshots (loader principal).
// Les autres responsabilites vivent dans :
//
//   - season_pass_repo_tracks.go   : LoadSeasonPassTracks +
//     loadItemMetadataMap +
//     fillItemsFromAssetIndex +
//     buildMinimalTrackSummary
//   - season_pass_repo_builders.go : buildTrackSummary + computeContentSummary +
//     aggregateRewardBucket + parseTrackPayload +
//     collectTrackItemPaths +
//     collectRewardBucketItemPaths +
//     buildTierSummaries + buildTierSummary
//   - season_pass_repo_helpers.go  : free rewards + currency + resolve
//     track/bg/xpPerRank/maxRank + completion +
//     payload helpers + status + ptr helpers
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"path"
	"strings"
	"time"
)

// SeasonPassRepo implémente port.SeasonPassRepository.
type SeasonPassRepo struct {
	pdb *PlayerDB
}

// NewSeasonPassRepo crée un SeasonPassRepo pour un joueur.
func NewSeasonPassRepo(pdb *PlayerDB) *SeasonPassRepo {
	return &SeasonPassRepo{pdb: pdb}
}

// localBPImageURL retourne l'URL proxy locale incluant le chemin GameCMS complet.
// Le handler décodera ce chemin pour construire l'URL GameCMS exacte lors du fetch.
// Ne retourne jamais une URL GameCMS directe — le browser ne peut pas la charger.
func localBPImageURL(gameCMSPath, subDir string) *string {
	trimmed := strings.TrimSpace(strings.ReplaceAll(gameCMSPath, "\\", "/"))
	if trimmed == "" {
		return nil
	}
	// Vérification de sécurité : pas de traversal de répertoire.
	cleaned := path.Clean("/" + strings.TrimLeft(trimmed, "/"))
	if cleaned == "/" || cleaned == "." {
		return nil
	}
	// Toujours retourner l'URL proxy — Go gérera le cache ou le fetch GameCMS.
	// Le chemin complet est inclus pour que le handler construise la bonne URL GameCMS.
	u := "/api/v1/assets/battlepass/" + subDir + cleaned
	return &u
}

// trackSnapshotState est l'état le plus récent d'un reward track pour un joueur.
type trackSnapshotState struct {
	Rank              int
	Partial           int
	IsOwned           bool
	HasReachedMaxRank bool
	IsActive          bool
	SnapshotAt        time.Time // dernier `snapshot_at` connu pour ce track ; zéro si aucun.
}

// trackProgressMap mappe reward_track_path → progression joueur récente.
type trackProgressMap map[string]trackSnapshotState

// loadTrackSnapshots charge la progression du joueur depuis battlepass_snapshots.
// Retourne une map vide (sans erreur) si aucune entrée n'existe.
func (r *SeasonPassRepo) loadTrackSnapshots(ctx context.Context) (trackProgressMap, string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().QueryRecovered(ctx, `
		SELECT reward_track_path, is_active, current_rank, partial_progress,
		       is_owned, has_reached_max_rank, snapshot_at
		FROM (
			SELECT reward_track_path, is_active, current_rank, partial_progress,
			       is_owned, has_reached_max_rank, snapshot_at,
			       ROW_NUMBER() OVER (PARTITION BY reward_track_path ORDER BY snapshot_at DESC) AS rn
			FROM battlepass_snapshots
			WHERE xuid = ?
		) t
		WHERE rn = 1`, r.pdb.XUID)
	if err != nil {
		if isTableNotFoundErr(err) {
			return trackProgressMap{}, "", nil
		}
		return nil, "", fmt.Errorf("season_pass_repo: track snapshots query: %w", err)
	}
	defer rows.Close()

	progressMap := trackProgressMap{}
	activeTrackPath := ""
	var activeSeenAt *time.Time
	for rows.Next() {
		var path string
		var state trackSnapshotState
		var snapshotAt time.Time
		if err := rows.Scan(
			&path,
			&state.IsActive,
			&state.Rank,
			&state.Partial,
			&state.IsOwned,
			&state.HasReachedMaxRank,
			&snapshotAt,
		); err != nil {
			return nil, "", fmt.Errorf("season_pass_repo: track snapshots scan: %w", err)
		}
		state.SnapshotAt = snapshotAt
		progressMap[path] = state
		if state.IsActive && (activeSeenAt == nil || snapshotAt.After(*activeSeenAt)) {
			t := snapshotAt
			activeSeenAt = &t
			activeTrackPath = path
		}
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	return progressMap, activeTrackPath, nil
}

// seasonPassTrackRow représente une ligne du JOIN tracks + translations.
type seasonPassTrackRow struct {
	rewardTrackPath     string
	xpPerRank           sql.NullInt64
	trackName           sql.NullString
	battlepassImagePath sql.NullString
	backgroundImagePath sql.NullString
	rawPayloadJSON      sql.NullString
}

type seasonPassTrackPayload struct {
	Name                any                 `json:"Name"`
	Description         any                 `json:"Description"`
	XpPerRank           int                 `json:"XpPerRank"`
	BattlePassImage     string              `json:"BattlePassImage"`
	SummaryImagePath    string              `json:"SummaryImagePath"`
	BackgroundImagePath string              `json:"BackgroundImagePath"`
	Ranks               []seasonPassRankRaw `json:"Ranks"`
}

type seasonPassRankRaw struct {
	Rank        int                    `json:"Rank"`
	FreeRewards seasonPassRewardBucket `json:"FreeRewards"`
	PaidRewards seasonPassRewardBucket `json:"PaidRewards"`
}

type seasonPassRewardBucket struct {
	InventoryRewards []seasonPassInventoryReward `json:"InventoryRewards"`
	CurrencyRewards  []seasonPassCurrencyReward  `json:"CurrencyRewards"`
}

type seasonPassInventoryReward struct {
	InventoryItemPath string `json:"InventoryItemPath"`
}

// seasonPassCurrencyReward représente une récompense en monnaie virtuelle (cR, XP boost…).
// Certains paliers de Battle Pass donnent uniquement de la monnaie, sans InventoryItem.
type seasonPassCurrencyReward struct {
	CurrencyPath string `json:"CurrencyPath"`
	Amount       int    `json:"Amount"`
}

type seasonPassItemMeta struct {
	Title       string
	Description *string
	ImageURL    *string
	Quality     *string
	ItemType    *string
}

// LoadSeasonPassTracks charge toutes les tracks connues avec traductions.
// La progression joueur est injectée depuis le payload Waypoint persisté.
