// no_repo_root_walk_test.go : interdit de RÉ-IMPLÉMENTER la remontée « depuis le cwd
// jusqu'au marqueur db_profiles.json ». Le helper canonique est title.FindRepoRoot()
// (internal/domain/title/repo_root.go) ; il existait déjà en deux copies identiques
// (cmd/mapquant-build, cmd/replay-build) au moment où une troisième allait naître —
// règle CLAUDE.md n°6 : à la 3e copie, on centralise ET on pose le garde-rail.
package archlint

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// repoRootWalkAllowlist : fichiers (chemin relatif depuis apps/go-api/) autorisés à porter
// une remontée. Le helper canonique, plus deux variantes ANTÉRIEURES (2026-07-26) qui ne
// sont pas des copies du même code : elles partent de l'exécutable, bornent la remontée et
// cherchent un autre marqueur. Les migrer exigerait de changer leur comportement au
// démarrage du serveur — hors périmètre du chantier rejeu ; à traiter avec le prochain
// passage sur la config.
var repoRootWalkAllowlist = map[string]bool{
	"internal/domain/title/repo_root.go":   true, // helper canonique
	"internal/config/config.go":            true, // autoDetectRepoRoot serveur (exe + cwd, marqueur .example)
	"cmd/migrate-to-shared-social/main.go": true, // remontée bornée depuis l'exe
}

// repoRootWalkMarkers : la remontée se reconnaît à la conjonction du marqueur de racine et
// de la boucle vers le parent. Les fichiers qui lisent seulement LEVELUP_REPO_ROOT ne sont
// pas visés (ils ne dupliquent rien).
var repoRootWalkMarkers = []string{"db_profiles.json", "filepath.Dir(dir)"}

func TestNoDuplicateRepoRootWalk(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile))) // .../apps/go-api

	var violations []string
	for _, sub := range []string{"cmd", "internal"} {
		err := filepath.WalkDir(filepath.Join(goAPIRoot, sub), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, _ := filepath.Rel(goAPIRoot, path)
			rel = filepath.ToSlash(rel)
			if repoRootWalkAllowlist[rel] {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			body := stripComments(string(data))
			for _, m := range repoRootWalkMarkers {
				if !strings.Contains(body, m) {
					return nil
				}
			}
			violations = append(violations, rel)
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s/: %v", sub, err)
		}
	}
	if len(violations) > 0 {
		t.Errorf("remontée maison vers la racine du dépôt interdite — appeler "+
			"title.FindRepoRoot() :\n  %s", strings.Join(violations, "\n  "))
	}
}

// stripComments retire les lignes de commentaire : une explication citant db_profiles.json
// n'est pas une réimplémentation.
func stripComments(src string) string {
	var b strings.Builder
	for _, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
			continue
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}
