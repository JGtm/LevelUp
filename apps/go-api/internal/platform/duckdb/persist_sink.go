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
		  AND snapshot_at > CURRENT_TIMESTAMP - INTERVAL 1 DAY`,
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

// ---------------------------------------------------------------------------
// Item Definitions (battlepass_item_definitions + battlepass_item_translations)
// ---------------------------------------------------------------------------

// itemDefCommonData est le sous-arbre CommonData d'un item GameCMS.
// GameCMS utilise "Type" (pas "ItemType") pour la rareté fonctionnelle — les deux
// champs sont lus pour couvrir toutes les versions de l'API.
type itemDefCommonData struct {
	Title       any    `json:"Title"`
	Description any    `json:"Description"`
	Quality     string `json:"Quality"`
	Type        string `json:"Type"`
	ItemType    string `json:"ItemType"`
	DisplayPath struct {
		Media struct {
			MediaURL struct {
				Path string `json:"Path"`
			} `json:"MediaUrl"`
		} `json:"Media"`
	} `json:"DisplayPath"`
}

type itemDefRaw struct {
	CommonData itemDefCommonData `json:"CommonData"`
}

// itemDefLocalizedText extrait le texte localisé depuis un champ Halo polymorphe.
// GameCMS retourne soit une string, soit {"value":"…","status":"Resolved"}, soit
// {"translations":{"fr-FR":"…","en-US":"…"}}.
func itemDefLocalizedText(v any, preferLang string) string {
	if v == nil {
		return ""
	}
	switch typed := v.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		if trans, ok := typed["translations"].(map[string]any); ok {
			for _, lang := range []string{preferLang, LangCodeFR, LangCodeEN, "en"} {
				if s, ok := trans[lang].(string); ok && strings.TrimSpace(s) != "" {
					return strings.TrimSpace(s)
				}
			}
		}
		if s, ok := typed["value"].(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

// UpsertItemDefinition implémente halo.ItemDefinitionPersister.
// Persiste la définition d'un item BP dans battlepass_item_definitions et
// battlepass_item_translations (fr-FR + en-US) de metadata.duckdb.
// Utilise UPDATE-first + INSERT-if-zero (idiome DuckDB) pour garantir l'idempotence.
func (s *PersistSink) UpsertItemDefinition(ctx context.Context, itemPath string, raw []byte) error {
	var def itemDefRaw
	_ = json.Unmarshal(raw, &def) // best-effort

	cd := def.CommonData
	itemType := strings.TrimSpace(cd.Type)
	if itemType == "" {
		itemType = strings.TrimSpace(cd.ItemType)
	}
	displayPath := strings.TrimSpace(cd.DisplayPath.Media.MediaURL.Path)

	var qualityArg, itemTypeArg, displayPathArg any
	if q := strings.TrimSpace(cd.Quality); q != "" {
		qualityArg = q
	}
	if itemType != "" {
		itemTypeArg = itemType
	}
	if displayPath != "" {
		displayPathArg = displayPath
	}

	relMeta, err := dblease.AcquireLease(s.MetaPath, dblease.MetadataLeaseTimeout)
	if err != nil {
		return fmt.Errorf("UpsertItemDefinition lease meta: %w", err)
	}
	defer relMeta()

	db, err := OpenReadWrite(s.MetaPath)
	if err != nil {
		return fmt.Errorf("open meta rw: %w", err)
	}
	defer db.Close()

	hash := persistHash(raw)
	now := time.Now()

	// Invalider les anciennes entrées de cet item (hash différent).
	_, _ = db.Exec(ctx, `
		UPDATE battlepass_item_definitions
		SET is_current = FALSE, last_seen_at = ?
		WHERE inventory_item_path = ? AND content_hash <> ? AND is_current = TRUE`,
		now, itemPath, hash)

	res, err := db.Exec(ctx, `
		UPDATE battlepass_item_definitions
		SET quality = ?, item_type = ?, display_path = ?,
		    raw_payload_json = ?, last_seen_at = ?, is_current = TRUE
		WHERE inventory_item_path = ? AND content_hash = ?`,
		qualityArg, itemTypeArg, displayPathArg, string(raw), now, itemPath, hash)
	if err != nil {
		return fmt.Errorf("update battlepass_item_definitions: %w", err)
	}
	if n, _ := res.RowsAffected(); n > 0 {
		slog.DebugContext(ctx, "persist_sink: bp item definition updated", "path", itemPath, "hash", hash)
		return s.upsertItemTranslations(ctx, db, itemPath, hash, cd, now)
	}

	_, err = db.Exec(ctx, `
		INSERT INTO battlepass_item_definitions
			(inventory_item_path, content_hash, quality, item_type,
			 display_path, raw_payload_json, is_current, first_seen_at, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?, TRUE, ?, ?)`,
		itemPath, hash, qualityArg, itemTypeArg, displayPathArg, string(raw), now, now)
	if err != nil {
		return fmt.Errorf("insert battlepass_item_definitions: %w", err)
	}
	slog.DebugContext(ctx, "persist_sink: bp item definition inserted", "path", itemPath, "hash", hash)
	return s.upsertItemTranslations(ctx, db, itemPath, hash, cd, now)
}

// upsertItemTranslations persiste les traductions fr-FR et en-US d'un item.
func (s *PersistSink) upsertItemTranslations(
	ctx context.Context,
	db *DB,
	itemPath, hash string,
	cd itemDefCommonData,
	now time.Time,
) error {
	type langEntry struct {
		lang        string
		title       string
		description string
	}

	entries := []langEntry{
		{
			lang:        LangCodeFR,
			title:       itemDefLocalizedText(cd.Title, LangCodeFR),
			description: itemDefLocalizedText(cd.Description, LangCodeFR),
		},
		{
			lang:        LangCodeEN,
			title:       itemDefLocalizedText(cd.Title, LangCodeEN),
			description: itemDefLocalizedText(cd.Description, LangCodeEN),
		},
	}

	for _, e := range entries {
		if e.title == "" && e.description == "" {
			continue
		}
		var titleArg, descArg any
		if e.title != "" {
			titleArg = e.title
		}
		if e.description != "" {
			descArg = e.description
		}
		res, err := db.Exec(ctx, `
			UPDATE battlepass_item_translations
			SET title = ?, description = ?, last_seen_at = ?
			WHERE inventory_item_path = ? AND content_hash = ? AND lang = ?`,
			titleArg, descArg, now, itemPath, hash, e.lang)
		if err != nil {
			return fmt.Errorf("update battlepass_item_translations %s: %w", e.lang, err)
		}
		if n, _ := res.RowsAffected(); n > 0 {
			continue
		}
		_, err = db.Exec(ctx, `
			INSERT INTO battlepass_item_translations
				(inventory_item_path, content_hash, lang, title, description,
				 first_seen_at, last_seen_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			itemPath, hash, e.lang, titleArg, descArg, now, now)
		if err != nil {
			return fmt.Errorf("insert battlepass_item_translations %s: %w", e.lang, err)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Challenges
// ---------------------------------------------------------------------------

// PersistChallengesSync persiste les défis de manière SYNCHRONE dans
// waypoint_assets_raw (metadata) et challenge_snapshots (player). L'écriture est
// liée au ctx appelant (HTTP / ticker live_refresh) au lieu d'une goroutine
// détachée en context.Background() — garantit qu'elle se termine avant le
// shutdown (lifecycle, W6 revue 2026-06-01).
func (s *PersistSink) PersistChallengesSync(ctx context.Context, rawBody []byte) error {
	if len(rawBody) == 0 {
		return nil
	}
	return s.writeChallenges(ctx, rawBody)
}

// deckChallengeRaw est le struct de parsing best-effort d'un challenge depuis /decks.
// Les champs sont lenients : si un champ est absent, il reste à zéro/vide.
type deckChallengeRaw struct {
	TrackingID      string `json:"TrackingId"`
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
		relMeta, leaseErr := dblease.AcquireLease(s.MetaPath, dblease.MetadataLeaseTimeout)
		if leaseErr != nil {
			slog.Warn("persist_sink: meta lease for challenges failed", "err", leaseErr)
		} else {
			defer relMeta()
			db, err := OpenReadWrite(s.MetaPath)
			if err != nil {
				slog.Warn("persist_sink: open meta rw for challenges failed", "err", err)
			} else {
				defer db.Close()
				if err := upsertWaypointAsset(ctx, db,
					s.TitleSlug,
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
		}
	}

	// 2. Persister les snapshots dans challenge_snapshots (stats.duckdb).
	if s.PlayerPath == "" || s.XUID == "" {
		return nil
	}

	relPlayer, err := dblease.AcquireLease(s.PlayerPath, dblease.PlayerLeaseTimeout)
	if err != nil {
		return fmt.Errorf("writeChallenges lease player: %w", err)
	}
	defer relPlayer()

	pdb, err := OpenReadWrite(s.PlayerPath)
	if err != nil {
		return fmt.Errorf("open player rw: %w", err)
	}
	defer pdb.Close()

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
// Retourne nil sans insérer si le challenge n'a pas de TrackingID.
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
	if ch.TrackingID == "" {
		return nil // pas d'identifiant stable → skip
	}

	chPath := "Challenges/Tracking/" + ch.TrackingID
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
		at, s.XUID, chPath, ch.TrackingID,
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
	// SELECT-then-write anti-ART (cf. (*DB).UpsertNoConflict) : pas d'ON CONFLICT
	// DO UPDATE sur metadata.duckdb, qui FATAL-invalide le handle partagé.
	// version_id = contentHash ici (clé naturelle de la version).
	err := db.UpsertNoConflict(ctx,
		`SELECT 1 FROM waypoint_assets_raw WHERE title_id = ? AND asset_id = ? AND version_id = ?`,
		[]any{titleID, assetID, contentHash},
		`UPDATE waypoint_assets_raw SET raw_json = ?, fetched_at = ?, content_hash = ?
		 WHERE title_id = ? AND asset_id = ? AND version_id = ?`,
		[]any{rawJSON, fetchedAt, contentHash, titleID, assetID, contentHash},
		`INSERT INTO waypoint_assets_raw
			(title_id, asset_id, asset_type, version_id,
			 name, description, raw_json, fetched_at, content_hash)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		[]any{titleID, assetID, assetType, contentHash,
			"", "", rawJSON, fetchedAt, contentHash},
	)
	if err != nil {
		return fmt.Errorf("upsertWaypointAsset %s: %w", assetID, err)
	}
	return nil
}
