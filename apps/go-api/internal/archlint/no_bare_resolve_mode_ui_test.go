// Package archlint — garde-fous d'architecture vérifiés en test (ratchet).
//
// no_bare_resolve_mode_ui_test.go : interdit les NOUVEAUX appels nus à
// analysis.ResolveModeUI (dérivation de mode pair-only). Depuis la
// centralisation de la convention « mode = pair, sinon game_variant »
// (analysis.ResolveModeUIWithVariant, fix filtres H5 2026-07-22), un caller qui
// dérive un libellé de mode UI doit passer par le helper WithVariant — un appel
// pair-only rate les titres sans pair_name (Halo 5) et re-crée exactement le
// bug « catégorie Modes vide » (leçon CLAUDE.md n°6 : une factorisation sans
// garde-rail re-diverge).
//
// Les sites historiques restants sont grandfathered dans l'allowlist ci-dessous
// avec leur justification ; toute NOUVELLE occurrence hors allowlist échoue.
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// bareResolveModeUIAllowlist : fichiers (chemin relatif depuis internal/) où un
// appel nu à analysis.ResolveModeUI reste TOLÉRÉ. Grandfathered le 2026-07-22
// (fix filtres H5) : ces sites implémentent déjà leur propre fallback variant
// (deux appels successifs) ou sont légitimement pair-only. Retrait par site à
// mesure de leur migration vers ResolveModeUIWithVariant.
var bareResolveModeUIAllowlist = map[string]bool{
	"sync/recent_matches.go":                true, // :153/:157 deux-étapes pair puis variant (candidat migration)
	"service/session_page_service.go":       true, // :431-443 deux-étapes avec logique EN/FR propre (candidat migration)
	"service/match_view_builders_header.go": true, // :111-118 pair-only assumé — ModeNameFR pré-résolu par le repo en amont
	"platform/duckdb/explorer_repo.go":      true, // :497/:524 résolution repo-side des noms de pair (pas un mode UI de filtre)
}

// bareResolveModeUIRE matche un appel qualifié analysis.ResolveModeUI( — la
// forme WithVariant ne matche pas (frontière : parenthèse ouvrante immédiate).
var bareResolveModeUIRE = regexp.MustCompile(`analysis\.ResolveModeUI\(`)

func TestNoNewBareResolveModeUI(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	internalRoot := filepath.Dir(filepath.Dir(thisFile)) // .../internal

	var violations []string
	err := filepath.WalkDir(internalRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(internalRoot, path)
		rel = filepath.ToSlash(rel)
		if bareResolveModeUIAllowlist[rel] {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for i, line := range strings.Split(string(data), "\n") {
			if bareResolveModeUIRE.MatchString(line) {
				violations = append(violations, rel+":"+strconv.Itoa(i+1)+" → "+strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("appel nu à analysis.ResolveModeUI hors allowlist (%d) — utiliser "+
			"analysis.ResolveModeUIWithVariant (convention mode = pair sinon game_variant, "+
			"sinon la catégorie Modes redevient vide sur les titres sans pair type Halo 5) :\n  %s",
			len(violations), strings.Join(violations, "\n  "))
	}
}
