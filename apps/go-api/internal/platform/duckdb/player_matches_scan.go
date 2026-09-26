package duckdb

import (
	"database/sql"
	"sort"
	"time"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

func (r *PlayerMatchesRepo) mergePlayerMatchRows(
	shared []playerMatchScanResult,
	enrichments map[string]MatchEnrichment,
	skillRanks map[string]playerMatchSkillRankRow,
	csrMeta map[string]matchCSRMeta,
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
			shared[i].engagementPaceRatio = e.EngagementPaceRatio
		}
		if s, ok := skillRanks[shared[i].matchID]; ok {
			shared[i].skillRatingType = s.ratingType
			shared[i].skillRatingValue = s.ratingValue
			shared[i].skillTier = s.tier
			shared[i].skillTierFR = s.tierFR
			shared[i].skillSubTier = s.subTier
			shared[i].skillDelta = s.delta
			shared[i].skillPlaylistGroup = s.playlistGroup
			shared[i].skillExpectedWinProb = s.expectedWinProb
		}
		// season_id + measurement_matches_remaining vivent dans match_csrs
		// (shared DB), pas dans match_skill_rank — source unique de vérité.
		if m, ok := csrMeta[shared[i].matchID]; ok {
			shared[i].skillSeasonID = m.seasonID
			shared[i].skillMeasurementRemaining = m.measurementRemaining
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

// scanSharedPlayerMatchRow scanne la partie shared (41 cols) en
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
		&s.team0RoundsWon, &s.team1RoundsWon, &s.roundsTotal,
		&s.maxKillingSpree, &s.personalScore, &s.rankInMatch,
		&s.grenadeKills, &s.meleeKills, &s.powerWeaponKills,
		&s.assassinationKills, &s.groundPoundKills, &s.shoulderBashKills,
		&s.shotsFired, &s.shotsHit,
		&s.killsExpected, &s.deathsExpected,
		&s.perfectKills,
		&s.t0Ms,
	); err != nil {
		return playerMatchScanResult{}, err
	}
	return s, nil
}

// playerMatchSkillRankRow porte les colonnes match_skill_rank chargées en
// étape 3. (playerMatchEnrichmentRow + playerMatchesEnrichmentTpl retirés au
// commit 9d.4 — remplacés par MatchEnrichment + LoadPlayerMatchEnrichments).
type playerMatchSkillRankRow struct {
	ratingType      sql.NullString
	ratingValue     sql.NullFloat64
	tier            sql.NullString
	tierFR          sql.NullString
	subTier         sql.NullInt64
	delta           sql.NullFloat64
	playlistGroup   sql.NullString
	expectedWinProb sql.NullFloat64
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
    playlist_group,
    expected_win_prob
FROM match_skill_rank_latest
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
	matchID, mapID, mapName, mapNameFR                      string
	playlistID, playlistName, playlistNameFR                string
	variantID, variantName, xuid, gamertag                  string
	pairID, pairName, pairNameFR                            string
	startTime                                               time.Time
	durationSeconds, teamID                                 int
	outcomeCode                                             sql.NullInt64
	kills, deaths, assists, headshotKills                   int
	timePlayedSeconds, dominanceFlag                        int
	isRanked, isFirefight, hadBotTeammate                   bool
	isWithFriends                                           bool
	engagementScoreBrut                                     sql.NullFloat64
	engagementPaceRatio                                     sql.NullFloat64
	kda, accuracy, teamMMR, enemyMMR                        sql.NullFloat64
	avgLifeSeconds                                          sql.NullFloat64
	damageDealt, damageTaken                                sql.NullFloat64
	performanceScore                                        sql.NullFloat64
	sessionID                                               sql.NullString // VARCHAR en prod (cf. ADR 0016 / commit 9d.4)
	sessionLabel                                            sql.NullString
	team0Score, team1Score                                  int
	team0RoundsWon, team1RoundsWon, roundsTotal             int
	skillRatingType                                         sql.NullString
	skillRatingValue                                        sql.NullFloat64
	skillTier                                               sql.NullString
	skillTierFR                                             sql.NullString
	skillSubTier                                            sql.NullInt64
	skillDelta                                              sql.NullFloat64
	skillPlaylistGroup                                      sql.NullString
	skillExpectedWinProb                                    sql.NullFloat64
	skillSeasonID                                           sql.NullString
	skillMeasurementRemaining                               sql.NullInt64
	maxKillingSpree, personalScore, rankInMatch             sql.NullInt64
	grenadeKills, meleeKills, powerWeaponKills              sql.NullInt64
	assassinationKills, groundPoundKills, shoulderBashKills sql.NullInt64
	shotsFired, shotsHit                                    sql.NullInt64
	killsExpected, deathsExpected                           sql.NullFloat64 // match_participants.kills_expected/deaths_expected (écart FDA attendu)
	perfectKills                                            sql.NullInt64
	t0Ms                                                    sql.NullInt64 // countdown pré-match en ms (Match Timeline T0)
}

// projectPlayerMatchRow construit la row canonique depuis les valeurs scannees.
