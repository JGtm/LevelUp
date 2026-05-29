// Command duckdb_7659_repro — repro minimal du bug DuckDB upstream #7659
// (`WAL Replay fails when attach alias changes`).
//
// Reproduit la séquence :
//  1. CREATE TABLE + INSERT bulk dans une DB DuckDB
//  2. exit avant CHECKPOINT (simule un kill brutal)
//  3. reopen → DuckDB tente le replay du WAL → INTERNAL Error:
//     "Failure while replaying WAL file ... Calling DatabaseManager::GetDefaultDatabase"
//
// Ce fichier sert d'artefact à attacher à un bug report upstream.
//
// Usage : go run ./apps/go-api/cmd/duckdb_7659_repro
//
// La séquence consiste à lancer le binaire 2 fois :
//   - 1er run (env LEVELUP_7659_PHASE=write) : créé + écrit + exit brutal
//   - 2e run (env LEVELUP_7659_PHASE=read) : tente le reopen → bug se manifeste
//
// Sans LEVELUP_7659_PHASE, le binaire lance les 2 phases en sub-process.
package main

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"
)

const (
	envPhase = "LEVELUP_7659_PHASE"
	envDir   = "LEVELUP_7659_DIR"
)

func main() {
	phase := os.Getenv(envPhase)

	switch phase {
	case "write":
		writePhase()
		// Exit brutal sans Close — simule un kill SIGKILL.
		os.Exit(0)
	case "read":
		readPhase()
		return
	}

	// Mode orchestrateur : crée tempdir + spawn child write + spawn child read.
	dir, err := os.MkdirTemp("", "duckdb_7659_repro_")
	if err != nil {
		die("mkdir temp", err)
	}
	defer os.RemoveAll(dir)

	fmt.Printf("=== DuckDB #7659 repro ===\n")
	fmt.Printf("tempdir: %s\n\n", dir)

	// Phase WRITE en sub-process — exit brutal.
	fmt.Printf("[1/2] sub-process WRITE (exit brutal sans Close)...\n")
	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), envPhase+"=write", envDir+"="+dir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		die("write subprocess", err)
	}

	// Phase READ en sub-process — tente le reopen.
	fmt.Printf("\n[2/2] sub-process READ (reopen — doit reproduire le bug)...\n")
	cmd = exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), envPhase+"=read", envDir+"="+dir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run() // on attend une erreur — c'est le bug
}

func writePhase() {
	dir := os.Getenv(envDir)
	if dir == "" {
		die("env "+envDir, fmt.Errorf("non set"))
	}
	dbPath := filepath.Join(dir, "repro.duckdb")

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		die("open RW", err)
	}
	// PAS de defer db.Close() — on exit brutalement.

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS media_files (
			id INTEGER PRIMARY KEY,
			file_path VARCHAR,
			indexed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`); err != nil {
		die("create table", err)
	}

	// Bulk insert pour grossir le WAL (rend le replay plus lourd, donc
	// plus susceptible de heurter le bug d'assertion).
	for i := 0; i < 100; i++ {
		if _, err := db.Exec(
			`INSERT INTO media_files (id, file_path) VALUES (?, ?)`,
			i, fmt.Sprintf("/m/%d.mp4", i),
		); err != nil {
			die(fmt.Sprintf("insert %d", i), err)
		}
	}

	fmt.Printf("  [WRITE] 100 rows insérées dans %s (PAS de CHECKPOINT, exit brutal imminent)\n", dbPath)
	// os.Exit(0) appelé par le caller — court-circuite tout cleanup.
}

func readPhase() {
	dir := os.Getenv(envDir)
	if dir == "" {
		die("env "+envDir, fmt.Errorf("non set"))
	}
	dbPath := filepath.Join(dir, "repro.duckdb")

	fmt.Printf("  [READ] reopen RW de %s...\n", dbPath)
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		fmt.Printf("  [READ] sql.Open échoué (attendu si bug se déclenche au connect) : %v\n", err)
		return
	}
	defer db.Close()

	// Le bug se déclenche typiquement au 1er Ping/Exec (lazy connection).
	if err := db.Ping(); err != nil {
		if strings.Contains(err.Error(), "Failure while replaying WAL file") {
			fmt.Printf("  [READ] !!! BUG #7659 REPRODUIT !!!\n")
			fmt.Printf("  Erreur DuckDB : %v\n", err)
			return
		}
		fmt.Printf("  [READ] erreur différente (pas le bug) : %v\n", err)
		return
	}

	// Si on arrive ici, le bug ne s'est pas manifesté (DuckDB a fait
	// un replay sans assertion fail).
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM media_files`).Scan(&count); err != nil {
		fmt.Printf("  [READ] query échouée : %v\n", err)
		return
	}
	fmt.Printf("  [READ] reopen OK — %d rows lues (bug PAS reproduit cette fois)\n", count)
}

func die(what string, err error) {
	fmt.Fprintf(os.Stderr, "ERREUR %s : %v\n", what, err)
	os.Exit(2)
}
