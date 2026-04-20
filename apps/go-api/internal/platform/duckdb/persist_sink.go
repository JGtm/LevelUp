// Package duckdb — persist_sink.go : écriture fire-and-forget battlepass / challenges.
//
// Le PersistSink ouvre des connexions read-write vers metadata.duckdb et
// stats.duckdb du joueur pour persister les données Waypoint reçues lors des
// appels live API.  Les goroutines sont détachées (fire-and-forget) : un échec
// de persistance ne fait jamais échouer la réponse HTTP.
//
// Connexions :
//   - metadata.duckdb : battlepass_track_definitions + waypoint_assets_raw
//   - stats.duckdb    : challenge_snapshots (append-only)
package duckdb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// PersistSink centralise les écritures battlepass/challenges (fire-and-forget).
// Les connexions RW sont maintenues en cache via openDBs pour ne pas ré-ouvrir
// à chaque appel (coût d'ouverture DuckDB mutualisé).
type PersistSink struct {
	MetaPath   string // chemin vers metadata.duckdb
	PlayerPath string // chemin vers stats.duckdb du joueur
	XUID       string // xuid du joueur authentifié
}

// NewPersistSink crée un PersistSink pour un joueur donné.
func NewPersistSink(metaPath, playerPath, xuid string) *PersistSink {
	return &PersistSink{
		MetaPath:   metaPath,
		PlayerPath: playerPath,
		XUID:       xuid,
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

// PersistBattlePass lance une goroutine fire-and-forget pour sauvegarder les
// données BP dans battlepass_track_definitions et waypoint_assets_raw.
func (s *PersistSink) PersistBattlePass(trackPath string, rawBody []byte) {
	if s.MetaPath == "" || trackPath == "" || len(rawBody) == 0 {
		return
	}
	go func() {
		ctx := context.Background()
		if err := s.writeBattlePass(ctx, trackPath, rawBody); err != nil {
			slog.Warn("persist_sink: battlepass write failed",
				"xuid", s.XUID, "track", trackPath, "err", err)
		}
	}()
}

// writeBattlePass effectue les UPSERTs dans metadata.duckdb.
func (s *PersistSink) writeBattlePass(ctx context.Context, trackPath string, body []byte) error {
	db, err := OpenReadWrite(s.MetaPath)
	if err != nil {
		return fmt.Errorf("open meta rw: %w", err)
	}

	hash := persistHash(body)
	now := time.Now()

	// 1. Sauvegarder le blob brut dans waypoint_assets_raw.
	if err := upsertWaypointAsset(ctx, db,
		"halo_infinite",
		s.XUID+"/battlepass_operations",
		"battlepass_operation",
		hash,
		string(body),
		now,
	); err != nil {
		// Non-fatal : on continue vers battlepass_track_definitions.
		slog.Warn("persist_sink: waypoint_assets_raw BP upsert failed",
			"xuid", s.XUID, "err", err)
	}

	// 2. Persister la définition de track active.
	_, err = db.Exec(ctx, `
		INSERT INTO battlepass_track_definitions
			(reward_track_path, content_hash, raw_payload_json,
			 is_current, first_seen_at, last_seen_at)
		VALUES (?, ?, ?, TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (reward_track_path, content_hash) DO UPDATE SET
			last_seen_at = CURRENT_TIMESTAMP,
			is_current   = TRUE`,
		trackPath, hash, string(body),
	)
	if err != nil {
		return fmt.Errorf("battlepass_track_definitions upsert: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Challenges
// ---------------------------------------------------------------------------

// PersistChallenges lance une goroutine fire-and-forget pour sauvegarder les
// défis dans waypoint_assets_raw (metadata) et challenge_snapshots (player).
func (s *PersistSink) PersistChallenges(rawBody []byte) {
	if len(rawBody) == 0 {
		return
	}
	go func() {
		ctx := context.Background()
		if err := s.writeChallenges(ctx, rawBody); err != nil {
			slog.Warn("persist_sink: challenges write failed",
				"xuid", s.XUID, "err", err)
		}
	}()
}

// deckChallengeRaw est le struct de parsing best-effort d'un challenge depuis /decks.
// Les champs sont lenients : si un champ est absent, il reste à zéro/vide.
type deckChallengeRaw struct {
	TrackingId      string `json:"TrackingId"`
	XPReward        int    `json:"XPReward"`
	SecXPReward     int    `json:"SecondaryXpReward"`
	Threshold       int    `json:"Threshold"`
	CurrentProgress int    `json:"CurrentProgress"`
	IsCompleted     bool   `json:"IsCompleted"`
	CanReroll       bool   `json:"CanReroll"`
	Expiration      struct {
		ISO8601Date string `json:"ISO8601Date"`
	} `json:"Expiration"`
}

// writeChallenges effectue les écritures dans metadata.duckdb et stats.duckdb.
func (s *PersistSink) writeChallenges(ctx context.Context, body []byte) error {
	hash := persistHash(body)
	now := time.Now()

	// Structure /decks telle que parsée par le provider.
	var raw struct {
		AssignedDecks []struct {
			Expiration struct {
				ISO8601Date string `json:"ISO8601Date"`
			} `json:"Expiration"`
			ActiveChallenges    []json.RawMessage `json:"ActiveChallenges"`
			CompletedChallenges []json.RawMessage `json:"CompletedChallenges"`
		} `json:"AssignedDecks"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return fmt.Errorf("parse challenges body: %w", err)
	}

	// 1. Sauvegarder le blob brut dans waypoint_assets_raw (metadata.duckdb).
	if s.MetaPath != "" {
		db, err := OpenReadWrite(s.MetaPath)
		if err != nil {
			slog.Warn("persist_sink: open meta rw for challenges failed", "err", err)
		} else if err := upsertWaypointAsset(ctx, db,
			"halo_infinite",
			s.XUID+"/challenge_deck",
			"challenge_deck",
			hash,
			string(body),
			now,
		); err != nil {
			slog.Warn("persist_sink: waypoint_assets_raw challenges upsert failed",
				"xuid", s.XUID, "err", err)
		}
	}

	// 2. Persister les snapshots dans challenge_snapshots (stats.duckdb).
	if s.PlayerPath == "" || s.XUID == "" {
		return nil
	}

	pdb, err := OpenReadWrite(s.PlayerPath)
	if err != nil {
		return fmt.Errorf("open player rw: %w", err)
	}

	for _, deck := range raw.AssignedDecks {
		deckExpiry := deck.Expiration.ISO8601Date
		for _, rawCh := range deck.ActiveChallenges {
			if err := s.insertSnapshot(ctx, pdb, rawCh, "Active", deckExpiry, now); err != nil {
				slog.Warn("persist_sink: snapshot insert failed",
					"status", "Active", "xuid", s.XUID, "err", err)
			}
		}
		for _, rawCh := range deck.CompletedChallenges {
			if err := s.insertSnapshot(ctx, pdb, rawCh, "Completed", deckExpiry, now); err != nil {
				slog.Warn("persist_sink: snapshot insert failed",
					"status", "Completed", "xuid", s.XUID, "err", err)
			}
		}
	}
	return nil
}

// insertSnapshot insère un snapshot de défi si l'état a changé depuis le dernier
// enregistrement (déduplication par state_hash sur les dernières 24h).
// Retourne nil sans insérer si le challenge n'a pas de TrackingId.
func (s *PersistSink) insertSnapshot(
	ctx context.Context,
	db *DB,
	rawCh json.RawMessage,
	status, deckExpiry string,
	at time.Time,
) error {
	var ch deckChallengeRaw
	if err := json.Unmarshal(rawCh, &ch); err != nil {
		return nil // skip malformed
	}
	if ch.TrackingId == "" {
		return nil // pas d'identifiant stable → skip
	}

	chPath := "Challenges/Tracking/" + ch.TrackingId
	stateHash := persistHash(rawCh)

	// Déduplication : ne pas insérer si un snapshot identique existe dans les 24h.
	var existing int
	err := db.QueryRow(ctx, `
		SELECT COUNT(*) FROM challenge_snapshots
		WHERE xuid = ? AND challenge_path = ? AND state_hash = ?
		  AND snapshot_at > CURRENT_TIMESTAMP - INTERVAL 1 DAY`,
		s.XUID, chPath, stateHash,
	).Scan(&existing)
	if err == nil && existing > 0 {
		return nil // état inchangé, pas besoin d'insérer
	}

	// Choix de l'expiration : priorité au champ du challenge, fallback sur le deck.
	expiry := deckExpiry
	if ch.Expiration.ISO8601Date != "" {
		expiry = ch.Expiration.ISO8601Date
	}

	var expiresAt interface{}
	if expiry != "" {
		expiresAt = expiry
	}

	_, err = db.Exec(ctx, `
		INSERT INTO challenge_snapshots
			(snapshot_at, xuid, challenge_path, challenge_id,
			 status, progress_current, progress_target, xp_reward,
			 can_reroll, expires_at, state_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		at, s.XUID, chPath, ch.TrackingId,
		status, ch.CurrentProgress, ch.Threshold, ch.XPReward,
		ch.CanReroll, expiresAt, stateHash,
	)
	return err
}

// ---------------------------------------------------------------------------
// Helper commun
// ---------------------------------------------------------------------------

// upsertWaypointAsset écrit ou met à jour un enregistrement dans waypoint_assets_raw.
func upsertWaypointAsset(
	ctx context.Context,
	db *DB,
	titleID, assetID, assetType, contentHash, rawJSON string,
	fetchedAt time.Time,
) error {
	_, err := db.Exec(ctx, `
		INSERT INTO waypoint_assets_raw
			(title_id, asset_id, asset_type, version_id,
			 name, description, raw_json, fetched_at, content_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (title_id, asset_id, version_id) DO UPDATE SET
			raw_json     = excluded.raw_json,
			fetched_at   = excluded.fetched_at,
			content_hash = excluded.content_hash`,
		titleID, assetID, assetType, contentHash,
		"", "", rawJSON, fetchedAt, contentHash,
	)
	if err != nil {
		return fmt.Errorf("upsertWaypointAsset %s: %w", assetID, err)
	}
	return nil
}
