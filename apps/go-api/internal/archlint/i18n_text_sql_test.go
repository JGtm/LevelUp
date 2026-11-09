// Package archlint — i18n_text_sql_test.go : garde-rail i18n typé (ADR 0040, plan i18n 5.7).
//
// Un `i18n.Text` est un `map[Locale]string` : ni une valeur SQL, ni une cible de `Scan`. Les
// API de base de données prennent des `any` (`rows.Scan(&x.Label)`, `db.Exec(q, x.Label)`,
// `appender.AppendRow(x.Label)`), donc le compilateur laisse passer un `Text` et l'erreur
// n'apparaît qu'à l'exécution, sur le chemin qui exécute la ligne. Le pont correct est
// colonne par colonne : deux variables `sql.NullString` / `string` puis `i18n.FromColumns`
// à la lecture, `Text.Ptr(l)` ou `Text[l]` à l'écriture.
//
// Le test charge les paquets du module AVEC leurs types (`golang.org/x/tools/go/packages`),
// deux fois (tags par défaut, puis `integration`), tests compris, et signale tout argument de
// type `i18n.Text` (ou `*i18n.Text`, ou type nommé de même type sous-jacent) passé à :
//
//   - `Scan`, `Exec`, `ExecContext`, `Query`, `QueryContext`, `QueryRow`, `QueryRowContext`,
//     `AppendRow`, partout dans le module ;
//   - toute fonction ou méthode déclarée sous `platform/duckdb`, `persist`, `migration`,
//     `sync`, `ops`, `cmd` dont le dernier paramètre est variadique `...any` (arguments
//     SQL transmis plus loin : `execArgs`, `queryRows`...) ;
//   - dans ces mêmes dossiers, tout élément `Text` d'un littéral `[]any{...}` ou d'un
//     `append(args, ...)` ou affectation `args[i] = ...` sur un `[]any` (le tableau d'arguments construit puis passé en
//     `args...`, que le contrôle d'appel ne voit pas).
//
// Pas d'allowlist, par décision (plan i18n 5.7) : un cas trouvé se corrige, il ne s'excuse pas.
package archlint

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"golang.org/x/tools/go/packages"
)

// textSQLModule est le chemin du module (go.mod) ; textSQLI18nPath, le paquet de `Text`.
const (
	textSQLModule   = "levelup/go-api"
	textSQLI18nPath = textSQLModule + "/internal/i18n"
)

// textSQLNomsAppel : méthodes dont tout argument `Text` est une faute, quel que soit le
// receveur (`*sql.Rows`, `*sql.DB`, `*sql.Tx`, `*sql.Conn`, l'appender DuckDB, les ports qui les enveloppent).
var textSQLNomsAppel = map[string]bool{
	"Scan": true, "Exec": true, "ExecContext": true,
	"Query": true, "QueryContext": true,
	"QueryRow": true, "QueryRowContext": true,
	"AppendRow": true,
}

// textSQLDossiers : sous-arbres (relatifs au module) où une fonction à `...any` est un
// passage d'arguments SQL, et où les tableaux `[]any` sont contrôlés.
var textSQLDossiers = []string{
	"internal/platform/duckdb", "internal/persist", "internal/migration",
	"internal/sync", "internal/ops", "cmd",
}

// textSQLFaute est un argument `Text` posé là où une valeur SQL est attendue.
type textSQLFaute struct {
	Pos   string // fichier relatif au module :ligne
	Cible string
	Type  string
}

func (f textSQLFaute) String() string {
	return fmt.Sprintf("%s  %s  (argument de type %s)", f.Pos, f.Cible, f.Type)
}

// estTextI18n dit si t est `i18n.Text`, un pointeur vers lui, ou un type nommé dont le type
// sous-jacent est `map[i18n.Locale]string`.
func estTextI18n(t types.Type) bool {
	for {
		p, ok := t.(*types.Pointer)
		if !ok {
			break
		}
		t = p.Elem()
	}
	m, ok := t.Underlying().(*types.Map)
	if !ok {
		return false
	}
	k, ok := m.Key().(*types.Named)
	if !ok || k.Obj().Pkg() == nil || k.Obj().Pkg().Path() != textSQLI18nPath || k.Obj().Name() != "Locale" {
		return false
	}
	e, ok := m.Elem().(*types.Basic)
	return ok && e.Kind() == types.String
}

// estVariadiqueAny dit si sig se termine par `...any` (ou `...driver.Value`, alias de any).
func estVariadiqueAny(sig *types.Signature) bool {
	if !sig.Variadic() || sig.Params().Len() == 0 {
		return false
	}
	last := sig.Params().At(sig.Params().Len() - 1).Type()
	sl, ok := last.(*types.Slice)
	if !ok {
		return false
	}
	it, ok := sl.Elem().Underlying().(*types.Interface)
	return ok && it.Empty()
}

// estTableauAny dit si t est `[]any` (ou `[]interface{}`, `[]driver.Value`).
func estTableauAny(t types.Type) bool {
	sl, ok := t.Underlying().(*types.Slice)
	if !ok {
		return false
	}
	it, ok := sl.Elem().Underlying().(*types.Interface)
	return ok && it.Empty()
}

// dansDossiers dit si le chemin de paquet pkgPath est sous un des dossiers (relatifs au module).
func dansDossiers(pkgPath string, dossiers []string) bool {
	rel := strings.TrimSuffix(strings.TrimPrefix(pkgPath, textSQLModule+"/"), "_test")
	for _, d := range dossiers {
		if rel == d || strings.HasPrefix(rel, d+"/") {
			return true
		}
	}
	return false
}

// calleeObjet rend l'objet appelé (fonction ou méthode) d'un appel, ou nil (conversion,
// fermeture, valeur de fonction).
func calleeObjet(info *types.Info, call *ast.CallExpr) types.Object {
	switch f := ast.Unparen(call.Fun).(type) {
	case *ast.Ident:
		return info.Uses[f]
	case *ast.SelectorExpr:
		if sel := info.Selections[f]; sel != nil {
			return sel.Obj()
		}
		return info.Uses[f.Sel]
	}
	return nil
}

// fautesDuPaquet applique les trois règles à un paquet du module, types en main.
func fautesDuPaquet(pkg *packages.Package, racine string) []textSQLFaute {
	var out []textSQLFaute
	dansPerimetre := dansDossiers(strings.SplitN(pkg.PkgPath, " [", 2)[0], textSQLDossiers)
	rel := func(p token.Pos) string {
		pos := pkg.Fset.Position(p)
		f := strings.TrimPrefix(strings.ReplaceAll(pos.Filename, "\\", "/"), racine+"/")
		return fmt.Sprintf("%s:%d", f, pos.Line)
	}
	signale := func(p token.Pos, cible string, t types.Type) {
		out = append(out, textSQLFaute{Pos: rel(p), Cible: cible, Type: t.String()})
	}
	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.CallExpr:
				fautesAppel(pkg.TypesInfo, x, dansPerimetre, signale)
			case *ast.AssignStmt:
				if dansPerimetre {
					fautesAffectation(pkg.TypesInfo, x, signale)
				}
			case *ast.CompositeLit:
				if dansPerimetre {
					if tv, ok := pkg.TypesInfo.Types[x]; ok && estTableauAny(tv.Type) {
						for _, e := range x.Elts {
							if t := pkg.TypesInfo.TypeOf(e); t != nil && estTextI18n(t) {
								signale(e.Pos(), "élément de []any", t)
							}
						}
					}
				}
			}
			return true
		})
	}
	return out
}

// fautesAffectation contrôle `args[i] = texte` sur un `[]any`.
func fautesAffectation(info *types.Info, as *ast.AssignStmt, signale func(token.Pos, string, types.Type)) {
	for i, l := range as.Lhs {
		ix, ok := l.(*ast.IndexExpr)
		if !ok || i >= len(as.Rhs) {
			continue
		}
		if t := info.TypeOf(ix.X); t == nil || !estTableauAny(t) {
			continue
		}
		if rt := info.TypeOf(as.Rhs[i]); rt != nil && estTextI18n(rt) {
			signale(as.Rhs[i].Pos(), "affectation dans []any", rt)
		}
	}
}

// fautesAppel contrôle un appel : méthodes SQL nommées, fonctions `...any` du périmètre, `append` sur `[]any`.
func fautesAppel(info *types.Info, call *ast.CallExpr, dansPerimetre bool, signale func(token.Pos, string, types.Type)) {
	// append(args, ..., texte) sur un []any, dans le périmètre.
	if id, ok := ast.Unparen(call.Fun).(*ast.Ident); ok && id.Name == "append" && dansPerimetre {
		if _, builtin := info.Uses[id].(*types.Builtin); builtin && len(call.Args) > 0 {
			if t := info.TypeOf(call.Args[0]); t != nil && estTableauAny(t) {
				for _, a := range call.Args[1:] {
					if at := info.TypeOf(a); at != nil && estTextI18n(at) {
						signale(a.Pos(), "append sur []any", at)
					}
				}
			}
		}
		return
	}
	obj := calleeObjet(info, call)
	fn, ok := obj.(*types.Func)
	if !ok {
		return
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return
	}
	nomme := textSQLNomsAppel[fn.Name()]
	variadique := fn.Pkg() != nil && dansDossiers(fn.Pkg().Path(), textSQLDossiers) && estVariadiqueAny(sig)
	if !nomme && !variadique {
		return
	}
	for i, a := range call.Args {
		// L'argument étalé (`args...`) est un tableau : ses éléments sont contrôlés à leur construction.
		if call.Ellipsis.IsValid() && i == len(call.Args)-1 {
			continue
		}
		if t := info.TypeOf(a); t != nil && estTextI18n(t) {
			signale(a.Pos(), "appel de "+fn.Name(), t)
		}
	}
}

// horsModuleSource dit si le fichier vient de la bibliothèque standard ou du cache des modules : seul
// ce code voit ses corps de fonction retirés. Les fichiers générés par cgo (dans le cache de
// compilation) restent entiers : ils portent le code des paquets du module qui utilisent cgo.
func horsModuleSource(filename string) bool {
	f := strings.ReplaceAll(filename, "\\", "/")
	return strings.HasPrefix(f, strings.ReplaceAll(runtime.GOROOT(), "\\", "/")+"/") || strings.Contains(f, "/pkg/mod/")
}

// chargerPaquetsTypes charge les paquets du module avec leurs types sous les tags donnés. Les
// dépendances hors module sont typées depuis leurs déclarations seules (corps de fonction
// retirés), ce qui évite d'analyser le corps de la bibliothèque standard et de DuckDB.
func chargerPaquetsTypes(t *testing.T, racine string, tags string) []*packages.Package {
	t.Helper()
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports |
			packages.NeedDeps | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		Dir:   racine,
		Tests: true,
		ParseFile: func(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
			f, err := parser.ParseFile(fset, filename, src, parser.SkipObjectResolution)
			if err != nil {
				return f, err
			}
			if horsModuleSource(filename) {
				for _, d := range f.Decls {
					if fd, ok := d.(*ast.FuncDecl); ok {
						fd.Body = nil
					}
				}
			}
			return f, nil
		},
	}
	if tags != "" {
		cfg.BuildFlags = []string{"-tags=" + tags}
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		t.Fatalf("chargement des paquets (tags %q) : %v", tags, err)
	}
	return pkgs
}

// TestI18nTextJamaisArgumentSQL : voir l'en-tête du fichier.
func TestI18nTextJamaisArgumentSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("charge tous les paquets du module avec leurs types : hors -short")
	}
	racine := strings.ReplaceAll(apiModuleRoot(t), "\\", "/")
	vues := map[string]textSQLFaute{}
	analyses := 0
	textTrouve := false
	for _, tags := range []string{"", "integration"} {
		debut := time.Now()
		pkgs := chargerPaquetsTypes(t, racine, tags)
		nb := 0
		for _, pkg := range pkgs {
			if !strings.HasPrefix(pkg.PkgPath, textSQLModule) || strings.HasSuffix(pkg.PkgPath, ".test") {
				continue
			}
			for _, e := range pkg.Errors {
				t.Errorf("paquet %s (tags %q) : %v", pkg.ID, tags, e)
			}
			if pkg.TypesInfo == nil {
				t.Errorf("paquet %s (tags %q) : sans informations de types", pkg.ID, tags)
				continue
			}
			nb++
			if pkg.PkgPath == textSQLI18nPath && tags == "" && pkg.Types != nil {
				// Auto-preuve : le prédicat reconnaît le vrai type, sous les trois formes.
				obj := pkg.Types.Scope().Lookup("Text")
				if obj == nil || !estTextI18n(obj.Type()) || !estTextI18n(types.NewPointer(obj.Type())) {
					t.Errorf("estTextI18n ne reconnaît plus i18n.Text : le garde-rail ne garde plus rien")
				}
				textTrouve = true
			}
			for _, f := range fautesDuPaquet(pkg, racine) {
				vues[f.Pos+"|"+f.Cible] = f
			}
		}
		analyses += nb
		t.Logf("tags %q : %d paquets du module analysés en %s", tags, nb, time.Since(debut).Round(time.Millisecond))
	}
	if !textTrouve {
		t.Fatal("le paquet internal/i18n (type Text) n'a pas été chargé")
	}
	if analyses < 100 {
		t.Fatalf("seulement %d paquets analysés : le chargement typé ne voit plus le module", analyses)
	}
	var restantes []string
	for _, f := range vues {
		restantes = append(restantes, f.String())
	}
	sort.Strings(restantes)
	if len(restantes) > 0 {
		t.Errorf("%d argument(s) i18n.Text passé(s) à une API de base de données (compile, échoue à l'exécution) ;\n"+
			"lire/écrire colonne par colonne (deux variables puis i18n.FromColumns, ou Text.Ptr(l) / Text[l]) :\n  %s",
			len(restantes), strings.Join(restantes, "\n  "))
	}
}
