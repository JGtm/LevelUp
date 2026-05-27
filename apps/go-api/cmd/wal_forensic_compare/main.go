// Command wal_forensic_compare — ADR 0021 Phase 3.3 / Gap 2.
//
// Génère 4 WAL DuckDB volontairement non-CHECKPOINT-és via 4 patterns
// distincts (ATTACH, CREATE TABLE, ALTER TABLE, INSERT) puis dump les
// signatures hex pour comparaison avec le WAL orphelin réel capturé en
// prod (testdata/wal_orphan_fixture/shared_social.duckdb.wal, 2509 B).
//
// Architecture : parent/child via env LEVELUP_FORENSIC_CHILD.
//   - Parent : spawn 4 sub-processes (1 par pattern), chaque child fait son
//     opération DuckDB + os.Exit(0) BRUTAL sans Close → le WAL n'est pas
//     checkpointé et reste sur disque.
//   - Parent ensuite lit chaque WAL, dump hex + strings, compare au WAL réel.
//
// Usage : go run ./apps/go-api/cmd/wal_forensic_compare
package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"
)

const childEnv = "LEVELUP_FORENSIC_CHILD"

func main() {
	// Mode child : produire le WAL selon le pattern demandé, puis exit brutal.
	if pattern := os.Getenv(childEnv); pattern != "" {
		runChild(pattern)
		// Si runChild retourne (erreur), le exit code != 0.
		os.Exit(2)
	}

	tempDir, err := os.MkdirTemp("", "wal_forensic_")
	if err != nil {
		die("mkdir temp", err)
	}
	// Pas de RemoveAll deferred — on veut garder les WAL pour inspection si crash.
	fmt.Println("=== WAL forensic comparison — ADR 0021 Gap 2 ===")
	fmt.Printf("tempdir: %s\n\n", tempDir)

	// Résoudre le path absolu du WAL réel (fixture testdata).
	realPath, err := findRealWAL()
	if err != nil {
		die("find real WAL", err)
	}

	patterns := []string{"ATTACH", "CREATE_TABLE", "ALTER_TABLE", "INSERT_BULK"}
	type sig struct {
		name    string
		size    int64
		hexHead string
		strings []string
		err     error
	}
	var sigs []sig

	for _, p := range patterns {
		dbPath := filepath.Join(tempDir, p+".duckdb")
		fmt.Printf("[gen] %s (sub-process)...\n", p)
		cmd := exec.Command(os.Args[0])
		cmd.Env = append(os.Environ(), childEnv+"="+p, "LEVELUP_FORENSIC_DB="+dbPath)
		_ = cmd.Run() // exit brutal attendu — on ignore l'exit code

		walPath := dbPath + ".wal"
		size, head, strs := analyzeWAL(walPath)
		sigs = append(sigs, sig{name: p, size: size, hexHead: head, strings: strs})
	}

	fmt.Printf("[real] %s\n", realPath)
	rsize, rhead, rstrings := analyzeWAL(realPath)
	sigs = append(sigs, sig{name: "REAL_PROD", size: rsize, hexHead: rhead, strings: rstrings})

	fmt.Println()
	fmt.Println("=== Signatures comparison ===")
	for _, s := range sigs {
		fmt.Printf("\n--- %s ---\n", s.name)
		if s.err != nil {
			fmt.Printf("  ERROR: %v\n", s.err)
			continue
		}
		fmt.Printf("  size: %d bytes\n", s.size)
		fmt.Printf("  head: %s\n", s.hexHead)
		if len(s.strings) > 0 {
			fmt.Printf("  strings (>=5 chars, first 12):\n")
			for i, str := range s.strings {
				if i >= 12 {
					break
				}
				fmt.Printf("    %q\n", str)
			}
		}
	}

	fmt.Println("\n=== Matching analysis (REAL_PROD vs témoins) ===")
	realStrs := sigs[len(sigs)-1].strings
	realSet := make(map[string]struct{})
	for _, s := range realStrs {
		realSet[s] = struct{}{}
	}
	for _, s := range sigs[:len(sigs)-1] {
		if s.err != nil {
			continue
		}
		common := 0
		for _, str := range s.strings {
			if _, ok := realSet[str]; ok {
				common++
			}
		}
		sizeDelta := s.size - rsize
		headMatch := false
		if len(s.hexHead) >= 20 && len(rhead) >= 20 {
			headMatch = s.hexHead[:20] == rhead[:20]
		}
		fmt.Printf("  %-15s size_delta=%+d common_strings=%d/%d  head_match=%v\n",
			s.name, sizeDelta, common, len(realStrs), headMatch)
	}
	fmt.Println()
	fmt.Println("CONCLUSION : le pattern avec le plus de common_strings + size_delta minimal")
	fmt.Println("est le candidat le plus probable pour le type d'opération coupable du")
	fmt.Println("WAL orphelin réel capturé en prod.")
	fmt.Printf("\n(tempdir conservé pour inspection : %s)\n", tempDir)
}

func runChild(pattern string) {
	dbPath := os.Getenv("LEVELUP_FORENSIC_DB")
	if dbPath == "" {
		fmt.Fprintln(os.Stderr, "CHILD: LEVELUP_FORENSIC_DB non set")
		return
	}
	switch pattern {
	case "ATTACH":
		childAttach(dbPath)
	case "CREATE_TABLE":
		childCreateTable(dbPath)
	case "ALTER_TABLE":
		childAlterTable(dbPath)
	case "INSERT_BULK":
		childInsertBulk(dbPath)
	default:
		fmt.Fprintf(os.Stderr, "CHILD: pattern inconnu %q\n", pattern)
		return
	}
	// EXIT BRUTAL : pas de Close, pas de CHECKPOINT → WAL non-checkpointé.
	os.Exit(0)
}

func childAttach(dbPath string) {
	// Setup : DB principale + DB attached, toutes deux préparées + CHECKPOINT.
	otherPath := dbPath + ".other"
	otherDB, _ := sql.Open("duckdb", otherPath)
	otherDB.Exec(`CREATE TABLE t (id INTEGER); CHECKPOINT;`)
	otherDB.Close()

	db, _ := sql.Open("duckdb", dbPath)
	db.Exec(`CREATE TABLE media_files (id INTEGER, indexed_at TIMESTAMP); CHECKPOINT;`)
	// ATTACH + opération mixte → ce qui mettait dans le WAL legacy.
	db.Exec(fmt.Sprintf(`ATTACH '%s' AS attached_other`, strings.ReplaceAll(otherPath, `\`, `/`)))
	db.Exec(`INSERT INTO media_files VALUES (1, NOW())`)
	// Pas de Close → exit brutal va couper le handle, WAL reste.
}

func childCreateTable(dbPath string) {
	db, _ := sql.Open("duckdb", dbPath)
	db.Exec(`CREATE TABLE init (id INTEGER); INSERT INTO init VALUES (0); CHECKPOINT;`)
	db.Exec(`CREATE TABLE media_files (id INTEGER, file_path VARCHAR, indexed_at TIMESTAMP, liked BOOLEAN);`)
}

func childAlterTable(dbPath string) {
	db, _ := sql.Open("duckdb", dbPath)
	db.Exec(`CREATE TABLE media_files (id INTEGER, file_path VARCHAR); CHECKPOINT;`)
	db.Exec(`ALTER TABLE media_files ADD COLUMN indexed_at TIMESTAMP`)
	db.Exec(`ALTER TABLE media_files ADD COLUMN liked BOOLEAN`)
}

func childInsertBulk(dbPath string) {
	db, _ := sql.Open("duckdb", dbPath)
	db.Exec(`
		CREATE TABLE media_files (id INTEGER, file_path VARCHAR, indexed_at TIMESTAMP, liked BOOLEAN, discord_notified BOOLEAN);
		CHECKPOINT;
	`)
	for i := 0; i < 100; i++ {
		db.Exec(`INSERT INTO media_files VALUES (?, ?, NOW(), FALSE, FALSE)`,
			i, fmt.Sprintf("/m/%d.mp4", i))
	}
}

// findRealWAL résout le path absolu du WAL orphelin fixture depuis le repo.
func findRealWAL() (string, error) {
	relPath := filepath.Join("internal", "platform", "duckdb", "testdata",
		"wal_orphan_fixture", "shared_social.duckdb.wal")
	// Si on est dans apps/go-api/, c'est direct.
	if _, err := os.Stat(relPath); err == nil {
		return filepath.Abs(relPath)
	}
	// Si on est à la racine du repo.
	repoPath := filepath.Join("apps", "go-api", relPath)
	if _, err := os.Stat(repoPath); err == nil {
		return filepath.Abs(repoPath)
	}
	return "", fmt.Errorf("WAL fixture introuvable depuis %s — relance depuis apps/go-api/ ou racine repo", mustGetwd())
}

func mustGetwd() string {
	wd, _ := os.Getwd()
	return wd
}

func analyzeWAL(path string) (size int64, hexHead string, strs []string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Sprintf("ERROR read: %v", err), nil
	}
	size = int64(len(data))
	headLen := 48
	if size < int64(headLen) {
		headLen = int(size)
	}
	var hex strings.Builder
	for i := 0; i < headLen; i++ {
		hex.WriteString(fmt.Sprintf("%02X ", data[i]))
	}
	hexHead = hex.String()
	strs = extractStrings(data, 5)
	return size, hexHead, strs
}

func extractStrings(data []byte, minLen int) []string {
	var out []string
	var current bytes.Buffer
	for _, b := range data {
		if b >= 32 && b <= 126 {
			current.WriteByte(b)
		} else {
			if current.Len() >= minLen {
				out = append(out, current.String())
			}
			current.Reset()
		}
	}
	if current.Len() >= minLen {
		out = append(out, current.String())
	}
	seen := make(map[string]struct{})
	dedup := out[:0]
	for _, s := range out {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		dedup = append(dedup, s)
	}
	return dedup
}

func die(what string, err error) {
	fmt.Fprintf(os.Stderr, "ERROR %s: %v\n", what, err)
	os.Exit(2)
}
