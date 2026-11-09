package replay

// powerup_pads_guard_test.go — LE GARDE-RAIL DU PREFIXE `powerup_`.
//
// POURQUOI IL EXISTE. Le prefixe qui designe un power-up dans le manifeste du titre a ete ecrit
// QUATRE fois dans ce paquet avant le 2026-08-19 : dans la production naissante, dans
// l'instrument des socles, dans celui des creations et dans celui de l'oracle. C'est le seuil
// que la regle du depot fixe (« a la 3e copie, centraliser ET poser un garde-rail ») — sans
// quoi la centralisation re-diverge au premier correctif. La lecon est datee : le predicat bot
// est passe de 8 a 36 copies apres avoir ete centralise SANS garde-rail.
//
// CE QU'IL INTERDIT, ET CE QU'IL AUTORISE. Il interdit le LITTERAL du prefixe (`"powerup_"`)
// partout ailleurs que dans sa declaration. Il autorise les NOMS COMPLETS de famille
// (`"powerup_overshield"`, `"powerup_camo"`) : ce sont des valeurs du manifeste que les tests
// citent pour se relier a lui, pas une seconde ecriture de la regle de prefixe.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// puPrefixeDeclare est le fichier qui DECLARE le prefixe. Tout autre fichier du paquet doit
// passer par la constante.
const puPrefixeDeclare = "powerup_pads.go"

// puLitteralPrefixe reconnait le littéral du prefixe SEUL — `"powerup_"` — et pas les noms
// complets de famille, qui portent des caracteres apres le tiret bas.
var puLitteralPrefixe = regexp.MustCompile(`"powerup_"`)

func TestPrefixePowerupNEstEcritQuUneFois(t *testing.T) {
	entrees, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du paquet : %v", err)
	}
	trouve := false
	for _, e := range entrees {
		nom := e.Name()
		if e.IsDir() || !strings.HasSuffix(nom, ".go") {
			continue
		}
		src, err := os.ReadFile(filepath.Clean(nom))
		if err != nil {
			t.Fatalf("%s : %v", nom, err)
		}
		n := len(puLitteralPrefixe.FindAll(src, -1))
		switch {
		case nom == puPrefixeDeclare:
			if n != 1 {
				t.Errorf("%s declare le prefixe %d fois, attendu 1", nom, n)
			}
			trouve = true
		case nom == "powerup_pads_guard_test.go":
			continue // ce fichier-ci porte le motif de recherche, pas une copie de la regle
		case n > 0:
			t.Errorf("%s reecrit le littéral du prefixe %d fois — passer par"+
				" `padPowerupPrefix` (powerup_pads.go). Une seconde ecriture diverge au premier"+
				" correctif, et le calque web lit le MEME prefixe (POWER_PAD_KEYS).", nom, n)
		}
	}
	if !trouve {
		t.Fatalf("%s ne declare plus le prefixe : le garde-rail ne garde plus rien",
			puPrefixeDeclare)
	}
}
