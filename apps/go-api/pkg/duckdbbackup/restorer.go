package duckdbbackup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"
)

// SnapshotInfo holds metadata about a restic snapshot.
type SnapshotInfo struct {
	ID       string    // full 64-char hash
	ShortID  string    // 8-char short form displayed by restic
	Time     time.Time // snapshot creation time (local timezone)
	Hostname string
}

// Restorer wraps the restic CLI for snapshot listing and extraction.
type Restorer struct {
	client *ResticClient
}

// NewRestorer creates a Restorer from cfg.
func NewRestorer(cfg Config) *Restorer {
	return &Restorer{client: NewResticClient(cfg)}
}

// ListSnapshots returns all snapshots in the configured repo, newest first.
func (r *Restorer) ListSnapshots(ctx context.Context) ([]SnapshotInfo, error) {
	args := append(r.client.noPasswordFlag(), "snapshots", "--json")
	cmd := exec.CommandContext(ctx, r.client.bin(), args...)
	cmd.Env = r.client.env()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("restic snapshots: %w: %s", err, bytes.TrimSpace(stderr.Bytes()))
	}

	var raw []struct {
		ID       string    `json:"id"`
		ShortID  string    `json:"short_id"`
		Time     time.Time `json:"time"`
		Hostname string    `json:"hostname"`
	}
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("parse snapshots JSON: %w", err)
	}

	snaps := make([]SnapshotInfo, len(raw))
	for i, s := range raw {
		snaps[i] = SnapshotInfo{
			ID:       s.ID,
			ShortID:  s.ShortID,
			Time:     s.Time.Local(),
			Hostname: s.Hostname,
		}
	}
	sort.Slice(snaps, func(i, j int) bool { return snaps[i].Time.After(snaps[j].Time) })
	return snaps, nil
}

// ExtractSnapshot runs restic restore for snapshotID into targetDir.
// Returns the path to the staging/ directory found within the restored tree.
func (r *Restorer) ExtractSnapshot(ctx context.Context, snapshotID, targetDir string) (string, error) {
	slog.InfoContext(ctx, "restore: extraction restic", "snapshot", snapshotID, "target", targetDir)

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", targetDir, err)
	}

	args := append(r.client.noPasswordFlag(), "restore", snapshotID, "--target", targetDir)
	cmd := exec.CommandContext(ctx, r.client.bin(), args...)
	cmd.Env = r.client.env()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	// restic exits non-zero on Windows when it cannot set timestamps on system
	// directories (C:\Users, etc.) that appear as intermediate path components.
	// We check for the staging/ directory independently and treat the error as
	// non-fatal if the payload was actually extracted.

	// The restored tree mirrors the full original absolute path. Walk it to find staging/.
	var stagingDir string
	_ = filepath.WalkDir(targetDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d == nil || !d.IsDir() {
			return nil
		}
		if d.Name() == "staging" {
			stagingDir = path
			return fs.SkipAll
		}
		return nil
	})
	if stagingDir == "" {
		if runErr != nil {
			return "", fmt.Errorf("restic restore: %w: %s", runErr, bytes.TrimSpace(stderr.Bytes()))
		}
		return "", fmt.Errorf("répertoire staging/ introuvable dans le snapshot restauré")
	}
	if runErr != nil {
		slog.WarnContext(ctx, "restore: restic non-zero (probables erreurs de timestamps Windows — ignoré)",
			"err", runErr)
	}

	slog.InfoContext(ctx, "restore: extraction terminée", "staging", stagingDir)
	return stagingDir, nil
}
