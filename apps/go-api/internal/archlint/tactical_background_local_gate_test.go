// Package archlint — tactical_background_local_gate_test.go : ratchet du montage des routes
// TACTIQUES HORS du garde local du rejeu.
//
// POURQUOI CE RATCHET EXISTE (revue R1 de la phase 4, constat G2). Le fond de carte de la
// grille est servi par `/players/{slug}/tactical/{map_id}/background{,.png}`, monte par
// `NewTacticalHandler` — DELIBEREMENT hors du groupe qui porte `LocalOnlyReplay`. Ce garde
// protege les TRAJECTOIRES decodees du film, dont la couverture n'est pas
// productionnalisable ; une image de carte est une donnee de reference versionnee, extraite
// des fichiers de jeu. Deplacer le montage tactique dans ce groupe viderait la grille des
// cartes en production sans rien proteger.
//
// LE TEST HTTP NE POUVAIT PAS COUVRIR CA, et c'est le defaut qui a ete releve : le test de
// handler reconstruit son propre routeur, sans middleware. Deplacer la ligne de montage de
// `server_apiv1.go` dans le groupe garde le laissait entierement vert — y compris le test
// qui affirmait couvrir ce point. Comme pour la porte de capability du rejeu
// (`replay_routes_capability_gate_test.go`, meme paquet, meme extracteur), seul un ratchet
// sur le SITE DE MONTAGE ferme ce trou.
//
// LE RATCHET EST A DEUX FACES, et c'est necessaire : affirmer seulement « le montage
// tactique n'est pas sous le garde » passerait tout aussi bien si le garde disparaissait du
// depot. On exige donc AUSSI que le montage du rejeu, lui, reste sous ce garde. La paire
// dit la vraie regle : le garde existe, et il ne couvre pas les routes tactiques.
package archlint

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

const (
	tacticalHandlerCtor = "NewTacticalHandler"
	gardeLocalRejeu     = "LocalOnlyReplay"
)

// TestTacticalBackgroundRoutesHorsGardeLocal — les deux faces de la regle.
func TestTacticalBackgroundRoutesHorsGardeLocal(t *testing.T) {
	apiRoot := archlintAPIRoot(t)

	tactiques := montagesSousGardeLocal(t, apiRoot, tacticalHandlerCtor)
	rejeu := montagesSousGardeLocal(t, apiRoot, replayHandlerCtor)

	if tactiques.sites == 0 {
		t.Fatalf("aucun site de montage de %s trouve sous %s — le ratchet ne mord plus sur rien "+
			"(constructeur renomme ? montage deplace ?) : corriger l'extracteur, ne pas supprimer "+
			"ce test", tacticalHandlerCtor, apiRoot)
	}
	if rejeu.sites == 0 {
		t.Fatalf("aucun site de montage de %s trouve — meme remarque", replayHandlerCtor)
	}

	for _, site := range tactiques.sousGarde {
		t.Errorf("les routes tactiques sont montees SOUS %s (%s). Le garde local protege les "+
			"trajectoires decodees du film ; le fond de carte de la grille est une donnee de "+
			"REFERENCE versionnee. Sous ce garde, la grille des cartes n'a plus aucun fond en "+
			"production, et rien de plus n'est protege. Sortir "+
			"`handlers.NewTacticalHandler(...).Mount(...)` du groupe qui porte %s.",
			gardeLocalRejeu, site, gardeLocalRejeu)
	}

	// Face 2 : le garde doit exister et couvrir le rejeu. Sinon la face 1 est vacante.
	if len(rejeu.sousGarde) == 0 {
		t.Errorf("le montage de %s n'est plus sous %s : ce ratchet affirmait que les routes "+
			"tactiques en sont EXCLUES, ce qui ne veut plus rien dire si le garde ne couvre "+
			"plus rien. Soit le garde a ete retire (mettre a jour ce test et sa justification, "+
			"cf. handlers/replay_local_gate.go), soit c'est une regression.",
			replayHandlerCtor, gardeLocalRejeu)
	}
}

type verdictGardeLocal struct {
	sites     int
	sousGarde []string
}

// montagesSousGardeLocal rend, pour un constructeur, le nombre de sites de montage trouves
// et ceux dont un bloc ancetre installe le garde local.
func montagesSousGardeLocal(t *testing.T, apiRoot, ctor string) verdictGardeLocal {
	t.Helper()
	var out verdictGardeLocal

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
		if !strings.Contains(string(octets), ctor+"(") {
			return nil
		}
		rel := filepath.ToSlash(mustRel(apiRoot, path))
		// La DEFINITION du constructeur n'est pas un montage.
		if strings.HasPrefix(rel, "internal/api/handlers/") {
			return nil
		}
		fset := token.NewFileSet()
		fichier, parseErr := parser.ParseFile(fset, path, octets, parser.SkipObjectResolution)
		if parseErr != nil {
			t.Fatalf("parser.ParseFile(%s): %v", rel, parseErr)
		}
		for _, site := range sitesDeMontage(fichier, ctor) {
			out.sites++
			if unAncetreInstalleLeGardeLocal(site.ancetres) {
				out.sousGarde = append(out.sousGarde,
					rel+":"+strconv.Itoa(fset.Position(site.appel.Pos()).Line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parcours de %s: %v", apiRoot, err)
	}
	return out
}

func mustRel(base, path string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return path
	}
	return rel
}

// unAncetreInstalleLeGardeLocal : le garde est-il pose sur l'un des blocs qui CONTIENNENT le
// montage ? Meme regle que la porte de capability (chi propage d'un groupe a ses enfants) et
// meme frontiere : on ne descend jamais dans une fonction litterale, sans quoi un groupe
// FRERE qui porte le garde crediterait a tort le montage.
func unAncetreInstalleLeGardeLocal(ancetres []*ast.BlockStmt) bool {
	for _, bloc := range ancetres {
		if blocInstalleLeGardeLocal(bloc) {
			return true
		}
	}
	return false
}

// blocInstalleLeGardeLocal dit si CE bloc-ci pose `r.Use(..., LocalOnlyReplay)`.
//
// Le garde est un middleware passe en ARGUMENT (`r.Use(handlers.LocalOnlyReplay)`), pas un
// appel : on cherche donc l'identifiant parmi les arguments d'un `Use`.
func blocInstalleLeGardeLocal(bloc ast.Node) bool {
	trouve := false
	ast.Inspect(bloc, func(n ast.Node) bool {
		if trouve {
			return false
		}
		if _, estFuncLit := n.(*ast.FuncLit); estFuncLit {
			return false // frontiere : au-dela, c'est un frere, pas nous
		}
		appel, ok := n.(*ast.CallExpr)
		if !ok || !appelNomme(appel, "Use") {
			return true
		}
		for _, arg := range appel.Args {
			if identNomme(arg, gardeLocalRejeu) {
				trouve = true
				return false
			}
		}
		return true
	})
	return trouve
}
