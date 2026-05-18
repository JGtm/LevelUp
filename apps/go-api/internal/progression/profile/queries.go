package profile

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/prestige"
)

// queries.go — helpers SQL pour BuildProfile() (Sections A1/A2/B/C V1).
//
// Toutes les queries acceptent une fenêtre [since, until] cohérente avec la
// fenêtre LOWESS du caller. Les données proviennent de stats.duckdb du joueur,
// qui doit avoir shared_matches_v2 attachée (alias `shared`) pour accéder à
// match_participants.

const queryTimeout = 10 * time.Second

// countMatchesInWindow retourne le nombre de matchs distincts du joueur dans
// la fenêtre. Source : match_participants côté shared (filtré par xuid).
func (s *Service) countMatchesInWindow(ctx context.Context, userID string, since, until time.Time) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	var count int
	err := s.db.QueryRow(ctx, `
		SELECT COUNT(DISTINCT mp.match_id)
		FROM shared.match_participants mp
		JOIN shared.match_registry mr ON mr.match_id = mp.match_id
		WHERE mp.xuid = ?
		  AND mr.start_time >= ? AND mr.start_time <= ?
	`, userID, since, until).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("countMatchesInWindow: %w", err)
	}
	return count, nil
}

// computeRadarAxes calcule les 6 axes radar narrative en agrégeant les stats
// du joueur sur la fenêtre. Mapping heuristique V1 (sans personal_score_awards) :
//
//   - combat   = kills moyens par match
//   - survival = max(0, kills - deaths) moyen par match
//   - support  = assists moyens
//   - score    = personal_score moyen / 100 (échelle ~point)
//   - objective= 0 en V1 (nécessite mapping award→axis title-specific)
//   - impact   = max_killing_spree moyen
func (s *Service) computeRadarAxes(ctx context.Context, userID string, since, until time.Time) (map[narrative.ParticipationAxis]float64, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	var (
		avgKills        sql.NullFloat64
		avgDeaths       sql.NullFloat64
		avgAssists      sql.NullFloat64
		avgScore        sql.NullFloat64
		avgKillingSpree sql.NullFloat64
		matchCount      int
	)
	err := s.db.QueryRow(ctx, `
		SELECT
			AVG(mp.kills),
			AVG(mp.deaths),
			AVG(mp.assists),
			AVG(mp.personal_score),
			AVG(mp.max_killing_spree),
			COUNT(*)
		FROM shared.match_participants mp
		JOIN shared.match_registry mr ON mr.match_id = mp.match_id
		WHERE mp.xuid = ?
		  AND mr.start_time >= ? AND mr.start_time <= ?
	`, userID, since, until).Scan(&avgKills, &avgDeaths, &avgAssists, &avgScore, &avgKillingSpree, &matchCount)
	if err != nil {
		return nil, fmt.Errorf("computeRadarAxes: %w", err)
	}
	if matchCount == 0 {
		return nil, nil
	}
	out := map[narrative.ParticipationAxis]float64{}
	if avgKills.Valid {
		out[narrative.AxisCombat] = avgKills.Float64
	}
	if avgKills.Valid && avgDeaths.Valid {
		survival := avgKills.Float64 - avgDeaths.Float64
		if survival < 0 {
			survival = 0
		}
		out[narrative.AxisSurvival] = survival
	}
	if avgAssists.Valid {
		out[narrative.AxisSupport] = avgAssists.Float64
	}
	if avgScore.Valid {
		// Score brut est ~quelques centaines : on l'échelonne à l'unité du
		// seuil "score" custom (150).
		out[narrative.AxisScore] = avgScore.Float64
	}
	if avgKillingSpree.Valid {
		out[narrative.AxisImpact] = avgKillingSpree.Float64 * 10 // amplification pour rester comparable
	}
	// objective laissé absent en V1 : nécessite mapping award→axis.
	return out, nil
}

// computeFKFD compte les First Kill / First Death du joueur sur la fenêtre.
// Source : shared.highlight_events (event_type IN ('first_kill','first_death')).
// Si la table n'est pas peuplée pour ces matchs, retourne (0, 0, nil).
func (s *Service) computeFKFD(ctx context.Context, userID string, since, until time.Time) (int, int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	var fk, fd int
	err := s.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN he.event_type = 'first_kill'  AND he.killer_xuid = ? THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN he.event_type = 'first_death' AND he.victim_xuid = ? THEN 1 ELSE 0 END), 0)
		FROM shared.highlight_events he
		JOIN shared.match_registry mr ON mr.match_id = he.match_id
		WHERE mr.start_time >= ? AND mr.start_time <= ?
	`, userID, userID, since, until).Scan(&fk, &fd)
	if err != nil {
		// La table highlight_events peut ne pas exister ou être vide selon
		// l'état de sync : on dégrade gracieusement.
		return 0, 0, nil
	}
	return fk, fd, nil
}

// computeEngagementSimple calcule un EngagementSnapshot heuristique :
//
//   - score = min(100, matches_per_day_avg × 25)
//   - tier  = low / regular / high / intense par paliers
//   - max_gap_days = plus longue interruption entre deux matchs
func (s *Service) computeEngagementSimple(ctx context.Context, userID string, since, until time.Time) (EngagementSnapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	var snap EngagementSnapshot

	rows, err := s.db.Query(ctx, `
		SELECT mr.start_time
		FROM shared.match_participants mp
		JOIN shared.match_registry mr ON mr.match_id = mp.match_id
		WHERE mp.xuid = ?
		  AND mr.start_time >= ? AND mr.start_time <= ?
		ORDER BY mr.start_time ASC
	`, userID, since, until)
	if err != nil {
		return snap, fmt.Errorf("computeEngagementSimple: %w", err)
	}
	defer rows.Close()
	var times []time.Time
	for rows.Next() {
		var t sql.NullTime
		if err := rows.Scan(&t); err != nil {
			return snap, err
		}
		if t.Valid {
			times = append(times, t.Time)
		}
	}
	if rerr := rows.Err(); rerr != nil {
		return snap, rerr
	}
	if len(times) == 0 {
		return snap, nil
	}
	windowDays := until.Sub(since).Hours() / 24
	if windowDays <= 0 {
		windowDays = 1
	}
	snap.MatchesPerDayAvg = float64(len(times)) / windowDays
	snap.Score = snap.MatchesPerDayAvg * 25
	if snap.Score > 100 {
		snap.Score = 100
	}
	// max gap entre matchs consécutifs.
	var maxGap time.Duration
	for i := 1; i < len(times); i++ {
		gap := times[i].Sub(times[i-1])
		if gap > maxGap {
			maxGap = gap
		}
	}
	snap.MaxGapDays = int(maxGap.Hours() / 24)
	snap.Tier = engagementTier(snap.Score)
	snap.RegularityCoach = "profile.engagement." + snap.Tier
	return snap, nil
}

func engagementTier(score float64) string {
	switch {
	case score < 20:
		return "low"
	case score < 50:
		return "regular"
	case score < 80:
		return "high"
	default:
		return "intense"
	}
}

// componentRow capture l'agrégat d'une composante LUSR.
type componentRow struct {
	currentAvg float64 // moyenne sur la fenêtre (0..1)
	top20      float64 // top 20% personnel sur la fenêtre
	trend      float64 // pente LOWESS simplifiée (diff dernière - première)
}

// loadLUSRComponentsBreakdown calcule, pour chaque composante LUSR :
//
//   - current_avg : moyenne brute sur les matchs du joueur (toutes les valeurs
//     sont déjà normalisées 0..1 par le scoring engine).
//   - top20 : seuil du quintile haut (top 20% personnel = QUANTILE_CONT(0.8)).
//   - trend : amélioration récente (différence entre les 10 derniers et les 10
//     plus anciens du dataset post-snapshot).
//
// Source : table `lusr_component_history` (V2 §1) alimentée live par
// sync.upsertLUSRRatings et au backfill via re-run de ComputeSkillRatingsBatch
// en mode force.
//
// Si la table est vide (joueur non-backfillé ou trop peu de matchs ratés),
// retourne map vide → UI dégrade gracieusement (current=0/top20=0/target).
func (s *Service) loadLUSRComponentsBreakdown(ctx context.Context, _userID string) (map[string]componentRow, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	out := make(map[string]componentRow, 8)
	rows, err := s.db.Query(ctx, `
		SELECT
			component_name,
			AVG(value)                            AS current_avg,
			QUANTILE_CONT(value, 0.8)             AS top20,
			COUNT(*)                              AS n
		FROM lusr_component_history
		GROUP BY component_name
	`)
	if err != nil {
		// Table absente (migration non encore appliquée par exemple) → vide.
		return out, nil
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var avg, top20 sql.NullFloat64
		var n int
		if err := rows.Scan(&name, &avg, &top20, &n); err != nil {
			return out, err
		}
		if n == 0 {
			continue
		}
		row := componentRow{}
		if avg.Valid {
			row.currentAvg = avg.Float64
		}
		if top20.Valid {
			row.top20 = top20.Float64
		}
		// Trend : différence entre la moyenne des 10 derniers matchs et la
		// moyenne des 10 plus anciens (par chronologie computed_at). Positif
		// = amélioration. Sur < 20 matchs le trend reste 0 (significativité).
		row.trend = s.computeComponentTrend(ctx, name)
		out[name] = row
	}
	return out, rows.Err()
}

// computeComponentTrend retourne la pente simple last10 - first10 sur la
// chronologie d'une composante. 0 si < 20 matchs (significativité minimale).
func (s *Service) computeComponentTrend(ctx context.Context, componentName string) float64 {
	var lastAvg, firstAvg sql.NullFloat64
	err := s.db.QueryRow(ctx, `
		WITH ordered AS (
			SELECT value, ROW_NUMBER() OVER (ORDER BY computed_at DESC) AS rk_desc,
			              ROW_NUMBER() OVER (ORDER BY computed_at ASC)  AS rk_asc,
			              COUNT(*) OVER ()                                AS n
			FROM lusr_component_history
			WHERE component_name = ?
		)
		SELECT
			AVG(CASE WHEN rk_desc <= 10 THEN value END) AS last_avg,
			AVG(CASE WHEN rk_asc  <= 10 THEN value END) AS first_avg
		FROM ordered
		WHERE n >= 20
	`, componentName).Scan(&lastAvg, &firstAvg)
	if err != nil || !lastAvg.Valid || !firstAvg.Valid {
		return 0
	}
	return lastAvg.Float64 - firstAvg.Float64
}

// listTemplatesByLUSRComponents retourne les templates dont les
// lusr_components recoupent l'ensemble fourni. Lecture directe depuis
// metadata.duckdb (CSV-encoded).
func (r *duckdbRepo) listTemplatesByLUSRComponents(ctx context.Context, titleSlug string, components map[string]struct{}) ([]prestige.Template, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}
	if len(components) == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.db.Query(ctx, `
		SELECT id, title_slug, metric, window_type, COALESCE(window_value, ''),
		       cadence, eval_type, COALESCE(mode_filter, 'universal'),
		       label_en, label_fr, COALESCE(description_en, ''), COALESCE(description_fr, ''),
		       normal_target, heroic_target, legendary_target, mythic_target,
		       COALESCE(lusr_components, ''), COALESCE(radar_axes, ''),
		       COALESCE(is_long_term, FALSE),
		       schema_version, updated_at
		FROM challenge_template
		WHERE title_slug = ?
	`, titleSlug)
	if err != nil {
		return nil, fmt.Errorf("listTemplatesByLUSRComponents: %w", err)
	}
	defer rows.Close()
	var out []prestige.Template
	for rows.Next() {
		t, err := scanTemplateRow(rows)
		if err != nil {
			return nil, err
		}
		// Filtre côté Go : on garde si au moins une composante du template
		// est dans l'ensemble des leviers.
		matched := false
		for _, c := range t.LUSRComponents {
			if _, ok := components[c]; ok {
				matched = true
				break
			}
		}
		if matched {
			out = append(out, t)
		}
	}
	return out, rows.Err()
}

// scanTemplateRow scanne une ligne challenge_template depuis un *sql.Rows.
// Volontairement dupliqué de platform/duckdb.scanTemplate pour éviter d'élever
// ce dernier en exporté (et la dépendance circulaire qui en résulterait).
func scanTemplateRow(row *sql.Rows) (prestige.Template, error) {
	var t prestige.Template
	var windowType, cadence, evalType string
	var lusrJSON, radarJSON string
	err := row.Scan(
		&t.ID, &t.TitleSlug, &t.Metric, &windowType, &t.WindowValue,
		&cadence, &evalType, &t.ModeFilter,
		&t.LabelEN, &t.LabelFR, &t.DescriptionEN, &t.DescriptionFR,
		&t.NormalTarget, &t.HeroicTarget, &t.LegendaryTarget, &t.MythicTarget,
		&lusrJSON, &radarJSON, &t.IsLongTerm,
		&t.SchemaVersion, &t.UpdatedAt,
	)
	if err != nil {
		return t, err
	}
	t.WindowType = prestige.WindowType(windowType)
	t.Cadence = prestige.Cadence(cadence)
	t.EvalType = prestige.EvalType(evalType)
	t.LUSRComponents = decodeStringListLocal(lusrJSON)
	t.RadarAxes = decodeStringListLocal(radarJSON)
	return t, nil
}

func decodeStringListLocal(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}
