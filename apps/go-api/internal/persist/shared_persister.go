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
// INVARIANT (ADR 0019/0026, anti-ART) : écriture INSERT-only — aucun UPDATE ni
// DELETE sur les tables critiques, jamais d'ON CONFLICT DO UPDATE concurrent.
// Toute mutation per-match passe par ici sous write-lease.
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
	if err := persistWeaponAccuracy(ctx, tx, s.WeaponAccuracy); err != nil {
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
	if err := persistObjectiveStats(ctx, tx, s.ObjectiveStats); err != nil {
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
			team_0_rounds_won, team_1_rounds_won, rounds_total,
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
		row.Team0RoundsWon, row.Team1RoundsWon, row.RoundsTotal,
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
				swap_detected, delayed_damage, player_index, generation_id,
				kill_kind
			) VALUES (?, ?, ?, CAST(? AS UBIGINT), CAST(? AS UBIGINT), ?, ?, ?, ?, ?, ?, ?, ?)`,
			r.MatchID, r.XUID, r.TimeMS,
			ubigintArg(r.WeaponID), ubigintArg(r.ReconciledAs),
			r.DeltaMS, r.Confidence, r.AttributionPath,
			r.SwapDetected, r.DelayedDamage, r.PlayerIndex, gen,
			nullableStr(r.KillKind),
		)
		if err != nil {
			return fmt.Errorf("persist: INSERT weapon_kills %s/%s/%d: %w",
				r.MatchID, r.XUID, r.TimeMS, err)
		}
	}
	return nil
}

// PersistWeaponKillsNewGeneration insère `rows` (tous les weapon_kills re-dérivés d'un
// match) en UNE nouvelle génération, INSERT-only, dans une transaction dédiée. Réutilise
// persistWeaponKills (allocation nextval('weapon_kills_generation_seq') + INSERT porteur
// de kill_kind) : la vue v_weapon_kills ne lit que la génération MAX par (match_id, xuid),
// donc cette nouvelle génération SUPERSÈDE l'ancienne SANS aucun DELETE/UPDATE — ART-safe
// (ADR 0026/0030).
//
// CONTRAT ANTI-PERTE (critique) : le caller DOIT fournir TOUS les kills du (ou des)
// couple(s) (match_id, xuid) concerné(s), jamais un sous-ensemble. Une génération
// incomplète supplanterait l'ancienne complète dans la vue = perte de kills. Utilisé par
// le backfill kill_kind de l'historique H5 (re-dérive weapon_kills complet d'un match).
// `db` doit tenir le write-lease exclusif (serveur arrêté ou dblease). rows vide => no-op
// (aucune génération allouée).
func PersistWeaponKillsNewGeneration(ctx context.Context, db txBeginner, rows []WeaponKillInsert) error {
	if len(rows) == 0 {
		return nil
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist: weapon_kills new gen: BeginTx: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op si Commit réussit
	if err := persistWeaponKills(ctx, tx, rows); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist: weapon_kills new gen: commit: %w", err)
	}
	return nil
}

func persistWeaponAccuracy(ctx context.Context, tx *sql.Tx, rows []WeaponAccuracyInsert) error {
	if len(rows) == 0 {
		return nil
	}
	// INSERT pur (table sans index/PK — ART-safe, ADR 0026). Idempotence assurée
	// par l'ancre match_registry : un 2e batch sur le même match est skippé en amont
	// (cf. CollectMatchBatch), donc pas de doublon par re-collecte.
	for _, r := range rows {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO weapon_accuracy
				(match_id, xuid, weapon_id, shots_fired, shots_landed, drops)
			VALUES (?, ?, CAST(? AS UBIGINT), ?, ?, ?)`,
			r.MatchID, r.XUID, r.WeaponID, r.ShotsFired, r.ShotsLanded, r.Drops,
		)
		if err != nil {
			return fmt.Errorf("persist: INSERT weapon_accuracy %s/%s/%d: %w",
				r.MatchID, r.XUID, r.WeaponID, err)
		}
	}
	return nil
}

// persistKillerVictim ecrit les couples tueur -> victime du match, DANS LES DEUX TABLES.
//
// DOUBLE ECRITURE ASSUMEE, ET DATEE (2026-08-02) : la table canonique `match_kill_events`
// recoit les couples comme une PASSE (append-only, ADR 0026) ; la table historique
// `killer_victim_pairs` continue de les recevoir ligne a ligne, parce que ses ~20 lecteurs de
// production n ont pas encore migre.
//
// POURQUOI ELLE N A PAS ETE COUPEE LE JOUR MEME, et c est une mesure qui l a decide : le
// remplacement pur aurait fait perdre 25 % des morts sur les matchs couverts par un film. Sur
// les 949 matchs concernes, l API en compte 100 266 (les evenements `death` de
// `highlight_events`, l oracle), l ancienne table en portait 98 662 (98,4 %) et la passe de
// film n en publie que 74 569 (74,4 %). La vue `_latest` retenant UNE passe par match, la passe
// de film — plus riche par ligne (source du degat, assistant) mais plus pauvre en couverture —
// aurait EFFACE de la lecture un quart des morts, sans erreur ni compteur pour le signaler.
//
// RETRAIT : conditionne a ce que la passe de film porte l integralite du kill-feed (le
// collecteur doit partir de la liste officielle des morts et l ENRICHIR, au lieu de publier la
// seule liste qu il sait decoder). Tant que le rapport `lignes_passe_film / morts_api` mesure
// sur les matchs a film n atteint pas celui de l ancienne table, cette seconde ecriture reste.
func persistKillerVictim(ctx context.Context, tx *sql.Tx, rows []KillerVictimInsert) error {
	if len(rows) == 0 {
		return nil
	}
	if err := persistCreditKillEvents(ctx, tx, CreditKillEventsFromPairs(ctx, rows[0].MatchID, rows)); err != nil {
		return err
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

// valeurTypeHint choisit ce qui part dans la colonne `type_hint`, que DEUX champs
// visent (cf. doc de HighlightEventInsert) : le canal numérique canonique TypeHint,
// sinon le canal hérité DetailsJSON (Halo 5, identifiant de médaille en chaîne),
// sinon NULL. L'ordre est un arbitrage, pas une fusion : un row ne renseigne jamais
// les deux.
func valeurTypeHint(e HighlightEventInsert) any {
	if e.TypeHint != nil {
		return *e.TypeHint
	}
	if e.DetailsJSON != nil {
		return *e.DetailsJSON
	}
	return nil
}

// persistHighlightEvents écrit la timeline des events de highlight.
//
// COLONNES SÉPARÉES : `type_hint` reçoit un NOMBRE (nature de l'event), `raw_json`
// reçoit un DOCUMENT (l'identité de la médaille pour Halo Infinite). Avant le
// 2026-09-02 la seule colonne écrite était `type_hint`, et elle recevait
// `DetailsJSON` — d'où 415 matchs dont les events medal n'avaient AUCUNE identité
// (le fil des éliminations lit `raw_json.medal_name`). Le rattrapage de ces matchs
// est une passe hors ligne (ops.BackfillIdentiteMedailles).
func persistHighlightEvents(ctx context.Context, tx *sql.Tx, rows []HighlightEventInsert) error {
	if len(rows) == 0 {
		return nil
	}
	for _, e := range rows {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO highlight_events (match_id, event_type, time_ms, xuid, type_hint, raw_json)
			VALUES (?, ?, ?, ?, ?, ?)`,
			e.MatchID, e.EventType, e.TimeMS, e.XUID, valeurTypeHint(e), e.RawJSON,
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

// persistObjectiveStats insère les stats objectifs par joueur (match_objective_stats).
// INSERT pur — table append-only créée directement (id PK seq + written_at + vue
// _latest). Colonnes du mode absent = NULL (pointeurs nil). Aucun UPDATE / ON CONFLICT
// (ART-safe #23046) ; la relecture passe par match_objective_stats_latest.
func persistObjectiveStats(ctx context.Context, tx *sql.Tx, rows []ObjectiveStatsInsert) error {
	if len(rows) == 0 {
		return nil
	}
	// 43 colonnes / 43 placeholders / 43 arguments — 2 clés + 11 CTF + 6 Zones +
	// 6 Oddball + 6 Stockpile + 5 Extraction + 7 VIP. Toute évolution : recompter les
	// 3 listes (le décalage silencieux d'un cran est l'erreur classique de cet INSERT).
	const q = `
		INSERT INTO match_objective_stats (
			match_id, xuid,
			flag_captures, flag_capture_assists, flag_grabs, flag_secures, flag_steals,
			flag_returns, flag_carriers_killed, flag_returners_killed,
			kills_as_flag_carrier, kills_as_flag_returner, time_as_flag_carrier_seconds,
			zone_captures, zone_secures, zone_offensive_kills, zone_defensive_kills,
			zone_scoring_ticks, time_in_zones_seconds,
			kills_as_skull_carrier, skull_carriers_killed, skull_grabs, skull_scoring_ticks,
			time_as_skull_carrier_seconds, longest_time_as_skull_carrier_seconds,
			kills_as_power_seed_carrier, power_seed_carriers_killed, power_seeds_deposited,
			power_seeds_stolen, time_as_power_seed_carrier_seconds, time_as_power_seed_driver_seconds,
			extraction_conversions_completed, extraction_conversions_denied,
			extraction_initiations_completed, extraction_initiations_denied, successful_extractions,
			kills_as_vip, vip_kills, vip_assists, times_selected_as_vip,
			max_killing_spree_as_vip, time_as_vip_seconds, longest_time_as_vip_seconds
		) VALUES (
			?, ?,
			?, ?, ?, ?, ?,
			?, ?, ?,
			?, ?, ?,
			?, ?, ?, ?,
			?, ?,
			?, ?, ?, ?,
			?, ?,
			?, ?, ?,
			?, ?, ?,
			?, ?,
			?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?
		)`
	for _, r := range rows {
		_, err := tx.ExecContext(ctx, q,
			r.MatchID, r.XUID,
			r.FlagCaptures, r.FlagCaptureAssists, r.FlagGrabs, r.FlagSecures, r.FlagSteals,
			r.FlagReturns, r.FlagCarriersKilled, r.FlagReturnersKilled,
			r.KillsAsFlagCarrier, r.KillsAsFlagReturner, r.TimeAsFlagCarrierSeconds,
			r.ZoneCaptures, r.ZoneSecures, r.ZoneOffensiveKills, r.ZoneDefensiveKills,
			r.ZoneScoringTicks, r.TimeInZonesSeconds,
			r.KillsAsSkullCarrier, r.SkullCarriersKilled, r.SkullGrabs, r.SkullScoringTicks,
			r.TimeAsSkullCarrierSeconds, r.LongestTimeAsSkullCarrierSeconds,
			r.KillsAsPowerSeedCarrier, r.PowerSeedCarriersKilled, r.PowerSeedsDeposited,
			r.PowerSeedsStolen, r.TimeAsPowerSeedCarrierSeconds, r.TimeAsPowerSeedDriverSeconds,
			r.ExtractionConversionsCompleted, r.ExtractionConversionsDenied,
			r.ExtractionInitiationsCompleted, r.ExtractionInitiationsDenied, r.SuccessfulExtractions,
			r.KillsAsVip, r.VipKills, r.VipAssists, r.TimesSelectedAsVip,
			r.MaxKillingSpreeAsVip, r.TimeAsVipSeconds, r.LongestTimeAsVipSeconds,
		)
		if err != nil {
			return fmt.Errorf("persist: INSERT match_objective_stats %s/%s: %w", r.MatchID, r.XUID, err)
		}
	}
	return nil
}

// InsertObjectiveStats est le point d'entrée EXPORTÉ (backfill CLI) pour écrire
// des rows match_objective_stats hors du chemin SharedBatch : ouvre une
// transaction sur db et réutilise persistObjectiveStats (INSERT-only ART-safe,
// #23046). DRY — une seule copie du SQL d'INSERT (persistObjectiveStats).
// Pré-requis : db en accès RW exclusif (serveur arrêté, un seul writer par DB).
func InsertObjectiveStats(ctx context.Context, db txBeginner, rows []ObjectiveStatsInsert) error {
	if len(rows) == 0 {
		return nil
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist: begin tx objective_stats: %w", err)
	}
	if err := persistObjectiveStats(ctx, tx, rows); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist: commit objective_stats: %w", err)
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

// nullableStr mappe une string vide sur NULL (sinon la valeur telle quelle). Sert
// aux colonnes optionnelles ou "vide == absent" (ex. weapon_kills.kill_kind : le
// chemin Infinite ne porte pas la mecanique de kill → NULL, pas chaine vide).
func nullableStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
