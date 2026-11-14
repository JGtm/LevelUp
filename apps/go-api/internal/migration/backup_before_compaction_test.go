package migration

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

func fauxVolume(string) (uint64, error) { return 1 << 40, nil }

func fichierBase(t *testing.T, racine, slug, contenu string) string {
	t.Helper()
	dir := filepath.Join(racine, slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "shared_matches_v2.duckdb")
	if err := os.WriteFile(path, []byte(contenu), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func noms(t *testing.T, dir string) []string {
	t.Helper()
	entrees, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entrees {
		out = append(out, e.Name())
	}
	sort.Strings(out)
	return out
}

// La copie d'avant montée de schéma survit à N compactions : la compaction a son propre dossier, son
// propre nom et sa propre rétention, et deux titres dont les bases portent le même nom de fichier ne
// s'effacent jamais l'un l'autre, même dans un dossier commun.
func TestBackupBeforeCompaction_NeToucheNiLaMigrationNiLAutreTitre(t *testing.T) {
	ctx := context.Background()
	racine := t.TempDir()
	infinite := fichierBase(t, racine, "halo_infinite", "infinite")
	halo5 := fichierBase(t, racine, "halo_5", "halo5")
	migrations := filepath.Join(racine, "backups", "migrations")
	compactions := filepath.Join(racine, "backups", "compaction")
	debut := time.Date(2026, 11, 14, 8, 0, 0, 0, time.UTC)

	if _, err := BackupBeforeUpgrade(ctx, BackupParams{
		DBPath: infinite, DestDir: migrations, Version: 238, Now: debut, FreeBytes: fauxVolume,
	}); err != nil {
		t.Fatalf("copie d'avant montée: %v", err)
	}
	avantMigration := noms(t, migrations)
	if len(avantMigration) != 1 {
		t.Fatalf("copies d'avant montée = %v, attendu 1", avantMigration)
	}

	const nCompactions = 6
	for i := 1; i <= nCompactions; i++ {
		for slug, path := range map[string]string{"halo_infinite": infinite, "halo_5": halo5} {
			if _, err := BackupBeforeCompaction(ctx, CompactionBackupParams{
				DBPath: path, DestDir: compactions, Slug: slug,
				Now: debut.Add(time.Duration(i) * time.Minute), FreeBytes: fauxVolume,
			}); err != nil {
				t.Fatalf("compaction %d (%s): %v", i, slug, err)
			}
		}
	}

	if got := noms(t, migrations); strings.Join(got, ",") != strings.Join(avantMigration, ",") {
		t.Errorf("la copie d'avant montée a changé : %v, attendu %v", got, avantMigration)
	}
	par := map[string]int{}
	for _, n := range noms(t, compactions) {
		switch {
		case strings.Contains(n, ".halo_infinite.avant-compaction-"):
			par["halo_infinite"]++
		case strings.Contains(n, ".halo_5.avant-compaction-"):
			par["halo_5"]++
		default:
			t.Errorf("copie de compaction sans titre dans son nom : %s", n)
		}
	}
	if par["halo_infinite"] != CompactionBackupRetention || par["halo_5"] != CompactionBackupRetention {
		t.Errorf("copies par titre = %v, attendu %d de chaque (rétention par titre)", par, CompactionBackupRetention)
	}
}

// Sans titre, la copie est refusée.
func TestBackupBeforeCompaction_RefuseSansTitre(t *testing.T) {
	path := fichierBase(t, t.TempDir(), "halo_infinite", "x")
	if _, err := BackupBeforeCompaction(context.Background(), CompactionBackupParams{
		DBPath: path, DestDir: t.TempDir(), Now: time.Now(), FreeBytes: fauxVolume,
	}); err == nil {
		t.Error("copie sans titre acceptée")
	}
}
