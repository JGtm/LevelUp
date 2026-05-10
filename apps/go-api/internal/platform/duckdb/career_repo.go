// Package duckdb — CareerRepo : données de progression de carrière.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
)

// careerEncountersTimeout : limite hard pour Q26 (encounters scope global,
// agrège tous les matchs du joueur + JOIN killer_victim_pairs).
const careerEncountersTimeout = 30 * time.Second

// careerRivalsTimeout : Q27 (agrégat global killer_victim_pairs sur le joueur).
const careerRivalsTimeout = 20 * time.Second

// CareerRepo implémente port.CareerRepository.
type CareerRepo struct {
	pdb *PlayerDB
}

// NewCareerRepo crée un CareerRepo depuis un PlayerDB.
func NewCareerRepo(pdb *PlayerDB) *CareerRepo {
	return &CareerRepo{pdb: pdb}
}

// GetLatestRank retourne la dernière entrée de progression de rang.
func (r *CareerRepo) GetLatestRank(ctx context.Context) (*domain.CareerRankData, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var row domain.CareerRankData
	err := r.pdb.ReadDB().QueryRow(ctx, Q6CareerLatestRank).Scan(
		&row.RankNumber,
		&row.CurrentXP,
		&row.RecordedAt,
		&row.RankLabel,
		&row.RankName,
		&row.RankTier,
		&row.XPForNextRank,
		&row.XPTotal,
		&row.IsMaxRank,
	)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetLatestRank: %w", err)
	}
	return &row, nil
}

// GetXPHistory retourne l'historique XP complet.
func (r *CareerRepo) GetXPHistory(ctx context.Context) ([]domain.XPHistoryPoint, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q7CareerXPHistory)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetXPHistory: %w", err)
	}
	defer rows.Close()

	var results []domain.XPHistoryPoint
	for rows.Next() {
		var p domain.XPHistoryPoint
		if err := rows.Scan(&p.RecordedAt, &p.Rank, &p.CurrentXP, &p.XPTotal); err != nil {
			return nil, fmt.Errorf("CareerRepo.GetXPHistory scan: %w", err)
		}
		results = append(results, p)
	}
	return results, rows.Err()
}

// GetLUSRHistory retourne les checkpoints LUSR.
func (r *CareerRepo) GetLUSRHistory(ctx context.Context) ([]domain.LUSRCheckpointDTO, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q8LUSRHistory)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetLUSRHistory: %w", err)
	}
	defer rows.Close()

	var results []domain.LUSRCheckpointDTO
	for rows.Next() {
		var cp domain.LUSRCheckpointDTO
		var tier sql.NullString
		var subTier sql.NullInt16
		if err := rows.Scan(
			&cp.MatchID, &cp.RatingType, &cp.RatingValue, &cp.TierLabel, &cp.PlaylistGroup, &cp.RecordedAt,
			&cp.RatingDelta, &cp.PlaylistName, &cp.PlaylistID, &tier, &subTier,
		); err != nil {
			return nil, fmt.Errorf("CareerRepo.GetLUSRHistory scan: %w", err)
		}
		tierLabel := ""
		if cp.TierLabel != nil {
			tierLabel = *cp.TierLabel
		}
		cp.BadgeImageURL = buildHomeSkillPeakBadgeURL(
			optionalNullStringValue(tier),
			tierLabel,
			optionalNullInt16Value(subTier),
			homeStaticTitleSlug,
		)
		results = append(results, cp)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	r.enrichLUSRPlaylistNames(ctx, results)
	return results, nil
}

// enrichLUSRPlaylistNames résout les noms de playlists FR via asset_translations.
// Même pattern que applyMatchHistoryFRTranslations : lookup par playlist_id (UUID).
// Best-effort : silencieux si Metadata absent ou résolution échoue.
func (r *CareerRepo) enrichLUSRPlaylistNames(ctx context.Context, cps []domain.LUSRCheckpointDTO) {
	if r.pdb.Metadata == nil || len(cps) == 0 {
		return
	}
	seen := make(map[string]struct{}, len(cps))
	var ids []string
	for _, cp := range cps {
		id := strings.TrimSpace(cp.PlaylistID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return
	}
	metaRepo := NewMetadataRepoFromDB(r.pdb.Metadata)
	names, err := metaRepo.ResolveAssetNamesBulk(ctx, "playlist", ids, PreferredLangsForLocale("fr"))
	if err != nil || len(names) == 0 {
		return
	}
	for i := range cps {
		id := strings.TrimSpace(cps[i].PlaylistID)
		if id == "" {
			continue
		}
		if name := strings.TrimSpace(names[id]); name != "" {
			cps[i].PlaylistName = name
		}
	}
}

// GetTopMatches retourne les N meilleurs matchs par performance_score.
func (r *CareerRepo) GetTopMatches(ctx context.Context) ([]domain.TopMatchRawRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q9TopMatches, r.pdb.XUID, r.pdb.XUID)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetTopMatches: %w", err)
	}
	defer rows.Close()

	var results []domain.TopMatchRawRow
	for rows.Next() {
		var m domain.TopMatchRawRow
		var _section int
		if err := rows.Scan(
			&m.MatchID, &m.PerformanceScore, &m.StartTime,
			&m.MapName, &m.PairName, &m.PlaylistName,
			&m.Outcome, &m.Kills, &m.Deaths, &m.KDA,
			&m.TeamMMR, &m.EnemyMMR, &m.DominanceFlag, &_section,
		); err != nil {
			return nil, fmt.Errorf("CareerRepo.GetTopMatches scan: %w", err)
		}
		results = append(results, m)
	}
	return results, rows.Err()
}

// buildHighlightFilterClause traduit `domain.CareerHighlightFilters` en
// clause SQL additionnelle (à injecter via Sprintf dans Q9bHighlightMatchIDsTpl)
// + les valeurs liées correspondantes. Retourne ("", nil) si aucun filtre actif.
//
// Forme produite, ex. avec ranked + 2 saisons :
//
//	" AND r.is_ranked = TRUE AND ((r.start_time_utc >= ? AND r.start_time_utc < ?) OR (r.start_time_utc >= ?))"
func buildHighlightFilterClause(filters domain.CareerHighlightFilters) (string, []any) {
	parts := make([]string, 0, 2)
	args := make([]any, 0, 4)

	switch strings.ToLower(strings.TrimSpace(filters.Experience)) {
	case "ranked":
		parts = append(parts, "COALESCE(r.is_ranked, FALSE) = TRUE")
	case "unranked":
		parts = append(parts, "COALESCE(r.is_ranked, FALSE) = FALSE")
	}

	if len(filters.SeasonRanges) > 0 {
		seasonExprs := make([]string, 0, len(filters.SeasonRanges))
		for _, win := range filters.SeasonRanges {
			ts := "COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC')"
			if win.End != nil {
				seasonExprs = append(seasonExprs, "("+ts+" >= ? AND "+ts+" < ?)")
				args = append(args, win.Start, *win.End)
			} else {
				seasonExprs = append(seasonExprs, "("+ts+" >= ?)")
				args = append(args, win.Start)
			}
		}
		parts = append(parts, "("+strings.Join(seasonExprs, " OR ")+")")
	}

	if len(parts) == 0 {
		return "", nil
	}
	return " AND " + strings.Join(parts, " AND "), args
}

// GetHighlightMatchIDs retourne 15 best + 15 worst match_ids triés par
// performance + dominance prio. Les rows sont retournées dans l'ordre
// _s ASC : best d'abord, worst ensuite.
//
// `filters` applique optionnellement Experience (is_ranked) et SeasonRanges
// (date windows) via une clause SQL additionnelle dérivée en interne.
func (r *CareerRepo) GetHighlightMatchIDs(ctx context.Context, filters domain.CareerHighlightFilters) ([]domain.HighlightMatchIDRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	clause, extraArgs := buildHighlightFilterClause(filters)
	sqlText := fmt.Sprintf(Q9bHighlightMatchIDsTpl, clause, clause)
	args := make([]any, 0, 2+2*len(extraArgs))
	args = append(args, r.pdb.XUID)
	args = append(args, extraArgs...)
	args = append(args, r.pdb.XUID)
	args = append(args, extraArgs...)

	rows, err := r.pdb.ReadDB().Query(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetHighlightMatchIDs: %w", err)
	}
	defer rows.Close()

	var results []domain.HighlightMatchIDRow
	for rows.Next() {
		var row domain.HighlightMatchIDRow
		if err := rows.Scan(&row.MatchID, &row.Outcome, &row.Section); err != nil {
			return nil, fmt.Errorf("CareerRepo.GetHighlightMatchIDs scan: %w", err)
		}
		results = append(results, row)
	}
	return results, rows.Err()
}

// GetHighlightPool retourne le pool complet des matchs éligibles pour la
// section "Matchs marquants" (mêmes critères d'éligibilité que Q9b, hors
// LIMIT et hors filtre best/worst). Utilisé pour calculer les cascade counts
// (available_experience, available_seasons) en respectant les autres filtres.
func (r *CareerRepo) GetHighlightPool(ctx context.Context) ([]domain.HighlightMatchPoolRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q9bHighlightPool, r.pdb.XUID)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetHighlightPool: %w", err)
	}
	defer rows.Close()

	var results []domain.HighlightMatchPoolRow
	for rows.Next() {
		var row domain.HighlightMatchPoolRow
		var startTime sql.NullTime
		if err := rows.Scan(&row.MatchID, &row.IsRanked, &startTime); err != nil {
			return nil, fmt.Errorf("CareerRepo.GetHighlightPool scan: %w", err)
		}
		if startTime.Valid {
			t := startTime.Time
			row.StartTime = &t
		}
		results = append(results, row)
	}
	return results, rows.Err()
}

// GetTopEncountersGlobal retourne les 10 joueurs les plus croisés au niveau
// carrière, hors XUIDs présents dans excludeXUIDs (typiquement les amis
// configurés). Lit shared.match_participants + shared.killer_victim_pairs.
func (r *CareerRepo) GetTopEncountersGlobal(ctx context.Context, excludeXUIDs []string) ([]domain.MatchEncounterRow, []domain.EncounterStatsRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, careerEncountersTimeout)
	defer cancel()

	// Construit la clause d'exclusion friends. Si liste vide, %s = "".
	excludeClause := ""
	args := []any{r.pdb.XUID, r.pdb.XUID, r.pdb.XUID, r.pdb.XUID, r.pdb.XUID, r.pdb.XUID, r.pdb.XUID}
	if len(excludeXUIDs) > 0 {
		placeholders := strings.Repeat("?,", len(excludeXUIDs))
		placeholders = strings.TrimRight(placeholders, ",")
		excludeClause = " AND es.xuid NOT IN (" + placeholders + ")"
		for _, x := range excludeXUIDs {
			args = append(args, x)
		}
	}
	sqlText := fmt.Sprintf(Q26CareerTopEncountersTpl, excludeClause)

	// Exécuté sur la connexion Player (qui a `shared` ATTACHé). La connexion
	// Shared directe ne connaît pas l'alias `shared` (cf. attachShared dans
	// pool.go).
	rows, err := r.pdb.ReadDB().Query(ctx, sqlText, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("CareerRepo.GetTopEncountersGlobal: %w", err)
	}
	defer rows.Close()

	var encounters []domain.MatchEncounterRow
	var stats []domain.EncounterStatsRaw
	for rows.Next() {
		var (
			xuid                                                          string
			gamertag                                                      string
			countTogether, allyCount, enemyCount                          int
			winsAsAlly, lossesAsAlly, winsVsEnemy, lossesVsEnemy          int
			killsDealt, deathsSuffered                                    int
			lastSeenAt                                                    sql.NullTime
		)
		if err := rows.Scan(
			&xuid, &gamertag, &countTogether,
			&allyCount, &enemyCount,
			&winsAsAlly, &lossesAsAlly, &winsVsEnemy, &lossesVsEnemy,
			&killsDealt, &deathsSuffered, &lastSeenAt,
		); err != nil {
			return nil, nil, fmt.Errorf("CareerRepo.GetTopEncountersGlobal scan: %w", err)
		}
		enc := domain.MatchEncounterRow{
			XUID:          xuid,
			Gamertag:      gamertag,
			CountTogether: countTogether,
			IsAlly:        allyCount >= enemyCount,
		}
		if allyCount > 0 || enemyCount > 0 {
			a := allyCount
			e := enemyCount
			enc.AllyCount = &a
			enc.EnemyCount = &e
		}
		if winsAsAlly+lossesAsAlly > 0 {
			r := float64(winsAsAlly) / float64(winsAsAlly+lossesAsAlly)
			enc.WinrateAsAlly = &r
		}
		if winsVsEnemy+lossesVsEnemy > 0 {
			r := float64(winsVsEnemy) / float64(winsVsEnemy+lossesVsEnemy)
			enc.WinrateVsEnemy = &r
		}
		kd := killsDealt
		ds := deathsSuffered
		enc.KillsDealt = &kd
		enc.DeathsSuffered = &ds
		if lastSeenAt.Valid {
			t := lastSeenAt.Time
			enc.LastSeenAt = &t
		}
		encounters = append(encounters, enc)

		stats = append(stats, domain.EncounterStatsRaw{
			XUID:           xuid,
			AllyCount:      allyCount,
			EnemyCount:     enemyCount,
			WinsAsAlly:     winsAsAlly,
			LossesAsAlly:   lossesAsAlly,
			WinsVsEnemy:    winsVsEnemy,
			LossesVsEnemy:  lossesVsEnemy,
			KillsDealt:     killsDealt,
			DeathsSuffered: deathsSuffered,
		})
	}
	return encounters, stats, rows.Err()
}

// GetRivals retourne les top némésis (par deaths DESC) et top souffre-douleur
// (par frags DESC), 10 chacun, depuis shared.killer_victim_pairs.
// Pas de seuil min — le ratio est calculé côté service.
func (r *CareerRepo) GetRivals(ctx context.Context) (nemeses, victims []domain.CareerRivalRawRow, err error) {
	ctx, cancel := context.WithTimeout(ctx, careerRivalsTimeout)
	defer cancel()

	nemeses, err = r.queryRivals(ctx, "deaths")
	if err != nil {
		return nil, nil, err
	}
	victims, err = r.queryRivals(ctx, "frags")
	if err != nil {
		return nil, nil, err
	}
	return nemeses, victims, nil
}

// queryRivals exécute Q27CareerRivalsTpl avec orderCol pour le tri (frags ou deaths).
func (r *CareerRepo) queryRivals(ctx context.Context, orderCol string) ([]domain.CareerRivalRawRow, error) {
	if orderCol != "frags" && orderCol != "deaths" {
		return nil, fmt.Errorf("CareerRepo.queryRivals: invalid order column %q", orderCol)
	}
	sqlText := fmt.Sprintf(Q27CareerRivalsTpl, orderCol)
	// On exécute sur la connexion Player (qui a `shared` ATTACHé) — elle
	// résout les `shared.*` du SQL. La connexion Shared directe ne connaît
	// pas l'alias `shared`.
	rows, err := r.pdb.ReadDB().Query(
		ctx, sqlText,
		r.pdb.XUID, r.pdb.XUID, r.pdb.XUID, r.pdb.XUID, r.pdb.XUID, r.pdb.XUID,
	)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.queryRivals(%s): %w", orderCol, err)
	}
	defer rows.Close()

	var results []domain.CareerRivalRawRow
	for rows.Next() {
		var row domain.CareerRivalRawRow
		if err := rows.Scan(&row.XUID, &row.Gamertag, &row.Frags, &row.Deaths, &row.MatchCount); err != nil {
			return nil, fmt.Errorf("CareerRepo.queryRivals(%s) scan: %w", orderCol, err)
		}
		results = append(results, row)
	}
	return results, rows.Err()
}

// GetEncounters retourne les adversaires/coéquipiers fréquents.
func (r *CareerRepo) GetEncounters(ctx context.Context) ([]domain.EncounterRawRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	rows, err := r.pdb.Shared.Query(ctx, Q10Encounters, r.pdb.XUID)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetEncounters: %w", err)
	}
	defer rows.Close()

	var results []domain.EncounterRawRow
	for rows.Next() {
		var e domain.EncounterRawRow
		if err := rows.Scan(
			&e.XUID, &e.Gamertag, &e.MatchCount, &e.AsTeammate, &e.AsEnemy, &e.AvgKDA,
		); err != nil {
			return nil, fmt.Errorf("CareerRepo.GetEncounters scan: %w", err)
		}
		results = append(results, e)
	}
	return results, rows.Err()
}
