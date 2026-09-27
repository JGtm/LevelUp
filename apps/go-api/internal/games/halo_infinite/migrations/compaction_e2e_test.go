//go:build cgo

// compaction_e2e_test.go — LA COMPACTION DES PASSES SUPERSÉDÉES sur le schéma partagé RÉEL
// (chaîne de migration complète, provider title-owned câblé : les 13 tables du registre, leurs
// vues, index et séquences tels que la production les porte).
//
// Ce qui est verrouillé ici, table par table : passes multiples, match à une seule passe, passe
// la plus récente dont le nom trie AVANT l'ancienne (l'arbitre est l'horloge, pas le nom), table
// vide (`match_weapon_hit_distance`), doublon À L'INTÉRIEUR de la passe retenue que la vue de
// `kill_positions` écarte (gardé : DC.2), joueurs d'usage qui suivent la passe de leur film,
// dernière ligne par clé (`match_bomb_stats`). Puis : sortie de chaque vue identique à l'octet
// (lecture ordonnée complète), DDL des tables, index, vues, contraintes et séquences identiques,
// séquence qui CONTINUE, idempotence, dry-run sans écriture.
package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

// tableKillPositions et passeDeDecodage : une seule occurrence de chaque littéral dans ce fichier
// (goconst compte les occurrences du paquet, tests compris, et signale les sites de production).
const (
	tableKillPositions = "kill_positions"
	passeDeDecodage    = "decode_pass"
)

// colonneDePasse : l'oracle du test, écrit indépendamment du registre. "" = dernière ligne par
// clé (match_id, xuid).
var colonneDePasse = map[string]string{
	"match_kill_events": passeDeDecodage, "match_lives": passeDeDecodage,
	"match_death_context": passeDeDecodage, "kill_openings": passeDeDecodage,
	tableKillPositions: passeDeDecodage, "match_weapon_shots": passeDeDecodage,
	"match_player_positions": "positions_pass", "match_usage_films": "summary_pass",
	"match_usage_players": "summary_pass", "match_pad_pickups_by_tier": passeDeDecodage,
	"match_flag_grabs_net": passeDeDecodage, "match_bomb_stats": "",
}

// passeSemee : une passe d'un match, écrite à l'instant `sec`, de `n` lignes.
type passeSemee struct {
	match, passe string
	sec, n       int
}

// Brutes 13, gardées 7 par table : m1 garde p1c (2), m2 sa passe unique (3), m3 garde p3a (2),
// écrite APRÈS p3b bien que son nom trie avant.
var passesSemees = []passeSemee{
	{"m1", "p1a", 1, 2}, {"m1", "p1b", 2, 2}, {"m1", "p1c", 3, 2},
	{"m2", "p2", 4, 3},
	{"m3", "p3b", 5, 2}, {"m3", "p3a", 6, 2},
}

func ouvrirSchemaPartageReel(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "shared.duckdb"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	migration.SetTitleStepsProvider(StepsFor)
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		t.Fatalf("RunForDB(TargetShared): %v", err)
	}
	return db
}

type colonne struct{ nom, typ string }

func colonnesDe(t *testing.T, db *sql.DB, table string) []colonne {
	t.Helper()
	rows, err := db.Query(`SELECT column_name, data_type FROM information_schema.columns
		WHERE table_schema = 'main' AND table_name = ? AND column_name <> 'id'
		ORDER BY ordinal_position`, table)
	if err != nil {
		t.Fatalf("colonnes %s: %v", table, err)
	}
	defer rows.Close() //nolint:errcheck
	var out []colonne
	for rows.Next() {
		var c colonne
		if err := rows.Scan(&c.nom, &c.typ); err != nil {
			t.Fatal(err)
		}
		out = append(out, c)
	}
	return out
}

// valeur : une valeur valide pour la colonne, distincte par ligne `i`.
func valeur(t *testing.T, table string, c colonne, p passeSemee, i int) any {
	t.Helper()
	switch {
	case c.nom == "match_id":
		return p.match
	case c.nom == colonneDePasse[table] && c.nom != "":
		return p.passe
	case c.nom == "written_at":
		return fmt.Sprintf("2026-01-01 00:00:%02d", p.sec)
	case strings.HasSuffix(c.nom, "xuid"):
		return fmt.Sprintf("x%d", i)
	case c.nom == "time_ms":
		return i * 1000
	}
	switch c.typ {
	case "VARCHAR":
		return fmt.Sprintf("v%s%d", p.passe, i)
	case "BOOLEAN":
		return i%2 == 0
	case "BIGINT", "INTEGER", "SMALLINT", "UTINYINT", "UINTEGER", "UBIGINT":
		return i + 1
	case "DOUBLE", "FLOAT":
		return float64(i) + 0.5
	case "TIMESTAMP":
		return "2026-01-01 00:00:00"
	}
	t.Fatalf("%s.%s : type %s non prévu par le semeur", table, c.nom, c.typ)
	return nil
}

func semerPasse(t *testing.T, db *sql.DB, table string, cols []colonne, p passeSemee, depuis int) {
	t.Helper()
	noms := make([]string, len(cols))
	marques := make([]string, len(cols))
	for k, c := range cols {
		noms[k], marques[k] = c.nom, "?"
	}
	q := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s)`, table,
		strings.Join(noms, ", "), strings.Join(marques, ", "))
	for i := depuis; i < depuis+p.n; i++ {
		args := make([]any, len(cols))
		for k, c := range cols {
			args[k] = valeur(t, table, c, p, i)
		}
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatalf("insert %s: %v", table, err)
		}
	}
}

// semerTout : les passes de chaque table du registre sauf `match_weapon_hit_distance` (laissée
// vide), plus un doublon (killer_xuid, time_ms) dans la passe retenue de m1 pour kill_positions.
func semerTout(t *testing.T, db *sql.DB) {
	t.Helper()
	for table := range colonneDePasse {
		cols := colonnesDe(t, db, table)
		for _, p := range passesSemees {
			if table == "match_usage_films" {
				p.n = 1 // un film = UNE ligne par passe (la vue des joueurs s y joint)
			}
			semerPasse(t, db, table, cols, p, 0)
		}
	}
	semerPasse(t, db, tableKillPositions, colonnesDe(t, db, tableKillPositions),
		passeSemee{"m1", "p1c", 3, 1}, 0)
}

func lireOrdonne(t *testing.T, db *sql.DB, requete string) []string {
	t.Helper()
	rows, err := db.Query(requete)
	if err != nil {
		t.Fatalf("%s: %v", requete, err)
	}
	defer rows.Close() //nolint:errcheck
	cols, _ := rows.Columns()
	var out []string
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			t.Fatal(err)
		}
		out = append(out, fmt.Sprint(vals...))
	}
	return out
}

// schemaComplet : tout ce que la compaction doit laisser identique, hors données.
func schemaComplet(t *testing.T, db *sql.DB) map[string][]string {
	t.Helper()
	return map[string][]string{
		"tables":      lireOrdonne(t, db, `SELECT table_name, sql FROM duckdb_tables() ORDER BY 1`),
		"index":       lireOrdonne(t, db, `SELECT index_name, sql FROM duckdb_indexes() ORDER BY 1`),
		"vues":        lireOrdonne(t, db, `SELECT view_name, sql FROM duckdb_views() WHERE NOT internal ORDER BY 1`),
		"contraintes": lireOrdonne(t, db, `SELECT table_name, constraint_type, constraint_text FROM duckdb_constraints() ORDER BY ALL`),
		"sequences":   lireOrdonne(t, db, `SELECT sequence_name, start_value, increment_by, min_value, max_value, cycle, last_value FROM duckdb_sequences() ORDER BY 1`),
	}
}

func lireVues(t *testing.T, db *sql.DB) map[string][]string {
	t.Helper()
	out := map[string][]string{}
	for _, table := range migration.CompactableTables() {
		out[table] = lireOrdonne(t, db, `SELECT * FROM `+table+`_latest ORDER BY id`)
	}
	return out
}

func compte(t *testing.T, db *sql.DB, table string) int64 {
	t.Helper()
	var n int64
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// TestCompaction_SchemaReel_BoutABout : dry-run, compaction, preuves, idempotence.
func TestCompaction_SchemaReel_BoutABout(t *testing.T) {
	db := ouvrirSchemaPartageReel(t)
	semerTout(t, db)
	ctx := context.Background()
	schemaAvant, vuesAvant := schemaComplet(t, db), lireVues(t, db)
	var maxIDAvant int64
	if err := db.QueryRow(`SELECT MAX(id) FROM match_kill_events`).Scan(&maxIDAvant); err != nil {
		t.Fatal(err)
	}

	sec, err := migration.CompactSupersededPasses(ctx, db, true)
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if !reflect.DeepEqual(schemaComplet(t, db), schemaAvant) || compte(t, db, "match_kill_events") != 13 {
		t.Fatal("le dry-run a écrit")
	}
	verifierRapport(t, sec, migration.CompactionACompacter)

	rs, err := migration.CompactSupersededPasses(ctx, db, false)
	if err != nil {
		t.Fatalf("compaction: %v", err)
	}
	verifierRapport(t, rs, migration.CompactionCompactee)
	if got := schemaComplet(t, db); !reflect.DeepEqual(got, schemaAvant) {
		t.Fatalf("schéma changé :\navant %v\naprès %v", schemaAvant, got)
	}
	if got := lireVues(t, db); !reflect.DeepEqual(got, vuesAvant) {
		t.Fatalf("sortie des vues changée :\navant %v\naprès %v", vuesAvant, got)
	}

	// La séquence CONTINUE : le prochain id suit le plus grand jamais tiré, aucun n'est réutilisé.
	semerPasse(t, db, "match_kill_events", colonnesDe(t, db, "match_kill_events"),
		passeSemee{"m9", "p9", 9, 1}, 0)
	var id int64
	if err := db.QueryRow(`SELECT MAX(id) FROM match_kill_events`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if id != maxIDAvant+1 {
		t.Fatalf("id après compaction = %d, attendu %d (la séquence doit continuer)", id, maxIDAvant+1)
	}

	encore, err := migration.CompactSupersededPasses(ctx, db, false)
	if err != nil {
		t.Fatalf("seconde passe: %v", err)
	}
	for _, r := range encore {
		if r.Status != migration.CompactionDejaCompacte {
			t.Fatalf("seconde passe : %s %q, attendu deja-compacte (idempotence)", r.Table, r.Status)
		}
	}
}

// verifierRapport : chaque table semée a 13 brutes (14 pour kill_positions, 6 pour les films) et
// garde 7 (8, 3) ; la table vide est déjà compacte.
func verifierRapport(t *testing.T, rs []migration.CompactionTable, statut string) {
	t.Helper()
	if len(rs) != len(colonneDePasse)+1 {
		t.Fatalf("%d tables au rapport, attendu %d", len(rs), len(colonneDePasse)+1)
	}
	for _, r := range rs {
		if r.Table == "match_weapon_hit_distance" {
			if r.Status != migration.CompactionDejaCompacte || r.Raw != 0 {
				t.Fatalf("table vide : %+v", r)
			}
			continue
		}
		raw, kept, latest := int64(13), int64(7), int64(7)
		switch r.Table {
		case tableKillPositions:
			raw, kept = 14, 8
		case "match_usage_films":
			raw, kept, latest = 6, 3, 3
		}
		after := kept // dry-run : ce qui resterait ; compaction : ce qui reste
		if r.Status != statut || r.Raw != raw || r.Kept != kept || r.Latest != latest || r.After != after {
			t.Fatalf("%s : %+v, attendu %s brutes=%d gardées=%d latest=%d après=%d",
				r.Table, r, statut, raw, kept, latest, after)
		}
	}
}
