// Package duckdb — CareerRepo : données de progression de carrière.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
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
//
// Split cross-DB en 2 phases (ADR 0016) :
//   - Phase A : match_skill_rank (player) sur pdb.Player.
//   - Phase B : match_registry (shared) pour start_time + playlist via
//     SharedReader avec WHERE match_id IN (...).
//   - Phase C : tri par start_time ASC + calcul rating_delta côté Go via
//     LAG manuel par (rating_type, playlist_group).
//
// lusrPlayerRow projette une ligne match_skill_rank côté player.
type lusrPlayerRow struct {
	MatchID       string
	RatingType    string
	RatingValue   float64
	TierLabel     *string
	PlaylistGroup *string
	Tier          sql.NullString
	SubTier       sql.NullInt16
}

// lusrRegistryInfo agrège start_time + playlist depuis shared.match_registry.
type lusrRegistryInfo struct {
	RecordedAt   *time.Time
	PlaylistName string
	PlaylistID   string
}

func (r *CareerRepo) GetLUSRHistory(ctx context.Context) ([]domain.LUSRCheckpointDTO, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	playerRows, matchIDs, err := r.loadLUSRPlayerRows(ctx)
	if err != nil {
		return nil, err
	}
	if len(playerRows) == 0 {
		return nil, nil
	}

	registryByMatch, err := r.loadLUSRRegistryInfo(ctx, matchIDs)
	if err != nil {
		return nil, err
	}

	results := assembleLUSRResults(playerRows, registryByMatch)
	sortLUSRResultsByRecordedAt(results)
	computeLUSRRatingDeltas(results)

	r.enrichLUSRPlaylistNames(ctx, results)
	return results, nil
}

// loadLUSRPlayerRows charge match_skill_rank du joueur (phase A).
func (r *CareerRepo) loadLUSRPlayerRows(ctx context.Context) ([]lusrPlayerRow, []string, error) {
	rows, err := r.pdb.Player.Query(ctx, Q8LUSRHistoryPlayer)
	if err != nil {
		return nil, nil, fmt.Errorf("CareerRepo.GetLUSRHistory: phase A: %w", err)
	}
	defer rows.Close()

	var playerRows []lusrPlayerRow
	matchIDs := make([]string, 0)
	for rows.Next() {
		var p lusrPlayerRow
		if err := rows.Scan(&p.MatchID, &p.RatingType, &p.RatingValue, &p.TierLabel,
			&p.PlaylistGroup, &p.Tier, &p.SubTier); err != nil {
			return nil, nil, fmt.Errorf("CareerRepo.GetLUSRHistory scan A: %w", err)
		}
		playerRows = append(playerRows, p)
		matchIDs = append(matchIDs, p.MatchID)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return playerRows, matchIDs, nil
}

// loadLUSRRegistryInfo enrichit match_registry (phase B) via SharedReader.
func (r *CareerRepo) loadLUSRRegistryInfo(ctx context.Context, matchIDs []string) (map[string]lusrRegistryInfo, error) {
	out := make(map[string]lusrRegistryInfo, len(matchIDs))
	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetLUSRHistory: shared reader: %w", err)
	}
	defer release()
	query := fmt.Sprintf(Q8LUSRHistoryRegistryTpl, Placeholders(len(matchIDs)))
	regRows, err := sharedDB.QueryContext(ctx, query, ToAnySlice(matchIDs)...)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetLUSRHistory: phase B: %w", err)
	}
	defer regRows.Close()
	for regRows.Next() {
		var mid string
		var info lusrRegistryInfo
		var ts sql.NullTime
		if err := regRows.Scan(&mid, &ts, &info.PlaylistName, &info.PlaylistID); err != nil {
			return nil, fmt.Errorf("CareerRepo.GetLUSRHistory scan B: %w", err)
		}
		if ts.Valid {
			t := ts.Time
			info.RecordedAt = &t
		}
		out[mid] = info
	}
	return out, nil
}

// assembleLUSRResults compose les DTOs en croisant phases A et B.
func assembleLUSRResults(playerRows []lusrPlayerRow, registryByMatch map[string]lusrRegistryInfo) []domain.LUSRCheckpointDTO {
	results := make([]domain.LUSRCheckpointDTO, 0, len(playerRows))
	for _, p := range playerRows {
		cp := domain.LUSRCheckpointDTO{
			MatchID:       p.MatchID,
			RatingType:    p.RatingType,
			RatingValue:   p.RatingValue,
			TierLabel:     p.TierLabel,
			PlaylistGroup: p.PlaylistGroup,
		}
		if info, ok := registryByMatch[p.MatchID]; ok {
			cp.RecordedAt = info.RecordedAt
			cp.PlaylistName = info.PlaylistName
			cp.PlaylistID = info.PlaylistID
		}
		tierLabel := ""
		if cp.TierLabel != nil {
			tierLabel = *cp.TierLabel
		}
		cp.BadgeImageURL = buildHomeSkillPeakBadgeURL(
			optionalNullStringValue(p.Tier),
			tierLabel,
			optionalNullInt16Value(p.SubTier),
			homeStaticTitleSlug,
			0,
		)
		results = append(results, cp)
	}
	return results
}

// sortLUSRResultsByRecordedAt trie par recorded_at ASC (NULLS LAST).
func sortLUSRResultsByRecordedAt(results []domain.LUSRCheckpointDTO) {
	sort.SliceStable(results, func(i, j int) bool {
		ai, aj := results[i].RecordedAt, results[j].RecordedAt
		if ai == nil && aj == nil {
			return false
		}
		if ai == nil {
			return false
		}
		if aj == nil {
			return true
		}
		return ai.Before(*aj)
	})
}

// computeLUSRRatingDeltas calcule rating_delta = current - prev par (rating_type, playlist_group).
func computeLUSRRatingDeltas(results []domain.LUSRCheckpointDTO) {
	type lagKey struct {
		ratingType    string
		playlistGroup string
	}
	prev := make(map[lagKey]float64)
	for i := range results {
		key := lagKey{ratingType: results[i].RatingType}
		if results[i].PlaylistGroup != nil {
			key.playlistGroup = *results[i].PlaylistGroup
		}
		if p, ok := prev[key]; ok {
			delta := results[i].RatingValue - p
			results[i].RatingDelta = &delta
		}
		prev[key] = results[i].RatingValue
	}
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

// GetTopMatches retourne les 10 meilleurs (WIN) + 10 moins bons (LOSS) matchs
// par performance_score.
//
// Split cross-DB en 2 phases (ADR 0016) :
//   - Phase A : player_match_enrichment (pme) sur pdb.Player avec filtre
//     performance_score IS NOT NULL + NOT had_bot_teammate.
//   - Phase B : match_participants + match_registry (shared) via SharedReader
//     avec filtres time_played>=180 + NOT is_firefight + IN match_ids.
//   - Phase C : merge + sections WIN/LOSS + tri par dominance flag (priorité
//     section) + perf_score + top 10 chaque section.
//
// topMatchPMERow projette player_match_enrichment (perf + dominance).
type topMatchPMERow struct {
	matchID       string
	perfScore     float64
	dominanceFlag int
}

func (r *CareerRepo) GetTopMatches(ctx context.Context) ([]domain.TopMatchRawRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	pmes, err := r.loadTopMatchPMERows(ctx)
	if err != nil {
		return nil, err
	}
	if len(pmes) == 0 {
		return nil, nil
	}

	matchIDs := make([]string, 0, len(pmes))
	for id := range pmes {
		matchIDs = append(matchIDs, id)
	}

	enriched, err := r.loadTopMatchSharedRows(ctx, matchIDs, pmes)
	if err != nil {
		return nil, err
	}

	wins, losses := splitWinsLossesAndSortTopMatches(enriched)
	results := make([]domain.TopMatchRawRow, 0, len(wins)+len(losses))
	results = append(results, wins...)
	results = append(results, losses...)
	return results, nil
}

// loadTopMatchPMERows charge phase A : player_match_enrichment.
func (r *CareerRepo) loadTopMatchPMERows(ctx context.Context) (map[string]topMatchPMERow, error) {
	pmes := make(map[string]topMatchPMERow)
	pmeRows, err := r.pdb.Player.Query(ctx, Q9TopMatchesPlayer)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetTopMatches: phase A: %w", err)
	}
	defer pmeRows.Close()
	for pmeRows.Next() {
		var p topMatchPMERow
		if err := pmeRows.Scan(&p.matchID, &p.perfScore, &p.dominanceFlag); err != nil {
			return nil, fmt.Errorf("CareerRepo.GetTopMatches scan A: %w", err)
		}
		pmes[p.matchID] = p
	}
	return pmes, nil
}

// loadTopMatchSharedRows charge phase B (shared) + merge avec pmes.
func (r *CareerRepo) loadTopMatchSharedRows(
	ctx context.Context, matchIDs []string, pmes map[string]topMatchPMERow,
) ([]domain.TopMatchRawRow, error) {
	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetTopMatches: shared reader: %w", err)
	}
	defer release()
	query := fmt.Sprintf(Q9TopMatchesSharedTpl, Placeholders(len(matchIDs)))
	sharedArgs := make([]any, 0, len(matchIDs)+1)
	sharedArgs = append(sharedArgs, r.pdb.XUID)
	sharedArgs = append(sharedArgs, ToAnySlice(matchIDs)...)
	sharedRows, err := sharedDB.QueryContext(ctx, query, sharedArgs...)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetTopMatches: phase B: %w", err)
	}
	defer sharedRows.Close()

	var enriched []domain.TopMatchRawRow
	for sharedRows.Next() {
		var m domain.TopMatchRawRow
		if err := sharedRows.Scan(
			&m.MatchID, &m.StartTime,
			&m.MapName, &m.PairName, &m.PlaylistName,
			&m.Outcome, &m.Kills, &m.Deaths, &m.KDA,
			&m.TeamMMR, &m.EnemyMMR,
		); err != nil {
			return nil, fmt.Errorf("CareerRepo.GetTopMatches scan B: %w", err)
		}
		if pme, ok := pmes[m.MatchID]; ok {
			m.PerformanceScore = pme.perfScore
			m.DominanceFlag = pme.dominanceFlag
			enriched = append(enriched, m)
		}
	}
	if err := sharedRows.Err(); err != nil {
		return nil, err
	}
	return enriched, nil
}

// splitWinsLossesAndSortTopMatches répartit en sections WIN/LOSS, trie chaque
// section et coupe au top 10.
//
// WIN : dominance ∈ (5,3,1) prioritaires (remontada/contre-remontada/domination), tri DESC.
// LOSS : dominance ∈ (4,2) prioritaires (débandade/humiliation), tri DESC.
// Tiebreak : perf_score DESC (WIN) ou ASC (LOSS, les moins bons en premier).
func splitWinsLossesAndSortTopMatches(enriched []domain.TopMatchRawRow) ([]domain.TopMatchRawRow, []domain.TopMatchRawRow) {
	var wins, losses []domain.TopMatchRawRow
	for _, m := range enriched {
		switch m.Outcome {
		case 2:
			wins = append(wins, m)
		case 3:
			losses = append(losses, m)
		}
	}
	sort.SliceStable(wins, func(i, j int) bool {
		pi := topMatchDominancePriority(wins[i].DominanceFlag, []int{5, 3, 1})
		pj := topMatchDominancePriority(wins[j].DominanceFlag, []int{5, 3, 1})
		if pi != pj {
			return pi > pj
		}
		return wins[i].PerformanceScore > wins[j].PerformanceScore
	})
	sort.SliceStable(losses, func(i, j int) bool {
		pi := topMatchDominancePriority(losses[i].DominanceFlag, []int{4, 2})
		pj := topMatchDominancePriority(losses[j].DominanceFlag, []int{4, 2})
		if pi != pj {
			return pi > pj
		}
		return losses[i].PerformanceScore < losses[j].PerformanceScore
	})
	if len(wins) > 10 {
		wins = wins[:10]
	}
	if len(losses) > 10 {
		losses = losses[:10]
	}
	return wins, losses
}

// topMatchDominancePriority retourne la valeur du dominance_flag si présent
// dans le set prioritaire, sinon 0. Reproduit le `CASE WHEN ... THEN flag ELSE 0`
// du SQL historique Q9.
func topMatchDominancePriority(flag int, priorities []int) int {
	for _, p := range priorities {
		if flag == p {
			return flag
		}
	}
	return 0
}

// buildHighlightFilterClause traduit `domain.CareerHighlightFilters` en
// clause SQL additionnelle (à injecter via Sprintf dans Q9bHighlightMatchIDsTpl)
// + les valeurs liées correspondantes. Retourne ("", nil) si aucun filtre actif.
//
// Dimensions gérées : Experience (is_ranked), Seasons (start_time), Modes
// (pair_name IN ...), Playlists (playlist_name IN ...).
func buildHighlightFilterClause(filters domain.CareerHighlightFilters) (string, []any) {
	parts := make([]string, 0, 4)
	args := make([]any, 0, 8)

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

	if len(filters.ModeRawSources) > 0 {
		ph := make([]string, len(filters.ModeRawSources))
		for i, m := range filters.ModeRawSources {
			ph[i] = "?"
			args = append(args, m)
		}
		// Comparaison sur la même expression que celle utilisée pour normaliser
		// côté pool (analysis.NormalizeModeLabel(coalesce(pair_name_fr, pair_name))).
		parts = append(parts, "COALESCE(NULLIF(r.pair_name_fr, ''), r.pair_name, '') IN ("+strings.Join(ph, ", ")+")")
	}

	if len(filters.PlaylistNamesRaw) > 0 {
		ph := make([]string, len(filters.PlaylistNamesRaw))
		for i, p := range filters.PlaylistNamesRaw {
			ph[i] = "?"
			args = append(args, p)
		}
		// Comparaison sur la même expression que celle utilisée pour normaliser
		// côté pool (COALESCE(playlist_name_fr, playlist_name)).
		parts = append(parts, "COALESCE(NULLIF(r.playlist_name_fr, ''), r.playlist_name, '') IN ("+strings.Join(ph, ", ")+")")
	}

	if len(parts) == 0 {
		return "", nil
	}
	return " AND " + strings.Join(parts, " AND "), args
}

// highlightSharedRow capture le résultat de Q9bHighlightSharedTpl (Phase B).
type highlightSharedRow struct {
	MatchID            string
	Outcome            int
	IsRanked           bool
	StartTime          *time.Time
	PairNameSource     string
	PlaylistNameSource string
	PlaylistID         string
}

// highlightPMEEntry capture les colonnes pme nécessaires au tri WIN/LOSS.
type highlightPMEEntry struct {
	PerfScore     float64
	DominanceFlag int
}

// loadHighlightCandidates exécute le pipeline split commun à GetHighlightMatchIDs
// et GetHighlightPool (ADR 0016) :
//   - Phase A : pme (player) avec filtre perf_score + NOT had_bot_teammate.
//   - Phase B : mp + r (shared via SharedReader) avec filtres time_played +
//     NOT is_firefight + outcome ∈ {2,3} + IN match_ids + clause dynamique.
//   - Phase C : retourne les rows enrichies + map pme[match_id]→{perf_score, dominance}.
func (r *CareerRepo) loadHighlightCandidates(
	ctx context.Context, extraClause string, extraArgs []any,
) ([]highlightSharedRow, map[string]highlightPMEEntry, error) {
	pmes := make(map[string]highlightPMEEntry)
	pmeRows, err := r.pdb.Player.Query(ctx, Q9TopMatchesPlayer)
	if err != nil {
		return nil, nil, fmt.Errorf("loadHighlightCandidates: phase A: %w", err)
	}
	matchIDs := make([]string, 0)
	for pmeRows.Next() {
		var mid string
		var p highlightPMEEntry
		if err := pmeRows.Scan(&mid, &p.PerfScore, &p.DominanceFlag); err != nil {
			pmeRows.Close()
			return nil, nil, fmt.Errorf("loadHighlightCandidates scan A: %w", err)
		}
		pmes[mid] = p
		matchIDs = append(matchIDs, mid)
	}
	pmeRows.Close()
	if len(matchIDs) == 0 {
		return nil, pmes, nil
	}

	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("loadHighlightCandidates: shared reader: %w", err)
	}
	defer release()
	query := fmt.Sprintf(Q9bHighlightSharedTpl, Placeholders(len(matchIDs)), extraClause)
	args := make([]any, 0, 1+len(matchIDs)+len(extraArgs))
	args = append(args, r.pdb.XUID)
	args = append(args, ToAnySlice(matchIDs)...)
	args = append(args, extraArgs...)
	rows, err := sharedDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("loadHighlightCandidates: phase B: %w", err)
	}
	defer rows.Close()

	var out []highlightSharedRow
	for rows.Next() {
		var row highlightSharedRow
		var startTime sql.NullTime
		if err := rows.Scan(&row.MatchID, &row.Outcome, &row.IsRanked, &startTime,
			&row.PairNameSource, &row.PlaylistNameSource, &row.PlaylistID); err != nil {
			return nil, nil, fmt.Errorf("loadHighlightCandidates scan B: %w", err)
		}
		if startTime.Valid {
			t := startTime.Time
			row.StartTime = &t
		}
		out = append(out, row)
	}
	return out, pmes, rows.Err()
}

// GetHighlightMatchIDs retourne 15 best + 15 worst match_ids triés par
// performance + dominance prio. Les rows sont retournées dans l'ordre
// _s ASC : best d'abord, worst ensuite.
//
// `filters` applique optionnellement Experience (is_ranked) et SeasonRanges
// (date windows) via une clause SQL additionnelle dérivée en interne.
//
// Split cross-DB en 2 phases (ADR 0016, cf. loadHighlightCandidates).
func (r *CareerRepo) GetHighlightMatchIDs(ctx context.Context, filters domain.CareerHighlightFilters) ([]domain.HighlightMatchIDRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	clause, extraArgs := buildHighlightFilterClause(filters)
	cands, pmes, err := r.loadHighlightCandidates(ctx, clause, extraArgs)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetHighlightMatchIDs: %w", err)
	}
	if len(cands) == 0 {
		return nil, nil
	}

	type sortableRow struct {
		MatchID       string
		Outcome       int
		PerfScore     float64
		DominanceFlag int
	}
	var wins, losses []sortableRow
	for _, c := range cands {
		pme, ok := pmes[c.MatchID]
		if !ok {
			continue
		}
		row := sortableRow{
			MatchID:       c.MatchID,
			Outcome:       c.Outcome,
			PerfScore:     pme.PerfScore,
			DominanceFlag: pme.DominanceFlag,
		}
		switch c.Outcome {
		case 2:
			wins = append(wins, row)
		case 3:
			losses = append(losses, row)
		}
	}
	sort.SliceStable(wins, func(i, j int) bool {
		pi := topMatchDominancePriority(wins[i].DominanceFlag, []int{5, 3, 1})
		pj := topMatchDominancePriority(wins[j].DominanceFlag, []int{5, 3, 1})
		if pi != pj {
			return pi > pj
		}
		return wins[i].PerfScore > wins[j].PerfScore
	})
	sort.SliceStable(losses, func(i, j int) bool {
		pi := topMatchDominancePriority(losses[i].DominanceFlag, []int{4, 2})
		pj := topMatchDominancePriority(losses[j].DominanceFlag, []int{4, 2})
		if pi != pj {
			return pi > pj
		}
		return losses[i].PerfScore < losses[j].PerfScore
	})
	if len(wins) > 15 {
		wins = wins[:15]
	}
	if len(losses) > 15 {
		losses = losses[:15]
	}
	results := make([]domain.HighlightMatchIDRow, 0, len(wins)+len(losses))
	for _, w := range wins {
		results = append(results, domain.HighlightMatchIDRow{MatchID: w.MatchID, Outcome: w.Outcome, Section: 1})
	}
	for _, l := range losses {
		results = append(results, domain.HighlightMatchIDRow{MatchID: l.MatchID, Outcome: l.Outcome, Section: 2})
	}
	return results, nil
}

// GetHighlightPool retourne le pool complet des matchs éligibles pour la
// section "Matchs marquants" (mêmes critères d'éligibilité que Q9b, hors
// LIMIT et hors filtre best/worst). Utilisé pour calculer les cascade counts.
//
// Split cross-DB en 2 phases (ADR 0016, cf. loadHighlightCandidates).
func (r *CareerRepo) GetHighlightPool(ctx context.Context) ([]domain.HighlightMatchPoolRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	cands, _, err := r.loadHighlightCandidates(ctx, "", nil)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetHighlightPool: %w", err)
	}

	results := make([]domain.HighlightMatchPoolRow, 0, len(cands))
	for _, c := range cands {
		row := domain.HighlightMatchPoolRow{
			MatchID:         c.MatchID,
			IsRanked:        c.IsRanked,
			StartTime:       c.StartTime,
			ModeUISource:    c.PairNameSource,
			PlaylistNameRaw: c.PlaylistNameSource,
			PlaylistID:      c.PlaylistID,
		}
		row.ModeUI = analysis.NormalizeModeLabel(row.ModeUISource)
		row.PlaylistName = row.PlaylistNameRaw
		results = append(results, row)
	}
	return results, nil
}

// LoadModeTranslationsFR retourne le mapping EN→FR depuis metadata.mode_name_tr
// (lang='fr'). Best-effort : silencieusement vide si Metadata absent ou table
// non trouvée. Calqué sur SquadRepo.LoadModeTranslationsFR.
func (r *CareerRepo) LoadModeTranslationsFR(ctx context.Context, modeENs []string) (map[string]string, error) {
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
		return nil, fmt.Errorf("CareerRepo.LoadModeTranslationsFR: %w", err)
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

// LoadPlaylistAssetTranslationsFR retourne le mapping playlist_id→nom FR via
// metadata.asset_translations. Best-effort : nil silencieux si absent.
// Calqué sur enrichLUSRPlaylistNames + SquadRepo.LoadAssetTranslationsFR.
func (r *CareerRepo) LoadPlaylistAssetTranslationsFR(ctx context.Context, playlistIDs []string) (map[string]string, error) {
	if len(playlistIDs) == 0 || r.pdb == nil || r.pdb.Metadata == nil {
		return nil, nil
	}
	out, err := NewMetadataRepoFromDB(r.pdb.Metadata).ResolveAssetNamesBulk(
		ctx, "playlist", playlistIDs, PreferredLangsForLocale("fr"),
	)
	if err != nil && isTableNotFoundErr(err) {
		return nil, nil
	}
	return out, err
}

// GetTopEncountersGlobal retourne les 10 joueurs les plus croisés au niveau
// carrière, hors XUIDs présents dans excludeXUIDs (typiquement les amis
// configurés). Lit match_participants + killer_victim_pairs via SharedReader.
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

	// migré vers SharedReader. La query est shared-only
	// (match_participants, match_registry, killer_victim_pairs, v_gamertag_lookup)
	// — tables/vues au niveau root du catalogue shared_matches_v2.duckdb.
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("CareerRepo.GetTopEncountersGlobal: shared reader: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("CareerRepo.GetTopEncountersGlobal: %w", err)
	}
	defer rows.Close()

	var encounters []domain.MatchEncounterRow
	var stats []domain.EncounterStatsRaw
	for rows.Next() {
		var (
			xuid                                                 string
			gamertag                                             string
			countTogether, allyCount, enemyCount                 int
			winsAsAlly, lossesAsAlly, winsVsEnemy, lossesVsEnemy int
			killsDealt, deathsSuffered                           int
			lastSeenAt                                           sql.NullTime
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
// (par frags DESC), 10 chacun, depuis killer_victim_pairs via SharedReader.
// Pas de seuil min — le ratio est calculé côté service.
//
// rivalsOrderColXxx : colonnes SQL acceptées par queryRivals.
const (
	rivalsOrderColFrags  = "frags"
	rivalsOrderColDeaths = "deaths"
)

func (r *CareerRepo) GetRivals(ctx context.Context) (nemeses, victims []domain.CareerRivalRawRow, err error) {
	ctx, cancel := context.WithTimeout(ctx, careerRivalsTimeout)
	defer cancel()

	nemeses, err = r.queryRivals(ctx, rivalsOrderColDeaths)
	if err != nil {
		return nil, nil, err
	}
	victims, err = r.queryRivals(ctx, rivalsOrderColFrags)
	if err != nil {
		return nil, nil, err
	}
	return nemeses, victims, nil
}

// queryRivals exécute Q27CareerRivalsTpl avec orderCol pour le tri (frags ou deaths).
func (r *CareerRepo) queryRivals(ctx context.Context, orderCol string) ([]domain.CareerRivalRawRow, error) {
	if orderCol != rivalsOrderColFrags && orderCol != rivalsOrderColDeaths {
		return nil, fmt.Errorf("CareerRepo.queryRivals: invalid order column %q", orderCol)
	}
	sqlText := fmt.Sprintf(Q27CareerRivalsTpl, orderCol)
	// migré vers SharedReader. Q27 est shared-only
	// (killer_victim_pairs + v_gamertag_lookup, tous root-level).
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.queryRivals(%s): shared reader: %w", orderCol, err)
	}
	defer release()

	rows, err := db.QueryContext(
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

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetEncounters: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, Q10Encounters, r.pdb.XUID)
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

// GetCSRSnapshots retourne les classements CSR du joueur depuis player_csr_snapshots.
// Retourne slice vide (pas d'erreur) si la table n'existe pas ou est vide.
func (r *CareerRepo) GetCSRSnapshots(ctx context.Context) ([]domain.CareerPlaylistCSR, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q26csrSnapshots)
	if err != nil {
		if isTableNotFoundErr(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("CareerRepo.GetCSRSnapshots: %w", err)
	}
	defer rows.Close()

	var out []domain.CareerPlaylistCSR
	for rows.Next() {
		var p domain.CareerPlaylistCSR
		var seasonID string // col 5 — stored in DB but not exposed in the DTO
		if err := rows.Scan(
			&p.PlaylistID, &p.PlaylistName, &p.Queue, &p.Input,
			&seasonID,
			&p.Current.Value, &p.Current.Tier, &p.Current.SubTier, &p.Current.MeasurementMatchesRemaining,
			&p.Season.Value, &p.Season.Tier, &p.Season.SubTier,
			&p.AllTime.Value, &p.AllTime.Tier, &p.AllTime.SubTier,
		); err != nil {
			return nil, fmt.Errorf("CareerRepo.GetCSRSnapshots scan: %w", err)
		}
		p.Current.BadgeImageURL = buildHomeSkillPeakBadgeURL(
			p.Current.Tier, "", p.Current.SubTier, homeStaticTitleSlug,
			p.Current.MeasurementMatchesRemaining,
		)
		p.Season.BadgeImageURL = buildHomeSkillPeakBadgeURL(
			p.Season.Tier, "", p.Season.SubTier, homeStaticTitleSlug, 0,
		)
		p.AllTime.BadgeImageURL = buildHomeSkillPeakBadgeURL(
			p.AllTime.Tier, "", p.AllTime.SubTier, homeStaticTitleSlug, 0,
		)
		out = append(out, p)
	}
	return out, rows.Err()
}
