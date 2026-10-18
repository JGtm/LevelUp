// Package ops — match_steps.go : écriture et lecture des instants mesurés du chemin d'un match
// (table match_step_events de la base monitoring), source du rapport « délai du débrief ».
//
// Même writer, même lease KindMonitoring que la file de construction. Écriture = INSERT pur ;
// lecture sans lease, comme le reste du store.
package ops

import (
	"context"
	"database/sql"
	"fmt"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// RecordMatchSteps écrit un lot d'instants mesurés, sous UNE acquisition du writer. Un lot vide
// n'acquiert rien.
func (s *MonitoringStore) RecordMatchSteps(ctx context.Context, events []domain.MatchStepEvent) error {
	if len(events) == 0 {
		return nil
	}
	if s == nil || s.db == nil {
		return fmt.Errorf("match steps: store non ouvert")
	}
	w, err := s.acquire()
	if err != nil {
		return err
	}
	defer w.Release()
	for _, ev := range events {
		if _, err := w.ExecContext(ctx,
			`INSERT INTO match_step_events (title_slug, match_id, step, step_at, detail) VALUES (?, ?, ?, ?, ?)`,
			ev.TitleSlug, ev.MatchID, string(ev.Step), ev.At.UTC(), ev.Detail,
		); err != nil {
			return fmt.Errorf("match steps: insert %s/%s: %w", ev.MatchID, ev.Step, err)
		}
	}
	return nil
}

// MatchDelays rend le délai du débrief des `limit` matchs les plus récents (par fin de match,
// à défaut par dernier instant mesuré), du plus récent au plus ancien.
func (s *MonitoringStore) MatchDelays(ctx context.Context, limit int) ([]domain.MatchDelay, error) {
	out := []domain.MatchDelay{}
	if s == nil || s.db == nil {
		return out, nil
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.Query(ctx, `
		WITH recents AS (
			SELECT match_id,
				MAX(CASE WHEN step = ? THEN step_at END) AS fin,
				MAX(step_at) AS dernier
			FROM match_step_events
			GROUP BY match_id
			ORDER BY COALESCE(fin, dernier) DESC, match_id
			LIMIT ?
		)
		SELECT e.title_slug, e.match_id, e.step, e.step_at, e.detail
		FROM match_step_events e JOIN recents r USING (match_id)
		ORDER BY COALESCE(r.fin, r.dernier) DESC, e.match_id, e.step_at, e.id`,
		string(domain.MatchStepEnd), limit)
	if err != nil {
		return out, fmt.Errorf("match steps: lecture: %w", err)
	}
	defer func() { _ = rows.Close() }()

	parMatch := map[string][]domain.MatchStepEvent{}
	var ordre []string
	for rows.Next() {
		var (
			ev           domain.MatchStepEvent
			slug, detail sql.NullString
			step         string
		)
		if err := rows.Scan(&slug, &ev.MatchID, &step, &ev.At, &detail); err != nil {
			return out, fmt.Errorf("match steps: lecture (scan): %w", err)
		}
		ev.TitleSlug, ev.Step, ev.Detail = slug.String, domain.MatchStep(step), detail.String
		if _, vu := parMatch[ev.MatchID]; !vu {
			ordre = append(ordre, ev.MatchID)
		}
		parMatch[ev.MatchID] = append(parMatch[ev.MatchID], ev)
	}
	if err := rows.Err(); err != nil {
		return out, fmt.Errorf("match steps: lecture: %w", err)
	}
	for _, id := range ordre {
		out = append(out, analysis.DebriefDelay(parMatch[id]))
	}
	return out, nil
}
