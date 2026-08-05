package migration

// written_at_utc_guard_test.go — GARDE-RAIL de la forme canonique du DEFAULT et des
// écritures de `written_at` (lot S2, suite de R1).
//
// `written_at` est la colonne de TRI des vues `<table>_latest` (ADR 0026) : y écrire
// deux horloges différentes, c'est y écrire deux préséances. `now()` et
// `CURRENT_TIMESTAMP` rendent un TIMESTAMPTZ que DuckDB coerce vers une colonne
// TIMESTAMP naive par le fuseau de SESSION — à UTC+2 la ligne se date deux heures dans
// le futur et gagne l'arbitrage contre toute ligne UTC écrite ensuite. La lecture perd
// alors l'enrichissement sans erreur, sans compteur et sans qu'un compte ne bouge.
//
// Le balayage S2 a ramené TOUT le dépôt à la forme canonique `WrittenAtDefaultUTC`.
// Sans ce garde-rail, la prochaine table append-only réintroduirait le défaut : une
// factorisation sans garde-rail re-diverge (règle 6 de CLAUDE.md).
//
// Analyse AST (pas un grep) : seules les CHAÎNES du code sont inspectées, jamais les
// commentaires — un commentaire qui raconte l'ancien défaut n'est pas une régression.
//
// PÉRIMÈTRE ASSUMÉ (2026-08-05) : ce garde-rail couvre la DÉCLARATION du DEFAULT
// (DDL, ALTER, colonne synthétique de rebuild, ajout de colonne) — ce que le lot S2 a
// ramené à la forme canonique. Il NE couvre PAS encore l'autre famille du même défaut :
// les INSERT qui alimentent `written_at` avec un `CURRENT_TIMESTAMP`/`now()` NU
// (shared_social, notifications, records, streaks, prestige, média — 37 sites de
// production relevés le 2026-08-05). Cette famille est CONSIGNÉE comme lot de suivi,
// hors du périmètre fermé de S2 ; l'étendre ici sans corriger les 37 sites ferait
// échouer le test. Quand ce lot passera, ajouter la règle « horloge nue dans un INSERT
// qui alimente written_at » et retirer ce paragraphe.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// horlogeSensibleAuFuseau : les deux expressions SQL qui rendent un TIMESTAMPTZ.
const horlogeSensibleAuFuseau = `(?i)(now\(\)|current_timestamp)`

var (
	// DDL : `written_at TIMESTAMP [NOT NULL] DEFAULT now()`.
	reDefautColonne = regexp.MustCompile(
		`(?i)written_at\s+TIMESTAMP\s+(?:NOT\s+NULL\s+)?DEFAULT\s+(?:now\(\)|current_timestamp)`)
	// Recette append-only (ADR 0026) : `ALTER ... written_at SET DEFAULT now()`.
	reDefautAlter = regexp.MustCompile(
		`(?i)written_at\s+SET\s+DEFAULT\s+(?:now\(\)|current_timestamp)`)
	// Colonne synthétique d'un rebuild : `CURRENT_TIMESTAMP AS written_at`.
	reColonneSynthetique = regexp.MustCompile(
		`(?i)(?:now\(\)|current_timestamp)\s+AS\s+written_at`)
	// Horloge nue, utilisée pour qualifier le DEFAULT d'un ajout de colonne.
	reHorlogeNue = regexp.MustCompile(horlogeSensibleAuFuseau)
	// La forme canonique elle-même — `now()` y figure, il ne faut pas la compter.
	reFormeCanonique = regexp.MustCompile(`(?i)now\(\)\s+AT\s+TIME\s+ZONE\s+'UTC'`)
)

// fixturesLegacyAutorisees — SEULE exemption, posée le 2026-08-05 avec le garde-rail.
// Ce fichier DOIT écrire le DEFAULT fautif : c'est le test qui prouve que la migration
// le répare (sans base legacy, il ne prouverait rien). Aucune autre entrée ne doit être
// ajoutée ici : une base réelle porteuse du défaut se répare, elle ne s'exempte pas.
var fixturesLegacyAutorisees = map[string]bool{
	"internal/migration/steps_written_at_utc_default_integration_test.go": true,
}

// TestWrittenAtEcrituresEnUTC : aucune chaîne du dépôt ne date `written_at` sur le fuseau
// de session. La forme canonique est migration.WrittenAtDefaultUTC.
func TestWrittenAtEcrituresEnUTC(t *testing.T) {
	racine := filepath.Join("..", "..") // apps/go-api
	var infractions []string

	err := filepath.Walk(racine, func(chemin string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if nom := info.Name(); nom == "vendor" || nom == "node_modules" || nom == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(chemin, ".go") {
			return nil
		}
		relatif, rerr := filepath.Rel(racine, chemin)
		if rerr == nil && fixturesLegacyAutorisees[filepath.ToSlash(relatif)] {
			return nil
		}
		infractions = append(infractions, inspecterFichier(t, chemin)...)
		return nil
	})
	if err != nil {
		t.Fatalf("parcours du dépôt: %v", err)
	}

	if len(infractions) > 0 {
		t.Errorf("written_at daté sur le fuseau de session (%d site(s)) — utiliser %s :\n  %s",
			len(infractions), WrittenAtDefaultUTC, strings.Join(infractions, "\n  "))
	}
}

// inspecterFichier retourne les infractions d'un fichier Go (chaînes uniquement).
func inspecterFichier(t *testing.T, chemin string) []string {
	t.Helper()
	fset := token.NewFileSet()
	fichier, err := parser.ParseFile(fset, chemin, nil, 0) // 0 = commentaires ignorés
	if err != nil {
		t.Fatalf("parse %s: %v", chemin, err)
	}
	var infractions []string
	position := func(pos token.Pos) string {
		p := fset.Position(pos)
		return filepath.ToSlash(chemin) + ":" + strconv.Itoa(p.Line)
	}

	ast.Inspect(fichier, func(n ast.Node) bool {
		switch noeud := n.(type) {
		case *ast.BasicLit:
			if noeud.Kind != token.STRING {
				return true
			}
			valeur, err := strconv.Unquote(noeud.Value)
			if err != nil {
				return true
			}
			if motif := motifFautif(valeur); motif != "" {
				infractions = append(infractions, position(noeud.Pos())+" — "+motif)
			}
		case *ast.CallExpr:
			// addColumnIfMissing(db, table, "written_at", "TIMESTAMP DEFAULT now()") :
			// le type et le nom de colonne sont deux arguments distincts, aucune chaîne
			// ne porte les deux — seul l'appel les rapproche.
			if m := motifAjoutColonne(noeud); m != "" {
				infractions = append(infractions, position(noeud.Pos())+" — "+m)
			}
		}
		return true
	})
	return infractions
}

// motifFautif retourne la description de l'infraction portée par une chaîne SQL, ou "".
func motifFautif(sql string) string {
	switch {
	case reDefautColonne.MatchString(sql):
		return "DEFAULT de colonne sensible au fuseau"
	case reDefautAlter.MatchString(sql):
		return "ALTER ... SET DEFAULT sensible au fuseau"
	case reColonneSynthetique.MatchString(sql):
		return "colonne synthétique written_at sensible au fuseau"
	}
	return ""
}

// motifAjoutColonne inspecte un appel add(C|c)olumnIfMissing ciblant written_at.
func motifAjoutColonne(appel *ast.CallExpr) string {
	nom := ""
	switch fn := appel.Fun.(type) {
	case *ast.Ident:
		nom = fn.Name
	case *ast.SelectorExpr:
		nom = fn.Sel.Name
	}
	if !strings.EqualFold(nom, "addColumnIfMissing") || len(appel.Args) < 4 {
		return ""
	}
	if litteral(appel.Args[2]) != "written_at" {
		return ""
	}
	typeSQL := litteral(appel.Args[3])
	if reHorlogeNue.MatchString(typeSQL) && !reFormeCanonique.MatchString(typeSQL) {
		return "ajout de colonne written_at avec un DEFAULT sensible au fuseau"
	}
	return ""
}

// litteral rend la valeur d'une chaîne littérale, ou "" si l'expression n'en est pas une.
func litteral(expr ast.Expr) string {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}
	valeur, err := strconv.Unquote(lit.Value)
	if err != nil {
		return ""
	}
	return valeur
}
