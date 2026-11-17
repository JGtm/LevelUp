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
	"os"
	"path/filepath"
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
