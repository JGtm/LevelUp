// Package archlint — no_local_longest_run_test.go : ratchet F2 (revue 2026-07-17).
//
// Interdit toute nouvelle copie locale du balayage « plus longue série
// consécutive » (idiome `cur++` suivi d'un `if cur > best {` de mise à jour du
// max). La source unique est analysis.LongestRun[T] (internal/analysis). Le motif
// avait été recopié en quatre exemplaires (longestOutcomeRun,
// sliceBestWinStreakCanonical, synthesis winStreak/maxStreak, detectTilt) —
// leçon CLAUDE.md règle 6.
//
// ALLOWLIST (accumulateurs sémantiquement distincts, PAS un LongestRun binaire) :
//   - longest_run.go       : la source unique elle-même.
//   - max_killing_spree.go : accumulateur À TROIS ÉTATS (kill=+1 / death=reset /
//     autre event=IGNORÉ), non réductible à LongestRun(items, pred) qui reset sur
//     tout non-match. Distinct par construction — daté 2026-07-17.
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// increment `x++` puis peu après `if y > z {` — signature du balayage best/cur.
//
// LES DEUX OPÉRANDES SONT DES IDENTIFIANTS, jamais un nombre (précision apportée le
// 2026-08-30). Le motif traqué compare un courant à un maximum — `if cur > best {` —, deux
// variables par construction. `\w+` acceptait aussi un littéral, ce qui faisait tomber dans
// le ratchet la forme la plus banale de Go : compter puis tester un seuil
// (`n++` sur la ligne d'avant, `if want > 0 {` sur la suivante — cas réel de
// `cmd/oddball-terrain/confront.go`, qui n'a aucune notion de série). Ce n'est PAS un
// assouplissement : aucun balayage best/cur ne compare son maximum à une constante, donc
// aucune violation réelle ne sort du filet — seule une classe de faux positifs disparaît.
var longestRunIdiomRE = regexp.MustCompile(`\+\+[ \t]*\n[ \t]*if\s+[A-Za-z_]\w*\s*>\s*[A-Za-z_]\w*\s*\{`)

var longestRunAllowed = map[string]bool{
	"longest_run.go":       true,
	"max_killing_spree.go": true,
}

// longestRunAllowedPrefixes — exemptions PAR CHEMIN, ajoutées le 2026-07-31 à la
// réconciliation du rejeu 2D et du décodeur killsource.
//
// POURQUOI UNE EXEMPTION ET PAS UNE MIGRATION VERS analysis.LongestRun : ces
// balayages ne cherchent pas « la plus longue série » d'un prédicat métier (série de
// victoires, série de kills). Ce sont des mesures de DÉCODAGE — la plus longue plage
// de bits stables, le plus grand nombre de records consécutifs consommés sans
// désynchronisation — dont le compteur porte des grandeurs de flux binaire, pas des
// items de domaine. `analysis.LongestRun[T]` prend une slice et un prédicat : il ne
// modélise pas un curseur de bits. La forme se ressemble, l'objet non.
//
// Le paquet `filmdec` est de surcroît importé tel quel des deux lignées de recherche
// (il n'existait ni dans la base commune ni sur main) : deux de ses fichiers
// déclenchaient déjà ce ratchet sur `feat/killsource-prod` AVANT toute réconciliation.
// L'exemption règle donc une dette antérieure, elle n'en crée pas.
//
// RETRAIT : quand `filmdec` aura sa propre primitive de plage de bits. Le volet
// `cmd/tmp_*` / `cmd/wf_*` de cette exemption est RETIRÉ le 2026-08-01 (lot A du
// plan de dette avant merge) : l'outillage de recherche est supprimé, l'exemption
// par préfixe n'a plus de cible. Le volet `frame_debug.go` est RETIRÉ le 2026-09-05
// (lot E, item E.2) : le fichier est supprimé (deux fonctions de debug sans aucun
// appelant, tests compris), l'exemption n'a plus de cible non plus.
var longestRunAllowedPrefixes = []string{
	"internal/analysis/filmdec/frame_records.go",
}

// longestRunExempt dit si le chemin relatif est couvert par une exemption datée.
func longestRunExempt(rel string) bool {
	for _, p := range longestRunAllowedPrefixes {
		if strings.HasPrefix(rel, p) {
			return true
		}
	}
	return false
}

func TestNoLocalLongestRun(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	internalRoot := filepath.Dir(filepath.Dir(thisFile))
	goAPIRoot := filepath.Dir(internalRoot)

	var violations []string
	for _, sub := range []string{"internal", "cmd"} {
		root := filepath.Join(goAPIRoot, sub)
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			if longestRunAllowed[filepath.Base(path)] {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			if longestRunIdiomRE.Match(data) {
				rel, _ := filepath.Rel(goAPIRoot, path)
				slash := filepath.ToSlash(rel)
				if longestRunExempt(slash) {
					return nil
				}
				violations = append(violations, slash)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s/: %v", sub, err)
		}
	}
	if len(violations) > 0 {
		t.Errorf("balayage « plus longue série » local interdit (F2) — utiliser "+
			"analysis.LongestRun[T] (internal/analysis) :\n  %s",
			strings.Join(violations, "\n  "))
	}
}
