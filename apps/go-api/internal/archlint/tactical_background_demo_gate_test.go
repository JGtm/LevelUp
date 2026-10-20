// Package archlint — tactical_background_demo_gate_test.go : ratchet du montage des routes
// TACTIQUES HORS de la garde des routes de rejeu.
//
// POURQUOI CE RATCHET EXISTE. Le fond de carte de la grille est servi par
// `/players/{slug}/tactical/{map_id}/background{,.png}`, monte par `NewTacticalHandler` —
// DELIBEREMENT hors du groupe qui porte `ReplayGate`. En demo, cette garde ne laisse passer que
// les matchs figes (elle lit le `match_id` du chemin ou du corps) : sous elle, une route de
// carte, qui n'a pas de match, repondrait 404 a tout visiteur et la grille des cartes serait
// vide. Une image de carte est une donnee de reference versionnee, extraite des fichiers de
// jeu ; elle n'a rien a faire derriere une garde de matchs.
//
// LE TEST HTTP NE PEUT PAS COUVRIR CA : le test de handler reconstruit son propre routeur, sans
// middleware. Deplacer la ligne de montage de `server_apiv1.go` dans le groupe garde le laisse
// entierement vert. Comme pour la porte de capability du rejeu
// (`replay_routes_capability_gate_test.go`, meme paquet, meme extracteur), seul un ratchet sur
// le SITE DE MONTAGE ferme ce trou.
//
// LE RATCHET EST A DEUX FACES, et c'est necessaire : affirmer seulement « le montage
// tactique n'est pas sous la garde » passerait tout aussi bien si la garde disparaissait du
// depot. On exige donc AUSSI que le montage du rejeu, lui, reste sous cette garde. La paire
// dit la vraie regle : la garde existe, et elle ne couvre pas les routes tactiques.
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
	// gardeRejeu : la fabrique de la garde des routes de rejeu, posee par
	// `r.Use(handlers.ReplayGate(...))` (rejeux figes en demo ; sans effet hors demo).
	gardeRejeu = "ReplayGate"
)

// TestTacticalBackgroundRoutesHorsGardeRejeu — les deux faces de la regle.
func TestTacticalBackgroundRoutesHorsGardeRejeu(t *testing.T) {
	apiRoot := archlintAPIRoot(t)

	tactiques := montagesSousGardeRejeu(t, apiRoot, tacticalHandlerCtor)
	rejeu := montagesSousGardeRejeu(t, apiRoot, replayHandlerCtor)

	if tactiques.sites == 0 {
		t.Fatalf("aucun site de montage de %s trouve sous %s — le ratchet ne mord plus sur rien "+
			"(constructeur renomme ? montage deplace ?) : corriger l'extracteur, ne pas supprimer "+
			"ce test", tacticalHandlerCtor, apiRoot)
	}
	if rejeu.sites == 0 {
		t.Fatalf("aucun site de montage de %s trouve — meme remarque", replayHandlerCtor)
	}

	for _, site := range tactiques.sousGarde {
		t.Errorf("les routes tactiques sont montees SOUS %s (%s). La garde ne laisse passer, en "+
			"demo, que les matchs figes ; le fond de carte de la grille est une donnee de "+
			"REFERENCE versionnee, sans match. Sous cette garde, la grille des cartes n'a plus "+
			"aucun fond en demo. Sortir "+
			"`handlers.NewTacticalHandler(...).Mount(...)` du groupe qui porte %s.",
			gardeRejeu, site, gardeRejeu)
	}

	// Face 2 : la garde doit exister et couvrir le rejeu. Sinon la face 1 est vacante.
	if len(rejeu.sousGarde) == 0 {
		t.Errorf("le montage de %s n'est plus sous %s : ce ratchet affirmait que les routes "+
			"tactiques en sont EXCLUES, ce qui ne veut plus rien dire si la garde ne couvre "+
			"plus rien. Soit la garde a ete retiree (mettre a jour ce test et sa justification, "+
			"cf. handlers/replay_demo_gate.go), soit c'est une regression.",
			replayHandlerCtor, gardeRejeu)
	}
}

type verdictGardeRejeu struct {
	sites     int
	sousGarde []string
}

// montagesSousGardeRejeu rend, pour un constructeur, le nombre de sites de montage trouves
// et ceux dont un bloc ancetre installe la garde du rejeu.
func montagesSousGardeRejeu(t *testing.T, apiRoot, ctor string) verdictGardeRejeu {
	t.Helper()
	var out verdictGardeRejeu

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
			if unAncetreInstalleLaGardeRejeu(site.ancetres) {
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

// unAncetreInstalleLaGardeRejeu : la garde est-elle posee sur l'un des blocs qui CONTIENNENT le
// montage ? Meme regle que la porte de capability (chi propage d'un groupe a ses enfants) et
// meme frontiere : on ne descend jamais dans une fonction litterale, sans quoi un groupe
// FRERE qui porte la garde crediterait a tort le montage.
func unAncetreInstalleLaGardeRejeu(ancetres []*ast.BlockStmt) bool {
	for _, bloc := range ancetres {
		if blocInstalleLaGardeRejeu(bloc) {
			return true
		}
	}
	return false
}

// blocInstalleLaGardeRejeu dit si CE bloc-ci pose `r.Use(..., ReplayGate(...))`.
//
// La garde est le RESULTAT d'un appel passe en ARGUMENT (`r.Use(handlers.ReplayGate(x))`) :
// on cherche donc, parmi les arguments d'un `Use`, un appel a la fabrique.
func blocInstalleLaGardeRejeu(bloc ast.Node) bool {
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
			if fabrique, ok := arg.(*ast.CallExpr); ok && appelNomme(fabrique, gardeRejeu) {
				trouve = true
				return false
			}
		}
		return true
	})
	return trouve
}
