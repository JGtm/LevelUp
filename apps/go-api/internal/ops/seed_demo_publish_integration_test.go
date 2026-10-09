//go:build integration

// Package ops — seed_demo_publish_integration_test.go : une génération qui échoue au contrôle
// des valeurs ne publie RIEN (revue R1 du lot recos-d, P1-4).
//
// Mutation de contrôle : remplacer l'appel à verifyDemoAnonymization dans SeedDemo par un
// nil rend ce test rouge (la seconde génération serait publiée avec la fuite).
package ops

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
)

func empreinteFichier(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("lecture %s: %v", path, err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(raw))
}

func seedMultiPourTest(ctx context.Context, repoRoot string) (SeedDemoMultiResult, error) {
	return SeedDemoMulti(ctx, SeedDemoMultiOptions{
		RepoRoot: repoRoot, OutDir: filepath.Join(repoRoot, "out"),
		ProfilesPath: filepath.Join(repoRoot, "db_profiles.json"), ServiceTag: "SPTA",
		Titles: []TitleSeedSpec{{Slug: titlePkg.DefaultSlug, Gamertag: "JGtm", MaxMatches: 2}},
	})
}

func TestSeedDemoMulti_FuiteDetecteeAucunePublication(t *testing.T) {
	ctx := context.Background()
	repoRoot := seedDemoCLIRepo(t)
	out := filepath.Join(repoRoot, "out")

	// 1. Une première génération saine est publiée.
	if _, err := seedMultiPourTest(ctx, repoRoot); err != nil {
		t.Fatalf("première génération: %v", err)
	}
	layout := titlePkg.NewDemoLayout(out)
	avantShared := empreinteFichier(t, layout.SharedDBPath(titlePkg.DefaultSlug))
	avantProfils := empreinteFichier(t, filepath.Join(out, "db_profiles.json"))

	// 2. La source gagne une colonne d'identité que l'anonymisation ne déclare pas.
	src, err := sql.Open("duckdb", titlePkg.NewPathResolver(repoRoot).SharedDBPath(titlePkg.DefaultSlug))
	if err != nil {
		t.Fatal(err)
	}
	mustExec(t, src, `ALTER TABLE match_registry ADD COLUMN note_libre VARCHAR`)
	mustExec(t, src, `UPDATE match_registry SET note_libre = 'vu par `+cliSourceXUID+`'`)
	if err := src.Close(); err != nil {
		t.Fatal(err)
	}

	// 3. La seconde génération échoue au contrôle, et rien n'est publié.
	_, err = seedMultiPourTest(ctx, repoRoot)
	if !errors.Is(err, ErrDemoIdentityLeak) {
		t.Fatalf("seconde génération : err = %v, attendu ErrDemoIdentityLeak", err)
	}
	if got := empreinteFichier(t, layout.SharedDBPath(titlePkg.DefaultSlug)); got != avantShared {
		t.Error("la base partagée publiée a changé malgré l'échec du contrôle")
	}
	if got := empreinteFichier(t, filepath.Join(out, "db_profiles.json")); got != avantProfils {
		t.Error("db_profiles.json publié a changé malgré l'échec du contrôle")
	}
	for _, reste := range []string{demoGenerationDir(out), demoPreviousDir(out)} {
		if fileExists(reste) {
			t.Errorf("%s subsiste après l'échec", reste)
		}
	}
}

// TestCopyAnonymizedTables_CorrespondanceRetireeSurErreur — revue R1 (P1-4) : les tables de
// correspondance (identités RÉELLES) sont retirées même quand une copie échoue.
func TestCopyAnonymizedTables_CorrespondanceRetireeSurErreur(t *testing.T) {
	ctx := context.Background()
	srcPath := sourceAnonymisation(t, ctx)
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "demo.duckdb"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.ExecContext(ctx, "ATTACH '"+srcPath+"' AS src (READ_ONLY)"); err != nil {
		t.Fatal(err)
	}
	tables := []extractTable{{name: "match_participants", where: "colonne_inexistante = 1",
		identity: [][2]string{{colXUID, colGamertag}}}}
	if _, err := copyAnonymizedTables(ctx, db, tables, "", []xuidRemap{{from: anonReelA, toXUID: "0000000000000000"}}, false); err == nil {
		t.Fatal("copie en échec attendue")
	}
	var n int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM duckdb_tables()
		WHERE table_name IN (?, ?)`, demoXUIDMapTable, demoGamertagMapTable).Scan(&n); err != nil || n != 0 {
		t.Errorf("tables de correspondance encore présentes après l'erreur : %d (err %v)", n, err)
	}
}
