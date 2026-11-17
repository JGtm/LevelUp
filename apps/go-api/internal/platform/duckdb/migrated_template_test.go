// Package duckdb — migrated_template_test.go : bases de test DÉJÀ MIGRÉES, copiées au lieu d'être
// migrées à neuf dans chaque test.
//
// Une migration complète coûte de 0,6 s (shared en mémoire) à 5 s (metadata) ; un test qui
// n'a besoin que du schéma livré la payait à chaque fois. Ce fichier migre UNE fois chaque
// cible (shared, player, metadata, shared_social) dans un fichier gabarit, puis rend à chaque
// test une copie :
//
//   - MigratedFileCopy / MigratedTempFile : copie du fichier gabarit (base sur disque) ;
//   - FillMigratedMemory / OpenMigratedMemory : base en mémoire remplie depuis le gabarit
//     (ATTACH en lecture seule, COPY FROM DATABASE, DETACH).
//
// Contrat :
//   - le gabarit est construit sous sync.Once par cible ; son erreur est mémorisée et rendue à
//     chaque appelant, qui échoue son test ;
//   - il est fermé et sans fichier .wal avant toute copie (vérifié à la construction) ;
//   - le dossier des gabarits est créé à la première construction (un sous-processus de test qui
//     n'en construit aucun ne laisse rien) et retiré par TestMain (removeMigratedTemplates) ;
//   - seules ces quatre cibles ont un gabarit : une autre cible (shared_pve) échoue le test.
//
// Les noms exportés servent aussi le paquet de test externe duckdb_test. Un test dont l'objet est
// la migration elle-même (ordre des étapes, reprise après incident, topologie de déploiement)
// migre à neuf : la liste de ces fichiers est tenue par no_fresh_migration_in_tests_test.go.
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

// gabaritMigre est le fichier migré d'une cible, construit au plus une fois.
type gabaritMigre struct {
	once sync.Once
	path string
	err  error
}

var (
	gabaritsRacineOnce sync.Once
	gabaritsRacine     string
	gabaritsRacineErr  error

	// gabarits est figée à l'initialisation : les lectures concurrentes sont sûres.
	gabarits = map[migration.TargetDB]*gabaritMigre{
		migration.TargetShared:       {},
		migration.TargetPlayer:       {},
		migration.TargetMetadata:     {},
		migration.TargetSharedSocial: {},
	}
)

// racineGabarits crée (une fois) le dossier qui porte les fichiers gabarits.
func racineGabarits() (string, error) {
	gabaritsRacineOnce.Do(func() {
		gabaritsRacine, gabaritsRacineErr = os.MkdirTemp("", "levelup-gabarits-*")
	})
	return gabaritsRacine, gabaritsRacineErr
}

// removeMigratedTemplates retire le dossier des gabarits. Appelée par TestMain APRÈS m.Run :
// os.Exit saute les defer.
func removeMigratedTemplates() {
	if gabaritsRacine != "" {
		_ = os.RemoveAll(gabaritsRacine)
	}
}

// gabaritPath rend le chemin du gabarit de la cible, construit au premier appel.
func gabaritPath(target migration.TargetDB) (string, error) {
	g, ok := gabarits[target]
	if !ok {
		return "", fmt.Errorf("pas de gabarit migré pour la cible %q", target)
	}
	g.once.Do(func() { g.path, g.err = construireGabarit(target) })
	return g.path, g.err
}

// construireGabarit migre une base neuve de la cible puis la ferme ; l'absence de .wal prouve
// que tout est dans le fichier avant la première copie.
func construireGabarit(target migration.TargetDB) (string, error) {
	racine, err := racineGabarits()
	if err != nil {
		return "", fmt.Errorf("dossier des gabarits: %w", err)
	}
	path := filepath.Join(racine, string(target)+".duckdb")
	raw, err := sql.Open("duckdb", path)
	if err != nil {
		return "", fmt.Errorf("ouverture du gabarit %s: %w", target, err)
	}
	if err := migration.RunForDB(raw, target); err != nil {
		_ = raw.Close()
		return "", fmt.Errorf("migration du gabarit %s: %w", target, err)
	}
	if _, err := raw.Exec("CHECKPOINT"); err != nil {
		_ = raw.Close()
		return "", fmt.Errorf("CHECKPOINT du gabarit %s: %w", target, err)
	}
	if err := raw.Close(); err != nil {
		return "", fmt.Errorf("fermeture du gabarit %s: %w", target, err)
	}
	if _, err := os.Stat(path + ".wal"); err == nil {
		return "", fmt.Errorf("le gabarit %s garde un .wal après fermeture", target)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("lecture du .wal du gabarit %s: %w", target, err)
	}
	return path, nil
}

// MigratedFileCopy copie le gabarit migré de `target` vers `dest` (dossier parent créé). La
// copie est une base fermée, que l'appelant ouvre comme il l'entend.
func MigratedFileCopy(t testing.TB, target migration.TargetDB, dest string) {
	t.Helper()
	src, err := gabaritPath(target)
	if err != nil {
		t.Fatalf("gabarit migré %s: %v", target, err)
	}
	if err := copierFichier(src, dest); err != nil {
		t.Fatalf("copie du gabarit %s vers %s: %v", target, dest, err)
	}
}

// MigratedTempFile copie le gabarit de `target` sous `name` dans un dossier propre au test et
// rend le chemin de la copie.
func MigratedTempFile(t testing.TB, target migration.TargetDB, name string) string {
	t.Helper()
	dest := filepath.Join(t.TempDir(), name)
	MigratedFileCopy(t, target, dest)
	return dest
}

func copierFichier(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// FillMigratedMemory remplit la base EN MÉMOIRE `db` (vide) avec le schéma et les données du
// gabarit de `target`. Les trois ordres s'exécutent sur une même connexion ; le gabarit n'est
// attaché qu'en lecture seule, donc plusieurs tests parallèles le lisent sans se gêner.
func FillMigratedMemory(t testing.TB, db *sql.DB, target migration.TargetDB) {
	t.Helper()
	src, err := gabaritPath(target)
	if err != nil {
		t.Fatalf("gabarit migré %s: %v", target, err)
	}
	if err := remplirDepuisGabarit(db, src); err != nil {
		t.Fatalf("remplissage mémoire depuis le gabarit %s: %v", target, err)
	}
}

// OpenMigratedMemory ouvre une base en mémoire déjà migrée pour `target`, fermée en fin de test.
func OpenMigratedMemory(t testing.TB, target migration.TargetDB) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open mem: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	FillMigratedMemory(t, db, target)
	return db
}

func remplirDepuisGabarit(db *sql.DB, gabarit string) error {
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	var cible string
	if err := conn.QueryRowContext(ctx, "SELECT current_database()").Scan(&cible); err != nil {
		return fmt.Errorf("base courante: %w", err)
	}
	chemin := strings.ReplaceAll(filepath.ToSlash(gabarit), "'", "''")
	if _, err := conn.ExecContext(ctx, "ATTACH '"+chemin+"' AS gabarit_migre (READ_ONLY)"); err != nil {
		return fmt.Errorf("ATTACH: %w", err)
	}
	_, errCopie := conn.ExecContext(ctx, `COPY FROM DATABASE gabarit_migre TO "`+cible+`"`)
	if _, err := conn.ExecContext(ctx, "DETACH gabarit_migre"); err != nil && errCopie == nil {
		return fmt.Errorf("DETACH: %w", err)
	}
	if errCopie != nil {
		return fmt.Errorf("COPY FROM DATABASE: %w", errCopie)
	}
	return nil
}

// TestMigratedTemplate_CopiesIndependantes : deux copies du même gabarit sont indépendantes (une
// écriture dans l'une n'apparaît ni dans l'autre ni dans le gabarit), pour le fichier comme pour
// la mémoire, et la cible sans gabarit est refusée avec une erreur.
func TestMigratedTemplate_CopiesIndependantes(t *testing.T) {
	const target = migration.TargetMetadata
	a, err := sql.Open("duckdb", MigratedTempFile(t, target, "a.duckdb"))
	if err != nil {
		t.Fatalf("open a: %v", err)
	}
	t.Cleanup(func() { _ = a.Close() })
	b, err := sql.Open("duckdb", MigratedTempFile(t, target, "b.duckdb"))
	if err != nil {
		t.Fatalf("open b: %v", err)
	}
	t.Cleanup(func() { _ = b.Close() })
	m1, m2 := OpenMigratedMemory(t, target), OpenMigratedMemory(t, target)

	const compte = `SELECT count(*) FROM schema_migrations WHERE name = 'cle_de_test'`
	if _, err := a.Exec(`INSERT INTO schema_migrations (name) VALUES ('cle_de_test')`); err != nil {
		t.Fatalf("écriture dans a: %v", err)
	}
	if _, err := m1.Exec(`INSERT INTO schema_migrations (name) VALUES ('cle_de_test')`); err != nil {
		t.Fatalf("écriture dans m1: %v", err)
	}
	for nom, db := range map[string]*sql.DB{"b": b, "m2": m2} {
		var n int
		if err := db.QueryRow(compte).Scan(&n); err != nil {
			t.Fatalf("lecture de %s: %v", nom, err)
		}
		if n != 0 {
			t.Errorf("%s voit l'écriture d'une autre copie (%d ligne)", nom, n)
		}
	}
	fraiche, err := sql.Open("duckdb", MigratedTempFile(t, target, "c.duckdb"))
	if err != nil {
		t.Fatalf("open c: %v", err)
	}
	t.Cleanup(func() { _ = fraiche.Close() })
	var n int
	if err := fraiche.QueryRow(compte).Scan(&n); err != nil || n != 0 {
		t.Errorf("le gabarit a gardé l'écriture d'une copie : n=%d err=%v", n, err)
	}

	if _, err := gabaritPath(migration.TargetSharedPvE); err == nil {
		t.Error("la cible shared_pve n'a pas de gabarit : une erreur était attendue")
	}
}

// --- Fidélité du gabarit à une migration neuve ---------------------------------------------

// empreinteBase décrit une base par ce qu'une migration y laisse : tables (DDL, contraintes
// comprises), vues, index, séquences, contraintes et nombre de lignes de chaque table. Les
// lignes sont triées : deux bases sont équivalentes quand leurs empreintes sont égales.
func empreinteBase(t *testing.T, db *sql.DB) []string {
	t.Helper()
	requetes := []struct{ genre, sql string }{
		{"table", `SELECT schema_name || '.' || table_name, coalesce(sql, '')
			FROM duckdb_tables() WHERE database_name = current_database() AND NOT internal`},
		{"vue", `SELECT schema_name || '.' || view_name, coalesce(sql, '')
			FROM duckdb_views() WHERE database_name = current_database() AND NOT internal`},
		{"index", `SELECT schema_name || '.' || index_name, table_name || ' ' || coalesce(sql, '')
			FROM duckdb_indexes() WHERE database_name = current_database()`},
		{"sequence", `SELECT schema_name || '.' || sequence_name,
				start_value || ' ' || increment_by || ' ' || min_value || ' ' || max_value || ' ' ||
				cycle || ' ' || coalesce(last_value::VARCHAR, '')
			FROM duckdb_sequences() WHERE database_name = current_database()`},
		{"contrainte", `SELECT schema_name || '.' || table_name,
				constraint_type || ' ' || coalesce(constraint_column_names::VARCHAR, '') || ' ' ||
				coalesce(constraint_text, '')
			FROM duckdb_constraints() WHERE database_name = current_database()`},
	}
	var lignes []string
	for _, q := range requetes {
		rows, err := db.Query(q.sql)
		if err != nil {
			t.Fatalf("empreinte %s: %v", q.genre, err)
		}
		for rows.Next() {
			var nom, detail string
			if err := rows.Scan(&nom, &detail); err != nil {
				_ = rows.Close()
				t.Fatalf("empreinte %s: scan: %v", q.genre, err)
			}
			lignes = append(lignes, q.genre+" "+nom+" | "+detail)
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			t.Fatalf("empreinte %s: %v", q.genre, err)
		}
		_ = rows.Close()
	}
	return append(lignes, comptesDeLignes(t, db)...)
}

// comptesDeLignes rend « lignes <table> | <n> » pour chaque table de la base.
func comptesDeLignes(t *testing.T, db *sql.DB) []string {
	t.Helper()
	rows, err := db.Query(`SELECT schema_name, table_name FROM duckdb_tables()
		WHERE database_name = current_database() AND NOT internal`)
	if err != nil {
		t.Fatalf("liste des tables: %v", err)
	}
	var tables []string
	for rows.Next() {
		var schema, table string
		if err := rows.Scan(&schema, &table); err != nil {
			_ = rows.Close()
			t.Fatalf("liste des tables: scan: %v", err)
		}
		tables = append(tables, `"`+schema+`"."`+table+`"`)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		t.Fatalf("liste des tables: %v", err)
	}
	_ = rows.Close()
	var lignes []string
	for _, tbl := range tables {
		var n int64
		if err := db.QueryRow("SELECT count(*) FROM " + tbl).Scan(&n); err != nil {
			t.Fatalf("count %s: %v", tbl, err)
		}
		lignes = append(lignes, fmt.Sprintf("lignes %s | %d", tbl, n))
	}
	return lignes
}

// exigerMemeEmpreinte échoue en listant les lignes présentes d'un seul côté.
func exigerMemeEmpreinte(t *testing.T, quoi string, want, got []string) {
	t.Helper()
	w, g := append([]string(nil), want...), append([]string(nil), got...)
	sort.Strings(w)
	sort.Strings(g)
	if slices.Equal(w, g) {
		return
	}
	var ecarts []string
	for _, l := range w {
		if _, ok := slices.BinarySearch(g, l); !ok {
			ecarts = append(ecarts, "manque  : "+l)
		}
	}
	for _, l := range g {
		if _, ok := slices.BinarySearch(w, l); !ok {
			ecarts = append(ecarts, "en trop : "+l)
		}
	}
	if len(ecarts) > 12 {
		ecarts = append(ecarts[:12], fmt.Sprintf("... et %d autres", len(ecarts)-12))
	}
	t.Errorf("%s : empreinte différente de la migration neuve (%d lignes attendues, %d obtenues) :\n%s",
		quoi, len(w), len(g), strings.Join(ecarts, "\n"))
}

// releveurApplied note la valeur « applied » de chaque ligne « migration: cycle terminé » : le
// nombre d'étapes qu'un RunForDB a réellement appliquées.
type releveurApplied struct {
	mu      sync.Mutex
	valeurs []int64
}

func (h *releveurApplied) Enabled(context.Context, slog.Level) bool { return true }
func (h *releveurApplied) WithAttrs([]slog.Attr) slog.Handler       { return h }
func (h *releveurApplied) WithGroup(string) slog.Handler            { return h }
func (h *releveurApplied) Handle(_ context.Context, r slog.Record) error {
	if r.Message != "migration: cycle terminé" {
		return nil
	}
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "applied" {
			h.mu.Lock()
			h.valeurs = append(h.valeurs, a.Value.Int64())
			h.mu.Unlock()
		}
		return true
	})
	return nil
}

// empreinteMigrationNeuve migre une base neuve, la ferme puis la rouvre avant d'en prendre
// l'empreinte : le gabarit est lui aussi relu depuis le disque, et DuckDB réécrit alors le texte
// de ses vues et le point de départ de ses séquences. Comparer à une base encore ouverte
// mesurerait cette réécriture, pas la fidélité de la copie.
func empreinteMigrationNeuve(t *testing.T, target migration.TargetDB) []string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "neuve.duckdb")
	raw, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := migration.RunForDB(raw, target); err != nil {
		_ = raw.Close()
		t.Fatalf("RunForDB(%s): %v", target, err)
	}
	if err := raw.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	relue, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("réouverture: %v", err)
	}
	defer relue.Close()
	return empreinteBase(t, relue)
}

// TestMigratedTemplate_Fidelite : pour chaque cible, la copie du gabarit (fichier et mémoire) a
// les mêmes tables, vues, index, séquences, contraintes et nombres de lignes qu'une migration
// neuve, et rejouer RunForDB sur la copie n'applique aucune étape. Ce test migre à neuf : il
// figure sur la liste de no_fresh_migration_in_tests_test.go.
func TestMigratedTemplate_Fidelite(t *testing.T) {
	for _, target := range []migration.TargetDB{
		migration.TargetShared, migration.TargetPlayer, migration.TargetMetadata, migration.TargetSharedSocial,
	} {
		t.Run(string(target), func(t *testing.T) {
			want := empreinteMigrationNeuve(t, target)
			if len(want) == 0 {
				t.Fatalf("empreinte vide pour %s : le test ne prouverait rien", target)
			}

			t.Run("fichier", func(t *testing.T) {
				copie, err := sql.Open("duckdb", MigratedTempFile(t, target, "copie.duckdb"))
				if err != nil {
					t.Fatalf("open copie: %v", err)
				}
				t.Cleanup(func() { _ = copie.Close() })
				exigerMemeEmpreinte(t, "copie fichier", want, empreinteBase(t, copie))
			})
			t.Run("memoire", func(t *testing.T) {
				exigerMemeEmpreinte(t, "copie mémoire", want, empreinteBase(t, OpenMigratedMemory(t, target)))
			})
			t.Run("rejeu_sans_effet", func(t *testing.T) {
				// Le gabarit se construit AVANT la pose du releveur : sa propre migration ne compte pas
				// parmi les étapes du rejeu, y compris quand le sous-test est lancé seul.
				cheminCopie := MigratedTempFile(t, target, "rejeu.duckdb")

				releveur := &releveurApplied{}
				avant := slog.Default()
				slog.SetDefault(slog.New(releveur))
				t.Cleanup(func() { slog.SetDefault(avant) })

				copie, err := sql.Open("duckdb", cheminCopie)
				if err != nil {
					t.Fatalf("open copie: %v", err)
				}
				t.Cleanup(func() { _ = copie.Close() })
				if err := migration.RunForDB(copie, target); err != nil {
					t.Fatalf("RunForDB sur la copie: %v", err)
				}
				releveur.mu.Lock()
				valeurs := append([]int64(nil), releveur.valeurs...)
				releveur.mu.Unlock()
				if len(valeurs) != 1 || valeurs[0] != 0 {
					t.Errorf("étapes appliquées au rejeu = %v, want [0]", valeurs)
				}
				exigerMemeEmpreinte(t, "après rejeu", want, empreinteBase(t, copie))
			})
		})
	}
}
