package archlint

// aides_uniques_film_test.go — DEUX AIDES CANONIQUES ET LEURS GARDE-RAILS (regle 6 du depot ; revue
// D1.4.6 du plan `.ai/PLAN_RI_ETAT_COMPLET_IMAGES_CLES_2026-10-08.md`, constat 7, 2026-10-09).
//
//  1. L ENSEMBLE DES FAMILLES D ARME CONNUES se construit par `weaponv3.FamillesConnues` et nulle part
//     ailleurs : il en existait cinq copies (`range weaponv3.KnownWeaponHigh32Copie()` puis
//     `m[f] = true`), production et tests compris.
//  2. LA CONVERSION D UN BOOLEEN EN 1 OU 0 existe au plus UNE fois par paquet et par type d entier
//     rendu, sous le paquet du film : six copies en `grammar` (`b2i`, `unSi`, `boolToInt`, `t526Un`,
//     `s3B`, `bit2i`), deux en `uint64` (`bit2u`, `b2u`), trois en `replay` (`unSiVrai`, `ctBit`,
//     `objBool`). Elle se reconnait a sa FORME (un parametre booleen, un entier rendu, `if b { return
//     1 }; return 0`), pas a son nom : un nom neuf ne la fait pas passer.
//
// Les deux tests lisent tous les fichiers Go, tests et instruments `research` compris.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// fichiersGo parcourt les fichiers Go sous `racine` (hors testdata et dossiers caches).
func fichiersGo(t *testing.T, racine string, f func(rel string, data []byte)) {
	t.Helper()
	base := racineGoAPI(t)
	err := filepath.WalkDir(racine, func(chemin string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if chemin != racine && (d.Name() == "testdata" || strings.HasPrefix(d.Name(), ".")) {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(chemin, ".go") {
			return nil
		}
		data, rerr := os.ReadFile(chemin) //nolint:gosec // source du module, lecture seule
		if rerr != nil {
			return rerr
		}
		rel, _ := filepath.Rel(base, chemin)
		f(filepath.ToSlash(rel), data)
		return nil
	})
	if err != nil {
		t.Fatalf("parcours : %v", err)
	}
}

// ensembleDesFamillesRE : un `m[f] = true` dans les quatre lignes qui suivent un appel de
// `KnownWeaponHigh32Copie()` — la forme des copies retirees.
var ensembleDesFamillesRE = regexp.MustCompile(`\[\w+\]\s*=\s*true`)

const maisonDesFamillesConnues = "internal/games/halo_infinite/film/internal/grammar/weaponv3/canon.go"

func TestLEnsembleDesFamillesConnuesSeConstruitUneFois(t *testing.T) {
	var violations []string
	maison := false
	fichiersGo(t, racineGoAPI(t), func(rel string, data []byte) {
		lignes := strings.Split(string(data), "\n")
		if rel == maisonDesFamillesConnues {
			maison = strings.Contains(string(data), "func FamillesConnues() map[uint32]bool")
			return
		}
		for i, l := range lignes {
			if strings.HasPrefix(strings.TrimSpace(l), "//") || !strings.Contains(l, "KnownWeaponHigh32Copie()") {
				continue
			}
			for j := i; j < len(lignes) && j <= i+4; j++ {
				if ensembleDesFamillesRE.MatchString(lignes[j]) {
					violations = append(violations, rel+":"+itoa(i+1))
					break
				}
			}
		}
	})
	if !maison {
		t.Errorf("%s ne porte plus `FamillesConnues` : le garde-rail ne garde plus rien", maisonDesFamillesConnues)
	}
	if len(violations) > 0 {
		t.Errorf("ensemble des familles connues reconstruit hors de weaponv3.FamillesConnues :\n  %s",
			strings.Join(violations, "\n  "))
	}
}

func TestUneSeuleConversionBooleenneParPaquet(t *testing.T) {
	racine := filepath.Join(racineGoAPI(t), "internal", "games", "halo_infinite", "film")
	parGroupe := map[string][]string{}
	fichiersGo(t, racine, func(rel string, data []byte) {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, rel, data, parser.SkipObjectResolution)
		if err != nil {
			t.Errorf("%s : %v", rel, err)
			return
		}
		paquet := strings.TrimSuffix(f.Name.Name, "_test")
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if typ, ok := conversionBooleenne(fn); ok {
				cle := filepath.ToSlash(filepath.Dir(rel)) + " (" + paquet + ", " + typ + ")"
				parGroupe[cle] = append(parGroupe[cle], fn.Name.Name+" "+rel)
			}
		}
	})
	if len(parGroupe) == 0 {
		t.Fatal("aucune conversion booleenne trouvee : le detecteur ne mord plus")
	}
	var violations []string
	for cle, fns := range parGroupe {
		if len(fns) > 1 {
			sort.Strings(fns)
			violations = append(violations, cle+" : "+strings.Join(fns, ", "))
		}
	}
	sort.Strings(violations)
	if len(violations) > 0 {
		t.Errorf("plus d une conversion booleen -> 1 ou 0 dans un meme paquet et pour un meme type "+
			"(regle 6 : appeler l aide du paquet) :\n  %s", strings.Join(violations, "\n  "))
	}
}

// conversionBooleenne dit si `fn` est une fonction `func x(b bool) T { if b { return 1 }; return 0 }`
// avec T entier, et rend T.
func conversionBooleenne(fn *ast.FuncDecl) (string, bool) {
	if fn.Recv != nil || fn.Body == nil || len(fn.Type.Params.List) != 1 || len(fn.Type.Params.List[0].Names) != 1 ||
		fn.Type.Results == nil || len(fn.Type.Results.List) != 1 || len(fn.Body.List) != 2 {
		return "", false
	}
	if p, ok := fn.Type.Params.List[0].Type.(*ast.Ident); !ok || p.Name != "bool" {
		return "", false
	}
	res, ok := fn.Type.Results.List[0].Type.(*ast.Ident)
	if !ok || !strings.Contains(" int int8 int16 int32 int64 uint uint8 uint16 uint32 uint64 ", " "+res.Name+" ") {
		return "", false
	}
	si, ok := fn.Body.List[0].(*ast.IfStmt)
	if !ok || si.Else != nil || len(si.Body.List) != 1 {
		return "", false
	}
	if c, ok := si.Cond.(*ast.Ident); !ok || c.Name != fn.Type.Params.List[0].Names[0].Name {
		return "", false
	}
	if !retourneLitteral(si.Body.List[0], "1") || !retourneLitteral(fn.Body.List[1], "0") {
		return "", false
	}
	return res.Name, true
}

// retourneLitteral dit si `s` est `return <lit>`.
func retourneLitteral(s ast.Stmt, lit string) bool {
	r, ok := s.(*ast.ReturnStmt)
	if !ok || len(r.Results) != 1 {
		return false
	}
	b, ok := r.Results[0].(*ast.BasicLit)
	return ok && b.Value == lit
}
