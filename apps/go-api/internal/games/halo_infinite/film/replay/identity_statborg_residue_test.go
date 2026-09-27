package replay

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/objectives"
)

// identity_statborg_residue_test.go — la voie `residu_de_manche` a une voie CANONIQUE et un
// compteur de repli (lot J8.4 du plan de suite d audit, constats FO-1 / RA2-4 : elle etait
// publiee « deduite » sans voie, et le compteur de retrait de cette inference etait aveugle).

// TestLeResiduDeMancheAUneVoieCanonique : la voie du residu se publie sous son nom.
//
// MUTATION : retirer le `case objectives.OriginRoundResidue` de [methodeStatborg] — ROUGE.
func TestLeResiduDeMancheAUneVoieCanonique(t *testing.T) {
	if got := methodeStatborg(objectives.OriginRoundResidue); got != canonical.MethodRoundResidue {
		t.Fatalf("methodeStatborg(%q) = %q, attendu %q", objectives.OriginRoundResidue, got,
			canonical.MethodRoundResidue)
	}
}

// TestChaqueOrigineDObjectivesAUneVoieCanonique : l EXHAUSTIVITE. Chaque constante `Origin*` que
// `objectives` declare (relue dans ses sources, pas recopiee ici) se traduit en une voie canonique
// NON VIDE. Une voie ajoutee a `objectives` sans son mapping rougit ici, au lieu de se publier
// `MethodNone` comme le residu l a fait du lot 6.7-B1 (2026-09-10) au lot J8.4.
func TestChaqueOrigineDObjectivesAUneVoieCanonique(t *testing.T) {
	origines := originesDObjectives(t)
	const plancher = 4 // mesure du 2026-09-27 : instants, triplet, elimination, residu
	if len(origines) < plancher {
		t.Fatalf("seulement %d constante(s) Origin* lue(s) dans `objectives` (plancher %d) : le "+
			"parcours est casse", len(origines), plancher)
	}
	for nom, valeur := range origines {
		if methodeStatborg(valeur) == canonical.MethodNone {
			t.Errorf("objectives.%s = %q n a aucune voie canonique dans methodeStatborg", nom, valeur)
		}
	}
}

// TestLesSlotsNommesParLeResiduSontComptes : chaque couple (manche, slot) nomme par le residu
// declenche `repli_identite_de_slot_par_residu_de_manche` ; les autres voies ne comptent pas.
//
// MUTATION : retirer le `Declenche` de [compterLesSlotsParResidu] — ROUGE.
func TestLesSlotsNommesParLeResiduSontComptes(t *testing.T) {
	fb := fallback.NouveauCompteur()
	slots := []IdentityStatborgSlot{
		{Slot: 10, Round: 0, XUID: "A", Link: canonical.Link{Method: canonical.MethodRoundResidue}},
		{Slot: 12, Round: 0, XUID: "B", Link: canonical.Link{Method: canonical.MethodDeathInstants}},
		{Slot: 14, Round: 1, XUID: "C", Link: canonical.Link{Method: canonical.MethodRoundResidue}},
		{Slot: 16, Round: 1, Link: canonical.Link{Method: canonical.MethodNone}},
	}
	compterLesSlotsParResidu(slots, fb)
	if got := fb.Compte(fallback.NomIdentiteDeSlotParResiduDeManche); got != 2 {
		t.Fatalf("%d declenchement(s), attendu 2 (les deux couples nommes par le residu)", got)
	}
}

// originesDObjectives relit les sources de `objectives` et rend `nom -> valeur` de chaque
// constante chaine dont le nom commence par `Origin`.
func originesDObjectives(t *testing.T) map[string]string {
	t.Helper()
	dir := filepath.Join("..", "internal", "facts", "objectives")
	entrees, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("lecture de %s : %v", dir, err)
	}
	out := map[string]string{}
	fset := token.NewFileSet()
	for _, e := range entrees {
		if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, 0)
		if err != nil {
			t.Fatalf("parse de %s : %v", e.Name(), err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			vs, ok := n.(*ast.ValueSpec)
			if !ok || len(vs.Names) != 1 || len(vs.Values) != 1 || !strings.HasPrefix(vs.Names[0].Name, "Origin") {
				return true
			}
			if lit, ok := vs.Values[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
				if v, err := strconv.Unquote(lit.Value); err == nil {
					out[vs.Names[0].Name] = v
				}
			}
			return true
		})
	}
	return out
}
