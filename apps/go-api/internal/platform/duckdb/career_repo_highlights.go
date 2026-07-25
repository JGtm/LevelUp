// Package duckdb — career_repo_highlights.go : GetHighlightMatchIDs +
// GetHighlightPool (section "Matchs marquants" page Carrière) + helpers de
// traduction FR. Découpé de career_repo.go (god-file split, refactor 2026-05-27).
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
			ts := StartTimeCanonicalSQL("r")
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
	PerfScore      float64
	DominanceFlag  int
	HadBotTeammate bool
}

// loadHighlightCandidates exécute le pipeline split commun à GetHighlightMatchIDs
// et GetHighlightPool (ADR 0016) :
//   - Phase A : pme (player) avec filtre perf_score (had_bot_teammate transmis
//     au tri Go pour exclusion asymétrique WIN-conservée / LOSS-rejetée).
//   - Phase B : mp + r (shared via SharedReader) avec filtres time_played +
//     NOT is_firefight + outcome ∈ {2,3} + IN match_ids + clause dynamique.
//   - Phase C : retourne les rows enrichies + map pme[match_id]→{perf_score, dominance, had_bot_teammate}.
func (r *CareerRepo) loadHighlightCandidates(
	ctx context.Context, extraClause string, extraArgs []any,
) ([]highlightSharedRow, map[string]highlightPMEEntry, error) {
	pmes := make(map[string]highlightPMEEntry)
	pmeRows, err := r.pdb.Player.QueryRecovered(ctx, Q9TopMatchesPlayer)
	if err != nil {
		return nil, nil, fmt.Errorf("loadHighlightCandidates: phase A: %w", err)
	}
	matchIDs := make([]string, 0)
	for pmeRows.Next() {
		var mid string
		var p highlightPMEEntry
		if err := pmeRows.Scan(&mid, &p.PerfScore, &p.DominanceFlag, &p.HadBotTeammate); err != nil {
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
	query := resolveCampaignExclusion(
		fmt.Sprintf(Q9bHighlightSharedTpl, Placeholders(len(matchIDs)), extraClause), r.titleSlug(), "r")
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
		MatchID        string
		Outcome        int
		PerfScore      float64
		DominanceFlag  int
		HadBotTeammate bool
	}
	var wins, losses []sortableRow
	for _, c := range cands {
		pme, ok := pmes[c.MatchID]
		if !ok {
			continue
		}
		row := sortableRow{
			MatchID:        c.MatchID,
			Outcome:        c.Outcome,
			PerfScore:      pme.PerfScore,
			DominanceFlag:  pme.DominanceFlag,
			HadBotTeammate: pme.HadBotTeammate,
		}
		switch c.Outcome {
		case 2:
			wins = append(wins, row)
		case 3:
			// Exclusion asymétrique : un LOSS avec bot coéquipier ne permet
			// pas d'isoler la responsabilité du joueur (équipe handicapée 4v3).
			// Les WIN avec bot sont conservés : la perf personnelle reste
			// méritoire malgré le handicap d'équipe.
			if pme.HadBotTeammate {
				continue
			}
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
		results = append(results, domain.HighlightMatchIDRow{
			MatchID: w.MatchID, Outcome: w.Outcome, Section: 1,
			HadBotTeammate: w.HadBotTeammate,
		})
	}
	for _, l := range losses {
		// LOSS sans bot par construction (les LOSS avec bot ont été skippés
		// au-dessus), HadBotTeammate est donc toujours false ici.
		results = append(results, domain.HighlightMatchIDRow{
			MatchID: l.MatchID, Outcome: l.Outcome, Section: 2,
			HadBotTeammate: l.HadBotTeammate,
		})
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
// non trouvée. Le SQL vit dans mode_name_tr.go, source unique du littéral
// (garde-rail no_mode_name_tr_literal_test.go).
func (r *CareerRepo) LoadModeTranslationsFR(ctx context.Context, modeENs []string) (map[string]string, error) {
	if len(modeENs) == 0 || r.pdb == nil || r.pdb.Metadata == nil {
		return nil, nil
	}
	out, err := queryModeNameTrFR(ctx, r.pdb.Metadata, modeENs)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.LoadModeTranslationsFR: %w", err)
	}
	return out, nil
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
