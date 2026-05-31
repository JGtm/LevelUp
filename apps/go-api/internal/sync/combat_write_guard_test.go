//go:build cgo

// Package sync — combat_write_guard_test.go : garde-fou anti-régression.
//
// Interdit toute écriture SQL directe (INSERT/DELETE/UPDATE) sur les tables
// combat de complétion (highlight_events, killer_victim_pairs) dans le package
// sync hors allowlist. Ces écritures DOIVENT passer par le package persist
// (persist.EventsCompletionPersister) sur le writer RW.
//
// Cause racine gardée : la complétion post-sync écrivait highlight_events /
// killer_victim_pairs en db.Exec direct sur un handle non garanti RW → 31h de
// panne silencieuse (graphiques onglet Détails vides). Cf.
// .ai/HANDOFF_sync_combat_completion.md. Modèle : no_art_patterns_test.go.
package sync

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// combatWritePatterns matche une écriture SQL ciblant une table combat de complétion.
var combatWritePatterns = map[string]*regexp.Regexp{
	"highlight_events":    regexp.MustCompile(`(?is)(INSERT[^;]*?INTO\s+highlight_events|DELETE\s+FROM\s+highlight_events|UPDATE\s+highlight_events\b)`),
	"killer_victim_pairs": regexp.MustCompile(`(?is)(INSERT[^;]*?INTO\s+killer_victim_pairs|DELETE\s+FROM\s+killer_victim_pairs|UPDATE\s+killer_victim_pairs\b)`),
}

// allowedCombatWriteFiles : sites legacy encore tolérés, par table → fichier →
// raison. killer_victim_pairs : allowlist VIDE (entièrement migré vers persist).
// highlight_events : writes.go reste toléré tant que InsertHighlightEvents est
// consommé par le service openspartan_import (à migrer dans un lot dédié).
var allowedCombatWriteFiles = map[string]map[string]string{
	"highlight_events": {
		"writes.go": "InsertHighlightEvents (INSERT OR IGNORE) encore consommé par service/openspartan_import — migrer vers persist dans un lot dédié",
	},
	"killer_victim_pairs": {},
}

// TestNoDirectCombatWritesOutsidePersist échoue si une écriture directe sur
// highlight_events ou killer_victim_pairs réapparaît dans le package sync hors
// allowlist. La complétion combat doit passer par persist.EventsCompletionPersister.
func TestNoDirectCombatWritesOutsidePersist(t *testing.T) {
	err := filepath.Walk(".", func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		base := filepath.Base(path)
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for table, re := range combatWritePatterns {
			if re.Match(content) {
				if _, ok := allowedCombatWriteFiles[table][base]; !ok {
					t.Errorf("écriture directe sur %s dans %s (hors allowlist) — router via persist.EventsCompletionPersister", table, base)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
}
