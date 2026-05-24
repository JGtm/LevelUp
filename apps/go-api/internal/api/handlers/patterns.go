package handlers

// patterns.go — endpoint GET /patterns pour le Pattern Engine v3.
//
// Route : GET /api/v1/players/{player_slug}/patterns?n=50
// Query params :
//   - n : nombre de matchs récents à analyser (défaut 50, min 10, max 200)
//
// Ref : .ai/PLAN_PATTERN_ENGINE_V3.md

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/patterns"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
)

const (
	patternDefaultN = 50
	patternMinN     = 10
	patternMaxN     = 200
	patternTimeout  = 30 * time.Second
)

// PatternsHandler gère l'endpoint GET /patterns.
type PatternsHandler struct {
	resolve   ProgressionResolver
	titleSlug string
}

// NewPatternsHandler construit le handler.
func NewPatternsHandler(resolve ProgressionResolver, titleSlug string) *PatternsHandler {
	if titleSlug == "" {
		titleSlug = title.DefaultSlug
	}
	return &PatternsHandler{resolve: resolve, titleSlug: titleSlug}
}

// Mount enregistre la route sur le router chi.
func (h *PatternsHandler) Mount(r chi.Router) {
	r.Get("/patterns", h.GetPatterns)
}

// GetPatterns : GET /patterns?n=50
func (h *PatternsHandler) GetPatterns(w http.ResponseWriter, r *http.Request) {
	playerSlug := chi.URLParam(r, "player_slug")
	pdb, err := h.resolve(r.Context(), playerSlug)
	if err != nil {
		slog.WarnContext(r.Context(), "patterns: player not found", "player_slug", playerSlug, "err", err)
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	n := patternDefaultN
	if nStr := r.URL.Query().Get("n"); nStr != "" {
		if v, parseErr := strconv.Atoi(nStr); parseErr == nil && v >= patternMinN && v <= patternMaxN {
			n = v
		} else {
			slog.DebugContext(r.Context(), "patterns: n param ignoré (hors plage ou invalide)", "raw", nStr, "default", patternDefaultN)
		}
	}

	slog.DebugContext(r.Context(), "patterns: chargement des rows", "player_slug", playerSlug, "n", n)

	rows, err := loadPatternRows(r.Context(), pdb, n)
	if err != nil {
		slog.ErrorContext(r.Context(), "patterns: échec chargement rows", "player_slug", playerSlug, "n", n, "err", err)
		writeError(r.Context(), w, http.StatusInternalServerError, "load_error", err.Error())
		return
	}
	if len(rows) == 0 {
		slog.InfoContext(r.Context(), "patterns: aucun row chargé — rapport vide retourné", "player_slug", playerSlug)
	}

	report := patterns.Analyze(patterns.AnalyzeInput{
		Rows:   rows,
		N:      n,
		Config: patterns.DefaultPatternConfig(),
		Now:    time.Now().UTC(),
	})
	slog.DebugContext(r.Context(), "patterns: analyse terminée",
		"player_slug", playerSlug,
		"rows", len(rows),
		"context_patterns", len(report.ContextPatterns),
		"behavior_patterns", len(report.BehaviorPatterns),
		"levers", len(report.Levers),
	)
	writeJSON(w, http.StatusOK, report)
}

// loadPatternRows charge les données nécessaires à l'analyse de patterns.
//
// Split cross-DB (ADR 0016) :
//   - Phase 1 (SharedReader) : match_participants + match_registry → stats brutes
//   - Phase 2 (Player) : player_match_enrichment + match_skill_rank → enrichissements
//   - Merge en Go + calcul des deltas LUSR/CSR
func loadPatternRows(ctx context.Context, pdb *duckdb.PlayerDB, limit int) ([]patterns.MatchRow, error) {
	ctx, cancel := context.WithTimeout(ctx, patternTimeout)
	defer cancel()

	shared, err := loadPatternShared(ctx, pdb, limit)
	if err != nil {
		return nil, fmt.Errorf("loadPatternRows shared: %w", err)
	}
	if len(shared) == 0 {
		return nil, nil
	}

	matchIDs := make([]string, len(shared))
	for i, r := range shared {
		matchIDs[i] = r.MatchID
	}

	enrichMap, err := loadPatternEnrichments(ctx, pdb, matchIDs)
	if err != nil {
		return nil, fmt.Errorf("loadPatternRows enrichments: %w", err)
	}

	skillMap, err := loadPatternSkillRanks(ctx, pdb, matchIDs)
	if err != nil {
		return nil, fmt.Errorf("loadPatternRows skill_ranks: %w", err)
	}

	rows := mergePatternRows(shared, enrichMap, skillMap)
	computeSkillDeltas(rows)
	return rows, nil
}

// sharedPatternRow est le résultat intermédiaire de la phase 1.
type sharedPatternRow struct {
	MatchID          string
	PlayedAt         time.Time
	Mode             string
	MapID            string
	Outcome          int
	DurationSec      int
	Kills            int
	Deaths           int
	Assists          int
	Accuracy         float64
	DamageDlt        float64
	DamageTkn        float64
	HeadshotKills    int
	IsRanked         bool
}

// enrichmentRow est le résultat intermédiaire de la phase 2 (player_match_enrichment).
type enrichmentRow struct {
	PerfScore    *float64
	SessionID    string
	IsWithFriends bool
	EngageScore  *float64
	ResidualBrut *float64
}

// skillRankRow est le résultat intermédiaire de la phase 2 (match_skill_rank).
type skillRankRow struct {
	RatingValue *float64
	RatingType  string
}

// loadPatternShared exécute la phase 1 (SharedReader).
func loadPatternShared(ctx context.Context, pdb *duckdb.PlayerDB, limit int) ([]sharedPatternRow, error) {
	db, release, err := pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("shared reader: %w", err)
	}
	defer release()

	const q = `
SELECT
    r.match_id,
    r.played_at,
    r.game_variant_category,
    r.map_id,
    p.outcome,
    r.duration_secs,
    p.kills, p.deaths, p.assists, p.accuracy,
    p.damage_dealt, p.damage_taken, p.headshot_kills,
    p.team_mmr IS NOT NULL AS is_ranked
FROM match_participants p
JOIN match_registry r USING (match_id)
WHERE p.xuid = ?
ORDER BY r.played_at DESC
LIMIT ?`

	sqlRows, err := db.QueryContext(ctx, q, pdb.XUID, limit)
	if err != nil {
		return nil, err
	}
	defer sqlRows.Close()

	var out []sharedPatternRow
	for sqlRows.Next() {
		var row sharedPatternRow
		var playedAt sql.NullTime
		var accuracy, damageDlt, damageTkn sql.NullFloat64
		var isRanked sql.NullBool
		err := sqlRows.Scan(
			&row.MatchID, &playedAt,
			&row.Mode, &row.MapID,
			&row.Outcome, &row.DurationSec,
			&row.Kills, &row.Deaths, &row.Assists,
			&accuracy, &damageDlt, &damageTkn, &row.HeadshotKills,
			&isRanked,
		)
		if err != nil {
			return nil, err
		}
		if playedAt.Valid {
			row.PlayedAt = playedAt.Time
		}
		if accuracy.Valid {
			row.Accuracy = accuracy.Float64
		}
		if damageDlt.Valid {
			row.DamageDlt = damageDlt.Float64
		}
		if damageTkn.Valid {
			row.DamageTkn = damageTkn.Float64
		}
		if isRanked.Valid {
			row.IsRanked = isRanked.Bool
		}
		out = append(out, row)
	}
	return out, sqlRows.Err()
}

// loadPatternEnrichments charge les enrichissements depuis la DB joueur.
func loadPatternEnrichments(ctx context.Context, pdb *duckdb.PlayerDB, matchIDs []string) (map[string]enrichmentRow, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}
	ph := duckdb.Placeholders(len(matchIDs))
	q := fmt.Sprintf(`
SELECT match_id, performance_score, session_id, is_with_friends,
       engagement_score, residual_brut
FROM player_match_enrichment
WHERE match_id IN (%s)`, ph)

	sqlRows, err := pdb.Player.Query(ctx, q, duckdb.ToAnySlice(matchIDs)...)
	if err != nil {
		return nil, err
	}
	defer sqlRows.Close()

	out := make(map[string]enrichmentRow, len(matchIDs))
	for sqlRows.Next() {
		var matchID string
		var row enrichmentRow
		var perfScore, engageScore, residualBrut sql.NullFloat64
		var sessionID sql.NullString
		var isWithFriends sql.NullBool
		if err := sqlRows.Scan(
			&matchID, &perfScore, &sessionID, &isWithFriends,
			&engageScore, &residualBrut,
		); err != nil {
			return nil, err
		}
		if perfScore.Valid {
			v := perfScore.Float64
			row.PerfScore = &v
		}
		if sessionID.Valid {
			row.SessionID = sessionID.String
		}
		if isWithFriends.Valid {
			row.IsWithFriends = isWithFriends.Bool
		}
		if engageScore.Valid {
			v := engageScore.Float64
			row.EngageScore = &v
		}
		if residualBrut.Valid {
			v := residualBrut.Float64
			row.ResidualBrut = &v
		}
		out[matchID] = row
	}
	return out, sqlRows.Err()
}

// loadPatternSkillRanks charge les ratings depuis la DB joueur.
func loadPatternSkillRanks(ctx context.Context, pdb *duckdb.PlayerDB, matchIDs []string) (map[string]skillRankRow, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}
	ph := duckdb.Placeholders(len(matchIDs))
	q := fmt.Sprintf(`
SELECT match_id, rating_value, rating_type
FROM match_skill_rank
WHERE match_id IN (%s)`, ph)

	sqlRows, err := pdb.Player.Query(ctx, q, duckdb.ToAnySlice(matchIDs)...)
	if err != nil {
		return nil, err
	}
	defer sqlRows.Close()

	out := make(map[string]skillRankRow, len(matchIDs))
	for sqlRows.Next() {
		var matchID string
		var row skillRankRow
		var ratingValue sql.NullFloat64
		if err := sqlRows.Scan(&matchID, &ratingValue, &row.RatingType); err != nil {
			return nil, err
		}
		if ratingValue.Valid {
			v := ratingValue.Float64
			row.RatingValue = &v
		}
		out[matchID] = row
	}
	return out, sqlRows.Err()
}

// mergePatternRows assemble les résultats des 3 phases en []patterns.MatchRow.
func mergePatternRows(shared []sharedPatternRow, enrichMap map[string]enrichmentRow, skillMap map[string]skillRankRow) []patterns.MatchRow {
	out := make([]patterns.MatchRow, 0, len(shared))
	for _, s := range shared {
		cy := analysis.ComputeCombatYield(s.Kills, s.Assists, s.DamageDlt, s.DamageTkn, s.Deaths)
		hsRate := 0.0
		if s.Kills > 0 {
			hsRate = float64(s.HeadshotKills) / float64(s.Kills)
		}
		row := patterns.MatchRow{
			MatchID:     s.MatchID,
			PlayedAt:    s.PlayedAt,
			Mode:        s.Mode,
			MapID:       s.MapID,
			Outcome:     s.Outcome,
			IsRanked:    s.IsRanked,
			DurationSec: s.DurationSec,
			Kills:       s.Kills,
			Deaths:      s.Deaths,
			Assists:     s.Assists,
			Accuracy:    s.Accuracy,
			OC:          cy.OffensiveConversion,
			DR:          cy.DefensiveResistance,
			HSRate:      hsRate,
		}
		// KDA : (kills + assists/2) / max(1, deaths)
		denom := s.Deaths
		if denom == 0 {
			denom = 1
		}
		row.KDA = (float64(s.Kills) + float64(s.Assists)/2.0) / float64(denom)

		if enr, ok := enrichMap[s.MatchID]; ok {
			row.PerfScore = enr.PerfScore
			row.SessionID = enr.SessionID
			row.IsWithFriends = enr.IsWithFriends
			row.EngageScore = enr.EngageScore
			row.ResidualBrut = enr.ResidualBrut
		}
		if sk, ok := skillMap[s.MatchID]; ok {
			if sk.RatingValue != nil {
				v := *sk.RatingValue
				switch sk.RatingType {
				case "LUSR":
					row.DeltaLUSR = &v
				case "CSR":
					row.CSRValue = &v
				}
			}
		}
		out = append(out, row)
	}
	return out
}

// computeSkillDeltas calcule les deltas LUSR/CSR entre matchs consécutifs.
// Les rows arrivent DESC (plus récent en premier) ; on trie ASC pour calculer
// les deltas puis on remet DESC.
func computeSkillDeltas(rows []patterns.MatchRow) {
	if len(rows) < 2 {
		return
	}
	// Trier ASC par PlayedAt
	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i].PlayedAt.Before(rows[j].PlayedAt)
	})
	// Calculer les deltas LUSR
	for i := 1; i < len(rows); i++ {
		if rows[i].DeltaLUSR != nil && rows[i-1].DeltaLUSR != nil {
			delta := *rows[i].DeltaLUSR - *rows[i-1].DeltaLUSR
			rows[i].DeltaLUSR = &delta
		}
	}
	if len(rows) > 0 {
		rows[0].DeltaLUSR = nil // pas de delta pour le premier match
	}
	// Calculer les deltas CSR
	for i := 1; i < len(rows); i++ {
		if rows[i].CSRValue != nil && rows[i-1].CSRValue != nil {
			delta := *rows[i].CSRValue - *rows[i-1].CSRValue
			rows[i].DeltaCSR = &delta
		}
	}
	if len(rows) > 0 {
		rows[0].DeltaCSR = nil
	}
	// Remettre DESC
	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i].PlayedAt.After(rows[j].PlayedAt)
	})
}
