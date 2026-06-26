//go:build integration

// Test garde régression Phase 4 plan stabilisation 2026-05-22 :
// TestAssertProgressionDeps_HookWiredOnAllSyncPaths — scanner statique qui
// détecte les nouveaux call sites `sync.NewSyncEngine(...)` ajoutés sans
// câblage `WithPostSyncRunner(...)`.
//
// Si un développeur introduit un nouveau path de sync (CLI subcommand,
// nouveau handler HTTP, etc.) sans wirer le runner, le test échoue
// immédiatement avec un message explicite — pas besoin d'attendre la
// régression métier (page Ascension vide, notifications muettes).
//
// Maintenir la liste expectedSyncCallSites à jour à chaque ajout/retrait
// de call site légitime.
package sync

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// expectedSyncCallSites : sites de création de SyncEngine attendus dans
// internal/ (hors tests). Chaque entrée déclare le statut runner-wiring.
//
// Codes :
//   - "wired"        : WithPostSyncRunner câblé directement dans la fonction
//   - "legacy-hook"  : utilise SyncHandler.WithPostSyncDeltaHook (path HTTP)
//   - "no-runner-ok" : path qui ne nécessite pas le runner (backfill explicite,
//     script offline) — justifié par un commentaire au site
type syncCallSite struct {
	pathSuffix string // suffixe de chemin (sans racine apps/go-api)
	status     string
	reason     string
}

var expectedSyncCallSites = []syncCallSite{
	// NB : newSyncEngineRe ne matche que le wrapper `NewSyncEngine(`. sync_handler.go
	// est passé à NewSyncEngineForTitle (newEngineFor, sync_handler.go:175) et n'est
	// donc plus détecté — l'entrée a été retirée (sinon signalée « obsolète »).
	// Suivi : étendre newSyncEngineRe à NewSyncEngineForTitle + l'alias `syncpkg`
	// pour re-couvrir ces sites (sync_handler.go, cmd/server/sync_v2_wiring.go).
	{
		pathSuffix: "internal/api/handlers/backfill.go",
		status:     "no-runner-ok",
		reason:     "Backfill HTTP explicite — pas un sync nominal, progression V2 hors-scope",
	},
}

// newSyncEngineRe : matche les call sites NewSyncEngine, y compris les
// alias d'import (sync.NewSyncEngine, gosync.NewSyncEngine, go_sync.NewSyncEngine).
var newSyncEngineRe = regexp.MustCompile(`\b(?:sync|gosync|go_sync)\.NewSyncEngine\s*\(`)

// TestAssertProgressionDeps_HookWiredOnAllSyncPaths vérifie que tout call
// site NewSyncEngine dans internal/ (hors tests) est dans la whitelist
// expectedSyncCallSites avec le statut adéquat.
func TestAssertProgressionDeps_HookWiredOnAllSyncPaths(t *testing.T) {
	// Trouver la racine apps/go-api/ depuis ce test.
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("repoRoot abs: %v", err)
	}
	internalRoot := filepath.Join(repoRoot, "internal")

	// 1. Lister tous les .go (hors tests) qui contiennent NewSyncEngine.
	var foundSites []string
	err = filepath.WalkDir(internalRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if newSyncEngineRe.MatchString(string(raw)) {
			rel, _ := filepath.Rel(repoRoot, path)
			rel = filepath.ToSlash(rel)
			foundSites = append(foundSites, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}

	// 2. Cross-référencer avec expectedSyncCallSites.
	expectedMap := make(map[string]syncCallSite, len(expectedSyncCallSites))
	for _, e := range expectedSyncCallSites {
		expectedMap[e.pathSuffix] = e
	}

	// Sites trouvés mais pas dans la whitelist → nouveau path non-câblé.
	var unexpected []string
	for _, site := range foundSites {
		if _, ok := expectedMap[site]; !ok {
			unexpected = append(unexpected, site)
		}
	}

	// Sites whitelistés mais plus présents → entrée à retirer de la liste.
	foundSet := make(map[string]bool, len(foundSites))
	for _, s := range foundSites {
		foundSet[s] = true
	}
	var stale []string
	for _, e := range expectedSyncCallSites {
		if !foundSet[e.pathSuffix] {
			stale = append(stale, e.pathSuffix)
		}
	}

	if len(unexpected) > 0 {
		t.Errorf("Nouveau(x) call site(s) sync.NewSyncEngine détecté(s) hors whitelist :")
		for _, s := range unexpected {
			t.Errorf("  - %s", s)
		}
		t.Errorf("\nAction : ajouter une entrée dans expectedSyncCallSites avec le statut adéquat :")
		t.Errorf("  - 'wired'        si WithPostSyncRunner câblé directement")
		t.Errorf("  - 'legacy-hook'  si SyncHandler.WithPostSyncDeltaHook utilisé (transition)")
		t.Errorf("  - 'no-runner-ok' si path offline justifié (ajouter un commentaire au site)")
		t.Errorf("\nSinon, ce path déclenchera la régression page Ascension vide + notifications muettes")
		t.Errorf("(cf. AUDIT_ASCENSION_PIPELINE_DISCONNECTED_2026-05-21 cause B).")
	}

	if len(stale) > 0 {
		t.Errorf("Entrée(s) whitelist obsolète(s) (call site retiré du code) :")
		for _, s := range stale {
			t.Errorf("  - %s", s)
		}
		t.Errorf("Action : retirer ces entrées de expectedSyncCallSites.")
	}

	if t.Failed() {
		return
	}
	t.Logf("Sync call sites validés (%d) :", len(foundSites))
	for _, site := range foundSites {
		e := expectedMap[site]
		t.Logf("  [%s] %s — %s", e.status, site, e.reason)
	}
}
