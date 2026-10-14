// Package sync — csr_ranked_rule_guard_test.go : GARDE-RAIL du point de décision unique de
// classement (D-10, csr_writes.go).
//
// Historique : l'extraction CSR était gardée par `!reg.IsRanked`, lui-même posé par une recherche de
// « ranked » dans un nom de playlist absent du payload au moment du sync. Résultat : aucun CSR par
// match écrit depuis le 2026-09-16, pour personne, sans erreur ni alerte (plan ajustements
// 2026-10-14, G5). Ces motifs ne doivent pas revenir comme garde de l'extraction CSR.
package sync

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// motifs interdits dans les fichiers de production du paquet sync.
var rankedRuleForbidden = map[string]*regexp.Regexp{
	"ancien décideur par nom de playlist": regexp.MustCompile(`isRankedPlaylist`),
	"garde `!reg.IsRanked` de l'extraction CSR": regexp.MustCompile(
		`!\s*(reg|registry|rankedReg)\.IsRanked`),
}

func TestRankedRule_NoRegistryOrNameGuardInSyncPackage(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	scanned := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Clean(name))
		if err != nil {
			t.Fatal(err)
		}
		scanned++
		for why, re := range rankedRuleForbidden {
			if loc := re.FindIndex(data); loc != nil {
				t.Errorf("%s contient %q (%s) : la décision de classement est SkillShowsRankedMatch "+
					"(csr_writes.go), is_ranked du registre en est une conséquence", name, re.String(), why)
			}
		}
	}
	if scanned == 0 {
		t.Fatal("aucun fichier scanné")
	}
}

// Les extracteurs CSR ne LISENT pas reg.IsRanked.
func TestRankedRule_CSRExtractorsDoNotReadRegistryFlag(t *testing.T) {
	for _, name := range []string{"csr_writes.go", "csr_shared_writes.go"} {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		// Les commentaires peuvent nommer reg.IsRanked pour dire qu'il ne décide pas.
		for i, line := range strings.Split(string(data), "\n") {
			code := line
			if c := strings.Index(code, "//"); c >= 0 {
				code = code[:c]
			}
			// Une AFFECTATION (reg.IsRanked = ...) est la conséquence de la règle ; seule une lecture
			// (garde, condition) est interdite.
			if strings.Contains(code, ".IsRanked") && !strings.Contains(code, ".IsRanked =") {
				t.Errorf("%s:%d lit .IsRanked dans le code : %q", name, i+1, strings.TrimSpace(line))
			}
		}
	}
}

// Les deux chemins de sync posent is_ranked par la règle unique.
func TestRankedRule_BothSyncPathsApplyTheRule(t *testing.T) {
	for _, name := range []string{"engine_fetch.go", "engine_v2bridge.go"} {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "ApplyRankedRule(") {
			t.Errorf("%s n'appelle pas ApplyRankedRule : is_ranked resterait faux au sync", name)
		}
	}
}

// L'import OpenSpartan et le mapper ne reposent ni sur le nom de playlist ni sur un
// `IsRanked = true` forcé : ils appellent la règle unique.
func TestRankedRule_ImportUsesTheSingleDecisionPoint(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "service", "openspartan_import_service.go"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	if !strings.Contains(src, "AnySkillShowsRankedMatch") {
		t.Error("l'import OpenSpartan doit appeler sync.AnySkillShowsRankedMatch")
	}
	if strings.Contains(src, "IsRanked = true") {
		t.Error("l'import OpenSpartan force `IsRanked = true` : la règle unique doit décider")
	}
	m, err := os.ReadFile(filepath.Join("..", "openspartan", "mapper", "mapper.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(m), "IsRanked = true") {
		t.Error("le mapper OpenSpartan ne doit pas poser IsRanked par nom de playlist")
	}
}

// La référence des playlists classées ne décide pas du classement d'un match : elle sert de
// contrôle (platform/duckdb/csr_coverage_repo.go). Aucun fichier d'extraction CSR, de fetch ou de
// transformation du registre ne la consulte.
func TestRankedRule_ReferenceNeverDecidesInExtraction(t *testing.T) {
	files := []string{"csr_writes.go", "csr_shared_writes.go", "csr_backfill.go", "csr_shared_backfill.go",
		"engine_fetch.go", "transforms.go", "transforms_helpers.go"}
	checked := 0
	for _, name := range files {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		checked++
		if strings.Contains(string(data), "rankedplaylists") {
			t.Errorf("%s consulte rankedplaylists : la référence est un contrôle, pas un décideur de classement", name)
		}
	}
	if checked == 0 {
		t.Fatal("aucun fichier scanné")
	}
}
