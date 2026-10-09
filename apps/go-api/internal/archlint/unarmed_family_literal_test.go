package archlint

// unarmed_family_literal_test.go — L IDENTIFIANT « MAINS NUES » `00007CA9` N A QU UNE ECRITURE EN
// CODE DE PRODUCTION (retours du rejeu, lot M6.3, 2026-09-24 ; regle 6 de CLAUDE.md).
//
// POURQUOI. L objet « mains nues » (`WeaponTags.unarmed` du script Lua global, sonde CA9) est
// exclu de TOUTE dotation affichee et sa remise par le jeu n est pas un ramassage : deux regles,
// deux chemins de publication aujourd hui (`document_pickups.go`, `loadouts.go`), un troisieme a
// la vague D (dotations de naissance, lot M3). Si chaque chemin ecrit son propre `0x00007ca9`, le
// jour ou l un l oublie, l objet reapparait comme une arme sur une fiche — sans erreur ni
// compteur. La constante canonique est `filmshell.UnarmedFamily` (et l identifiant d arme 64 bits
// qui en derive, `filmshell.UnarmedWeaponID`) ; tout autre fichier de PRODUCTION qui ecrirait le
// litteral fait echouer ce test. Les tests gardent leurs litteraux (discipline des fixtures : un
// attendu derive de la constante testee rendrait le test tautologique) ; les commentaires aussi.

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// unarmedLiteralRE : le GlobalID ecrit en hexadecimal, seul (`0x00007ca9`, `0x7CA9`) ou en tete
// d un identifiant d arme 64 bits (`0x00007ca942c9679f`).
var unarmedLiteralRE = regexp.MustCompile(`(?i)\b0x0*7ca9(\b|[0-9a-f]{8}\b)`)

// unarmedLiteralMaison : le SEUL fichier de production qui porte le litteral.
const unarmedLiteralMaison = "internal/games/weapons/filmshell/unarmed.go"

func TestUnarmedFamilyLiteralUnique(t *testing.T) {
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(ici))) // .../apps/go-api
	var violations []string
	maisonVue := false
	err := filepath.WalkDir(goAPIRoot, func(chemin string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if chemin != goAPIRoot && (d.Name() == "testdata" || strings.HasPrefix(d.Name(), ".")) {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(chemin, ".go") || strings.HasSuffix(chemin, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(goAPIRoot, chemin)
		rel = filepath.ToSlash(rel)
		data, rerr := os.ReadFile(chemin) //nolint:gosec // source du module, lecture seule
		if rerr != nil {
			return rerr
		}
		for i, ligne := range strings.Split(string(data), "\n") {
			code := strings.TrimSpace(ligne)
			if strings.HasPrefix(code, "//") || !unarmedLiteralRE.MatchString(code) {
				continue
			}
			if rel == unarmedLiteralMaison {
				maisonVue = true
				continue
			}
			violations = append(violations, rel+":"+strconv.Itoa(i+1)+" -> "+code)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parcours du module : %v", err)
	}
	if !maisonVue {
		t.Errorf("%s ne porte plus le litteral : le garde-rail ne garde plus rien (constante "+
			"deplacee ? mettre a jour `unarmedLiteralMaison`)", unarmedLiteralMaison)
	}
	if len(violations) > 0 {
		t.Errorf("identifiant « mains nues » ecrit hors de %s (%d) — passer par "+
			"`filmshell.UnarmedFamily` / `filmshell.IsUnarmedFamily` :\n  %s",
			unarmedLiteralMaison, len(violations), strings.Join(violations, "\n  "))
	}
}
