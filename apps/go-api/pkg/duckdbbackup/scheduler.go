package duckdbbackup

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// Scheduler runs periodic DuckDB backups via restic.
// Instantiate with New; call Run in a goroutine.
type Scheduler struct {
	cfg      Config
	discover func() ([]Target, error)
	restic   *ResticClient
}

// New creates a Scheduler.
// discover is called at every cycle to obtain the current list of DBs to protect.
// The function may return a dynamic list (e.g. players added after server start).
func New(cfg Config, discover func() ([]Target, error)) *Scheduler {
	return &Scheduler{
		cfg:      cfg,
		discover: discover,
		restic:   NewResticClient(cfg),
	}
}

// Run starts the periodic backup loop. Must be called in a goroutine.
// The first cycle runs immediately at startup (same pattern as HealthScheduler).
func (s *Scheduler) Run(ctx context.Context) {
	if !s.cfg.Enabled {
		slog.InfoContext(ctx, "backup: désactivé par config")
		return
	}
	if !s.restic.IsAvailable() {
		slog.WarnContext(ctx, "backup: binaire restic introuvable dans le PATH — scheduler désactivé",
			"bin", s.restic.bin())
		return
	}

	interval := s.cfg.Interval
	if interval <= 0 {
		interval = 6 * time.Hour
	}

	slog.InfoContext(ctx, "backup: scheduler démarré", "interval", interval)
	s.runCycle(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.runCycle(ctx)
		case <-ctx.Done():
			slog.InfoContext(ctx, "backup: arrêt scheduler")
			return
		}
	}
}

// RunOnce executes a single backup cycle synchronously.
// Useful for tests and CLI one-shots without a goroutine.
func (s *Scheduler) RunOnce(ctx context.Context) (*Result, error) {
	return s.cycle(ctx)
}

func (s *Scheduler) runCycle(ctx context.Context) {
	if _, err := s.cycle(ctx); err != nil {
		slog.WarnContext(ctx, "backup: cycle échoué (non-bloquant)", "err", err)
	}
}

func (s *Scheduler) cycle(ctx context.Context) (*Result, error) {
	start := time.Now()

	if err := os.MkdirAll(s.cfg.BackupDir, 0o755); err != nil {
		return nil, fmt.Errorf("backup: mkdir staging: %w", err)
	}

	manifestPath := filepath.Join(s.cfg.BackupDir, ".manifest.json")
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("backup: chargement manifest: %w", err)
	}

	targets, err := s.discover()
	if err != nil {
		return nil, fmt.Errorf("backup: découverte DBs: %w", err)
	}

	var exported []string
	for _, t := range targets {
		changed, chErr := manifest.HasChanged(t)
		if chErr != nil {
			// DB inaccessible (pas encore créée, chemin incorrect…) — skip silencieux.
			slog.DebugContext(ctx, "backup: fingerprint skip", "key", t.Key, "err", chErr)
			continue
		}
		if !changed {
			continue
		}

		ic := CheckIntegrity(ctx, t)
		manifest.SetIntegrityResult(t.Key, ic)
		if !ic.OK {
			slog.WarnContext(ctx, "backup: intégrité DB dégradée (sauvegarde maintenue)",
				"key", t.Key, "detail", ic.Detail)
		}

		outDir := filepath.Join(s.cfg.BackupDir, "staging", t.Key)
		exportCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		_, exportErr := ExportTarget(exportCtx, t, outDir, s.cfg.CompressionLevel)
		cancel()
		if exportErr != nil {
			slog.WarnContext(ctx, "backup: export échoué", "key", t.Key, "err", exportErr)
			continue
		}
		if err := manifest.MarkSaved(t); err != nil {
			slog.WarnContext(ctx, "backup: MarkSaved échoué", "key", t.Key, "err", err)
		}
		exported = append(exported, t.Key)
	}

	if len(exported) == 0 {
		slog.InfoContext(ctx, "backup: aucune modification détectée — cycle ignoré",
			"targets", len(targets),
			"duration", time.Since(start).Round(time.Millisecond))
		return &Result{Skipped: true, Duration: time.Since(start)}, nil
	}

	initCtx, initCancel := context.WithTimeout(ctx, 2*time.Minute)
	if err := s.restic.EnsureInit(initCtx); err != nil {
		initCancel()
		return nil, fmt.Errorf("backup: restic init: %w", err)
	}
	initCancel()

	stagingDir := filepath.Join(s.cfg.BackupDir, "staging")
	backupCtx, backupCancel := context.WithTimeout(ctx, 5*time.Minute)
	snapshotID, err := s.restic.Backup(backupCtx, stagingDir)
	backupCancel()
	if err != nil {
		return nil, fmt.Errorf("backup: restic backup: %w", err)
	}

	forgetCtx, forgetCancel := context.WithTimeout(ctx, 2*time.Minute)
	if fErr := s.restic.Forget(forgetCtx); fErr != nil {
		slog.WarnContext(ctx, "backup: restic forget échoué (non-bloquant)", "err", fErr)
	}
	forgetCancel()

	dur := time.Since(start)
	manifest.SetLastResult(snapshotID, exported, dur)
	if err := manifest.Save(); err != nil {
		slog.WarnContext(ctx, "backup: sauvegarde manifest échouée", "err", err)
	}

	slog.InfoContext(ctx, "backup: cycle terminé",
		"snapshot_id", snapshotID,
		"exported", exported,
		"duration", dur.Round(time.Millisecond))

	return &Result{
		SnapshotID: snapshotID,
		Exported:   exported,
		Duration:   dur,
	}, nil
}

// SchedulerStatus is the snapshot returned by Status() for the settings UI.
type SchedulerStatus struct {
	Enabled         bool                       `json:"enabled"`
	Available       bool                       `json:"available"`
	LastBackupAt    string                     `json:"last_backup_at,omitempty"` // RFC3339, empty if never
	LastSnapshotID  string                     `json:"last_snapshot_id,omitempty"`
	LastExported    []string                   `json:"last_exported,omitempty"`
	LastDurationMs  int64                      `json:"last_duration_ms,omitempty"`
	IntegrityChecks map[string]IntegrityResult `json:"integrity_checks,omitempty"`
	Config          struct {
		Interval    string `json:"interval"`
		KeepDaily   int    `json:"keep_daily"`
		KeepWeekly  int    `json:"keep_weekly"`
		KeepMonthly int    `json:"keep_monthly"`
	} `json:"config"`
}

// Status reads the persisted manifest and returns a snapshot for the UI.
// Safe to call at any time; returns sane zero values if no backup has run yet.
func (s *Scheduler) Status() SchedulerStatus {
	st := SchedulerStatus{
		Enabled:   s.cfg.Enabled,
		Available: s.restic.IsAvailable(),
	}
	st.Config.Interval = s.cfg.Interval.String()
	st.Config.KeepDaily = s.cfg.KeepDaily
	st.Config.KeepWeekly = s.cfg.KeepWeekly
	st.Config.KeepMonthly = s.cfg.KeepMonthly

	manifestPath := filepath.Join(s.cfg.BackupDir, ".manifest.json")
	if m, err := LoadManifest(manifestPath); err == nil && !m.LastBackupAt.IsZero() {
		st.LastBackupAt = m.LastBackupAt.UTC().Format(time.RFC3339)
		st.LastSnapshotID = m.LastSnapshotID
		st.LastExported = m.LastExported
		st.LastDurationMs = m.LastDurationMs
		st.IntegrityChecks = m.IntegrityChecks
	}
	return st
}
