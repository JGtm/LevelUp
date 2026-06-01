// Package sync — writes.go : helpers d'insertion/upsert dans DuckDB.
//
// Portage de SharedWritesMixin + EnrichedWritesMixin (Python).
// Toutes les fonctions prennent un *sql.DB et travaillent en-dehors de toute
// transaction — la gestion du commit est faite par l'engine.
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
)

// ──────────────────────────────────────────────────────────────────────────────
// Shared DB writes
// ──────────────────────────────────────────────────────────────────────────────

// InsertRegistryIfNotExists insère un match dans match_registry (INSERT OR IGNORE).
// Portage de _insert_shared_registry() (Python _shared_writes.py).
//
// start_time_utc / end_time_utc (TIMESTAMPTZ) sont remplis explicitement avec
// row.StartTime / row.EndTime convertis en UTC : ces colonnes sont la source
// de vérité non-ambiguë pour les requêtes d'affichage. start_time / end_time
// (TIMESTAMP naïf) restent écrits par compat ; leur convention bytes dépend
// de la session TZ DuckDB et n'est plus utilisée pour exposer la date au front.
func InsertRegistryIfNotExists(ctx context.Context, db *sql.DB, row MatchRegistryRow) error {
	now := time.Now().UTC()
	startUTC := row.StartTime.UTC()
	var endUTC interface{}
	if row.EndTime != nil {
		t := row.EndTime.UTC()
		endUTC = t
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO match_registry (
			match_id, start_time, end_time, start_time_utc, end_time_utc,
			playlist_id, playlist_name, playlist_version_id,
			map_id, map_name, map_version_id,
			pair_id, pair_name, pair_version_id,
			game_variant_id, game_variant_name, game_variant_version_id,
			mode_category, is_ranked, is_firefight, season_id,
			duration_seconds, playable_duration_seconds,
			real_start_time, team_0_score, team_1_score,
			team_0_ps_score, team_1_ps_score,
			first_sync_by, first_sync_at, last_updated_at,
			created_at, updated_at
		) VALUES (
			?, ?, ?, ?, ?,
			?, ?, ?,
			?, ?, ?,
			?, ?, ?,
			?, ?, ?,
			?, ?, ?, ?,
			?, ?,
			?, ?, ?,
			?, ?,
			?, ?, ?,
			?, ?
		)
		ON CONFLICT (match_id) DO UPDATE SET
			team_0_ps_score = COALESCE(EXCLUDED.team_0_ps_score, match_registry.team_0_ps_score),
			team_1_ps_score = COALESCE(EXCLUDED.team_1_ps_score, match_registry.team_1_ps_score),
			season_id       = COALESCE(EXCLUDED.season_id, match_registry.season_id),
			last_updated_at = EXCLUDED.last_updated_at`,
		row.MatchID, row.StartTime, row.EndTime, startUTC, endUTC,
		row.PlaylistID, row.PlaylistName, row.PlaylistVersionID,
		row.MapID, row.MapName, row.MapVersionID,
		row.PairID, row.PairName, row.PairVersionID,
		row.GameVariantID, row.GameVariantName, row.GameVariantVersionID,
		row.ModeCategory, row.IsRanked, row.IsFirefight, row.SeasonID,
		row.DurationSeconds, row.PlayableDurationSeconds,
		row.RealStartTime, row.Team0Score, row.Team1Score,
		row.Team0PSScore, row.Team1PSScore,
		row.FirstSyncBy, now, now,
		now, now,
	)
	if err != nil {
		return fmt.Errorf("InsertRegistry(%s): %w", row.MatchID, err)
	}
	return nil
}

// InsertParticipants UPSERT les participants d'un match.
// Portage de batch_upsert_rows sur match_participants (Python _shared_writes.py).
//
// Sur conflit (match_id, xuid) : COALESCE(EXCLUDED, existing) — les valeurs
// non-nulles entrantes écrasent l'existant ; les NULL entrants préservent
// l'existant. Cela permet de re-syncer pour combler des champs vides
// (typiquement team_mmr/enemy_mmr quand le skill endpoint a échoué la
// première fois) sans détruire les données déjà persistées.
func InsertParticipants(ctx context.Context, db *sql.DB, rows []ParticipantRow) error {
	if len(rows) == 0 {
		return nil
	}
	now := time.Now().UTC()
	for _, row := range rows {
		if err := insertParticipantRow(ctx, db, row, now); err != nil {
			observability.IncCounter("upsert_match_participants_total_error")
			return err
		}
		observability.IncCounter("upsert_match_participants_total_ok")
	}
	return nil
}

// insertParticipantRow exécute l'UPSERT SQL d'un seul ParticipantRow. Extrait
// de InsertParticipants pour être appelable via singleflight (Phase 1.3).
func insertParticipantRow(ctx context.Context, db *sql.DB, row ParticipantRow, now time.Time) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO match_participants (
			match_id, xuid, gamertag,
			team_id, outcome, rank, score,
			kills, deaths, assists,
			shots_fired, shots_hit,
			damage_dealt, damage_taken,
			kda, accuracy, personal_score,
			time_played_seconds, avg_life_seconds,
			kills_expected, deaths_expected, kills_stddev, deaths_stddev,
			team_mmr, enemy_mmr,
			headshot_kills,
			max_killing_spree, grenade_kills, melee_kills, power_weapon_kills,
			present_at_beginning, present_at_completion, joined_in_progress, left_in_progress,
			created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (match_id, xuid) DO UPDATE SET
			gamertag              = COALESCE(EXCLUDED.gamertag,              match_participants.gamertag),
			team_id               = COALESCE(EXCLUDED.team_id,               match_participants.team_id),
			outcome               = COALESCE(EXCLUDED.outcome,               match_participants.outcome),
			rank                  = COALESCE(EXCLUDED.rank,                  match_participants.rank),
			score                 = COALESCE(EXCLUDED.score,                 match_participants.score),
			kills                 = COALESCE(EXCLUDED.kills,                 match_participants.kills),
			deaths                = COALESCE(EXCLUDED.deaths,                match_participants.deaths),
			assists               = COALESCE(EXCLUDED.assists,               match_participants.assists),
			shots_fired           = COALESCE(EXCLUDED.shots_fired,           match_participants.shots_fired),
			shots_hit             = COALESCE(EXCLUDED.shots_hit,             match_participants.shots_hit),
			damage_dealt          = COALESCE(EXCLUDED.damage_dealt,          match_participants.damage_dealt),
			damage_taken          = COALESCE(EXCLUDED.damage_taken,          match_participants.damage_taken),
			kda                   = COALESCE(EXCLUDED.kda,                   match_participants.kda),
			accuracy              = COALESCE(EXCLUDED.accuracy,              match_participants.accuracy),
			personal_score        = COALESCE(EXCLUDED.personal_score,        match_participants.personal_score),
			time_played_seconds   = COALESCE(EXCLUDED.time_played_seconds,   match_participants.time_played_seconds),
			avg_life_seconds      = COALESCE(EXCLUDED.avg_life_seconds,      match_participants.avg_life_seconds),
			kills_expected        = COALESCE(EXCLUDED.kills_expected,        match_participants.kills_expected),
			deaths_expected       = COALESCE(EXCLUDED.deaths_expected,       match_participants.deaths_expected),
			kills_stddev          = COALESCE(EXCLUDED.kills_stddev,          match_participants.kills_stddev),
			deaths_stddev         = COALESCE(EXCLUDED.deaths_stddev,         match_participants.deaths_stddev),
			team_mmr              = COALESCE(EXCLUDED.team_mmr,              match_participants.team_mmr),
			enemy_mmr             = COALESCE(EXCLUDED.enemy_mmr,             match_participants.enemy_mmr),
			headshot_kills        = COALESCE(EXCLUDED.headshot_kills,        match_participants.headshot_kills),
			max_killing_spree     = COALESCE(EXCLUDED.max_killing_spree,     match_participants.max_killing_spree),
			grenade_kills         = COALESCE(EXCLUDED.grenade_kills,         match_participants.grenade_kills),
			melee_kills           = COALESCE(EXCLUDED.melee_kills,           match_participants.melee_kills),
			power_weapon_kills    = COALESCE(EXCLUDED.power_weapon_kills,    match_participants.power_weapon_kills),
			present_at_beginning  = COALESCE(EXCLUDED.present_at_beginning,  match_participants.present_at_beginning),
			present_at_completion = COALESCE(EXCLUDED.present_at_completion, match_participants.present_at_completion),
			joined_in_progress    = COALESCE(EXCLUDED.joined_in_progress,    match_participants.joined_in_progress),
			left_in_progress      = COALESCE(EXCLUDED.left_in_progress,      match_participants.left_in_progress)`,
		row.MatchID, row.XUID, row.Gamertag,
		row.TeamID, row.Outcome, row.Rank, row.Score,
		row.Kills, row.Deaths, row.Assists,
		row.ShotsFired, row.ShotsHit,
		row.DamageDealt, row.DamageTaken,
		row.KDA, row.Accuracy, row.PersonalScore,
		row.TimePlayedSeconds, row.AvgLifeSeconds,
		row.KillsExpected, row.DeathsExpected, row.KillsStddev, row.DeathsStddev,
		row.TeamMMR, row.EnemyMMR,
		row.HeadshotKills,
		row.MaxKillingSpree, row.GrenadeKills, row.MeleeKills, row.PowerWeaponKills,
		row.PresentAtBeginning, row.PresentAtCompletion, row.JoinedInProgress, row.LeftInProgress,
		now,
	)
	if err != nil {
		return fmt.Errorf("InsertParticipants(%s/%s): %w", row.MatchID, row.XUID, err)
	}
	return nil
}

// InsertMedals insère les médailles d'un match (INSERT OR IGNORE).
// Portage de batch_upsert_rows sur medals_earned (Python _shared_writes.py).
func InsertMedals(ctx context.Context, db *sql.DB, rows []MedalRow) error {
	if len(rows) == 0 {
		return nil
	}
	now := time.Now().UTC()
	for _, row := range rows {
		_, err := db.ExecContext(ctx, `
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
// Pour les bots, le nom d'affichage est toujours résolu via BotDisplayName
// ("bid(3.0)" → "343 Bot 3") indépendamment du gamertag brut de l'API.
func UpsertXUIDAlias(ctx context.Context, db *sql.DB, xuid, gamertag string) error {
	if xuid == "" || gamertag == "" {
		return nil
	}
	if analysis.IsBot(xuid) {
		gamertag = analysis.BotDisplayName(xuid)
	}
	now := time.Now().UTC()
	_, err := db.ExecContext(ctx, `
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
func UpsertPlayerEnrichment(ctx context.Context, db *sql.DB, matchID, teammatesSig string) error {
	now := time.Now().UTC()
	_, err := db.ExecContext(ctx, `
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
func SetSyncMeta(ctx context.Context, db *sql.DB, key, value string) error {
	now := time.Now().UTC()
	_, err := db.ExecContext(ctx, `
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

// WriteSessionAssignments écrit session_id et session_label dans
// player_match_enrichment via le batch path (Phase 3 du refactor ART :
// le path legacy row-by-row UPDATE a été supprimé car à risque ART).
// Retourne le nombre de lignes affectées.
func WriteSessionAssignments(ctx context.Context, db *sql.DB, assignments []domain.SessionAssignment) (int, error) {
	return writeSessionAssignmentsBatch(ctx, db, assignments, false)
}

// ─────────────────────────────────────────────────────────────────────────────
// Weapon kills (Sprint 41 T2)
// ─────────────────────────────────────────────────────────────────────────────

// InsertWeaponKills remplace les entrées weapon_kills existantes pour (matchID, xuid)
// par les nouvelles attributions. Opération atomique : DELETE + INSERT batch.
//
// weapon_id et reconciled_as sont des UBIGINT (uint64) et sont injectés sous forme
// de string décimale puis castés dans le SQL : le driver duckdb-go rejette
// `database/sql` les uint64 dont le bit 63 est set ("values with high bit set are
// not supported"), or de nombreux IDs filmshell Halo (ex `f408190f42c9679f`)
// dépassent 2^63. Le cast côté DuckDB préserve la valeur unsigned exacte.
func InsertWeaponKills(ctx context.Context, db *sql.DB, matchID, xuid string, attrs []WeaponKillRow) error {
	// Garde-fou anti-perte de données : un appel avec attrs=[] doit être un
	// no-op total. Le DELETE+INSERT-batch a vidé silencieusement les rows
	// existantes pour ~1010 matchs en mai 2026 quand un --weapons --force
	// retombait sur des films expirés (extraction = 0 kills) (cf. thought_log
	// 2026-05-09). On préserve désormais l'existant si l'extraction n'a rien
	// produit — la décision "remplacer" doit être explicite.
	if len(attrs) == 0 {
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("InsertWeaponKills begin: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx, `DELETE FROM weapon_kills WHERE match_id = ? AND xuid = ?`, matchID, xuid); err != nil {
		return fmt.Errorf("InsertWeaponKills delete: %w", err)
	}

	for _, r := range attrs {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO weapon_kills (
				match_id, xuid, time_ms, weapon_id, reconciled_as,
				delta_ms, confidence, attribution_path,
				swap_detected, delayed_damage, player_index
			) VALUES (?, ?, ?, CAST(? AS UBIGINT), CAST(? AS UBIGINT), ?, ?, ?, ?, ?, ?)`,
			matchID, xuid, r.TimeMS, ubigintArg(r.WeaponID), ubigintArg(r.ReconciledAs),
			r.DeltaMS, r.Confidence, r.AttributionPath,
			r.SwapDetected, r.DelayedDamage, r.PlayerIndex,
		); err != nil {
			return fmt.Errorf("InsertWeaponKills insert: %w", err)
		}
	}
	return tx.Commit()
}

// ubigintArg sérialise un *uint64 en string décimale pour CAST(? AS UBIGINT) côté
// DuckDB, ou nil pour un NULL. Workaround driver duckdb-go (cf. InsertWeaponKills).
func ubigintArg(p *uint64) any {
	if p == nil {
		return nil
	}
	return strconv.FormatUint(*p, 10)
}

// WeaponKillRow — alias vers domain.WeaponKillRow.
// La définition canonique vit dans internal/domain/match_rows.go (déplacé
// 2026-05-23 pour casser le cycle d'import sync ⇄ persist).
type WeaponKillRow = domain.WeaponKillRow

// MarkWeaponKillsDone met à jour le bit MBitWeaponKills ou MBitWeaponKillsNoFilm
// dans match_registry.backfill_completed.
func MarkWeaponKillsDone(ctx context.Context, db *sql.DB, matchID string, noFilm bool) error {
	bit := MBitWeaponKills
	if noFilm {
		bit = MBitWeaponKillsNoFilm
	}
	_, err := db.ExecContext(ctx, `
		UPDATE match_registry
		SET backfill_completed = COALESCE(backfill_completed, 0) | ?
		WHERE match_id = ?
	`, bit, matchID)
	if err != nil {
		return fmt.Errorf("MarkWeaponKillsDone(%s): %w", matchID, err)
	}
	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Personal score awards writes
// ──────────────────────────────────────────────────────────────────────────────

// InsertPersonalScoreAwards remplace atomiquement les awards existants pour
// (matchID, xuid) dans player.personal_score_awards. Pattern identique à
// InsertWeaponKills : DELETE puis INSERT batch dans une transaction.
//
// Idempotent : ré-exécuter pour le même (matchID, xuid) écrase les rows
// précédentes, ce qui est attendu pour réparer une extraction défaillante.
func InsertPersonalScoreAwards(ctx context.Context, db *sql.DB, matchID, xuid string, rows []PersonalScoreAwardRow) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("InsertPersonalScoreAwards begin: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx, `DELETE FROM personal_score_awards WHERE match_id = ? AND xuid = ?`, matchID, xuid); err != nil {
		return fmt.Errorf("InsertPersonalScoreAwards delete(%s,%s): %w", matchID, xuid, err)
	}
	for _, r := range rows {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO personal_score_awards
				(match_id, xuid, award_name, award_category, award_count, award_score)
			VALUES (?, ?, ?, ?, ?, ?)`,
			r.MatchID, r.XUID, r.AwardName, r.AwardCategory, r.AwardCount, r.AwardScore,
		); err != nil {
			return fmt.Errorf("InsertPersonalScoreAwards insert(%s,%s): %w", matchID, xuid, err)
		}
	}
	return tx.Commit()
}

// ──────────────────────────────────────────────────────────────────────────────
// Highlight events writes
// ──────────────────────────────────────────────────────────────────────────────

// InsertHighlightEvents insère les événements highlight en lot (INSERT OR IGNORE).
// Retourne le nombre de lignes effectivement insérées.
func InsertHighlightEvents(ctx context.Context, db *sql.DB, matchID string, events []analysis.HighlightEvent) (int, error) {
	if len(events) == 0 {
		return 0, nil
	}
	stmt, err := db.PrepareContext(ctx, `
		INSERT OR IGNORE INTO highlight_events
			(match_id, event_type, time_ms, xuid, type_hint)
		VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, fmt.Errorf("InsertHighlightEvents prepare(%s): %w", matchID, err)
	}
	defer stmt.Close()

	inserted := 0
	for _, ev := range events {
		xuid := strconv.FormatUint(ev.XUID, 10)
		res, execErr := stmt.ExecContext(ctx, matchID, ev.EventType, ev.TimeMS, xuid, ev.TypeHint)
		if execErr != nil {
			return inserted, fmt.Errorf("InsertHighlightEvents exec(%s): %w", matchID, execErr)
		}
		if n, _ := res.RowsAffected(); n > 0 {
			inserted++
		}
	}
	return inserted, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Phase 2 du plan PLAN_BITMASKS_AUDIT_FIX (mai 2026) — Mark* manquantes pour
// que les filtres detection backfill (skill/participants/PVE) reflètent l'état
// réel sur les nouveaux matchs synchronisés via le code Go.
// ──────────────────────────────────────────────────────────────────────────────

// skillBitsCombined regroupe les 4 PBit* positionnés ensemble quand l'API
// GetMatchSkill renvoie des données : team_mmr, enemy_mmr, kills_expected,
// deaths_expected. Ils sont obtenus en bloc, on les marque en bloc.
const skillBitsCombined = PBitTeamMMR | PBitEnemyMMR | PBitKillsExp | PBitDeathsExp

// backfillFlagSkill est la valeur legacy de BackfillFlags["skill"] = 1<<2.
// Stocké dans match_registry.backfill_completed (et non dans
// match_participants.backfill_bits comme les PBit*).
const backfillFlagSkill = 4

// backfillFlagParticipants est la valeur legacy de BackfillFlags["participants"]
// = 1<<9. Stocké dans match_registry.backfill_completed.
const backfillFlagParticipants = 512

// MarkSkillLoaded positionne les bits skill sur le registry et tous les
// participants du match dont le team_mmr est non-NULL. Idempotent (|=).
//
// À appeler depuis insertFetchedMatch APRÈS un InsertParticipants réussi
// quand fm.SkillError est nil ET que MergeSkillIntoParticipants a été appelé
// (= skillData non vide).
func MarkSkillLoaded(ctx context.Context, db *sql.DB, matchID string) error {
	// PBit* sur match_participants — uniquement les rows ayant team_mmr
	// effectivement rempli (sinon mensonge sur un participant skipped).
	if _, err := db.ExecContext(ctx, `
		UPDATE match_participants
		SET backfill_bits = COALESCE(backfill_bits, 0) | ?
		WHERE match_id = ? AND team_mmr IS NOT NULL`, skillBitsCombined, matchID); err != nil {
		return fmt.Errorf("MarkSkillLoaded participants(%s): %w", matchID, err)
	}
	// BackfillFlags["skill"] sur match_registry.
	if _, err := db.ExecContext(ctx, `
		UPDATE match_registry
		SET backfill_completed = COALESCE(backfill_completed, 0) | ?
		WHERE match_id = ?`, backfillFlagSkill, matchID); err != nil {
		return fmt.Errorf("MarkSkillLoaded registry(%s): %w", matchID, err)
	}
	return nil
}

// MarkParticipantsDone positionne le bit BackfillFlags["participants"] sur
// match_registry.backfill_completed. À appeler APRÈS un InsertParticipants
// réussi.
func MarkParticipantsDone(ctx context.Context, db *sql.DB, matchID string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE match_registry
		SET backfill_completed = COALESCE(backfill_completed, 0) | ?
		WHERE match_id = ?`, backfillFlagParticipants, matchID)
	if err != nil {
		return fmt.Errorf("MarkParticipantsDone(%s): %w", matchID, err)
	}
	return nil
}
