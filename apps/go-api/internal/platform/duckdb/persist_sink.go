// Package duckdb — persist_sink.go : écriture fire-and-forget battlepass / challenges.
//
// Le PersistSink ouvre des connexions read-write vers metadata.duckdb et
// stats.duckdb du joueur pour persister les données Waypoint reçues lors des
// appels live API.  Les goroutines sont détachées (fire-and-forget) : un échec
// de persistance ne fait jamais échouer la réponse HTTP.
//
// Connexions :
//   - metadata.duckdb : waypoint_assets_raw (blob brut archivage)
//   - stats.duckdb    : battlepass_snapshots + challenge_snapshots (append-only)
package duckdb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/port"
)

// PersistSink satisfait le port consommé par HomeService (Phase 2 : le service
// dépend de l'interface, pas du type concret).
var _ port.HomePersistSink = (*PersistSink)(nil)

// PersistSink centralise les écritures battlepass/challenges (fire-and-forget).
// Les ouvertures RW passent par le cache `openDBs`, puis sont relâchées en fin
// d'écriture pour ne pas invalider les autres utilisateurs du même fichier.
type PersistSink struct {
	MetaPath   string // chemin vers metadata.duckdb
	PlayerPath string // chemin vers stats.duckdb du joueur
	XUID       string // xuid du joueur authentifié
	TitleSlug  string // titre courant (waypoint_assets_raw.title_slug) — multi-titres
}

// NewPersistSink crée un PersistSink pour un joueur donné.
func NewPersistSink(metaPath, playerPath, xuid, titleSlug string) *PersistSink {
	return &PersistSink{
		MetaPath:   metaPath,
		PlayerPath: playerPath,
		XUID:       xuid,
		TitleSlug:  titleSlug,
	}
}

// persistHash retourne les 16 premiers caractères du SHA-256 hex (64 bits).
func persistHash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:8])
}

// ---------------------------------------------------------------------------
// Battle Pass
// ---------------------------------------------------------------------------

// PersistBattlePassSync persiste les données BP de manière synchrone : archivage
// dans waypoint_assets_raw (metadata) + battlepass_snapshots (par joueur), via
// fetchRewardTrackDefinition (battlepass_details.go). Bloque jusqu'à la fin des
// écritures sur le ctx appelant, garantissant que les snapshots sont en DB avant
// la prochaine lecture et avant le shutdown (W6 — plus de goroutine détachée).
func (s *PersistSink) PersistBattlePassSync(ctx context.Context, trackPath string, rawBody []byte) error {
	if s.MetaPath == "" || trackPath == "" || len(rawBody) == 0 {
		return nil
	}
	return s.writeBattlePass(ctx, trackPath, rawBody)
}

// battlePassTrackRaw est le struct de parsing best-effort d'un track depuis /operations.
type battlePassTrackRaw struct {
	RewardTrackPath string `json:"RewardTrackPath"`
	CurrentProgress struct {
		Rank              int  `json:"Rank"`
		PartialProgress   int  `json:"PartialProgress"`
		IsOwned           bool `json:"IsOwned"`
		HasReachedMaxRank bool `json:"HasReachedMaxRank"`
	} `json:"CurrentProgress"`
	IsOwned bool `json:"IsOwned"`
	BaseXP  int  `json:"BaseXp"`
	BoostXP int  `json:"BoostXp"`
}

// writeBattlePass persiste le blob brut /operations dans waypoint_assets_raw et
// écrit des snapshots append-only dans stats.duckdb pour refléter la progression
// réelle du joueur par reward track.
//
// NOTE : battlepass_track_definitions n'est plus écrit ici.
// La persistance des définitions de tracks (raw_payload_json, xp_per_rank, images,
// translations) est la responsabilité du HaloProvider qui dispose des tokens Halo
// et fetche la définition GameCMS (/hi/Progression/file/{trackPath}) — voir
// battlepass_details.go / fetchRewardTrackDefinition.
func (s *PersistSink) writeBattlePass(ctx context.Context, _ string, body []byte) error {
	relMeta, err := dblease.AcquireLease(s.MetaPath, dblease.MetadataLeaseTimeout)
	if err != nil {
		return fmt.Errorf("writeBattlePass lease meta: %w", err)
	}
	defer relMeta()

	db, err := OpenReadWrite(s.MetaPath)
	if err != nil {
		return fmt.Errorf("open meta rw: %w", err)
	}
	defer db.Close()

	hash := persistHash(body)
	now := time.Now()

	// 1. Sauvegarder le blob brut dans waypoint_assets_raw (archivage).
	if err := upsertWaypointAsset(ctx, db,
		s.TitleSlug,
		s.XUID+"/battlepass_operations",
		"battlepass_operation",
		hash,
		string(body),
		now,
	); err != nil {
		slog.Warn("persist_sink: waypoint_assets_raw BP upsert failed",
			"xuid", s.XUID, "err", err)
	}

	if s.PlayerPath == "" || s.XUID == "" {
		return nil
	}

	var payload struct {
		ActiveOperationRewardTrackPath string            `json:"ActiveOperationRewardTrackPath"`
		OperationRewardTracks          []json.RawMessage `json:"OperationRewardTracks"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Errorf("parse battlepass body: %w", err)
	}

	relPlayer, err := dblease.AcquireLease(s.PlayerPath, dblease.PlayerLeaseTimeout)
	if err != nil {
		return fmt.Errorf("writeBattlePass lease player: %w", err)
	}
	defer relPlayer()

	pdb, err := OpenReadWrite(s.PlayerPath)
	if err != nil {
		return fmt.Errorf("open player rw: %w", err)
	}
	defer pdb.Close()

	for _, rawTrack := range payload.OperationRewardTracks {
		if err := s.insertBattlePassSnapshot(
			ctx,
			pdb,
			rawTrack,
			payload.ActiveOperationRewardTrackPath,
			now,
		); err != nil {
			slog.Warn("persist_sink: battlepass snapshot insert failed",
				"xuid", s.XUID, "err", err)
		}
	}
	return nil
}

// insertBattlePassSnapshot insère une snapshot de progression battle pass si
// l'état du track a changé dans les dernières 24h.
func (s *PersistSink) insertBattlePassSnapshot(
	ctx context.Context,
	db *DB,
	rawTrack json.RawMessage,
	activeTrackPath string,
	at time.Time,
) error {
	var track battlePassTrackRaw
	if err := json.Unmarshal(rawTrack, &track); err != nil {
		return nil
	}
	if track.RewardTrackPath == "" {
		return nil
	}

	isActive := track.RewardTrackPath == activeTrackPath
	isOwned := track.IsOwned || track.CurrentProgress.IsOwned
	statePayload, err := json.Marshal(struct {
		IsActive bool            `json:"is_active"`
		Track    json.RawMessage `json:"track"`
	}{
		IsActive: isActive,
		Track:    rawTrack,
	})
	if err != nil {
		return fmt.Errorf("marshal battlepass state: %w", err)
	}
	stateHash := persistHash(statePayload)

	var existing int
	err = db.QueryRow(ctx, `
		SELECT COUNT(*) FROM battlepass_snapshots
		WHERE xuid = ? AND reward_track_path = ? AND state_hash = ?
		  AND snapshot_at > CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP) - INTERVAL 1 DAY`,
		s.XUID, track.RewardTrackPath, stateHash,
	).Scan(&existing)
	if err == nil && existing > 0 {
		return nil
	}

	_, err = db.Exec(ctx, `
		INSERT INTO battlepass_snapshots
			(snapshot_at, xuid, reward_track_path, is_active,
			 current_rank, partial_progress, is_owned, has_reached_max_rank,
			 base_xp, boost_xp, state_hash, raw_payload_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		at,
		s.XUID,
		track.RewardTrackPath,
		isActive,
		track.CurrentProgress.Rank,
		track.CurrentProgress.PartialProgress,
		isOwned,
		track.CurrentProgress.HasReachedMaxRank,
		track.BaseXP,
		track.BoostXP,
		stateHash,
		string(rawTrack),
	)
	return err
}

// ---------------------------------------------------------------------------
// Track Definitions (battlepass_track_definitions)
// ---------------------------------------------------------------------------

// trackDefRaw est le struct de parsing minimal d'un JSON Reward Track GameCMS.
// GameCMS utilise BattlePassImage (S05+) ou SummaryImagePath (S03/S04) selon les saisons.
type trackDefRaw struct {
	BattlePassImage     string `json:"BattlePassImage"`
	SummaryImagePath    string `json:"SummaryImagePath"`
	BackgroundImagePath string `json:"BackgroundImagePath"`
	XpPerRank           int    `json:"XpPerRank"`
}

// UpsertTrackDefinition implémente halo.TrackDefinitionPersister.
// Persiste la définition d'un Reward Track dans battlepass_track_definitions de metadata.duckdb.
// Utilise UPDATE-first + INSERT-if-zero (idiome DuckDB) pour garantir l'idempotence.
func (s *PersistSink) UpsertTrackDefinition(ctx context.Context, trackPath string, raw []byte) error {
	var def trackDefRaw
	_ = json.Unmarshal(raw, &def) // best-effort : champs manquants restent vides

	relMeta, err := dblease.AcquireLease(s.MetaPath, dblease.MetadataLeaseTimeout)
	if err != nil {
		return fmt.Errorf("UpsertTrackDefinition lease meta: %w", err)
	}
	defer relMeta()

	db, err := OpenReadWrite(s.MetaPath)
	if err != nil {
		return fmt.Errorf("open meta rw: %w", err)
	}
	defer db.Close()

	hash := persistHash(raw)
	now := time.Now()

	var xpArg any
	if def.XpPerRank > 0 {
		xpArg = def.XpPerRank
	}
	var bpImgArg, bgImgArg any
	bpPath := strings.TrimSpace(def.BattlePassImage)
	if bpPath == "" {
		bpPath = strings.TrimSpace(def.SummaryImagePath)
	}
	if bpPath != "" {
		bpImgArg = bpPath
	}
	if bg := strings.TrimSpace(def.BackgroundImagePath); bg != "" {
		bgImgArg = bg
	}

	// Invalider les anciennes entrées de ce track (hash différent).
	_, _ = db.Exec(ctx, `
		UPDATE battlepass_track_definitions
		SET is_current = FALSE, last_seen_at = ?
		WHERE reward_track_path = ? AND content_hash <> ? AND is_current = TRUE`,
		now, trackPath, hash)

	// Tenter la mise à jour si le hash existe déjà.
	res, err := db.Exec(ctx, `
		UPDATE battlepass_track_definitions
		SET xp_per_rank = ?, battlepass_image_path = ?, background_image_path = ?,
		    raw_payload_json = ?, last_seen_at = ?, is_current = TRUE
		WHERE reward_track_path = ? AND content_hash = ?`,
		xpArg, bpImgArg, bgImgArg, string(raw), now, trackPath, hash)
	if err != nil {
		return fmt.Errorf("update battlepass_track_definitions: %w", err)
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return nil
	}

	// Aucune ligne mise à jour → insérer.
	_, err = db.Exec(ctx, `
		INSERT INTO battlepass_track_definitions
			(reward_track_path, content_hash, xp_per_rank,
			 battlepass_image_path, background_image_path,
			 raw_payload_json, is_current, first_seen_at, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?, TRUE, ?, ?)`,
		trackPath, hash, xpArg, bpImgArg, bgImgArg, string(raw), now, now)
	if err != nil {
		return fmt.Errorf("insert battlepass_track_definitions: %w", err)
	}
	return nil
}
