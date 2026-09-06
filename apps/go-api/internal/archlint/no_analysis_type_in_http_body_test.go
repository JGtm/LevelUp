// no_analysis_type_in_http_body_test.go — AUCUN TYPE D'`internal/analysis/` EN CORPS DE
// REQUETE OU DE REPONSE HUMA.
//
// # POURQUOI CE GARDE-RAIL EXISTE, ET IL A UNE DATE
//
// 2026-09-05, lot B du plan v2. Jusqu'a ce jour, `api/handlers/replay.go:68` declarait
// `type replayOutput struct{ Body replay.ReplayDocument }` — et comme `api/openapi.yaml` est
// GENERE depuis les types Go, le format du FICHIER d'artefact de rejeu etait, litteralement,
// le contrat public : 99 des 165 types exportes d'`internal/analysis/replay` etaient des
// entrees de `components.schemas`, donc autant de types TypeScript. Deux consequences
// mesurees : `SchemaVersion` a monte 43 fois en cinq semaines, chaque montee regenerant le
// contrat pour un champ que le client ne lisait pas ; et aucun champ ne pouvait etre renomme
// pour le client sans invalider le parc d'artefacts cuits.
//
// L'exemption ecrite de 2026-08-06 ne couvrait que `ReplayDocument`. `MapBackground`
// (2026-08-12) et `MapCalloutsEntry` (2026-08-13) sont entres ensuite, sans que rien ne le
// signale — c'est exactement la derive qu'un garde-rail attrape et qu'un commentaire ne
// retient pas. Le meme mois, `api/handlers/match_view_positions.go` posait la convention
// INVERSE en refusant de serialiser `positions.PlayerPosition` brut : le depot sait ce qu'il
// veut, il lui manquait la contrainte.
//
// # CE QU'IL VERIFIE, ET COMMENT
//
// Il PARSE (go/ast, pas un grep : un nom de type cite en commentaire ne doit pas declencher)
// tous les fichiers Go de `internal/api/`, tests compris — une route enregistree depuis un
// test compte autant qu'une route de production. Pour chaque struct declaree, il regarde :
//
//   - les champs `Body` et `RawBody` : la convention Huma pour le corps ;
//   - TOUS les champs des structs dont le nom se termine par `Input` ou `Output` : la
//     convention du depot pour les entrees/sorties d'operation (en-tetes et parametres
//     compris — un type `analysis` n'a pas plus sa place dans un en-tete) ;
//   - toute DECLARATION DE TYPE d'`internal/api/` dont le membre de droite est un type
//     d'ailleurs — alias (`type X = replay.ReplayDocument`) comme type defini
//     (`type X replay.ReplayDocument`).
//
// LA TROISIEME VOIE A ETE OUVERTE PAR LA REVUE ADVERSARIALE DU 2026-09-05, qui a joue la
// contournement en une ligne : declarer l'alias, puis ecrire `Body storedReplayDocument`. Le
// scan ne voyait qu'un identifiant local, donc VERT — alors que le corps de `/replay` etait
// redevenu `analysis/replay.ReplayDocument`. Et un alias est le MEME type pour `reflect` :
// Huma nomme le schema `ReplayDocument` comme avant et regenere `openapi.yaml` octet pour
// octet, si bien que le golden ne dit rien non plus. Le type DEFINI, lui, serait rattrape par
// le golden (Huma renommerait le schema) — mais il est interdit ici aussi : un garde-rail qui
// depend d'un second gate pour la moitie de ses cas n'en est pas un.
//
// Les litteraux de struct et d'interface sont ecartes de cette voie : leurs champs relevent
// des deux premieres (cf. `relaisDeType`).
//
// Un type est en faute des que son expression cite un paquet importe sous
// `levelup/go-api/internal/analysis/` — a n'importe quelle profondeur (`[]T`, `*T`,
// `map[string]T`).
//
// LE COMPTE DE CORPS SCANNES EST VERIFIE : un ratchet qui ne trouve plus rien a scanner est
// un ratchet mort, et il le reste en silence. Si la convention `Body` disparait, ce test
// echoue en le disant plutot que de passer sur zero fichier.
package archlint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// analysisPrefix : le prefixe des paquets d'algorithmes. Leurs types decrivent un modele de
// CALCUL ou un format de STOCKAGE ; le contrat public se decrit dans `domain/`.
const analysisPrefix = "levelup/go-api/internal/analysis/"

// corpsHumaMinimum : le nombre de corps Huma que le scan doit voir au minimum. Mesure a 104
// le 2026-09-05 ; le plancher est volontairement bas pour ne pas transformer ce garde-rail en
// compteur a maintenir, et assez haut pour qu'un scan qui ne trouve plus rien echoue.
const corpsHumaMinimum = 80

// corpsAnalysisTolere : les corps de route qui citent encore un type d'`internal/analysis/`,
// par `fichier:Struct.Champ`, avec la DATE de l'inscription, la DATE CIBLE de retrait et le
// CRITERE mesurable. Une entree posee sans ces trois elements est une exemption sans fin — le
// defaut que ce garde-rail existe pour empecher.
//
// Les trois corps du rejeu (`replayOutput`, `backgroundOutput`, `calloutsOutput`) N'Y SONT
// PAS : le lot B les a projetes sur `domain/replaydoc`. Il reste UNE entree, decouverte par ce
// scan meme et anterieure au lot.
var corpsAnalysisTolere = map[string]string{
	"internal/api/handlers/patterns.go:patternsOutput.Body": "2026-09-05 — " +
		"`analysis/patterns.PatternReport` et les trois types qu'il porte (`BehavioralPattern`, " +
		"`ContextualPattern`, `Lever`) sont 4 schemas du contrat public. Le defaut est le meme " +
		"que celui du rejeu, mais il est ANTERIEUR au lot B et HORS de son perimetre (le lot " +
		"separe le document de REJEU, pas le rapport de patrons) : le corriger ici aurait ete un " +
		"fix opportuniste. Cible de retrait : 2026-11-01. Critere mesurable : l'intersection " +
		"entre les types exportes d'`internal/analysis/patterns` et `components.schemas` " +
		"d'`api/openapi.yaml` vaut 0 (elle vaut 4 au 2026-09-05).",
}

func TestAucunTypeAnalysisEnCorpsHuma(t *testing.T) {
	_, thisFile, ok := caller()
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	apiRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile))) // archlint -> internal -> apps/go-api
	racine := filepath.Join(apiRoot, "internal", "api")

	fset := token.NewFileSet()
	var corps int
	var violations []string
	vus := map[string]bool{}

	err := filepath.WalkDir(racine, func(chemin string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(chemin, ".go") {
			return nil
		}
		f, perr := parser.ParseFile(fset, chemin, nil, 0)
		if perr != nil {
			return perr
		}
		alias := aliasAnalysis(f)
		rel, _ := filepath.Rel(apiRoot, chemin)
		rel = filepath.ToSlash(rel)
		juger := func(nom, suffixe string, typ ast.Expr) {
			for _, pkg := range paquetsCites(typ, alias) {
				cle := rel + ":" + nom
				vus[cle] = true
				if motif, tolere := corpsAnalysisTolere[cle]; tolere {
					if strings.TrimSpace(motif) == "" {
						violations = append(violations, cle+"  (tolere SANS justification datee)")
					}
					continue
				}
				violations = append(violations, cle+suffixe+"  -> "+pkg)
			}
		}
		for _, champ := range champsDeCorps(f, &corps) {
			juger(champ.nom, "", champ.typ)
		}
		for _, r := range relaisDeType(f) {
			juger(r.nom, r.forme, r.typ)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parcours de internal/api : %v", err)
	}

	if corps < corpsHumaMinimum {
		t.Fatalf("%d corps Huma scannes, au moins %d attendus : le scan ne voit plus les routes "+
			"(convention `Body` disparue, ou arborescence deplacee) — un ratchet qui ne scanne "+
			"rien passe en silence, ce qui est pire que pas de ratchet", corps, corpsHumaMinimum)
	}
	for cle := range corpsAnalysisTolere {
		if !vus[cle] {
			t.Errorf("corpsAnalysisTolere cite %q, que le code ne porte plus — une exemption qui "+
				"survit a son site finit par en couvrir un autre", cle)
		}
	}
	if len(violations) > 0 {
		t.Errorf("un type d'`internal/analysis/` atteint le contrat public — soit en corps de "+
			"requete ou de reponse Huma, soit par un type d'`internal/api/` qui le relaie. Le "+
			"contrat serait derive d'un modele de calcul (le rejeu a paye 43 montees de schema "+
			"pour ca). Poser un type dans `domain/` et projeter au service :\n  %s",
			strings.Join(violations, "\n  "))
	}
}

// caller isole runtime.Caller pour garder le test lisible.
func caller() (int, string, bool) {
	_, fichier, ligne, ok := runtime.Caller(1)
	return ligne, fichier, ok
}

// aliasAnalysis rend les alias locaux des imports d'`internal/analysis/...` du fichier.
func aliasAnalysis(f *ast.File) map[string]string {
	out := map[string]string{}
	for _, imp := range f.Imports {
		chemin := strings.Trim(imp.Path.Value, `"`)
		if !strings.HasPrefix(chemin, analysisPrefix) {
			continue
		}
		nom := chemin[strings.LastIndexByte(chemin, '/')+1:]
		if imp.Name != nil {
			nom = imp.Name.Name
		}
		out[nom] = chemin
	}
	return out
}

// champCorps : un champ candidat au corps HTTP, avec son nom qualifie pour le message.
type champCorps struct {
	nom string // Struct.Champ
	typ ast.Expr
}

// champsDeCorps rend les champs a juger d'un fichier et incremente le compte de corps vus.
func champsDeCorps(f *ast.File, corps *int) []champCorps {
	var out []champCorps
	ast.Inspect(f, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok || st.Fields == nil {
			return true
		}
		operation := strings.HasSuffix(ts.Name.Name, "Input") || strings.HasSuffix(ts.Name.Name, "Output")
		for _, champ := range st.Fields.List {
			for _, nom := range champ.Names {
				estCorps := nom.Name == "Body" || nom.Name == "RawBody"
				if estCorps {
					*corps++
				}
				if estCorps || operation {
					out = append(out, champCorps{nom: ts.Name.Name + "." + nom.Name, typ: champ.Type})
				}
			}
		}
		return true
	})
	return out
}

// relaisTypeDecl : une declaration de type d'`internal/api/` qui REDONNE un nom local a un
// type d'ailleurs.
type relaisTypeDecl struct {
	nom   string // nom du type declare
	forme string // "  (ALIAS `=`)" ou "  (type defini)", pour le message
	typ   ast.Expr
}

// relaisDeType rend les declarations de type d'un fichier dont le membre de droite EST un
// type d'ailleurs — alias (`type X = pkg.T`) comme type defini (`type X pkg.T`), a travers
// pointeurs, tranches, tableaux et maps.
//
// LES LITTERAUX DE STRUCT ET D'INTERFACE SONT ECARTES, et c'est deliberé : leurs champs sont
// deja juges par la premiere voie (corps `Body`/`RawBody` et structs `*Input`/`*Output`), et
// un `type X struct{ ... }` dont un champ cite `analysis` sans etre un corps de route n'est
// pas une fuite de contrat — c'est une dependance interne de handler.
func relaisDeType(f *ast.File) []relaisTypeDecl {
	var out []relaisTypeDecl
	ast.Inspect(f, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		switch ts.Type.(type) {
		case *ast.StructType, *ast.InterfaceType:
			return true
		}
		forme := "  (type defini)"
		if ts.Assign.IsValid() {
			forme = "  (ALIAS `=` — le plus silencieux : meme type pour reflect, meme nom de schema)"
		}
		out = append(out, relaisTypeDecl{nom: ts.Name.Name, forme: forme, typ: ts.Type})
		return true
	})
	return out
}

// paquetsCites rend les paquets `analysis` cites par une expression de type, a toute
// profondeur (`[]T`, `*T`, `map[string]T`, `[][2]T`).
func paquetsCites(e ast.Expr, alias map[string]string) []string {
	var out []string
	ast.Inspect(e, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		if chemin, ok := alias[ident.Name]; ok {
			out = append(out, chemin+"."+sel.Sel.Name)
		}
		return true
	})
	return out
}
