// Package ops — seed_demo_wal_test.go : le couple (base DuckDB, WAL) des seeds démo.
//
// Un `<db>.wal` laissé par la génération précédente survit au retrait du seul
// fichier `.duckdb` et est REJOUÉ par DuckDB contre la base reconstruite. Les seeds
// démo recréent 6 bases à chaque regen : le traitement du WAL ne peut donc pas être
// une politesse locale, il est centralisé dans removeDuckDBForFreshWrite.
//
// Deux protections complémentaires ici :
//   - TestRemoveDuckDBForFreshWrite_* : comportement du point de passage unique ;
//   - TestSeedDemoNoBareOsRemove : garde-rail source (scan des seed_demo*.go) —
//     sans lui le pattern re-diverge à la première base ajoutée.
//
// Le scénario end-to-end (WAL orphelin préexistant → extraction saine) est dans
// seed_demo_integration_test.go (DuckDB live, build tag integration).
package ops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── removeDuckDBForFreshWrite ────────────────────────────────────────────────

func TestRemoveDuckDBForFreshWrite_RemovesDBAndWAL(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "shared_matches_v2.duckdb")
	if err := os.WriteFile(db, []byte("ANCIENNE_BASE"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(db+".wal", []byte("WAL_ANCIENNE_GENERATION"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := removeDuckDBForFreshWrite(db); err != nil {
		t.Fatalf("removeDuckDBForFreshWrite: %v", err)
	}

	if _, err := os.Stat(db); !os.IsNotExist(err) {
		t.Errorf("base toujours présente, stat err = %v", err)
	}
	if _, err := os.Stat(db + ".wal"); !os.IsNotExist(err) {
		t.Errorf("WAL orphelin survivant : DuckDB le rejouerait contre la base fraîche (stat err = %v)", err)
	}
}

func TestRemoveDuckDBForFreshWrite_AbsentIsNotAnError(t *testing.T) {
	dir := t.TempDir()
	// Ni base ni WAL : premier seed sur un volume vide.
	if err := removeDuckDBForFreshWrite(filepath.Join(dir, "absent.duckdb")); err != nil {
		t.Fatalf("chemin inexistant doit être toléré, got %v", err)
	}
	// WAL seul (base déjà retirée par un seed interrompu) : c'est précisément l'état
	// corrupteur, il doit disparaître.
	only := filepath.Join(dir, "orphan.duckdb")
	if err := os.WriteFile(only+".wal", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := removeDuckDBForFreshWrite(only); err != nil {
		t.Fatalf("WAL seul: %v", err)
	}
	if _, err := os.Stat(only + ".wal"); !os.IsNotExist(err) {
		t.Errorf("WAL seul non retiré, stat err = %v", err)
	}
}

// TestRemoveDuckDBForFreshWrite_WALBlockedKeepsDB : si le WAL résiste, on échoue
// AVANT de délier la base — sinon on publierait le couple (base absente, WAL
// présent), exactement l'état qu'on cherche à éviter.
func TestRemoveDuckDBForFreshWrite_WALBlockedKeepsDB(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "stats.duckdb")
	if err := os.WriteFile(db, []byte("BASE"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Répertoire NON VIDE à la place du WAL → os.Remove échoue (portable).
	if err := os.MkdirAll(filepath.Join(db+".wal", "inner"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := removeDuckDBForFreshWrite(db); err == nil {
		t.Fatal("attendu : erreur quand le WAL ne peut pas être retiré")
	}
	if _, err := os.Stat(db); err != nil {
		t.Errorf("base déliée malgré l'échec sur le WAL: %v", err)
	}
}

// ── Garde-rail source ────────────────────────────────────────────────────────

// allowedBareOsRemove : les SEULES lignes de seed_demo*.go (hors tests) autorisées à
// appeler os.Remove/os.RemoveAll directement, par fichier puis par fragment de ligne.
// Tout le reste doit passer par removeDuckDBForFreshWrite.
//
// Ajouter une entrée = affirmer, avec justification datée, que le chemin visé n'est
// PAS une base DuckDB (donc n'a pas de WAL adjacent).
var allowedBareOsRemove = map[string][]string{
	"seed_demo.go": {
		// Temporaire de copyMetadataFile : créé par os.CreateTemp, jamais ouvert par
		// DuckDB (2026-07-26).
		"_ = os.Remove(tmpPath)",
		// removeForInodeSwap : l'implémentation elle-même, appelée par
		// removeDuckDBForFreshWrite (2026-07-26).
		"if err := os.Remove(path); err != nil && !os.IsNotExist(err) {",
	},
}

func TestSeedDemoNoBareOsRemove(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read dir package: %v", err)
	}
	scanned, allowlistHits := 0, 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, "seed_demo") ||
			!strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		scanned++
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		for i, line := range strings.Split(string(src), "\n") {
			if !strings.Contains(line, "os.Remove(") && !strings.Contains(line, "os.RemoveAll(") {
				continue
			}
			trimmed := strings.TrimSpace(line)
			allowed := false
			for _, frag := range allowedBareOsRemove[name] {
				if strings.Contains(trimmed, frag) {
					allowed, allowlistHits = true, allowlistHits+1
					break
				}
			}
			if !allowed {
				t.Errorf("%s:%d — os.Remove nu sur un chemin de seed démo :\n\t%s\n"+
					"Utiliser removeDuckDBForFreshWrite (retire la base ET son WAL) ; "+
					"si le chemin n'est pas une base DuckDB, l'ajouter à allowedBareOsRemove avec justification.",
					name, i+1, trimmed)
			}
		}
	}
	// Sentinelles anti-faux-vert : un scan qui ne voit plus rien passerait au vert.
	if scanned < 5 {
		t.Fatalf("seuls %d fichiers seed_demo*.go scannés — scan cassé ou fichiers renommés", scanned)
	}
	if allowlistHits != len(allowedBareOsRemove["seed_demo.go"]) {
		t.Errorf("allowlist matchée %d fois, attendu %d — entrée périmée (code déplacé/réécrit) : la nettoyer",
			allowlistHits, len(allowedBareOsRemove["seed_demo.go"]))
	}
}
