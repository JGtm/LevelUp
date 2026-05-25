package duckdbbackup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
		return nil
	}
	return r.run(ctx, "init")
}

// Backup creates a restic snapshot of stagingDir.
// Returns the snapshot ID on success.
func (r *ResticClient) Backup(ctx context.Context, stagingDir string) (string, error) {
	cmd := exec.CommandContext(ctx, r.bin(), "backup", "--json", stagingDir)
	cmd.Env = r.env()
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("restic backup: %w", err)
	}
	return parseSnapshotID(out), nil
}

// Forget applies the configured retention policy and prunes unreferenced data.
func (r *ResticClient) Forget(ctx context.Context) error {
	return r.run(ctx,
		"forget", "--prune",
		fmt.Sprintf("--keep-daily=%d", r.cfg.KeepDaily),
		fmt.Sprintf("--keep-weekly=%d", r.cfg.KeepWeekly),
		fmt.Sprintf("--keep-monthly=%d", r.cfg.KeepMonthly),
	)
}

func (r *ResticClient) run(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, r.bin(), args...)
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
