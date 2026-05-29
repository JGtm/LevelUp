// Command rebuild_shared_social — reconstruit shared_social.duckdb à partir
// d'une instance corrompue par un WAL legacy non-rejouable.
//
// Contexte : bug DuckDB upstream #7659 — un ATTACH/DDL legacy a écrit dans
// shared_social.duckdb.wal (puis dans le header de la DB principale) une
// entrée non-rejouable. La DB refuse de s'ouvrir en READ_WRITE même après
// quarantaine du .wal. En READ_ONLY, DuckDB skip le replay → la DB est
// lisible mais pas écrivable.
//
// Stratégie :
//  1. Ouvrir l'ancienne DB en READ_ONLY.
//  2. EXPORT DATABASE vers un répertoire temporaire (format parquet).
//  3. Renommer l'ancienne DB en <path>.corrupt-<ts> (preuve forensique).
//  4. Créer une nouvelle DB vide + IMPORT DATABASE depuis l'export → rejoue
//     schema.sql original (PAS les migrations Go, qui peuvent diverger) +
//     load.sql qui COPY FROM les parquets.
//  5. CHECKPOINT explicite pour vider tout WAL résiduel.
//  6. Vérification post : COUNT(*) par table == baseline pré-rebuild.
//
// Usage :
//
//	go run ./apps/go-api/cmd/rebuild_shared_social \
//	     --db <shared_social.duckdb path> \
//	     [--dry-run]
//
// Toutes les écritures sont guarded par confirmation explicite côté shell.
// Idempotent : si on relance après succès, --dry-run montre que la baseline
// matche déjà.
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

type tableCount struct {
	name  string
	count int64
}

func main() {
	dbPath := flag.String("db", "", "chemin de shared_social.duckdb (obligatoire)")
	dryRun := flag.Bool("dry-run", false, "n'effectue aucune écriture, n'affiche que les counts pré-rebuild")
	flag.Parse()

	if *dbPath == "" {
		fmt.Fprintln(os.Stderr, "--db obligatoire")
		os.Exit(2)
	}
	abs, err := filepath.Abs(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "abs: %v\n", err)
		os.Exit(2)
	}
	*dbPath = abs

	fmt.Printf("=== rebuild_shared_social ===\n")
	fmt.Printf("db: %s\n", *dbPath)
	fmt.Printf("dry-run: %v\n\n", *dryRun)

	// Phase 1 : snapshot baseline (RO).
	baseline, err := snapshotCounts(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "snapshot baseline: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("=== baseline counts (RO) — %d tables ===\n", len(baseline))
	for _, tc := range baseline {
		fmt.Printf("  %-50s %d\n", tc.name, tc.count)
	}
	fmt.Println()

	if *dryRun {
		fmt.Println("[dry-run] skip export + rebuild + import.")
		return
	}

	// Phase 2 : export vers tempdir.
	ts := time.Now().UTC().Format("20060102-150405Z")
	exportDir := filepath.Join(filepath.Dir(*dbPath), "shared_social_export_"+ts)
	if err := os.MkdirAll(exportDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir export: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[1/5] EXPORT DATABASE -> %s\n", exportDir)
	if err := exportDatabase(*dbPath, exportDir); err != nil {
		fmt.Fprintf(os.Stderr, "export: %v\n", err)
		os.Exit(1)
	}

	// Phase 3 : quarantaine de l'ancien fichier corrompu.
	corruptPath := *dbPath + ".corrupt-" + ts
	fmt.Printf("[2/5] rename %s -> %s\n", *dbPath, corruptPath)
	if err := os.Rename(*dbPath, corruptPath); err != nil {
		fmt.Fprintf(os.Stderr, "rename corrupt: %v\n", err)
		os.Exit(1)
	}
	// renommer aussi le .wal s'il existe (orphan ou non) pour ne pas le voir
	// re-pris en compte par DuckDB au CREATE.
	for _, suffix := range []string{".wal", ".tmp"} {
		src := *dbPath + suffix
		if _, statErr := os.Stat(src); statErr == nil {
			_ = os.Rename(src, src+".corrupt-"+ts)
		}
	}

	// Phase 4+5 fusionnées : créer une nouvelle DB vide et faire IMPORT
	// DATABASE qui rejoue schema.sql (CREATE TABLE) + load.sql (COPY FROM
	// parquet). Pas d'application des migrations Go : on préserve le
	// schema réel de la DB live (qui peut diverger des migrations à cause
	// d'historique — création originale via ops.IndexMedia avant que les
	// migrations Go n'existent).
	//
	// schema_migrations est aussi restaurée → le pool au boot verra "12/12
	// déjà appliquées" → no-op (CREATE TABLE IF NOT EXISTS). Le runtime
	// code tolère ce schema (preuve : il marchait avant la corruption WAL).
	//
	// Conséquence : la divergence schema (ancien vs migrations) reste — à
	// résoudre dans une phase séparée (audit Phase 3.1). Ici on PRÉSERVE
	// l'état existant 1:1, c'est la condition non-régression la plus stricte.
	fmt.Printf("[3/5] create fresh empty DB + IMPORT DATABASE\n")
	if err := importExportedDatabase(*dbPath, exportDir); err != nil {
		fmt.Fprintf(os.Stderr, "import: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[4/5] (skipped — IMPORT DATABASE fusionné avec phase 3)\n")

	// Phase 6 : vérification post-import.
	fmt.Printf("[5/5] CHECKPOINT + verification post-rebuild\n")
	if err := checkpointDB(*dbPath); err != nil {
		fmt.Fprintf(os.Stderr, "checkpoint: %v\n", err)
		os.Exit(1)
	}

	final, err := snapshotCounts(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "snapshot final: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\n=== final counts (RO) — %d tables ===\n", len(final))
	for _, tc := range final {
		fmt.Printf("  %-50s %d\n", tc.name, tc.count)
	}

	// Diff baseline vs final.
	diffs := diffCounts(baseline, final)
	if len(diffs) == 0 {
		fmt.Println("\n[OK] non-regression: tous les counts utilisateur préservés")
	} else {
		fmt.Printf("\n[DIFF] %d table(s) avec un delta — vérifier :\n", len(diffs))
		for _, d := range diffs {
			fmt.Printf("  %-50s %s\n", d.name, d.detail)
		}
	}

	fmt.Printf("\n=== done ===\n")
	fmt.Printf("corrupt backup: %s\n", corruptPath)
	fmt.Printf("export tempdir: %s (à supprimer après validation)\n", exportDir)
}

func snapshotCounts(dbPath string) ([]tableCount, error) {
	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")
	if err != nil {
		return nil, fmt.Errorf("open RO: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'main' AND table_type = 'BASE TABLE'
		ORDER BY 1
	`)
	if err != nil {
		return nil, fmt.Errorf("list tables: %w", err)
	}
	var tables []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			rows.Close()
			return nil, err
		}
		tables = append(tables, t)
	}
	rows.Close()
	sort.Strings(tables)

	out := make([]tableCount, 0, len(tables))
	for _, t := range tables {
		var n int64
		if err := db.QueryRow(`SELECT COUNT(*) FROM "` + t + `"`).Scan(&n); err != nil {
			return nil, fmt.Errorf("count %s: %w", t, err)
		}
		out = append(out, tableCount{name: t, count: n})
	}
	return out, nil
}

func exportDatabase(dbPath, exportDir string) error {
	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")
	if err != nil {
		return fmt.Errorf("open RO: %w", err)
	}
	defer db.Close()
	// EXPORT DATABASE écrit schema.sql + load.sql + parquet par table.
	exportPath := strings.ReplaceAll(exportDir, `\`, `/`)
	_, err = db.Exec(fmt.Sprintf("EXPORT DATABASE '%s' (FORMAT PARQUET)", exportPath))
	return err
}

// importExportedDatabase crée une nouvelle DB vide à dbPath puis exécute
// `IMPORT DATABASE 'exportDir'` qui rejoue schema.sql + load.sql exportés
// par EXPORT DATABASE. Préserve le schema d'origine de la DB corrompue,
// pas le schema des migrations Go (qui peut diverger pour cause d'historique).
func importExportedDatabase(dbPath, exportDir string) error {
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return fmt.Errorf("open RW fresh: %w", err)
	}
	defer db.Close()

	exportPath := strings.ReplaceAll(exportDir, `\`, `/`)
	if _, err := db.Exec(fmt.Sprintf("IMPORT DATABASE '%s'", exportPath)); err != nil {
		return fmt.Errorf("IMPORT DATABASE '%s': %w", exportPath, err)
	}
	return nil
}

func checkpointDB(dbPath string) error {
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec("CHECKPOINT")
	return err
}

type tableDiff struct {
	name   string
	detail string
}

func diffCounts(baseline, final []tableCount) []tableDiff {
	bm := make(map[string]int64, len(baseline))
	for _, tc := range baseline {
		bm[tc.name] = tc.count
	}
	fm := make(map[string]int64, len(final))
	for _, tc := range final {
		fm[tc.name] = tc.count
	}

	var diffs []tableDiff
	// Tables présentes en baseline qui doivent matcher (sauf bak_* legacy).
	for name, bc := range bm {
		if strings.HasPrefix(name, "media_match_associations_bak_") {
			continue // legacy backup, skipped by design
		}
		fc, ok := fm[name]
		if !ok {
			diffs = append(diffs, tableDiff{name: name, detail: fmt.Sprintf("disparue (baseline=%d)", bc)})
			continue
		}
		if fc != bc {
			diffs = append(diffs, tableDiff{name: name, detail: fmt.Sprintf("baseline=%d final=%d delta=%+d", bc, fc, fc-bc)})
		}
	}
	// Tables nouvelles dans final qui n'étaient pas en baseline.
	for name := range fm {
		if _, ok := bm[name]; !ok {
			diffs = append(diffs, tableDiff{name: name, detail: "nouvelle (créée par migrations)"})
		}
	}
	sort.Slice(diffs, func(i, j int) bool { return diffs[i].name < diffs[j].name })
	return diffs
}
