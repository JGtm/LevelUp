// Package ops — build_queue_read.go : le CHEMIN DE LECTURE de la file durable.
//
// Toutes les lectures passent par build_jobs_latest / build_workers_latest, et
// JAMAIS par les tables d'événements : lire build_job_events directement
// servirait l'état d'un job à l'avant-dernière transition (règle ART n°2).
//
// Ces lectures ne prennent pas le lease — comme les autres lectures du store
// (Detections, LatestCronRuns) : lire pendant qu'un writer écrit est sûr sur un
// même handle DuckDB, et prendre le writer pour un poll d'admin bloquerait la
// prise d'un job.
package ops

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
)

// scannedJob : un job lu, plus le JSON BRUT de son travail — que le chemin de
// prise doit reporter d'événement en événement sans le désérialiser.
type scannedJob struct {
	domain.BuildQueueJob
	rawPayload string
}

// scanFn : la signature commune de (*sql.Row).Scan et (*sql.Rows).Scan, pour que
// la lecture d'une ligne de job n'existe qu'une fois.
type scanFn func(dest ...any) error

// scanBuildJob lit une ligne de build_jobs_latest (ordre buildJobColumns).
func scanBuildJob(scan scanFn) (scannedJob, error) {
	var (
		j                                    scannedJob
		titleSlug, matchID, workerID         sql.NullString
		payload, result, errCode, errMessage sql.NullString
		status                               string
		priority, attempt                    sql.NullInt64
		lease, enqueued, written             sql.NullTime
	)
	if err := scan(&j.JobID, &j.JobType, &titleSlug, &matchID, &status, &priority, &attempt,
		&workerID, &lease, &payload, &result, &errCode, &errMessage, &enqueued, &written); err != nil {
		return j, err
	}
	j.TitleSlug = titleSlug.String
	j.MatchID = matchID.String
	j.Status = domain.JobStatus(status)
	j.Priority = int(priority.Int64)
	j.Attempt = int(attempt.Int64)
	j.WorkerID = workerID.String
	j.LeaseExpiresAt = rfc3339OrEmpty(lease)
	j.EnqueuedAt = rfc3339OrEmpty(enqueued)
	j.UpdatedAt = rfc3339OrEmpty(written)
	j.ResultJSON = result.String
	j.ErrorCode = errCode.String
	j.ErrorMessage = errMessage.String
	j.rawPayload = payload.String
	return j, nil
}

// parseRFC3339 rend l'instant d'une chaîne RFC3339, ou le zéro si elle est vide
// ou illisible (les horodatages viennent de la base, ils ne mentent pas ; le zéro
// est le repli sûr des lignes anciennes).
func parseRFC3339(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

// DecodePayload désérialise le travail d'un job (URL pré-signées) — utilisé par
// la réponse de prise, seul endroit où l'ouvrier a besoin du détail.
func (j scannedJob) DecodePayload(ctx context.Context) *domain.BuildQueuePayload {
	if j.rawPayload == "" {
		return nil
	}
	var p domain.BuildQueuePayload
	if err := json.Unmarshal([]byte(j.rawPayload), &p); err != nil {
		slog.WarnContext(ctx, "build queue: travail de job illisible",
			"module", "monitoring", "job_id", j.JobID, "err", err)
		return nil
	}
	return &p
}

// BuildQueueReport agrège l'état de bout en bout : les jobs récents, les
// ouvriers, et les compteurs de file. C'est LA source du panneau admin — l'état
// vit côté web, donc le dashboard n'interroge jamais l'ouvrier (piste F §4bis).
//
// Le travail (payload) n'est PAS servi ici : l'admin n'a que faire de 20 Ko d'URL
// pré-signées par job, et elles expirent.
func (s *MonitoringStore) BuildQueueReport(ctx context.Context, limit int) (domain.AdminBuildQueueResponse, error) {
	resp := domain.AdminBuildQueueResponse{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Jobs:        []domain.BuildQueueJob{},
		Workers:     []domain.BuildQueueWorker{},
	}
	if s == nil || s.db == nil {
		return resp, nil
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	jobs, counts, err := s.listBuildJobs(ctx, limit)
	if err != nil {
		return resp, err
	}
	workers, err := s.listBuildWorkers(ctx)
	if err != nil {
		return resp, err
	}
	resp.Jobs = jobs
	resp.Counts = counts
	resp.Workers = workers
	return resp, nil
}

// listBuildJobs rend les jobs récents (actifs d'abord) et les compteurs de file.
// Les compteurs sont calculés par une requête d'agrégat SÉPARÉE : les compter sur
// la page servie donnerait un « 3 en file » faux dès que la file dépasse limit.
func (s *MonitoringStore) listBuildJobs(ctx context.Context, limit int) ([]domain.BuildQueueJob, domain.BuildQueueCounts, error) {
	var counts domain.BuildQueueCounts
	crows, err := s.db.Query(ctx, `SELECT status, COUNT(*) FROM build_jobs_latest GROUP BY status`)
	if err != nil {
		return nil, counts, fmt.Errorf("build queue: compteurs: %w", err)
	}
	for crows.Next() {
		var status string
		var n int
		if err := crows.Scan(&status, &n); err != nil {
			_ = crows.Close()
			return nil, counts, fmt.Errorf("build queue: compteurs (scan): %w", err)
		}
		switch domain.JobStatus(status) {
		case domain.JobStatusQueued:
			counts.Queued = n
		case domain.JobStatusRunning:
			counts.Running = n
		case domain.JobStatusSucceeded:
			counts.Succeeded = n
		case domain.JobStatusFailed:
			counts.Failed = n
		}
	}
	if err := crows.Err(); err != nil {
		_ = crows.Close()
		return nil, counts, fmt.Errorf("build queue: compteurs: %w", err)
	}
	if err := crows.Close(); err != nil {
		return nil, counts, fmt.Errorf("build queue: compteurs: %w", err)
	}

	// Ordre d'affichage : en cours, puis en file, puis l'historique du plus récent
	// au plus ancien — l'admin veut voir ce qui bouge en haut de tableau.
	rows, err := s.db.Query(ctx, `SELECT `+buildJobColumns+` FROM build_jobs_latest
		ORDER BY CASE status WHEN 'running' THEN 0 WHEN 'queued' THEN 1 ELSE 2 END,
			written_at DESC, job_id DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, counts, fmt.Errorf("build queue: liste des jobs: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := make([]domain.BuildQueueJob, 0, limit)
	for rows.Next() {
		job, err := scanBuildJob(rows.Scan)
		if err != nil {
			return nil, counts, fmt.Errorf("build queue: liste des jobs (scan): %w", err)
		}
		out = append(out, job.BuildQueueJob)
	}
	return out, counts, rows.Err()
}

// listBuildWorkers rend les ouvriers connus, le plus récemment vu en premier.
// `online` se calcule ICI, à la lecture, contre WorkerOfflineAfter : un état « en
// ligne » persisté serait faux à la seconde suivante.
func (s *MonitoringStore) listBuildWorkers(ctx context.Context) ([]domain.BuildQueueWorker, error) {
	rows, err := s.db.Query(ctx, `SELECT worker_id, hostname, version, current_job_id,
			jobs_done, jobs_failed, note, beat_at
		FROM build_workers_latest ORDER BY beat_at DESC NULLS LAST`)
	if err != nil {
		return nil, fmt.Errorf("build queue: liste des ouvriers: %w", err)
	}
	defer func() { _ = rows.Close() }()
	now := time.Now().UTC()
	out := []domain.BuildQueueWorker{}
	for rows.Next() {
		var (
			wk                              domain.BuildQueueWorker
			hostname, version, curJob, note sql.NullString
			done, failed                    sql.NullInt64
			beatAt                          sql.NullTime
		)
		if err := rows.Scan(&wk.WorkerID, &hostname, &version, &curJob,
			&done, &failed, &note, &beatAt); err != nil {
			return nil, fmt.Errorf("build queue: liste des ouvriers (scan): %w", err)
		}
		wk.Hostname = hostname.String
		wk.Version = version.String
		wk.CurrentJobID = curJob.String
		wk.Note = note.String
		wk.JobsDone = done.Int64
		wk.JobsFailed = failed.Int64
		wk.LastBeatAt = rfc3339OrEmpty(beatAt)
		wk.Online = beatAt.Valid && now.Sub(beatAt.Time.UTC()) < domain.WorkerOfflineAfter
		out = append(out, wk)
	}
	return out, rows.Err()
}
