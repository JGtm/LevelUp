// Package ops — milestone_dates.go : backfill one-off des dates de franchissement
// des jalons (A6). Les earned_at ont toutes été estampillées à la date du premier
// run (input.Now) par le detector → date fausse. On recalcule la VRAIE date de
// franchissement par jalon en rejouant les matchs du joueur chronologiquement
// (fragment timezone canonique COALESCE(start_time_utc, start_time AT TIME ZONE
// 'UTC')), cumul par métrique jusqu'au seuil. Si non dérivable (métrique inconnue
// ou seuil jamais atteint dans l'historique disponible) → earned_at NULL (le
// front n'affiche alors pas de date, jamais une date fausse).
//
// milestone_earned n'est PAS append-only : PK (user_id, title_slug, milestone_id),
// earned_at NON indexée → un UPDATE de cette seule colonne est ART-safe (aucune
// churn d'index ART). Pas de rebuild ici (contraste avec la purge records A5).
package ops

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"
)

// Constantes de qualification par match, ALIGNÉES sur le detector milestones
// (internal/api/wire/post_sync_progression_queries.go — loadPlayerStats). À garder
// synchronisées si les seuils P80 / régularité y changent.
const (
	combatOCP80           = 0.83 // OffensiveConversion P80 (combat_precision/excellence)
	combatDRP80           = 1.59 // DefensiveResistance P80 (combat_endurance/excellence)
	accuracyThresholdDays = 0.50 // accuracy minimale d'un « jour régulier »
)

// MilestoneCrossingMatch est un match du joueur, dans l'ordre chronologique, avec
// les champs nécessaires pour rejouer toutes les métriques de jalon.
type MilestoneCrossingMatch struct {
	PlayedAt    time.Time // start canonique UTC
	Win         bool
	Kills       int
	Headshots   int
	Assists     int
	Deaths      int
	Accuracy    float64
	DamageDealt float64
	DamageTaken float64
}

// MilestoneTarget est un jalon débloqué dont on recalcule la date : sa métrique
// et son seuil (issus du catalogue).
type MilestoneTarget struct {
	MilestoneID string
	Metric      string
	Threshold   float64
}

// preppedMatch précalcule les qualifications par match (OC/DR/accuracy) pour
// éviter de les recomputer par cible.
type preppedMatch struct {
	playedAt          time.Time
	win               bool
	kills, headshots  int
	assists           int
	accuracyQualifies bool
	ocQualifies       bool
	drQualifies       bool
}

// ComputeMilestoneCrossings retourne, par milestone_id, la date de franchissement
// recalculée (nil si non dérivable). Fonction PURE (testable sans DB). `hpToKill`
// est le baseline PV-pour-tuer du titre (games.EffectiveHpToKill).
func ComputeMilestoneCrossings(matches []MilestoneCrossingMatch, targets []MilestoneTarget, hpToKill float64) map[string]*time.Time {
	sorted := make([]MilestoneCrossingMatch, len(matches))
	copy(sorted, matches)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].PlayedAt.Before(sorted[j].PlayedAt) })

	prepped := make([]preppedMatch, len(sorted))
	for i, m := range sorted {
		prepped[i] = prepMatch(m, hpToKill)
	}

	out := make(map[string]*time.Time, len(targets))
	for _, t := range targets {
		out[t.MilestoneID] = crossingFor(t, prepped)
	}
	return out
}

func prepMatch(m MilestoneCrossingMatch, hp float64) preppedMatch {
	p := preppedMatch{
		playedAt:          m.PlayedAt,
		win:               m.Win,
		kills:             m.Kills,
		headshots:         m.Headshots,
		assists:           m.Assists,
		accuracyQualifies: m.Accuracy >= accuracyThresholdDays,
	}
	if m.DamageDealt > 0 {
		oc := hp * (float64(m.Kills) + float64(m.Assists)/3.0) / m.DamageDealt
		p.ocQualifies = oc >= combatOCP80
	}
	if m.DamageTaken > 0 && m.Deaths > 0 {
		dr := m.DamageTaken / (hp * float64(m.Deaths))
		p.drQualifies = dr >= combatDRP80
	}
	return p
}

// crossingFor calcule la date de franchissement d'une cible selon sa métrique.
func crossingFor(target MilestoneTarget, matches []preppedMatch) *time.Time {
	switch target.Metric {
	case "matches_played":
		return nthMatch(matches, target.Threshold, func(preppedMatch) bool { return true })
	case "wins":
		return nthMatch(matches, target.Threshold, func(m preppedMatch) bool { return m.win })
	case "combat_precision_matches":
		return nthMatch(matches, target.Threshold, func(m preppedMatch) bool { return m.ocQualifies })
	case "combat_endurance_matches":
		return nthMatch(matches, target.Threshold, func(m preppedMatch) bool { return m.drQualifies })
	case "combat_excellence_matches":
		return nthMatch(matches, target.Threshold, func(m preppedMatch) bool { return m.ocQualifies && m.drQualifies })
	case "kills":
		return cumSum(matches, target.Threshold, func(m preppedMatch) float64 { return float64(m.kills) })
	case "headshots":
		return cumSum(matches, target.Threshold, func(m preppedMatch) float64 { return float64(m.headshots) })
	case "assists":
		return cumSum(matches, target.Threshold, func(m preppedMatch) float64 { return float64(m.assists) })
	case "accuracy_threshold_days":
		return nthDistinctDay(matches, target.Threshold)
	default:
		return nil // métrique inconnue → non dérivable
	}
}

// nthMatch retourne la date du (threshold)-ème match satisfaisant pred.
func nthMatch(matches []preppedMatch, threshold float64, pred func(preppedMatch) bool) *time.Time {
	count := 0
	for _, m := range matches {
		if !pred(m) {
			continue
		}
		count++
		if float64(count) >= threshold {
			t := m.playedAt
			return &t
		}
	}
	return nil
}

// cumSum retourne la date du premier match où la somme cumulée atteint threshold.
func cumSum(matches []preppedMatch, threshold float64, val func(preppedMatch) float64) *time.Time {
	sum := 0.0
	for _, m := range matches {
		sum += val(m)
		if sum >= threshold {
			t := m.playedAt
			return &t
		}
	}
	return nil
}

// nthDistinctDay retourne la date du match introduisant le (threshold)-ème jour
// distinct qualifiant (accuracy >= seuil).
func nthDistinctDay(matches []preppedMatch, threshold float64) *time.Time {
	seen := make(map[string]struct{})
	for _, m := range matches {
		if !m.accuracyQualifies {
			continue
		}
		day := m.playedAt.UTC().Format("2006-01-02")
		if _, ok := seen[day]; ok {
			continue
		}
		seen[day] = struct{}{}
		if float64(len(seen)) >= threshold {
			t := m.playedAt
			return &t
		}
	}
	return nil
}

// ─── Orchestration DB ────────────────────────────────────────────────────────

const winOutcome = 2 // victoire = outcome 2 (canonical OutcomeWin, cf. db-schema)

// MilestoneBackfillResult résume le backfill pour un joueur (xuid).
type MilestoneBackfillResult struct {
	XUID    string
	Total   int
	Updated int // jalons avec date dérivée
	Nulled  int // jalons non dérivables (earned_at -> NULL)
}

// BackfillMilestoneDates recalcule les earned_at des jalons débloqués d'un titre.
// `sharedDB` = shared_matches_v2 (RO), `statsDB` = stats.duckdb du joueur (RW),
// `catalog` = milestone_id -> cible (métrique + seuil). apply=false → dry-run
// (aucune écriture). Un résultat par xuid présent dans milestone_earned.
func BackfillMilestoneDates(ctx context.Context, sharedDB, statsDB *sql.DB,
	titleSlug string, catalog map[string]MilestoneTarget, hpToKill float64, apply bool) ([]MilestoneBackfillResult, error) {
	exists, err := recordsTableExists(ctx, statsDB, "milestone_earned")
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	byXUID, err := loadEarnedByXUID(ctx, statsDB, titleSlug)
	if err != nil {
		return nil, err
	}
	if len(byXUID) == 0 {
		return nil, nil
	}
	if apply {
		if err := ensureEarnedAtNullable(ctx, statsDB); err != nil {
			return nil, err
		}
	}

	var results []MilestoneBackfillResult
	for xuid, milestoneIDs := range byXUID {
		matches, err := loadCrossingMatches(ctx, sharedDB, xuid)
		if err != nil {
			return nil, err
		}
		targets := make([]MilestoneTarget, 0, len(milestoneIDs))
		for _, id := range milestoneIDs {
			t := catalog[id]   // {} si absent du catalogue
			t.MilestoneID = id // garantit l'id même hors catalogue (métrique vide → non dérivable)
			targets = append(targets, t)
		}
		crossings := ComputeMilestoneCrossings(matches, targets, hpToKill)

		res := MilestoneBackfillResult{XUID: xuid, Total: len(milestoneIDs)}
		for _, id := range milestoneIDs {
			at := crossings[id]
			if at != nil {
				res.Updated++
			} else {
				res.Nulled++
			}
			if apply {
				if err := updateEarnedAt(ctx, statsDB, xuid, titleSlug, id, at); err != nil {
					return nil, err
				}
			}
		}
		results = append(results, res)
	}
	return results, nil
}

// loadEarnedByXUID retourne les milestone_id débloqués groupés par user_id (xuid).
func loadEarnedByXUID(ctx context.Context, statsDB *sql.DB, titleSlug string) (map[string][]string, error) {
	rows, err := statsDB.QueryContext(ctx,
		`SELECT user_id, milestone_id FROM milestone_earned WHERE title_slug = ? ORDER BY user_id, milestone_id`, titleSlug)
	if err != nil {
		return nil, fmt.Errorf("load earned: %w", err)
	}
	defer rows.Close()
	out := make(map[string][]string)
	for rows.Next() {
		var xuid, id string
		if err := rows.Scan(&xuid, &id); err != nil {
			return nil, fmt.Errorf("scan earned: %w", err)
		}
		out[xuid] = append(out[xuid], id)
	}
	return out, rows.Err()
}

// loadCrossingMatches charge les matchs d'un xuid, triés par start canonique ASC.
func loadCrossingMatches(ctx context.Context, sharedDB *sql.DB, xuid string) ([]MilestoneCrossingMatch, error) {
	// Fragment timezone canonique OBLIGATOIRE (CLAUDE.md n°8) : start_time_utc
	// (TIMESTAMPTZ UTC garanti) sinon start_time interprété en UTC.
	rows, err := sharedDB.QueryContext(ctx, `
		SELECT
			COALESCE(mr.start_time_utc, mr.start_time AT TIME ZONE 'UTC') AS played_at,
			mp.outcome,
			COALESCE(mp.kills, 0), COALESCE(mp.headshot_kills, 0), COALESCE(mp.assists, 0),
			COALESCE(mp.deaths, 0), COALESCE(mp.accuracy, 0),
			COALESCE(mp.damage_dealt, 0), COALESCE(mp.damage_taken, 0)
		FROM match_participants mp
		JOIN match_registry mr ON mp.match_id = mr.match_id
		WHERE mp.xuid = ?
		ORDER BY played_at ASC`, xuid)
	if err != nil {
		return nil, fmt.Errorf("load matches for %s: %w", xuid, err)
	}
	defer rows.Close()
	var out []MilestoneCrossingMatch
	for rows.Next() {
		var (
			m        MilestoneCrossingMatch
			playedAt sql.NullTime
			outcome  int
		)
		if err := rows.Scan(&playedAt, &outcome, &m.Kills, &m.Headshots, &m.Assists,
			&m.Deaths, &m.Accuracy, &m.DamageDealt, &m.DamageTaken); err != nil {
			return nil, fmt.Errorf("scan match: %w", err)
		}
		if playedAt.Valid {
			m.PlayedAt = playedAt.Time
		}
		m.Win = outcome == winOutcome
		out = append(out, m)
	}
	return out, rows.Err()
}

// updateEarnedAt écrit la date (ou NULL) pour un jalon. earned_at n'est PAS
// indexée → UPDATE ART-safe.
func updateEarnedAt(ctx context.Context, statsDB *sql.DB, xuid, titleSlug, milestoneID string, at *time.Time) error {
	var val sql.NullTime
	if at != nil {
		val = sql.NullTime{Time: *at, Valid: true}
	}
	if _, err := statsDB.ExecContext(ctx,
		`UPDATE milestone_earned SET earned_at = ? WHERE user_id = ? AND title_slug = ? AND milestone_id = ?`,
		val, xuid, titleSlug, milestoneID); err != nil {
		return fmt.Errorf("update earned_at %s/%s: %w", xuid, milestoneID, err)
	}
	return nil
}

// ensureEarnedAtNullable rend milestone_earned.earned_at nullable si elle ne
// l'est pas déjà (le backfill peut poser NULL). Idempotent.
func ensureEarnedAtNullable(ctx context.Context, statsDB *sql.DB) error {
	var nullable string
	err := statsDB.QueryRowContext(ctx,
		`SELECT is_nullable FROM information_schema.columns
		 WHERE table_name = 'milestone_earned' AND column_name = 'earned_at'`).Scan(&nullable)
	if err != nil {
		return fmt.Errorf("check earned_at nullable: %w", err)
	}
	if nullable == "YES" {
		return nil
	}
	// DuckDB refuse ALTER COLUMN tant qu'un index secondaire dépend de la table :
	// on drop l'index, on retire NOT NULL, on recrée l'index à l'identique.
	stmts := []string{
		`DROP INDEX IF EXISTS idx_ms_earned_user_title`,
		`ALTER TABLE milestone_earned ALTER earned_at DROP NOT NULL`,
		`CREATE INDEX IF NOT EXISTS idx_ms_earned_user_title ON milestone_earned(user_id, title_slug)`,
	}
	for _, stmt := range stmts {
		if _, err := statsDB.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("drop not null earned_at (%q): %w", stmt, err)
		}
	}
	return nil
}
