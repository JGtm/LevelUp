//go:build cgo

// Package ops — seed_demo_shared_tables_schema_test.go : les listes d'extraction de la démo
// sont relues CONTRE LE SCHÉMA MIGRÉ, pas contre elles-mêmes.
//
// Deux garde-rails, un par façon de rendre la démo fausse sans qu'aucun test existant ne le voie :
//  1. une table de la base partagée portant `match_id` que ni l'extraction ni la liste des
//     exclusions justifiées ne nomme : la démo l'ignore, et toute page qui la lit s'affiche
//     vide ou « non mesurée » (la panne de la démo d'avant le 2026-10-19 : aucune table dérivée
//     du film) ;
//  2. une colonne `*xuid*` / `*gamertag*` d'une table extraite que sa déclaration d'identité
//     ne cite pas : la copie la laisserait telle quelle, et seul le contrôle des valeurs de fin
//     de seed (échec du seed, jamais un fichier publié) la rattraperait.
package ops

import (
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"

	_ "github.com/duckdb/duckdb-go/v2"
)

// sharedTablesNotExtracted : les tables de la base partagée portant `match_id` que la démo
// n'extrait PAS, chacune avec sa raison. Une table neuve ne s'ajoute ici qu'avec le lecteur
// examiné et la raison écrite ; sinon elle rejoint filmDerivedTables / sharedTablesWhere.
var sharedTablesNotExtracted = map[string]string{
	"match_weapon_shots": "écrite par le collecteur de kills et le rejeu de films, aucun lecteur " +
		"applicatif (recherche du nom et de sa vue dans internal/ et cmd/ au 2026-10-19 : écrivains, " +
		"migrations et commentaires seuls)",
	"match_weapon_hit_distance": "écrite à la cuisson du film, aucun lecteur applicatif (même " +
		"recherche, 2026-10-19)",
}

// colonnesNommeesSansIdentite : les colonnes dont le nom contient `xuid` ou `gamertag` sans porter
// d'identité, chacune avec sa raison et sa date. Une entrée ne s'ajoute qu'après lecture du DDL.
var colonnesNommeesSansIdentite = map[string]string{
	"match_kill_heads.n_killer_xuid": "2026-10-21 (ADR 0039) : compte INTEGER des morts dont le " +
		"tueur porte un xuid, aucune identité (steps_shared_kill_events_summaries.go)",
}

// migratedTableColumns : les colonnes de chaque TABLE (vues exclues) d'une base fraîche migrée
// vers `target`.
func migratedTableColumns(t *testing.T, target migration.TargetDB) map[string][]string {
	t.Helper()
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "schema.duckdb"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if err := migration.RunForDB(db, target); err != nil {
		t.Fatalf("migrations %s: %v", target, err)
	}
	rows, err := db.Query(`SELECT c.table_name, c.column_name FROM duckdb_columns() c
		JOIN duckdb_tables() t ON t.table_name = c.table_name AND t.schema_name = c.schema_name
		ORDER BY c.table_name, c.column_index`)
	if err != nil {
		t.Fatalf("colonnes du schéma %s: %v", target, err)
	}
	defer rows.Close()
	out := map[string][]string{}
	for rows.Next() {
		var table, col string
		if err := rows.Scan(&table, &col); err != nil {
			t.Fatal(err)
		}
		out[table] = append(out[table], col)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// declaredIdentityColumns : toutes les colonnes qu'une table extraite déclare à l'anonymisation.
func declaredIdentityColumns(tb extractTable) map[string]bool {
	out := map[string]bool{}
	for _, pair := range tb.identity {
		for _, c := range pair {
			if c != "" {
				out[c] = true
			}
		}
	}
	for _, list := range [][]string{tb.gamertagOnly, tb.xuidLists, tb.jsonIdentity} {
		for _, c := range list {
			out[c] = true
		}
	}
	return out
}

func carriesIdentityName(col string) bool {
	c := strings.ToLower(col)
	return strings.Contains(c, "xuid") || strings.Contains(c, "gamertag")
}

// TestSharedTablesCoverEveryMatchKeyedTable : garde-rail (1).
func TestSharedTablesCoverEveryMatchKeyedTable(t *testing.T) {
	schema := migratedTableColumns(t, migration.TargetShared)
	extracted := map[string]bool{}
	for _, tb := range sharedTablesFor(title.DefaultSlug) {
		extracted[tb.name] = true
	}
	var missing []string
	for table, cols := range schema {
		keyed := false
		for _, c := range cols {
			keyed = keyed || c == "match_id"
		}
		if !keyed || extracted[table] {
			continue
		}
		if _, why := sharedTablesNotExtracted[table]; !why {
			missing = append(missing, table)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("tables de la base partagée portant match_id, ni extraites par la démo ni exclues avec une "+
			"raison : %v — les pages qui les lisent seraient vides dans la démo", missing)
	}
	for table, why := range sharedTablesNotExtracted {
		switch {
		case why == "":
			t.Errorf("exclusion %s sans raison", table)
		case extracted[table]:
			t.Errorf("%s est à la fois extraite et exclue : retirer l'exclusion", table)
		case schema[table] == nil:
			t.Errorf("exclusion %s : la table n'existe plus dans le schéma migré, retirer l'exclusion", table)
		}
	}
}

// TestSharedTablesDeclareEveryIdentityColumn : garde-rail (2), sur les tables shared ET player.
func TestSharedTablesDeclareEveryIdentityColumn(t *testing.T) {
	vues := map[string]bool{}
	for _, tc := range []struct {
		target migration.TargetDB
		tables []extractTable
	}{
		{migration.TargetShared, sharedTablesFor(title.DefaultSlug)},
		{migration.TargetPlayer, playerTablesWhere},
	} {
		schema := migratedTableColumns(t, tc.target)
		for _, tb := range tc.tables {
			declared := declaredIdentityColumns(tb)
			for _, col := range schema[tb.name] {
				if _, compte := colonnesNommeesSansIdentite[tb.name+"."+col]; compte {
					vues[tb.name+"."+col] = true
					continue
				}
				if carriesIdentityName(col) && !declared[col] {
					t.Errorf("%s.%s (%s) : colonne d'identité extraite sans déclaration d'anonymisation — "+
						"l'identité réelle traverserait dans la démo", tb.name, col, tc.target)
				}
			}
		}
	}
	for col := range colonnesNommeesSansIdentite {
		if !vues[col] {
			t.Errorf("exemption %s : colonne absente des tables extraites, retirer l'exemption", col)
		}
	}
}

// TestIdentityColumnNamesStayCentralised : `killer_xuid`, `victim_xuid` et `feed_killer_xuid` portent chacune
// plusieurs déclarations d'identité (tables du socle et dérivées du film) ; leur littéral vit dans
// colKillerXUID / colVictimXUID / colFeedKillerXUID, jamais recopié dans une liste d'extraction (règle n°6).
func TestIdentityColumnNamesStayCentralised(t *testing.T) {
	for _, f := range []string{"seed_demo.go", "seed_demo_shared_tables.go"} {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("lecture %s: %v", f, err)
		}
		for i, line := range strings.Split(string(raw), "\n") {
			if strings.Contains(line, "colKillerXUID =") || strings.Contains(line, "colVictimXUID =") ||
				strings.Contains(line, "colFeedKillerXUID =") {
				continue
			}
			for _, lit := range []string{`"killer_xuid"`, `"victim_xuid"`, `"feed_killer_xuid"`} {
				if strings.Contains(line, lit) {
					t.Errorf("%s:%d recopie le littéral %s : utiliser colKillerXUID / colVictimXUID / colFeedKillerXUID", f, i+1, lit)
				}
			}
		}
	}
}
