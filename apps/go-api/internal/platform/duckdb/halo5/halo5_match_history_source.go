// Package duckdb — halo5_match_history_source.go : lecture de l'historique de
// matchs Halo 5 depuis le substrat LOCAL synchronisé (shared_matches_v2.duckdb du
// titre h5) projeté en canonical.MatchSummary.
//
// C'est le pendant h5 du MatchHistoryRepo d'Infinite, mais ciblé canonical.MatchSummary
// (la voie LoadMatchSummaries de l'adapter) plutôt que domain.MatchHistoryRawRow.
// AUCUN fetch live : la donnée (match_registry + match_participants) est déjà écrite
// par le livesync h5. Cf. AXE A de .ai/PLAN_H5_PROD_GATE.md.
//
// Identité joueur : Halo 5 résout le gamertag en xuid Xbox réel au sync, mais
// l'adapter h5 reçoit le GAMERTAG comme clé (cf. internal/games/halo_5/adapter_data.go).
// Le filtre se fait donc sur match_participants.gamertag (colonne peuplée pour h5),
// pas sur xuid — l'adapter n'a pas à résoudre l'xuid en lecture.
//
// Cette implémentation satisfait STRUCTURELLEMENT l'interface halo_5.MatchHistorySource
// (définie côté package consommateur), sans cycle d'import.
package halo5

import (
	"context"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/platform/duckdb"
)

// Halo5MatchHistorySource lit l'historique h5 d'un joueur (fixé à la construction)
// depuis le shared via un SharedReader title-aware.
type Halo5MatchHistorySource struct {
	shared   duckdb.SharedReader
	gamertag string
}

// NewHalo5MatchHistorySource construit la source liée à un joueur (gamertag) et au
// SharedReader du titre h5. Le gamertag est normalisé (trim) ; la comparaison SQL
// est insensible à la casse (LOWER), h5 étant gamertag-keyé.
func NewHalo5MatchHistorySource(shared duckdb.SharedReader, gamertag string) *Halo5MatchHistorySource {
	return &Halo5MatchHistorySource{shared: shared, gamertag: strings.TrimSpace(gamertag)}
}

// h5DefaultHistoryLimit borne le nombre de matchs renvoyés quand matchIDs est vide
// (les N derniers). Aligné sur une page d'historique large ; le service paginera.
const h5DefaultHistoryLimit = 250

// h5MatchSummarySelect projette match_registry ⨝ match_participants vers les colonnes
// nécessaires à un canonical.MatchSummary. Pattern TZ canonique projet (préférer
// start_time_utc, sinon interpréter start_time naïf comme UTC). team_0/1_score à -1
// quand NULL (= absent, exclu de Teams, cf. projectTeamScores).
var h5MatchSummarySelect = `
SELECT
    p.match_id,
    ` + duckdb.StartTimeCanonicalSQL("r") + ` AS start_time,
    r.duration_seconds,
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
    COALESCE(r.is_ranked, FALSE)                      AS is_ranked,
    COALESCE(r.is_firefight, FALSE)                   AS is_firefight,
    COALESCE(p.outcome, 0)                            AS outcome_code,
    COALESCE(r.team_0_score, -1)                      AS team_0_score,
    COALESCE(r.team_1_score, -1)                      AS team_1_score
FROM match_participants p
JOIN match_registry r ON r.match_id = p.match_id
WHERE LOWER(COALESCE(p.gamertag, '')) = LOWER(?)`

// h5MatchSummaryRow porte les colonnes scannées d'une ligne d'historique h5.
type h5MatchSummaryRow struct {
	matchID        string
	startTime      time.Time
	durationSec    *int
	mapID          string
	mapName        string
	mapNameFR      string
	playlistID     string
	playlistName   string
	playlistNameFR string
	variantID      string
	variantName    string
	pairID         string
	pairName       string
	pairNameFR     string
	isRanked       bool
	isFirefight    bool
	outcomeCode    int
	team0Score     int
	team1Score     int
}

// GetMatchSummaries implémente halo_5.MatchHistorySource. matchIDs nil/vide → les N
// derniers matchs du joueur (start_time DESC) ; sinon filtre sur ces IDs et préserve
// l'ordre d'entrée (un ID absent du shared est silencieusement omis).
func (s *Halo5MatchHistorySource) GetMatchSummaries(ctx context.Context, matchIDs []string) ([]canonical.MatchSummary, error) {
	if strings.TrimSpace(s.gamertag) == "" {
		return nil, nil // pas d'identité → pas d'historique (dégradation neutre)
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	rows, err := s.loadRows(ctx, matchIDs)
	if err != nil {
		return nil, err
	}
	if len(matchIDs) == 0 {
		return projectH5Summaries(rows), nil
	}
	return orderH5SummariesByIDs(rows, matchIDs), nil
}

// loadRows exécute la query shared (filtre IDs optionnel) et retourne les lignes brutes.
func (s *Halo5MatchHistorySource) loadRows(ctx context.Context, matchIDs []string) ([]h5MatchSummaryRow, error) {
	db, release, err := s.shared.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("h5 match history shared reader: %w", err)
	}
	defer release()

	query := h5MatchSummarySelect
	args := []any{s.gamertag}
	// Masquage read-side des matchs Campagne (source unique
	// analysis.SQLExcludeCampaignVariants). Source h5-only → titre fixe "halo_5".
	// Alias "r" = match_registry dans le SELECT. Clause littérale (aucun arg).
	query += analysis.SQLExcludeCampaignVariants("halo_5", "r")
	if len(matchIDs) > 0 {
		query += fmt.Sprintf(" AND p.match_id IN (%s)", duckdb.Placeholders(len(matchIDs)))
		args = append(args, duckdb.ToAnySlice(matchIDs)...)
	} else {
		query += fmt.Sprintf(" ORDER BY start_time DESC LIMIT %d", h5DefaultHistoryLimit)
	}

	dbRows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("h5 match history query: %w", err)
	}
	defer dbRows.Close()

	var out []h5MatchSummaryRow
	for dbRows.Next() {
		var m h5MatchSummaryRow
		if err := dbRows.Scan(
			&m.matchID, &m.startTime, &m.durationSec,
			&m.mapID, &m.mapName, &m.mapNameFR,
			&m.playlistID, &m.playlistName, &m.playlistNameFR,
			&m.variantID, &m.variantName,
			&m.pairID, &m.pairName, &m.pairNameFR,
			&m.isRanked, &m.isFirefight, &m.outcomeCode,
			&m.team0Score, &m.team1Score,
		); err != nil {
			return nil, fmt.Errorf("h5 match history scan: %w", err)
		}
		out = append(out, m)
	}
	return out, dbRows.Err()
}

// projectH5Summaries mappe les lignes (déjà ordonnées par la query) vers MatchSummary.
func projectH5Summaries(rows []h5MatchSummaryRow) []canonical.MatchSummary {
	out := make([]canonical.MatchSummary, 0, len(rows))
	for i := range rows {
		out = append(out, projectH5MatchSummary(rows[i]))
	}
	return out
}

// orderH5SummariesByIDs réordonne les lignes selon l'ordre d'entrée de matchIDs.
// Un ID demandé absent du shared est omis (pas de trou nil dans la liste).
func orderH5SummariesByIDs(rows []h5MatchSummaryRow, matchIDs []string) []canonical.MatchSummary {
	byID := make(map[string]h5MatchSummaryRow, len(rows))
	for i := range rows {
		byID[rows[i].matchID] = rows[i]
	}
	out := make([]canonical.MatchSummary, 0, len(matchIDs))
	for _, id := range matchIDs {
		if r, ok := byID[id]; ok {
			out = append(out, projectH5MatchSummary(r))
		}
	}
	return out
}

// projectH5MatchSummary projette une ligne shared h5 vers canonical.MatchSummary.
// Réutilise les helpers partagés du package (assetReference, outcomeFromInt,
// matchTypeFromFlags, projectTeamScores via team scores -1=absent).
//
// Champs canonical non couverts par le substrat h5 :
//   - PairMode : pair_name h5 souvent absent → nil quand vide (assetReference le gère) ;
//   - T0Ms : pas de countdown/real_start_time exploité côté summary h5 → nil ;
//   - Teams.MMR / ParticipantsXUIDs : indisponibles en h5 (cf. mapping live).
func projectH5MatchSummary(r h5MatchSummaryRow) canonical.MatchSummary {
	isRanked := r.isRanked
	isFirefight := r.isFirefight
	return canonical.MatchSummary{
		MatchID:         r.matchID,
		StartedAtUTC:    r.startTime,
		DurationSeconds: r.durationSec,
		MatchType:       duckdb.MatchTypeFromFlags(r.isRanked, r.isFirefight),
		Playlist:        duckdb.AssetReference("playlist", r.playlistID, r.playlistName, r.playlistNameFR),
		Map:             duckdb.AssetReference("map", r.mapID, r.mapName, r.mapNameFR),
		GameVariant:     duckdb.AssetReference("game_variant", r.variantID, r.variantName, ""),
		PairMode:        duckdb.AssetReference("pair_mode", r.pairID, r.pairName, r.pairNameFR),
		IsRanked:        &isRanked,
		IsPvE:           &isFirefight,
		Outcome:         duckdb.OutcomeFromInt(r.outcomeCode),
		Teams:           h5TeamScores(r.team0Score, r.team1Score),
		T0Ms:            nil,
	}
}

// h5TeamScores reproduit projectTeamScores : -1 (COALESCE) = absent, exclu.
func h5TeamScores(team0, team1 int) []canonical.TeamSnapshot {
	var teams []canonical.TeamSnapshot
	if team0 >= 0 {
		score := team0
		teams = append(teams, canonical.TeamSnapshot{TeamID: 0, Score: &score})
	}
	if team1 >= 0 {
		score := team1
		teams = append(teams, canonical.TeamSnapshot{TeamID: 1, Score: &score})
	}
	return teams
}
