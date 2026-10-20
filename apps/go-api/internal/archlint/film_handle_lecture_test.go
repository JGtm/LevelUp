package archlint

// film_handle_lecture_test.go — LE HANDLE D UN RECORD SE LIT EN UN POINT (lot J5.1 du
// PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, decision DT-7).
//
// # CE QUE CE RATCHET TIENT
//
// Le handle d un en-tete de record — `R(13)` de slot puis `R(2)` de generation — se lit par
// `grammar.LireHandle` / `grammar.LireHandleDelta` (`film/internal/grammar/handle.go`). Avant le
// lot, cinq balayages de production et un instrument le relisaient a la main, sous le litteral
// `p+14, 2)` ou sous des constantes nommees (`woNewSlotBits`, `decalageDuTag`) ; les trois copies
// du filtre « tag == 1 » du bipede en sont nees (constat GB-1 de l audit du 2026-09-24), chacune
// decidant pour son compte de ce qu est une generation acceptable.
//
// # LES DEUX FORMES, MECANIQUEMENT (par l AST, jamais par le texte)
//
// Un appel a une convention de lecture de la couche source (`BitsStricts`, `BitsTolerants`,
// `BitsBourres`, `BitsAt`) dont :
//
//	LARGEUR = 13          un slot de handle. La largeur est EVALUEE : un litteral, une constante du
//	                      paquet (`bipedSlotBits`, `woNewSlotBits`, `slotDEnTeteBits`) ou une
//	                      expression qui n en combine que, conversions entieres retirees.
//	LARGEUR = 2 ET        une generation de handle : la somme des termes CONSTANTS de la position
//	POSITION DE GENERATION vaut 14 (`p+14`, `p+decalageDuTag`), ou l un de ses termes constants vaut
//	                      13 (`p+woNewTypeBits+woNewSlotBits`). Une lecture de 2 bits ailleurs (le
//	                      type d un record NEW a `p+1`, la porte d un record delta a `p+16`) n est
//	                      pas un handle.
//
// Perimetre : les fichiers de PRODUCTION (non `_test.go`) des racines du ratchet « une seule porte
// aux octets » (`racinesOctetsBruts`, memes exclusions) — les outils de recherche tagues
// `research` compris, puisque leurs fichiers ne sont pas des tests. Seul `handle.go` est exempte :
// c est le lecteur.
//
// ANGLE MORT CONNU : une position ou une largeur passee par une VARIABLE locale (`w := 13`) n est
// pas evaluee — le balayage n a pas de type-checker. Le lecteur sequentiel (`Lecteur.ReadBits`,
// `readRecordID`, port de FUN_1406d3140) n est pas vise : c est la marche de trame, pas une relecture
// positionnelle du handle.
//
// # MUTATIONS JOUEES (2026-09-27), ROUGES, PUIS RETIREES
//
// Consignees au rapport du lot J5.1 : le litteral d origine reintroduit dans
// `grammar/offline_biped.go` (`source.BitsStricts(pay, p+14, 2)`), et la forme en constantes
// nommees (`source.BitsTolerants(pay, p+woNewTypeBits, woNewSlotBits)`) dans
// `grammar/debut_de_liste.go`.

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// fichierDuLecteurDeHandle : le seul fichier autorise a lire un handle bit a bit.
const fichierDuLecteurDeHandle = "internal/games/halo_infinite/film/internal/grammar/handle.go"

// Les largeurs du handle, telles que le ratchet les reconnait.
const (
	largeurSlotDeHandle       = 13
	largeurGenerationDeHandle = 2
	positionGenerationDelta   = 14 // prefixe(1) + slot(13)
)

// conventionsDeLectureDeBits : les lecteurs positionnels de la couche source.
var conventionsDeLectureDeBits = map[string]bool{
	"BitsStricts": true, "BitsTolerants": true, "BitsBourres": true, "BitsAt": true,
}

// lectureDeHandle : une lecture artisanale du handle, la ou elle est.
type lectureDeHandle struct {
	fichier string
	ligne   int
	detail  string
}

// TestLeHandleSeLitParLeLecteurUnique — LE RATCHET.
func TestLeHandleSeLitParLeLecteurUnique(t *testing.T) {
	sites, fichiers := balayerLecturesDeHandle(t)
	if fichiers < 100 {
		t.Fatalf("seulement %d fichiers de production parcourus : les racines ont-elles demenage ?", fichiers)
	}
	if len(sites) == 0 {
		return
	}
	lignes := make([]string, 0, len(sites))
	for _, s := range sites {
		lignes = append(lignes, fmt.Sprintf("%s:%d : %s", s.fichier, s.ligne, s.detail))
	}
	t.Fatalf("lecture artisanale du HANDLE (slot 13 bits, generation 2 bits) hors de %s :\n  %s\n"+
		"Lire le handle par `grammar.LireHandle(pay, debutDuHandle)` ou `grammar.LireHandleDelta(pay, "+
		"p)` : une copie decide pour son compte de ce qu est une generation acceptable (constat GB-1).",
		fichierDuLecteurDeHandle, strings.Join(lignes, "\n  "))
}

// balayerLecturesDeHandle parcourt les fichiers de production et rend les lectures artisanales du
// handle, plus le nombre de fichiers parcourus (plancher).
func balayerLecturesDeHandle(t *testing.T) ([]lectureDeHandle, int) {
	t.Helper()
	racineAPI := apiRootDepuisIci(t)
	paquets := paquetsSurveillesOctets(t, racineAPI)
	var sites []lectureDeHandle
	fichiers := 0
	for _, dir := range clesTrieesOctets(paquets) {
		fset := token.NewFileSet()
		asts := map[string]*ast.File{}
		for _, chemin := range paquets[dir] {
			f, err := parser.ParseFile(fset, chemin, nil, 0)
			if err != nil {
				t.Fatalf("parse de %s : %v", chemin, err)
			}
			rel, _ := filepath.Rel(racineAPI, chemin)
			asts[filepath.ToSlash(rel)] = f
			fichiers++
		}
		consts := valeursDesConstantes(asts)
		for _, rel := range clesTrieesOctets(asts) {
			if rel == fichierDuLecteurDeHandle {
				continue
			}
			sites = append(sites, lecturesDeHandleDuFichier(rel, asts[rel], consts, fset)...)
		}
	}
	sort.Slice(sites, func(i, j int) bool {
		if sites[i].fichier != sites[j].fichier {
			return sites[i].fichier < sites[j].fichier
		}
		return sites[i].ligne < sites[j].ligne
	})
	return sites, fichiers
}

// lecturesDeHandleDuFichier rend les lectures artisanales du handle d un fichier.
func lecturesDeHandleDuFichier(rel string, f *ast.File, consts map[string]ast.Expr, fset *token.FileSet) []lectureDeHandle {
	var out []lectureDeHandle
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) != 3 || !conventionsDeLectureDeBits[nomDeLaFonctionAppelee(call.Fun)] {
			return true
		}
		if quoi, ok := lectureDeHandleDe(call.Args[1], call.Args[2], consts); ok {
			out = append(out, lectureDeHandle{fichier: rel, ligne: fset.Position(call.Pos()).Line,
				detail: nomDeLaFonctionAppelee(call.Fun) + " : " + quoi})
		}
		return true
	})
	return out
}

// lectureDeHandleDe classe un appel (position, largeur) : slot de handle, generation de handle, ou
// rien.
func lectureDeHandleDe(pos, largeur ast.Expr, consts map[string]ast.Expr) (string, bool) {
	w, ok := evaluerConstante(largeur, consts, 0)
	if !ok {
		return "", false
	}
	switch w {
	case largeurSlotDeHandle:
		return "slot de handle (largeur 13)", true
	case largeurGenerationDeHandle:
		somme, termeDeSlot := termesConstantsDeLaPosition(pos, consts)
		if somme == positionGenerationDelta || termeDeSlot {
			return "generation de handle (largeur 2 a la position de la generation)", true
		}
	}
	return "", false
}

// termesConstantsDeLaPosition aplatit une position `a + b + ...` : rend la somme de ses termes
// CONSTANTS, et si l un d eux vaut la largeur d un slot de handle.
func termesConstantsDeLaPosition(pos ast.Expr, consts map[string]ast.Expr) (int64, bool) {
	var somme int64
	termeDeSlot := false
	var parcourir func(e ast.Expr)
	parcourir = func(e ast.Expr) {
		e = sansParenthesesNiConversion(e)
		if b, ok := e.(*ast.BinaryExpr); ok && b.Op == token.ADD {
			parcourir(b.X)
			parcourir(b.Y)
			return
		}
		if v, ok := evaluerConstante(e, consts, 0); ok {
			somme += v
			termeDeSlot = termeDeSlot || v == largeurSlotDeHandle
		}
	}
	parcourir(pos)
	return somme, termeDeSlot
}

// valeursDesConstantes rend, par nom, l expression des constantes declarees du paquet — celles de
// paquet ET celles declarees DANS une fonction (une copie du handle sous `const decalageDuTag =
// 1 + 13` locale a passe la premiere version du ratchet : mutation du 2026-09-27). Sans iota : une
// constante sans valeur explicite n est pas evaluable et n est pas rendue. Deux constantes locales
// homonymes de deux fonctions se recouvrent ; le ratchet n en evalue qu une, ce qui ne peut que le
// rendre plus strict ou plus lache sur CE nom — assume, sans type-checker.
func valeursDesConstantes(asts map[string]*ast.File) map[string]ast.Expr {
	out := map[string]ast.Expr{}
	for _, f := range asts {
		ast.Inspect(f, func(n ast.Node) bool {
			g, ok := n.(*ast.GenDecl)
			if !ok || g.Tok != token.CONST {
				return true
			}
			for _, s := range g.Specs {
				vs := s.(*ast.ValueSpec)
				for i, nom := range vs.Names {
					if i < len(vs.Values) {
						out[nom.Name] = vs.Values[i]
					}
				}
			}
			return true
		})
	}
	return out
}

// evaluerConstante evalue une expression entiere faite de litteraux et de constantes du paquet
// (`+ - * <<`, parentheses, conversions entieres). `profondeur` borne la recursion.
func evaluerConstante(e ast.Expr, consts map[string]ast.Expr, profondeur int) (int64, bool) {
	if profondeur > 16 {
		return 0, false
	}
	switch v := sansParenthesesNiConversion(e).(type) {
	case *ast.BasicLit:
		if v.Kind != token.INT {
			return 0, false
		}
		n, err := strconv.ParseInt(v.Value, 0, 64)
		return n, err == nil
	case *ast.Ident:
		def, ok := consts[v.Name]
		if !ok {
			return 0, false
		}
		return evaluerConstante(def, consts, profondeur+1)
	case *ast.BinaryExpr:
		a, okA := evaluerConstante(v.X, consts, profondeur+1)
		b, okB := evaluerConstante(v.Y, consts, profondeur+1)
		if !okA || !okB {
			return 0, false
		}
		switch v.Op {
		case token.ADD:
			return a + b, true
		case token.SUB:
			return a - b, true
		case token.MUL:
			return a * b, true
		case token.SHL:
			return a << uint(b), b >= 0 && b < 63 //nolint:gosec // borne verifiee
		}
	}
	return 0, false
}
