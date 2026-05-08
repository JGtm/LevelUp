// Package sync — bitmask_dead_declarations_test.go : garde-fou contre la
// ré-introduction de constantes bitmask orphelines.
//
// Phase 3 du plan PLAN_BITMASKS_AUDIT_FIX (mai 2026).
//
// Le test extrait toutes les constantes `MBit*`, `PBit*`, `PveBit*` déclarées
// dans `backfill_flags.go` puis vérifie que chacune est référencée au moins
// une fois dans un autre fichier `.go` du package sync (write SQL, condition
// SQL, ou test d'invariant). Si une constante n'est référencée nulle part en
// dehors de sa déclaration, le test fail avec un message qui suggère soit
// l'utilisation soit la suppression.
//
// Justification : sans ce garde-fou, l'audit Phase 1ter de
// PLAN_HIGHLIGHT_EVENTS_BACKFILL avait dû découvrir manuellement 17 bits
// orphelins. Le test attrape la prochaine régression en CI.
package sync

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestNoDeadBitDeclaration(t *testing.T) {
	declared, err := extractBitConstants("backfill_flags.go")
	if err != nil {
		t.Fatalf("extractBitConstants: %v", err)
	}
	if len(declared) == 0 {
		t.Fatal("aucune constante bit extraite — regex cassée ?")
	}

	// Lire tous les .go du package SAUF backfill_flags.go (déclaration)
	// et SAUF ce fichier (qui mentionne nommément certains bits dans des
	// strings de doc/erreur).
	others, err := loadOtherSourceFiles([]string{
		"backfill_flags.go",
		"bitmask_dead_declarations_test.go",
	})
	if err != nil {
		t.Fatalf("loadOtherSourceFiles: %v", err)
	}

	// Whitelist : constantes dont l'absence d'utilisation est INTENTIONNELLE
	// (ex. groupes pré-calculés laissés pour l'API publique). Justifier
	// chaque entrée. Si une whitelist devient obsolète, le test l'indique.
	whitelist := map[string]string{
		"PBitMMR":          "groupe public PBitTeamMMR|PBitEnemyMMR — utile pour callers externes",
		"PBitExpected":     "groupe public PBitKillsExp|PBitDeathsExp",
		"PBitSkill":        "groupe public PBitMMR|PBitExpected",
		"PBitCombat":       "groupe public PBitAccuracy|PBitShots|PBitDamage",
		"PBitKillsDetail":  "groupe public",
		"PBitCoreStats":    "groupe public",
		"PBitAllStats":     "groupe public PBitSkill|PBitCoreStats|PBitMedals|PBitKillerVictim",
		"PveBitAllEnemies": "groupe public sommant tous les enemy types",
		"PveBitFullPVE":    "groupe public PveBitTotalKills|PveBitBossKills|PveBitAllEnemies",
	}

	var dead []string
	for _, name := range declared {
		if _, ok := whitelist[name]; ok {
			continue
		}
		used := false
		for _, body := range others {
			// On cherche le nom comme identifier (entouré de non-word chars).
			// Regex simple : bordure de mot.
			pat := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`)
			if pat.MatchString(body) {
				used = true
				break
			}
		}
		if !used {
			dead = append(dead, name)
		}
	}

	if len(dead) > 0 {
		t.Errorf("constantes bitmask déclarées mais jamais utilisées (orphelines) : %v\n"+
			"  → soit ajouter un call-site WRITE/READ (Phase 2 du plan PLAN_BITMASKS_AUDIT_FIX)\n"+
			"  → soit supprimer la constante (Phase 3)\n"+
			"  → soit ajouter à la whitelist du test avec une justification écrite",
			dead)
	}
}

// extractBitConstants parse le fichier `backfill_flags.go` pour récupérer
// toutes les constantes nommées `MBit*`, `PBit*`, `PveBit*`. Approche regex
// plutôt qu'AST pour rester ultra-simple : la convention du fichier garantit
// `Name = expr` ligne-à-ligne dans des blocs `const ()`.
func extractBitConstants(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	re := regexp.MustCompile(`(?m)^\s*((?:MBit|PBit|PveBit)\w+)\s*=`)
	matches := re.FindAllStringSubmatch(string(data), -1)
	out := make([]string, 0, len(matches))
	seen := map[string]struct{}{}
	for _, m := range matches {
		name := m[1]
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out, nil
}

// loadOtherSourceFiles retourne le contenu de tous les .go du package
// (working dir = `internal/sync/`) sauf ceux listés dans `excludes`.
func loadOtherSourceFiles(excludes []string) (map[string]string, error) {
	excludeSet := map[string]struct{}{}
	for _, e := range excludes {
		excludeSet[e] = struct{}{}
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		if _, skip := excludeSet[e.Name()]; skip {
			continue
		}
		data, err := os.ReadFile(filepath.Join(".", e.Name()))
		if err != nil {
			return nil, err
		}
		out[e.Name()] = string(data)
	}
	return out, nil
}
