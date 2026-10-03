// Package games — capability_loader_guard_test.go : garde-rail de la factorisation
// de LoadCapabilityMap (regle des <= 2 copies, CLAUDE.md n°6).
//
// La recette « registre ephemere -> GetCapabilities -> CapabilityMapFromMappings »
// existait en TROIS copies identiques (killcollector/classifier.go,
// killcollector/postsync.go, cmd/levelup/cmd_backfill_killsource.go) quand le resume
// d'usage de session (2026-09-04) en a demande une quatrieme : centralisation dans
// games.LoadCapabilityMap, copies migrees. Ce test interdit la re-divergence — une
// factorisation sans garde-rail re-diverge (lecon documentee : le predicat bot est
// passe de 8 a 36 copies APRES sa centralisation).
//
// LE LITTERAL SENTINELLE est le message d'erreur du cas « fichier absent », present
// dans chaque copie d'origine et desormais uniquement dans le helper : le retrouver
// hors de internal/games signale une copie recrite a la main.

package games

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoHandWrittenCapabilityLoaderCopies echoue si le litteral sentinelle de la
// recette de chargement reapparait hors de internal/games (tests exclus : un test
// peut citer le message pour l'asserter).
func TestNoHandWrittenCapabilityLoaderCopies(t *testing.T) {
	const sentinelle = "capabilities.toml absent pour le titre"

	// Racine du module go-api : ce fichier vit dans internal/games/.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	moduleRoot := filepath.Dir(filepath.Dir(wd)) // internal/games -> internal -> go-api

	var violations []string
	err = filepath.Walk(moduleRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			name := info.Name()
			if name == "vendor" || name == ".git" || name == "node_modules" || name == "data" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, relErr := filepath.Rel(moduleRoot, path)
		if relErr != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "internal/games/") {
			return nil // le foyer canonique a le droit de porter le litteral
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(raw), sentinelle) {
			violations = append(violations, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("recette de chargement des capabilities recrite a la main (utiliser "+
			"games.LoadCapabilityMap) dans :\n  %s", strings.Join(violations, "\n  "))
	}
}
