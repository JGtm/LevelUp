package archlint

// film_pont_identite_test.go — LE COLLECTEUR ET LA CUISSON LISENT LE PONT D IDENTITE PAR LE MEME
// ETAGE (lot J4.3 du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, decision DU-3 = S1, constat RA1-3).
//
// # LE DEFAUT QUE CE RATCHET FERME
//
// Jusqu au lot J4.3, `sync/killcollector` (`buildPositionRows`) recopiait la SEQUENCE des lectures
// du pont d identite — positions bipedes, origine d horloge, fil des morts, table d index,
// creations de bipede — en appelant chaque balayage a la main. La recopie avait deja diverge :
// les positions du collecteur etaient lues SANS les exemptions de translocation que la cuisson
// applique (decision D2 du PLAN_LECTURE_FIABLE_EQUIPEMENT), donc une arrivee de teleportation
// rejetee par le filtre de vitesse, et une position de mort absente ou prise ailleurs. Les deux
// producteurs du meme fait ne lisaient pas le meme film.
//
// # CE QU IL EXIGE, PAR L AST DES FICHIERS DE PRODUCTION
//
//   - `sync/killcollector` et `film/replay` appellent l etage unique `ScanPontDIdentite` ;
//   - aucun des deux n appelle l une des SIX lectures que l etage compose
//     ([lecturesDuPontDIdentite]) — ni sous `grammar.`, ni sous `decfilm.` ;
//   - `sync/killcollector` n appelle aucun balayage (`Scan*`) de `replay` : les lectures du film
//     ne vivent plus dans la couche de publication.
//
// Les POLITIQUES restent chez chaque appelant (roster de l index, capture des directions,
// injectivite, fatalite des erreurs) : elles ne sont pas des lectures.
//
// # MUTATIONS QUI DOIVENT LE FAIRE ROUGIR
//
//   - remettre `decfilm.ScanDeaths(film)` dans `sync/killcollector/positions.go` ;
//   - appeler `grammar.ScanBipedPositions` depuis `replay/film_scan.go` a cote de l etage ;
//   - retirer l appel de l etage d un des deux paquets.

import (
	"go/ast"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// etageDuPontDIdentite : le nom de l etage unique, en `grammar` et re-exporte par `decfilm`.
const etageDuPontDIdentite = "ScanPontDIdentite"

// lecturesDuPontDIdentite : les six lectures que l etage compose, et qu aucun des deux appelants
// ne doit plus faire lui-meme.
var lecturesDuPontDIdentite = map[string]bool{
	"ScanTranslocatorTeleports": true,
	"ScanBipedPositions":        true,
	"ScanBipedCreations":        true,
	"ScanDeaths":                true,
	"ScanPlayerIndices":         true,
	"ScanClockOrigin":           true,
}

// appelantsDuPontDIdentite : les deux producteurs du pont d identite, relatifs a `apps/go-api`.
var appelantsDuPontDIdentite = []string{
	"internal/sync/killcollector",
	"internal/games/halo_infinite/film/replay",
}

// TestCollecteurEtCuissonAppellentLeMemeEtage — LE RATCHET.
func TestCollecteurEtCuissonAppellentLeMemeEtage(t *testing.T) {
	racine := apiRootDepuisIci(t)
	var violations []string
	for _, pkg := range appelantsDuPontDIdentite {
		fichiers := fichiersGoNonTest(t, filepath.Join(racine, filepath.FromSlash(pkg)))
		if len(fichiers) == 0 {
			t.Fatalf("%s : aucun fichier de production lu — le ratchet ne garde plus rien", pkg)
		}
		appelleLEtage := false
		for nom, f := range fichiers {
			for _, a := range appelsQualifies(f) {
				switch {
				case a.sel == etageDuPontDIdentite:
					appelleLEtage = true
				case lecturesDuPontDIdentite[a.sel]:
					violations = append(violations, pkg+"/"+nom+" : "+a.qual+"."+a.sel+
						" (lecture que l etage compose)")
				case pkg == "internal/sync/killcollector" && a.qual == "replay" &&
					strings.HasPrefix(a.sel, "Scan"):
					violations = append(violations, pkg+"/"+nom+" : replay."+a.sel+
						" (la couche de publication ne lit pas le film)")
				}
			}
		}
		if !appelleLEtage {
			violations = append(violations, pkg+" : n appelle pas "+etageDuPontDIdentite)
		}
	}
	if len(violations) == 0 {
		return
	}
	sort.Strings(violations)
	t.Fatalf("le pont d identite n est pas lu par l etage unique :\n  %s\n"+
		"La cuisson et le collecteur appellent `%s` (grammar, re-exporte par decfilm) et gardent "+
		"leurs POLITIQUES ; une seconde sequence de lectures diverge (constat RA1-3 : positions "+
		"du collecteur sans exemptions de translocation).",
		strings.Join(violations, "\n  "), etageDuPontDIdentite)
}

// appelQualifie : un appel `qual.Sel(...)`.
type appelQualifie struct{ qual, sel string }

// appelsQualifies rend les appels dont la fonction est un selecteur sur un identifiant.
func appelsQualifies(f *ast.File) []appelQualifie {
	var out []appelQualifie
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if id, ok := sel.X.(*ast.Ident); ok {
			out = append(out, appelQualifie{qual: id.Name, sel: sel.Sel.Name})
		}
		return true
	})
	return out
}
