package migration

// arbitration_clocks_utc_guard_test.go — GARDE-RAIL de la forme canonique des HORODATAGES
// (lot S6 ; nomme written_at_utc_guard_test.go jusqu'au lot S5).
//
// `now()` et `CURRENT_TIMESTAMP` rendent un TIMESTAMPTZ que DuckDB coerce vers une colonne
// TIMESTAMP naive par le fuseau de SESSION — a UTC+2 la ligne se date deux heures dans le
// futur. Quand la colonne ARBITRE une vue `<table>_latest` (ADR 0026), cette ligne gagne la
// preseance contre toute ligne UTC ecrite ensuite : la lecture perd l'enrichissement sans
// erreur, sans compteur et sans qu'un compte ne bouge (mecanisme demontre par R1).
//
// CE QUE CE FICHIER TIENT, ET POURQUOI LA CIBLE A CHANGE. Le garde-rail visait le NOM
// `written_at`. C'etait la bonne colonne mais la mauvaise definition : le lot S6 a trouve
// `lusr_component_history`, dont la vue `_latest` arbitre sur `computed_at` — meme defaut,
// meme consequence, invisible d'un detecteur indexe sur un nom. La cible est donc devenue
// la CLASSE « horodatage d'arbitrage ou de tri », et les DEFAULT de TOUTE colonne
// d'horodatage. Deux regles distinctes en decoulent :
//
//	REGLE A — DEFAULT de colonne, pour TOUTE colonne TIMESTAMP naive, quel que soit son
//	  nom. Le critere est le TYPE : la colonne peut devenir arbitrale demain (c'est
//	  exactement ce qui est arrive a `computed_at`), et une ligne dont les horodatages ne
//	  partagent pas une horloge se contredit elle-meme. C'est l'elargissement de S6c : S5
//	  EXEMPTAIT explicitement le DEFAULT local d'une autre colonne, cette exemption est
//	  retiree et les 200 sites DDL correspondants sont passes en forme canonique. Le
//	  suffixe `_at` avait d'abord servi de critere — il laissait passer `last_updated`.
//	REGLE B — ECRITURE/LECTURE d'une colonne de la classe d'arbitrage (colonnesArbitrage).
//	  Plus stricte, donc restreinte aux colonnes dont il est ETABLI SUR PIECES qu'elles
//	  ordonnent une deduplication ou une preseance.
//
// Analyse AST (pas un grep) : seules les CHAINES du code sont inspectees, jamais les
// commentaires — un commentaire qui raconte l'ancien defaut n'est pas une regression.
//
// QUATRE FAMILLES couvertes, aucune laissee dehors :
//
//  1. DECLARATION du DEFAULT — DDL, `ALTER ... SET DEFAULT`, colonne synthetique de
//     rebuild, ajout de colonne (perimetre initial S2, generalise a toute colonne en S6).
//  2. ECRITURE SQL — tout ordre qui alimente une colonne d'arbitrage avec une horloge NUE
//     (`INSERT ... VALUES (..., CURRENT_TIMESTAMP)`, `INSERT ... SELECT ..., now()`,
//     `SET computed_at = now()`). La LECTURE releve de la meme regle : soustraire un
//     TIMESTAMPTZ d'un horodatage naif sous-estime l'age de l'offset.
//  3. ECRITURE Go — une horloge `time.Now()` sans `.UTC()` posee dans un champ/variable
//     nomme d'apres une colonne d'arbitrage. Les chaines SQL ne voient pas cette famille :
//     la valeur y arrive par un parametre lie.
//  4. ECRITURE PAR INTERPOLATION — un fragment d'horloge porte par une variable et injecte
//     (`fmt.Sprintf`) dans un gabarit qui nomme la colonne. Aucune chaine ne porte les deux
//     moities : l'angle mort par lequel le backfill de media_match_associations avait
//     echappe au releve initial des 37 sites.

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

// colonnesArbitrage — les colonnes dont il est ETABLI SUR PIECES qu'elles ordonnent une
// deduplication (`QUALIFY ROW_NUMBER() ... ORDER BY <col>`) ou une preseance de lecture.
// Elles seules relevent de la REGLE B, la plus stricte.
//
// Chacune est justifiee par son site d'arbitrage — pas par son nom :
//   - written_at    : les vues `<table>_latest` de toutes les tables append-only (ADR 0026) ;
//   - computed_at   : `lusr_component_history_latest` (ORDER BY computed_at DESC, id DESC) ;
//   - liked_at      : tri des likers de `media_likes_latest` (GetMediaLikers) ;
//   - snapshot_at   : dedup des snapshots de defis / season pass (home_repo_cache,
//     season_pass_repo) ;
//   - last_seen_at  : dedup des reward tracks et items (season_pass_repo_tracks) ;
//   - associated_at : compare a une borne Go UTC par loadRecentMediaMatchIDs ;
//   - achieved_at   : records joueur, compare a une borne Go UTC ;
//   - archived_at   : bornage des defis d'escouade archives.
//
// AJOUTER UNE COLONNE ICI quand une nouvelle vue `_latest` trie dessus. Ne PAS y ajouter
// une colonne « par prudence » : la regle A couvre deja son DEFAULT, et une entree non
// justifiee rendrait le detecteur bruyant sans rien prouver.
var colonnesArbitrage = []string{
	"written_at", "computed_at", "liked_at", "snapshot_at",
	"last_seen_at", "associated_at", "achieved_at", "archived_at",
}

var (
	// REGLE A — DDL : `<colonne> TIMESTAMP [NOT NULL] DEFAULT now()`, pour N'IMPORTE QUEL
	// nom de colonne. La regle a d'abord vise le suffixe `_at`, et ce suffixe etait un
	// trou : `engagement_coefficients.last_updated` est un horodatage naif au DEFAULT
	// fautif qu'aucun `_at` ne designe (releve par l'invariant de squash, pas par ce
	// garde-rail). Le critere est donc le TYPE, pas le nom — exactement celui du step de
	// reparation, qui est data-driven sans filtre de nom (steps_zz_arbitration_clocks_utc_default).
	//
	// TIMESTAMPTZ est EXCLU a dessein : il conserve l'instant absolu, il n'a pas le defaut,
	// et le convertir changerait sa semantique pour rien. L'exclusion tient au `\s` exige
	// apres TIMESTAMP — `TIMESTAMPTZ` ne matche pas.
	reDefautColonne = regexp.MustCompile(
		`(?i)[a-z_]+\s+TIMESTAMP\s+(?:NOT\s+NULL\s+)?DEFAULT\s+(?:now\(\)|current_timestamp)`)
	// Recette append-only (ADR 0026) : `ALTER ... <col>_at SET DEFAULT now()`.
	reDefautAlter = regexp.MustCompile(
		`(?i)[a-z_]*_at\s+SET\s+DEFAULT\s+(?:now\(\)|current_timestamp)`)
	// Colonne synthetique d'un rebuild : `CURRENT_TIMESTAMP AS written_at`.
	reColonneSynthetique = regexp.MustCompile(
		`(?i)(?:now\(\)|current_timestamp)\s+AS\s+[a-z_]*_at\b`)
	// Horloge nue, utilisee pour qualifier le DEFAULT d'un ajout de colonne.
	reHorlogeNue = regexp.MustCompile(horlogeSensibleAuFuseau)
	// La forme canonique elle-meme — `now()` y figure, il ne faut pas la compter.
	reFormeCanonique = regexp.MustCompile(`(?i)now\(\)\s+AT\s+TIME\s+ZONE\s+'UTC'`)
	// Mention d'une colonne d'arbitrage, quel que soit son role dans l'ordre SQL.
	reMentionColonne = regexp.MustCompile(`(?i)\b(` + strings.Join(colonnesArbitrage, "|") + `)\b`)
	// Chaine SQL entre apostrophes : une horloge y est une DONNEE, pas un appel.
	reLitteralSQL = regexp.MustCompile(`'[^']*'`)
	// Commentaire SQL `-- ...` : du texte, jamais execute. Un DDL peut y expliquer
	// pourquoi une colonne d'arbitrage a ete RETIREE (media_store.go), ce qui n'est pas
	// une mention executable de cette colonne.
	reCommentaireSQL = regexp.MustCompile(`--[^\n]*`)
	// Colonne TIMESTAMPTZ : elle conserve l'instant absolu, `now()` y est CORRECT. La
	// retirer avant examen evite d'accuser un DDL sain voisin d'une colonne d'arbitrage.
	reColonneTZ = regexp.MustCompile(
		`(?i)[a-z_]*\s+TIMESTAMPTZ\s+(?:NOT\s+NULL\s+)?DEFAULT\s+(?:now\(\)|current_timestamp)`)
	// Identifiant Go portant une colonne d'arbitrage (`WrittenAt`, `computedAt`, ...).
	// Les underscores sont retires des DEUX cotes de la comparaison (`written_at` ->
	// `writtenat`), sans quoi le nom Go ne rejoindrait jamais le nom de colonne.
	reIdentArbitrage = regexp.MustCompile(`(?i)^(` +
		strings.ReplaceAll(strings.Join(colonnesArbitrage, "|"), "_", "") + `)$`)
)

// fixturesLegacyAutorisees — les seules exemptions, chacune datee et justifiee. Elles ne
// relevent que de DEUX categories, toutes deux internes a la campagne elle-meme ; aucune
// exemption de code applicatif ne doit etre ajoutee ici. Une base reelle porteuse du
// defaut se repare, elle ne s'exempte pas.
//
//  1. Les fixtures d'INTEGRATION des deux steps de reparation (2026-08-05 pour S2,
//     2026-08-07 pour S6) DOIVENT ecrire le DEFAULT fautif : ce sont les tests qui
//     prouvent que la migration le repare, et sans base legacy ils ne prouveraient rien.
//     Leur exemption est bornee par leur propre assertion : si la reparation cessait
//     d'operer, ils echoueraient.
//  2. Le fichier du DETECTEUR lui-meme (2026-08-07) porte les MOTIFS, qui sont par
//     construction les formes fautives ecrites en toutes lettres. Un detecteur ne peut pas
//     etre son propre sujet. Sa correction reste couverte : les tests de mutation
//     (arbitration_clocks_utc_guard_mutation_test.go) verifient qu'il MORD sur chaque
//     famille — c'est le fichier de mutation, LUI, qui n'est pas exempte : ses formes
//     fautives y sont coupees par concatenation.
var fixturesLegacyAutorisees = map[string]bool{
	"internal/migration/steps_written_at_utc_default_integration_test.go":            true,
	"internal/migration/steps_zz_arbitration_clocks_utc_default_integration_test.go": true,
	"internal/migration/arbitration_clocks_utc_guard_test.go":                        true,
}

// TestWrittenAtEcrituresEnUTC : aucune chaine du depot ne date un horodatage sur le fuseau
// de session. La forme canonique est migration.TimestampDefaultUTC.
//
// Nom conserve depuis S2 bien que le perimetre soit desormais toute la classe : le
// renommer sortirait le test de la baseline `.ai/baselines/tests_pre_migration.jsonl`,
// qui indexe les tests par nom.
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
		t.Errorf("horodatage daté sur le fuseau de session (%d site(s)) — utiliser %s :\n  %s",
			len(infractions), TimestampDefaultUTC, strings.Join(infractions, "\n  "))
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
			// `WrittenAt: time.Now()` / `ComputedAt: time.Now()` dans un littéral de structure.
			if estNomArbitrage(noeud.Key) && estHorlogeGoNonUTC(noeud.Value) {
				infractions = append(infractions,
					position(noeud.Pos())+" — champ d'arbitrage daté sur l'horloge locale")
			}
		case *ast.AssignStmt:
			// `writtenAt := time.Now()` / `computedAt = time.Now()`.
			for i, cible := range noeud.Lhs {
				if i < len(noeud.Rhs) && estNomArbitrage(cible) && estHorlogeGoNonUTC(noeud.Rhs[i]) {
					infractions = append(infractions,
						position(noeud.Pos())+" — variable d'arbitrage datée sur l'horloge locale")
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
		return "colonne synthétique d'horodatage sensible au fuseau"
	case horlogeNueAvecColonneArbitrage(sql):
		return "horloge nue dans un ordre SQL portant une colonne d'arbitrage"
	}
	return ""
}

// horlogeNueAvecColonneArbitrage : la chaîne nomme une colonne d'arbitrage ET porte une
// horloge sensible au fuseau AILLEURS que dans la forme canonique (RÈGLE B).
//
// Les colonnes et les valeurs d'un INSERT sont deux listes distantes — aucun motif local
// ne les rapproche, et la valeur peut aussi venir d'un `SELECT`. La règle est donc posée
// sur l'INSTRUCTION ENTIÈRE : dans ce dépôt, un ordre SQL qui nomme une colonne
// d'arbitrage et invoque `now()`/`CURRENT_TIMESTAMP` hors forme canonique mélange deux
// horloges — qu'il écrive la colonne ou qu'il la compare.
//
// Quatre retraits avant l'examen, chacun contre un faux positif CONSTATÉ :
//   - la forme canonique, sans quoi le `now()` qu'elle contient déclencherait la règle
//     contre elle-même ;
//   - les commentaires SQL `-- ...` : le CREATE de media_files explique en commentaire que
//     `liked_at` a été RETIRÉE de la table — une colonne nommée pour dire qu'elle n'existe
//     plus n'est pas une mention exécutable ;
//   - les chaînes SQL entre apostrophes : `column_default LIKE '%now()%'` (le prédicat qui
//     DÉTECTE le défaut, steps_zz_arbitration_clocks_utc_default.go) nomme l'horloge sans
//     jamais l'appeler ;
//   - les colonnes TIMESTAMPTZ : `indexed_at TIMESTAMPTZ DEFAULT NOW()` est CORRECT (le
//     type conserve l'instant absolu), et le convertir changerait sa sémantique.
//
// Le retrait `DEFAULT <horloge>` d'une AUTRE colonne, lui, a DISPARU avec le lot S6 : ce
// cas n'est plus un faux positif mais une infraction à la règle A, et les 194 sites DDL
// concernés sont passés en forme canonique.
//
// L'examen porte sur CHAQUE INSTRUCTION, pas sur la chaîne : un script de schéma de test
// enchaîne un `CREATE VIEW` qui trie sur `written_at` et un `INSERT` sur une autre table —
// deux instructions séparées, aucun mélange d'horloge sur une même ligne.
func horlogeNueAvecColonneArbitrage(sql string) bool {
	sql = reCommentaireSQL.ReplaceAllString(sql, "")
	if !reMentionColonne.MatchString(sql) {
		return false
	}
	for _, instruction := range strings.Split(sql, ";") {
		if !reMentionColonne.MatchString(instruction) {
			continue
		}
		reste := reFormeCanonique.ReplaceAllString(instruction, "")
		reste = reLitteralSQL.ReplaceAllString(reste, "")
		reste = reColonneTZ.ReplaceAllString(reste, "")
		if reHorlogeNue.MatchString(reste) {
			return true
		}
	}
	return false
}

// motifAjoutColonne inspecte un appel add(C|c)olumnIfMissing ciblant un horodatage.
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
	if !strings.HasSuffix(litteral(appel.Args[2]), "_at") {
		return ""
	}
	typeSQL := litteral(appel.Args[3])
	if reHorlogeNue.MatchString(typeSQL) && !reFormeCanonique.MatchString(typeSQL) {
		return "ajout de colonne d'horodatage avec un DEFAULT sensible au fuseau"
	}
	return ""
}

// variablesPortantUneHorlogeNue relève les variables du fichier dont la valeur est un
// FRAGMENT SQL contenant une horloge sensible au fuseau (`atExpr := "CURRENT_TIMESTAMP"`).
//
// Sans elles, l'analyse par chaîne a un angle mort : quand l'ordre SQL est assemblé par
// `fmt.Sprintf`, aucune chaîne ne porte à la fois la colonne et l'horloge — c'est
// exactement ainsi que le backfill de media_match_associations avait échappé au relevé
// initial des 37 sites, alors qu'il datait bien written_at sur le fuseau de session.
//
// Les fragments qui nomment déjà une colonne d'arbitrage sont ignorés : ils relèvent des
// règles par chaîne, qui les décrivent plus précisément.
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

// motifInterpolation : un appel rapproche un gabarit SQL nommant une colonne d'arbitrage
// et une variable porteuse d'horloge nue. Le rapprochement suffit — quel que soit le rang
// des arguments, la seule raison d'interpoler un fragment d'horloge dans ce gabarit est de
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
			return "horloge nue interpolée dans un ordre SQL portant une colonne d'arbitrage (" + nom.Name + ")"
		}
	}
	return ""
}

// estNomArbitrage : l'expression est l'identifiant du champ/de la variable qui porte une
// colonne d'arbitrage (`WrittenAt`, `computedAt`, `r.LikedAt`). La comparaison ignore la
// casse et les underscores — `written_at`, `writtenAt` et `WrittenAt` sont le même nom.
func estNomArbitrage(expr ast.Expr) bool {
	nom := ""
	switch e := expr.(type) {
	case *ast.Ident:
		nom = e.Name
	case *ast.SelectorExpr:
		nom = e.Sel.Name
	default:
		return false
	}
	return reIdentArbitrage.MatchString(strings.ReplaceAll(nom, "_", ""))
}

// estHorlogeGoNonUTC : l'expression est exactement `time.Now()`, sans conversion.
//
// `time.Now()` rend un instant PORTEUR de son fuseau ; le driver DuckDB l'écrit dans une
// colonne TIMESTAMP naïve en projetant sur ce fuseau — à UTC+2, deux heures dans le futur,
// exactement le défaut du DEFAULT. `time.Now().UTC()` n'est pas visé : le nœud examiné est
// alors l'appel à `.UTC()`, dont le sélecteur n'est pas `Now`.
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
