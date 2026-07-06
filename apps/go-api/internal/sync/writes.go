// Package sync — writes.go : helpers d'insertion/upsert dans DuckDB.
//
// Portage de SharedWritesMixin + EnrichedWritesMixin (Python).
// Toutes les fonctions prennent un *sql.DB et travaillent en-dehors de toute
// transaction — la gestion du commit est faite par l'engine.
package sync

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// ──────────────────────────────────────────────────────────────────────────────
// Shared DB writes
// ──────────────────────────────────────────────────────────────────────────────

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
	// ART-safe : SELECT-then-UPDATE-or-INSERT (pas d'ON CONFLICT DO UPDATE, qui
	// réécrit la ligne via l'index ART de la PK xuid et peut FATAL-invalider la base
	// partagée). xuid_aliases n'a aucun index secondaire → l'UPDATE n'en touche aucun.
	var dummy int
	err := db.QueryRowContext(ctx, `SELECT 1 FROM xuid_aliases WHERE xuid = ?`, xuid).Scan(&dummy)
	switch {
	case err == nil:
		if _, execErr := db.ExecContext(ctx, `
			UPDATE xuid_aliases SET gamertag = ?, last_seen = ?, updated_at = ? WHERE xuid = ?`,
			gamertag, now, now, xuid); execErr != nil {
			return fmt.Errorf("UpsertXUIDAlias(%s): %w", xuid, execErr)
		}
	case errors.Is(err, sql.ErrNoRows):
		if _, execErr := db.ExecContext(ctx, `
			INSERT INTO xuid_aliases (xuid, gamertag, last_seen, source, updated_at)
			VALUES (?, ?, ?, 'sync', ?)`,
			xuid, gamertag, now, now); execErr != nil {
			return fmt.Errorf("UpsertXUIDAlias(%s): %w", xuid, execErr)
		}
	default:
		return fmt.Errorf("UpsertXUIDAlias(%s) lookup: %w", xuid, err)
	}
	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Player DB writes
// ──────────────────────────────────────────────────────────────────────────────

// UpsertPlayerEnrichment écrit la row baseline stage='live' d'un match collecté
// (chemin legacy non-batch : engine_fetch / engine_process_match). Append-only #23046 :
// INSERT pur (plus d'ON CONFLICT). teammates_signature écrit si fourni (sinon NULL —
// un stage 'teammates' ultérieur ou la baseline fournira la valeur via la vue merge).
// Marque le match comme collecté pour le known-set (loadKnownMatchIDs). L'idempotence
// est assurée en amont par le delta de découverte (match déjà connu = non re-traité).
func UpsertPlayerEnrichment(ctx context.Context, db *sql.DB, matchID, teammatesSig string) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO player_match_enrichment (match_id, teammates_signature, stage)
		VALUES (?, ?, 'live')`,
		matchID, nullStr(teammatesSig),
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
	// ART-safe : SELECT-then-UPDATE-or-INSERT (pas d'ON CONFLICT sur la PK key).
	var dummy int
	err := db.QueryRowContext(ctx, `SELECT 1 FROM sync_meta WHERE key = ?`, key).Scan(&dummy)
	switch {
	case err == nil:
		_, err = db.ExecContext(ctx, `UPDATE sync_meta SET value = ?, updated_at = ? WHERE key = ?`, value, now, key)
	case errors.Is(err, sql.ErrNoRows):
		_, err = db.ExecContext(ctx, `INSERT INTO sync_meta (key, value, updated_at) VALUES (?, ?, ?)`, key, value, now)
	}
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

	// Append-only #23046 (Phase 2) : plus de DELETE WHERE (match_id,xuid) sur idx_wk
	// (vecteur ART, DB shared multi-writer). Chaque write alloue UNE génération
	// (weapon_kills_generation_seq) partagée par tous les kills du (match,xuid) ; la
	// vue v_weapon_kills ne lit que la génération MAX → supersède l'ancienne.
	var gen int64
	if err := tx.QueryRowContext(ctx, `SELECT nextval('weapon_kills_generation_seq')`).Scan(&gen); err != nil {
		return fmt.Errorf("InsertWeaponKills gen(%s,%s): %w", matchID, xuid, err)
	}

	for _, r := range attrs {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO weapon_kills (
				match_id, xuid, time_ms, weapon_id, reconciled_as,
				delta_ms, confidence, attribution_path,
				swap_detected, delayed_damage, player_index, generation_id
			) VALUES (?, ?, ?, CAST(? AS UBIGINT), CAST(? AS UBIGINT), ?, ?, ?, ?, ?, ?, ?)`,
			matchID, xuid, r.TimeMS, ubigintArg(r.WeaponID), ubigintArg(r.ReconciledAs),
			r.DeltaMS, r.Confidence, r.AttributionPath,
			r.SwapDetected, r.DelayedDamage, r.PlayerIndex, gen,
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

// InsertPersonalScoreAwards remplace l'ENSEMBLE des awards d'un (matchID, xuid)
// par la nouvelle extraction, en APPEND-ONLY (#23046, Phase 2 — plus de
// DELETE+INSERT, vecteur ART sur les 4 index idx_psa_*).
//
// Sémantique REPLACE préservée sans mutation : chaque appel alloue UN
// generation_id (séquence psa_generation_seq, partagé par tout le batch) et
// INSÈRE pur. La vue personal_score_awards_latest ne lit que la génération MAX
// par (match_id,xuid) — donc cette extraction supersède la précédente.
// Extraction vide (clear) : on INSÈRE un TOMBSTONE (is_tombstone=TRUE) ; la vue
// (filtre NOT is_tombstone) ne retourne alors rien pour ce (match,xuid).
//
// Idempotent au sens lecture : ré-exécuter alloue une nouvelle génération
// identique ; la table physique grossit (append-only) mais _latest est stable.
func InsertPersonalScoreAwards(ctx context.Context, db *sql.DB, matchID, xuid string, rows []PersonalScoreAwardRow) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("InsertPersonalScoreAwards begin: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	var gen int64
	if err := tx.QueryRowContext(ctx, `SELECT nextval('psa_generation_seq')`).Scan(&gen); err != nil {
		return fmt.Errorf("InsertPersonalScoreAwards gen(%s,%s): %w", matchID, xuid, err)
	}
	if len(rows) == 0 {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO personal_score_awards
				(match_id, xuid, award_name, generation_id, is_tombstone)
			VALUES (?, ?, '', ?, TRUE)`,
			matchID, xuid, gen,
		); err != nil {
			return fmt.Errorf("InsertPersonalScoreAwards tombstone(%s,%s): %w", matchID, xuid, err)
		}
		return tx.Commit()
	}
	for _, r := range rows {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO personal_score_awards
				(match_id, xuid, award_name, award_category, award_count, award_score, generation_id)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			r.MatchID, r.XUID, r.AwardName, r.AwardCategory, r.AwardCount, r.AwardScore, gen,
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
