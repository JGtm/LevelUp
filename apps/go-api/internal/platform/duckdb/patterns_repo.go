package duckdb

// patterns_repo.go — accès données du Pattern Engine v3.
//
// Déplacé depuis internal/api/handlers/patterns.go (refactor découplage Axe 1) :
// le handler ne fait plus d'accès DB ni de SQL ; il dépend de
// port.PatternsRepository, implémenté ici.
//
// Split cross-DB (ADR 0016) :
//   - Phase 1 (SharedReader) : match_participants + match_registry → stats brutes
//   - Phase 2 (Player) : player_match_enrichment + match_skill_rank → enrichissements
//   - Merge en Go + calcul des deltas LUSR/CSR

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/patterns"
	"levelup/go-api/internal/games"
)

// patternsLoadTimeout borne le chargement des données de patterns (3 queries
// cross-DB + merge). Valeur historique conservée du handler.
const patternsLoadTimeout = 30 * time.Second

// PatternsRepo charge les MatchRow analysées pour un joueur depuis DuckDB.
// Implémente port.PatternsRepository. Construit par joueur (porte un PlayerDB).
type PatternsRepo struct {
	pdb *PlayerDB
}

// NewPatternsRepo construit le repo pour un joueur donné.
func NewPatternsRepo(pdb *PlayerDB) *PatternsRepo {
	return &PatternsRepo{pdb: pdb}
}

// LoadRows charge les données nécessaires à l'analyse de patterns.
//
//	Phase 1 (SharedReader) : stats brutes
//	Phase 2 (Player)       : enrichissements + ratings
//	Merge en Go + calcul des deltas LUSR/CSR
func (r *PatternsRepo) LoadRows(ctx context.Context, limit int) ([]patterns.MatchRow, error) {
	ctx, cancel := context.WithTimeout(ctx, patternsLoadTimeout)
	defer cancel()

	shared, err := r.loadShared(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("loadPatternRows shared: %w", err)
	}
	if len(shared) == 0 {
		return nil, nil
	}

	matchIDs := make([]string, len(shared))
	for i, row := range shared {
		matchIDs[i] = row.MatchID
	}

	enrichMap, err := r.loadEnrichments(ctx, matchIDs)
	if err != nil {
		return nil, fmt.Errorf("loadPatternRows enrichments: %w", err)
	}

	skillMap, err := r.loadSkillRanks(ctx, matchIDs)
	if err != nil {
		return nil, fmt.Errorf("loadPatternRows skill_ranks: %w", err)
	}

	rows := mergePatternRows(shared, enrichMap, skillMap, games.EffectiveHpToKill(r.pdb.TitleSlug))
	computePatternSkillDeltas(rows)
	return rows, nil
}

// patternSharedRow est le résultat intermédiaire de la phase 1.
type patternSharedRow struct {
	MatchID       string
	PlayedAt      time.Time
	Mode          string
	MapID         string
	Outcome       int
	DurationSec   int
	Kills         int
	Deaths        int
	Assists       int
	Accuracy      float64
	DamageDlt     float64
	DamageTkn     float64
	HeadshotKills int
	IsRanked      bool
}

// patternEnrichmentRow est le résultat intermédiaire de la phase 2 (player_match_enrichment).
type patternEnrichmentRow struct {
	PerfScore     *float64
	SessionID     string
	IsWithFriends bool
	EngageScore   *float64
	ResidualBrut  *float64
}

// patternSkillRankRow est le résultat intermédiaire de la phase 2 (match_skill_rank).
type patternSkillRankRow struct {
	RatingValue *float64
	RatingType  string
}

// loadShared exécute la phase 1 (SharedReader).
func (r *PatternsRepo) loadShared(ctx context.Context, limit int) ([]patternSharedRow, error) {
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("shared reader: %w", err)
	}
	defer release()

	// played_at : pattern timezone canonique du projet — start_time_utc est
	// TIMESTAMPTZ UTC garanti, fallback sur start_time AT TIME ZONE 'UTC' pour
	// les matchs sans start_time_utc (cf. media_repo, ADR add_start_time_utc).
	// mode_raw : sous-mode (pair_name), normalisé en Go via NormalizeModeLabel.
	q := `
SELECT
    r.match_id,
    ` + StartTimeCanonicalSQL("r") + ` AS played_at,
    COALESCE(NULLIF(r.pair_name_fr, ''), r.pair_name, '') AS mode_raw,
    r.map_id,
    p.outcome,
    r.duration_seconds,
    p.kills, p.deaths, p.assists, p.accuracy,
    p.damage_dealt, p.damage_taken, p.headshot_kills,
    p.team_mmr IS NOT NULL AS is_ranked
FROM match_participants p
JOIN match_registry r USING (match_id)
WHERE p.xuid = ?
ORDER BY ` + StartTimeCanonicalSQL("r") + ` DESC
LIMIT ?`

	sqlRows, err := db.QueryContext(ctx, q, r.pdb.XUID, limit)
	if err != nil {
		return nil, err
	}
	defer sqlRows.Close()

	var out []patternSharedRow
	for sqlRows.Next() {
		var row patternSharedRow
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

// loadEnrichments charge les enrichissements depuis la DB joueur.
func (r *PatternsRepo) loadEnrichments(ctx context.Context, matchIDs []string) (map[string]patternEnrichmentRow, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}
	ph := Placeholders(len(matchIDs))
	q := fmt.Sprintf(`
SELECT match_id, performance_score, session_id, is_with_friends,
       engagement_score, engagement_score_brut AS residual_brut
FROM player_match_enrichment_latest
WHERE match_id IN (%s)`, ph)

	sqlRows, err := r.pdb.Player.Query(ctx, q, ToAnySlice(matchIDs)...)
	if err != nil {
		return nil, err
	}
	defer sqlRows.Close()

	out := make(map[string]patternEnrichmentRow, len(matchIDs))
	for sqlRows.Next() {
		var matchID string
		var row patternEnrichmentRow
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

// loadSkillRanks charge les ratings depuis la DB joueur.
func (r *PatternsRepo) loadSkillRanks(ctx context.Context, matchIDs []string) (map[string]patternSkillRankRow, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}
	ph := Placeholders(len(matchIDs))
	q := fmt.Sprintf(`
SELECT match_id, rating_value, rating_type
FROM match_skill_rank_latest
WHERE match_id IN (%s)`, ph)

	sqlRows, err := r.pdb.Player.Query(ctx, q, ToAnySlice(matchIDs)...)
	if err != nil {
		return nil, err
	}
	defer sqlRows.Close()

	out := make(map[string]patternSkillRankRow, len(matchIDs))
	for sqlRows.Next() {
		var matchID string
		var row patternSkillRankRow
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
func mergePatternRows(shared []patternSharedRow, enrichMap map[string]patternEnrichmentRow, skillMap map[string]patternSkillRankRow, effectiveHpToKill float64) []patterns.MatchRow {
	out := make([]patterns.MatchRow, 0, len(shared))
	for _, s := range shared {
		cy := analysis.ComputeCombatYield(s.Kills, s.Assists, s.DamageDlt, s.DamageTkn, s.Deaths, effectiveHpToKill)
		hsRate := 0.0
		if s.Kills > 0 {
			hsRate = float64(s.HeadshotKills) / float64(s.Kills)
		}
		row := patterns.MatchRow{
			MatchID:     s.MatchID,
			PlayedAt:    s.PlayedAt,
			Mode:        analysis.NormalizeModeLabel(s.Mode),
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

// computePatternSkillDeltas calcule les deltas LUSR/CSR entre matchs consécutifs.
// Les rows arrivent DESC (plus récent en premier) ; on trie ASC pour calculer
// les deltas puis on remet DESC.
func computePatternSkillDeltas(rows []patterns.MatchRow) {
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
