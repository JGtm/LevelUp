package duckdbbackup

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Scheduler runs on-demand DuckDB backups via restic.
// Instantiate with New; call RunOnce for a single cycle. Periodic scheduling is
// intentionally NOT provided in-app: backups are planned externally (systemd
// timers, cf. scripts/systemd/levelup-restic-backup.timer).
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

// RunOnce executes a single backup cycle synchronously.
// This is the sole entry point: invoked by the manual trigger
// (POST /settings/backup/run) and the backup-once CLI. Periodic scheduling is
// external (systemd), not embedded in the app.
func (s *Scheduler) RunOnce(ctx context.Context) (*Result, error) {
	return s.cycle(ctx)
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
	var integrityChecked bool
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
		integrityChecked = true
		if !ic.OK {
			slog.WarnContext(ctx, "backup: intégrité DB dégradée (sauvegarde maintenue)",
				"key", t.Key, "detail", ic.Detail)
		}

		// ":" is the namespace separator in keys but is forbidden in Windows
		// directory names; replace with "/" so filepath.Join creates sub-dirs.
		stagingRelPath := filepath.FromSlash(strings.ReplaceAll(t.Key, ":", "/"))
		outDir := filepath.Join(s.cfg.BackupDir, "staging", stagingRelPath)
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
		// Persist integrity results even when no export succeeded — so the UI can
		// surface warnings even if restic never ran (e.g. every export failed).
		if integrityChecked {
			if err := manifest.SaveIntegrityOnly(); err != nil {
				slog.WarnContext(ctx, "backup: sauvegarde résultats intégrité échouée", "err", err)
			}
		}
		slog.InfoContext(ctx, "backup: aucune modification détectée — cycle ignoré",
			"targets", len(targets),
			"duration", time.Since(start).Round(time.Millisecond))
		return &Result{Skipped: true, DurationMs: time.Since(start).Milliseconds()}, nil
	}

	// Libère les verrous restic périmés d'un éventuel process précédent SIGKILL'd
	// (Air hot-reload Windows). restic unlock est un no-op si le dépôt n'est pas
	// verrouillé — sans ça, "restic backup" et "restic forget" échouent avec
	// exit 11 "repository is already locked by PID <orphelin>".
	unlockCtx, unlockCancel := context.WithTimeout(ctx, 30*time.Second)
	if ulErr := s.restic.Unlock(unlockCtx); ulErr != nil {
		slog.WarnContext(ctx, "backup: restic unlock (pre-cycle) non-bloquant", "err", ulErr)
	}
	unlockCancel()

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
		DurationMs: dur.Milliseconds(),
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
