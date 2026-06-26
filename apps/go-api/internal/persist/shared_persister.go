// Package persist — shared_persister.go : écriture INSERT-only atomique du
// sous-batch SharedBatch dans shared_matches_v2.duckdb.
//
// Architecture (refactor 2026-05-23) :
//
//   - INSERT-only : aucun UPDATE, aucun DELETE — évite le bug ART DuckDB
//     observé en prod (cf. .ai/INCIDENT_ART_CORRUPTION_DUCKDB.md +
//     .ai/REFACTOR_COLLECT_PERSIST.md).
//   - Atomique : 1 transaction par batch. BEGIN → INSERT × N → COMMIT ou
//     ROLLBACK sur erreur. Aucun état partiel possible.
//   - Idempotent : si match_id existe déjà dans match_registry, return nil
//     (le batch est considéré déjà persisté — ACK silently côté worker).
//
// **Pré-requis** : le caller (worker) DOIT avoir acquis le lease exclusif
// sur shared.duckdb via `dblease.AcquireWriterCtx` avant d'appeler Persist.
// L'interface txBeginner accepte aussi bien *sql.DB que *LeasedWriter pour
// rester découplée du package dblease.

package persist

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"levelup/go-api/internal/domain"
)

// txBeginner abstrait *sql.DB ou *dblease.LeasedWriter — le persister ne
// dépend que de BeginTx pour rester découplé du package dblease.
type txBeginner interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// SharedPersister écrit un SharedBatch dans shared_matches_v2.duckdb en
// INSERT-only et atomic.
type SharedPersister struct {
	db txBeginner
}

// NewSharedPersister construit un persister. `db` doit pointer vers une
// connexion ayant un write lease actif sur shared_matches_v2.duckdb.
func NewSharedPersister(db txBeginner) *SharedPersister {
	return &SharedPersister{db: db}
}

// Persist écrit le SharedBatch en 1 transaction INSERT-only.
//
// Cas particuliers :
//
//   - batch == nil               → error (defensive).
//   - batch.Shared.Match == nil  → no-op (le SharedPersister ne traite que
//     les batches avec une row match_registry à créer).
//   - match_id déjà dans match_registry → skip (return nil) sans rollback
//     d'erreur — le worker ACK et supprime le WAL.
//
// En cas d'erreur sur n'importe quelle INSERT, la transaction est ROLLBACK
// (defer tx.Rollback() — no-op après Commit réussi).
func (p *SharedPersister) Persist(ctx context.Context, batch *MatchBatch) error {
	if batch == nil {
		return errors.New("persist: SharedPersister.Persist: batch nil")
	}
	s := &batch.Shared
	if s.Match == nil {
		return nil // pas de match à créer → no-op
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist: BeginTx: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op si Commit réussit

	// Idempotence : skip si match_id existe déjà.
	var exists bool
	err = tx.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM match_registry WHERE match_id = ?)`,
		s.Match.MatchID,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("persist: check exists %s: %w", s.Match.MatchID, err)
	}
	if exists {
		// Batch déjà persisté précédemment. Retourner nil pour que le
		// worker ACK et supprime le WAL.
		return nil
	}

	// events_loaded est dérivé de la présence RÉELLE de highlight_events dans le
	// batch (pas d'un flag précalculé qui pourrait dériver) — même sémantique que
	// la complétion legacy markCompletionRegistry (events_loaded = events_inserted>0).
	// Posé à l'INSERT (mode INSERT-only, pas de heal post-coup) : un match
	// synchronisé avec film n'est plus jamais re-flaggé candidat backfill events.
	if err := persistMatchRegistry(ctx, tx, s.Match, len(s.HighlightEvents) > 0); err != nil {
		return err
	}
	if err := persistParticipants(ctx, tx, s.Participants); err != nil {
		return err
	}
	if err := persistMedals(ctx, tx, s.Medals); err != nil {
		return err
	}
	if err := persistWeaponKills(ctx, tx, s.WeaponKills); err != nil {
		return err
	}
	if err := persistKillerVictim(ctx, tx, s.KillerVictim); err != nil {
		return err
	}
	if err := persistKillPositions(ctx, tx, s.KillPositions); err != nil {
		return err
	}
	if err := persistHighlightEvents(ctx, tx, s.HighlightEvents); err != nil {
		return err
	}
	if err := persistXUIDAliases(ctx, tx, s.XUIDAliases); err != nil {
		return err
	}
	if err := persistMatchCSRs(ctx, tx, s.MatchCSRs); err != nil {
		return err
	}
	if err := persistCommendations(ctx, tx, s.Commendations); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist: Commit %s: %w", s.Match.MatchID, err)
	}
	return nil
}

// ─── Helpers INSERT par table ──────────────────────────────────────────────

func persistMatchRegistry(ctx context.Context, tx *sql.Tx, row *domain.MatchRegistryRow, eventsLoaded bool) error {
	now := time.Now().UTC()
	startUTC := row.StartTime.UTC()
	var endUTC any
	if row.EndTime != nil {
		t := row.EndTime.UTC()
		endUTC = t
	}
	_, err := tx.ExecContext(ctx, `
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
			match_intensity, backfill_completed, events_loaded,
			first_sync_by, first_sync_at, last_updated_at,
			player_count,
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
			?, ?, ?,
			?,
			?, ?
		)`,
		row.MatchID, row.StartTime, row.EndTime, startUTC, endUTC,
		row.PlaylistID, row.PlaylistName, row.PlaylistVersionID,
		row.MapID, row.MapName, row.MapVersionID,
		row.PairID, row.PairName, row.PairVersionID,
		row.GameVariantID, row.GameVariantName, row.GameVariantVersionID,
		row.ModeCategory, row.IsRanked, row.IsFirefight, row.SeasonID,
		row.DurationSeconds, row.PlayableDurationSeconds,
		row.RealStartTime, row.Team0Score, row.Team1Score,
		row.Team0PSScore, row.Team1PSScore,
		row.MatchIntensity, row.BackfillCompleted, eventsLoaded,
		row.FirstSyncBy, now, now,
		row.PlayerCount,
		now, now,
	)
	if err != nil {
		return fmt.Errorf("persist: INSERT match_registry %s: %w", row.MatchID, err)
	}
	return nil
}

func persistParticipants(ctx context.Context, tx *sql.Tx, rows []domain.MatchParticipantRow) error {
	if len(rows) == 0 {
		return nil
	}
	now := time.Now().UTC()
	for _, row := range rows {
		_, err := tx.ExecContext(ctx, `
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
				assassination_kills, ground_pound_kills, shoulder_bash_kills,
				present_at_beginning, present_at_completion, joined_in_progress, left_in_progress,
				first_joined_time, last_leave_time,
				backfill_bits,
				created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
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
			row.AssassinationKills, row.GroundPoundKills, row.ShoulderBashKills,
			row.PresentAtBeginning, row.PresentAtCompletion, row.JoinedInProgress, row.LeftInProgress,
			row.FirstJoinedTime, row.LastLeaveTime,
			row.BackfillBits,
			now,
		)
		if err != nil {
			return fmt.Errorf("persist: INSERT match_participants %s/%s: %w", row.MatchID, row.XUID, err)
		}
	}
	return nil
}

func persistMedals(ctx context.Context, tx *sql.Tx, rows []domain.MedalRow) error {
	if len(rows) == 0 {
		return nil
	}
	now := time.Now().UTC()
	for _, row := range rows {
		// INSERT OR IGNORE pour tolérer les doublons dans le payload
		// (l'API Halo peut occasionnellement renvoyer la même médaille 2x).
		_, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO medals_earned (match_id, xuid, medal_name_id, count, created_at)
			VALUES (?, ?, ?, ?, ?)`,
			row.MatchID, row.XUID, row.MedalNameID, row.Count, now,
		)
		if err != nil {
			return fmt.Errorf("persist: INSERT medals_earned %s/%s/%d: %w",
				row.MatchID, row.XUID, row.MedalNameID, err)
		}
	}
	return nil
}

func persistWeaponKills(ctx context.Context, tx *sql.Tx, rows []WeaponKillInsert) error {
	if len(rows) == 0 {
		return nil
	}
	// Append-only #23046 (Phase 2) : alloue UNE génération partagée par le batch
	// (weapon_kills_generation_seq) ; la vue v_weapon_kills ne lit que la génération
	// MAX par (match_id,xuid). Plus de DELETE préalable (vecteur ART sur idx_wk).
	var gen int64
	if err := tx.QueryRowContext(ctx, `SELECT nextval('weapon_kills_generation_seq')`).Scan(&gen); err != nil {
		return fmt.Errorf("persist: weapon_kills generation: %w", err)
	}
	for _, r := range rows {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO weapon_kills (
				match_id, xuid, time_ms,
				weapon_id, reconciled_as,
				delta_ms, confidence, attribution_path,
				swap_detected, delayed_damage, player_index, generation_id
			) VALUES (?, ?, ?, CAST(? AS UBIGINT), CAST(? AS UBIGINT), ?, ?, ?, ?, ?, ?, ?)`,
			r.MatchID, r.XUID, r.TimeMS,
			ubigintArg(r.WeaponID), ubigintArg(r.ReconciledAs),
			r.DeltaMS, r.Confidence, r.AttributionPath,
			r.SwapDetected, r.DelayedDamage, r.PlayerIndex, gen,
		)
		if err != nil {
			return fmt.Errorf("persist: INSERT weapon_kills %s/%s/%d: %w",
				r.MatchID, r.XUID, r.TimeMS, err)
		}
	}
	return nil
}

func persistKillerVictim(ctx context.Context, tx *sql.Tx, rows []KillerVictimInsert) error {
	if len(rows) == 0 {
		return nil
	}
	now := time.Now().UTC()
	for _, p := range rows {
		// Forme par-kill complète : killer_gamertag / victim_gamertag / time_ms
		// sont lus par le match-view (Q20KVPairs — tug-of-war, KD timeline,
		// antagonistes). Parité avec la complétion legacy EventsCompletionPersister.
		// INSERT pur (table sans index/PK, cf. steps_shared.go) — ART-safe.
		_, err := tx.ExecContext(ctx, `
			INSERT INTO killer_victim_pairs
				(match_id, killer_xuid, killer_gamertag, victim_xuid, victim_gamertag, kill_count, time_ms, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			p.MatchID, p.KillerXUID, p.KillerGamertag, p.VictimXUID, p.VictimGamertag, p.Count, p.TimeMS, now,
		)
		if err != nil {
			return fmt.Errorf("persist: INSERT killer_victim_pairs %s/%s→%s: %w",
				p.MatchID, p.KillerXUID, p.VictimXUID, err)
		}
	}
	return nil
}

func persistKillPositions(ctx context.Context, tx *sql.Tx, rows []KillPositionInsert) error {
	if len(rows) == 0 {
		return nil
	}
	for _, r := range rows {
		// INSERT pur — table append-only (positions par kill, jamais ré-écrites).
		_, err := tx.ExecContext(ctx, `
			INSERT INTO kill_positions (
				match_id, killer_xuid, time_ms,
				killer_x, killer_y, killer_z, victim_x, victim_y, victim_z
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			r.MatchID, r.KillerXUID, r.TimeMS,
			r.KillerX, r.KillerY, r.KillerZ, r.VictimX, r.VictimY, r.VictimZ,
		)
		if err != nil {
			return fmt.Errorf("persist: INSERT kill_positions %s/%s/%d: %w",
				r.MatchID, r.KillerXUID, r.TimeMS, err)
		}
	}
	return nil
}

func persistHighlightEvents(ctx context.Context, tx *sql.Tx, rows []HighlightEventInsert) error {
	if len(rows) == 0 {
		return nil
	}
	for _, e := range rows {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO highlight_events (match_id, event_type, time_ms, xuid, type_hint)
			VALUES (?, ?, ?, ?, ?)`,
			e.MatchID, e.EventType, e.TimeMS, e.XUID, e.DetailsJSON,
		)
		if err != nil {
			return fmt.Errorf("persist: INSERT highlight_events %s/%s/%d: %w",
				e.MatchID, e.EventType, e.TimeMS, err)
		}
	}
	return nil
}

func persistXUIDAliases(ctx context.Context, tx *sql.Tx, rows []XUIDAliasInsert) error {
	if len(rows) == 0 {
		return nil
	}
	now := time.Now().UTC()
	for _, a := range rows {
		// INSERT OR IGNORE : préserve les xuid_aliases existants. Le
		// rafraîchissement des aliases (si gamertag a changé) est laissé
		// à un job dédié, pas au flux sync de matchs.
		_, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO xuid_aliases (xuid, gamertag, last_seen, source, updated_at)
			VALUES (?, ?, ?, 'sync', ?)`,
			a.XUID, a.Gamertag, a.LastSeen, now,
		)
		if err != nil {
			return fmt.Errorf("persist: INSERT xuid_aliases %s: %w", a.XUID, err)
		}
	}
	return nil
}

func persistMatchCSRs(ctx context.Context, tx *sql.Tx, rows []MatchCSRInsert) error {
	if len(rows) == 0 {
		return nil
	}
	for _, c := range rows {
		ratingType := c.RatingType
		if ratingType == "" {
			ratingType = "CSR"
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO match_csrs (
				match_id, xuid, rating_type,
				rating_value, tier, sub_tier, tier_label,
				rating_delta, measurement_matches_remaining, season_id
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			c.MatchID, c.XUID, ratingType,
			c.RatingValue, c.Tier, c.SubTier, c.TierLabel,
			c.RatingDelta, c.MeasurementMatchesRemaining, c.SeasonID,
		)
		if err != nil {
			return fmt.Errorf("persist: INSERT match_csrs %s/%s: %w", c.MatchID, c.XUID, err)
		}
	}
	return nil
}

func persistCommendations(ctx context.Context, tx *sql.Tx, rows []CommendationInsert) error {
	if len(rows) == 0 {
		return nil
	}
	now := time.Now().UTC()
	for _, row := range rows {
		// INSERT OR IGNORE sur la clé naturelle (match_id, xuid, commendation_id) —
		// même garantie ART-safe que medals_earned : aucun UPDATE sur count, clé
		// jamais mutée. Tolère un delta dupliqué dans le payload (best-effort).
		_, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO match_commendations (match_id, xuid, commendation_id, count, progress, created_at)
			VALUES (?, ?, ?, ?, ?, ?)`,
			row.MatchID, row.XUID, row.CommendationID, row.Count, row.Progress, now,
		)
		if err != nil {
			return fmt.Errorf("persist: INSERT match_commendations %s/%s/%s: %w",
				row.MatchID, row.XUID, row.CommendationID, err)
		}
	}
	return nil
}

// ubigintArg sérialise un *uint64 en string décimale pour CAST(? AS UBIGINT)
// côté DuckDB. Workaround driver duckdb-go qui ne supporte pas l'envoi natif
// UBIGINT (cf. writes.go::ubigintArg).
func ubigintArg(p *uint64) any {
	if p == nil {
		return nil
	}
	return strconv.FormatUint(*p, 10)
}
