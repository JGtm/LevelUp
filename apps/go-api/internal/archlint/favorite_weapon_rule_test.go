package archlint

// favorite_weapon_rule_test.go — « L ARME FAVORITE » NE SE DECIDE PLUS PAR `weaponID == 0` (revue
// adverse du lot M6 des retours du rejeu, constat R1, 2026-09-24 ; regles 6 et 8 de CLAUDE.md).
//
// Trois lecteurs de `platform/duckdb` (scoreboard du match, top armes de l Explorateur, arme
// favorite de l Accueil) ecartaient les objets hors arsenal par `weaponID == 0` / `!= 0`. Depuis
// que la bobine a fusion porte un identifiant, c etait faux : la regle est nommee,
// `domain.IsFavoriteWeaponCandidate`, et ce test interdit que le litteral revienne dans le paquet.

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

var weaponIDZeroRE = regexp.MustCompile(`(?i)\b(weaponID|numericID)\s*[!=]=\s*0\b`)

func TestArmeFavoriteParLaRegleNommee(t *testing.T) {
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	dir := filepath.Join(filepath.Dir(filepath.Dir(ici)), "platform", "duckdb")
	entrees, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("lecture de %s : %v", dir, err)
	}
	var violations []string
	for _, e := range entrees {
		nom := e.Name()
		if e.IsDir() || !strings.HasSuffix(nom, ".go") || strings.HasSuffix(nom, "_test.go") {
			continue
		}
		data, rerr := os.ReadFile(filepath.Join(dir, nom)) //nolint:gosec // source du module
		if rerr != nil {
			t.Fatal(rerr)
		}
		for i, ligne := range strings.Split(string(data), "\n") {
			code := strings.TrimSpace(ligne)
			if !strings.HasPrefix(code, "//") && weaponIDZeroRE.MatchString(code) {
				violations = append(violations, nom+":"+strconv.Itoa(i+1)+" -> "+code)
			}
		}
	}
	if len(violations) > 0 {
		t.Errorf("test d arsenal par l identifiant nul (%d) — passer par "+
			"`domain.IsFavoriteWeaponCandidate` :\n  %s", len(violations), strings.Join(violations, "\n  "))
	}
}
