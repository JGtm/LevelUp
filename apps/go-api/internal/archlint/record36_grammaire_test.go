// Package archlint — record36_grammaire_test.go : LE RECORD DE TIR NE SE LIT QUE PAR SA GRAMMAIRE
// (lot M4b de la campagne « retours rejeu », 2026-09-24).
//
// LE DEFAUT QUE CE RATCHET FERME. Le record 36 (`action_weapon_fire`) etait lu a OFFSETS FIXES :
// l indice de tireur sur quatre bits aux bits 36..39 (saturant a 15 au-dela de seize joueurs :
// 39 joueurs de BTB sans aucun tir publie, 1 205 identifiants decales), l arme aux bits 44..107, la
// visee au bit 113, derriere un filtre sur l octet de tete 0xD2 qui laissait passer le type 37.
// Ces offsets ne valaient que pour la disposition canonique ; sur la build HI_1_4_1 le champ lu
// comme tireur est une constante. Depuis M4b la tete se lit par la grammaire du record
// (`grammar/fire_events.go`, `lireEnteteTir36`), qui rend aussi la reference 0 (l unite tireuse).
//
// CE QUI EST INTERDIT, dans tout fichier .go NON-TEST du module : les noms de l ancienne API a
// offsets fixes. Leur retour signerait une seconde lecture du record, a cote de la grammaire.
//
// L ALLOWLIST est DATEE et porte son critere de retrait (CLAUDE.md, garde-rails) : le scanner de
// `weaponscan` lit le DENOMINATEUR de la precision par arme (`shared.match_weapon_shots`) par son
// marqueur universel, a ses propres offsets. Il ne nomme aucun des identifiants ci-dessous ; il est
// cite ici pour que personne ne l y ajoute sans le dire : il sortira quand la precision lira ses
// tirs par la grammaire du record (`grammar.ScanFireEvents`), lot a ouvrir sur decision.
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// record36OffsetsFixes : les identifiants de l ancienne lecture a offsets fixes du record 36.
var record36OffsetsFixes = regexp.MustCompile(
	`\b(ShooterIndex5|ReadAttackerIndex|ReadShooterIndex5|FireEventType|FireHeadBits|fireAttackerBit|fireFlagsBit|fireAimBit|fireWeaponBit)\b`)

// record36Allowlist : les fichiers de production qui lisent encore un tir a offsets fixes, avec
// leur date et leur critere de retrait (cf. l en-tete). Aucun ne nomme les identifiants interdits.
var record36Allowlist = map[string]string{
	"internal/games/halo_infinite/film/internal/grammar/weaponscan/scanner.go": "2026-09-24 — " +
		"denominateur de la precision par arme ; retrait quand la precision lit ses tirs par " +
		"grammar.ScanFireEvents",
}

// TestRecord36SeLitParSaGrammaire refuse tout retour de l ancienne API a offsets fixes.
func TestRecord36SeLitParSaGrammaire(t *testing.T) {
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	racine := filepath.Dir(filepath.Dir(filepath.Dir(ici))) // apps/go-api
	var trouves []string
	err := filepath.WalkDir(racine, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") || d.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(racine, p)
		rel = filepath.ToSlash(rel)
		if _, autorise := record36Allowlist[rel]; autorise {
			return nil
		}
		src, err := os.ReadFile(p) //nolint:gosec // fichier du module, parcouru en lecture
		if err != nil {
			return err
		}
		for i, ligne := range strings.Split(string(src), "\n") {
			if record36OffsetsFixes.MatchString(ligne) {
				trouves = append(trouves, rel+":"+itoa(i+1)+" : "+strings.TrimSpace(ligne))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parcours du module : %v", err)
	}
	if len(trouves) > 0 {
		t.Errorf("le record 36 est relu a OFFSETS FIXES hors de sa grammaire (%d ligne(s)) :\n  %s\n"+
			"La tete du record se lit par `grammar.lireEnteteTir36` (fire_events.go) ; un nouveau "+
			"champ s y ajoute, jamais a cote.", len(trouves), strings.Join(trouves, "\n  "))
	}
	for f := range record36Allowlist {
		if _, err := os.Stat(filepath.Join(racine, filepath.FromSlash(f))); err != nil {
			t.Errorf("allowlist : %s n existe plus — retirer son entree (%v)", f, err)
		}
	}
}
