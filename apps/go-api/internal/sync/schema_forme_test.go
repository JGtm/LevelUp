//go:build integration

package sync_test

// schema_forme_test.go — LA FORME D'UNE BASE JOUEUR NE DÉPEND PAS DE SON ORDRE DE CRÉATION.
//
// Une player DB naît chez son premier ouvreur : la synchronisation (OpenPlayerDB, qui joue
// EnsurePlayerSchema) ou la chaîne de migrations (la première requête d'API). Le second ne refait
// pas les tables existantes, si bien que la forme d'une colonne (type, valeur par défaut, NOT NULL)
// reste celle de la naissance. schema_authority_test.go ne compare que des NOMS ; ce test compare,
// dans les deux sens, la forme « synchronisation puis migrations » à la forme « migrations seules »
// sur les colonnes, les contraintes de table, les vues, les index et les séquences.
//
// La forme de référence est celle que portent les bases existantes. Les écarts qui restent sont
// listés dans formeEcartsResiduels, un par un, avec leur date et leur raison ; la liste ne fait
// que rétrécir : une entrée qui ne diffère plus fait échouer le test.

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/sync"

	_ "github.com/duckdb/duckdb-go/v2"
)

// formeEcartsResiduels : clés de forme (« colonne table.colonne ») qui diffèrent encore entre les
// deux ordres de création. Chaque raison commence par une date AAAA-MM-JJ.
var formeEcartsResiduels = map[string]string{
	"colonne player_match_enrichment.performance_score":        "2026-11-18 : FLOAT à la synchronisation et dans les bases existantes, DOUBLE dans la définition de create_baseline_player_v1 que reprend la chaîne ; aligner la chaîne sur FLOAT demande de modifier cette étape déjà jouée en production (et son instantané de référence), ou de convertir le type d'une table qu'une vue lit.",
	"colonne player_match_enrichment_latest.performance_score": "2026-11-18 : même cause que player_match_enrichment.performance_score (la vue en reprend le type).",
	"colonne player_match_enrichment.is_with_friends":          "2026-11-18 : défaut FALSE à la synchronisation, absent des bases existantes et de la chaîne (la reconstruction append-only perd les défauts) ; TestPlayerMatchEnrichment_DefaultFalse garde ce défaut contre la régression « Solo qui revient » (un NULL) ; le retirer contredit cette garde, l'ajouter à la chaîne demande de modifier une étape déjà jouée en production.",
	"colonne player_match_enrichment.is_excluded":              "2026-11-18 : même cause que is_with_friends (défaut FALSE à la synchronisation, absent des bases existantes et de la chaîne).",
	"colonne sync_meta.updated_at":                             "2026-11-18 : défaut UTC à la synchronisation et dans les bases existantes, absent de la définition de create_baseline_player_v1 que reprend la chaîne ; l'aligner demande de modifier cette étape déjà jouée en production.",
}

// formePlayerDB décrit une player DB par ce que les deux chemins de création peuvent faire
// diverger : pour chaque colonne le type, la valeur par défaut et le NOT NULL (clé
// « colonne table.colonne »), pour chaque table ses contraintes, pour chaque vue, index et
// séquence sa présence. La base est lue sur son fichier, fermé puis rouvert : DuckDB réécrit alors
// le texte des défauts comme il le fait à tout rechargement.
func formePlayerDB(t *testing.T, path string) map[string]string {
	t.Helper()
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("réouverture de %s: %v", filepath.Base(path), err)
	}
	defer db.Close()

	requetes := []string{
		`SELECT 'colonne ' || table_name || '.' || column_name,
			data_type || ' | défaut ' || coalesce(column_default, '-') || ' | ' ||
			CASE WHEN is_nullable THEN 'nullable' ELSE 'NOT NULL' END
		 FROM duckdb_columns() WHERE schema_name = 'main' AND NOT internal`,
		`SELECT 'contrainte ' || table_name || ' ' || constraint_type || ' ' ||
			coalesce(constraint_column_names::VARCHAR, ''), ''
		 FROM duckdb_constraints() WHERE schema_name = 'main' AND constraint_type <> 'NOT NULL'`,
		`SELECT 'vue ' || view_name, '' FROM duckdb_views() WHERE schema_name = 'main' AND NOT internal`,
		`SELECT 'index ' || index_name || ' sur ' || table_name, '' FROM duckdb_indexes() WHERE schema_name = 'main'`,
		`SELECT 'sequence ' || sequence_name, '' FROM duckdb_sequences() WHERE schema_name = 'main'`,
	}
	forme := map[string]string{}
	for _, q := range requetes {
		rows, err := db.Query(q)
		if err != nil {
			t.Fatalf("lecture de la forme (%s): %v", q, err)
		}
		for rows.Next() {
			var cle, detail string
			if err := rows.Scan(&cle, &detail); err != nil {
				rows.Close()
				t.Fatalf("lecture de la forme: scan: %v", err)
			}
			forme[cle] = detail
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			t.Fatalf("lecture de la forme: %v", err)
		}
		rows.Close()
	}
	if len(forme) == 0 {
		t.Fatal("forme vide : la lecture du catalogue est cassée, le test se mentirait")
	}
	return forme
}

// creerPlayerDB construit un fichier player DB dans un dossier propre au test : `etapes` reçoit la
// base ouverte et la remplit. Le fichier est rendu fermé.
func creerPlayerDB(t *testing.T, nom string, etapes func(db *sql.DB)) string {
	t.Helper()
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	path := filepath.Join(t.TempDir(), nom+".duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open %s: %v", nom, err)
	}
	etapes(db)
	if err := db.Close(); err != nil {
		t.Fatalf("close %s: %v", nom, err)
	}
	return path
}

func exigerSucces(t *testing.T, quoi string, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", quoi, err)
	}
}

// playerDBSynchroPuisMigrations : naissance par la synchronisation (EnsurePlayerSchema, ce
// qu'OpenPlayerDB joue), puis la chaîne de migrations.
func playerDBSynchroPuisMigrations(t *testing.T) string {
	return creerPlayerDB(t, "synchro_puis_migrations", func(db *sql.DB) {
		exigerSucces(t, "EnsurePlayerSchema sur base vierge", sync.EnsurePlayerSchema(context.Background(), db))
		exigerSucces(t, "RunForDB(player) après la synchronisation", migration.RunForDB(db, migration.TargetPlayer))
	})
}

// playerDBMigrationsSeules : naissance par la chaîne de migrations, ce que fait la première
// ouverture d'une requête d'API.
func playerDBMigrationsSeules(t *testing.T) string {
	return creerPlayerDB(t, "migrations_seules", func(db *sql.DB) {
		exigerSucces(t, "RunForDB(player)", migration.RunForDB(db, migration.TargetPlayer))
	})
}

// playerDBMigrationsPuisSynchro : naissance par les migrations, puis l'ouverture de la
// synchronisation.
func playerDBMigrationsPuisSynchro(t *testing.T) string {
	return creerPlayerDB(t, "migrations_puis_synchro", func(db *sql.DB) {
		exigerSucces(t, "RunForDB(player)", migration.RunForDB(db, migration.TargetPlayer))
		exigerSucces(t, "EnsurePlayerSchema sur base migrée", sync.EnsurePlayerSchema(context.Background(), db))
	})
}

// ecartsDeForme rend, par clé, la description de chaque écart entre deux formes, dans les deux
// sens : une clé présente d'un seul côté, ou présente des deux avec un détail différent.
func ecartsDeForme(gauche, droite map[string]string) map[string]string {
	cles := map[string]bool{}
	for k := range gauche {
		cles[k] = true
	}
	for k := range droite {
		cles[k] = true
	}
	ecarts := map[string]string{}
	for k := range cles {
		g, okG := gauche[k]
		d, okD := droite[k]
		switch {
		case okG && !okD:
			ecarts[k] = fmt.Sprintf("%s : à gauche seulement (%s)", k, g)
		case !okG && okD:
			ecarts[k] = fmt.Sprintf("%s : à droite seulement (%s)", k, d)
		case g != d:
			ecarts[k] = fmt.Sprintf("%s : gauche [%s] contre droite [%s]", k, g, d)
		}
	}
	return ecarts
}

func lignesTriees(ecarts map[string]string) []string {
	lignes := make([]string, 0, len(ecarts))
	for _, l := range ecarts {
		lignes = append(lignes, l)
	}
	sort.Strings(lignes)
	return lignes
}

// TestPlayerSchemaForme_NaissanceParLaSynchroOuParLesMigrations : après la chaîne de migrations,
// une base née de la synchronisation et une base née des migrations ont la même forme, aux écarts
// résiduels nommés près ; une base migrée puis ouverte par la synchronisation ne change pas.
func TestPlayerSchemaForme_NaissanceParLaSynchroOuParLesMigrations(t *testing.T) {
	synchro := formePlayerDB(t, playerDBSynchroPuisMigrations(t))
	migrations := formePlayerDB(t, playerDBMigrationsSeules(t))

	ecarts := ecartsDeForme(synchro, migrations)
	var inattendus []string
	for k, l := range ecarts {
		if _, ok := formeEcartsResiduels[k]; !ok {
			inattendus = append(inattendus, l)
		}
	}
	sort.Strings(inattendus)
	if len(inattendus) > 0 {
		t.Errorf("la forme d'une player DB dépend de son ordre de création (%d écart(s) hors liste "+
			"résiduelle ; à gauche la synchronisation puis les migrations, à droite les migrations seules) :\n  - %s",
			len(inattendus), strings.Join(inattendus, "\n  - "))
	}
	for k := range formeEcartsResiduels {
		if _, ok := ecarts[k]; !ok {
			t.Errorf("%s ne diffère plus : retirer l'entrée de formeEcartsResiduels", k)
		}
	}

	if reste := ecartsDeForme(migrations, formePlayerDB(t, playerDBMigrationsPuisSynchro(t))); len(reste) > 0 {
		t.Errorf("EnsurePlayerSchema a changé la forme d'une base migrée :\n  - %s",
			strings.Join(lignesTriees(reste), "\n  - "))
	}
}

// TestPlayerSchemaForme_DatesDesResiduels : la raison de chaque écart résiduel commence par une date.
func TestPlayerSchemaForme_DatesDesResiduels(t *testing.T) {
	forme := regexp.MustCompile(`^\d{4}-\d{2}-\d{2} : `)
	for k, raison := range formeEcartsResiduels {
		if !forme.MatchString(raison) {
			t.Errorf("%s : la raison ne commence pas par « AAAA-MM-JJ : » : %q", k, raison)
		}
	}
}

// TestPlayerSchemaForme_BiteProof : la comparaison voit un type, un défaut et un NOT NULL qui
// changent, et le dit dans les deux sens.
func TestPlayerSchemaForme_BiteProof(t *testing.T) {
	reference := playerDBMigrationsSeules(t)
	muee := creerPlayerDB(t, "muee", func(db *sql.DB) {
		exigerSucces(t, "RunForDB(player)", migration.RunForDB(db, migration.TargetPlayer))
		for _, stmt := range []string{
			`ALTER TABLE match_skill_rank ALTER sub_tier SET DEFAULT 0`,
			`ALTER TABLE sync_meta ALTER value SET NOT NULL`,
			`ALTER TABLE career_progression ALTER rank SET DATA TYPE DOUBLE`,
		} {
			if _, err := db.Exec(stmt); err != nil {
				t.Fatalf("mutation sonde (%s): %v", stmt, err)
			}
		}
	})

	ecarts := ecartsDeForme(formePlayerDB(t, reference), formePlayerDB(t, muee))
	for _, cle := range []string{
		"colonne match_skill_rank.sub_tier",
		"colonne sync_meta.value",
		"colonne career_progression.rank",
	} {
		if _, ok := ecarts[cle]; !ok {
			t.Errorf("morsure absente : %s n'apparaît pas dans les écarts %v", cle, lignesTriees(ecarts))
		}
	}
	inverse := ecartsDeForme(formePlayerDB(t, muee), formePlayerDB(t, reference))
	if len(inverse) != len(ecarts) {
		t.Errorf("la comparaison n'est pas symétrique : %d écart(s) dans un sens, %d dans l'autre", len(ecarts), len(inverse))
	}
}
