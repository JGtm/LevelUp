// Package archlint — replay_routes_capability_gate_test.go : ratchet du MONTAGE des routes
// `/replay*` derriere la capability de titre `replay`.
//
// POURQUOI CE RATCHET EXISTE. Le lot v2(C.2) a pose la porte de titre du rejeu 2D :
// `r.Use(middleware.RequireCapability(titleRegistry, titlePkg.CapReplay))` sur le groupe qui
// monte `ReplayHandler`. Un test de handler la couvrait — mais en RECONSTRUISANT son propre
// routeur, donc il continuait de passer si la ligne disparaissait du montage reel
// (`internal/api/server_apiv1.go`). La revue adversariale C-R1 l'a montre par mutation (M5) :
// ligne supprimee, quatre routes `/replay*` a nouveau servies sur halo_5 — titre ACTIF en
// production — et zero test rouge.
//
// Ce ratchet ferme ce trou-la, et lui seul : il n'exerce aucun comportement HTTP (le test de
// handler garde ce role), il verifie que le SITE DE MONTAGE porte bien sa porte.
//
// POURQUOI UN RATCHET ET PAS UN TEST DU ROUTEUR REEL. `mountAPIV1` prend l'integralite des
// dependances du serveur (registre de services, session, settings, ownership) : le construire
// exigerait des bases DuckDB et un boot complet. Le cout serait sans commune mesure avec ce
// qu'il prouverait de plus.
package archlint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// replayHandlerCtor : le constructeur dont le montage doit etre garde.
const replayHandlerCtor = "NewReplayHandler"

// replayGateCall / replayGateCap : les deux marqueurs exiges dans le bloc de montage.
const (
	replayGateCall = "RequireCapability"
	replayGateCap  = "CapReplay"
)

// TestReplayRoutesMountedUnderCapabilityGate — tout site qui monte `ReplayHandler` doit se
// trouver dans un bloc qui installe `RequireCapability(..., CapReplay)`.
//
// Le test echoue AUSSI si plus aucun site de montage n'est trouve : un ratchet qui ne mord
// plus sur rien est un ratchet mort (le constructeur a pu etre renomme).
func TestReplayRoutesMountedUnderCapabilityGate(t *testing.T) {
	apiRoot := archlintAPIRoot(t)

	sitesTrouves := 0
	var violations []string

	err := filepath.WalkDir(apiRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			switch d.Name() {
			case "vendor", ".git", "node_modules", "tmp", "testdata":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		octets, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		// Le constructeur est DEFINI dans handlers/replay.go : seuls ses APPELS nous
		// interessent, et un fichier qui ne le nomme pas n'a pas besoin d'etre parse.
		if !strings.Contains(string(octets), replayHandlerCtor+"(") {
			return nil
		}
		rel, _ := filepath.Rel(apiRoot, path)
		rel = filepath.ToSlash(rel)
		if rel == "internal/api/handlers/replay.go" {
			return nil // la definition, pas un montage
		}

		fset := token.NewFileSet()
		fichier, parseErr := parser.ParseFile(fset, path, octets, parser.SkipObjectResolution)
		if parseErr != nil {
			t.Fatalf("parser.ParseFile(%s): %v", rel, parseErr)
		}
		for _, site := range sitesDeMontageReplay(fichier) {
			sitesTrouves++
			if !blocInstalleLaPorte(site.bloc) {
				violations = append(violations,
					rel+":"+strconv.Itoa(fset.Position(site.appel.Pos()).Line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parcours de %s: %v", apiRoot, err)
	}

	if sitesTrouves == 0 {
		t.Fatalf("aucun site de montage de %s trouve sous %s — le ratchet ne mord plus sur rien "+
			"(constructeur renomme ? montage deplace ?) : corriger l'extracteur, ne pas supprimer ce test",
			replayHandlerCtor, apiRoot)
	}

	for _, v := range violations {
		t.Errorf("les routes /replay* sont montees SANS porte de titre (%s) : le bloc de montage "+
			"n'installe pas %s(..., %s). Sans elle, les quatre routes /replay* sont servies par un "+
			"titre qui n'a aucun decodeur de film (halo_5 est ACTIF en production) : 404 « ce match "+
			"n'a pas de rejeu » au lieu de 503 « ce titre n'a pas de rejeu ». Remettre "+
			"`r.Use(middleware.RequireCapability(titleRegistry, titlePkg.%s))` dans le groupe qui "+
			"monte ReplayHandler.", v, replayGateCall, replayGateCap, replayGateCap)
	}
}

// siteMontageReplay : un appel a NewReplayHandler et le bloc qui l'englobe.
type siteMontageReplay struct {
	appel *ast.CallExpr
	bloc  ast.Node
}

// sitesDeMontageReplay rend, pour chaque appel a NewReplayHandler, le plus PETIT bloc qui le
// contient — c'est-a-dire, au montage reel, le corps du `r.Group(func(r chi.Router) { ... })`.
// Chercher la porte dans ce bloc-la et non dans la fonction entiere est ce qui rend le
// ratchet exact : un `RequireCapability` pose sur un AUTRE groupe de la meme fonction ne
// garde pas ces routes.
func sitesDeMontageReplay(fichier *ast.File) []siteMontageReplay {
	var out []siteMontageReplay
	var pile []ast.Node

	ast.Inspect(fichier, func(n ast.Node) bool {
		if n == nil {
			if len(pile) > 0 {
				pile = pile[:len(pile)-1]
			}
			return true
		}
		pile = append(pile, n)

		appel, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if !appelNomme(appel, replayHandlerCtor) {
			return true
		}
		// Le bloc englobant le plus proche, en remontant la pile (le dernier element est
		// l'appel lui-meme).
		var bloc ast.Node = fichier
		for i := len(pile) - 2; i >= 0; i-- {
			if b, estBloc := pile[i].(*ast.BlockStmt); estBloc {
				bloc = b
				break
			}
		}
		out = append(out, siteMontageReplay{appel: appel, bloc: bloc})
		return true
	})
	return out
}

// appelNomme dit si l'appel porte le nom donne, qualifie (`handlers.NewReplayHandler`) ou nu.
func appelNomme(appel *ast.CallExpr, nom string) bool {
	switch fn := appel.Fun.(type) {
	case *ast.Ident:
		return fn.Name == nom
	case *ast.SelectorExpr:
		return fn.Sel.Name == nom
	}
	return false
}

// blocInstalleLaPorte dit si le bloc contient un appel `RequireCapability(...)` dont l'un des
// arguments nomme `CapReplay`.
func blocInstalleLaPorte(bloc ast.Node) bool {
	trouve := false
	ast.Inspect(bloc, func(n ast.Node) bool {
		appel, ok := n.(*ast.CallExpr)
		if !ok || !appelNomme(appel, replayGateCall) {
			return true
		}
		for _, arg := range appel.Args {
			if identNomme(arg, replayGateCap) {
				trouve = true
				return false
			}
		}
		return true
	})
	return trouve
}

// identNomme dit si l'expression est l'identifiant donne, qualifie (`titlePkg.CapReplay`) ou nu.
func identNomme(expr ast.Expr, nom string) bool {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name == nom
	case *ast.SelectorExpr:
		return e.Sel.Name == nom
	}
	return false
}

// archlintAPIRoot remonte de internal/archlint a apps/go-api.
func archlintAPIRoot(t *testing.T) string {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(ici)))
}
