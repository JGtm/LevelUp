// Package archlint — no_local_str_coalesce_test.go : ratchet F4 (revue 2026-07-17).
//
// Interdit toute nouvelle coalescence *string→string à 2 pointeurs dans le package
// service (`func <name>(a, b *string) string`). La source unique est
// service.coalesceStr (variadique, premier non-nil/non-vide, "" sinon). Le motif
// avait été dupliqué : `coalesce` (enrich.go) ET `coalesceStr` (briefing.go),
// résultats identiques — leçon CLAUDE.md règle 6.
//
// Portée : uniquement internal/service (là où vivait la duplication et le helper
// canonique). sync.coalesceStrPtr (retourne *string, contrat distinct) et
// sync.resolvedRegistryName (trim + égalité asset_id, autre finalité) sont HORS
// périmètre F4 et volontairement laissés.
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

// `func <ident>(x, y *string) string` — la signature de coalescence 2-args que
// coalesceStr (variadique `...*string`) remplace. Ne matche PAS le variadique ni
// les retours *string (coalesceStrPtr).
var strCoalesceSigRE = regexp.MustCompile(`func \w+\([a-z_]+, [a-z_]+ \*string\) string`)

func TestNoLocalStrCoalesce(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	internalRoot := filepath.Dir(filepath.Dir(thisFile))
	serviceRoot := filepath.Join(internalRoot, "service")

	var violations []string
	err := filepath.WalkDir(serviceRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, _ := filepath.Rel(internalRoot, path)
		rel = filepath.ToSlash(rel)
		for i, line := range strings.Split(string(data), "\n") {
			if strCoalesceSigRE.MatchString(line) {
				violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk service/: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("coalescence *string locale interdite dans service/ (F4) — utiliser "+
			"coalesceStr (variadique) :\n  %s", strings.Join(violations, "\n  "))
	}
}
