// Package duckdb — home_repo_matches.go : chargement des matchs Home (Q26) +
// sessions (Q27) + médias récents (Q28) + count total (Q26b).
//
// Sous-module de home_repo.go (split god-file 2026-05-21).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

// LoadHomeMatches charge tous les matchs du joueur (Q26).
//
// Phase 3.bis plan stabilisation 2026-05-22 : split en 2 phases Go-side pour
// éliminer le mix cross-DB (player + shared) qui forçait l'ATTACH shared sur
// la player conn — interdit depuis ADR 0016.
//
//   - Phase A : Q26HomeMatchesSharedPart via SharedReader → toutes les colonnes
//     shared + tri chronologique + LIMIT 150. Source de vérité de la liste.
//   - Phase B : Q26HomeMatchesPlayerEnrichTpl sur pdb.Player → enrichissement
//     pme + msr pour les match_ids retournés par Phase A.
//   - Merge : map[match_id]playerEnrich + iteration Go-side, calcul du
//     skill_rating_type via CASE Go (cf. Q26 SQL original).
func (r *HomeRepo) LoadHomeMatches(ctx context.Context) ([]legacymatch.HomeMatchRow, error) {
	if r.pdb.SharedReader == nil {
		return nil, nil
	}
	// ── Phase A — shared ───────────────────────────────────────────────────
	result, err := r.loadHomeMatchesSharedPart(ctx)
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return result, nil
	}
	// ── Phase B — player enrichment ────────────────────────────────────────
	matchIDs := make([]string, 0, len(result))
	for i := range result {
		matchIDs = append(matchIDs, result[i].MatchID)
	}
	enrichByMatchID, err := r.loadHomeMatchesPlayerEnrich(ctx, matchIDs)
	if err != nil {
		// Best-effort : log + dégradation gracieuse (sans pme/msr).
		slog.WarnContext(ctx, "LoadHomeMatches: player enrich phase failed, dégradation gracieuse",
			"xuid", r.pdb.XUID, "matches", len(matchIDs), "err", err)
		enrichByMatchID = nil
	}
	// ── Merge Go-side ──────────────────────────────────────────────────────
	for i := range result {
		row := &result[i]
		applyHomeMatchPlayerEnrich(row, enrichByMatchID[row.MatchID])
		row.SkillRatingType = resolveHomeMatchSkillRatingType(*row, enrichByMatchID[row.MatchID])
		tier := ""
		if row.SkillTier != nil {
			tier = *row.SkillTier
		}
		tierLabel := ""
		if row.SkillTierLabel != nil {
			tierLabel = *row.SkillTierLabel
		}
		row.SkillRankImageURL = buildHomeSkillPeakBadgeURL(tier, tierLabel, row.SkillSubTier, r.titleSlug(), 0)
	}

	r.enrichHomeMatchTranslations(ctx, result)
	return result, nil
}

// loadHomeMatchesSharedPart : Phase A — query shared via SharedReader. Retourne
// les rows partielles (champs shared remplis, champs player vides → enrichis
// par Phase B).
func (r *HomeRepo) loadHomeMatchesSharedPart(ctx context.Context) ([]legacymatch.HomeMatchRow, error) {
	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("LoadHomeMatches sharedReader: %w", err)
	}
	defer release()

	// Set perfect-kill résolu pour le titre du joueur (HINF byte-identique).
	// J7 : 3 params xuid — base (fenêtre 150), perfect (médailles bornées), principale.
	q := resolvePerfectKillClause(Q26HomeMatchesSharedPart, "medal_name_id", r.titleSlug())
	// Masquage read-side des matchs Campagne (Halo 5) dans la fenêtre CTE `base`.
	q = resolveCampaignExclusion(q, r.titleSlug(), "r")
	rows, err := sharedDB.QueryContext(ctx, q, r.pdb.XUID, r.pdb.XUID, r.pdb.XUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []legacymatch.HomeMatchRow
	for rows.Next() {
		var row legacymatch.HomeMatchRow
		if err := rows.Scan(
			&row.MatchID,
			&row.StartTime,
			&row.MapID,
			&row.MapName,
			&row.MapNameFR,
			&row.PairID,
			&row.PairName,
			&row.PairNameFR,
			&row.GameVariantID,
			&row.GameVariantName,
			&row.PlaylistID,
			&row.PlaylistName,
			&row.PlaylistNameFR,
			&row.IsFirefight,
			&row.IsRanked,
			&row.Outcome,
			&row.TeamID,
			&row.Team0Score,
			&row.Team1Score,
			&row.Kills,
			&row.Deaths,
			&row.Assists,
			&row.KDA,
			&row.Ratio,
			&row.Accuracy,
			&row.AvgLifeSeconds,
			&row.TimePlayedSecs,
			&row.DamageDealt,
			&row.DamageTaken,
			&row.TeamMMR,
			&row.EnemyMMR,
			&row.RankInTeam,
			&row.HeadshotKills,
			&row.PerfectKills,
			&row.MaxKillingSpree,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// homeMatchPlayerEnrich : projection Phase B (pme + msr).
type homeMatchPlayerEnrich struct {
	sessionLabel     sql.NullString
	isWithFriends    bool
	dominanceFlag    int
	performanceScore sql.NullFloat64
	ratingType       sql.NullString
	ratingValue      sql.NullFloat64
	tier             sql.NullString
	subTier          int
	tierLabel        sql.NullString
	ratingDelta      sql.NullFloat64
	playlistGroup    sql.NullString
}

// loadHomeMatchesPlayerEnrich : Phase B — query player conn pour pme + msr.
func (r *HomeRepo) loadHomeMatchesPlayerEnrich(ctx context.Context, matchIDs []string) (map[string]*homeMatchPlayerEnrich, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(matchIDs))
	args := make([]interface{}, 0, len(matchIDs))
	for i, id := range matchIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	query := fmt.Sprintf(Q26HomeMatchesPlayerEnrichTpl, strings.Join(placeholders, ", "))

	rows, err := r.pdb.ReadDB().QueryRecovered(ctx, query, args...)
	if err != nil {
		// Table pme peut être absente sur DB fraîche → dégradation gracieuse.
		if isTableNotFoundErr(err) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]*homeMatchPlayerEnrich, len(matchIDs))
	for rows.Next() {
		var (
			matchID string
			e       homeMatchPlayerEnrich
		)
		if err := rows.Scan(
			&matchID,
			&e.sessionLabel,
			&e.isWithFriends,
			&e.dominanceFlag,
			&e.performanceScore,
			&e.ratingType,
			&e.ratingValue,
			&e.tier,
			&e.subTier,
			&e.tierLabel,
			&e.ratingDelta,
			&e.playlistGroup,
		); err != nil {
			return nil, err
		}
		out[matchID] = &e
	}
	return out, rows.Err()
}

// applyHomeMatchPlayerEnrich applique les champs Phase B sur une ligne Phase A.
// Si enrich est nil (joueur jamais ingéré pour ce match), tous les champs
// restent à leur valeur zéro / nil — le frontend dégrade gracieusement.
func applyHomeMatchPlayerEnrich(row *legacymatch.HomeMatchRow, enrich *homeMatchPlayerEnrich) {
	if enrich == nil {
		return
	}
	if enrich.sessionLabel.Valid {
		s := enrich.sessionLabel.String
		row.SessionLabel = &s
	}
	row.IsWithFriends = enrich.isWithFriends
	row.DominanceFlag = enrich.dominanceFlag
	if enrich.performanceScore.Valid {
		v := enrich.performanceScore.Float64
		row.PerformanceScore = &v
	}
	if enrich.ratingValue.Valid {
		v := enrich.ratingValue.Float64
		row.SkillRatingValue = &v
	}
	if enrich.tier.Valid {
		s := enrich.tier.String
		row.SkillTier = &s
	}
	row.SkillSubTier = enrich.subTier
	if enrich.tierLabel.Valid {
		s := enrich.tierLabel.String
		row.SkillTierLabel = &s
	}
	if enrich.ratingDelta.Valid {
		v := enrich.ratingDelta.Float64
		row.SkillRatingDelta = &v
	}
	if enrich.playlistGroup.Valid {
		s := enrich.playlistGroup.String
		row.SkillPlaylistGroup = &s
	}
}

// resolveHomeMatchSkillRatingType : équivalent Go de la CASE SQL Q26 originale.
//
//	WHEN is_ranked OR playlist_name contains 'ranked' OR pair_name contains 'ranked' → 'CSR'
//	WHEN UPPER(TRIM(msr.rating_type)) = 'CSR' → 'CSR'
//	ELSE 'LUSR'
func resolveHomeMatchSkillRatingType(row legacymatch.HomeMatchRow, enrich *homeMatchPlayerEnrich) string {
	if row.IsRanked ||
		strings.Contains(strings.ToLower(row.PlaylistName), "ranked") ||
		strings.Contains(strings.ToLower(row.PairName), "ranked") {
		return "CSR"
	}
	if enrich != nil && enrich.ratingType.Valid {
		rt := strings.ToUpper(strings.TrimSpace(enrich.ratingType.String))
		if rt == ratingTypeCSR {
			return ratingTypeCSR
		}
	}
	return ratingTypeLUSR
}

// CountPlayerMatches retourne le nombre total de matchs du joueur (Q26b).
//
// Phase 3 plan stabilisation 2026-05-22 : migré de pdb.ReadDB() (player conn)
// vers SharedReader.Get(). match_participants vit dans shared_matches_v2 et
// l'ATTACH shared sur la player conn a été retiré au sprint P0→P7 (ADR 0016).
// La query référence désormais `match_participants` sans préfixe `shared.`
// (cf. Q26bCountPlayerMatches dans queries_home_citations.go).
func (r *HomeRepo) CountPlayerMatches(ctx context.Context) (int, error) {
	if r.pdb.SharedReader == nil {
		return 0, nil
	}
	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return 0, err
	}
	defer release()
	var count int
	q := resolveCampaignExclusion(Q26bCountPlayerMatches, r.titleSlug(), "r")
	err = sharedDB.QueryRowContext(ctx, q, r.pdb.XUID).Scan(&count)
	return count, err
}

// LoadHomeSessions charge les sessions avec label depuis player_match_enrichment.
//
// Phase 3.bis plan stabilisation 2026-05-22 : split en 2 phases Go-side pour
// éliminer le mix cross-DB (player + shared).
//
//   - Phase A : Q27HomeSessionsPlayerPart sur pdb.Player → match_id, session_id,
//     session_label, is_with_friends pour tous les pme avec label non-NULL.
//   - Phase B : Q27HomeSessionsSharedStartTimesTpl sur SharedReader →
//     start_time pour le lot de match_ids retourné par Phase A.
//   - Merge + sort by start_time DESC Go-side.
func (r *HomeRepo) LoadHomeSessions(ctx context.Context) ([]legacymatch.HomeSessionRow, error) {
	// ── Phase A — player conn ──────────────────────────────────────────────
	rows, err := r.pdb.ReadDB().QueryRecovered(ctx, Q27HomeSessionsPlayerPart)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []legacymatch.HomeSessionRow
	matchIDs := make([]string, 0)
	for rows.Next() {
		var row legacymatch.HomeSessionRow
		if err := rows.Scan(
			&row.MatchID,
			&row.SessionID,
			&row.SessionLabel,
			&row.IsWithFriends,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
		matchIDs = append(matchIDs, row.MatchID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return result, nil
	}

	// ── Phase B — shared conn pour start_time ──────────────────────────────
	startTimeByMatchID := r.loadHomeSessionsStartTimes(ctx, matchIDs)

	// ── Merge ──────────────────────────────────────────────────────────────
	for i := range result {
		if st, ok := startTimeByMatchID[result[i].MatchID]; ok {
			tCopy := st
			result[i].StartTime = &tCopy
		}
	}
	// Sort by StartTime DESC (nil StartTime → fin).
	sort.SliceStable(result, func(i, j int) bool {
		a, b := result[i].StartTime, result[j].StartTime
		switch {
		case a == nil && b == nil:
			return false
		case a == nil:
			return false
		case b == nil:
			return true
		default:
			return a.After(*b)
		}
	})
	return result, nil
}

// loadHomeSessionsStartTimes : Phase B helper. Retourne un map vide en cas
// d'erreur (dégradation gracieuse — les rows sans start_time iront en fin
// de liste après le tri).
func (r *HomeRepo) loadHomeSessionsStartTimes(ctx context.Context, matchIDs []string) map[string]time.Time {
	out := make(map[string]time.Time, len(matchIDs))
	if len(matchIDs) == 0 || r.pdb.SharedReader == nil {
		return out
	}
	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		slog.WarnContext(ctx, "LoadHomeSessions: shared reader unavailable for start_time enrichment",
			"xuid", r.pdb.XUID, "err", err)
		return out
	}
	defer release()

	placeholders := make([]string, len(matchIDs))
	args := make([]interface{}, 0, len(matchIDs))
	for i, id := range matchIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	query := fmt.Sprintf(Q27HomeSessionsSharedStartTimesTpl, strings.Join(placeholders, ", "))
	rows, err := sharedDB.QueryContext(ctx, query, args...)
	if err != nil {
		slog.WarnContext(ctx, "LoadHomeSessions: shared query failed",
			"xuid", r.pdb.XUID, "err", err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var (
			matchID string
			st      sql.NullTime
		)
		if err := rows.Scan(&matchID, &st); err != nil {
			continue
		}
		if st.Valid {
			out[matchID] = st.Time
		}
	}
	return out
}

// LoadRecentMedia charge les médias récents du joueur (Q28).
// Retourne une liste vide si la table media_files n'existe pas.
//
// Phase 3 plan stabilisation 2026-05-22 : migré de pdb.ReadDB() (player conn)
// vers pdb.SharedSocial. La table media_files (et media_match_associations)
// a été déplacée vers shared_social.duckdb via migration
// drop_media_from_player_db (cf. steps_player.go:370). Avant ce fix, la
// query lançait "Catalog Error: Table with name media_files does not exist"
// sur la player conn — cf. AUDIT_DUCKDB_ATTACH_2026-05-21 §2.
func (r *HomeRepo) LoadRecentMedia(ctx context.Context, limit int) ([]domain.HomeMediaRow, error) {
	if r.pdb.SharedSocial == nil {
		return nil, nil
	}
	rows, err := r.pdb.SharedSocial.Query(ctx, Q28RecentMedia, limit)
	if err != nil {
		// La table media_files peut ne pas exister — dégradation silencieuse.
		if isTableNotFoundErr(err) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	var result []domain.HomeMediaRow
	for rows.Next() {
		var row domain.HomeMediaRow
		var matchID sql.NullString
		var matchStartTime sql.NullTime
		if err := rows.Scan(&row.FileName, &matchID, &matchStartTime); err != nil {
			return nil, err
		}
		if matchID.Valid {
			row.MatchID = &matchID.String
		}
		if matchStartTime.Valid {
			row.MatchStartTime = &matchStartTime.Time
		}
		result = append(result, row)
	}
	return result, rows.Err()
}
