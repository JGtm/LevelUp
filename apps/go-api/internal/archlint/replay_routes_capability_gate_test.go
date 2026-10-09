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
// # LA REGLE EXACTE : LA PORTE DOIT ETRE POSEE SUR UN ANCETRE DU MONTAGE
//
// Premiere version (2026-09-06) : « chercher la porte dans le plus petit bloc englobant »,
// avec une recherche RECURSIVE dans ce bloc. La seconde ronde de revue (C-R2, defaut N1) l'a
// prise en defaut DANS LES DEUX SENS, parce que « le plus petit bloc » et « recherche
// recursive » ne parlent pas de la meme chose :
//
//   - FAUX NEGATIF — sortir le montage du groupe gate, en gardant a cote un autre `r.Group`
//     gate (pour d'autres routes) : la recherche recursive descendait dans ce groupe FRERE et
//     y trouvait la porte. Quatre routes /replay* de nouveau servies sur halo_5, ratchet vert.
//   - FAUX POSITIF — montage imbrique dans un sous-groupe, porte posee sur le groupe PARENT :
//     refuse a tort, alors que chi propage le middleware d'un groupe a ses groupes imbriques
//     (verifie par sonde en revue).
//
// D'ou la regle appliquee ici : on remonte la chaine des blocs qui CONTIENNENT le montage
// (son bloc, puis le bloc de celui-ci, etc.), et a chaque niveau on cherche la porte SANS
// descendre dans les fonctions litterales imbriquees — c'est-a-dire sans jamais regarder chez
// un frere. Un ancetre garde le montage (chi propage) ; un frere, jamais.
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
		for _, site := range sitesDeMontage(fichier, replayHandlerCtor) {
			sitesTrouves++
			if !unAncetreInstalleLaPorte(site.ancetres) {
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
		t.Errorf("les routes /replay* sont montees SANS porte de titre (%s) : ni leur bloc de "+
			"montage ni aucun bloc qui le contient n'installe %s(..., %s). Sans elle, les quatre "+
			"routes /replay* sont servies par un "+
			"titre qui n'a aucun decodeur de film (halo_5 est ACTIF en production) : 404 « ce match "+
			"n'a pas de rejeu » au lieu de 503 « ce titre n'a pas de rejeu ». Remettre "+
			"`r.Use(middleware.RequireCapability(titleRegistry, titlePkg.%s))` dans le groupe qui "+
			"monte ReplayHandler.", v, replayGateCall, replayGateCap, replayGateCap)
	}
}

// siteMontage : un appel a un constructeur de handler et la CHAINE des blocs qui le
// contiennent, du plus proche au plus lointain. Ce sont ses ancetres, et eux seuls : aucun
// frere.
type siteMontage struct {
	appel    *ast.CallExpr
	ancetres []*ast.BlockStmt
}

// sitesDeMontage rend, pour chaque appel au constructeur donne, la chaine de ses blocs
// englobants — au montage reel : le corps du `r.Group(func(r chi.Router) { ... })`, puis
// celui de `mountAPIV1`, etc.
//
// Parametree par le constructeur depuis le 2026-09-06 : le ratchet du garde local des
// routes tactiques (`tactical_background_local_gate_test.go`) a exactement le meme besoin
// d'extraction, et une seconde copie de cette remontee de pile aurait diverge.
func sitesDeMontage(fichier *ast.File, ctor string) []siteMontage {
	var out []siteMontage
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
		if !appelNomme(appel, ctor) {
			return true
		}
		// On remonte la pile (le dernier element est l'appel lui-meme) et on retient TOUS les
		// blocs traverses : ce sont exactement les blocs dont un middleware s'appliquerait au
		// montage.
		var ancetres []*ast.BlockStmt
		for i := len(pile) - 2; i >= 0; i-- {
			if b, estBloc := pile[i].(*ast.BlockStmt); estBloc {
				ancetres = append(ancetres, b)
			}
		}
		out = append(out, siteMontage{appel: appel, ancetres: ancetres})
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

// unAncetreInstalleLaPorte dit si LA PORTE est posee sur l'un des blocs qui contiennent le
// montage. Vrai des le premier ancetre qui la porte : chi propage le middleware d'un groupe a
// tout ce qu'il contient, y compris aux groupes imbriques.
func unAncetreInstalleLaPorte(ancetres []*ast.BlockStmt) bool {
	for _, bloc := range ancetres {
		if blocInstalleLaPorte(bloc) {
			return true
		}
	}
	return false
}

// blocInstalleLaPorte dit si CE bloc-ci installe la porte, SANS DESCENDRE dans les fonctions
// litterales qu'il contient.
//
// C'est cette coupure qui distingue un ancetre d'un frere. Un `r.Group(func(r chi.Router){
// r.Use(RequireCapability(..., CapReplay)); ... })` voisin est, pour l'AST, un `FuncLit` a
// l'interieur du meme bloc : descendre dedans reviendrait a créditer le montage d'une porte
// qui ne le garde pas (defaut N1, faux negatif). Les `r.Use(...)` qui gardent reellement un
// bloc y sont, eux, des instructions DIRECTES.
func blocInstalleLaPorte(bloc ast.Node) bool {
	trouve := false
	ast.Inspect(bloc, func(n ast.Node) bool {
		if trouve {
			return false
		}
		if _, estFuncLit := n.(*ast.FuncLit); estFuncLit {
			return false // frontiere : au-dela, c'est un frere (ou un descendant), pas nous
		}
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
