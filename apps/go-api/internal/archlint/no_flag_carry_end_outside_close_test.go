// no_flag_carry_end_outside_close_test.go — UN SEUL ENDROIT ECRIT LA FIN D'UN PORTAGE DE DRAPEAU.
//
// # CE QUE CE GARDE-RAIL EMPECHE DE REECRIRE
//
// Le P0 de la revue de la vague 6 (2026-09-11, `.ai/V7.5/REVUE_VAGUE6_2026-09-11.md`, constat C1) :
// quatre chaines fermaient un portage en ecrivant chacune `raws[i].t1, raws[i].closed = at, true`
// a sa facon, et l'etat pose par la precedente (`homed`, `captured`) n'etait jamais dementi par
// une chaine plus precoce. Resultat mesure sur le parc : 21 portages comptes dans deux `closedBy*`
// a la fois, et 46,3 s de drapeau dessine a sa base alors qu'il gisait au sol.
//
// La correction a centralise l'ecriture dans `flag_carries_close.go` (`flagCloseAt`) : instant
// strictement interieur, reecriture complete de l'etat de fin, compteur derive du fermoir en
// vigueur. Quatre copies centralisees = la regle n° 6 du depot exige le garde-rail dans le meme
// lot (« une factorisation sans garde-rail re-diverge »). La ronde 2 de la revue l'a demande
// (constat 3) : le voici.
//
// LE MOTIF : toute affectation d'un des cinq champs de fin d'un `flagCarryRaw` par selecteur
// (`.t1 =`, `.closed =`, `.homed =`, `.captured =`, `.closedBy =`). La CONSTRUCTION par litteral
// (`flagCarryRaw{t1: ..., closed: ...}` dans `boundFlagCarries`) n'est pas visee : elle pose la
// borne initiale, pas un fermoir.
//
// LES `_test.go` NE SONT PAS BALAYES : les tests montent des portages synthetiques.
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// flagCarryEndAllowlist : le seul fichier autorise a ecrire la fin d'un portage.
//
//	flag_carries_close.go   2026-09-11, revue 6.R C1 — `flagCloseAt`, l'unique fermoir.
var flagCarryEndAllowlist = map[string]bool{
	"flag_carries_close.go": true,
}

var flagCarryEndWrite = regexp.MustCompile(`\.(t1|closed|homed|captured|closedBy)\s*=[^=]`)

func TestFinDePortageDeDrapeauEcriteSeulementDansFlagCarriesClose(t *testing.T) {
	_, here, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(here), "..", "games", "halo_infinite", "film", "replay")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("lecture de %s : %v", dir, err)
	}
	var offenders []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, "flag_") || !strings.HasSuffix(name, ".go") ||
			strings.HasSuffix(name, "_test.go") || flagCarryEndAllowlist[name] {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("lecture de %s : %v", name, err)
		}
		for i, line := range strings.Split(string(raw), "\n") {
			if flagCarryEndWrite.MatchString(line) {
				offenders = append(offenders, name+":"+itoa(i+1)+" "+strings.TrimSpace(line))
			}
		}
	}
	if len(offenders) > 0 {
		t.Fatalf("la fin d'un portage de drapeau ne s'ecrit que par flagCloseAt (flag_carries_close.go) ; "+
			"passer par lui au lieu d'ecrire le champ :\n  %s", strings.Join(offenders, "\n  "))
	}
}
