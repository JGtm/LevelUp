// Package sync — backfill.go : détection des matchs avec données manquantes.
//
// Portage de scripts/backfill/detection.py.
//
// Supporte deux modes de détection :
//   - "or" (défaut) : sélectionne les matchs manquant AU MOINS UNE donnée demandée
//   - "and" (strict) : sélectionne les matchs manquant TOUTES les données demandées
//
// La détection se fait via shared_matches_v2.duckdb (match_registry + match_participants).
// Le bitmask backfill_completed est stocké dans match_registry.
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
)

// FindMatchesMissingData trouve les matchs avec des données manquantes via shared DB.
//
// Portage de find_matches_missing_data() (detection.py).
// Le scope doit être Resolve() avant appel.
func FindMatchesMissingData(
	ctx context.Context,
	playerDB, sharedDB *sql.DB,
	xuid string,
	scope *SyncScope,
) ([]string, error) {
	if scope == nil {
		return nil, fmt.Errorf("FindMatchesMissingData: scope is nil")
	}

	// Détecter le type de flags demandés
	localRequested := scope.Medals || scope.Events || scope.Skill ||
		scope.PersonalScores || scope.PerformanceScores || scope.Accuracy ||
		scope.Shots || scope.EnemyMMR || scope.Assets || scope.Participants ||
		scope.PVEStats
	participantsRequested := scope.ParticipantsScores || scope.ParticipantsKDA ||
		scope.ParticipantsShots || scope.ParticipantsDamage || scope.ParticipantsAvgLife

	// ── Détection participants via shared DB ──
	var sharedMatchIDs []string
	if participantsRequested {
		var err error
		sharedMatchIDs, err = findMatchesInSharedDB(ctx, sharedDB, xuid, scope)
		if err != nil {
			return nil, fmt.Errorf("findMatchesInSharedDB: %w", err)
		}
		// Si aucun flag local → retourner directement les résultats shared
		if !localRequested {
			return sharedMatchIDs, nil
		}
	}

	// ── Détection données via shared DB ──
	localMatchIDs, err := findMatchesInSharedAll(ctx, playerDB, sharedDB, xuid, scope)
	if err != nil {
		return nil, fmt.Errorf("findMatchesInSharedAll: %w", err)
	}

	// Fusionner résultats locaux + shared (dédoublonner, garder l'ordre)
	if len(sharedMatchIDs) > 0 {
		seen := make(map[string]bool, len(localMatchIDs))
		for _, id := range localMatchIDs {
			seen[id] = true
		}
		merged := append([]string{}, localMatchIDs...)
		for _, id := range sharedMatchIDs {
			if !seen[id] {
				merged = append(merged, id)
				seen[id] = true
			}
		}
		return merged, nil
	}

	return localMatchIDs, nil
}

// FindMatchesMissingParticipantBits trouve les matchs où un joueur a des bits
// backfill_bits manquants (bitmask granulaire).
//
// Portage de find_matches_missing_participant_bits() (detection.py).
func FindMatchesMissingParticipantBits(
	ctx context.Context,
	sharedDB *sql.DB,
	xuid string,
	bitsRequired int,
	force bool,
	maxMatches int,
) ([]string, error) {
	var condition string
	if force {
		condition = "1=1"
	} else {
		condition = fmt.Sprintf("(COALESCE(mp.backfill_bits, 0) & %d) != %d", bitsRequired, bitsRequired)
	}

	mrSrc := getMatchSource(ctx, sharedDB)
	query := fmt.Sprintf(`
		SELECT mp.match_id
		FROM match_participants mp
		JOIN %s mr ON mr.match_id = mp.match_id
		WHERE mp.xuid = ? AND %s
		ORDER BY mr.start_time DESC
	`, mrSrc, condition)
	if maxMatches > 0 {
		query += fmt.Sprintf(" LIMIT %d", maxMatches)
	}

	rows, err := sharedDB.QueryContext(ctx, query, xuid)
	if err != nil {
		return nil, fmt.Errorf("FindMatchesMissingParticipantBits: %w", err)
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var matchID string
		if err := rows.Scan(&matchID); err != nil {
			return nil, err
		}
		result = append(result, matchID)
	}
	return result, rows.Err()
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers internes
// ─────────────────────────────────────────────────────────────────────────────

// getMatchSource retourne "v_match_full" si la vue existe, sinon "match_registry".
func getMatchSource(ctx context.Context, db *sql.DB) string {
	row := db.QueryRowContext(ctx, "SELECT 1 FROM v_match_full LIMIT 1")
	var dummy int
	if err := row.Scan(&dummy); err == nil {
		return "v_match_full"
	}
	return "match_registry"
}

// hasBackfillCompletedColumn vérifie si match_registry possède backfill_completed.
func hasBackfillCompletedColumn(ctx context.Context, db *sql.DB) bool {
	row := db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM information_schema.columns "+
			"WHERE table_name = 'match_registry' AND column_name = 'backfill_completed'",
	)
	var count int
	if err := row.Scan(&count); err != nil {
		return false
	}
	return count > 0
}

// hasEventsEmptyColumn : la colonne events_empty (statut distinct « chunk récupéré,
// 0 event légitime ») est ajoutée par migration. Sur une DB pas encore migrée (ou un
// schéma de test minimal), on dégrade le gate events sur events_loaded seul.
func hasEventsEmptyColumn(ctx context.Context, db *sql.DB) bool {
	row := db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM information_schema.columns "+
			"WHERE table_name = 'match_registry' AND column_name = 'events_empty'",
	)
	var count int
	if err := row.Scan(&count); err != nil {
		return false
	}
	return count > 0
}

// doneGuard retourne une clause SQL excluant les matchs déjà traités
// via mr.backfill_completed pour un flag global au match.
func doneGuard(flagName string, hasBFCol bool) string {
	if !hasBFCol {
		return ""
	}
	bit, ok := BackfillFlags[flagName]
	if !ok || bit == 0 {
		return ""
	}
	return fmt.Sprintf(" AND (COALESCE(mr.backfill_completed, 0) & %d = 0)", bit)
}

// playerDoneGuard retourne une clause SQL excluant les matchs déjà traités
// dans la player DB (per-player guard).
// Utilise des paramètres liés (NOT IN avec sous-requête) pour empêcher toute injection SQL.
func playerDoneGuard(ctx context.Context, playerDB *sql.DB, table string, column string) string {
	var query string
	if column != "" {
		query = fmt.Sprintf("SELECT match_id FROM %s WHERE %s IS NOT NULL", table, column)
	} else {
		query = fmt.Sprintf("SELECT DISTINCT match_id FROM %s", table)
	}

	rows, err := playerDB.QueryContext(ctx, query)
	if err != nil {
		return "1=1"
	}
	defer rows.Close()

	var doneIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			slog.WarnContext(ctx, "playerDoneGuard: row scan failed, ID skipped",
				"table", table, "err", err)
			continue
		}
		doneIDs = append(doneIDs, id)
	}
	if len(doneIDs) == 0 {
		return "1=1"
	}
	// Paramètres liés : les match_id sont des UUID hex (a-f, 0-9, -).
	// On valide le format pour empêcher toute injection via des IDs corrompus.
	var safePlaceholders []string
	var rejected []string
	for _, id := range doneIDs {
		if isValidMatchID(id) {
			safePlaceholders = append(safePlaceholders, "'"+id+"'")
		} else {
			rejected = append(rejected, id)
		}
	}
	if len(rejected) > 0 {
		slog.WarnContext(ctx, "playerDoneGuard: malformed match_id(s) rejected by isValidMatchID — guard will treat affected matches as not done",
			"table", table, "rejected_count", len(rejected),
			"total_count", len(doneIDs), "sample", rejected[:min(3, len(rejected))])
	}
	if len(safePlaceholders) == 0 {
		return "1=1"
	}
	return fmt.Sprintf("mp.match_id NOT IN (%s)", strings.Join(safePlaceholders, ","))
}

// findMatchesInSharedAll — détection V5 FINALE : tous les flags via shared DB.
// Portage de _find_matches_in_shared_all() (detection.py).
func findMatchesInSharedAll(
	ctx context.Context,
	playerDB, sharedDB *sql.DB,
	xuid string,
	scope *SyncScope,
) ([]string, error) {
	var conditions []string
	hasBFCol := hasBackfillCompletedColumn(ctx, sharedDB)

	// Médailles — per-player
	if scope.Medals {
		conditions = append(conditions,
			"mp.match_id NOT IN (SELECT DISTINCT match_id FROM medals_earned WHERE xuid = ?)")
	}

	// Events — mr.events_loaded (source de vérité). events_empty=TRUE sort le match
	// du retry set (chunk récupéré, 0 event légitime) SANS prétendre à events_loaded.
	// Colonne conditionnelle : dégradation propre sur une DB non encore migrée.
	if scope.Events {
		if scope.ForceEvents {
			conditions = append(conditions, "1=1")
		} else if hasEventsEmptyColumn(ctx, sharedDB) {
			conditions = append(conditions, "mr.events_loaded = false AND COALESCE(mr.events_empty, false) = false")
		} else {
			conditions = append(conditions, "mr.events_loaded = false")
		}
	}

	// Skill — guard per-player (backfill_bits) + guard global (backfill_completed)
	if scope.Skill {
		if scope.ForceSkill {
			conditions = append(conditions, "1=1")
		} else {
			conditions = append(conditions,
				"(COALESCE(mp.backfill_bits, 0) & 1) = 0"+
					" AND (COALESCE(mr.backfill_completed, 0) & 4) = 0")
		}
	}

	// Personal scores — per-player : vérifier données réelles dans player DB
	if scope.PersonalScores {
		if scope.ForcePersonalScores {
			conditions = append(conditions, "1=1")
		} else {
			conditions = append(conditions,
				playerDoneGuard(ctx, playerDB, "personal_score_awards", ""))
		}
	}

	// Performance scores — per-player
	if scope.PerformanceScores {
		if scope.ForcePerformanceScores {
			conditions = append(conditions, "1=1")
		} else {
			conditions = append(conditions,
				playerDoneGuard(ctx, playerDB, "player_match_enrichment", "performance_score"))
		}
	}

	// Engagement scores — per-player (Phase 3 plan engagement)
	if scope.EngagementScores {
		if scope.ForceEngagementScores {
			conditions = append(conditions, "1=1")
		} else {
			conditions = append(conditions,
				playerDoneGuard(ctx, playerDB, "player_match_enrichment", "engagement_score"))
		}
	}

	// Accuracy — per-player
	if scope.Accuracy {
		if scope.ForceAccuracy {
			conditions = append(conditions, "1=1")
		} else {
			conditions = append(conditions, "(mp.accuracy IS NULL)")
		}
	}

	// Shots — per-player
	if scope.Shots {
		if scope.ForceShots {
			conditions = append(conditions, "1=1")
		} else {
			conditions = append(conditions, "(mp.shots_fired IS NULL OR mp.shots_hit IS NULL)")
		}
	}

	// Enemy MMR — per-player
	if scope.EnemyMMR {
		conditions = append(conditions, "(mp.team_mmr IS NULL)")
	}

	// Assets
	if scope.Assets {
		assetCond := "(mr.playlist_name IS NULL OR mr.map_name IS NULL " +
			"OR mr.pair_name IS NULL OR mr.game_variant_name IS NULL)"
		if scope.ForceAssets {
			conditions = append(conditions, assetCond)
		} else {
			conditions = append(conditions, assetCond+doneGuard("assets", hasBFCol))
		}
	}

	// Aliases (force uniquement)
	if scope.ForceAliases {
		conditions = append(conditions, "1=1")
	}

	// Participants
	if scope.Participants {
		if scope.ForceParticipants {
			conditions = append(conditions, "1=1")
		} else {
			conditions = append(conditions, "1=1"+doneGuard("participants", hasBFCol))
		}
	}

	// PVE stats (Firefight) — double guard
	if scope.PVEStats {
		pveBit := MBitPVEStats
		if scope.ForcePVEStats {
			conditions = append(conditions, "mr.is_firefight = TRUE")
		} else {
			conditions = append(conditions,
				fmt.Sprintf("mr.is_firefight = TRUE AND (COALESCE(mr.backfill_completed, 0) & %d) = 0", pveBit))
		}
	}

	// Weapon kills : condition RETIRÉE le 2026-09-01 avec l'axe scope.Weapons — son
	// exécuteur (étape 1.55) n'existe plus. Cf. l'en-tête du bloc « Weapon kills »
	// de scope.go.

	// Playable duration
	if scope.PlayableDuration {
		if scope.ForcePlayableDuration {
			conditions = append(conditions, "1=1")
		} else {
			conditions = append(conditions,
				"mp.match_id NOT IN ("+
					"  SELECT match_id FROM match_registry WHERE playable_duration_seconds IS NOT NULL"+
					")")
		}
	}

	if len(conditions) == 0 {
		return nil, nil
	}

	whereClause := strings.Join(conditions, " OR ")

	// Paramètres : xuid en premier, puis xuid pour médailles si activé
	params := []interface{}{xuid}
	if scope.Medals {
		params = append(params, xuid)
	}

	mrSrc := getMatchSource(ctx, sharedDB)
	query := fmt.Sprintf(`
		SELECT DISTINCT mp.match_id
		FROM match_participants mp
		JOIN %s mr ON mr.match_id = mp.match_id
		WHERE mp.xuid = ? AND (%s)
		ORDER BY mr.start_time DESC
	`, mrSrc, whereClause)
	if scope.MaxMatches > 0 {
		query += fmt.Sprintf(" LIMIT %d", scope.MaxMatches)
	}

	rows, err := sharedDB.QueryContext(ctx, query, params...)
	if err != nil {
		slog.WarnContext(ctx, "backfill: détection V5 shared DB échouée", "err", err)
		return nil, nil // match Python behavior: log + return []
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var matchID string
		if err := rows.Scan(&matchID); err != nil {
			continue
		}
		result = append(result, matchID)
	}
	return result, rows.Err()
}

// findMatchesInSharedDB — détection participants-only dans shared DB.
// Portage de _find_matches_in_shared_db() (detection.py).
func findMatchesInSharedDB(
	ctx context.Context,
	sharedDB *sql.DB,
	xuid string,
	scope *SyncScope,
) ([]string, error) {
	var conditions []string
	hasBFCol := hasBackfillCompletedColumn(ctx, sharedDB)

	// Participants scores/rank
	if scope.ParticipantsScores {
		conditions = append(conditions,
			"(mp.score IS NULL OR mp.rank IS NULL)"+doneGuard("participants_scores", hasBFCol))
	}

	// Participants K/D/A
	if scope.ParticipantsKDA {
		conditions = append(conditions,
			"(mp.kills IS NULL OR mp.deaths IS NULL OR mp.assists IS NULL)"+
				doneGuard("participants_kda", hasBFCol))
	}

	// Participants shots
	if scope.ParticipantsShots {
		if scope.ForceParticipantsShots {
			conditions = append(conditions, "1=1")
		} else {
			conditions = append(conditions,
				"(mp.shots_fired IS NULL OR mp.shots_hit IS NULL)"+
					doneGuard("participants_shots", hasBFCol))
		}
	}

	// Participants damage
	if scope.ParticipantsDamage {
		if scope.ForceParticipantsDamage {
			conditions = append(conditions, "1=1")
		} else {
			conditions = append(conditions,
				"(mp.damage_dealt IS NULL OR mp.damage_taken IS NULL)"+
					doneGuard("participants_damage", hasBFCol))
		}
	}

	// Participants avg_life_seconds
	if scope.ParticipantsAvgLife {
		if scope.ForceParticipantsAvgLife {
			conditions = append(conditions, "1=1")
		} else {
			conditions = append(conditions,
				"mp.avg_life_seconds IS NULL"+doneGuard("participants_avg_life", hasBFCol))
		}
	}

	if len(conditions) == 0 {
		return nil, nil
	}

	whereClause := strings.Join(conditions, " OR ")

	query := fmt.Sprintf(`
		SELECT DISTINCT mp.match_id
		FROM match_participants mp
		JOIN match_registry mr ON mr.match_id = mp.match_id
		WHERE mp.xuid = ? AND (%s)
		ORDER BY mr.end_time DESC
	`, whereClause)
	if scope.MaxMatches > 0 {
		query += fmt.Sprintf(" LIMIT %d", scope.MaxMatches)
	}

	rows, err := sharedDB.QueryContext(ctx, query, xuid)
	if err != nil {
		slog.WarnContext(ctx, "backfill: détection shared DB échouée", "err", err)
		return nil, nil
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var matchID string
		if err := rows.Scan(&matchID); err != nil {
			continue
		}
		result = append(result, matchID)
	}
	return result, rows.Err()
}

// isValidMatchID vérifie qu'un match_id est un UUID Halo valide (hex + tirets).
// Empêche toute injection SQL via des IDs corrompus.
func isValidMatchID(id string) bool {
	if len(id) == 0 || len(id) > 64 {
		return false
	}
	for _, c := range id {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') && c != '-' {
			return false
		}
	}
	return true
}
