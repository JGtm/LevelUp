// Package duckdb — seed_schema_column_parity_test.go : garde-rail de parité de
// colonnes entre les seeds de test qui recréent le schéma shared A LA MAIN et
// les requêtes de production qui le lisent (même famille que
// persist/demo_seed_columns_test.go et ops/seed_demo_column_parity_test.go,
// doctrine appliquée ici aux seeds d'intégration de CE package).
//
// Incident 2026-08-30 : la migration ADR 0032 add_team_rounds_to_match_registry
// a ajouté team_0_rounds_won / team_1_rounds_won / rounds_total, aussitôt lues
// par Q5SharedHistory, Q13MatchMeta et playerMatchesSharedBaseSelect — et ~20
// tests d'intégration du package sont tombés en « Binder Error » cryptiques
// parce que seedPlayerSchema, seedSharedDBSchema et seedSharedDBForPoolTest
// recréaient match_registry sans ces colonnes (corrigés à la main par
// e484a68ae). Ce fichier transforme la prochaine occurrence en UN échec
// explicite : colonne + requête + seed. Volontairement SANS tag build : il
// tourne dans la suite rapide ET sous -tags=integration.
//
// Politique — PLANCHER, pas parité totale : les seeds sont volontairement
// minimaux (seedSharedDBForPoolTest ne porte que ce que LoadAll lit). Le
// contrat : toute colonne référencée sous l'alias registre (r) ou participants
// (p) par une requête ENROLEE sur un seed doit exister dans le CREATE TABLE
// correspondant de ce seed. v_match_full étant un SELECT * de match_registry
// dans les trois seeds, les références r.* se vérifient contre match_registry.
// La clause d'exclusion campagne (token /*__EXCLUDE_CAMPAIGN__*/) ne référence
// que <alias>.game_variant_id, déjà projetée par Q5 : l'extraction sur les
// chaînes brutes ne perd aucune colonne.
//
// Quand ce test casse après une migration shared :
//  1. ajouter la colonne au CREATE TABLE des seeds nommés par l'échec
//     (modèle : commit e484a68ae) ;
//  2. OU, si la requête ne s'exécute réellement pas sur ce seed, retirer le
//     couple de la table de cas avec justification datée (voir les
//     désenrôlements assumés dans TestSeedSchemaColumnParity).
//
// Nouvelle requête prod lisant registry/participants sur ces seeds : l'ENROLER
// (une entrée de plus dans la table de cas).
package duckdb

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// seedParityStripSQLComments retire les commentaires `-- ...` (fin de ligne) :
// une colonne citée en prose ne doit pas être comptée, ni en masquer une autre.
func seedParityStripSQLComments(s string) string {
	var b strings.Builder
	for _, line := range strings.Split(s, "\n") {
		if i := strings.Index(line, "--"); i >= 0 {
			line = line[:i]
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

// seedParityAliasRefs extrait, commentaires exclus, l'ensemble des colonnes
// référencées `<alias>.<colonne>` dans une requête SQL.
func seedParityAliasRefs(sqlText, alias string) map[string]bool {
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(alias) + `\.([A-Za-z_][A-Za-z0-9_]*)`)
	out := map[string]bool{}
	for _, m := range re.FindAllStringSubmatch(seedParityStripSQLComments(sqlText), -1) {
		out[m[1]] = true
	}
	return out
}

// seedParityFuncBody isole le texte d'une fonction top-level `func <name>(`
// jusqu'au prochain `\nfunc ` (ou la fin du fichier) — deux seeds du même
// fichier peuvent créer la même table, le périmètre par fonction est essentiel.
func seedParityFuncBody(t *testing.T, src, funcName string) string {
	t.Helper()
	start := strings.Index(src, "func "+funcName+"(")
	if start < 0 {
		t.Fatalf("fonction %s introuvable (seed renommé ? mettre à jour la table de cas)", funcName)
	}
	rest := src[start:]
	if end := strings.Index(rest[1:], "\nfunc "); end >= 0 {
		return rest[:end+1]
	}
	return rest
}

// seedParitySplitTopLevel découpe sur les virgules de profondeur 0 — les
// virgules internes de DECIMAL(10,2) ou DEFAULT f(x, y) ne séparent rien.
func seedParitySplitTopLevel(s string) []string {
	var out []string
	depth, start := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, s[start:i])
				start = i + 1
			}
		}
	}
	return append(out, s[start:])
}

// seedParityCreateColumns retourne les colonnes du `CREATE TABLE
// [shared.]<table> (...)` trouvé dans le corps de la fonction seed donnée
// (contraintes de table PRIMARY/FOREIGN/UNIQUE/CHECK/CONSTRAINT ignorées).
func seedParityCreateColumns(t *testing.T, src, funcName, table string) map[string]bool {
	t.Helper()
	body := seedParityStripSQLComments(seedParityFuncBody(t, src, funcName))
	re := regexp.MustCompile(`CREATE TABLE (?:IF NOT EXISTS )?(?:shared\.)?` + regexp.QuoteMeta(table) + `\s*\(`)
	loc := re.FindStringIndex(body)
	if loc == nil {
		t.Fatalf("%s : CREATE TABLE %s introuvable dans le corps du seed", funcName, table)
	}
	depth, end := 1, -1
	for i := loc[1]; i < len(body) && end < 0; i++ {
		switch body[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				end = i
			}
		}
	}
	if end < 0 {
		t.Fatalf("%s : CREATE TABLE %s — parenthèse fermante introuvable", funcName, table)
	}
	cols := map[string]bool{}
	for _, entry := range seedParitySplitTopLevel(body[loc[1]:end]) {
		fields := strings.Fields(entry)
		if len(fields) == 0 {
			continue
		}
		switch strings.ToUpper(fields[0]) {
		case "PRIMARY", "FOREIGN", "UNIQUE", "CHECK", "CONSTRAINT":
			continue
		}
		cols[fields[0]] = true
	}
	if len(cols) == 0 {
		t.Fatalf("%s : CREATE TABLE %s — aucune colonne extraite (format inattendu ?)", funcName, table)
	}
	return cols
}

func seedParitySortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// seedParityQuery : une requête prod enrôlée sur un seed — l'alias détermine la
// table du seed contre laquelle ses références sont vérifiées.
type seedParityQuery struct {
	name  string // nom de la var Go, pour le message d'échec
	sql   string
	alias string // "r" (registre / v_match_full) ou "p" (participants)
	table string
}

// seedParitySeed : un seed de test et les requêtes de prod qui s'exécutent
// réellement dessus.
type seedParitySeed struct {
	seedFunc, file string
	queries        []seedParityQuery
}

// seedParityCarte : LA SOURCE UNIQUE de l'enrôlement — carte véridique (verte au
// 2026-08-30) de quelle requête tourne sur quel seed. TestSeedSchemaColumnParity
// la parcourt pour vérifier les seeds ; TestSeedParityEnrollmentRatchet
// (seed_schema_enrollment_ratchet_test.go) en dérive la couverture offerte.
// Deux listes séparées se seraient désynchronisées dès le premier ajout — c'est
// arrivé pendant l'écriture de ce lot, d'où cette centralisation (règle CLAUDE.md
// n°6 : à la 3e copie on centralise, ici dès la 2e).
//
// Désenrôlements assumés, chacun établi par mesure et non par lecture de
// commentaire :
//   - Q5 / Q13 sur seedSharedDBSchema : ces requêtes passent par SharedReader =
//     LegacySharedReader(player) dans newTestPlayerDB, donc lisent le schéma de
//     seedPlayerSchema. Preuve vivante : ce seed n'a ni season_id (lu par Q5) ni
//     map_version_id (lu par Q13), et la suite est verte.
//   - Q13 / baseSelect sur seedSharedDBForPoolTest : les tests pool n'exercent
//     que MatchHistoryRepo.LoadAll (Q5) — un seul appel de repo, ligne 48.
//
// Q12MatchScoreboard enrôlée le 2026-08-30 sur SIGNALEMENT du cliquet : elle lit
// kills_stddev / deaths_stddev, que le trio initial ne projetait pas.
var seedParityCarte = []seedParitySeed{
	{"seedPlayerSchema", "player_repos_test.go", []seedParityQuery{
		{"Q5SharedHistory", Q5SharedHistory, "r", "match_registry"},
		{"Q5SharedHistory", Q5SharedHistory, "p", "match_participants"},
		{"Q13MatchMeta", Q13MatchMeta, "r", "match_registry"},
		{"playerMatchesSharedBaseSelect", playerMatchesSharedBaseSelect, "r", "match_registry"},
		{"playerMatchesSharedBaseSelect", playerMatchesSharedBaseSelect, "p", "match_participants"},
		{"Q12MatchScoreboard", Q12MatchScoreboard, "p", "match_participants"},
	}},
	{"seedSharedDBSchema", "player_repos_test.go", []seedParityQuery{
		{"playerMatchesSharedBaseSelect", playerMatchesSharedBaseSelect, "r", "match_registry"},
		{"playerMatchesSharedBaseSelect", playerMatchesSharedBaseSelect, "p", "match_participants"},
		{"Q12MatchScoreboard", Q12MatchScoreboard, "p", "match_participants"},
	}},
	{"seedSharedDBForPoolTest", "pool_migration_test.go", []seedParityQuery{
		{"Q5SharedHistory", Q5SharedHistory, "r", "match_registry"},
		{"Q5SharedHistory", Q5SharedHistory, "p", "match_participants"},
	}},
}

// TestSeedSchemaColumnParity : le garde-rail. Pour chaque seed, pour chaque
// requête prod qui s'exécute réellement dessus, chaque colonne référencée doit
// exister dans le CREATE TABLE du seed — sinon UN échec nommant colonne,
// requête et seed, au lieu de 20 Binder Errors éparpillées.
func TestSeedSchemaColumnParity(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	dir := filepath.Dir(thisFile)

	srcCache := map[string]string{}
	for _, tc := range seedParityCarte {
		src, cached := srcCache[tc.file]
		if !cached {
			data, err := os.ReadFile(filepath.Join(dir, tc.file))
			if err != nil {
				t.Fatalf("lecture %s : %v", tc.file, err)
			}
			src = string(data)
			srcCache[tc.file] = src
		}
		for _, q := range tc.queries {
			refs := seedParityAliasRefs(q.sql, q.alias)
			if len(refs) < 5 {
				t.Errorf("%s : %d référence(s) %s.* extraite(s) — extracteur cassé ou alias renommé dans la requête ?",
					q.name, len(refs), q.alias)
				continue
			}
			cols := seedParityCreateColumns(t, src, tc.seedFunc, q.table)
			for _, ref := range seedParitySortedKeys(refs) {
				if !cols[ref] {
					t.Errorf("%s : colonne %q référencée par %s (alias %s) absente du CREATE TABLE de %s (%s) — "+
						"ajouter la colonne au seed (modèle : commit e484a68ae) ou désenrôler le couple ici avec justification datée",
						q.table, ref, q.name, q.alias, tc.seedFunc, tc.file)
				}
			}
		}
	}
}

// TestSeedSchemaColumnParity_Extracteurs verrouille les deux extracteurs sur
// des sources fabriquées : commentaires SQL, parenthèses imbriquées, contrainte
// de table, et deux fonctions créant la même table (périmètre par fonction).
func TestSeedSchemaColumnParity_Extracteurs(t *testing.T) {
	sqlText := "SELECT COALESCE(r.a_col, r.b_col) AS x, -- prose citant r.fantome\n" +
		"    epoch_ms(r.c_col AT TIME ZONE 'UTC') AS y, p.autre\n" +
		"FROM v_match_full r JOIN t2 p ON r.match_id = p.match_id WHERE p.xuid = ?"

	refs := seedParityAliasRefs(sqlText, "r")
	wantR := "a_col,b_col,c_col,match_id"
	if got := strings.Join(seedParitySortedKeys(refs), ","); got != wantR {
		t.Errorf("refs r.* : %q, attendu %q (le commentaire -- doit être exclu)", got, wantR)
	}
	prefs := seedParityAliasRefs(sqlText, "p")
	wantP := "autre,match_id,xuid"
	if got := strings.Join(seedParitySortedKeys(prefs), ","); got != wantP {
		t.Errorf("refs p.* : %q, attendu %q", got, wantP)
	}

	src := "func seedA(t *testing.T) {\n" +
		"\tddl := CREATE TABLE shared.demo_tbl (\n" +
		"\t\tmatch_id VARCHAR PRIMARY KEY, -- clef, citee en prose\n" +
		"\t\tscore DECIMAL(10,2) DEFAULT f(1, 2),\n" +
		"\t\trounds_total SMALLINT)\n" +
		"}\n" +
		"func seedB(t *testing.T) {\n" +
		"\tddl := CREATE TABLE demo_tbl (autre VARCHAR, PRIMARY KEY (autre))\n" +
		"}\n"
	colsA := seedParityCreateColumns(t, src, "seedA", "demo_tbl")
	wantA := "match_id,rounds_total,score"
	if got := strings.Join(seedParitySortedKeys(colsA), ","); got != wantA {
		t.Errorf("colonnes seedA : %q, attendu %q", got, wantA)
	}
	colsB := seedParityCreateColumns(t, src, "seedB", "demo_tbl")
	wantB := "autre"
	if got := strings.Join(seedParitySortedKeys(colsB), ","); got != wantB {
		t.Errorf("colonnes seedB : %q, attendu %q (périmètre par fonction + contrainte ignorée)", got, wantB)
	}
}
