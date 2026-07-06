// Package archlint — sync_root_freeze_test.go : ratchet de gel du god-package sync/ (K3c).
//
// internal/sync/ est un god-package (112 fichiers .go racine hors _test.go au 2026-07-06,
// ARCHI 19 de l'audit). ADR 0027 : le neuf va dans internal/sync/v2/ (cycle orchestrator)
// ou dans un sous-package cohésif (sync/skill/, sync/snapshot/), JAMAIS un nouveau fichier
// à la racine. Ce ratchet GÈLE le compte : il ne doit que DÉCROÎTRE au fil des extractions.
// Abaisser syncRootFileBaseline à chaque extraction ; ne JAMAIS l'augmenter sans
// justification datée (l'augmenter = accroître la dette god-package, interdit CLAUDE.md).
package archlint

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// syncRootFileBaseline : plafond GELÉ de fichiers .go (hors _test.go) à la racine de
// internal/sync/. Décroît uniquement (K3c / ADR 0027).
// 112 → 106 (2026-07-06) : cluster snapshot (6 fichiers) extrait vers internal/sync/snapshot.
const syncRootFileBaseline = 106

func TestSyncRootPackageFrozen(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	internalRoot := filepath.Dir(filepath.Dir(thisFile))
	syncRoot := filepath.Join(internalRoot, "sync")
	entries, err := os.ReadDir(syncRoot)
	if err != nil {
		t.Fatalf("ReadDir sync/: %v", err)
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			count++
		}
	}
	if count > syncRootFileBaseline {
		t.Errorf("god-package internal/sync/ : %d fichiers racine (.go non-test) > baseline gelée %d.\n"+
			"Le neuf va dans sync/v2/ (ADR 0027) ou un sous-package cohésif (sync/skill/, sync/snapshot/), "+
			"jamais un nouveau fichier racine. Cf. K3c.", count, syncRootFileBaseline)
	}
	if count < syncRootFileBaseline {
		t.Errorf("baseline sync/ obsolète : %d fichiers racine < baseline %d — ABAISSER "+
			"syncRootFileBaseline à %d dans le même commit (le ratchet ne fait que décroître).",
			count, syncRootFileBaseline, count)
	}
}
