package duckdbbackup

import (
	"encoding/json"
	"os"
	"time"
)

// Manifest persists per-Target fingerprints of the last successful backup.
// Stored as {BackupDir}/.manifest.json.
type Manifest struct {
	LastBackupAt    time.Time                  `json:"last_backup_at"`
	LastSnapshotID  string                     `json:"last_snapshot_id,omitempty"`
	LastExported    []string                   `json:"last_exported,omitempty"`
	LastDurationMs  int64                      `json:"last_duration_ms,omitempty"`
	IntegrityChecks map[string]IntegrityResult `json:"integrity_checks,omitempty"`
	Databases       map[string]fingerprint     `json:"databases"`
	path            string
}

type fingerprint struct {
	Path           string    `json:"path"`
	Mtime          time.Time `json:"mtime"`
	SizeBytes      int64     `json:"size_bytes"`
	LastBackedUpAt time.Time `json:"last_backed_up_at"`
}

// LoadManifest reads the manifest from path; returns an empty one if absent.
func LoadManifest(path string) (*Manifest, error) {
	m := &Manifest{
		Databases: make(map[string]fingerprint),
		path:      path,
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return m, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, m); err != nil {
		return nil, err
	}
	m.path = path
	return m, nil
}

// HasChanged reports whether t's file has changed since the last recorded backup.
// Returns true if the target has never been backed up.
func (m *Manifest) HasChanged(t Target) (bool, error) {
	fi, err := os.Stat(t.Path)
	if err != nil {
		return false, err
	}
	prev, ok := m.Databases[t.Key]
	if !ok {
		return true, nil
	}
	return !fi.ModTime().Equal(prev.Mtime) || fi.Size() != prev.SizeBytes, nil
}

// MarkSaved records the current fingerprint of t in the in-memory manifest.
// Call Save() afterwards to persist to disk.
func (m *Manifest) MarkSaved(t Target) error {
	fi, err := os.Stat(t.Path)
	if err != nil {
		return err
	}
	m.Databases[t.Key] = fingerprint{
		Path:           t.Path,
		Mtime:          fi.ModTime(),
		SizeBytes:      fi.Size(),
		LastBackedUpAt: time.Now().UTC(),
	}
	return nil
}

// SetLastResult stores the outcome of the last successful backup cycle.
// Must be called before Save() so the values are persisted.
func (m *Manifest) SetLastResult(snapshotID string, exported []string, duration time.Duration) {
	m.LastSnapshotID = snapshotID
	m.LastExported = exported
	m.LastDurationMs = duration.Milliseconds()
}

// SetIntegrityResult stores the integrity check result for a DB key.
// Must be called before Save() so the value is persisted.
func (m *Manifest) SetIntegrityResult(key string, res IntegrityResult) {
	if m.IntegrityChecks == nil {
		m.IntegrityChecks = make(map[string]IntegrityResult)
	}
	m.IntegrityChecks[key] = res
}

// Save writes the manifest to disk atomically and stamps LastBackupAt = now.
// Call this after a successful restic snapshot.
func (m *Manifest) Save() error {
	m.LastBackupAt = time.Now().UTC()
	return m.writeAtomic()
}

// SaveIntegrityOnly persists integrity check results without touching LastBackupAt.
// Call this when a cycle ran integrity checks but no exports completed
// (e.g. all exports failed), so the UI can still surface integrity warnings.
func (m *Manifest) SaveIntegrityOnly() error {
	return m.writeAtomic()
}

func (m *Manifest) writeAtomic() error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	tmp := m.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, m.path)
}
