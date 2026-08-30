// Package archlint — no_objective_submode_list_test.go : interdit une 2e copie de
// la liste des sous-modes de la famille OBJECTIF (Halo Infinite).
//
// Source unique : internal/games/halo_infinite/skillchain/objective_family.go
// (IsObjectiveSubMode). Deux consommateurs en dépendent — la chaîne LUSR sociale
// (arena_objectif) et la chaîne du score de performance classé (ranked_objectif,
// lot 1 du plan .ai/PLAN_PERF_NOTE_OBJECTIFS.md) : une copie divergerait au premier
// ajout de sous-mode fait d'un seul côté, et le MÊME match tomberait en famille
// objectif pour le LUSR et en famille slayer pour la performance.
//
// La source unique porte 17 entrées depuis le lot 1bis (2026-08-27 : ajout de
// `vip`, `neutral bomb`, `one bomb`, `neutral bomb squad`, `ctf 3 captures`) ainsi
// que la RÈGLE DU PRÉFIXE pour les pair_name inversés. Les marqueurs de détection
// ci-dessous sont inchangés : ils signent la liste, pas ses règles de lecture.
//
// DÉTECTION — 3 littéraux MARQUEURS, variantes CTF propres à cette liste :
// "neutral flag ctf", "one flag ctf", "covert one flag". Mesuré le 2026-08-27 :
// ces 3 chaînes n'apparaissent QUE dans la source unique et dans l'outil de
// simulation du lot 0. Les autres entrées de la liste ("total control",
// "land grab", "king of the hill", "stockpile"…) sont volontairement HORS
// détection : elles vivent légitimement dans 3 catalogues de modes voisins
// (analysis/objectiveevents/extract.go, games/halo_infinite/catalog_adapter.go,
// ops/catalog_refresh.go) et les allowlister noierait le signal. Toute recopie de
// la liste de classification embarque les variantes CTF — c'est la signature.
//
// La détection est CASE-SENSITIVE sur la forme normalisée (minuscules, entre
// guillemets) : les pair_names de fixtures ("Arena:One Flag CTF on Aquarius")
// ne déclenchent pas le ratchet.
package archlint

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// objectiveListMarkers : littéraux Go signant la liste des sous-modes objectif.
var objectiveListMarkers = []string{
	`"neutral flag ctf"`,
	`"one flag ctf"`,
	`"covert one flag"`,
}

// objectiveListSourceFile : LA source unique (relative à apps/go-api).
const objectiveListSourceFile = "internal/games/halo_infinite/skillchain/objective_family.go"

// objectiveListAllowlist : fichiers autorisés à porter les marqueurs.
// Toute nouvelle entrée exige une justification datée ET une date de retrait.
var objectiveListAllowlist = map[string]bool{
	objectiveListSourceFile: true, // source unique (IsObjectiveSubMode)
	// Tests directs du helper : ils vérifient la liste, donc la citent.
	"internal/games/halo_infinite/skillchain/objective_family_test.go": true,
	// Ce ratchet lui-même (il porte les marqueurs pour les chercher).
	"internal/archlint/no_objective_submode_list_test.go": true,
	// Outil de simulation offline du lot 0 (2026-08-27, plan PERF_NOTE_OBJECTIFS
	// B0.1) : copie locale assumée par le plan — l'outil ne dépend pas du seam
	// title-aware câblé au boot. MIROIR à resynchroniser à chaque évolution de la
	// liste (fait au lot 1bis, 2026-08-27).
	// Retrait de cette entrée avec l'outil (jetable, non versionné à ce jour).
	"cmd/diag_perfsim/score.go": true,
}

// objectiveListSkipDirs : répertoires ignorés au parcours (non-Go / hors module).
var objectiveListSkipDirs = map[string]bool{
	".git": true, "node_modules": true, "data": true, "tmp": true, "testdata": true,
}

func TestNoDuplicateObjectiveSubModeList(t *testing.T) {
	root := goAPIRoot(t)

	var violations []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if objectiveListSkipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		if objectiveListAllowlist[rel] {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for i, line := range strings.Split(string(data), "\n") {
			for _, marker := range objectiveListMarkers {
				if strings.Contains(line, marker) {
					violations = append(violations,
						rel+":"+strconv.Itoa(i+1)+"  "+strings.TrimSpace(line))
					break
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk apps/go-api : %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("liste des sous-modes objectif dupliquée — elle ne doit exister QUE dans %s "+
			"(skillchain.IsObjectiveSubMode) ; brancher l'appelant sur le helper (seam "+
			"SetObjectiveFamilyClassifier côté sync) au lieu de recopier :\n  %s",
			objectiveListSourceFile, strings.Join(violations, "\n  "))
	}
}

// TestObjectiveListMarkersStillInSource empêche le ratchet de s'endormir : si la
// source unique est renommée/déplacée ou si les marqueurs disparaissent de la
// liste, le test ci-dessus deviendrait vert à vide. Ici on exige que la source
// existe ET porte les 3 marqueurs.
func TestObjectiveListMarkersStillInSource(t *testing.T) {
	root := goAPIRoot(t)
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(objectiveListSourceFile)))
	if err != nil {
		t.Fatalf("source unique de la liste introuvable (%s) — si elle a bougé, "+
			"mettre à jour objectiveListSourceFile ET l'allowlist : %v", objectiveListSourceFile, err)
	}
	for _, marker := range objectiveListMarkers {
		if !strings.Contains(string(data), marker) {
			t.Errorf("marqueur %s absent de %s : le ratchet ne détecterait plus une copie "+
				"(retirer le marqueur de la liste OU corriger objectiveListMarkers)",
				marker, objectiveListSourceFile)
		}
	}
}

// goAPIRoot retourne la racine du module go-api depuis l'emplacement de ce fichier
// (.../apps/go-api/internal/archlint/x_test.go → .../apps/go-api).
func goAPIRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
}
