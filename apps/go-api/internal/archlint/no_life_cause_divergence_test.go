// Package archlint — no_life_cause_divergence_test.go : ratchet « LES VALEURS DE `end_cause` ET
// `named_by` NE DIVERGENT PAS » (lot 7C, 2026-09-07).
//
// # CE QU'IL EMPECHE
//
// Trois endroits nomment les memes valeurs, et ils ne peuvent pas partager une constante :
//
//	internal/games/halo_infinite/film/replay        les PRODUIT (CauseVie*, NomPar*) — paquet d'analyse pur ;
//	internal/persist                les VALIDE (CauseFin*, NommePar*) — `persist` ne doit pas
//	                                importer un paquet d'analyse pour verifier une colonne ;
//	internal/migration              les DOCUMENTE dans le DDL de `match_lives`.
//
// Une divergence ne casserait pas la compilation : le producteur ecrirait `film_end`, le
// validateur attendrait `filmend`, et la passe entiere serait REFUSEE a l'ecriture — un match
// sans faits d'isolement, sans autre symptome qu'une ligne d'erreur par film. Ce ratchet
// compare les deux jeux de litteraux et rougit a la premiere divergence.
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// sourcesDesCauses : les deux fichiers qui declarent les valeurs, et le prefixe de leurs
// constantes.
var sourcesDesCauses = []struct {
	fichier string
	prefixe string
}{
	{"internal/games/halo_infinite/film/replay/lives.go", "CauseVie"},
	{"internal/persist/lives_persister.go", "CauseFin"},
}

var sourcesDesNommages = []struct {
	fichier string
	prefixe string
}{
	// Le REPERTOIRE, pas le seul lives.go : le registre d'identite (E2, P2) declare ses voies
	// dans identity_registry_*.go, et le garde qui ne lisait que lives.go a laisse passer
	// quatre voies inconnues du persister (736 films refuses le 2026-09-09).
	{"internal/games/halo_infinite/film/replay", "NomPar"},
	{"internal/persist/lives_persister.go", "NommePar"},
}

// TestValeursDeFinDeVieNeDivergentPas — le ratchet, deux volets (les causes, les nommages).
//
// SELF-CHECK POSITIF : chaque source doit declarer AU MOINS une valeur, sinon le garde ne garde
// plus rien — c'est ce qui arriverait si les constantes etaient renommees ou deplacees.
func TestValeursDeFinDeVieNeDivergentPas(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	apiRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	comparer(t, apiRoot, "end_cause", sourcesDesCauses)
	comparer(t, apiRoot, "named_by", sourcesDesNommages)
}

// comparer lit les valeurs litterales declarees par chaque source et exige qu'elles coincident.
func comparer(t *testing.T, apiRoot, colonne string, sources []struct {
	fichier string
	prefixe string
},
) {
	t.Helper()
	var reference []string
	var refFichier string
	for _, src := range sources {
		vals := valeursDeclarees(t, filepath.Join(apiRoot, filepath.FromSlash(src.fichier)), src.prefixe)
		if len(vals) == 0 {
			t.Fatalf("%s : aucune constante %s* trouvee dans %s — le garde-rail de la colonne "+
				"%s ne verifie plus rien (constantes renommees ou deplacees ?)",
				colonne, src.prefixe, src.fichier, colonne)
		}
		if reference == nil {
			reference, refFichier = vals, src.fichier
			continue
		}
		if strings.Join(vals, ",") != strings.Join(reference, ",") {
			t.Errorf("colonne %s : %s declare %v, %s declare %v — une divergence fait REFUSER "+
				"toute la passe a l'ecriture, sans autre symptome qu'une ligne d'erreur par film",
				colonne, refFichier, reference, src.fichier, vals)
		}
	}
}

// motifConstante capture la valeur litterale d'une constante `Prefixe... = "valeur"`.
func motifConstante(prefixe string) *regexp.Regexp {
	// `const NomParX = "..."` sur une ligne ET `NomParX = "..."` dans un bloc const : les deux
	// formes coexistent dans le paquet replay (lives.go en bloc, identity_registry_*.go en ligne).
	return regexp.MustCompile(`(?m)^\s*(?:const\s+)?` + prefixe + `\w*\s*=\s*"([^"]+)"`)
}

// valeursDeclarees rend les valeurs litterales, triees (l'ordre de declaration n'est pas une
// propriete a figer, seul l'ENSEMBLE compte).
func valeursDeclarees(t *testing.T, chemin, prefixe string) []string {
	t.Helper()
	info, err := os.Stat(chemin)
	if err != nil {
		t.Fatalf("stat %s: %v", chemin, err)
	}
	var fichiers []string
	if info.IsDir() {
		entrees, err := os.ReadDir(chemin)
		if err != nil {
			t.Fatalf("lecture du repertoire %s: %v", chemin, err)
		}
		for _, e := range entrees {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
				continue
			}
			fichiers = append(fichiers, filepath.Join(chemin, e.Name()))
		}
	} else {
		fichiers = []string{chemin}
	}
	var out []string
	for _, f := range fichiers {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("lecture %s: %v", f, err)
		}
		for _, m := range motifConstante(prefixe).FindAllStringSubmatch(string(data), -1) {
			out = append(out, m[1])
		}
	}
	sort.Strings(out)
	return out
}
