package duckdbbackup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
)

// ResticClient wraps the restic CLI binary.
type ResticClient struct {
	cfg Config
}

// NewResticClient creates a ResticClient from cfg.
func NewResticClient(cfg Config) *ResticClient {
	return &ResticClient{cfg: cfg}
}

// IsAvailable reports whether the restic binary is reachable in PATH.
func (r *ResticClient) IsAvailable() bool {
	_, err := exec.LookPath(r.bin())
	return err == nil
}

// EnsureInit initialises the restic repository if not already done.
// Idempotent: safe to call even when the repo already exists.
func (r *ResticClient) EnsureInit(ctx context.Context) error {
	// `restic snapshots` exits 0 if the repo is initialised, non-0 otherwise.
	if err := r.run(ctx, "snapshots", "--quiet"); err == nil {
		slog.DebugContext(ctx, "backup: repo restic déjà initialisé", "repo", r.cfg.ResticRepo)
		return nil
	}
	slog.InfoContext(ctx, "backup: initialisation repo restic", "repo", r.cfg.ResticRepo)
	return r.run(ctx, "init")
}

// Backup creates a restic snapshot of stagingDir.
// Returns the snapshot ID on success.
func (r *ResticClient) Backup(ctx context.Context, stagingDir string) (string, error) {
	slog.DebugContext(ctx, "backup: lancement restic backup", "dir", stagingDir)
	args := append([]string{"backup", "--json"}, r.noPasswordFlag()...)
	args = append(args, stagingDir)
	cmd := exec.CommandContext(ctx, r.bin(), args...)
	cmd.Env = r.env()
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("restic backup: %w", err)
	}
	snapshotID := parseSnapshotID(out)
	slog.InfoContext(ctx, "backup: snapshot restic créé", "snapshot_id", snapshotID)
	return snapshotID, nil
}

// Forget applies the configured retention policy and prunes unreferenced data.
func (r *ResticClient) Forget(ctx context.Context) error {
	slog.DebugContext(ctx, "backup: restic forget — nettoyage anciens snapshots",
		"keep_daily", r.cfg.KeepDaily,
		"keep_weekly", r.cfg.KeepWeekly,
		"keep_monthly", r.cfg.KeepMonthly)
	return r.run(ctx,
		"forget", "--prune",
		fmt.Sprintf("--keep-daily=%d", r.cfg.KeepDaily),
		fmt.Sprintf("--keep-weekly=%d", r.cfg.KeepWeekly),
		fmt.Sprintf("--keep-monthly=%d", r.cfg.KeepMonthly),
	)
}

func (r *ResticClient) run(ctx context.Context, args ...string) error {
	all := append(r.noPasswordFlag(), args...)
	cmd := exec.CommandContext(ctx, r.bin(), all...)
	cmd.Env = r.env()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return err
		}
		return fmt.Errorf("%w: %s", err, msg)
	}
	return nil
}

func (r *ResticClient) bin() string {
	if r.cfg.ResticBin != "" {
		return r.cfg.ResticBin
	}
	return "restic"
}

// noPasswordFlag returns ["--insecure-no-password"] when neither ResticPassword
// nor ResticPwdFile is configured. Restic requires this explicit flag to confirm
// that the absence of a password is intentional (supported since restic 0.17).
func (r *ResticClient) noPasswordFlag() []string {
	if r.cfg.ResticPassword == "" && r.cfg.ResticPwdFile == "" {
		return []string{"--insecure-no-password"}
	}
	return nil
}

// env merges the process environment with restic-specific overrides.
// Using a dedicated env slice avoids polluting the parent process environment.
func (r *ResticClient) env() []string {
	base := os.Environ()
	extra := make([]string, 0, 3)
	if r.cfg.ResticRepo != "" {
		extra = append(extra, "RESTIC_REPOSITORY="+r.cfg.ResticRepo)
	}
	if r.cfg.ResticPassword != "" {
		extra = append(extra, "RESTIC_PASSWORD="+r.cfg.ResticPassword)
	}
	if r.cfg.ResticPwdFile != "" {
		extra = append(extra, "RESTIC_PASSWORD_FILE="+r.cfg.ResticPwdFile)
	}
	return append(base, extra...)
}

// parseSnapshotID extracts the snapshot_id from `restic backup --json` output.
// The output contains multiple JSON objects (one per line); the summary is last.
func parseSnapshotID(out []byte) string {
	lines := bytes.Split(bytes.TrimSpace(out), []byte("\n"))
	for i := len(lines) - 1; i >= 0; i-- {
		var obj map[string]any
		if err := json.Unmarshal(lines[i], &obj); err != nil {
			continue
		}
		if obj["message_type"] == "summary" {
			if id, ok := obj["snapshot_id"].(string); ok {
				return id
			}
		}
	}
	return ""
}
