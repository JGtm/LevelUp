package settings

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestGuard_AucunEcrivainDuServeurNAppelleSaveDirectement : un écrivain du serveur modifie app_settings.json par
// `Store.Update` (verrou d'écriture partagé), jamais par `settingsStore.Save` : un Load puis Save hors du verrou
// perd l'écriture d'un autre écrivain. Le garde cherche l'ancien littéral dans le code de production du serveur
// (`internal/` et `cmd/server/`) ; les outils à processus unique (`cmd/levelup`) n'y sont pas.
func TestGuard_AucunEcrivainDuServeurNAppelleSaveDirectement(t *testing.T) {
	racine := filepath.Join("..", "..", "..") // apps/go-api
	re := regexp.MustCompile(`settingsStore\.Save\(`)
	for _, dir := range []string{"internal", filepath.Join("cmd", "server")} {
		err := filepath.WalkDir(filepath.Join(racine, dir), func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return nil
			}
			src, rerr := os.ReadFile(p)
			if rerr != nil {
				return rerr
			}
			if re.Match(src) {
				t.Errorf("%s appelle settingsStore.Save : passer par settingsStore.Update", p)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
