package config

// demo_state_guard_test.go — garde-rail : le paquet config ne lit jamais la valeur brute de
// DemoMode pour choisir entre fixture démo et données réelles. Cette décision passe par
// demoState() (config_demo.go), qui distingue le serveur démo d'un binaire opérateur chargé
// par Load() sous LEVELUP_DEMO_MODE=true (lot L8-3 du plan sécurité du 2026-10-15 : six lectures
// brutes servaient la fixture à `levelup recompute-friends`).

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// demoModeReadAllowlist : les seules lectures brutes admises, avec leur raison.
//
//   - config.go (1) : Validate lit la valeur DÉCLARÉE du mode (garde-fou de production,
//     SecurityWarnings), pas un choix de source de données ; le mode déclaré est celui de
//     l'environnement, y compris pour un binaire opérateur (2026-10-16).
var demoModeReadAllowlist = map[string]int{
	"config.go": 1,
}

var demoModeRead = regexp.MustCompile(`\.DemoMode\b`)

func TestDemoMode_NeSeLitQuePar_demoState(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du paquet : %v", err)
	}
	seen := map[string]int{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || name == "config_demo.go" {
			continue
		}
		src, err := os.ReadFile(filepath.Clean(name))
		if err != nil {
			t.Fatalf("lecture de %s : %v", name, err)
		}
		for _, line := range strings.Split(string(src), "\n") {
			code := line
			if i := strings.Index(code, "//"); i >= 0 {
				code = code[:i]
			}
			seen[name] += len(demoModeRead.FindAllString(code, -1))
		}
	}
	for name, n := range seen {
		if n != demoModeReadAllowlist[name] {
			t.Errorf("%s : %d lecture(s) de .DemoMode, %d admise(s) — utiliser cfg.demoState() "+
				"(sauf lecture du mode déclaré, à allowlister avec sa raison datée)", name, n, demoModeReadAllowlist[name])
		}
	}
	for name, want := range demoModeReadAllowlist {
		if seen[name] != want {
			t.Errorf("allowlist périmée : %s déclare %d lecture(s), %d trouvée(s)", name, want, seen[name])
		}
	}
}

// Preuve que le garde-rail détecte : la regex reconnaît la lecture brute et pas demoState.
func TestDemoModeRead_Regex(t *testing.T) {
	if !demoModeRead.MatchString("if cfg.DemoMode {") || demoModeRead.MatchString("if cfg.demoState() {") {
		t.Fatal("le motif ne distingue pas la lecture brute de demoState()")
	}
}
