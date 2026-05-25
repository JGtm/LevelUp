package duckdbbackup

import (
	"encoding/json"
	"os"
	"time"
)

// Manifest persists per-Target fingerprints of the last successful backup.
// Stored as {BackupDir}/.manifest.json.
type Manifest struct {
	LastBackupAt time.Time              `json:"last_backup_at"`
	Databases    map[string]fingerprint `json:"databases"`
	path         string
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

// Save writes the manifest to disk atomically (write temp + rename).
func (m *Manifest) Save() error {
	m.LastBackupAt = time.Now().UTC()
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
