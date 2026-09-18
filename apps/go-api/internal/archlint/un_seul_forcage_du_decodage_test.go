package archlint

// un_seul_forcage_du_decodage_test.go — LE REGLAGE QUI FORCE LE DECODAGE N A QU UN APPELANT
// (lot 4.1.3 du PLAN_DECODEUR_FILM, 2026-09-18).
//
// # CE QU IL GARDE
//
// `replaybuild.Builder.SansFaitsPersistes()` fait IGNORER les faits persistes d un film et force
// le decodage. Il existe pour UNE raison, mesurable : le test S8 (`cmd/replay-equiv -deux-passes`)
// doit jouer LES DEUX BRANCHES DU MEME COMMIT et comparer leurs artefacts a l octet. Sans lui, la
// seconde passe relirait les faits que la premiere vient d ecrire, et le harnais comparerait
// « faits contre faits » — une equivalence VACUANTE.
//
// # POURQUOI UN RATCHET, ET PAS SEULEMENT UN COMMENTAIRE
//
// CLAUDE.md regle 11 interdit le drapeau qui laisse une fonctionnalite OFF « pour plus tard ». Ce
// reglage n en est pas un — la bascule est ACTIVE par defaut et le reste —, mais il en a la forme,
// et c est exactement ce qui se met a deriver : un second appelant « juste pour ce diagnostic »,
// puis un troisieme dans un chemin de production, et la bascule est morte sans que personne ne
// l ait decidee.
//
// UN SEUL APPELANT, NOMME. Le critere de retrait est ecrit chez le reglage : le jour ou le harnais
// saurait forcer la branche autrement (deux racines de cache, par exemple), le reglage se retire
// avec son dernier appelant — et ce test rougit si l allowlist devient perimee.

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// nomDuForcageDuDecodage — le reglage garde. En clair, parce que ce test ne peut pas importer
// `replaybuild` sans creer un cycle avec ses propres tests.
const nomDuForcageDuDecodage = "SansFaitsPersistes"

// appelantsDuForcageDuDecodage — LES SEULS appelants, avec leur raison. Chemins relatifs a
// `apps/go-api`. 2026-09-18, lot 4.1.3.
var appelantsDuForcageDuDecodage = map[string]string{
	"internal/replaybuild/filmfacts_cuisson.go": "la DEFINITION du reglage et sa lecture — " +
		"c est le porteur, pas un appelant",
	"internal/replaybuild/replaybuild.go": "le CHAMP du Builder et son godoc, qui renvoie au " +
		"reglage — une declaration, pas un appel",
	"cmd/replay-equiv/child.go": "l enfant de la passe `film` du mode S8 : il doit mesurer le " +
		"DECODAGE, pas relire les faits que la passe precedente a ecrits",
}

func TestUnSeulForcageDuDecodage(t *testing.T) {
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	racine := filepath.Dir(filepath.Dir(filepath.Dir(ici))) // .../apps/go-api
	vus := map[string]bool{}
	err := filepath.WalkDir(racine, func(chemin string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			nom := d.Name()
			if chemin != racine && (strings.HasPrefix(nom, ".") || nom == "node_modules") {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(chemin, ".go") || strings.HasSuffix(chemin, "_test.go") {
			return nil
		}
		blob, rerr := os.ReadFile(chemin) //nolint:gosec // chemin derive du module
		if rerr != nil {
			return rerr
		}
		if !strings.Contains(string(blob), nomDuForcageDuDecodage) {
			return nil
		}
		rel, rerr := filepath.Rel(racine, chemin)
		if rerr != nil {
			return rerr
		}
		vus[filepath.ToSlash(rel)] = true
		return nil
	})
	if err != nil {
		t.Fatalf("balayage de %s : %v", racine, err)
	}
	if len(vus) == 0 {
		t.Fatalf("%q n apparait NULLE PART : le reglage a ete renomme ou retire — mettre ce "+
			"ratchet a jour, ou le supprimer avec lui.", nomDuForcageDuDecodage)
	}
	for rel := range vus {
		if _, autorise := appelantsDuForcageDuDecodage[rel]; !autorise {
			t.Errorf("%s cite %q hors allowlist : un second forceur de decodage tue la bascule "+
				"sans que personne ne l ait decide. L inscrire avec sa raison DATEE, ou passer "+
				"par la porte ordinaire.", rel, nomDuForcageDuDecodage)
		}
	}
	for rel, raison := range appelantsDuForcageDuDecodage {
		if !vus[rel] {
			t.Errorf("%s est dans l allowlist (%q) mais ne cite plus %q : entree perimee, la "+
				"retirer dans le commit qui l a resolue.", rel, raison, nomDuForcageDuDecodage)
		}
	}
	if len(vus) > len(appelantsDuForcageDuDecodage) {
		noms := make([]string, 0, len(vus))
		for rel := range vus {
			noms = append(noms, rel)
		}
		sort.Strings(noms)
		t.Errorf("%d fichier(s) citent %q : %s", len(noms), nomDuForcageDuDecodage,
			strings.Join(noms, " "))
	}
}
