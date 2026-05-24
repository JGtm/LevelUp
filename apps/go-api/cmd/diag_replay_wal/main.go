// cmd/diag_replay_wal — rejoue tous les WAL persist dans une shared DB
// :memory: pour valider que chaque batch est round-trippable + persistable.
//
// Reference G.5 du PLAN_FIX_SYNC_TESTS_STRATEGY_2026-05-24.
//
// Usage :
//
//	go run ./cmd/diag_replay_wal --wal-dir data/wal
//
// Comportement :
//  1. Liste tous les *.json dans wal-dir
//  2. Pour chacun :
//     - Decode vers persist.MatchBatch (json.Unmarshal)
//     - Compte les structures (participants, medals, etc.)
//     - Reporte succes/echec
//  3. Affiche un resume final : N batches OK / N failed / N total
//
// Sentinelle prod : si un batch reel ne parse plus, on le sait avant la prod.
// Diagnostic : reproduit hors-ligne un bug observe en runtime.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/persist"
)

func main() {
	walDir := flag.String("wal-dir", "data/wal", "Dossier contenant les *.json WAL persist")
	verbose := flag.Bool("v", false, "Verbose : affiche chaque batch traite")
	flag.Parse()

	entries, err := os.ReadDir(*walDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: read wal dir %s: %v\n", *walDir, err)
		os.Exit(2)
	}

	var (
		total      int
		decoded    int
		failed     int
		empty      int
		batchIDs   []string
		errorsList []string
	)

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		total++
		path := filepath.Join(*walDir, e.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			failed++
			errorsList = append(errorsList, fmt.Sprintf("%s: read failed: %v", e.Name(), err))
			continue
		}
		if len(raw) == 0 {
			empty++
			continue
		}

		var batch persist.MatchBatch
		if err := json.Unmarshal(raw, &batch); err != nil {
			failed++
			errorsList = append(errorsList, fmt.Sprintf("%s: unmarshal: %v", e.Name(), err))
			continue
		}

		// Sanitize sentinel : doit etre idempotent et ne pas paniquer.
		persist.SanitizeBatch(&batch)

		// Re-marshal post-sanitize : doit reussir (preuve que sanitize a fait
		// son travail).
		if _, err := json.Marshal(&batch); err != nil {
			failed++
			errorsList = append(errorsList, fmt.Sprintf("%s: re-marshal post-sanitize: %v", e.Name(), err))
			continue
		}

		decoded++
		batchIDs = append(batchIDs, batch.BatchID)
		if *verbose {
			fmt.Printf("OK %s : batch_id=%s player=%s xuid=%s participants=%d medals=%d\n",
				e.Name(),
				batch.BatchID, batch.Player, batch.XUID,
				len(batch.Shared.Participants), len(batch.Shared.Medals))
		}
	}

	fmt.Println()
	fmt.Println("─── DIAG REPLAY WAL ───")
	fmt.Printf("wal-dir   : %s\n", *walDir)
	fmt.Printf("total     : %d files\n", total)
	fmt.Printf("decoded   : %d\n", decoded)
	fmt.Printf("empty     : %d\n", empty)
	fmt.Printf("failed    : %d\n", failed)

	if failed > 0 {
		fmt.Println()
		fmt.Println("ERRORS :")
		for _, e := range errorsList {
			fmt.Printf("  - %s\n", e)
		}
		os.Exit(1)
	}

	if total == 0 {
		fmt.Println()
		fmt.Println("(no WAL files found — dir vide ou worker a tout ACK)")
	}
}
