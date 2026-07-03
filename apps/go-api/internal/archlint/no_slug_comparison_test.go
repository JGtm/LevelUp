// Package archlint — garde-fous d'architecture vérifiés en test (ratchet).
//
// no_slug_comparison_test.go : interdit les NOUVEAUX gating de titre par
// comparaison de slug (`slug == "halo_infinite"`, `!= title.DefaultSlug`,
// `TitleSlug(ctx) == "halo_infinite"`), conformément à ADR 0025 + master
// title-agnostic §7. Tout gating de FEATURE titre doit passer par une capability
// (HasCapability / FeatureChecker), jamais par le slug.
//
// La regex couvre aussi la forme d'appel (`TitleSlug(ctx) == ...`) et tout
// préfixe de package sur DefaultSlug (`titlePkg.DefaultSlug`) — sinon un nouveau
// feature-gate `TitleSlug(ctx) == "halo_infinite"` passait au travers (F10,
// 2026-07-03). Les gardes de PARITÉ de base (défaut = comportement HINF
// byte-identique, autres titres divergent) sont grandfathered dans l'allowlist :
// ce ne sont pas des feature-gates. Toute occurrence hors allowlist échoue.
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

// slugCompareAllowlist : fichiers (chemin relatif depuis internal/) où une
// comparaison de slug au titre par défaut est TOLÉRÉE — gardes de PARITÉ de base
// (défaut = comportement/layout HINF byte-identique ; autres titres divergent),
// PAS des feature-gates. Grandfathered le 2026-07-03 (F10) quand la regex a été
// élargie (préfixe de package + forme d'appel `TitleSlug(ctx)`) : ces sites
// existaient déjà mais échappaient à l'ancienne regex. Le ratchet reste MORDANT
// pour toute NOUVELLE comparaison hors de cette liste. Retrait par site quand la
// garde passe à une capability/mapping TOML (comeback.go:34 suivi en F15-15).
var slugCompareAllowlist = map[string]bool{
	"sync/comeback.go":            true, // :34 medal Steaktacular (→ mapping TOML F15-15) ; :95 dominance HINF byte-identique
	"sync/coordinator.go":         true, // :316 gateKey dedup (défaut = clé nue, parité gate MT-11)
	"api/server.go":               true, // :461 skip HINF déjà chargé (le vrai gate suivant est par capability career.rank_catalog)
	"ops/seed_demo_multititle.go": true, // layout FS démo (défaut = layout legacy byte-identique)
	"ops/seed_demo.go":            true, // :391 extraction média démo du titre par défaut
}

// slugCompareRE matche un gating de titre : une variable/appel *Slug (ou le
// littéral "halo_infinite") comparé par == / != à "halo_infinite" ou à
// <pkg>.DefaultSlug. Couvre la forme d'appel `TitleSlug(ctx) == ...` et tout
// préfixe de package (`titlePkg.DefaultSlug`).
var slugCompareRE = regexp.MustCompile(
	`(?:[Ss]lug(?:\([^)]*\))?\s*[!=]=\s*(?:"halo_infinite"|(?:\w+\.)?DefaultSlug))` +
		`|(?:"halo_infinite"\s*[!=]=)`)

func TestNoNewSlugComparison(t *testing.T) {
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
		if slugCompareAllowlist[rel] {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for i, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
				continue // ligne de commentaire : explication tolérée
			}
			if slugCompareRE.MatchString(line) {
				violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+trimmed)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk internal/: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("gating de titre par slug interdit (ADR 0025) — passer par une capability "+
			"(HasCapability / FeatureChecker) ou allowlister un hard-gate transitoire :\n  %s",
			strings.Join(violations, "\n  "))
	}
}

// TestSlugCompareRE_CatchesBroadenedForms verrouille l'élargissement F10 : la
// regex DOIT attraper la forme d'appel `TitleSlug(ctx) == "halo_infinite"` (le
// feature-gate qui passait au travers avant F10) et le préfixe de package sur
// DefaultSlug. Elle NE DOIT PAS attraper une égalité slug↔slug légitime.
func TestSlugCompareRE_CatchesBroadenedForms(t *testing.T) {
	mustMatch := []string{
		`if ctxkeys.TitleSlug(ctx) == "halo_infinite" {`,
		`if ctxkeys.TitleSlug(ctx) == titlePkg.DefaultSlug {`,
		`if slug == titlePkg.DefaultSlug {`,
		`if td.Slug != titlePkg.DefaultSlug {`,
	}
	for _, s := range mustMatch {
		if !slugCompareRE.MatchString(s) {
			t.Errorf("regex devrait matcher (gate slug) : %q", s)
		}
	}
	mustNotMatch := []string{
		`return s.rankCatalog.TitleSlug() == s.titleSlug`, // slug↔slug, pas un gate défaut
		`titleSlug := ctxkeys.TitleSlug(ctx)`,             // affectation
	}
	for _, s := range mustNotMatch {
		if slugCompareRE.MatchString(s) {
			t.Errorf("regex ne devrait PAS matcher : %q", s)
		}
	}
}
