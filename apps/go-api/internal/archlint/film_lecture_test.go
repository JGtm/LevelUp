package archlint

// film_lecture_test.go — LA STRUCTURE DE LECTURE DU FILM : UNE FEUILLE SANS LOGIQUE, INVISIBLE DE
// LA PUBLICATION (ADR 0037 IR-9).
//
// `film/internal/grammar/lecture` porte les types de la représentation intermédiaire : ce que la
// marche de la grammaire a lu, paquet par paquet. Trois règles, lues dans l'AST :
//
//	L1 (la feuille)    ses fichiers de production n'importent, dans le dépôt, que `film/types` :
//	                   la forme de la structure ne fait monter que `grammar.Rev` ;
//	L2 (sans logique)  ils ne déclarent ni fonction, ni méthode, ni variable : la marche qui la
//	                   remplit vit dans `grammar`, les lectures qui la consomment chez leurs canaux ;
//	L3 (la frontière)  ni la couche de publication (`replay` et ses sous-paquets) ni la façade
//	                   `decfilm` ne l'importent : `replay` publie depuis les faits, la façade ne
//	                   grandit pas.
//
// Hors de `film/`, le compilateur refuse déjà l'import (répertoire `internal/`) ; ce ratchet garde
// ce que le compilateur ne sait pas dire.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : importer `film/internal/source` dans un fichier de
// `lecture` (L1) ; y déclarer `func (p *Paquet) Vide() bool { return len(p.Records) == 0 }`
// (L2) ; importer `lecture` dans un fichier de production de `film/replay` (L3).

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	// paquetLectureDuFilm : le paquet gardé, relatif à `apps/go-api`.
	paquetLectureDuFilm = "internal/games/halo_infinite/film/internal/grammar/lecture"
	// paquetTypesDuFilm : le seul import du dépôt que L1 admet.
	paquetTypesDuFilm = "internal/games/halo_infinite/film/types"
	// facadeDuFilm : la façade, que L3 garde avec la couche de publication.
	facadeDuFilm = "internal/games/halo_infinite/film/decfilm"
)

// TestLectureEstUneFeuilleSansLogique : L1 et L2.
func TestLectureEstUneFeuilleSansLogique(t *testing.T) {
	racineAPI := apiRootDepuisIci(t)
	imports, fichiers := importsDeProductionDuPaquet(t, racineAPI, paquetLectureDuFilm)
	if fichiers == 0 {
		t.Fatalf("balayage muet : aucun fichier de production dans %s", paquetLectureDuFilm)
	}
	for _, imp := range imports {
		if imp != paquetTypesDuFilm {
			t.Errorf("L1 : %s importe %s ; la structure de lecture n'importe, dans le dépôt, que "+
				"%s (ADR 0037 IR-9). Passer la donnée par un type de `film/types`, ou garder la "+
				"logique dans `grammar`.", paquetLectureDuFilm, imp, paquetTypesDuFilm)
		}
	}
	for _, d := range declarationsDeLogique(t, racineAPI, paquetLectureDuFilm) {
		t.Errorf("L2 : %s ; la structure de lecture ne porte que des types et des constantes "+
			"(ADR 0037 IR-9) — la marche qui la remplit vit dans `grammar`.", d)
	}
}

// TestNiReplayNiDecfilmNImportentLaLecture : L3.
func TestNiReplayNiDecfilmNImportentLaLecture(t *testing.T) {
	graphe, _ := balayerPaquetsDuDecodeur(t)
	gardes := 0
	for _, de := range clesTrieesFilm(graphe) {
		if !importeurInterditDeLaLecture(de) {
			continue
		}
		gardes++
		for _, vers := range graphe[de] {
			if souscheminDeFilm(vers, paquetLectureDuFilm) {
				t.Errorf("L3 : %s importe %s ; la publication et la façade ne voient pas la "+
					"structure de lecture (ADR 0037 IR-9) : elles reçoivent des faits.", de, vers)
			}
		}
	}
	if gardes < 2 {
		t.Fatalf("balayage muet : %d paquet(s) de publication ou de façade vus, au moins 2 "+
			"attendus (`replay`, `decfilm`) — la table des couches a bougé", gardes)
	}
}

// importeurInterditDeLaLecture dit si le paquet `rel` est de ceux que L3 garde : la couche de
// publication et la façade.
func importeurInterditDeLaLecture(rel string) bool {
	return couchesDuDecodeur[rel] == coucheReplay || rel == facadeDuFilm
}

// declarationsDeLogique rend, pour les fichiers de production du paquet `rel`, chaque fonction,
// méthode ou variable déclarée au niveau du paquet, sous la forme `fichier:ligne nom`.
func declarationsDeLogique(t *testing.T, racineAPI, rel string) []string {
	t.Helper()
	dir := filepath.Join(racineAPI, filepath.FromSlash(rel))
	entrees, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("paquet %s introuvable : %v", rel, err)
	}
	fset := token.NewFileSet()
	var out []string
	for _, e := range entrees {
		nom := e.Name()
		if e.IsDir() || !strings.HasSuffix(nom, ".go") || strings.HasSuffix(nom, "_test.go") {
			continue
		}
		f, perr := parser.ParseFile(fset, filepath.Join(dir, nom), nil, parser.SkipObjectResolution)
		if perr != nil {
			t.Fatalf("analyse de %s/%s : %v", rel, nom, perr)
		}
		for _, decl := range f.Decls {
			out = append(out, nomsDeLogique(fset, decl)...)
		}
	}
	return out
}

// nomsDeLogique rend la fonction, la méthode ou les variables qu'une déclaration de niveau paquet
// introduit ; rien pour un type, une constante ou un import.
func nomsDeLogique(fset *token.FileSet, decl ast.Decl) []string {
	pos := func(n ast.Node) string {
		p := fset.Position(n.Pos())
		return filepath.Base(p.Filename) + ":" + itoa(p.Line)
	}
	switch d := decl.(type) {
	case *ast.FuncDecl:
		return []string{pos(d) + " func " + d.Name.Name}
	case *ast.GenDecl:
		if d.Tok != token.VAR {
			return nil
		}
		var out []string
		for _, s := range d.Specs {
			vs, ok := s.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, n := range vs.Names {
				out = append(out, pos(n)+" var "+n.Name)
			}
		}
		return out
	}
	return nil
}
