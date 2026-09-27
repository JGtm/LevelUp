package replay

// versement_des_replis_test.go — LA TABLE DE VERSEMENT, CHAMP PAR CHAMP (lot J8.7, 2026-09-27).
//
// LE MAILLON TENU : un compte que la couche basse rend en donnees ARRIVE au compteur de la cuisson,
// sous le nom de SON entree. Chaque champ source est pose seul, la table est jouee, et le compteur
// doit porter ce compte-la, une fois, sous un nom du registre dont le compteur est branche.
//
// MUTATION JOUEE (2026-09-27) : retirer la ligne `fallback.NomChunksApresTrouAbandonnes` de
// [versementsDesReplis] fait rougir `TestChaqueChampDuRapportDeGrammaireEstVerse` (« le champ
// ChunksApresTrouAbandonnes n arrive pas au compteur »).

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// TestChaqueChampDuRapportDeGrammaireEstVerse : chaque champ de [grammar.ComptesDesReplis] est verse
// par la table, sous UN nom, et ce nom est une entree du registre. Que l entree soit BRANCHEE est
// tenu ailleurs (`fallback.TestChaqueRepliEstCompte`, direction (E) d `archlint`).
func TestChaqueChampDuRapportDeGrammaireEstVerse(t *testing.T) {
	var zero grammar.ComptesDesReplis
	typ := reflect.TypeOf(zero)
	nomsVus := map[fallback.Nom]string{}
	for i := 0; i < typ.NumField(); i++ {
		var r grammar.ComptesDesReplis
		reflect.ValueOf(&r).Elem().Field(i).SetInt(7)
		fb := fallback.NouveauCompteur()
		versementDuBalayage(fb, grammarContexteAvec(r))
		rap := fb.Rapport()
		champ := typ.Field(i).Name
		if len(rap) != 1 || rap[0].Declenchements != 7 {
			t.Errorf("le champ %s n arrive pas au compteur sous UN nom : rapport %v", champ, rap)
			continue
		}
		if autre, deja := nomsVus[rap[0].Nom]; deja {
			t.Errorf("%s et %s sont verses sous le MEME nom %s", autre, champ, rap[0].Nom)
		}
		nomsVus[rap[0].Nom] = champ
		if _, ok := fallback.Lire(rap[0].Nom); !ok {
			t.Errorf("le champ %s est verse sous %s, qui n est pas une entree du registre", champ, rap[0].Nom)
		}
	}
}

// TestUnRapportVideNeVerseRien : une source a zero ne cree aucune ligne — le rapport ne porte que ce
// qui s est declenche.
func TestUnRapportVideNeVerseRien(t *testing.T) {
	fb := fallback.NouveauCompteur()
	verserLesReplis(fb, sourcesDeReplis{})
	if rap := fb.Rapport(); len(rap) != 0 {
		t.Fatalf("sources vides : rapport %v, attendu vide", rap)
	}
}

// TestLeRapportDuBalayageEstVerseAvantLaCaptureDesFaits : le maillon que la seule bobine du depot ne
// peut pas exercer (elle n a pas d image-cle de bipede, le balayage des positions la refuse) se tient
// sur la SOURCE — [BuildFromFilmAvecFaits] verse le rapport du contexte AVANT `faitsDuBalayage`.
// Apres, il manquerait aux faits persistes et une republication le perdrait ; absent, la cuisson ne
// le publierait jamais.
//
// MUTATION JOUEE (2026-09-27) : deplacer l appel apres `faitsDuBalayage` fait rougir ce test.
func TestLeRapportDuBalayageEstVerseAvantLaCaptureDesFaits(t *testing.T) {
	appels := appelsDansLOrdre(t, "build_from_film.go", "BuildFromFilmAvecFaits")
	iVerse, iFaits := -1, -1
	for i, a := range appels {
		switch a {
		case "versementDuBalayage":
			iVerse = i
		case "faitsDuBalayage":
			iFaits = i
		}
	}
	if iVerse < 0 || iFaits < 0 || iVerse > iFaits {
		t.Fatalf("BuildFromFilmAvecFaits : versementDuBalayage en %d, faitsDuBalayage en %d (appels %v) — le "+
			"rapport des replis du balayage doit rejoindre le compteur AVANT la capture des faits", iVerse, iFaits, appels)
	}
}

// appelsDansLOrdre rend les noms des fonctions appelees par `fonction` dans `fichier`, dans l ordre
// du source.
func appelsDansLOrdre(t *testing.T, fichier, fonction string) []string {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), fichier, nil, 0)
	if err != nil {
		t.Fatalf("parse de %s : %v", fichier, err)
	}
	var out []string
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Name.Name != fonction {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if c, ok := n.(*ast.CallExpr); ok {
				if id, ok := c.Fun.(*ast.Ident); ok {
					out = append(out, id.Name)
				}
			}
			return true
		})
	}
	return out
}

// grammarContexteAvec rend un contexte de film (sans film) dont le rapport vaut `r`.
func grammarContexteAvec(r grammar.ComptesDesReplis) *grammar.FilmContext {
	fc := grammar.NewFilmContext(nil)
	fc.NoterReplis(r)
	return fc
}
