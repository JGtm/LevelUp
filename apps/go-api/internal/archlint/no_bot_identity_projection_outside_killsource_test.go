// Package archlint — no_bot_identity_projection_outside_killsource_test.go : ratchet lot 5.1.
//
// `replay.BotIdentity` se construit à UN SEUL endroit en code de production :
// `internal/games/halo_infinite/replayidentity/bot_identities.go` (`BotIdentities`). Avant ce
// lot, `internal/replaybuild` en portait sa PROPRE copie et `internal/sync/killcollector` n'en
// avait AUCUNE — ce qui laissait le collecteur de sync sans les identités de bot que le registre
// d'identité a besoin pour départager un siège d'index partagé bot/humain (revue adversariale de
// la vague 4, 2026-09-10, constat P2 sur `killcollector/positions.go:484-489`).
//
// Le paquet dédié (plutôt qu'un ajout à `killsource` ou à `replay`) existe pour éviter un cycle
// d'IMPORT DE TEST : `replay` a un instrument de mesure (`visee_lunette_research_test.go`,
// `package replay`) qui importe déjà `killsource` — un `killsource` qui importerait `replay` en
// production aurait fermé la boucle à la compilation des tests de `replay`. Voir l'en-tête de
// `replayidentity/bot_identities.go`.
//
// Ce garde-rail interdit qu'une SECONDE copie renaisse : `replaybuild` et `killcollector`
// appellent tous deux `replayidentity.BotIdentities`, jamais un littéral local. Une divergence
// entre deux copies (filtre des bots non nommés, des bots non épinglés) romprait exactement la
// garantie D11 (« deux producteurs, un seul nommage ») que `identity_registry.go` documente.
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

// botIdentityLiteralRE matche la construction du type, pas son usage (un appelant qui range le
// résultat de `killsource.BotIdentities` dans une variable ne matche pas).
var botIdentityLiteralRE = regexp.MustCompile(`\breplay\.BotIdentity\{`)

// botIdentityAllowlist : le seul fichier de production autorisé à construire le type.
const botIdentityAllowlistFile = "bot_identities.go"

func TestNoBotIdentityProjectionOutsideKillsource(t *testing.T) {
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
			base := filepath.Base(path)
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			if base == botIdentityAllowlistFile {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			rel, _ := filepath.Rel(goAPIRoot, path)
			rel = filepath.ToSlash(rel)
			for i, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(strings.TrimSpace(line), "//") {
					continue
				}
				if botIdentityLiteralRE.MatchString(line) {
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
		t.Errorf("construction de replay.BotIdentity hors de killsource.BotIdentities — "+
			"appelez killsource.BotIdentities(res) plutôt que de reprojeter le roster :\n  %s",
			strings.Join(violations, "\n  "))
	}
}
