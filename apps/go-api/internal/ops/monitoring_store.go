// Package ops — monitoring_store.go : writer unique de la base monitoring
// globale (data/global/monitoring.duckdb, DC-1 du plan monitoring 2026-07).
//
// Rôle : persister l'observabilité du dashboard admin pour survivre au restart
// (l'ancien monitoring était un instantané mémoire — un reboot = amnésie). Trois
// flux d'écriture :
//   - détections (flush périodique du delta de l'ErrorCollector) + leur cycle de
//     vie (open/acked/muted/resolved) ;
//   - historique des crons (RecordCronRun) ;
//   - dernier audit data-health sérialisé (RecordDataHealthRun).
//
// Invariants (ADR 0019/0026) : écriture = INSERT pur (append-only), lecture via
// les vues `_latest`, un seul writer = le process serveur. L'ouverture passe par
// le provider platform (duckdb.OpenReadWrite, jamais de bare connect) ; la
// sérialisation des écritures concurrentes (flush goroutine, hooks crons,
// data-health) passe par un lease dblease.KindMonitoring.
package ops

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/platform/duckdb"
)

// monitoringLeaseTimeout borne l'attente d'acquisition du writer monitoring.
// Court : mono-process, contention seulement entre goroutines internes.
const monitoringLeaseTimeout = 5 * time.Second

// DetectionSample est l'échantillon d'entrée d'un flush de détections — mappé
// depuis observability.ErrorBucket par le caller (découple ops d'observability).
// Count est CUMULATIF depuis le boot ; le store calcule le delta persisté.
type DetectionSample struct {
	Title        string
	Level        string
	Module       string
	Message      string
	SampleDetail string
	Count        int64
	FirstSeen    time.Time
	LastSeen     time.Time
}

// MonitoringStore est le writer/lecteur unique de la base monitoring globale.
type MonitoringStore struct {
	db   *duckdb.DB
	path string

	mu         sync.Mutex       // protège lastCounts
	lastCounts map[string]int64 // fingerprint -> count cumulatif au dernier flush (mémoire process)
}

// NewMonitoringStore ouvre (ou crée) la base monitoring globale et pose son
// schéma idempotemment. À appeler une fois au boot ; Close() au shutdown.
func NewMonitoringStore(ctx context.Context, path string) (*MonitoringStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("monitoring store: mkdir: %w", err)
	}
	db, err := duckdb.OpenReadWrite(path)
	if err != nil {
		return nil, fmt.Errorf("monitoring store: open %s: %w", path, err)
	}
	if err := migration.EnsureMonitoringSchema(ctx, db.SQLDb()); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("monitoring store: schema: %w", err)
	}
	return &MonitoringStore{db: db, path: path, lastCounts: make(map[string]int64)}, nil
}

// Close relâche le handle DB (le lease éventuel est déjà relâché par méthode).
func (s *MonitoringStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// acquire prend le writer exclusif monitoring (sérialise les goroutines internes).
func (s *MonitoringStore) acquire() (*dblease.LeasedWriter, error) {
	return dblease.AcquireWriter(s.db.SQLDb(), s.path, dblease.KindMonitoring, monitoringLeaseTimeout)
}

// FlushDetections persiste, pour chaque échantillon, le delta d'occurrences
// depuis le dernier flush (INSERT append-only). Une détection dont le statut
// courant est `resolved` est ré-ouverte (append d'un status 'open') dès qu'une
// nouvelle occurrence arrive (DC-2). No-op si aucun delta.
func (s *MonitoringStore) FlushDetections(ctx context.Context, samples []DetectionSample) error {
	if len(samples) == 0 {
		return nil
	}
	w, err := s.acquire()
	if err != nil {
		return err
	}
	defer w.Release()

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, sm := range samples {
		fp := domain.DetectionFingerprint(sm.Title, sm.Level, sm.Module, sm.Message)
		delta := sm.Count - s.lastCounts[fp]
		if delta <= 0 {
			continue
		}
		if _, err := w.ExecContext(ctx,
			`INSERT INTO detection_events
				(fingerprint, level, module, message, title_slug, occurred_at, count_delta, sample_detail)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			fp, sm.Level, sm.Module, sm.Message, sm.Title, sm.LastSeen, delta, sm.SampleDetail,
		); err != nil {
			return fmt.Errorf("monitoring store: insert detection: %w", err)
		}
		s.lastCounts[fp] = sm.Count
		if err := s.reopenIfResolved(ctx, w, fp); err != nil {
			return err
		}
	}
	return nil
}

// reopenIfResolved ré-ouvre une détection résolue quand une occurrence arrive.
// Appelé sous lease + lock. Best-effort sur la lecture du statut courant.
func (s *MonitoringStore) reopenIfResolved(ctx context.Context, w *dblease.LeasedWriter, fingerprint string) error {
	var status string
	err := w.QueryRowContext(ctx,
		`SELECT status FROM detection_status_latest WHERE fingerprint = ?`, fingerprint,
	).Scan(&status)
	if err == sql.ErrNoRows {
		return nil // jamais statué → déjà 'open' par défaut
	}
	if err != nil {
		return fmt.Errorf("monitoring store: read status: %w", err)
	}
	if status != domain.DetectionStatusResolved {
		return nil
	}
	if _, err := w.ExecContext(ctx,
		`INSERT INTO detection_status_events (fingerprint, status, note) VALUES (?, ?, ?)`,
		fingerprint, domain.DetectionStatusOpen, "ré-ouvert (nouvelle occurrence)",
	); err != nil {
		return fmt.Errorf("monitoring store: reopen: %w", err)
	}
	return nil
}

// SetDetectionStatus append un événement de cycle de vie (open/acked/muted/
// resolved) pour un fingerprint. Le statut est validé côté domain.
func (s *MonitoringStore) SetDetectionStatus(ctx context.Context, fingerprint, status, note string) error {
	if !domain.IsValidDetectionStatus(status) {
		return fmt.Errorf("monitoring store: statut invalide %q", status)
	}
	w, err := s.acquire()
	if err != nil {
		return err
	}
	defer w.Release()
	if _, err := w.ExecContext(ctx,
		`INSERT INTO detection_status_events (fingerprint, status, note) VALUES (?, ?, ?)`,
		fingerprint, status, note,
	); err != nil {
		return fmt.Errorf("monitoring store: set status: %w", err)
	}
	return nil
}

// DetectionFilter filtre la lecture des détections (tous champs optionnels).
type DetectionFilter struct {
	Status string
	Level  string
	Module string
	Title  string
	Limit  int
}

// Detections lit la vue detections_latest filtrée, triée par last_seen desc.
// OpenCount de la réponse compte les détections au statut open (source badges).
func (s *MonitoringStore) Detections(ctx context.Context, f DetectionFilter) (domain.AdminDetectionsResponse, error) {
	resp := domain.AdminDetectionsResponse{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Detections:  []domain.MonitoringDetection{},
	}
	where, args := detectionWhere(f)
	limit := f.Limit
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	q := `SELECT fingerprint, level, module, message, title_slug, count,
			first_seen, last_seen, sample_detail, status, note, status_at
		FROM detections_latest` + where + `
		ORDER BY last_seen DESC NULLS LAST
		LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return resp, fmt.Errorf("monitoring store: list detections: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		d, err := scanDetection(rows)
		if err != nil {
			return resp, err
		}
		if d.Status == domain.DetectionStatusOpen {
			resp.OpenCount++
		}
		resp.Detections = append(resp.Detections, d)
	}
	return resp, rows.Err()
}

// detectionWhere construit la clause WHERE et ses args depuis le filtre.
func detectionWhere(f DetectionFilter) (string, []any) {
	var conds []string
	var args []any
	add := func(col, val string) {
		if val != "" {
			conds = append(conds, col+" = ?")
			args = append(args, val)
		}
	}
	add("status", f.Status)
	add("level", f.Level)
	add("module", f.Module)
	add("title_slug", f.Title)
	if len(conds) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

// scanDetection mappe une ligne detections_latest vers le DTO (timestamps
// nullable → RFC3339 ou "").
func scanDetection(rows *sql.Rows) (domain.MonitoringDetection, error) {
	var (
		d                               domain.MonitoringDetection
		module, titleSlug, sample, note sql.NullString
		firstSeen, lastSeen, statusAt   sql.NullTime
		count                           sql.NullInt64
	)
	if err := rows.Scan(&d.Fingerprint, &d.Level, &module, &d.Message, &titleSlug,
		&count, &firstSeen, &lastSeen, &sample, &d.Status, &note, &statusAt); err != nil {
		return d, fmt.Errorf("monitoring store: scan detection: %w", err)
	}
	d.Module = module.String
	d.TitleSlug = titleSlug.String
	d.SampleDetail = sample.String
	d.Note = note.String
	d.Count = count.Int64
	d.FirstSeen = rfc3339OrEmpty(firstSeen)
	d.LastSeen = rfc3339OrEmpty(lastSeen)
	d.StatusAt = rfc3339OrEmpty(statusAt)
	return d, nil
}

func rfc3339OrEmpty(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.UTC().Format(time.RFC3339)
}

// OpenDetectionCount compte les détections au statut open (source du badge nav,
// exposé en gauge expvar par le flush → overview zéro I/O).
func (s *MonitoringStore) OpenDetectionCount(ctx context.Context) (int, error) {
	var n int
	if err := s.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM detections_latest WHERE status = 'open'`,
	).Scan(&n); err != nil {
		return 0, fmt.Errorf("monitoring store: open detection count: %w", err)
	}
	return n, nil
}

// RecordCronRun persiste une exécution de cron (append-only, historique).
func (s *MonitoringStore) RecordCronRun(ctx context.Context, name string, startedAt time.Time, ok bool, errStr string, durationMs int64) error {
	w, err := s.acquire()
	if err != nil {
		return err
	}
	defer w.Release()
	if _, err := w.ExecContext(ctx,
		`INSERT INTO cron_runs (cron_name, started_at, ok, err, duration_ms) VALUES (?, ?, ?, ?, ?)`,
		name, startedAt, ok, errStr, durationMs,
	); err != nil {
		return fmt.Errorf("monitoring store: record cron run: %w", err)
	}
	return nil
}

// CronRunCount compte les exécutions enregistrées d'un cron (ex. marqueur
// server_boot → compteur de démarrages persistant, A5.1).
func (s *MonitoringStore) CronRunCount(ctx context.Context, name string) (int64, error) {
	var n int64
	if err := s.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM cron_runs WHERE cron_name = ?`, name,
	).Scan(&n); err != nil {
		return 0, fmt.Errorf("monitoring store: cron run count: %w", err)
	}
	return n, nil
}

// RecordDataHealthRun persiste le résultat sérialisé (JSON) du dernier audit
// data-health, pour que l'overview survive au restart.
func (s *MonitoringStore) RecordDataHealthRun(ctx context.Context, resultJSON string) error {
	w, err := s.acquire()
	if err != nil {
		return err
	}
	defer w.Release()
	if _, err := w.ExecContext(ctx,
		`INSERT INTO data_health_runs (result_json) VALUES (?)`, resultJSON,
	); err != nil {
		return fmt.Errorf("monitoring store: record data health: %w", err)
	}
	return nil
}

// LatestDataHealthJSON retourne le JSON du dernier audit data-health persisté
// (chaîne vide si aucun). Sert à réhydrater l'overview après restart.
func (s *MonitoringStore) LatestDataHealthJSON(ctx context.Context) (string, error) {
	var js sql.NullString
	err := s.db.QueryRow(ctx,
		`SELECT result_json FROM data_health_runs ORDER BY written_at DESC, id DESC LIMIT 1`,
	).Scan(&js)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("monitoring store: latest data health: %w", err)
	}
	return js.String, nil
}
