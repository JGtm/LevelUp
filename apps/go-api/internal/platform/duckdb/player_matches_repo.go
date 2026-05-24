// Package duckdb — player_matches_repo.go : implementation DuckDB du loader
// unifie des matchs joueur (port.PlayerMatchesRepository).
//
// Per-player : un PlayerMatchesRepo est lie a un PlayerDB precis. La resolution
// (slug, gamertag) -> PlayerDB se fait via pool.GetOrOpen au niveau de
// l'adapter qui consomme le repo (chunk ulterieur).
//
// Capability gating : laisse au service appelant pour cette implementation.
// Le repo execute la requete telle quelle ; si le titre n'a pas la capability
// "match.history", c'est au service de retourner games.ErrCapabilityNotSupported
// avant d'appeler le repo.
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// PlayerMatchesRepo charge les matchs d'un joueur depuis sa DB stats avec
// shared attache. Implemente une variante per-player de
// port.PlayerMatchesRepository (le slug et le gamertag sont fixes par le
// PlayerDB injecte au constructeur).
type PlayerMatchesRepo struct {
	pdb *PlayerDB
}

// NewPlayerMatchesRepo cree un PlayerMatchesRepo lie a un PlayerDB.
func NewPlayerMatchesRepo(pdb *PlayerDB) *PlayerMatchesRepo {
	return &PlayerMatchesRepo{pdb: pdb}
}

// Load charge les matchs du joueur en suivant les filtres fournis. Retourne
// les rows projetees en canonical.PlayerMatchRow. Trie par r.start_time DESC
// par defaut, override possible via filters.OrderBy (whitelist).
//
// L'appelant doit avoir valide les filtres via filters.Validate() en amont.
// Le repo re-applique aussi sa propre validation defensive (input untrusted).
//
// split+merge cross-DB.
//
//	Étape 1 (SharedReader) : query shared (v_match_full ⨝ match_participants ⨝
//	subquery medals_earned) avec tous les filtres shared (Period, Outcome,
//	IsFirefight, IsRanked, MinTimePlayed, BTBExcluded, PlaylistKind, MapIDs,
//	ExcludeFriendsXUIDs) + ORDER BY si tri sur colonne shared.
//	Étape 2 (pdb.Player) : player_match_enrichment WHERE match_id IN (...)
//	Étape 3 (pdb.Player) : match_skill_rank WHERE match_id IN (...)
//	Étape 4 (Go) : merge LEFT JOIN, application HadBotTeammate filter post-merge,
//	re-tri sur performance_score (PME) si nécessaire, LIMIT.
func (r *PlayerMatchesRepo) Load(
	ctx context.Context,
	filters port.PlayerMatchFilters,
) ([]canonical.PlayerMatchRow, error) {
	if err := filters.Validate(); err != nil {
		return nil, fmt.Errorf("PlayerMatchesRepo.Load: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Étape 1 : query shared.
	sharedResults, err := r.loadSharedRows(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("PlayerMatchesRepo.Load: %w", err)
	}
	if len(sharedResults) == 0 {
		return nil, nil
	}

	matchIDs := make([]string, 0, len(sharedResults))
	for i := range sharedResults {
		matchIDs = append(matchIDs, sharedResults[i].matchID)
	}

	// Étape 2 + 3 : enrichments + skill ranks.
	enrichments, err := r.loadEnrichmentsForMatches(ctx, matchIDs)
	if err != nil {
		return nil, fmt.Errorf("PlayerMatchesRepo.Load: %w", err)
	}
	skillRanks, err := r.loadSkillRanksForMatches(ctx, matchIDs)
	if err != nil {
		return nil, fmt.Errorf("PlayerMatchesRepo.Load: %w", err)
	}

	// Étape 4 : merge + filtres player + tri PME + LIMIT.
	out := r.mergePlayerMatchRows(sharedResults, enrichments, skillRanks, filters)
	return out, nil
}

// buildSharedQuery compose la partie shared (v_match_full ⨝ match_participants
// + subquery medals_earned) avec les filtres applicables shared-only.
// HadBotTeammate (PME) reste filtré côté Go après merge.
//
// ORDER BY : si sur colonne shared (start_time), ajouté ici + LIMIT propagé.
// Si sur colonne player (performance_score), order/limit appliqués post-merge.
func (r *PlayerMatchesRepo) buildSharedQuery(f port.PlayerMatchFilters) (string, []any, sharedQueryHints, error) {
	var sb strings.Builder
	sb.WriteString(playerMatchesSharedBaseSelect)

	args := []any{r.pdb.XUID}

	appendPlayerMatchScalarFilters(&sb, &args, f)
	if err := appendPlayerMatchSetFilters(&sb, &args, f); err != nil {
		return "", nil, sharedQueryHints{}, err
	}

	hints, orderBy, err := classifyOrderBy(f.OrderBy)
	if err != nil {
		return "", nil, sharedQueryHints{}, err
	}
	sb.WriteString(" ORDER BY ")
	sb.WriteString(orderBy)

	// LIMIT côté SQL uniquement si ORDER BY shared ET pas de filtre PME.
	// Sinon (PME order ou filtre HadBotTeammate), récupère tout et applique en Go.
	if hints.canPushLimit && f.Limit > 0 && f.HadBotTeammate == nil {
		sb.WriteString(" LIMIT ?")
		args = append(args, f.Limit)
	}

	return sb.String(), args, hints, nil
}

// appendPlayerMatchScalarFilters ajoute les filtres scalaires (Period, IsFirefight,
// IsRanked, MinTimePlayedSeconds, BTBExcluded).
func appendPlayerMatchScalarFilters(sb *strings.Builder, args *[]any, f port.PlayerMatchFilters) {
	if since := periodSince(f.Period); since != nil {
		sb.WriteString(" AND r.start_time >= ?")
		*args = append(*args, *since)
	}
	if f.IsFirefight != nil {
		sb.WriteString(" AND COALESCE(r.is_firefight, FALSE) = ?")
		*args = append(*args, *f.IsFirefight)
	}
	if f.IsRanked != nil {
		sb.WriteString(` AND (CASE
			WHEN COALESCE(r.is_ranked, FALSE)
				OR STRPOS(LOWER(COALESCE(r.playlist_name, '')), 'ranked') > 0
				OR STRPOS(LOWER(COALESCE(r.pair_name, '')), 'ranked') > 0
			THEN TRUE ELSE FALSE END) = ?`)
		*args = append(*args, *f.IsRanked)
	}
	if f.MinTimePlayedSeconds != nil {
		sb.WriteString(" AND COALESCE(p.time_played_seconds, 0) >= ?")
		*args = append(*args, *f.MinTimePlayedSeconds)
	}
	if f.BTBExcluded {
		sb.WriteString(" AND (r.pair_name IS NULL OR LOWER(r.pair_name) NOT LIKE '%btb%')")
	}
}

// appendPlayerMatchSetFilters ajoute les filtres IN (OutcomeIn, ExcludeFriendsXUIDs,
// MapIDs) et PlaylistKind. Peut retourner une erreur sur PlaylistKind invalide.
func appendPlayerMatchSetFilters(sb *strings.Builder, args *[]any, f port.PlayerMatchFilters) error {
	if len(f.OutcomeIn) > 0 {
		placeholders := make([]string, 0, len(f.OutcomeIn))
		for _, o := range f.OutcomeIn {
			placeholders = append(placeholders, "?")
			*args = append(*args, outcomeToInt(o))
		}
		fmt.Fprintf(sb, " AND COALESCE(p.outcome, 0) IN (%s)",
			strings.Join(placeholders, ","))
	}
	if len(f.ExcludeFriendsXUIDs) > 0 {
		placeholders := make([]string, 0, len(f.ExcludeFriendsXUIDs))
		for _, x := range f.ExcludeFriendsXUIDs {
			placeholders = append(placeholders, "?")
			*args = append(*args, x)
		}
		fmt.Fprintf(sb, " AND p.match_id NOT IN (SELECT match_id FROM match_participants WHERE xuid IN (%s))",
			strings.Join(placeholders, ","))
	}
	if f.PlaylistKind != nil {
		clause, err := playlistKindClause(*f.PlaylistKind)
		if err != nil {
			return err
		}
		if clause != "" {
			sb.WriteString(" AND ")
			sb.WriteString(clause)
		}
	}
	if len(f.MapIDs) > 0 {
		placeholders := make([]string, 0, len(f.MapIDs))
		for _, id := range f.MapIDs {
			placeholders = append(placeholders, "?")
			*args = append(*args, id)
		}
		fmt.Fprintf(sb, " AND COALESCE(r.map_id, '') IN (%s)",
			strings.Join(placeholders, ","))
	}
	return nil
}

// sharedQueryHints regroupe les hints sur le découpage ORDER BY + LIMIT entre
// SQL et Go.
type sharedQueryHints struct {
	canPushLimit  bool // ORDER BY est sur colonne shared, LIMIT peut être SQL
	postMergeSort string
}

// classifyOrderBy détermine si l'ORDER BY peut s'appliquer côté SQL (shared col)
// ou doit s'appliquer post-merge en Go (PME col). Retourne aussi la clause SQL
// à utiliser (vers shared cols seulement).
func classifyOrderBy(s string) (sharedQueryHints, string, error) {
	switch strings.TrimSpace(s) {
	case "", "start_time DESC":
		return sharedQueryHints{canPushLimit: true}, "r.start_time DESC", nil
	case "start_time ASC":
		return sharedQueryHints{canPushLimit: true}, "r.start_time ASC", nil
	case "performance_score DESC":
		// Tri post-merge. SQL garde un ordre stable mais non significatif.
		return sharedQueryHints{canPushLimit: false, postMergeSort: "performance_score DESC"},
			"r.start_time DESC", nil
	case "performance_score ASC":
		return sharedQueryHints{canPushLimit: false, postMergeSort: "performance_score ASC"},
			"r.start_time DESC", nil
	}
	return sharedQueryHints{}, "", fmt.Errorf("%w: %q", ErrUnknownOrderBy, s)
}

// playerMatchesSharedBaseSelect : (ADR 0016) — partie shared du split
// PlayerMatchesRepo.Load. Toutes les tables/vues référencées sont au niveau root
// du catalogue shared_matches_v2.duckdb (pas de préfixe `shared.`).
//
// 39 colonnes : match metadata + participant stats + team_id + team_0/1_score
// + perfect_kills (subquery sur medals_earned). Les colonnes PME (session,
// performance, dominance, had_bot, is_with_friends) et match_skill_rank (tier,
// rating, etc.) sont hydratées en étape 2/3 (cf. mergePlayerMatchRows).
//
// Bug #2/#7 : on ne fallback PAS sur l'EN dans la projection FR. Si NULL en
// DB, on renvoie chaîne vide ; HomeRepo.EnrichCanonicalAssetTranslations
// remplit ensuite Labels["fr"] depuis metadata.asset_translations.
//
// Bug #3 : projeter damage_dealt / damage_taken pour ComputeCombatYield.
const playerMatchesSharedBaseSelect = `
SELECT
    p.match_id,
    r.start_time,
    COALESCE(r.duration_seconds, 0)                   AS duration_seconds,
    COALESCE(r.map_id, '')                            AS map_id,
    COALESCE(r.map_name, '')                          AS map_name,
    COALESCE(r.map_name_fr, '')                       AS map_name_fr,
    COALESCE(r.playlist_id, '')                       AS playlist_id,
    COALESCE(r.playlist_name, '')                     AS playlist_name,
    COALESCE(r.playlist_name_fr, '')                  AS playlist_name_fr,
    COALESCE(r.game_variant_id, '')                   AS variant_id,
    COALESCE(r.game_variant_name, '')                 AS variant_name,
    COALESCE(r.pair_id, '')                           AS pair_id,
    COALESCE(r.pair_name, '')                         AS pair_name,
    COALESCE(r.pair_name_fr, '')                      AS pair_name_fr,
    CASE
        WHEN COALESCE(r.is_ranked, FALSE)
            OR STRPOS(LOWER(COALESCE(r.playlist_name, '')), 'ranked') > 0
            OR STRPOS(LOWER(COALESCE(r.pair_name, '')), 'ranked') > 0
        THEN TRUE ELSE FALSE
    END                                                  AS is_ranked,
    COALESCE(r.is_firefight, FALSE)                   AS is_firefight,
    COALESCE(p.team_id, 0)                            AS team_id,
    p.outcome                                         AS outcome_code,
    COALESCE(p.kills, 0)                              AS kills,
    COALESCE(p.deaths, 0)                             AS deaths,
    COALESCE(p.assists, 0)                            AS assists,
    p.kda,
    COALESCE(p.headshot_kills, 0)                     AS headshot_kills,
    p.accuracy,
    COALESCE(p.time_played_seconds, 0)                AS time_played_seconds,
    p.avg_life_seconds,
    p.damage_dealt,
    p.damage_taken,
    p.team_mmr,
    p.enemy_mmr,
    COALESCE(r.team_0_score, -1)                      AS team_0_score,
    COALESCE(r.team_1_score, -1)                      AS team_1_score,
    p.max_killing_spree,
    p.personal_score,
    p.rank                                               AS rank_in_match,
    p.grenade_kills,
    p.melee_kills,
    p.power_weapon_kills,
    p.shots_fired,
    p.shots_hit,
    COALESCE((
        SELECT SUM(me.count)
        FROM medals_earned me
        WHERE me.match_id = p.match_id
          AND me.xuid = p.xuid
          AND me.medal_name_id = 1512363953
    ), 0)::INTEGER                                       AS perfect_kills
FROM match_participants p
JOIN v_match_full r ON r.match_id = p.match_id
WHERE p.xuid = ?`

// loadSharedRows exécute l'étape 1 du split (query shared) et retourne les
// playerMatchScanResult partiellement remplis (cols shared seulement).
func (r *PlayerMatchesRepo) loadSharedRows(ctx context.Context, filters port.PlayerMatchFilters) ([]playerMatchScanResult, error) {
	q, args, _, err := r.buildSharedQuery(filters)
	if err != nil {
		return nil, fmt.Errorf("build shared query: %w", err)
	}

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("shared reader: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("shared query: %w", err)
	}
	defer rows.Close()

	var results []playerMatchScanResult
	for rows.Next() {
		s, err := scanSharedPlayerMatchRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan shared row: %w", err)
		}
		s.xuid = r.pdb.XUID
		s.gamertag = r.pdb.Gamertag
		results = append(results, s)
	}
	return results, rows.Err()
}

// loadEnrichmentsForMatches récupère player_match_enrichment via le helper
// partagé LoadPlayerMatchEnrichments (commit 9d.4).
func (r *PlayerMatchesRepo) loadEnrichmentsForMatches(ctx context.Context, matchIDs []string) (map[string]MatchEnrichment, error) {
	return LoadPlayerMatchEnrichments(ctx, r.pdb.Player, matchIDs)
}

// loadSkillRanksForMatches récupère match_skill_rank pour la liste de match_ids
// (étape 3 du split). Retourne une map indexée par match_id.
func (r *PlayerMatchesRepo) loadSkillRanksForMatches(ctx context.Context, matchIDs []string) (map[string]playerMatchSkillRankRow, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}
	query := fmt.Sprintf(playerMatchesSkillRankTpl, Placeholders(len(matchIDs)))
	rows, err := r.pdb.Player.Query(ctx, query, ToAnySlice(matchIDs)...)
	if err != nil {
		return nil, fmt.Errorf("skill_rank query: %w", err)
	}
	defer rows.Close()

	out := make(map[string]playerMatchSkillRankRow, len(matchIDs))
	for rows.Next() {
		var (
			mid string
			s   playerMatchSkillRankRow
		)
		if err := rows.Scan(&mid, &s.ratingType, &s.ratingValue, &s.tier,
			&s.tierFR, &s.subTier, &s.delta, &s.playlistGroup); err != nil {
			return nil, fmt.Errorf("skill_rank scan: %w", err)
		}
		out[mid] = s
	}
	return out, rows.Err()
}

// mergePlayerMatchRows assemble les rows finaux depuis les 3 sources :
// shared + enrichments + skill_ranks. Applique le filtre HadBotTeammate (player)
// post-merge, le re-tri éventuel par performance_score, et le LIMIT.
func (r *PlayerMatchesRepo) mergePlayerMatchRows(
	shared []playerMatchScanResult,
	enrichments map[string]MatchEnrichment,
	skillRanks map[string]playerMatchSkillRankRow,
	filters port.PlayerMatchFilters,
) []canonical.PlayerMatchRow {
	// Hydrate les cols player dans chaque playerMatchScanResult.
	for i := range shared {
		if e, ok := enrichments[shared[i].matchID]; ok {
			shared[i].sessionID = e.SessionID
			shared[i].sessionLabel = e.SessionLabel
			shared[i].performanceScore = e.PerformanceScore
			shared[i].dominanceFlag = e.DominanceFlag
			shared[i].hadBotTeammate = e.HadBotTeammate
			shared[i].isWithFriends = e.IsWithFriends
			shared[i].engagementScoreBrut = e.EngagementScoreBrut
		}
		if s, ok := skillRanks[shared[i].matchID]; ok {
			shared[i].skillRatingType = s.ratingType
			shared[i].skillRatingValue = s.ratingValue
			shared[i].skillTier = s.tier
			shared[i].skillTierFR = s.tierFR
			shared[i].skillSubTier = s.subTier
			shared[i].skillDelta = s.delta
			shared[i].skillPlaylistGroup = s.playlistGroup
		}
	}

	// Filtre HadBotTeammate (player-only, post-merge).
	if filters.HadBotTeammate != nil {
		want := *filters.HadBotTeammate
		filtered := shared[:0]
		for _, s := range shared {
			if s.hadBotTeammate == want {
				filtered = append(filtered, s)
			}
		}
		shared = filtered
	}

	// Re-tri sur performance_score si demandé (l'ordre SQL était sur start_time).
	hints, _, _ := classifyOrderBy(filters.OrderBy)
	switch hints.postMergeSort {
	case "performance_score DESC":
		sortByPerformanceScore(shared, true)
	case "performance_score ASC":
		sortByPerformanceScore(shared, false)
	}

	// LIMIT post-merge si non poussé en SQL.
	if filters.Limit > 0 && (!hints.canPushLimit || filters.HadBotTeammate != nil) {
		if len(shared) > filters.Limit {
			shared = shared[:filters.Limit]
		}
	}

	out := make([]canonical.PlayerMatchRow, 0, len(shared))
	for _, s := range shared {
		out = append(out, projectPlayerMatchRow(s))
	}
	return out
}

// scanSharedPlayerMatchRow scanne la partie shared (39 cols) en
// playerMatchScanResult (cols PME/skill_rank/identité restent zero).
func scanSharedPlayerMatchRow(rows *sql.Rows) (playerMatchScanResult, error) {
	var s playerMatchScanResult
	if err := rows.Scan(
		&s.matchID, &s.startTime, &s.durationSeconds,
		&s.mapID, &s.mapName, &s.mapNameFR,
		&s.playlistID, &s.playlistName, &s.playlistNameFR,
		&s.variantID, &s.variantName,
		&s.pairID, &s.pairName, &s.pairNameFR,
		&s.isRanked, &s.isFirefight,
		&s.teamID, &s.outcomeCode,
		&s.kills, &s.deaths, &s.assists,
		&s.kda, &s.headshotKills, &s.accuracy,
		&s.timePlayedSeconds, &s.avgLifeSeconds, &s.damageDealt, &s.damageTaken,
		&s.teamMMR, &s.enemyMMR,
		&s.team0Score, &s.team1Score,
		&s.maxKillingSpree, &s.personalScore, &s.rankInMatch,
		&s.grenadeKills, &s.meleeKills, &s.powerWeaponKills,
		&s.shotsFired, &s.shotsHit,
		&s.perfectKills,
	); err != nil {
		return playerMatchScanResult{}, err
	}
	return s, nil
}

// playerMatchSkillRankRow porte les colonnes match_skill_rank chargées en
// étape 3. (playerMatchEnrichmentRow + playerMatchesEnrichmentTpl retirés au
// commit 9d.4 — remplacés par MatchEnrichment + LoadPlayerMatchEnrichments).
type playerMatchSkillRankRow struct {
	ratingType    sql.NullString
	ratingValue   sql.NullFloat64
	tier          sql.NullString
	tierFR        sql.NullString
	subTier       sql.NullInt64
	delta         sql.NullFloat64
	playlistGroup sql.NullString
}

// playerMatchesSkillRankTpl : SQL pour l'étape 3 du split (match_skill_rank
// pour une liste de match_ids).
const playerMatchesSkillRankTpl = `
SELECT
    match_id,
    rating_type,
    rating_value,
    tier,
    tier_fr,
    sub_tier,
    rating_delta,
    playlist_group
FROM match_skill_rank
WHERE match_id IN (%s)`

// sortByPerformanceScore trie les rows par performanceScore (sql.NullFloat64).
// NULLS LAST pour DESC ; NULLS LAST pour ASC aussi (NULL = pas de valeur).
func sortByPerformanceScore(rows []playerMatchScanResult, desc bool) {
	less := func(i, j int) bool {
		a, b := rows[i].performanceScore, rows[j].performanceScore
		if !a.Valid && !b.Valid {
			return false
		}
		if !a.Valid {
			return false // a est NULL → j (b) en premier
		}
		if !b.Valid {
			return true
		}
		if desc {
			return a.Float64 > b.Float64
		}
		return a.Float64 < b.Float64
	}
	// Tri stable pour conserver l'ordre start_time DESC en cas d'égalité.
	sort.SliceStable(rows, less)
}

// playerMatchScanResult agrege les valeurs scannees pour faciliter la
// projection en canonical.PlayerMatchRow.
type playerMatchScanResult struct {
	matchID, mapID, mapName, mapNameFR          string
	playlistID, playlistName, playlistNameFR    string
	variantID, variantName, xuid, gamertag      string
	pairID, pairName, pairNameFR                string
	startTime                                   time.Time
	durationSeconds, teamID                     int
	outcomeCode                                 sql.NullInt64
	kills, deaths, assists, headshotKills       int
	timePlayedSeconds, dominanceFlag            int
	isRanked, isFirefight, hadBotTeammate       bool
	isWithFriends                               bool
	engagementScoreBrut                         sql.NullFloat64
	kda, accuracy, teamMMR, enemyMMR            sql.NullFloat64
	avgLifeSeconds                              sql.NullFloat64
	damageDealt, damageTaken                    sql.NullFloat64
	performanceScore                            sql.NullFloat64
	sessionID                                   sql.NullString // VARCHAR en prod (cf. ADR 0016 / commit 9d.4)
	sessionLabel                                sql.NullString
	team0Score, team1Score                      int
	skillRatingType                             sql.NullString
	skillRatingValue                            sql.NullFloat64
	skillTier                                   sql.NullString
	skillTierFR                                 sql.NullString
	skillSubTier                                sql.NullInt64
	skillDelta                                  sql.NullFloat64
	skillPlaylistGroup                          sql.NullString
	maxKillingSpree, personalScore, rankInMatch sql.NullInt64
	grenadeKills, meleeKills, powerWeaponKills  sql.NullInt64
	shotsFired, shotsHit                        sql.NullInt64
	perfectKills                                sql.NullInt64
}

// projectPlayerMatchRow construit la row canonique depuis les valeurs scannees.
func projectPlayerMatchRow(s playerMatchScanResult) canonical.PlayerMatchRow {
	outcome := projectOutcome(s)
	teams := projectTeamScores(s)
	skillSnap := projectSkillSnapshot(s)
	dmgDealt, dmgTaken := projectDamageStats(s)

	return canonical.PlayerMatchRow{
		Summary:    projectMatchSummary(s, outcome, teams),
		Self:       projectSelfParticipant(s, outcome, dmgDealt, dmgTaken),
		Enrichment: projectEnrichment(s, skillSnap),
	}
}

// projectOutcome retourne l'Outcome canonique. Outcome vide si NULL/0 en DB.
func projectOutcome(s playerMatchScanResult) canonical.Outcome {
	if s.outcomeCode.Valid && s.outcomeCode.Int64 != 0 {
		return outcomeFromInt(int(s.outcomeCode.Int64))
	}
	return ""
}

// projectTeamScores assemble les TeamSnapshot depuis team_0_score / team_1_score.
// Une valeur -1 (COALESCE) signifie absent et est exclue.
func projectTeamScores(s playerMatchScanResult) []canonical.TeamSnapshot {
	var teams []canonical.TeamSnapshot
	if s.team0Score >= 0 {
		score := s.team0Score
		teams = append(teams, canonical.TeamSnapshot{TeamID: 0, Score: &score})
	}
	if s.team1Score >= 0 {
		score := s.team1Score
		teams = append(teams, canonical.TeamSnapshot{TeamID: 1, Score: &score})
	}
	return teams
}

// projectSkillSnapshot extrait le SkillSnapshot depuis match_skill_rank.
// Retourne nil si LEFT JOIN absent.
func projectSkillSnapshot(s playerMatchScanResult) *canonical.SkillSnapshot {
	if !s.skillRatingType.Valid || s.skillRatingType.String == "" {
		return nil
	}
	snap := canonical.SkillSnapshot{
		RatingType:    canonical.RatingType(strings.ToLower(s.skillRatingType.String)),
		RatingValue:   nullFloatPtr(s.skillRatingValue),
		Delta:         nullFloatPtr(s.skillDelta),
		PlaylistGroup: nullStringPtr(s.skillPlaylistGroup),
	}
	if s.skillTier.Valid && s.skillTier.String != "" {
		tier := strings.ToLower(s.skillTier.String)
		snap.TierCode = &tier
	}
	if s.skillTierFR.Valid && s.skillTierFR.String != "" {
		tierFR := s.skillTierFR.String
		snap.TierCodeFR = &tierFR
	}
	if s.skillSubTier.Valid {
		st := int(s.skillSubTier.Int64)
		snap.SubTier = &st
	}
	return &snap
}

// projectDamageStats convertit damage_dealt / damage_taken (DOUBLE en DB) en *int.
func projectDamageStats(s playerMatchScanResult) (*int, *int) {
	var dmgDealt, dmgTaken *int
	if s.damageDealt.Valid {
		v := int(s.damageDealt.Float64)
		dmgDealt = &v
	}
	if s.damageTaken.Valid {
		v := int(s.damageTaken.Float64)
		dmgTaken = &v
	}
	return dmgDealt, dmgTaken
}

// projectMatchSummary projette la section Summary depuis les champs scannés.
func projectMatchSummary(s playerMatchScanResult, outcome canonical.Outcome, teams []canonical.TeamSnapshot) canonical.MatchSummary {
	durationPtr := s.durationSeconds
	return canonical.MatchSummary{
		MatchID:         s.matchID,
		StartedAtUTC:    s.startTime,
		DurationSeconds: &durationPtr,
		MatchType:       matchTypeFromFlags(s.isRanked, s.isFirefight),
		Playlist:        assetReference("playlist", s.playlistID, s.playlistName, s.playlistNameFR),
		Map:             assetReference("map", s.mapID, s.mapName, s.mapNameFR),
		GameVariant:     assetReference("game_variant", s.variantID, s.variantName, ""),
		PairMode:        assetReference("pair_mode", s.pairID, s.pairName, s.pairNameFR),
		IsRanked:        &s.isRanked,
		IsPvE:           &s.isFirefight,
		Outcome:         outcome,
		Teams:           teams,
	}
}

// projectSelfParticipant projette la section Self depuis les champs scannés.
func projectSelfParticipant(s playerMatchScanResult, outcome canonical.Outcome, dmgDealt, dmgTaken *int) canonical.MatchParticipant {
	teamIDPtr := s.teamID
	killsPtr, deathsPtr, assistsPtr := s.kills, s.deaths, s.assists
	headshotPtr := s.headshotKills
	timePlayedPtr := s.timePlayedSeconds
	return canonical.MatchParticipant{
		Identity:         canonical.PlayerIdentity{XUID: s.xuid, Gamertag: s.gamertag},
		TeamID:           &teamIDPtr,
		Outcome:          outcome,
		Kills:            &killsPtr,
		Deaths:           &deathsPtr,
		Assists:          &assistsPtr,
		HeadshotKills:    &headshotPtr,
		KDA:              nullFloatPtr(s.kda),
		Accuracy:         nullFloatPtr(s.accuracy),
		AvgLifeSeconds:   nullFloatPtr(s.avgLifeSeconds),
		TimePlayed:       &timePlayedPtr,
		DamageDealt:      dmgDealt,
		DamageTaken:      dmgTaken,
		MaxKillingSpree:  nullInt64ToIntPtr(s.maxKillingSpree),
		PersonalScore:    nullInt64ToIntPtr(s.personalScore),
		RankInMatch:      nullInt64ToIntPtr(s.rankInMatch),
		GrenadeKills:     nullInt64ToIntPtr(s.grenadeKills),
		MeleeKills:       nullInt64ToIntPtr(s.meleeKills),
		PowerWeaponKills: nullInt64ToIntPtr(s.powerWeaponKills),
		ShotsFired:       nullInt64ToIntPtr(s.shotsFired),
		ShotsHit:         nullInt64ToIntPtr(s.shotsHit),
		PerfectKills:     nullInt64ToIntPtr(s.perfectKills),
	}
}

// projectEnrichment projette la section Enrichment (PME + skill).
func projectEnrichment(s playerMatchScanResult, skillSnap *canonical.SkillSnapshot) canonical.PlayerMatchEnrichment {
	return canonical.PlayerMatchEnrichment{
		SessionID:           nullStringPtr(s.sessionID),
		SessionLabel:        nullStringPtr(s.sessionLabel),
		PerformanceScore:    nullFloatPtr(s.performanceScore),
		DominanceFlag:       canonical.DominanceFlag(s.dominanceFlag),
		HadBotTeammate:      s.hadBotTeammate,
		IsWithFriends:       s.isWithFriends,
		TeamMMR:             nullFloatPtr(s.teamMMR),
		EnemyMMR:            nullFloatPtr(s.enemyMMR),
		SkillSnapshot:       skillSnap,
		EngagementScoreBrut: nullFloatPtr(s.engagementScoreBrut),
	}
}

// outcomeToInt convertit un canonical.Outcome (string) vers le code int stocke
// en DB (1=tie, 2=win, 3=loss, 4=dnf).
func outcomeToInt(o canonical.Outcome) int {
	switch o {
	case canonical.OutcomeTie:
		return 1
	case canonical.OutcomeWin:
		return 2
	case canonical.OutcomeLoss:
		return 3
	case canonical.OutcomeDNF:
		return 4
	}
	return 0
}

// outcomeFromInt convertit le code int DB vers un canonical.Outcome.
func outcomeFromInt(i int) canonical.Outcome {
	switch i {
	case 1:
		return canonical.OutcomeTie
	case 2:
		return canonical.OutcomeWin
	case 3:
		return canonical.OutcomeLoss
	case 4:
		return canonical.OutcomeDNF
	}
	return canonical.Outcome("")
}

// matchTypeFromFlags choisit un MatchType canonique a partir de is_ranked /
// is_firefight (selection prioritaire : firefight > ranked > social).
func matchTypeFromFlags(isRanked, isFirefight bool) canonical.MatchType {
	if isFirefight {
		return canonical.MatchTypeFirefight
	}
	if isRanked {
		return canonical.MatchTypeRanked
	}
	return canonical.MatchTypeSocial
}

// assetReference compose un canonical.AssetReference depuis les colonnes DB.
// Retourne nil si aucun ID ni label.
func assetReference(kind, id, name, nameFR string) *canonical.AssetReference {
	if id == "" && name == "" && nameFR == "" {
		return nil
	}
	ref := &canonical.AssetReference{
		Kind:         kind,
		ID:           id,
		DefaultLabel: name,
	}
	if nameFR != "" || name != "" {
		ref.Labels = map[string]string{}
		if name != "" {
			ref.Labels["en"] = name
		}
		if nameFR != "" {
			ref.Labels["fr"] = nameFR
		}
	}
	return ref
}

// periodSince extrait le timestamp depuis temporal.Period, ou nil si absente.
func periodSince(p *temporal.Period) *time.Time {
	if p == nil {
		return nil
	}
	return p.Since(time.Now())
}

// playlistKindClause traduit l'alias court en clause SQL safe (pas de regex
// libre interpolee). Whitelist fermee, conforme au design § 5.3.5 du meta-plan.
//
// Erreurs : retourne ErrUnknownPlaylistKind si l'alias n'est pas dans la
// whitelist (input untrusted).
func playlistKindClause(kind string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "":
		return "", nil
	case "ranked":
		return "COALESCE(r.is_ranked, FALSE) = TRUE", nil
	case "firefight":
		return "COALESCE(r.is_firefight, FALSE) = TRUE", nil
	case "social":
		return "COALESCE(r.is_ranked, FALSE) = FALSE AND COALESCE(r.is_firefight, FALSE) = FALSE", nil
	case "btb":
		return "LOWER(COALESCE(r.pair_name, '')) LIKE '%btb%'", nil
	}
	return "", fmt.Errorf("%w: %q", ErrUnknownPlaylistKind, kind)
}

// ErrUnknownPlaylistKind est retournee par buildQuery si PlaylistKind n'est pas
// dans la whitelist des alias supportes.
var ErrUnknownPlaylistKind = errors.New("PlayerMatchesRepo: unknown PlaylistKind")

// ErrUnknownOrderBy est retournee si OrderBy n'est pas dans la whitelist.
// Utilisée par classifyOrderBy (cf. partie split shared/post-merge).
var ErrUnknownOrderBy = errors.New("PlayerMatchesRepo: unknown OrderBy")

// nullFloatPtr convertit sql.NullFloat64 en *float64.
func nullFloatPtr(n sql.NullFloat64) *float64 {
	if !n.Valid {
		return nil
	}
	v := n.Float64
	return &v
}

// nullInt64ToIntPtr convertit sql.NullInt64 en *int.
func nullInt64ToIntPtr(n sql.NullInt64) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int64)
	return &v
}
