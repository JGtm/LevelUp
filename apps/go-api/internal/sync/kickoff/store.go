package kickoff

// store.go — la LECTURE de l'état rangé : registre et participants d'un lot de matchs, liste des
// matchs, révision rangée. Les lectures passent par le handle que l'appelant tient (segment de
// lecture, ou writer du burst en cours).

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/analysis/timeline"
	"levelup/go-api/internal/platform/duckdb/sqlfrag"
)

// SyncMetaKey est la clé de `sync_meta` (base partagée) qui porte la dernière révision
// `timeline.KickoffRev` dont le rattrapage complet a réussi.
const SyncMetaKey = "kickoff_revision"

// queryer est la surface de lecture commune de *sql.DB et *sql.Tx.
type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// matchIDs rend tous les matchs du registre, dans l'ordre de leur identifiant (l'ordre des lots).
func matchIDs(ctx context.Context, q queryer) ([]string, error) {
	rows, err := q.QueryContext(ctx, `SELECT match_id FROM match_registry ORDER BY match_id`)
	if err != nil {
		return nil, fmt.Errorf("kickoff: liste des matchs: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("kickoff: liste des matchs (scan): %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// sourceColumn rend l'expression qui lit la source rangée : la colonne, ou NULL sur une base que la
// migration `shared_registry_real_start_source_v1` n'a pas encore reçue (simulation en lecture
// seule, sans migration) — une telle base ne range aucune source.
func sourceColumn(ctx context.Context, q queryer) (string, error) {
	var n int
	if err := q.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name = 'match_registry' AND column_name = 'real_start_source'`).Scan(&n); err != nil {
		return "", fmt.Errorf("kickoff: colonne real_start_source: %w", err)
	}
	if n == 0 {
		return "CAST(NULL AS VARCHAR)", nil
	}
	return "real_start_source", nil
}

// loadMatches lit l'état rangé des matchs ids. Un match absent du registre n'est pas rendu.
func loadMatches(ctx context.Context, q queryer, ids []string) (map[string]*Match, error) {
	out := make(map[string]*Match, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	src, err := sourceColumn(ctx, q)
	if err != nil {
		return nil, err
	}
	in, args := placeholders(ids)
	if err := loadRegistry(ctx, q, src, in, args, out); err != nil {
		return nil, err
	}
	if err := loadParticipants(ctx, q, in, args, out); err != nil {
		return nil, err
	}
	return out, nil
}

func loadRegistry(ctx context.Context, q queryer, src, in string, args []any, out map[string]*Match) error {
	rows, err := q.QueryContext(ctx, `
		SELECT match_id, `+sqlfrag.SQLStartTimeCanonical("")+` AS start_utc,
		       duration_seconds, real_start_time, COALESCE(`+src+`, '')
		FROM match_registry WHERE match_id IN (`+in+`)`, args...)
	if err != nil {
		return fmt.Errorf("kickoff: lecture du registre: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var m Match
		var dur sql.NullInt64
		var realStart sql.NullTime
		var source string
		if err := rows.Scan(&m.MatchID, &m.StartUTC, &dur, &realStart, &source); err != nil {
			return fmt.Errorf("kickoff: lecture du registre (scan): %w", err)
		}
		m.StartUTC = m.StartUTC.UTC()
		if dur.Valid {
			d := int(dur.Int64)
			m.DurationSeconds = &d
		}
		m.RealStart = utcPtr(realStart)
		m.Source = timeline.KickoffSource(source)
		out[m.MatchID] = &m
	}
	return rows.Err()
}

func loadParticipants(ctx context.Context, q queryer, in string, args []any, out map[string]*Match) error {
	rows, err := q.QueryContext(ctx, `
		SELECT match_id, xuid, first_joined_time, last_leave_time,
		       COALESCE(present_at_beginning, false), time_played_seconds
		FROM match_participants WHERE match_id IN (`+in+`)
		ORDER BY match_id, xuid`, args...)
	if err != nil {
		return fmt.Errorf("kickoff: lecture des participants: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id string
		var p Participant
		var joined, left sql.NullTime
		var played sql.NullInt64
		if err := rows.Scan(&id, &p.XUID, &joined, &left, &p.PresentAtBeginning, &played); err != nil {
			return fmt.Errorf("kickoff: lecture des participants (scan): %w", err)
		}
		m, ok := out[id]
		if !ok {
			continue
		}
		p.FirstJoined, p.LastLeave = utcPtr(joined), utcPtr(left)
		if played.Valid {
			v := int(played.Int64)
			p.TimePlayed = &v
		}
		m.Participants = append(m.Participants, p)
	}
	return rows.Err()
}

// storedRevision rend la révision rangée dans `sync_meta`, vide quand aucune ne l'est.
func storedRevision(ctx context.Context, q queryer) (string, error) {
	var v sql.NullString
	err := q.QueryRowContext(ctx, `SELECT value FROM sync_meta WHERE key = ?`, SyncMetaKey).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("kickoff: révision rangée: %w", err)
	}
	return v.String, nil
}

// placeholders rend la liste `?, ?, …` d'un IN et ses arguments.
func placeholders(ids []string) (string, []any) {
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	return strings.TrimSuffix(strings.Repeat("?, ", len(ids)), ", "), args
}

func utcPtr(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	u := t.Time.UTC()
	return &u
}
