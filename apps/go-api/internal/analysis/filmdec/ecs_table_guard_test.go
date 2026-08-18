package filmdec

// Garde-rails de la table ECS (`testdata/ecs_table.tsv`), la reference versionnee de la
// grammaire archetype x composant du film. Trois controles independants :
//
//	G1 code <-> table   : toute etiquette `case` de consumeByName est dans la table avec le
//	                      statut que l AST lui donne, et reciproquement. Toujours joue.
//	G2 film <-> table   : archetypes, composants ET niveaux de la table = ceux du registre
//	                      (chunk_00) des films temoins. Garde ECS_TABLE_FILM, SKIP sans film.
//	G3 table <-> doc    : chaque champ cite en `doc_field` existe dans replay/document.go.
//
// Le lecteur du TSV vit ici et non dans un fichier de production : la table n a aucun
// lecteur applicatif, un `ecs_table.go` serait du code mort.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const ecsTablePath = "testdata/ecs_table.tsv"

// ecsTableFilmEnv : liste de repertoires de film (chunk_00.bin), separes par `;` ou `,`.
const ecsTableFilmEnv = "ECS_TABLE_FILM"

// ecsRow est une ligne de la table. Les colonnes de prose ne servent a aucun controle.
type ecsRow struct {
	TI, I      int
	Component  string
	Level      uint32
	Status     string
	CodeSource string
	DocField   string
	Notes      string
	LineNo     int
}

const ecsTableColumns = 16

// loadECSTable lit le TSV et verifie sa forme (en-tete, 16 colonnes, tri, unicite).
func loadECSTable(t *testing.T) []ecsRow {
	t.Helper()
	raw, err := os.ReadFile(ecsTablePath)
	if err != nil {
		t.Fatalf("table ECS illisible : %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if len(lines) < 2 {
		t.Fatal("table ECS vide")
	}
	head := strings.Split(lines[0], "\t")
	if len(head) != ecsTableColumns || head[0] != "ti" || head[3] != "component" || head[5] != "status" {
		t.Fatalf("en-tete inattendu : %q", lines[0])
	}
	var out []ecsRow
	seen := map[string]int{}
	for n, l := range lines[1:] {
		c := strings.Split(l, "\t")
		if len(c) != ecsTableColumns {
			t.Fatalf("ligne %d : %d colonnes, 16 attendues", n+2, len(c))
		}
		ti, err1 := strconv.Atoi(c[0])
		i, err2 := strconv.Atoi(c[2])
		lv, err3 := strconv.ParseInt(c[4], 10, 64)
		if err1 != nil || err2 != nil || err3 != nil {
			t.Fatalf("ligne %d : ti/i/level non numeriques (%q %q %q)", n+2, c[0], c[2], c[4])
		}
		key := c[0] + "|" + c[2] + "|" + c[3]
		if ti >= 0 {
			if prev, dup := seen[key]; dup {
				t.Fatalf("ligne %d : doublon de (ti,i) avec la ligne %d", n+2, prev)
			}
			seen[key] = n + 2
		}
		out = append(out, ecsRow{TI: ti, I: i, Component: c[3], Level: uint32(lv), Status: c[5],
			CodeSource: c[9], DocField: c[10], Notes: c[15], LineNo: n + 2})
	}
	for k := 1; k < len(out); k++ {
		if out[k-1].TI > out[k].TI || (out[k-1].TI == out[k].TI && out[k-1].I > out[k].I) {
			t.Fatalf("ligne %d : table non triee par (ti, i)", out[k].LineNo)
		}
	}
	return out
}

// ecsCase decrit le cas de consumeByName qui traite un composant.
type ecsCase struct {
	Kind string // porte | partiel
	Line int
}

// scanConsumeByNameCases rend, par nom de composant, le cas du switch qui le traite.
// `porte` = tous les retours du cas rendent le litteral `true` ; `partiel` sinon (retour
// data-dependant ou garde par un drapeau : le traverseur peut desynchroniser proprement).
func scanConsumeByNameCases(t *testing.T) map[string]ecsCase {
	t.Helper()
	fset := token.NewFileSet()
	ents, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	consts := map[string]string{}
	var files []*ast.File
	for _, e := range ents {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(".", n), nil, 0)
		if err != nil {
			t.Fatalf("%s : %v", n, err)
		}
		files = append(files, f)
	}
	var fn *ast.FuncDecl
	for _, f := range files {
		for _, d := range f.Decls {
			switch v := d.(type) {
			case *ast.GenDecl:
				if v.Tok == token.CONST {
					collectStringConsts(v, consts)
				}
			case *ast.FuncDecl:
				if v.Name.Name == "consumeByName" {
					fn = v
				}
			}
		}
	}
	if fn == nil {
		t.Fatal("consumeByName introuvable dans le paquet")
	}
	return caseKinds(fn, fset, consts)
}

func collectStringConsts(gd *ast.GenDecl, out map[string]string) {
	for _, s := range gd.Specs {
		vs, ok := s.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for i, nm := range vs.Names {
			if i >= len(vs.Values) {
				continue
			}
			if bl, ok := vs.Values[i].(*ast.BasicLit); ok && bl.Kind == token.STRING {
				if v, err := strconv.Unquote(bl.Value); err == nil {
					out[nm.Name] = v
				}
			}
		}
	}
}

func caseKinds(fn *ast.FuncDecl, fset *token.FileSet, consts map[string]string) map[string]ecsCase {
	out := map[string]ecsCase{}
	ast.Inspect(fn, func(n ast.Node) bool {
		cc, ok := n.(*ast.CaseClause)
		if !ok || cc.List == nil { // `default` a une List nil
			return true
		}
		kind := "porte"
		ast.Inspect(cc, func(m ast.Node) bool {
			rs, ok := m.(*ast.ReturnStmt)
			if !ok || len(rs.Results) != 3 {
				return true
			}
			if id, ok := rs.Results[2].(*ast.Ident); !ok || id.Name != "true" {
				kind = "partiel"
			}
			return true
		})
		line := fset.Position(cc.Pos()).Line
		for _, e := range cc.List {
			var name string
			switch v := e.(type) {
			case *ast.BasicLit:
				name, _ = strconv.Unquote(v.Value)
			case *ast.Ident:
				name = consts[v.Name]
			}
			if name != "" {
				out[name] = ecsCase{Kind: kind, Line: line}
			}
		}
		return true
	})
	return out
}

// TestG1TableSuitLeCode : la table et consumeByName ne peuvent plus diverger en silence.
func TestG1TableSuitLeCode(t *testing.T) {
	rows := loadECSTable(t)
	cases := scanConsumeByNameCases(t)

	byName := map[string][]ecsRow{}
	for _, r := range rows {
		byName[r.Component] = append(byName[r.Component], r)
	}
	// sens 1 : tout `case` du code est dans la table, avec le bon statut.
	for name, c := range cases {
		list, ok := byName[name]
		if !ok {
			t.Errorf("G1 : `case %q` (traverse.go:%d) n a AUCUNE ligne dans la table — porter un composant sans mettre la table a jour est interdit", name, c.Line)
			continue
		}
		for _, r := range list {
			want := c.Kind
			if r.TI < 0 {
				want = "alias"
			}
			if r.Status != want {
				t.Errorf("G1 : ligne %d (ti=%d i=%d %s) statut %q, le code dit %q (traverse.go:%d)", r.LineNo, r.TI, r.I, name, r.Status, want, c.Line)
			}
		}
	}
	// sens 2 : tout statut de la table correspond a un etat reel du code.
	for _, r := range rows {
		_, isCase := cases[r.Component]
		switch r.Status {
		case "porte", "partiel", "alias":
			if !isCase {
				t.Errorf("G1 : ligne %d (%s) porte le statut %q mais consumeByName ne la traite PAS", r.LineNo, r.Component, r.Status)
			}
		case "non_porte", "deser_non_cable":
			if isCase {
				t.Errorf("G1 : ligne %d (%s) est declaree %q mais consumeByName la traite (traverse.go:%d)", r.LineNo, r.Component, r.Status, cases[r.Component].Line)
			}
		default:
			t.Errorf("G1 : ligne %d : statut inconnu %q", r.LineNo, r.Status)
		}
		checkCodeSource(t, r)
		checkAliasTarget(t, r, byName)
	}
}

// checkCodeSource : la source `fichier:ligne` designe un fichier du paquet et une ligne qui existe.
func checkCodeSource(t *testing.T, r ecsRow) {
	t.Helper()
	if r.CodeSource == "" {
		if r.Status == "porte" || r.Status == "partiel" || r.Status == "deser_non_cable" {
			t.Errorf("G1 : ligne %d (%s) est %q sans source fichier:ligne", r.LineNo, r.Component, r.Status)
		}
		return
	}
	file, num, ok := strings.Cut(r.CodeSource, ":")
	if !ok {
		t.Errorf("G1 : ligne %d : source %q n est pas `fichier.go:ligne`", r.LineNo, r.CodeSource)
		return
	}
	n, err := strconv.Atoi(num)
	if err != nil {
		t.Errorf("G1 : ligne %d : source %q sans numero de ligne", r.LineNo, r.CodeSource)
		return
	}
	b, err := os.ReadFile(file)
	if err != nil {
		t.Errorf("G1 : ligne %d : source %q introuvable (%v)", r.LineNo, r.CodeSource, err)
		return
	}
	src := strings.Split(string(b), "\n")
	if n < 1 || n > len(src) {
		t.Errorf("G1 : ligne %d : %s n a que %d lignes", r.LineNo, file, len(src))
		return
	}
	if r.Status == "deser_non_cable" && !strings.Contains(string(b), r.Component) {
		t.Errorf("G1 : ligne %d : %s est declare deser_non_cable mais son nom n apparait pas dans %s", r.LineNo, r.Component, file)
	}
}

// checkAliasTarget : une ligne `alias` doit designer un composant reel de la table.
func checkAliasTarget(t *testing.T, r ecsRow, byName map[string][]ecsRow) {
	t.Helper()
	if r.Status != "alias" {
		return
	}
	target := strings.TrimPrefix(r.Notes, "alias de ")
	if target == r.Notes || target == "" {
		t.Errorf("G1 : ligne %d : alias sans cible (`notes` = %q)", r.LineNo, r.Notes)
		return
	}
	for _, o := range byName[target] {
		if o.TI >= 0 {
			return
		}
	}
	t.Errorf("G1 : ligne %d : l alias %s vise %q, absent du registre de la table", r.LineNo, r.Component, target)
}

// TestG2TableSuitLeRegistreDuFilm : la table est bien celle que le FILM declare.
func TestG2TableSuitLeRegistreDuFilm(t *testing.T) {
	dirs := ecsTableFilmDirs()
	if len(dirs) == 0 {
		t.Skipf("%s absent : aucun film pour confronter la table au registre", ecsTableFilmEnv)
	}
	rows := loadECSTable(t)
	want := map[string]uint32{}
	nAlias := 0
	for _, r := range rows {
		if r.TI < 0 {
			nAlias++
			continue
		}
		want[strconv.Itoa(r.TI)+"|"+strconv.Itoa(r.I)+"|"+r.Component] = r.Level
	}
	for _, dir := range dirs {
		raw, err := os.ReadFile(filepath.Join(dir, "chunk_00.bin"))
		if err != nil {
			t.Fatalf("%s : chunk_00 illisible (%v)", dir, err)
		}
		reg, err := ParseRegistryChunk(raw)
		if err != nil {
			t.Fatalf("%s : %v", dir, err)
		}
		got := map[string]uint32{}
		porteurs := 0
		for _, a := range reg.Archetypes {
			if len(a.Components) > 0 {
				porteurs++
			}
			for i, c := range a.Components {
				got[strconv.Itoa(a.Index)+"|"+strconv.Itoa(i)+"|"+c] = a.Flags[i]
			}
		}
		for k, lv := range got {
			w, ok := want[k]
			if !ok {
				t.Errorf("G2 [%s] : le film declare %q, absent de la table", filepath.Base(dir), k)
				continue
			}
			if w != lv {
				t.Errorf("G2 [%s] : %q niveau %d dans la table, %d dans le film", filepath.Base(dir), k, w, lv)
			}
		}
		for k := range want {
			if _, ok := got[k]; !ok {
				t.Errorf("G2 [%s] : la table porte %q, absent du registre du film", filepath.Base(dir), k)
			}
		}
		t.Logf("G2 [%s] : %d blocs, %d porteurs, %d lignes de registre confrontees a %d lignes de table (+%d alias)",
			filepath.Base(dir), len(reg.Archetypes), porteurs, len(got), len(want), nAlias)
	}
}

func ecsTableFilmDirs() []string {
	v := os.Getenv(ecsTableFilmEnv)
	if v == "" {
		return nil
	}
	var out []string
	for _, p := range strings.FieldsFunc(v, func(r rune) bool { return r == ';' || r == ',' }) {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// TestG3TableSuitLeDocument : chaque champ cite en `doc_field` existe dans le document de rejeu.
//
// LE PAQUET ENTIER, PAS UN SEUL FICHIER (2026-08-18, lot E phase 1). La garde lisait
// `document.go` seul, alors que la forme publiee est repartie sur plusieurs fichiers depuis que
// les gros calques ont leur chronique a part (`document_score.go`, `document_ground_weapons.go`,
// `document_aim.go`...). Elle etait donc AVEUGLE a tout champ deplace — et le deplacement de
// `Point` l a rendue rouge alors que le champ existait toujours. Lire le paquet ferme les deux
// defauts d un coup : plus de faux rouge, et plus de faux vert.
func TestG3TableSuitLeDocument(t *testing.T) {
	const docDir = "../replay"
	entries, err := os.ReadDir(docDir)
	if err != nil {
		t.Fatalf("%s : %v", docDir, err)
	}
	fset := token.NewFileSet()
	fields := map[string]bool{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(docDir, name), nil, 0)
		if err != nil {
			t.Fatalf("%s : %v", name, err)
		}
		collectStructFields(f, fields)
	}
	if len(fields) == 0 {
		t.Fatalf("aucun champ lu dans %s", docDir)
	}
	n := 0
	for _, r := range loadECSTable(t) {
		if r.DocField == "" {
			continue
		}
		for _, ref := range strings.Split(r.DocField, ";") {
			n++
			if !fields[ref] {
				t.Errorf("G3 : ligne %d (%s) cite %q, absent du paquet %s", r.LineNo, r.Component, ref, docDir)
			}
		}
	}
	t.Logf("G3 : %d references de champ verifiees contre %d champs du paquet %s", n, len(fields), docDir)
}

// collectStructFields empile les champs de toutes les structures d un fichier, sous la forme
// `Type.Champ`.
func collectStructFields(f *ast.File, fields map[string]bool) {
	ast.Inspect(f, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			return true
		}
		for _, fl := range st.Fields.List {
			for _, nm := range fl.Names {
				fields[ts.Name.Name+"."+nm.Name] = true
			}
		}
		return true
	})
}
