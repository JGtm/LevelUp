// Package skill — lusr_watermark_guardrail_test.go : le « déjà traité » LUSR v2 (lot
// perf L6, 2026-09-23) n'a qu'un prédicat, lusrWatermarkCovers, pour ses trois
// consommateurs : le scoreur sous l'écrivain (processOneShadowMatch), le pré-filtre
// sous le lecteur (selectShadowWorkUnderRead) et le détecteur de trous
// (ScanLUSRGaps). CLAUDE.md n°6 : à la 3e copie, un helper ET un garde-rail — une
// copie qui passerait de « ≤ » à « < » rejouerait (ou sauterait) le match posé pile
// sur le filigrane, et le pré-filtre divergerait du scoreur.
package skill

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestLUSRWatermarkCovers_Boundary(t *testing.T) {
	wm := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name  string
		wm    *time.Time
		start time.Time
		want  bool
	}{
		{"groupe jamais scoré", nil, wm, false},
		{"avant le filigrane", &wm, wm.Add(-time.Minute), true},
		{"pile sur le filigrane : déjà traité", &wm, wm, true},
		{"après le filigrane", &wm, wm.Add(time.Microsecond), false},
	}
	for _, tc := range cases {
		if got := lusrWatermarkCovers(tc.wm, tc.start); got != tc.want {
			t.Errorf("%s : lusrWatermarkCovers = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestNoDuplicateLUSRWatermarkPredicate(t *testing.T) {
	// Le SEUL fichier autorisé à comparer un début de match à un filigrane.
	const helperFile = "skill_v2_watermark.go"
	// Comparaison temporelle sur un début de match ou un filigrane : signature d'une
	// copie du prédicat (les deux copies retirées lisaient
	// `!m.startTime.After(*st.LastMatchAt)` et `!m.startTime.After(*wm)`).
	copyRe := regexp.MustCompile(`(?i)\b(start_?time|last_?match_?at)\)?\.(After|Before|Equal|Compare)\(`)

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du répertoire package: %v", err)
	}
	scanned := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || name == helperFile {
			continue
		}
		data, err := os.ReadFile(filepath.Clean(name))
		if err != nil {
			t.Fatalf("lecture %s: %v", name, err)
		}
		scanned++
		for i, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}
			if copyRe.MatchString(line) {
				t.Errorf("%s:%d recopie le prédicat du filigrane LUSR (%s) — utiliser lusrWatermarkCovers (%s).",
					name, i+1, strings.TrimSpace(line), helperFile)
			}
		}
	}
	if scanned == 0 {
		t.Fatal("aucun fichier source scanné hors allowlist — le garde-rail ne protège rien")
	}
}
