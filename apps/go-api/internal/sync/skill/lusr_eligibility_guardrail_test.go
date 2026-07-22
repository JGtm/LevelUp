// Package skill — lusr_eligibility_guardrail_test.go : garde-rail anti-dérive du
// prédicat d'éligibilité LUSR (CLAUDE.md n°6 « ≤ 2 copies + garde-rail »).
//
// L'éligibilité d'un match au calcul LUSR est définie par TROIS maillons
// mono-source :
//   - le filtre SQL de sélection (is_ranked=FALSE, is_firefight=FALSE,
//     duration >= 30) — dans loadShadowMatches (v2) et loadLUSRMatchData (v1) ;
//   - la résolution de chaîne GetLUSRChainForTitle ;
//   - le prédicat rosters/équilibre/outcome classifyLUSREligibility.
//
// Le scoreur (processOneShadowMatch) ET le détecteur de trous (ScanLUSRGaps)
// consomment ces maillons ; ils ne doivent JAMAIS ré-écrire le littéral du
// filtre SQL en 3e endroit — sinon le détecteur re-diverge du scoreur
// (over/under-count des trous). Ce test échoue si la combinaison des 3 littéraux
// de filtre apparaît hors de l'allowlist ci-dessous.
package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoDuplicateLUSREligibilityFilter(t *testing.T) {
	// Les deux SEULS fichiers autorisés à porter le littéral du filtre LUSR :
	// le chargeur v1 (loadLUSRMatchData) et le chargeur v2 (loadShadowMatches).
	// Toute 3e occurrence = copie divergente → factoriser via loadShadowMatches.
	allowed := map[string]bool{
		"skill_rating_loaders.go": true, // loadLUSRMatchData (v1)
		"skill_v2_shadow.go":      true, // loadShadowMatches (v2)
	}
	// Marqueurs qui, réunis dans un même fichier, signent une copie du filtre
	// d'éligibilité LUSR (et pas un simple usage isolé de is_ranked ailleurs).
	markers := []string{"is_ranked", "is_firefight", "duration_seconds >= 30"}

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du répertoire package: %v", err)
	}
	scanned := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		if allowed[name] {
			continue
		}
		data, err := os.ReadFile(filepath.Clean(name))
		if err != nil {
			t.Fatalf("lecture %s: %v", name, err)
		}
		scanned++
		content := string(data)
		hasAll := true
		for _, mk := range markers {
			if !strings.Contains(content, mk) {
				hasAll = false
				break
			}
		}
		if hasAll {
			t.Errorf("%s ré-écrit le filtre d'éligibilité LUSR (is_ranked + is_firefight + "+
				"duration_seconds >= 30) — 3e copie interdite. Réutiliser loadShadowMatches "+
				"+ classifyLUSREligibility au lieu de dupliquer le SQL (CLAUDE.md n°6).", name)
		}
	}
	if scanned == 0 {
		t.Fatal("aucun fichier source scanné hors allowlist — le garde-rail ne protège rien")
	}
}
