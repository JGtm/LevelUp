// Package duckdb â€” squad_repo.go : accÃ¨s DB pour la page Escouade et SynthÃ¨se.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

// SquadRepo implÃ©mente port.SquadRepository.
type SquadRepo struct {
	pdb *PlayerDB
}

// NewSquadRepo crÃ©e un SquadRepo pour un joueur.
func NewSquadRepo(pdb *PlayerDB) *SquadRepo {
	return &SquadRepo{pdb: pdb}
}

// LoadTopTeammates charge les meilleurs coÃ©quipiers du joueur (Q29, top 50).
//
// Sprint B1 commit 9c.1 : split cross-DB en 2 étapes.
//
//	Étape 1 (pdb.Player) : match_ids du joueur avec is_with_friends = TRUE.
//	Étape 2 (SharedReader) : aggregation sur match_participants restreinte
//	  aux match_ids retournés en étape 1, groupée par teammate xuid.
func (r *SquadRepo) LoadTopTeammates(ctx context.Context, xuid string) ([]domain.TopTeammateRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	matchIDs, err := r.loadWithFriendsMatchIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("LoadTopTeammates: %w", err)
	}
	if len(matchIDs) == 0 {
		return nil, nil
	}

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("LoadTopTeammates: shared reader: %w", err)
	}
	defer release()

	query := fmt.Sprintf(Q29TopTeammatesSharedTpl, Placeholders(len(matchIDs)))
	args := make([]any, 0, 2+len(matchIDs))
	args = append(args, xuid)
	args = append(args, ToAnySlice(matchIDs)...)
	args = append(args, xuid)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("LoadTopTeammates: %w", err)
	}
	defer rows.Close()

	var result []domain.TopTeammateRow
	for rows.Next() {
		var row domain.TopTeammateRow
		if err := rows.Scan(
			&row.XUID,
			&row.Gamertag,
			&row.GamesTogether,
			&row.WinsTogether,
			&row.WinRate,
			&row.AvgKills,
			&row.AvgDeaths,
			&row.AvgKDA,
		); err != nil {
			return nil, fmt.Errorf("LoadTopTeammates scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// loadWithFriendsMatchIDs retourne les match_ids où player_match_enrichment.is_with_friends
// est TRUE. Helper pour le split LoadTopTeammates (commit 9c.1).
func (r *SquadRepo) loadWithFriendsMatchIDs(ctx context.Context) ([]string, error) {
	ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := r.pdb.Player.Query(ctx2,
		`SELECT match_id FROM player_match_enrichment WHERE is_with_friends = TRUE`)
	if err != nil {
		return nil, fmt.Errorf("loadWithFriendsMatchIDs: %w", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("loadWithFriendsMatchIDs scan: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// LookupXUIDByGamertag rÃ©sout un gamertag (ILIKE, case-insensitive) vers son
// XUID via shared.xuid_aliases. Sert de fallback pour les coÃ©quipiers sÃ©lectionnÃ©s
// qui sortent du top 50 LoadTopTeammates (saisie libre dans la combobox).
//
// Si plusieurs aliases correspondent au mÃªme gamertag (changement de pseudo
// historique), on retourne le plus rÃ©cent. Si aucun alias, retourne ("", false, nil).
func (r *SquadRepo) LookupXUIDByGamertag(ctx context.Context, gamertag string) (string, bool, error) {
	gamertag = strings.TrimSpace(gamertag)
	if gamertag == "" {
		return "", false, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// xuid_aliases : (xuid PK, gamertag, last_seen, source, updated_at)
	// Source canonique pour ce titre — peuplée par le sync engine. La DB globale
	// est obsolète (migration one-shot souvent vide).
	//
	// Sprint B1 commit 9c.2 : shared-only via SharedReader, naming root-level.
	const q = `
SELECT xuid
FROM xuid_aliases
WHERE gamertag ILIKE ?
ORDER BY last_seen DESC NULLS LAST
LIMIT 1`

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return "", false, fmt.Errorf("LookupXUIDByGamertag(%q): shared reader: %w", gamertag, err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, q, gamertag)
	if err != nil {
		return "", false, fmt.Errorf("LookupXUIDByGamertag(%q): %w", gamertag, err)
	}
	defer rows.Close()
	if !rows.Next() {
		return "", false, rows.Err()
	}
	var xuid string
	if err := rows.Scan(&xuid); err != nil {
		return "", false, fmt.Errorf("LookupXUIDByGamertag scan: %w", err)
	}
	return xuid, xuid != "", nil
}

// LoadSquadMatches charge les matchs communs joueur+coÃ©quipier (Q30).
//
// Sprint B1 commit 9c.2 : split cross-DB en 3 étapes.
//
//	Étape 1 (SharedReader) : Q30SquadMatchesSharedQuery — match_participants
//	  ⨝ v_match_full ⨝ match_participants (teammate filter) + subquery
//	  medals_earned. 25 cols shared.
//	Étape 2 (pdb.Player) : player_match_enrichment WHERE match_id IN (...).
//	Étape 3 (Go) : merge LEFT JOIN — hydrate session_id, session_label,
//	  performance_score, is_with_friends.
func (r *SquadRepo) LoadSquadMatches(ctx context.Context, playerXUID, teammateXUID string) ([]domain.SquadMatchRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Étape 1 : shared rows.
	results, err := r.loadSquadMatchesShared(ctx, playerXUID, teammateXUID)
	if err != nil {
		return nil, fmt.Errorf("LoadSquadMatches: %w", err)
	}
	if len(results) == 0 {
		return nil, nil
	}

	// Étape 2 + 3 : merge enrichments.
	matchIDs := make([]string, 0, len(results))
	for _, m := range results {
		matchIDs = append(matchIDs, m.MatchID)
	}
	enrichments, err := r.loadSquadEnrichments(ctx, matchIDs)
	if err != nil {
		return nil, fmt.Errorf("LoadSquadMatches: %w", err)
	}
	for i := range results {
		if e, ok := enrichments[results[i].MatchID]; ok {
			if e.sessionID.Valid {
				v := int(e.sessionID.Int64)
				results[i].SessionID = &v
			}
			if e.sessionLabel.Valid {
				v := e.sessionLabel.String
				results[i].SessionLabel = &v
			}
			if e.performanceScore.Valid {
				v := e.performanceScore.Float64
				results[i].PerformanceScore = &v
			}
			results[i].IsWithFriends = e.isWithFriends
		}
	}
	return results, nil
}

// loadSquadMatchesShared exécute l'étape 1 du split LoadSquadMatches.
// Retourne les SquadMatchRow sans les cols PME (zero values).
func (r *SquadRepo) loadSquadMatchesShared(ctx context.Context, playerXUID, teammateXUID string) ([]domain.SquadMatchRow, error) {
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("shared reader: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, Q30SquadMatchesSharedQuery, teammateXUID, playerXUID)
	if err != nil {
		return nil, fmt.Errorf("shared query: %w", err)
	}
	defer rows.Close()

	var result []domain.SquadMatchRow
	for rows.Next() {
		var row domain.SquadMatchRow
		if err := rows.Scan(
			&row.MatchID,
			&row.StartTime,
			&row.MapName,
			&row.MapUI,
			&row.PairName,
			&row.PlaylistName,
			&row.IsFirefight,
			&row.IsRanked,
			&row.Outcome,
			&row.Kills,
			&row.Deaths,
			&row.Assists,
			&row.KDA,
			&row.Accuracy,
			&row.TimePlayedSecs,
			&row.DurationSeconds,
			&row.TeamMMR,
			&row.HeadshotKills,
			&row.PerfectKills,
			&row.EnemyMMR,
			&row.MyTeamScore,
			&row.EnemyTeamScore,
			&row.MapID,
			&row.PlaylistID,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// squadEnrichment porte les colonnes player_match_enrichment hydratées en
// étape 2 du split LoadSquadMatches.
type squadEnrichment struct {
	sessionID        sql.NullInt64
	sessionLabel     sql.NullString
	performanceScore sql.NullFloat64
	isWithFriends    bool
}

// loadSquadEnrichments exécute l'étape 2 du split LoadSquadMatches.
func (r *SquadRepo) loadSquadEnrichments(ctx context.Context, matchIDs []string) (map[string]squadEnrichment, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}
	query := fmt.Sprintf(`
		SELECT match_id, session_id, session_label, performance_score,
		       COALESCE(is_with_friends, FALSE)
		FROM player_match_enrichment
		WHERE match_id IN (%s)`, Placeholders(len(matchIDs)))
	rows, err := r.pdb.Player.Query(ctx, query, ToAnySlice(matchIDs)...)
	if err != nil {
		return nil, fmt.Errorf("enrichment query: %w", err)
	}
	defer rows.Close()

	out := make(map[string]squadEnrichment, len(matchIDs))
	for rows.Next() {
		var mid string
		var e squadEnrichment
		if err := rows.Scan(&mid, &e.sessionID, &e.sessionLabel, &e.performanceScore, &e.isWithFriends); err != nil {
			return nil, fmt.Errorf("enrichment scan: %w", err)
		}
		out[mid] = e
	}
	return out, rows.Err()
}

// LoadTeammateMatches charge les stats du coÃ©quipier sur les matchs communs (Q31).
//
// Sprint B1 commit 9c.1 : query shared-only (match_participants x2 + v_match_full)
// migrée vers SharedReader.Get.
func (r *SquadRepo) LoadTeammateMatches(ctx context.Context, playerXUID, teammateXUID string) ([]domain.TeammateMatchRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("LoadTeammateMatches: shared reader: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, Q31TeammateMatches, playerXUID, teammateXUID)
	if err != nil {
		return nil, fmt.Errorf("LoadTeammateMatches: %w", err)
	}
	defer rows.Close()

	var result []domain.TeammateMatchRow
	for rows.Next() {
		var row domain.TeammateMatchRow
		if err := rows.Scan(
			&row.MatchID,
			&row.StartTime,
			&row.MapUI,
			&row.PairName,
			&row.Outcome,
			&row.Kills,
			&row.Deaths,
			&row.Assists,
			&row.Ratio,
			&row.TimePlayedSecs,
			&row.TeamMMR,
			&row.Accuracy,
			&row.MyTeamScore,
			&row.EnemyTeamScore,
		); err != nil {
			return nil, fmt.Errorf("LoadTeammateMatches scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadImpactEvents charge les Ã©vÃ©nements highlight pour une liste de match_ids (Q32 dynamique).
// matchIDs est la liste des identifiants â€” si vide, retourne nil directement.
func (r *SquadRepo) LoadImpactEvents(ctx context.Context, matchIDs []string) ([]domain.ImpactEventRow, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Construire la clause IN dynamiquement.
	placeholders := make([]string, len(matchIDs))
	args := make([]interface{}, len(matchIDs))
	for i, id := range matchIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	query := fmt.Sprintf(Q32SquadImpactEventsTemplate, strings.Join(placeholders, ","))

	// Sprint B1 commit 8k.13 : shared-only via SharedReader.
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("LoadImpactEvents: shared reader: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("LoadImpactEvents: %w", err)
	}
	defer rows.Close()

	var result []domain.ImpactEventRow
	for rows.Next() {
		var row domain.ImpactEventRow
		if err := rows.Scan(
			&row.MatchID,
			&row.XUID,
			&row.Gamertag,
			&row.EventType,
			&row.TimeMS,
		); err != nil {
			return nil, fmt.Errorf("LoadImpactEvents scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadMainTeamParticipants charge tous les participants de l'équipe alliée
// du joueur principal pour une liste de matchs (Q32b). Pour chaque match,
// retourne les rows match_participants où team_id = team_id du mainXUID dans
// ce match (le main inclus).
func (r *SquadRepo) LoadMainTeamParticipants(ctx context.Context, mainXUID string, matchIDs []string) ([]domain.AllyParticipant, error) {
	if len(matchIDs) == 0 || mainXUID == "" {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	placeholders := make([]string, len(matchIDs))
	args := make([]interface{}, 0, 1+len(matchIDs))
	args = append(args, mainXUID)
	for i, id := range matchIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	query := fmt.Sprintf(Q32bMainTeamParticipantsTemplate, strings.Join(placeholders, ","))

	// Sprint B1 commit 8k.13 : shared-only via SharedReader.
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("LoadMainTeamParticipants: shared reader: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("LoadMainTeamParticipants: %w", err)
	}
	defer rows.Close()

	var result []domain.AllyParticipant
	for rows.Next() {
		var row domain.AllyParticipant
		if err := rows.Scan(
			&row.MatchID,
			&row.XUID,
			&row.Gamertag,
			&row.Kills,
			&row.Deaths,
			&row.Assists,
			&row.Outcome,
		); err != nil {
			return nil, fmt.Errorf("LoadMainTeamParticipants scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadSynthesisHeatmap charge les donnÃ©es heatmap mapÃ—mode (Q33).
func (r *SquadRepo) LoadSynthesisHeatmap(ctx context.Context, xuid string) ([]domain.SynthesisHeatmapRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Sprint B1 commit 8k.13 : shared-only via SharedReader.
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("LoadSynthesisHeatmap: shared reader: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, Q33SynthesisHeatmap, xuid)
	if err != nil {
		return nil, fmt.Errorf("LoadSynthesisHeatmap: %w", err)
	}
	defer rows.Close()

	var result []domain.SynthesisHeatmapRow
	for rows.Next() {
		var row domain.SynthesisHeatmapRow
		if err := rows.Scan(
			&row.MapName,
			&row.ModeName,
			&row.MatchCount,
			&row.Wins,
		); err != nil {
			return nil, fmt.Errorf("LoadSynthesisHeatmap scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadSynthesisMatches charge les matchs du joueur pour le calcul top_weeks (Q33b).
//
// Sprint B1 commit 9c.3 : split cross-DB en 3 étapes.
//
//	Étape 1 (SharedReader) : Q33bSynthesisSharedQuery — match_participants ⨝
//	  match_registry. 11 cols shared.
//	Étape 2 (pdb.Player) : player_match_enrichment WHERE match_id IN (...).
//	Étape 3 (Go) : merge LEFT JOIN — hydrate is_with_friends, performance_score,
//	  session_label.
func (r *SquadRepo) LoadSynthesisMatches(ctx context.Context, xuid string) ([]legacymatch.SynthesisMatchRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Étape 1 : shared.
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("LoadSynthesisMatches: shared reader: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, Q33bSynthesisSharedQuery, xuid)
	if err != nil {
		return nil, fmt.Errorf("LoadSynthesisMatches: %w", err)
	}
	defer rows.Close()

	var result []legacymatch.SynthesisMatchRow
	for rows.Next() {
		var row legacymatch.SynthesisMatchRow
		if err := rows.Scan(
			&row.MatchID,
			&row.StartTime,
			&row.Outcome,
			&row.Kills,
			&row.Deaths,
			&row.KDA,
			&row.Accuracy,
			&row.TimePlayedSecs,
			&row.IsRanked,
			&row.IsFirefight,
			&row.PlaylistName,
		); err != nil {
			return nil, fmt.Errorf("LoadSynthesisMatches scan: %w", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return result, nil
	}

	// Étape 2 + 3 : enrichment merge.
	matchIDs := make([]string, 0, len(result))
	for _, m := range result {
		matchIDs = append(matchIDs, m.MatchID)
	}
	if err := r.mergeSynthesisEnrichments(ctx, result, matchIDs); err != nil {
		return nil, fmt.Errorf("LoadSynthesisMatches: %w", err)
	}
	return result, nil
}

// mergeSynthesisEnrichments hydrate is_with_friends + performance_score +
// session_label depuis player_match_enrichment (étape 2/3 du split).
func (r *SquadRepo) mergeSynthesisEnrichments(ctx context.Context, rows []legacymatch.SynthesisMatchRow, matchIDs []string) error {
	query := fmt.Sprintf(`
		SELECT match_id, COALESCE(is_with_friends, FALSE),
		       performance_score, session_label
		FROM player_match_enrichment
		WHERE match_id IN (%s)`, Placeholders(len(matchIDs)))
	ctx2, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	dbRows, err := r.pdb.Player.Query(ctx2, query, ToAnySlice(matchIDs)...)
	if err != nil {
		return fmt.Errorf("synthesis enrichment query: %w", err)
	}
	defer dbRows.Close()

	type synthEnrich struct {
		isWithFriends bool
		performance   sql.NullFloat64
		sessionLabel  sql.NullString
	}
	enrichments := make(map[string]synthEnrich, len(matchIDs))
	for dbRows.Next() {
		var mid string
		var e synthEnrich
		if err := dbRows.Scan(&mid, &e.isWithFriends, &e.performance, &e.sessionLabel); err != nil {
			return fmt.Errorf("synthesis enrichment scan: %w", err)
		}
		enrichments[mid] = e
	}
	if err := dbRows.Err(); err != nil {
		return err
	}
	for i := range rows {
		e, ok := enrichments[rows[i].MatchID]
		if !ok {
			continue
		}
		rows[i].IsWithFriends = e.isWithFriends
		if e.performance.Valid {
			v := e.performance.Float64
			rows[i].PerformanceScore = &v
		}
		if e.sessionLabel.Valid {
			v := e.sessionLabel.String
			rows[i].SessionLabel = &v
		}
	}
	return nil
}

// LoadMapStatsForSquad calcule par carte (map_id) le winrate et la performance
// moyenne du joueur principal sur les matchs où TOUS les xuids du squad sont
// participants. Aucun filtre temporel — c'est l'historique complet "avec cette
// escouade exacte".
//
// Sprint B1 commit 9c.4 : split cross-DB en 3 étapes.
//
//	Étape 1 (SharedReader) : Q42MapStatsForSquadSharedTpl — retourne per-match
//	  rows (match_id, map_id, outcome) avec le CTE squad_matches (filtre
//	  cardinality du squad).
//	Étape 2 (pdb.Player) : SELECT match_id, performance_score FROM
//	  player_match_enrichment WHERE match_id IN (...).
//	Étape 3 (Go) : aggregation par map_id — total, wins, perf_avg.
//
// Comportement :
//   - squadXUIDs vide : retourne nil, nil (pas de squad sélectionné).
//   - squadXUIDs ne contenant que mainXUID : tombe sur les stats solo du main
//     (cas dégénéré utile pour le mode solo).
//   - mainXUID inclus dans squadXUIDs : pas de doublonnage côté SQL grâce au
//     COUNT(DISTINCT xuid) dans squad_matches.
//
// Retour : map keyée sur map_id (jamais vide ; clé absente = aucun match avec
// ce squad sur cette carte).
func (r *SquadRepo) LoadMapStatsForSquad(ctx context.Context, mainXUID string, squadXUIDs []string) (map[string]domain.MapSquadStats, error) {
	if mainXUID == "" || len(squadXUIDs) == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Étape 1 : shared per-match rows.
	matchRows, err := r.loadMapStatsSquadShared(ctx, mainXUID, squadXUIDs)
	if err != nil {
		return nil, fmt.Errorf("LoadMapStatsForSquad: %w", err)
	}
	if len(matchRows) == 0 {
		return map[string]domain.MapSquadStats{}, nil
	}

	// Étape 2 : performance_score depuis player_match_enrichment.
	matchIDs := make([]string, 0, len(matchRows))
	for _, m := range matchRows {
		matchIDs = append(matchIDs, m.matchID)
	}
	perfMap, err := r.loadMatchPerformanceScores(ctx, matchIDs)
	if err != nil {
		return nil, fmt.Errorf("LoadMapStatsForSquad: %w", err)
	}

	// Étape 3 : aggregation par map_id en Go.
	type mapAgg struct {
		total, wins int
		perfSum     float64
		perfCount   int
	}
	aggs := make(map[string]*mapAgg)
	for _, m := range matchRows {
		if m.mapID == "" {
			continue
		}
		a, ok := aggs[m.mapID]
		if !ok {
			a = &mapAgg{}
			aggs[m.mapID] = a
		}
		a.total++
		if m.outcome == 2 {
			a.wins++
		}
		if perf, ok := perfMap[m.matchID]; ok {
			a.perfSum += perf
			a.perfCount++
		}
	}

	result := make(map[string]domain.MapSquadStats, len(aggs))
	for mapID, a := range aggs {
		s := domain.MapSquadStats{Wins: a.wins, Total: a.total}
		if a.perfCount > 0 {
			v := a.perfSum / float64(a.perfCount)
			s.PerfAvg = &v
		}
		result[mapID] = s
	}
	return result, nil
}

// mapStatsSquadMatchRow porte une row per-match retournée par l'étape 1 du
// split LoadMapStatsForSquad.
type mapStatsSquadMatchRow struct {
	matchID string
	mapID   string
	outcome int
}

// loadMapStatsSquadShared exécute l'étape 1 du split LoadMapStatsForSquad.
func (r *SquadRepo) loadMapStatsSquadShared(ctx context.Context, mainXUID string, squadXUIDs []string) ([]mapStatsSquadMatchRow, error) {
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("shared reader: %w", err)
	}
	defer release()

	q := fmt.Sprintf(Q42MapStatsForSquadSharedTpl, Placeholders(len(squadXUIDs)), len(uniqueXUIDs(squadXUIDs)))
	args := make([]any, 0, len(squadXUIDs)+1)
	for _, x := range squadXUIDs {
		args = append(args, x)
	}
	args = append(args, mainXUID)

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("shared query: %w", err)
	}
	defer rows.Close()

	var out []mapStatsSquadMatchRow
	for rows.Next() {
		var m mapStatsSquadMatchRow
		var outcome sql.NullInt64
		if err := rows.Scan(&m.matchID, &m.mapID, &outcome); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		if outcome.Valid {
			m.outcome = int(outcome.Int64)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// loadMatchPerformanceScores retourne le performance_score par match_id depuis
// player_match_enrichment (étape 2 du split LoadMapStatsForSquad).
func (r *SquadRepo) loadMatchPerformanceScores(ctx context.Context, matchIDs []string) (map[string]float64, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}
	query := fmt.Sprintf(`
		SELECT match_id, performance_score
		FROM player_match_enrichment
		WHERE match_id IN (%s) AND performance_score IS NOT NULL`,
		Placeholders(len(matchIDs)))
	rows, err := r.pdb.Player.Query(ctx, query, ToAnySlice(matchIDs)...)
	if err != nil {
		return nil, fmt.Errorf("performance scores query: %w", err)
	}
	defer rows.Close()
	out := make(map[string]float64, len(matchIDs))
	for rows.Next() {
		var mid string
		var perf float64
		if err := rows.Scan(&mid, &perf); err != nil {
			return nil, fmt.Errorf("performance scores scan: %w", err)
		}
		out[mid] = perf
	}
	return out, rows.Err()
}

// uniqueXUIDs déduplique sans modifier l'ordre — utile uniquement pour le
// HAVING COUNT(DISTINCT) du SQL squad_matches (la cardinalité ne doit pas
// compter les doublons applicatifs).
func uniqueXUIDs(xs []string) []string {
	seen := make(map[string]struct{}, len(xs))
	out := xs[:0:0]
	for _, x := range xs {
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	return out
}

// LoadAssetTranslationsFR retourne les traductions FR depuis metadata.asset_translations.
// Wrapper mince autour du résolveur unifié `MetadataRepo.ResolveAssetNamesBulk`
// (cf. metadata_repo_assets.go). Cohérent avec home_repo.resolveAssetNames.
// assetType : "map" | "playlist" | "game_variant" | "pair".
func (r *SquadRepo) LoadAssetTranslationsFR(ctx context.Context, assetType string, assetIDs []string) (map[string]string, error) {
	if len(assetIDs) == 0 || r.pdb == nil || r.pdb.Metadata == nil {
		return nil, nil
	}
	out, err := NewMetadataRepoFromDB(r.pdb.Metadata).ResolveAssetNamesBulk(
		ctx, assetType, assetIDs, PreferredLangsForLocale("fr"),
	)
	if err != nil && isTableNotFoundErr(err) {
		return nil, nil
	}
	return out, err
}

// LoadModeTranslationsFR retourne les traductions FR depuis metadata.mode_name_tr.
// Calqué sur homeRepo.loadHomeModeNameTranslations — même table, même logique.
func (r *SquadRepo) LoadModeTranslationsFR(ctx context.Context, modeENs []string) (map[string]string, error) {
	if len(modeENs) == 0 || r.pdb == nil || r.pdb.Metadata == nil {
		return nil, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(modeENs)), ",")
	q := fmt.Sprintf(`SELECT mode_en, name FROM mode_name_tr WHERE lang = 'fr' AND mode_en IN (%s)`, placeholders)
	args := make([]any, len(modeENs))
	for i, n := range modeENs {
		args[i] = n
	}
	rows, err := r.pdb.Metadata.Query(ctx, q, args...)
	if err != nil {
		if isTableNotFoundErr(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("LoadModeTranslationsFR: %w", err)
	}
	defer rows.Close()
	result := make(map[string]string, len(modeENs))
	for rows.Next() {
		var en, fr string
		if err := rows.Scan(&en, &fr); err != nil {
			continue
		}
		if strings.TrimSpace(fr) != "" {
			result[en] = fr
		}
	}
	return result, rows.Err()
}

// Ensure SquadRepo implements port.SquadRepository at compile time.
// (Vérification implicite via injection dans le service.)
