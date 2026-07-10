package handlers

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// TestNoCapabilityErrorDup — garde-rail Lot B (CLAUDE.md règle #6). Le mapping
// games.ErrCapabilityNotSupported → 503 est centralisé dans MapCapabilityError
// (capability_error.go). Toute copie du littéral `errors.Is(..., games.
// ErrCapabilityNotSupported)` dans un handler re-diverge (messages/logs/compteurs
// incohérents) → interdite. Utiliser MapCapabilityError(ctx, err, probe).
func TestNoCapabilityErrorDup(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	dir := filepath.Dir(thisFile) // internal/api/handlers
	re := regexp.MustCompile(`errors\.Is\([^,]+,\s*games\.ErrCapabilityNotSupported\)`)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var offenders []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || name == "capability_error.go" {
			continue
		}
		src, rerr := os.ReadFile(filepath.Join(dir, name))
		if rerr != nil {
			continue
		}
		if re.Match(src) {
			offenders = append(offenders, name)
		}
	}
	if len(offenders) > 0 {
		t.Errorf("mapping ErrCapabilityNotSupported dupliqué hors capability_error.go — "+
			"remplacer par MapCapabilityError(ctx, err, probe) :\n  %s", strings.Join(offenders, "\n  "))
	}
}
