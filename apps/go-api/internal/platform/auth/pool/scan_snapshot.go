// Package pool — scan_snapshot.go : snapshot en mémoire du dernier Scan de
// Discovery. Expose, par joueur, la source de credentials effectivement
// retenue (watcher_*, duckdb_*, env_oauth, watcher_legacy) pour le dashboard
// admin « Santé des tokens » — sans I/O supplémentaire (le scan a déjà eu lieu
// au boot et à chaque cycle d'auto-sync).
package pool

import (
	"sync"
	"time"
)

// ScanSource décrit la source de credentials retenue pour un joueur lors du
// dernier Scan.
type ScanSource struct {
	Source string    // label composite, ex. "watcher_oauth" ou "duckdb_msal+env_oauth"
	At     time.Time // horodatage du scan
}

var (
	scanSnapshotMu sync.RWMutex
	scanSnapshot   = map[string]ScanSource{} // clé : titleSlug + "|" + gamertag
)

func scanSnapshotKey(titleSlug, gamertag string) string {
	return titleSlug + "|" + gamertag
}

// recordScanSource enregistre la source retenue pour un joueur (appelé par
// Discovery.Scan).
func recordScanSource(titleSlug, gamertag, source string) {
	scanSnapshotMu.Lock()
	scanSnapshot[scanSnapshotKey(titleSlug, gamertag)] = ScanSource{
		Source: source,
		At:     time.Now().UTC(),
	}
	scanSnapshotMu.Unlock()
}

// LastScanSource retourne la source de credentials du dernier Scan pour un
// joueur, ou ("", false) si aucun scan n'a eu lieu depuis le boot.
func LastScanSource(titleSlug, gamertag string) (ScanSource, bool) {
	scanSnapshotMu.RLock()
	defer scanSnapshotMu.RUnlock()
	s, ok := scanSnapshot[scanSnapshotKey(titleSlug, gamertag)]
	return s, ok
}
