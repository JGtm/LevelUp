// Package archlint — no_players_root_join_test.go : ratchet dédup #6 (K1l, partie).
//
// Interdit toute reconstruction à la main du répertoire racine des joueurs d'un
// titre — `filepath.Join(<resolver>.TitleDataDir(slug), "players")`. Ce sous-chemin
// était recopié 6 fois (ops/backup_service, ops/healthcheck ×2,
// scheduler/data_health_check, service/media_index_service ×2). Source unique :
// PathResolver.PlayersRootDir(titleSlug) (internal/domain/title/registry.go).
// Leçon CLAUDE.md règle 6 (centraliser + garde-rail) + règle chemins (tout via
// PathResolver, jamais de sous-chemin data en dur).
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

// playersRootJoinRE matche `TitleDataDir(...), "players")` sur une même ligne —
// la reconstruction manuelle du répertoire racine des joueurs. Les chemins plus
// profonds (`..., "players", gamertag)` passent AUSSI par PlayerDir qui délègue à
// PlayersRootDir, donc ne doivent pas non plus réapparaître ici : le motif ancre
// sur `"players")` (fermeture immédiate du Join) pour cibler exactement la racine.
var playersRootJoinRE = regexp.MustCompile(`TitleDataDir\([^)]*\),\s*"players"\)`)

// playersRootJoinAllowed : seul le resolver porte la construction canonique.
var playersRootJoinAllowed = map[string]bool{
	"internal/domain/title/registry.go": true,
}

func TestNoManualPlayersRootJoin(t *testing.T) {
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
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			rel, _ := filepath.Rel(goAPIRoot, path)
			rel = filepath.ToSlash(rel)
			if playersRootJoinAllowed[rel] {
				return nil
			}
			for i, line := range strings.Split(string(data), "\n") {
				if playersRootJoinRE.MatchString(line) {
					violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+strings.TrimSpace(line))
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s/: %v", sub, err)
		}
	}
	if len(violations) > 0 {
		t.Errorf("reconstruction manuelle du répertoire racine joueurs interdite (dédup #6 K1l) — "+
			"utiliser PathResolver.PlayersRootDir(titleSlug) :\n  %s",
			strings.Join(violations, "\n  "))
	}
}
