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
// PÉRIMÈTRE (étendu le 2026-08-07, lot S5). Le garde-rail couvre les QUATRE familles du
// même défaut, plus aucune n'est laissée dehors :
//
//  1. DÉCLARATION du DEFAULT — DDL, `ALTER ... SET DEFAULT`, colonne synthétique de
//     rebuild, ajout de colonne (périmètre initial S2).
//  2. ÉCRITURE SQL — tout ordre qui alimente `written_at` avec une horloge NUE
//     (`INSERT ... VALUES (..., CURRENT_TIMESTAMP)`, `INSERT ... SELECT ..., now()`,
//     `SET written_at = now()`). C'est la famille consignée par S2 et corrigée par S5.
//  3. ÉCRITURE Go — une horloge `time.Now()` sans `.UTC()` posée dans un champ/variable
//     `WrittenAt`. Les chaînes SQL ne voient pas cette famille : la valeur y arrive par
//     un paramètre lié.
//  4. ÉCRITURE PAR INTERPOLATION — un fragment d'horloge porté par une variable et
//     injecté (`fmt.Sprintf`) dans un gabarit qui nomme `written_at`. Aucune chaîne ne
//     porte les deux moitiés : c'est l'angle mort par lequel le backfill de
//     media_match_associations avait échappé au relevé initial des 37 sites.
//
// Le défaut de la famille 2 est le MÊME que celui de la famille 1, sans le DEFAULT : la
// valeur écrite explicitement par `CURRENT_TIMESTAMP` subit la même coercition de fuseau
// que celle écrite par le DEFAULT. Une base peut donc porter un DEFAULT canonique et,
// malgré cela, des lignes datées au fuseau local — S2 seul ne suffisait pas.
//
// La lecture relève de la même erreur (soustraire un TIMESTAMPTZ d'un `written_at` naïf
// sous-estime l'âge de l'offset) et tombe sous la règle 2 : une chaîne qui nomme
// `written_at` et une horloge nue est fautive, qu'elle écrive ou qu'elle lise.

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
	// Mention de la colonne, quel que soit le rôle qu'elle joue dans l'ordre SQL.
	reMentionColonne = regexp.MustCompile(`(?i)\bwritten_at\b`)
	// Chaîne SQL entre apostrophes : une horloge y est une DONNÉE, pas un appel.
	reLitteralSQL = regexp.MustCompile(`'[^']*'`)
	// DEFAULT d'une colonne. Quand il porte sur `written_at` il est déjà attrapé par
	// les motifs précis ci-dessus ; ici il porte donc sur une AUTRE colonne.
	reDefautAutreColonne = regexp.MustCompile(
		`(?i)DEFAULT\s+(?:now\(\)|current_timestamp)`)
)

// fixturesLegacyAutorisees — les DEUX seules exemptions, chacune datée et justifiée.
// Aucune troisième ne doit être ajoutée : une base réelle porteuse du défaut se répare,
// elle ne s'exempte pas.
//
//   - la fixture d'intégration (2026-08-05) DOIT écrire le DEFAULT fautif : c'est le
//     test qui prouve que la migration le répare — sans base legacy, il ne prouverait
//     rien ;
//   - ce fichier-ci (2026-08-07) porte les MOTIFS de détection, qui sont par
//     construction les formes fautives écrites en toutes lettres. Un détecteur ne peut
//     pas être son propre sujet. Sa correction reste couverte : les tests de mutation
//     (written_at_utc_guard_mutation_test.go) vérifient qu'il MORD sur chaque famille.
var fixturesLegacyAutorisees = map[string]bool{
	"internal/migration/steps_written_at_utc_default_integration_test.go": true,
	"internal/migration/written_at_utc_guard_test.go":                     true,
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
	porteurs := variablesPortantUneHorlogeNue(fichier)

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
			// fmt.Sprintf(`INSERT ... written_at ... %s`, atExpr) : l'horloge est
			// dans une AUTRE chaîne, l'ordre SQL n'existe qu'après interpolation.
			if m := motifInterpolation(noeud, porteurs); m != "" {
				infractions = append(infractions, position(noeud.Pos())+" — "+m)
			}
		case *ast.KeyValueExpr:
			// `WrittenAt: time.Now()` dans un littéral de structure.
			if estNomWrittenAt(noeud.Key) && estHorlogeGoNonUTC(noeud.Value) {
				infractions = append(infractions,
					position(noeud.Pos())+" — champ WrittenAt daté sur l'horloge locale")
			}
		case *ast.AssignStmt:
			// `writtenAt := time.Now()` / `writtenAt = time.Now()`.
			for i, cible := range noeud.Lhs {
				if i < len(noeud.Rhs) && estNomWrittenAt(cible) && estHorlogeGoNonUTC(noeud.Rhs[i]) {
					infractions = append(infractions,
						position(noeud.Pos())+" — writtenAt daté sur l'horloge locale")
				}
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
	case horlogeNueAvecWrittenAt(sql):
		return "horloge nue dans un ordre SQL portant written_at"
	}
	return ""
}

// horlogeNueAvecWrittenAt : la chaîne nomme `written_at` ET porte une horloge
// sensible au fuseau AILLEURS que dans la forme canonique.
//
// Les colonnes et les valeurs d'un INSERT sont deux listes distantes — aucun motif
// local ne les rapproche, et la valeur peut aussi venir d'un `SELECT`. La règle est
// donc posée sur la CHAÎNE ENTIÈRE : dans ce dépôt, un ordre SQL qui nomme
// `written_at` et invoque `now()`/`CURRENT_TIMESTAMP` hors forme canonique mélange
// deux horloges — qu'il écrive la colonne ou qu'il la compare.
//
// Trois retraits avant l'examen, chacun contre un faux positif CONSTATÉ :
//   - la forme canonique, sans quoi le `now()` qu'elle contient déclencherait la
//     règle contre elle-même ;
//   - les chaînes SQL entre apostrophes : `column_default LIKE '%now()%'` (le
//     prédicat qui DÉTECTE le défaut, steps_written_at_utc_default.go) nomme
//     l'horloge sans jamais l'appeler ;
//   - les `DEFAULT <horloge>`, qui portent forcément sur une AUTRE colonne à ce
//     stade du switch — un DDL peut légitimement dater `created_at` sur le fuseau
//     local à côté d'un `written_at` canonique (steps_player_schema_authority.go).
//     `created_at` n'arbitre aucune vue `_latest` : le corriger déborderait S5.
//
// L'examen porte sur CHAQUE INSTRUCTION, pas sur la chaîne : un script de schéma de
// test enchaîne un `CREATE VIEW` qui trie sur `written_at` et un `INSERT` qui date
// `liked_at` sur l'horloge locale — deux instructions séparées, aucun mélange
// d'horloge sur une même ligne (media_delete_test.go).
func horlogeNueAvecWrittenAt(sql string) bool {
	if !reMentionColonne.MatchString(sql) {
		return false
	}
	for _, instruction := range strings.Split(sql, ";") {
		if !reMentionColonne.MatchString(instruction) {
			continue
		}
		reste := reFormeCanonique.ReplaceAllString(instruction, "")
		reste = reLitteralSQL.ReplaceAllString(reste, "")
		reste = reDefautAutreColonne.ReplaceAllString(reste, "")
		if reHorlogeNue.MatchString(reste) {
			return true
		}
	}
	return false
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

// variablesPortantUneHorlogeNue relève les variables du fichier dont la valeur est un
// FRAGMENT SQL contenant une horloge sensible au fuseau (`atExpr := "CURRENT_TIMESTAMP"`).
//
// Sans elles, l'analyse par chaîne a un angle mort : quand l'ordre SQL est assemblé par
// `fmt.Sprintf`, aucune chaîne ne porte à la fois `written_at` et l'horloge — c'est
// exactement ainsi que le backfill de media_match_associations avait échappé au relevé
// initial des 37 sites, alors qu'il datait bien written_at sur le fuseau de session.
//
// Les fragments qui nomment déjà `written_at` sont ignorés : ils relèvent des règles par
// chaîne, qui les décrivent plus précisément.
func variablesPortantUneHorlogeNue(fichier *ast.File) map[string]bool {
	porteurs := map[string]bool{}
	noter := func(cible ast.Expr, valeur ast.Expr) {
		nom, ok := cible.(*ast.Ident)
		if !ok {
			return
		}
		fragment := litteral(valeur)
		if fragment == "" || reMentionColonne.MatchString(fragment) {
			return
		}
		if reHorlogeNue.MatchString(reFormeCanonique.ReplaceAllString(fragment, "")) {
			porteurs[nom.Name] = true
		}
	}
	ast.Inspect(fichier, func(n ast.Node) bool {
		switch noeud := n.(type) {
		case *ast.AssignStmt:
			for i, cible := range noeud.Lhs {
				if i < len(noeud.Rhs) {
					noter(cible, noeud.Rhs[i])
				}
			}
		case *ast.ValueSpec:
			for i, nom := range noeud.Names {
				if i < len(noeud.Values) {
					noter(nom, noeud.Values[i])
				}
			}
		}
		return true
	})
	return porteurs
}

// motifInterpolation : un appel rapproche un gabarit SQL nommant `written_at` et une
// variable porteuse d'horloge nue. Le rapprochement suffit — quel que soit le rang des
// arguments, la seule raison d'interpoler un fragment d'horloge dans ce gabarit est de
// le faire exécuter.
func motifInterpolation(appel *ast.CallExpr, porteurs map[string]bool) string {
	if len(porteurs) == 0 {
		return ""
	}
	gabarit := false
	for _, arg := range appel.Args {
		if reMentionColonne.MatchString(litteral(arg)) {
			gabarit = true
			break
		}
	}
	if !gabarit {
		return ""
	}
	for _, arg := range appel.Args {
		if nom, ok := arg.(*ast.Ident); ok && porteurs[nom.Name] {
			return "horloge nue interpolée dans un ordre SQL portant written_at (" + nom.Name + ")"
		}
	}
	return ""
}

// estNomWrittenAt : l'expression est l'identifiant du champ/de la variable qui porte
// la colonne (`WrittenAt`, `writtenAt`, `r.WrittenAt`).
func estNomWrittenAt(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.Ident:
		return strings.EqualFold(e.Name, "writtenAt")
	case *ast.SelectorExpr:
		return strings.EqualFold(e.Sel.Name, "writtenAt")
	}
	return false
}

// estHorlogeGoNonUTC : l'expression est exactement `time.Now()`, sans conversion.
//
// `time.Now()` rend un instant PORTEUR de son fuseau ; le driver DuckDB l'écrit dans
// une colonne TIMESTAMP naïve en projetant sur ce fuseau — à UTC+2, deux heures dans
// le futur, exactement le défaut du DEFAULT. `time.Now().UTC()` n'est pas visé : le
// nœud examiné est alors l'appel à `.UTC()`, dont le sélecteur n'est pas `Now`.
func estHorlogeGoNonUTC(expr ast.Expr) bool {
	appel, ok := expr.(*ast.CallExpr)
	if !ok || len(appel.Args) != 0 {
		return false
	}
	sel, ok := appel.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Now" {
		return false
	}
	paquet, ok := sel.X.(*ast.Ident)
	return ok && paquet.Name == "time"
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
