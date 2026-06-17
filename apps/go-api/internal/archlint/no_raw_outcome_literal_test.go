// Package archlint — no_raw_outcome_literal_test.go : ratchet PMT-5 Contract.
//
// Interdit les NOUVELLES comparaisons Go d'outcome sur un littéral brut Halo
// (`Outcome == 2`, `outcome != 4`, …) hors du seam canonique. Les codes 2/3/1/4
// sont l'encodage RAW d'Halo ; le code applicatif doit comparer aux constantes
// `domain.Outcome*` (équivalentes en valeur, nommées) ou router via le seam
// `mappings.OutcomeMappingSet` pour les titres non-Halo.
//
// Allowlist DÉCROISSANTE : les fichiers encore non migrés (littéral dans une
// chaîne SQL ou fonction sans seam injecté — cf. PMT-5 §2, threading requis) y
// figurent transitoirement. Chaque migration RETIRE une entrée jusqu'à vide.
// Toute occurrence HORS allowlist (donc nouvelle, ou régression d'un fichier déjà
// migré) fait échouer le test.
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

// rawOutcomeAllowlist : fichiers (relatifs depuis internal/) où une comparaison
// d'outcome sur littéral brut est tolérée transitoirement (PMT-5 Contract non
// terminé pour ces sites — littéral SQL et/ou pas de seam injecté). À décrémenter.
var rawOutcomeAllowlist = map[string]bool{
	// SEUL site restant : littéral `outcome != 4` à l'INTÉRIEUR d'une chaîne SQL
	// (filtre DNF title-agnostic). Indé-magickable par const Go ; la résolution
	// propre est le fragment SQL du seam `mappings.OutcomeMappingSet` (title-aware),
	// threading différé tant qu'aucun 2e titre n'existe. Toutes les comparaisons Go
	// (analysis/patterns, service/*) ont été migrées vers `domain.Outcome*`.
	"sync/assists_model.go": true,
}

// rawOutcomeRE matche une comparaison Go `Outcome|outcome [==|!=] <1..4>` — soit
// un littéral brut Halo comparé directement. Ne matche PAS les définitions
// `OutcomeWin = 2` (assignation simple `=`, pas `==`/`!=`) ni les littéraux SQL
// `outcome = 2` (un seul `=`).
var rawOutcomeRE = regexp.MustCompile(`[Oo]utcome\s*[!=]=\s*[1-4]\b`)

func TestNoNewRawOutcomeLiteral(t *testing.T) {
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
		// domain/ = définition source des constantes ; mappings/ = le seam canonique.
		if strings.HasPrefix(rel, "domain/") || strings.HasPrefix(rel, "games/mappings/") {
			return nil
		}
		if rawOutcomeAllowlist[rel] {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for i, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
				continue
			}
			if rawOutcomeRE.MatchString(line) {
				violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+trimmed)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk internal/: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("comparaison d'outcome sur littéral brut Halo interdite (PMT-5 Contract) — "+
			"utiliser les constantes domain.Outcome* (Go) ou le seam mappings.OutcomeMappingSet "+
			"(title-aware), ou allowlister transitoirement si threading requis :\n  %s",
			strings.Join(violations, "\n  "))
	}
}
