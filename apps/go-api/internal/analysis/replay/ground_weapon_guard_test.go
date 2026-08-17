package replay

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// GARDE-RAIL de la chaine des ARMES AU SOL, portee en production a la phase 3 du plan
// `.ai/V7.5/replay2d/PLAN_ARMES_AU_SOL_2E_LECTURE.md`.
//
// LA REGLE N°6 DU DEPOT : une factorisation SANS garde-rail re-diverge, et la lecon est
// chiffree (predicat bot passe de 8 a 36 copies apres centralisation). Ici la divergence serait
// INVISIBLE et couteuse : les instruments sous garde publient les VERDICTS du chantier (combien
// de socles, quel cycle, quel accord) et l artefact publie les SOCLES. Si l un des deux
// reimplemente la regle de grappe, le bornage ou le verdict de stabilite, le controle d ancrage
// ne prouve plus rien — il comparerait deux chaines differentes, et le premier correctif porte
// d un seul cote passerait inapercu.
//
// CE QUE CE TEST INTERDIT : declarer une seconde fois un seuil du plan, ou redefinir une des
// fonctions de regle ailleurs que chez son proprietaire. Il ne peut pas empecher qu on recopie
// la LOGIQUE sous un autre nom — aucun grep ne le peut — mais il attrape le cas qui se produit
// vraiment : la copie faite « juste pour un instrument », avec le meme nom ou le meme seuil.

// gwThresholdOwners : chaque seuil du plan, et le SEUL fichier qui a le droit de le declarer.
// Les seuils ont ete ecrits AVANT la mesure et n ont pas bouge ; les redeclarer, c est ouvrir la
// porte a deux valeurs pour un meme seuil.
var gwThresholdOwners = map[string]string{
	"gwPadRadiusM":       "ground_weapon_rules.go",
	"gwPadMinHits":       "ground_weapon_rules.go",
	"gwPadCycleMinGaps":  "ground_weapon_rules.go",
	"gwPadCycleMaxCV":    "ground_weapon_rules.go",
	"gwPickupTrackTolUS": "ground_weapon_rules.go",
	"originDropMaxDist":  "equipment_placements.go",
	"originDropWindowUS": "equipment_placements.go",
}

// gwRuleOwners : chaque fonction de REGLE, et le seul fichier qui a le droit de la definir.
var gwRuleOwners = map[string]string{
	"gwPadsClusterAssign":     "ground_weapon_rules.go",
	"gwPadsKeep":              "ground_weapon_rules.go",
	"gwPadsCycle":             "ground_weapon_rules.go",
	"gwPadsCycleFromGaps":     "ground_weapon_rules.go",
	"gwPadsClass":             "ground_weapon_rules.go",
	"gwPadsIdentity":          "ground_weapon_rules.go",
	"gwPickupBoundsFrom":      "ground_weapon_bounds.go",
	"gwPickupNearestPass":     "ground_weapon_bounds.go",
	"gwPickupSeenWithin":      "ground_weapon_bounds.go",
	"gwPickupRefPos":          "ground_weapon_bounds.go",
	"gwPickupLifeTrack":       "ground_weapon_bounds.go",
	"groundWeaponObjects":     "ground_weapon_objects.go",
	"gwPickupPadGaps":         "ground_weapon_objects.go",
	"equipmentLives":          "equipment_placements.go",
	"decodeFilmGroundWeapons": "build_ground_weapons.go",
	"gwInstallMPPWidths":      "build_ground_weapons.go",
}

// TestUneSeuleDeclarationParSeuilDArmeAuSol : un seuil du plan ne se declare qu une fois.
//
// LE TEST COUVRE AUSSI LES `_test.go`, et c est le point : la copie qui divergerait viendrait
// d un instrument, pas de la production — c est de la que la chaine sortait avant la phase 3.
func TestUneSeuleDeclarationParSeuilDArmeAuSol(t *testing.T) {
	for name, owner := range gwThresholdOwners {
		pattern := regexp.MustCompile(`(?m)^\s*(const\s+)?` + regexp.QuoteMeta(name) + `\s*=`)
		gwAssertSingleOwner(t, name, owner, pattern)
	}
}

// TestUneSeuleDefinitionParRegleDArmeAuSol : une regle de la chaine ne se definit qu une fois.
//
// LA PARENTHESE OUVRANTE FAIT PARTIE DU MOTIF, et ce n est pas cosmetique : sans elle,
// « gwPadsCycle » attrapait aussi `gwPadsCycleSummary` et `gwPadsCycleFromGaps`, si bien qu une
// regle dont le nom est le PREFIXE d une autre ne pouvait pas entrer dans cette table
// (correctif de revue du 2026-08-17).
func TestUneSeuleDefinitionParRegleDArmeAuSol(t *testing.T) {
	for name, owner := range gwRuleOwners {
		pattern := regexp.MustCompile(`(?m)^func\s+(\([^)]*\)\s*)?` + regexp.QuoteMeta(name) + `\(`)
		gwAssertSingleOwner(t, name, owner, pattern)
	}
}

// gwAssertSingleOwner verifie qu un motif n apparait QUE dans le fichier qui le possede — et
// qu il y apparait encore.
//
// LES DEUX MOITIES COMPTENT : un garde qui ne peut plus echouer ne garde rien (lecon J4.0). Si
// le motif disparait de son proprietaire, c est que la regle a ete renommee ou deplacee, et le
// test doit le dire au lieu de passer en silence.
func gwAssertSingleOwner(t *testing.T, name, owner string, pattern *regexp.Regexp) {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du paquet : %v", err)
	}
	var offenders []string
	seenOwner := false
	for _, e := range entries {
		file := e.Name()
		if e.IsDir() || !strings.HasSuffix(file, ".go") {
			continue
		}
		raw, err := os.ReadFile(filepath.Clean(file))
		if err != nil {
			t.Fatalf("lecture %s : %v", file, err)
		}
		if !pattern.Match(raw) {
			continue
		}
		if file == owner {
			seenOwner = true
			continue
		}
		offenders = append(offenders, file)
	}
	if !seenOwner {
		t.Fatalf("%q introuvable dans %s : le garde-rail ne verifie plus rien (regle renommee"+
			" ou deplacee ?)", name, owner)
	}
	if len(offenders) > 0 {
		t.Fatalf("%q est redefini hors de %s : %v. La chaine des armes au sol n a qu UNE"+
			" copie de chaque regle et de chaque seuil — les instruments sous garde APPELLENT"+
			" la production, ils ne la recopient pas (sans quoi le controle d ancrage du plan"+
			" comparerait deux chaines differentes)", name, owner, offenders)
	}
}
