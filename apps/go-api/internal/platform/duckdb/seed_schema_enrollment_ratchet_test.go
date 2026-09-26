// Package duckdb — seed_schema_enrollment_ratchet_test.go : cliquet de
// COMPLETUDE du garde-rail de parité de seeds (seed_schema_column_parity_test.go).
//
// Le garde-rail voisin vérifie que les colonnes lues par TROIS requêtes enrôlées
// existent dans les seeds. Sa faiblesse assumée : l'enrôlement est manuel. Une
// nouvelle requête de prod lisant match_registry / match_participants sur ces
// seeds n'est couverte par rien, et personne n'est prévenu — on retombe alors
// sur le binder error d'origine.
//
// Ce cliquet ferme ce point SANS allowlist par nom de requête (qui deviendrait
// une liste à rallonge, donc un tampon). Il raisonne par COLONNE, c'est-à-dire
// sur le risque réel : toute colonne lue sous l'alias du registre ou des
// participants par UNE QUELCONQUE requête du package doit être lue aussi par au
// moins une requête enrôlée — sinon elle n'est protégée par aucun seed.
//
// Conséquence pratique, et c'est ce qui rend le cliquet tenable : ajouter une
// requête qui ne lit que des colonnes déjà couvertes ne demande AUCUNE action.
// Mesuré au 2026-08-30 : Q30SquadMatchesSharedQuery (escouade), non enrôlée,
// n'introduit aucune colonne nouvelle — les trois requêtes enrôlées sont les
// lecteurs larges (playerMatchesSharedBaseSelect projette 42 colonnes).
//
// Quand ce cliquet casse : une requête lit une colonne que le garde-rail ne
// protège pas. Deux issues, à STATUER (jamais à ignorer) :
//  1. la requête tourne sur ces seeds → l'ENROLER dans la table de cas de
//     TestSeedSchemaColumnParity (une ligne) ;
//  2. elle n'y tourne pas → l'ajouter à ratchetColonnesHorsPerimetre avec la
//     raison ET la date, comme toute allowlist de ce dépôt.
package duckdb

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// ratchetBindingRE capture TOUTE liaison `FROM|JOIN <source> [AS] <alias>` —
// pas seulement celles des tables visées. Indispensable pour détecter les
// alias RELIÉS (cf. ratchetAliasBinding).
var ratchetBindingRE = regexp.MustCompile(`(?i)(?:FROM|JOIN)\s+([A-Za-z_][\w.]*)\s+(?:AS\s+)?([a-z][a-z0-9_]*)`)

// ratchetAliasMotsCles : mots-clés SQL qui suivent une table sans être un alias.
var ratchetAliasMotsCles = map[string]bool{
	"on": true, "where": true, "group": true, "order": true, "limit": true,
	"using": true, "left": true, "right": true, "inner": true, "join": true,
	"qualify": true, "cross": true, "full": true, "outer": true, "having": true,
}

// ratchetAliasBinding retourne les alias liés à l'une des `tables` dans la
// requête — et UNIQUEMENT ceux dont la liaison est NON AMBIGUE.
//
// Le piège qui a motivé cette précision (mesuré le 2026-08-30 sur
// Q29HistoryForAvg) : un même alias peut être lié à la table visée PUIS à une
// CTE dans la même requête (`FROM match_participants p ... LEFT JOIN perfect p`).
// Les références `p.<col>` deviennent alors inattribuables — `p.perfect_kills`
// y désigne la CTE, pas une colonne de match_participants. Compter un alias
// relié produirait un faux positif, c'est-à-dire un cliquet qu'on apprend à
// ignorer : on l'écarte plutôt, quitte à couvrir un peu moins.
func ratchetAliasBinding(sqlText string, tables []string) []string {
	cible := map[string]bool{}
	for _, tbl := range tables {
		cible[strings.ToLower(tbl)] = true
	}
	sources := map[string]map[string]bool{}
	var ordre []string
	for _, m := range ratchetBindingRE.FindAllStringSubmatch(sqlText, -1) {
		source, alias := strings.ToLower(m[1]), strings.ToLower(m[2])
		if ratchetAliasMotsCles[alias] {
			continue
		}
		if sources[alias] == nil {
			sources[alias] = map[string]bool{}
			ordre = append(ordre, alias)
		}
		sources[alias][source] = true
	}
	var out []string
	for _, alias := range ordre {
		if len(sources[alias]) != 1 {
			continue // alias relié : références inattribuables
		}
		for source := range sources[alias] {
			if cible[source] {
				out = append(out, alias)
			}
		}
	}
	return out
}

// ratchetQueryStrings retourne, pour chaque déclaration `var`/`const` de niveau
// paquet des fichiers NON-test du répertoire, la concaténation de ses littéraux
// chaîne. Les appels intercalés (StartTimeCanonicalSQL("r"), templates %s) sont
// ignorés : seules les parties littérales portent les noms de tables/colonnes
// qui nous intéressent.
// (parser.ParseFile fichier par fichier plutôt que parser.ParseDir, déprécié
// depuis Go 1.22 — on n'a pas besoin de sa carte de paquets, seulement des
// déclarations de chaque fichier.)
func ratchetQueryStrings(t *testing.T, dir string) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("lecture du répertoire %s : %v", dir, err)
	}
	fset := token.NewFileSet()
	out := map[string]string{}
	for _, e := range entries {
		nom := e.Name()
		if e.IsDir() || !strings.HasSuffix(nom, ".go") || strings.HasSuffix(nom, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(dir, nom), nil, 0)
		if err != nil {
			t.Fatalf("parsing de %s : %v", nom, err)
		}
		ratchetCollectDecls(file, out)
	}
	return out
}

// ratchetCollectDecls ajoute à `out` les littéraux chaîne concaténés de chaque
// déclaration `var`/`const` de niveau paquet du fichier.
func ratchetCollectDecls(file *ast.File, out map[string]string) {
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || (gen.Tok != token.VAR && gen.Tok != token.CONST) {
			continue
		}
		for _, spec := range gen.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || len(vs.Names) == 0 || len(vs.Values) == 0 {
				continue
			}
			var sb strings.Builder
			ast.Inspect(vs.Values[0], func(n ast.Node) bool {
				if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					if s, err := strconv.Unquote(lit.Value); err == nil {
						sb.WriteString(s)
						sb.WriteByte('\n')
					}
				}
				return true
			})
			if sb.Len() > 0 {
				out[vs.Names[0].Name] = sb.String()
			}
		}
	}
}

// ratchetColonnesHorsPerimetre : colonnes lues par une requête du package mais
// par AUCUNE requête enrôlée, dont on a établi qu'elles ne risquent rien parce
// que leur requête ne s'exécute pas sur les seeds gardés. Toute entrée porte sa
// raison et sa date (règle du dépôt : pas d'allowlist sans justification datée).
//
// PREUVE COMMUNE aux trois entrées (mesurée le 2026-08-30) : la colonne est
// absente des TROIS seeds (`grep` sur player_repos_test.go et
// pool_migration_test.go : aucune occurrence) ALORS QUE la suite d'intégration
// est verte. Si le chemin qui exécute ces requêtes était couvert par ces seeds,
// il tomberait déjà en binder error. Elles ne peuvent donc pas être enrôlées :
// l'enrôlement rendrait le garde-rail rouge sans qu'aucun test ne soit en
// danger. Le jour où un test exercera l'un de ces chemins, il tombera — et ce
// sera le signal d'enrôler la requête ET d'ajouter la colonne au seed.
var ratchetColonnesHorsPerimetre = map[string]string{
	"backfill_bits": "2026-08-30 — lue par Q17PlayerMatchStats (vue match, mono-match). " +
		"Absente des trois seeds, suite verte : ce chemin n'est pas couvert par eux.",
	"present_at_beginning": "2026-08-30 — lue par qElapsedSecondsByMatchTpl (temps de jeu effectif). " +
		"Absente des trois seeds, suite verte : ce chemin n'est pas couvert par eux.",
	"present_at_completion": "2026-08-30 — lue par qElapsedSecondsByMatchTpl (temps de jeu effectif). " +
		"Absente des trois seeds, suite verte : ce chemin n'est pas couvert par eux.",
}

// TestSeedParityEnrollmentRatchet : le cliquet de complétude. Toute colonne lue
// sous l'alias registre/participants par une requête du package doit l'être
// aussi par une requête enrôlée dans TestSeedSchemaColumnParity — sinon elle
// n'est protégée par aucun seed et la prochaine migration rejouera l'incident.
func TestSeedParityEnrollmentRatchet(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	queries := ratchetQueryStrings(t, filepath.Dir(thisFile))
	if len(queries) < 50 {
		t.Fatalf("seulement %d déclarations chaîne analysées — extracteur AST cassé ?", len(queries))
	}

	registryTables := []string{"match_registry", "v_match_full"}
	participantsTables := []string{"match_participants"}

	// Couverture offerte par l'enrôlement, DERIVEE de seedParityCarte — la même
	// source unique que le garde-rail voisin, jamais une seconde liste (les deux
	// se désynchroniseraient au premier ajout : constaté pendant l'écriture de
	// ce lot, Q12 enrôlée d'un côté et toujours signalée de l'autre).
	// Les valeurs sont celles d'EXECUTION : les fragments concaténés comme
	// StartTimeCanonicalSQL y sont déjà résolus.
	couvert := map[string]bool{}
	for _, seed := range seedParityCarte {
		for _, q := range seed.queries {
			for col := range seedParityAliasRefs(q.sql, q.alias) {
				couvert[col] = true
			}
		}
	}
	if len(couvert) < 20 {
		t.Fatalf("seulement %d colonnes couvertes extraites des requêtes enrôlées — extracteur cassé ?", len(couvert))
	}

	analysees := 0
	for name, sqlText := range queries {
		for _, tables := range [][]string{registryTables, participantsTables} {
			aliases := ratchetAliasBinding(sqlText, tables)
			if len(aliases) > 0 {
				analysees++
			}
			for _, alias := range aliases {
				for _, col := range seedParitySortedKeys(seedParityAliasRefs(sqlText, alias)) {
					if couvert[col] || ratchetColonnesHorsPerimetre[col] != "" {
						continue
					}
					t.Errorf("colonne %q lue par %s (alias %s) mais par AUCUNE requête enrôlée — "+
						"elle n'est protégée par aucun seed : soit ENROLER %s dans la table de cas de "+
						"TestSeedSchemaColumnParity, soit (si elle ne tourne pas sur ces seeds) l'ajouter à "+
						"ratchetColonnesHorsPerimetre avec raison et date",
						col, name, alias, name)
				}
			}
		}
	}
	if analysees == 0 {
		t.Fatal("aucune requête liant registre/participants à un alias — détection d'alias cassée ?")
	}
	t.Logf("cliquet : %d liaisons requête/table analysées, %d colonnes couvertes par l'enrôlement",
		analysees, len(couvert))
}

// TestSeedParityEnrollmentRatchet_DetectionAlias verrouille la détection d'alias
// sur des cas fabriqués : AS explicite, alias numéroté, mot-clé pris pour un
// alias, et table non ciblée.
func TestSeedParityEnrollmentRatchet_DetectionAlias(t *testing.T) {
	cases := []struct {
		nom, sqlText, want string
		tables             []string
	}{
		{"alias simple", "SELECT 1 FROM v_match_full r JOIN x ON 1=1", "r", []string{"match_registry", "v_match_full"}},
		{"AS explicite", "SELECT 1 FROM match_registry AS reg", "reg", []string{"match_registry", "v_match_full"}},
		{"alias numérotés", "FROM match_participants p1 JOIN match_participants p2 ON 1=1", "p1,p2", []string{"match_participants"}},
		{"mot-clé non pris pour alias", "SELECT 1 FROM match_registry WHERE x = 1", "", []string{"match_registry"}},
		{"table non ciblée", "SELECT 1 FROM match_skill_rank r", "", []string{"match_registry", "v_match_full"}},
		// Le cas Q29HistoryForAvg : `p` est lié à match_participants PUIS à une
		// CTE. Références inattribuables → l'alias doit être écarté.
		{"alias relié à une CTE", "FROM match_participants p JOIN x ON 1=1 ... LEFT JOIN perfect p ON 1=1", "", []string{"match_participants"}},
	}
	for _, tc := range cases {
		got := strings.Join(ratchetAliasBinding(tc.sqlText, tc.tables), ",")
		if got != tc.want {
			t.Errorf("%s : alias %q, attendu %q", tc.nom, got, tc.want)
		}
	}
}
