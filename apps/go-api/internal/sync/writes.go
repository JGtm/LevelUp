// Package sync — writes.go : helpers d'insertion/upsert dans DuckDB.
//
// Portage de SharedWritesMixin + EnrichedWritesMixin (Python).
// Toutes les fonctions prennent un *sql.DB et travaillent en-dehors de toute
// transaction — la gestion du commit est faite par l'engine.
package sync

import (
	"database/sql"
	"fmt"
	"time"
)

// ──────────────────────────────────────────────────────────────────────────────
// Shared DB writes
// ──────────────────────────────────────────────────────────────────────────────

// InsertRegistryIfNotExists insère un match dans match_registry (INSERT OR IGNORE).
// Portage de _insert_shared_registry() (Python _shared_writes.py).
func InsertRegistryIfNotExists(db *sql.DB, row MatchRegistryRow) error {
	now := time.Now().UTC()
	_, err := db.Exec(`
		INSERT OR IGNORE INTO match_registry (
			match_id, start_time, end_time,
			playlist_id, playlist_name,
			map_id, map_name,
			pair_id, pair_name,
			game_variant_id, game_variant_name,
			mode_category, is_ranked, is_firefight,
			duration_seconds, playable_duration_seconds,
			real_start_time, team_0_score, team_1_score,
			first_sync_by, first_sync_at, last_updated_at,
			created_at, updated_at
		) VALUES (
			?, ?, ?,
			?, ?,
			?, ?,
			?, ?,
			?, ?,
			?, ?, ?,
			?, ?,
			?, ?, ?,
			?, ?, ?,
			?, ?
		)`,
		row.MatchID, row.StartTime, row.EndTime,
		row.PlaylistID, row.PlaylistName,
		row.MapID, row.MapName,
		row.PairID, row.PairName,
		row.GameVariantID, row.GameVariantName,
		row.ModeCategory, row.IsRanked, row.IsFirefight,
		row.DurationSeconds, row.PlayableDurationSeconds,
		row.RealStartTime, row.Team0Score, row.Team1Score,
		row.FirstSyncBy, now, now,
		now, now,
	)
	if err != nil {
		return fmt.Errorf("InsertRegistry(%s): %w", row.MatchID, err)
	}
	return nil
}

// InsertParticipants insère les participants d'un match (INSERT OR IGNORE).
// Portage de batch_upsert_rows sur match_participants (Python _shared_writes.py).
func InsertParticipants(db *sql.DB, rows []ParticipantRow) error {
	if len(rows) == 0 {
		return nil
	}
	now := time.Now().UTC()
	for _, row := range rows {
		_, err := db.Exec(`
			INSERT OR IGNORE INTO match_participants (
				match_id, xuid, gamertag,
				team_id, outcome, rank, score,
				kills, deaths, assists,
				shots_fired, shots_hit,
				damage_dealt, damage_taken,
				kda, accuracy, personal_score,
				time_played_seconds, avg_life_seconds,
				kills_expected, deaths_expected, kills_stddev,
				team_mmr, enemy_mmr,
				created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			row.MatchID, row.XUID, row.Gamertag,
			row.TeamID, row.Outcome, row.Rank, row.Score,
			row.Kills, row.Deaths, row.Assists,
			row.ShotsFired, row.ShotsHit,
			row.DamageDealt, row.DamageTaken,
			row.KDA, row.Accuracy, row.PersonalScore,
			row.TimePlayedSeconds, row.AvgLifeSeconds,
			row.KillsExpected, row.DeathsExpected, row.KillsStddev,
			row.TeamMMR, row.EnemyMMR,
			now,
		)
		if err != nil {
			return fmt.Errorf("InsertParticipants(%s/%s): %w", row.MatchID, row.XUID, err)
		}
	}
	return nil
}

// InsertMedals insère les médailles d'un match (INSERT OR IGNORE).
// Portage de batch_upsert_rows sur medals_earned (Python _shared_writes.py).
func InsertMedals(db *sql.DB, rows []MedalRow) error {
	if len(rows) == 0 {
		return nil
	}
	now := time.Now().UTC()
	for _, row := range rows {
		_, err := db.Exec(`
			INSERT OR IGNORE INTO medals_earned (match_id, xuid, medal_name_id, count, created_at)
			VALUES (?, ?, ?, ?, ?)`,
			row.MatchID, row.XUID, row.MedalNameID, row.Count, now,
		)
		if err != nil {
			return fmt.Errorf("InsertMedals(%s/%s/%d): %w", row.MatchID, row.XUID, row.MedalNameID, err)
		}
	}
	return nil
}

// UpsertXUIDAlias met à jour xuid_aliases avec le gamertag le plus récent.
// Portage de _upsert_xuid_alias() (Python _shared_writes.py).
func UpsertXUIDAlias(db *sql.DB, xuid, gamertag string) error {
	if xuid == "" || gamertag == "" {
		return nil
	}
	now := time.Now().UTC()
	_, err := db.Exec(`
		INSERT INTO xuid_aliases (xuid, gamertag, last_seen, source, updated_at)
		VALUES (?, ?, ?, 'sync', ?)
		ON CONFLICT (xuid) DO UPDATE SET
			gamertag   = EXCLUDED.gamertag,
			last_seen  = EXCLUDED.last_seen,
			updated_at = EXCLUDED.updated_at`,
		xuid, gamertag, now, now,
	)
	if err != nil {
		return fmt.Errorf("UpsertXUIDAlias(%s): %w", xuid, err)
	}
	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Player DB writes
// ──────────────────────────────────────────────────────────────────────────────

// UpsertPlayerEnrichment insère/met à jour une ligne dans player_match_enrichment.
// Portage de _insert_enrichment_row() (Python _engine_writes.py).
func UpsertPlayerEnrichment(db *sql.DB, matchID, teammatesSig string) error {
	now := time.Now().UTC()
	_, err := db.Exec(`
		INSERT INTO player_match_enrichment
			(match_id, teammates_signature, created_at, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (match_id) DO UPDATE SET
			teammates_signature = COALESCE(EXCLUDED.teammates_signature, player_match_enrichment.teammates_signature),
			updated_at          = EXCLUDED.updated_at`,
		matchID, nullStr(teammatesSig), now, now,
	)
	if err != nil {
		return fmt.Errorf("UpsertPlayerEnrichment(%s): %w", matchID, err)
	}
	return nil
}

// SetSyncMeta met à jour une clé dans sync_meta.
// Portage de _update_sync_meta() (Python engine.py).
func SetSyncMeta(db *sql.DB, key, value string) error {
	now := time.Now().UTC()
	_, err := db.Exec(`
		INSERT INTO sync_meta (key, value, updated_at) VALUES (?, ?, ?)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at`,
		key, value, now,
	)
	if err != nil {
		return fmt.Errorf("SetSyncMeta(%s): %w", key, err)
	}
	return nil
}

// nullStr retourne nil si la chaîne est vide (pour les colonnes SQL NULL).
func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ─────────────────────────────────────────────────────────────────────────────
// Weapon kills (Sprint 41 T2)
// ─────────────────────────────────────────────────────────────────────────────

// InsertWeaponKills remplace les entrées weapon_kills existantes pour (matchID, xuid)
// par les nouvelles attributions. Opération atomique : DELETE + INSERT batch.
func InsertWeaponKills(db *sql.DB, matchID, xuid string, attrs []WeaponKillRow) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("InsertWeaponKills begin: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.Exec(`DELETE FROM weapon_kills WHERE match_id = ? AND xuid = ?`, matchID, xuid); err != nil {
		return fmt.Errorf("InsertWeaponKills delete: %w", err)
	}

	for _, r := range attrs {
		if _, err := tx.Exec(`
			INSERT INTO weapon_kills (
				match_id, xuid, time_ms, weapon_id, reconciled_as,
				delta_ms, confidence, attribution_path,
				swap_detected, delayed_damage, player_index
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			matchID, xuid, r.TimeMS, r.WeaponID, r.ReconciledAs,
			r.DeltaMS, r.Confidence, r.AttributionPath,
			r.SwapDetected, r.DelayedDamage, r.PlayerIndex,
		); err != nil {
			return fmt.Errorf("InsertWeaponKills insert: %w", err)
		}
	}
	return tx.Commit()
}

// WeaponKillRow est la représentation d'une ligne weapon_kills.
type WeaponKillRow struct {
	TimeMS          int
	WeaponID        *uint64
	ReconciledAs    *uint64
	DeltaMS         *int
	Confidence      string
	AttributionPath string
	SwapDetected    bool
	DelayedDamage   bool
	PlayerIndex     *int
}

// MarkWeaponKillsDone met à jour le bit MBitWeaponKills ou MBitWeaponKillsNoFilm
// dans match_registry.backfill_completed.
func MarkWeaponKillsDone(db *sql.DB, matchID string, noFilm bool) error {
	bit := MBitWeaponKills
	if noFilm {
		bit = MBitWeaponKillsNoFilm
	}
	_, err := db.Exec(`
		UPDATE match_registry
		SET backfill_completed = COALESCE(backfill_completed, 0) | ?
		WHERE match_id = ?`, bit, matchID)
	if err != nil {
		return fmt.Errorf("MarkWeaponKillsDone(%s): %w", matchID, err)
	}
	return nil
}
