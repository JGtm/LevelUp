package revision

// ordre_imports_test.go — L ORDRE DES IMPORTS DES SOURCES HACHEES EST UN GARDE-RAIL DE PREMIERE
// CLASSE, avec un message qui nomme sa cause.
//
// # POURQUOI
//
// Le hachage d une source porte sur son flux de JETONS, specifications d import comprises dans
// l ordre ou elles s ecrivent (cf. `cadre.go`, « L ORDRE DES IMPORTS N EST PAS DANS LE CADRE »).
// Un `goimports -w` ou un tri d imports d editeur qui remet une source en ordre « standard »
// change donc l empreinte de sa couche sans qu aucune ligne de decodage n ait bouge. Le gate de
// couche rougit, mais son message parle d un changement de sources : il ne nomme ni le fichier,
// ni l import deplace, ni le geste qui evite la regeneration. Ce test le fait avant lui.
//
// # LA REGLE QUE LES SOURCES SUIVENT (mesuree sur les sources hachees de chaque couche)
//
//   - GOFMT-STABLE, pour toutes : dans chaque groupe d imports (bloc separe par une ligne vide),
//     les chemins sont tries.
//   - ORDRE STANDARD, pour toutes sauf quinze : la bibliotheque standard, triee, puis le reste
//     (le module et les tiers), trie — ce que `goimports` rend a partir de l ensemble des
//     imports, quelle que soit la mise en page des blocs.
//   - ORDRE FIGE, pour les quinze de [ordresFiges] : l ordre qu avaient leurs imports sous le
//     chemin de module d avant `libs/film` (`levelup/go-api/...` se rangeait entre `cmp` et
//     `slices`). Elles le gardent en groupes separes par une ligne vide, chacun trie, pour que
//     gofmt ne les regroupe pas, et la table redit leur suite EXACTE : une suite qui devient
//     l ordre standard est une ERREUR, pas un progres (un outil a regroupe les imports, l empreinte
//     a change).

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// marqueDeLaBrique ouvre, dans [ordresFiges], un chemin d import de la brique : `@/types` est
// `<chemin du module>/types`, quel que soit le module qui porte la brique.
const marqueDeLaBrique = "@"

// ordresFiges : la suite d imports, a plat, des sources hachees qui ne suivent pas l ordre
// standard, relatives a la racine du module, en slash. Un fichier n entre ici que par un choix
// ecrit : sa mise en groupes est alors le seul garant de l empreinte.
//
// Critere de retrait (2026-11-08) : le jour ou les empreintes des cinq couches sont regenerees
// ensemble, a revision decidee, apres le changement de chemin de module, la table se vide et
// l ordre standard s applique partout.
var ordresFiges = map[string][]string{
	"internal/facts/killsource/hybrid.go": {
		"cmp",
		"@/types",
		"slices",
	},
	"internal/facts/objectives/awards.go": {
		"cmp",
		"@/types",
		"slices",
		"strings",
	},
	"internal/facts/objectives/flag_grabs_net.go": {
		"@/types",
		"time",
	},
	"internal/facts/objectives/named_series.go": {
		"cmp",
		"@/internal/constat",
		"@/types",
		"slices",
		"sort",
	},
	"internal/facts/objectives/round_bounds.go": {
		"cmp",
		"@/types",
		"slices",
		"sort",
	},
	"internal/facts/objectives/rounds_decision.go": {
		"@/types",
		"sort",
	},
	"internal/facts/objectives/score.go": {
		"cmp",
		"@/types",
		"slices",
	},
	"internal/grammar/biped_creation.go": {
		"cmp",
		"errors",
		"fmt",
		"@/types",
		"slices",
	},
	"internal/grammar/deaths_source.go": {
		"cmp",
		"errors",
		"fmt",
		"slices",
		"strings",
		"@/highlightevent",
		"@/finalise",
		"@/internal/source",
		"@/types",
	},
	"internal/grammar/grenade_events.go": {
		"@/internal/constat",
		"@/internal/profile",
		"@/internal/source",
		"slices",
	},
	"internal/grammar/navpoint_radial_rises.go": {
		"@/types",
		"slices",
	},
	"internal/grammar/offline_filters.go": {
		"@/types",
		"math",
		"slices",
		"sort",
	},
	"internal/grammar/player_table_record.go": {
		"@/types",
		"unicode/utf16",
	},
	"internal/grammar/quantize.go": {
		"@/internal/profile",
		"math/bits",
	},
	"internal/grammar/registry_fingerprint.go": {
		"hash",
		"hash/fnv",
		"@/internal/constat",
		"sync",
	},
}

// consequenceDeLOrdre : la fin commune des messages, qui dit pourquoi l ordre compte et les deux
// issues.
const consequenceDeLOrdre = "L'ordre des imports entre dans l'empreinte de la couche : remettre l'ordre en place " +
	"evite une regeneration d'empreinte. Si l'ordre a change volontairement, la revision de la " +
	"couche se decide et l'empreinte se regenere par la porte a deux verrous (revision.Porte), " +
	"et la table ordresFiges suit la decision."

// groupesDImports rend les chemins d import de la source, groupe par groupe : un groupe est un
// bloc de specifications contigues d une meme declaration, sans ligne vide entre elles.
func groupesDImports(nom string, src []byte) ([][]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, nom, src, parser.ImportsOnly|parser.ParseComments)
	if err != nil {
		return nil, err
	}
	var groupes [][]string
	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.IMPORT {
			continue
		}
		fin := -2
		for _, spec := range gen.Specs {
			imp := spec.(*ast.ImportSpec)
			chemin, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return nil, err
			}
			debut := fset.Position(imp.Pos()).Line
			if imp.Doc != nil {
				debut = fset.Position(imp.Doc.Pos()).Line
			}
			if debut > fin+1 {
				groupes = append(groupes, nil)
			}
			groupes[len(groupes)-1] = append(groupes[len(groupes)-1], chemin)
			fin = fset.Position(imp.End()).Line
		}
	}
	return groupes, nil
}

// estStandard : un chemin de la bibliotheque standard n a pas de point dans son premier element.
func estStandard(chemin string) bool {
	premier, _, _ := strings.Cut(chemin, "/")
	return !strings.Contains(premier, ".")
}

// aplati met les groupes bout a bout.
func aplati(groupes [][]string) []string {
	var out []string
	for _, g := range groupes {
		out = append(out, g...)
	}
	return out
}

// ordreStandard rend l ordre que goimports donne a cet ensemble d imports : la bibliotheque
// standard, triee, puis le reste, trie.
func ordreStandard(imports []string) []string {
	var std, autres []string
	for _, p := range imports {
		if estStandard(p) {
			std = append(std, p)
		} else {
			autres = append(autres, p)
		}
	}
	sort.Strings(std)
	sort.Strings(autres)
	return append(std, autres...)
}

// suiteFigee rend la suite de [ordresFiges] avec les chemins de la brique sous `brique`.
func suiteFigee(fige []string, brique string) []string {
	out := make([]string, len(fige))
	for i, p := range fige {
		if reste, ok := strings.CutPrefix(p, marqueDeLaBrique+"/"); ok {
			p = brique + "/" + reste
		}
		out[i] = p
	}
	return out
}

// premierEcart rend l index du premier import qui differe entre deux suites de meme longueur, ou -1.
func premierEcart(a, b []string) int {
	for i := range a {
		if a[i] != b[i] {
			return i
		}
	}
	return -1
}

// ecartDeGroupe rend le message du premier groupe dont les imports ne sont pas tries, ou "".
func ecartDeGroupe(rel string, groupes [][]string) string {
	for n, g := range groupes {
		for i := 1; i < len(g); i++ {
			if g[i-1] > g[i] {
				return fmt.Sprintf("%s : dans le groupe d'imports %d, l'import %q est place apres %q, "+
					"que gofmt rangerait apres lui. %s", rel, n+1, g[i], g[i-1], consequenceDeLOrdre)
			}
		}
	}
	return ""
}

// ecartDeLOrdreFige rend le message d une source de [ordresFiges] dont la suite d imports a
// bouge, ou "".
func ecartDeLOrdreFige(rel string, vu, fige []string) string {
	if len(vu) != len(fige) {
		return fmt.Sprintf("%s : %d imports, la table ordresFiges en fige %d : l'ensemble des imports a "+
			"change, la table et le golden de perimetre de la couche suivent. %s",
			rel, len(vu), len(fige), consequenceDeLOrdre)
	}
	if i := premierEcart(vu, fige); i >= 0 {
		return fmt.Sprintf("%s : l'import %q est a la place %d, l'ordre fige (ordresFiges) y met %q. %s",
			rel, vu[i], i+1, fige[i], consequenceDeLOrdre)
	}
	return ""
}

// diagnosticDeLOrdre rend le message d echec d une source (vide si son ordre est le bon).
// `fige` est sa suite dans [ordresFiges], nil pour une source en ordre standard.
func diagnosticDeLOrdre(rel string, groupes [][]string, fige []string, brique string) string {
	if msg := ecartDeGroupe(rel, groupes); msg != "" {
		return msg
	}
	vu := aplati(groupes)
	standard := ordreStandard(vu)
	if fige == nil {
		if i := premierEcart(vu, standard); i >= 0 {
			return fmt.Sprintf("%s : l'import %q est a la place %d, goimports le mettrait avant %q "+
				"(bibliotheque standard, puis le reste, chaque bloc trie). Si ce fichier doit garder un "+
				"ordre historique (revision/cadre.go), l'ajouter a ordresFiges, groupes separes par une "+
				"ligne vide ; sinon remettre l'ordre en place. %s",
				rel, vu[i], i+1, standard[i], consequenceDeLOrdre)
		}
		return ""
	}
	if premierEcart(vu, standard) < 0 {
		return fmt.Sprintf("%s : figure dans ordresFiges, mais ses imports sont en ordre standard : un "+
			"outil (goimports, tri d'imports d'editeur) les a regroupes. Remettre les groupes d'origine "+
			"(le module et ses voisins separes par une ligne vide, chacun trie). %s", rel, consequenceDeLOrdre)
	}
	return ecartDeLOrdreFige(rel, vu, suiteFigee(fige, brique))
}

// sourcesHachees rend les sources de production de tout ce que l empreinte d une couche lit :
// l arbre de chaque couche et les paquets qu elle importe. Cle : chemin relatif au module, en
// slash ; valeur : chemin sur le disque.
func sourcesHachees(t *testing.T, m Module) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, c := range CouchesRevisees() {
		p, err := m.Fermeture(c.Nom, CouchesRevisees())
		if err != nil {
			t.Fatalf("fermeture de la couche %s : %v", c.Nom, err)
		}
		for _, paquet := range p.Paquets {
			dossier := m.SurDisque(paquet)
			noms, err := sourcesDuDossier(dossier)
			if err != nil {
				t.Fatalf("sources de %s : %v", dossier, err)
			}
			for _, nom := range noms {
				abs := filepath.Join(dossier, nom)
				rel, err := filepath.Rel(m.Racine, abs)
				if err != nil {
					t.Fatalf("chemin relatif de %s : %v", abs, err)
				}
				out[filepath.ToSlash(rel)] = abs
			}
		}
	}
	return out
}

func TestOrdreDesImportsDesSourcesHachees(t *testing.T) {
	m, err := ModuleDe(".")
	if err != nil {
		t.Fatalf("module : %v", err)
	}
	sources := sourcesHachees(t, m)
	for rel := range ordresFiges {
		if _, ok := sources[rel]; !ok {
			t.Errorf("%s figure dans ordresFiges mais n'est plus une source hachee (fichier renomme, "+
				"deplace ou supprime) : mettre la table a jour en meme temps que le fichier, l'ordre de "+
				"ses imports n'entre plus dans l'empreinte", rel)
		}
	}
	rels := make([]string, 0, len(sources))
	for rel := range sources {
		rels = append(rels, rel)
	}
	sort.Strings(rels)
	for _, rel := range rels {
		src, err := os.ReadFile(sources[rel])
		if err != nil {
			t.Fatalf("lecture de %s : %v", rel, err)
		}
		groupes, err := groupesDImports(rel, src)
		if err != nil {
			t.Fatalf("imports de %s : %v", rel, err)
		}
		if msg := diagnosticDeLOrdre(rel, groupes, ordresFiges[rel], m.cheminBrique()); msg != "" {
			t.Error(msg)
		}
	}
}

// sourceAvecImports ecrit une source dont le bloc d imports est `bloc`.
func sourceAvecImports(bloc string) []byte {
	return []byte("package g\n\n" + bloc + "\n")
}

// TestOrdreDesImportsRougitSurUnOutil joue les gestes d un outil sur des sources synthetiques :
// un echange dans un groupe, le regroupement d une source figee, un import qui change de groupe,
// et un fichier neuf a la mise en page historique.
func TestOrdreDesImportsRougitSurUnOutil(t *testing.T) {
	const brique = "github.com/JGtm/LevelUp/libs/film"
	const mise = "import (\n\t\"cmp\"\n\n\t\"" + brique + "/types\"\n\n\t\"slices\"\n)"
	figee := []string{"cmp", marqueDeLaBrique + "/types", "slices"}
	cas := []struct {
		nom     string
		bloc    string
		fige    []string
		attendu []string // sous-chaines du message ; vide = aucun echec
	}{
		{"source figee intacte", mise, figee, nil},
		{"source standard intacte", "import (\n\t\"fmt\"\n\n\t\"" + brique + "/types\"\n)", nil, nil},
		{"deux imports echanges dans un groupe", "import (\n\t\"slices\"\n\t\"cmp\"\n)", nil,
			[]string{"g.go", `"cmp"`, `"slices"`, "evite une regeneration d'empreinte", "porte a deux verrous"}},
		{"source figee regroupee par un outil",
			"import (\n\t\"cmp\"\n\t\"slices\"\n\n\t\"" + brique + "/types\"\n)", figee,
			[]string{"g.go", "ordresFiges", "regroupes", "evite une regeneration d'empreinte"}},
		{"import figee passe dans le mauvais groupe",
			"import (\n\t\"" + brique + "/types\"\n\n\t\"cmp\"\n\t\"slices\"\n)", figee,
			[]string{"g.go", `"cmp"`, "ordre fige", "evite une regeneration d'empreinte"}},
		{"import ajoute a une source figee",
			"import (\n\t\"cmp\"\n\n\t\"" + brique + "/types\"\n\n\t\"slices\"\n\t\"sort\"\n)", figee,
			[]string{"g.go", "4 imports", "ordresFiges"}},
		{"source neuve a la mise en page historique", mise, nil,
			[]string{"g.go", `"slices"`, "goimports", "ordresFiges"}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			groupes, err := groupesDImports("g.go", sourceAvecImports(c.bloc))
			if err != nil {
				t.Fatalf("imports : %v", err)
			}
			msg := diagnosticDeLOrdre("g.go", groupes, c.fige, brique)
			if len(c.attendu) == 0 {
				if msg != "" {
					t.Fatalf("aucun echec attendu, message : %s", msg)
				}
				return
			}
			if msg == "" {
				t.Fatal("l'ordre errone n'a pas rougi")
			}
			for _, morceau := range c.attendu {
				if !strings.Contains(msg, morceau) {
					t.Errorf("le message ne contient pas %q : %s", morceau, msg)
				}
			}
		})
	}
}

// TestGroupesDImportsLitLesBlocsEtLesDeclarationsSeules : une declaration `import "x"` est son
// propre groupe, une ligne vide separe deux groupes, un commentaire de tete reste dans le sien.
func TestGroupesDImportsLitLesBlocsEtLesDeclarationsSeules(t *testing.T) {
	src := sourceAvecImports("import \"b\"\nimport \"a\"\n\nimport (\n\t// note\n\t\"d\"\n\t\"e\"\n\n\t\"c\"\n)")
	groupes, err := groupesDImports("g.go", src)
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprint(groupes); got != "[[b] [a] [d e] [c]]" {
		t.Fatalf("groupes = %s", got)
	}
}
