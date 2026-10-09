// Package ops — build_queue.go : le CHEMIN D'ÉCRITURE de la file durable de
// construction (mise en file, prise, battement, résultat, reprise).
//
// Porté par MonitoringStore (même base globale, même handle, même lease
// KindMonitoring) : la file n'introduit ni base, ni writer, ni modèle
// d'écriture. Toutes les écritures sont des INSERT (append-only, ADR 0026) ;
// l'état courant se lit par build_jobs_latest / build_workers_latest.
//
// ATOMICITÉ DE LA PRISE. Deux ouvriers ne doivent jamais repartir avec le même
// job. La section critique « lire le prochain en file → écrire sa prise » tient
// sous DEUX verrous cumulés : queueMu (les goroutines HTTP de ce process) et le
// lease dblease KindMonitoring (le writer unique de la base). Le modèle
// mono-process writer (ADR 0013/0016) garantit qu'aucun autre process n'écrit
// cette base — la prise est donc bien atomique, sans transaction distribuée.
//
// LA MORT D'UN OUVRIER NE PERD AUCUN JOB. Chaque prise pose un BAIL
// (lease_expires_at). Un job pris dont le bail a expiré retourne en file
// (attempt+1) au premier passage du ramasseur — que l'ouvrier soit mort,
// débranché, ou que ce soit le serveur web qui ait redémarré. C'est le même
// mécanisme pour les trois cas, donc il n'y en a qu'un à tester.
package ops

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/platform/jobs"
)

// ErrBuildJobNotClaimed : le rendu de résultat ne correspond à aucune prise
// vivante (job inconnu, déjà terminal, ou repris par un autre ouvrier après
// expiration du bail). L'appelant HTTP en fait un 409 : le travail de cet
// ouvrier est périmé, il doit redemander un job.
var ErrBuildJobNotClaimed = errors.New("build queue: job non détenu par cet ouvrier")

// buildJobColumns : les colonnes de build_jobs_latest, dans l'ordre de scanBuildJob.
const buildJobColumns = `job_id, job_type, title_slug, match_id, status, priority, attempt,
	worker_id, lease_expires_at, payload_json, result_json, error_code, error_message,
	enqueued_at, written_at`

// EnqueueBuildJobRequest décrit un travail à mettre en file.
type EnqueueBuildJobRequest struct {
	JobType   string
	TitleSlug string
	MatchID   string
	Priority  int
	// Payload : le travail RÉSOLU par le web (URL CDN pré-signées). Résolu à la
	// mise en file, pas à la prise — c'est le web qui a les tokens.
	Payload *domain.BuildQueuePayload
}

// EnqueueBuildJob met un travail en file et rend (job, créé). créé=false quand un
// job du même (type, titre, match) est déjà en file ou en cours : re-cliquer sur
// « construire » ne double pas le travail (idempotence de la mise en file — la
// reconstruction et la purge ne se courent pas après, piste F §1bis).
func (s *MonitoringStore) EnqueueBuildJob(ctx context.Context, req EnqueueBuildJobRequest) (domain.BuildQueueJob, bool, error) {
	if s == nil || s.db == nil {
		return domain.BuildQueueJob{}, false, fmt.Errorf("build queue: store non ouvert")
	}
	if strings.TrimSpace(req.JobType) == "" || strings.TrimSpace(req.MatchID) == "" {
		return domain.BuildQueueJob{}, false, fmt.Errorf("build queue: job_type et match_id requis")
	}
	payload, err := marshalPayload(req.Payload)
	if err != nil {
		return domain.BuildQueueJob{}, false, err
	}

	s.queueMu.Lock()
	defer s.queueMu.Unlock()
	w, err := s.acquire()
	if err != nil {
		return domain.BuildQueueJob{}, false, err
	}
	defer w.Release()

	existing, err := s.activeJobFor(ctx, w, req.JobType, req.TitleSlug, req.MatchID)
	if err != nil {
		return domain.BuildQueueJob{}, false, err
	}
	if existing != nil {
		return *existing, false, nil
	}

	now := time.Now().UTC()
	job := domain.BuildQueueJob{
		JobID:      jobs.NewID(),
		JobType:    req.JobType,
		TitleSlug:  req.TitleSlug,
		MatchID:    req.MatchID,
		Status:     domain.JobStatusQueued,
		Priority:   req.Priority,
		EnqueuedAt: now.Format(time.RFC3339),
		UpdatedAt:  now.Format(time.RFC3339),
	}
	if err := insertJobEvent(ctx, w, jobEvent{
		jobID: job.JobID, jobType: job.JobType, titleSlug: job.TitleSlug, matchID: job.MatchID,
		status: domain.JobStatusQueued, priority: req.Priority, attempt: 0,
		payload: payload, enqueuedAt: now,
	}); err != nil {
		return domain.BuildQueueJob{}, false, err
	}
	slog.InfoContext(ctx, "build queue: job mis en file",
		"module", "monitoring", "job_id", job.JobID, "job_type", job.JobType,
		"match_id", job.MatchID, "title", job.TitleSlug, "priority", req.Priority)
	return job, true, nil
}

// ClaimBuildJob prend le prochain job de la file pour un ouvrier, ou rend nil si
// la file est vide. Le ramasseur de bails expirés tourne D'ABORD : c'est le
// moment naturel où reprendre le travail d'un ouvrier disparu, et il n'y a pas
// besoin d'un cron pour ça.
//
// Ordre de service : priorité décroissante, puis ancienneté de mise en file —
// un rattrapage massif ne double jamais un match qu'un admin vient de demander.
func (s *MonitoringStore) ClaimBuildJob(ctx context.Context, workerID, hostname, version string) (*domain.BuildQueueJob, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("build queue: store non ouvert")
	}
	if strings.TrimSpace(workerID) == "" {
		return nil, fmt.Errorf("build queue: worker_id requis")
	}
	s.queueMu.Lock()
	defer s.queueMu.Unlock()
	w, err := s.acquire()
	if err != nil {
		return nil, err
	}
	defer w.Release()

	if _, err := s.reclaimLocked(ctx, w); err != nil {
		return nil, err
	}

	row := w.QueryRowContext(ctx, `SELECT `+buildJobColumns+`
		FROM build_jobs_latest WHERE status = ?
		ORDER BY priority DESC, enqueued_at ASC, job_id ASC LIMIT 1`,
		string(domain.JobStatusQueued))
	job, err := scanBuildJob(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		// File vide : on enregistre quand même le battement — un ouvrier au repos
		// est un ouvrier VIVANT, et l'admin doit voir la différence avec un mort.
		return nil, s.beatLocked(ctx, w, workerBeat{workerID: workerID, hostname: hostname, version: version})
	}
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	lease := now.Add(domain.BuildJobLeaseDuration)
	job.Status = domain.JobStatusRunning
	job.WorkerID = workerID
	job.LeaseExpiresAt = lease.Format(time.RFC3339)
	job.UpdatedAt = now.Format(time.RFC3339)
	if err := insertJobEvent(ctx, w, jobEvent{
		jobID: job.JobID, jobType: job.JobType, titleSlug: job.TitleSlug, matchID: job.MatchID,
		status: domain.JobStatusRunning, priority: job.Priority, attempt: job.Attempt,
		workerID: workerID, lease: &lease, payload: job.rawPayload,
		enqueuedAt: parseRFC3339(job.EnqueuedAt),
	}); err != nil {
		return nil, err
	}
	if err := s.beatLocked(ctx, w, workerBeat{
		workerID: workerID, hostname: hostname, version: version, currentJobID: job.JobID,
	}); err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "build queue: job pris par un ouvrier",
		"module", "monitoring", "job_id", job.JobID, "worker_id", workerID,
		"match_id", job.MatchID, "attempt", job.Attempt)
	// Le travail résolu (URL CDN pré-signées) n'est servi qu'ICI : c'est la seule
	// réponse dont l'ouvrier a besoin, et la seule où ces URL ont un sens.
	out := job.BuildQueueJob
	out.Payload = job.DecodePayload(ctx)
	return &out, nil
}

// CompleteBuildJobRequest : le résultat rendu par un ouvrier.
type CompleteBuildJobRequest struct {
	JobID    string
	WorkerID string
	// Succeeded=false → le job part en `failed` avec son code/message.
	Succeeded    bool
	ResultJSON   string
	ErrorCode    string
	ErrorMessage string
}

// CompleteBuildJob enregistre le résultat d'un job. Refuse (ErrBuildJobNotClaimed)
// un rendu qui ne correspond pas à une prise vivante de CET ouvrier : sans ce
// contrôle, un ouvrier lent dont le bail a expiré écraserait le travail de celui
// qui a repris son job.
func (s *MonitoringStore) CompleteBuildJob(ctx context.Context, req CompleteBuildJobRequest) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("build queue: store non ouvert")
	}
	s.queueMu.Lock()
	defer s.queueMu.Unlock()
	w, err := s.acquire()
	if err != nil {
		return err
	}
	defer w.Release()

	row := w.QueryRowContext(ctx, `SELECT `+buildJobColumns+` FROM build_jobs_latest WHERE job_id = ?`, req.JobID)
	job, err := scanBuildJob(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w (job inconnu %s)", ErrBuildJobNotClaimed, req.JobID)
	}
	if err != nil {
		return err
	}
	if job.Status != domain.JobStatusRunning || job.WorkerID != req.WorkerID {
		return fmt.Errorf("%w (job %s: statut %s, ouvrier %q)", ErrBuildJobNotClaimed,
			req.JobID, job.Status, job.WorkerID)
	}

	status := domain.JobStatusSucceeded
	if !req.Succeeded {
		status = domain.JobStatusFailed
	}
	if err := insertJobEvent(ctx, w, jobEvent{
		jobID: job.JobID, jobType: job.JobType, titleSlug: job.TitleSlug, matchID: job.MatchID,
		status: status, priority: job.Priority, attempt: job.Attempt, workerID: req.WorkerID,
		result: req.ResultJSON, errCode: req.ErrorCode, errMsg: req.ErrorMessage,
		enqueuedAt: parseRFC3339(job.EnqueuedAt),
	}); err != nil {
		return err
	}
	slog.InfoContext(ctx, "build queue: résultat rendu",
		"module", "monitoring", "job_id", job.JobID, "worker_id", req.WorkerID,
		"status", string(status), "err_code", req.ErrorCode)
	return nil
}

// HeartbeatRequest : un signe de vie d'ouvrier. Structure plutôt que huit
// paramètres positionnels — au-delà de cinq, l'ordre des arguments devient un
// piège (seuil du dépôt, ratchet revive argument-limit).
type HeartbeatRequest struct {
	WorkerID string
	Hostname string
	Version  string
	// JobID : le travail en cours, s'il y en a un. Vide = ouvrier au repos.
	JobID string
	Note  string
	// JobsDone / JobsFailed : compteurs CUMULÉS de l'ouvrier depuis son démarrage.
	JobsDone   int64
	JobsFailed int64
}

// HeartbeatBuildWorker enregistre un battement d'ouvrier et PROLONGE le bail du
// job qu'il traite. Le bail n'est renouvelé qu'à sa seconde moitié : un battement
// toutes les 30 s n'écrit pas 10 événements de job pendant un décodage de 5 min.
func (s *MonitoringStore) HeartbeatBuildWorker(ctx context.Context, req HeartbeatRequest) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("build queue: store non ouvert")
	}
	if strings.TrimSpace(req.WorkerID) == "" {
		return fmt.Errorf("build queue: worker_id requis")
	}
	s.queueMu.Lock()
	defer s.queueMu.Unlock()
	w, err := s.acquire()
	if err != nil {
		return err
	}
	defer w.Release()

	if err := s.beatLocked(ctx, w, workerBeat{
		workerID: req.WorkerID, hostname: req.Hostname, version: req.Version,
		currentJobID: req.JobID, note: req.Note, done: req.JobsDone, failed: req.JobsFailed,
	}); err != nil {
		return err
	}
	if strings.TrimSpace(req.JobID) == "" {
		return nil
	}
	return s.renewLeaseLocked(ctx, w, req.JobID, req.WorkerID)
}

// renewLeaseLocked prolonge le bail d'un job en cours détenu par cet ouvrier.
// Silencieux si le job n'est plus à lui (il a été repris) : c'est le rendu de
// résultat qui tranchera, pas le battement.
func (s *MonitoringStore) renewLeaseLocked(ctx context.Context, w *dblease.LeasedWriter, jobID, workerID string) error {
	row := w.QueryRowContext(ctx, `SELECT `+buildJobColumns+` FROM build_jobs_latest WHERE job_id = ?`, jobID)
	job, err := scanBuildJob(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if job.Status != domain.JobStatusRunning || job.WorkerID != workerID {
		return nil
	}
	now := time.Now().UTC()
	if expiry := parseRFC3339(job.LeaseExpiresAt); !expiry.IsZero() &&
		expiry.Sub(now) > domain.BuildJobLeaseDuration/2 {
		return nil // bail encore large : rien à écrire
	}
	lease := now.Add(domain.BuildJobLeaseDuration)
	return insertJobEvent(ctx, w, jobEvent{
		jobID: job.JobID, jobType: job.JobType, titleSlug: job.TitleSlug, matchID: job.MatchID,
		status: domain.JobStatusRunning, priority: job.Priority, attempt: job.Attempt,
		workerID: workerID, lease: &lease, payload: job.rawPayload,
		enqueuedAt: parseRFC3339(job.EnqueuedAt),
	})
}

// ReclaimExpiredBuildJobs rend à la file les jobs dont le bail a expiré. Rend le
// nombre de jobs repris. Appelé à chaque prise (le moment naturel) et par la
// boucle de flush monitoring (pour que l'admin voie la reprise même sans ouvrier
// vivant pour la déclencher).
func (s *MonitoringStore) ReclaimExpiredBuildJobs(ctx context.Context) (int, error) {
	if s == nil || s.db == nil {
		return 0, nil
	}
	s.queueMu.Lock()
	defer s.queueMu.Unlock()
	w, err := s.acquire()
	if err != nil {
		return 0, err
	}
	defer w.Release()
	return s.reclaimLocked(ctx, w)
}

// reclaimLocked : le ramasseur proprement dit (appelé sous queueMu + lease).
// Un job repris trop de fois part en `failed` plutôt que de tourner en rond —
// un film corrompu ne se décodera pas mieux au quatrième essai.
func (s *MonitoringStore) reclaimLocked(ctx context.Context, w *dblease.LeasedWriter) (int, error) {
	now := time.Now().UTC()
	rows, err := w.QueryContext(ctx, `SELECT `+buildJobColumns+`
		FROM build_jobs_latest
		WHERE status = ? AND lease_expires_at IS NOT NULL AND lease_expires_at < ?`,
		string(domain.JobStatusRunning), now)
	if err != nil {
		return 0, fmt.Errorf("build queue: lecture des bails expirés: %w", err)
	}
	var stale []scannedJob
	for rows.Next() {
		job, serr := scanBuildJob(rows.Scan)
		if serr != nil {
			_ = rows.Close()
			return 0, serr
		}
		stale = append(stale, job)
	}
	if err := errors.Join(rows.Err(), rows.Close()); err != nil {
		return 0, fmt.Errorf("build queue: lecture des bails expirés: %w", err)
	}

	reclaimed := 0
	for _, job := range stale {
		ev := jobEvent{
			jobID: job.JobID, jobType: job.JobType, titleSlug: job.TitleSlug, matchID: job.MatchID,
			status: domain.JobStatusQueued, priority: job.Priority, attempt: job.Attempt + 1,
			payload: job.rawPayload, enqueuedAt: parseRFC3339(job.EnqueuedAt),
		}
		if ev.attempt >= domain.BuildJobMaxAttempts {
			ev.status = domain.JobStatusFailed
			ev.errCode = "lease_expired"
			ev.errMessage()
		}
		if err := insertJobEvent(ctx, w, ev); err != nil {
			return reclaimed, err
		}
		reclaimed++
		slog.WarnContext(ctx, "build queue: bail expiré — job repris",
			"module", "monitoring", "job_id", job.JobID, "worker_id", job.WorkerID,
			"attempt", ev.attempt, "status", string(ev.status))
	}
	return reclaimed, nil
}

// ─── Écriture bas niveau ─────────────────────────────────────────────────────

// jobEvent : une transition d'état à écrire (INSERT pur, jamais d'UPDATE).
type jobEvent struct {
	jobID, jobType, titleSlug, matchID string
	status                             domain.JobStatus
	priority, attempt                  int
	workerID                           string
	lease                              *time.Time
	payload, result                    string
	errCode, errMsg                    string
	enqueuedAt                         time.Time
}

// errMessage pose le message d'échec standard d'un abandon sur bail expiré.
func (e *jobEvent) errMessage() {
	e.errMsg = fmt.Sprintf("ouvrier disparu %d fois de suite (bail expiré)", e.attempt)
}

// insertJobEvent écrit une transition d'état. enqueuedAt est REPORTÉ d'événement
// en événement : c'est la date de mise en file d'origine qui ordonne la file,
// pas celle du dernier événement — sinon un job repris passerait devant.
func insertJobEvent(ctx context.Context, w *dblease.LeasedWriter, e jobEvent) error {
	var lease any
	if e.lease != nil {
		lease = *e.lease
	}
	enqueued := e.enqueuedAt
	if enqueued.IsZero() {
		enqueued = time.Now().UTC()
	}
	if _, err := w.ExecContext(ctx,
		`INSERT INTO build_job_events
			(job_id, job_type, title_slug, match_id, status, priority, attempt,
			 worker_id, lease_expires_at, payload_json, result_json,
			 error_code, error_message, enqueued_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.jobID, e.jobType, e.titleSlug, e.matchID, string(e.status), e.priority, e.attempt,
		e.workerID, lease, e.payload, e.result, e.errCode, e.errMsg, enqueued,
	); err != nil {
		return fmt.Errorf("build queue: insert événement de job: %w", err)
	}
	return nil
}

// workerBeat : un battement d'ouvrier à écrire.
type workerBeat struct {
	workerID, hostname, version string
	currentJobID, note          string
	done, failed                int64
}

// beatLocked écrit un battement (appelé sous queueMu + lease).
func (s *MonitoringStore) beatLocked(ctx context.Context, w *dblease.LeasedWriter, b workerBeat) error {
	if _, err := w.ExecContext(ctx,
		`INSERT INTO build_worker_events
			(worker_id, hostname, version, current_job_id, jobs_done, jobs_failed, note, beat_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		b.workerID, b.hostname, b.version, b.currentJobID, b.done, b.failed, b.note,
		time.Now().UTC(),
	); err != nil {
		return fmt.Errorf("build queue: insert battement d'ouvrier: %w", err)
	}
	return nil
}

// activeJobFor rend le job non terminal du même (type, titre, match), s'il existe.
func (s *MonitoringStore) activeJobFor(ctx context.Context, w *dblease.LeasedWriter, jobType, titleSlug, matchID string) (*domain.BuildQueueJob, error) {
	row := w.QueryRowContext(ctx, `SELECT `+buildJobColumns+`
		FROM build_jobs_latest
		WHERE job_type = ? AND COALESCE(title_slug, '') = ? AND match_id = ?
		  AND status IN (?, ?) LIMIT 1`,
		jobType, titleSlug, matchID,
		string(domain.JobStatusQueued), string(domain.JobStatusRunning))
	job, err := scanBuildJob(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &job.BuildQueueJob, nil
}

// marshalPayload sérialise le travail résolu (URL pré-signées). Payload nil =
// chaîne vide : un job sans travail résolu reste licite (l'ouvrier saura le
// refuser), on ne veut pas d'un "null" en base.
func marshalPayload(p *domain.BuildQueuePayload) (string, error) {
	if p == nil {
		return "", nil
	}
	raw, err := json.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("build queue: sérialisation du travail: %w", err)
	}
	return string(raw), nil
}
