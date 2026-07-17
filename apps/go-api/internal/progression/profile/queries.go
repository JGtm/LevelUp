package profile

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/prestige"
)

// queries.go — helpers SQL pour BuildProfile() (Sections A1/A2/B/C V1).
//
// Toutes les queries acceptent une fenêtre [since, until] cohérente avec la
// fenêtre LOWESS du caller. Depuis ADR 0016, les accès cross-DB
// (match_registry, match_participants, highlight_events) passent par
// SharedReader (s.pdb.SharedReadDB) sans préfixe `shared.`. Les tables player
// (`lusr_component_history`, `personal_score_awards`) restent lues sur s.db
// (= pdb.Player).

const queryTimeout = 10 * time.Second

// startTimeCanonicalMR : expression timezone-canonique du début de match pour
// l'alias `mr` (E5) — jamais de `start_time` brut dans un filtre/tri temporel
// (piège start_time_utc NULL / start_time naïf). Source unique :
// duckdb.StartTimeCanonicalSQL (cf. reference_timezone_canonical_pattern).
var startTimeCanonicalMR = duckdb.StartTimeCanonicalSQL("mr")

// campaignExcl retourne la clause littérale de masquage des matchs Campagne pour
// l'alias registre donné (title-aware via le titre du joueur ; no-op Infinite).
// Source unique analysis.SQLExcludeCampaignVariants (item H1). La page Ascension
// (profil radar / engagement / cadence) agrège les matchs de la fenêtre → sans ce
// filtre les matchs Campagne (Halo 5) biaiseraient les axes et compteurs.
func (s *Service) campaignExcl(alias string) string {
	if s.pdb == nil {
		return ""
	}
	return analysis.SQLExcludeCampaignVariants(s.pdb.TitleSlug, alias)
}

// countMatchesInWindow retourne le nombre de matchs distincts du joueur dans
// la fenêtre. Source : match_participants côté shared (filtré par xuid),
// lu via SharedReader (ADR 0016).
func (s *Service) countMatchesInWindow(ctx context.Context, userID string, since, until time.Time) (int, error) {
	if s.pdb == nil {
		return 0, fmt.Errorf("countMatchesInWindow: PlayerDB non initialisée")
	}
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	sharedDB, release, err := s.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return 0, fmt.Errorf("countMatchesInWindow: shared reader: %w", err)
	}
	defer release()
	var count int
	err = sharedDB.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT mp.match_id)
		FROM match_participants mp
		JOIN match_registry mr ON mr.match_id = mp.match_id
		WHERE mp.xuid = ?`+s.campaignExcl("mr")+`
		  AND `+startTimeCanonicalMR+` >= ? AND `+startTimeCanonicalMR+` <= ?
	`, userID, since, until).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("countMatchesInWindow: %w", err)
	}
	return count, nil
}

// computeRadarAxes calcule les 6 axes radar narrative en agrégeant les stats
// du joueur sur la fenêtre.
//
// Deux sources de signal :
//   - match_participants : agrégats Halo (kills/deaths/assists/score/spree)
//     → fournit les axes combat / survival / support / score / impact.
//   - personal_score_awards : awards spécifiques (flag_captured, zone_secured,
//     destroyed_*, etc.) → fournit l'axe objective + enrichit support/impact
//     via le mapping awards.toml (V2 §2).
//
// Sans AwardMappingSet injecté, l'axe Objective reste à 0 (fallback V1).
// Avec mapping, l'axe Objective devient un signal réel et les axes
// support/impact gagnent en granularité.
func (s *Service) computeRadarAxes(ctx context.Context, userID string, since, until time.Time) (map[narrative.ParticipationAxis]float64, error) {
	out, matchCount, err := s.computeRadarAxesBase(ctx, userID, since, until)
	if err != nil {
		return nil, err
	}
	if matchCount == 0 {
		return nil, nil
	}
	// Enrichissement awards : objective + boosts ciblés sur support/impact.
	if s.awards != nil {
		s.applyAwardsRadarAxes(ctx, userID, since, until, matchCount, out)
	}
	return out, nil
}

// computeRadarAxesBase calcule les axes via match_participants (heuristique
// V1 conservée). Retourne aussi le matchCount pour normaliser les awards.
func (s *Service) computeRadarAxesBase(
	ctx context.Context, userID string, since, until time.Time,
) (map[narrative.ParticipationAxis]float64, int, error) {
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
	sharedDB, release, err := s.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("computeRadarAxesBase: shared reader: %w", err)
	}
	defer release()
	err = sharedDB.QueryRowContext(ctx, `
		SELECT
			AVG(mp.kills),
			AVG(mp.deaths),
			AVG(mp.assists),
			AVG(mp.personal_score),
			AVG(mp.max_killing_spree),
			COUNT(*)
		FROM match_participants mp
		JOIN match_registry mr ON mr.match_id = mp.match_id
		WHERE mp.xuid = ?`+s.campaignExcl("mr")+`
		  AND `+startTimeCanonicalMR+` >= ? AND `+startTimeCanonicalMR+` <= ?
	`, userID, since, until).Scan(&avgKills, &avgDeaths, &avgAssists, &avgScore, &avgKillingSpree, &matchCount)
	if err != nil {
		return nil, 0, fmt.Errorf("computeRadarAxesBase: %w", err)
	}
	out := map[narrative.ParticipationAxis]float64{}
	if matchCount == 0 {
		return out, 0, nil
	}
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
		out[narrative.AxisScore] = avgScore.Float64
	}
	if avgKillingSpree.Valid {
		out[narrative.AxisImpact] = avgKillingSpree.Float64 * 10
	}
	return out, matchCount, nil
}

// applyAwardsRadarAxes lit personal_score_awards dans la fenêtre, somme par
// award_name, applique le mapping (axes + weight) et accumule par axe en
// moyenne par match. Mutates out in place.
//
// Split cross-DB (ADR 0016) : Phase A charge les match_ids dans la fenêtre
// via SharedReader, Phase B agrège personal_score_awards sur Player avec
// WHERE match_id IN (...).
func (s *Service) applyAwardsRadarAxes(
	ctx context.Context, userID string, since, until time.Time,
	matchCount int, out map[narrative.ParticipationAxis]float64,
) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	// Phase A : match_ids dans la fenêtre via SharedReader.
	sharedDB, release, err := s.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		// Dégrade gracieusement comme la version legacy (table absente / shared
		// indispo → on garde la base V1 sans erreur).
		return
	}
	defer release()
	matchRows, err := sharedDB.QueryContext(ctx, `
		SELECT match_id FROM match_registry mr
		WHERE `+startTimeCanonicalMR+` >= ? AND `+startTimeCanonicalMR+` <= ?`+s.campaignExcl("mr")+`
	`, since, until)
	if err != nil {
		return
	}
	var matchIDs []string
	for matchRows.Next() {
		var id string
		if err := matchRows.Scan(&id); err == nil && id != "" {
			matchIDs = append(matchIDs, id)
		}
	}
	matchRows.Close()
	if len(matchIDs) == 0 {
		return
	}

	// Phase B : agrégat sur personal_score_awards (player) avec IN.
	placeholders := make([]string, len(matchIDs))
	args := make([]any, 0, len(matchIDs)+1)
	args = append(args, userID)
	for i, id := range matchIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	rows, err := s.db.Query(ctx, `
		SELECT psa.award_name, SUM(psa.award_count)
		FROM personal_score_awards_latest psa
		WHERE psa.xuid = ?
		  AND psa.match_id IN (`+strings.Join(placeholders, ",")+`)
		GROUP BY psa.award_name
	`, args...)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var total sql.NullInt64
		if err := rows.Scan(&name, &total); err != nil {
			return
		}
		if !total.Valid || total.Int64 <= 0 {
			continue
		}
		mapping, ok := s.awards.Get(name)
		if !ok {
			continue
		}
		contribution := float64(total.Int64) * mapping.Weight / float64(matchCount)
		for _, axis := range mapping.Axes {
			out[narrative.ParticipationAxis(axis)] += contribution
		}
	}
}

// computeFKFD compte les First Kill / First Death du joueur sur la fenêtre.
// Source : shared.highlight_events (event_type IN ('first_kill','first_death')).
// Si la table n'est pas peuplée pour ces matchs, retourne (0, 0, nil).
func (s *Service) computeFKFD(ctx context.Context, userID string, since, until time.Time) (int, int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	sharedDB, release, err := s.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return 0, 0, nil
	}
	defer release()
	var fk, fd int
	err = sharedDB.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN he.event_type = 'first_kill'  AND he.killer_xuid = ? THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN he.event_type = 'first_death' AND he.victim_xuid = ? THEN 1 ELSE 0 END), 0)
		FROM highlight_events he
		JOIN match_registry mr ON mr.match_id = he.match_id
		WHERE `+startTimeCanonicalMR+` >= ? AND `+startTimeCanonicalMR+` <= ?`+s.campaignExcl("mr")+`
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

	sharedDB, release, err := s.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return snap, fmt.Errorf("computeEngagementSimple: shared reader: %w", err)
	}
	defer release()
	rows, err := sharedDB.QueryContext(ctx, `
		SELECT `+startTimeCanonicalMR+`
		FROM match_participants mp
		JOIN match_registry mr ON mr.match_id = mp.match_id
		WHERE mp.xuid = ?`+s.campaignExcl("mr")+`
		  AND `+startTimeCanonicalMR+` >= ? AND `+startTimeCanonicalMR+` <= ?
		ORDER BY `+startTimeCanonicalMR+` ASC
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
//   - trend : amélioration récente (différence entre la moyenne des 10 derniers
//     matchs et la moyenne des 10 plus anciens dans la fenêtre — ≥ 20 matchs
//     requis pour significativité).
//
// Source : table `lusr_component_history` (V2 §1) alimentée live par
// sync.upsertLUSRRatings et au backfill via re-run de ComputeSkillRatingsBatch
// en mode force.
//
// Une seule query CTE batchée groupe AVG / QUANTILE / trend pour les 8
// composantes (V2 §1 polish — évite les 8 round-trips initiaux). Si la table
// est vide ou la query échoue, retourne map vide silencieusement (UI dégrade
// gracieusement avec current=0/top20=0/target).
func (s *Service) loadLUSRComponentsBreakdown(ctx context.Context, _userID string) (map[string]componentRow, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	out := make(map[string]componentRow, 8)
	rows, err := s.db.Query(ctx, `
		WITH ranked AS (
			SELECT
				component_name,
				value,
				ROW_NUMBER() OVER (PARTITION BY component_name ORDER BY computed_at DESC) AS rk_desc,
				ROW_NUMBER() OVER (PARTITION BY component_name ORDER BY computed_at ASC)  AS rk_asc,
				COUNT(*)    OVER (PARTITION BY component_name)                            AS n
			FROM lusr_component_history_latest
		)
		SELECT
			component_name,
			AVG(value)                                         AS current_avg,
			QUANTILE_CONT(value, 0.8)                          AS top20,
			AVG(CASE WHEN n >= 20 AND rk_desc <= 10 THEN value END) AS last_avg,
			AVG(CASE WHEN n >= 20 AND rk_asc  <= 10 THEN value END) AS first_avg,
			MAX(n)                                             AS n
		FROM ranked
		GROUP BY component_name
	`)
	if err != nil {
		return out, nil
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var avg, top20, lastAvg, firstAvg sql.NullFloat64
		var n int
		if err := rows.Scan(&name, &avg, &top20, &lastAvg, &firstAvg, &n); err != nil {
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
		if lastAvg.Valid && firstAvg.Valid {
			row.trend = lastAvg.Float64 - firstAvg.Float64
		}
		out[name] = row
	}
	return out, rows.Err()
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
