// cmd_restore_csr.go — sous-commande restore-csr.
//
// Restaure les valeurs CSR par-match depuis un backup DuckDB legacy
// (typiquement `shared_matches_v2.duckdb` extrait d'une archive). One-shot,
// idempotent, conçu pour le cas où les CSR ont été écrasés par LUSR avant
// que le garde-fou SQL ne soit en place.
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
)

func runRestoreCSR(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("restore-csr", flag.ExitOnError)
	gamertag := fs.String("gamertag", "", "Gamertag du joueur (obligatoire)")
	titleSlug := fs.String("title", titlePkg.DefaultSlug, "Slug du titre (ex: halo_infinite)")
	backup := fs.String("backup", "", "Path vers le .duckdb extrait du backup (obligatoire)")
	dryRun := fs.Bool("dry-run", false, "Inspecter le schéma et compter sans écrire")
	mode := fs.String("mode", "overwrite",
		"preserve|overwrite — overwrite supprime les LUSR fautifs sur les matchs à restaurer")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *gamertag == "" || *backup == "" {
		return fmt.Errorf("--gamertag et --backup sont obligatoires")
	}
	if *mode != "preserve" && *mode != "overwrite" {
		return fmt.Errorf("--mode doit valoir 'preserve' ou 'overwrite' (got %q)", *mode)
	}
	if _, err := os.Stat(*backup); err != nil {
		return fmt.Errorf("backup introuvable %s: %w", *backup, err)
	}

	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	playerDBPath := pr.PlayerDBPath(*titleSlug, *gamertag)
	if _, err := os.Stat(playerDBPath); err != nil {
		return fmt.Errorf("player DB introuvable %s: %w", playerDBPath, err)
	}
	fmt.Printf("Player DB    : %s\n", playerDBPath)
	fmt.Printf("Backup legacy: %s\n", *backup)
	fmt.Printf("Mode         : %s (dry-run=%t)\n", *mode, *dryRun)

	if !*dryRun {
		if err := applyMigrationsOnDB(playerDBPath, migration.TargetPlayer); err != nil {
			return fmt.Errorf("migrations player: %w", err)
		}
	}

	db, err := sql.Open("duckdb", playerDBPath)
	if err != nil {
		return fmt.Errorf("ouverture player DB: %w", err)
	}
	defer db.Close()

	backupEsc := strings.ReplaceAll(*backup, "'", "''")
	if _, err := db.Exec(fmt.Sprintf("ATTACH '%s' AS legacy (READ_ONLY)", backupEsc)); err != nil {
		return fmt.Errorf("ATTACH backup: %w", err)
	}
	defer func() { _, _ = db.Exec("DETACH legacy") }()

	sourceTable, nCSR, err := findLegacyCSRTable(db)
	if err != nil {
		return fmt.Errorf("inspection backup: %w", err)
	}
	if sourceTable == "" {
		fmt.Println("Aucun candidat n'a fourni de CSR. Listing complet du schéma legacy :")
		if listErr := listLegacyTables(db); listErr != nil {
			return fmt.Errorf("list tables: %w", listErr)
		}
		fmt.Println()
		fmt.Println("Inspection des colonnes 'csr*' / 'rank*' / 'skill*' dans toutes les tables :")
		if scanErr := scanLegacyForCSRColumns(db); scanErr != nil {
			return fmt.Errorf("scan colonnes: %w", scanErr)
		}
		return fmt.Errorf("aucune table CSR trouvée (essais: match_skill_rank, match_stats, match_csr_snapshots)")
	}
	fmt.Printf("Source CSR   : legacy.%s (%d lignes 'CSR')\n", sourceTable, nCSR)

	if err := describeLegacyTable(db, sourceTable); err != nil {
		return fmt.Errorf("describe legacy.%s: %w", sourceTable, err)
	}

	if *dryRun {
		fmt.Println("[dry-run] aucune écriture")
		return nil
	}

	var deleted int64
	if *mode == "overwrite" {
		res, err := db.Exec(fmt.Sprintf(`
			DELETE FROM match_skill_rank
			WHERE rating_type = 'LUSR'
			  AND match_id IN (SELECT match_id FROM legacy.%s WHERE rating_type='CSR')`,
			sourceTable))
		if err != nil {
			return fmt.Errorf("DELETE LUSR fautifs: %w", err)
		}
		deleted, _ = res.RowsAffected()
	}

	res, err := db.Exec(fmt.Sprintf(`
		INSERT INTO match_skill_rank
			(match_id, rating_type, rating_value, rating_deviation,
			 tier, tier_fr, sub_tier, tier_label,
			 rating_delta, playlist_group, start_time, created_at, updated_at)
		SELECT
			l.match_id, 'CSR', l.rating_value, l.rating_deviation,
			l.tier, NULL, COALESCE(l.sub_tier, 0), l.tier_label,
			l.rating_delta, COALESCE(l.playlist_group, 'ranked'),
			l.start_time, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		FROM legacy.%s l
		WHERE l.rating_type = 'CSR'
		ON CONFLICT (match_id) DO NOTHING`, sourceTable))
	if err != nil {
		return fmt.Errorf("INSERT CSR: %w", err)
	}
	inserted, _ := res.RowsAffected()

	fmt.Printf("✅ Restauration terminée\n")
	fmt.Printf("   LUSR fautifs supprimés : %d\n", deleted)
	fmt.Printf("   CSR insérés            : %d\n", inserted)
	fmt.Printf("   CSR skippés (conflit)  : %d\n", int64(nCSR)-inserted)
	return nil
}

// findLegacyCSRTable cherche la table contenant les CSR dans la base attachée
// sous l'alias `legacy`. Essaie chaque candidat par SELECT direct (catche
// l'erreur "table not found") et retourne la première qui contient ≥1 CSR.
func findLegacyCSRTable(db *sql.DB) (string, int, error) {
	candidates := []string{"match_skill_rank", "match_stats", "match_csr_snapshots"}
	for _, table := range candidates {
		var n int
		err := db.QueryRow(fmt.Sprintf(
			`SELECT COUNT(*) FROM legacy.%s WHERE rating_type = 'CSR'`, table),
		).Scan(&n)
		if err != nil {
			continue
		}
		if n > 0 {
			return table, n, nil
		}
	}
	return "", 0, nil
}

func listLegacyTables(db *sql.DB) error {
	rows, err := db.Query(`
		SELECT table_name, estimated_size
		FROM duckdb_tables()
		WHERE database_name = 'legacy'
		ORDER BY table_name`)
	if err != nil {
		// Fallback : information_schema.
		rows2, err2 := db.Query(`
			SELECT table_name, 0 AS estimated_size
			FROM information_schema.tables
			WHERE table_catalog = 'legacy'
			ORDER BY table_name`)
		if err2 != nil {
			return fmt.Errorf("duckdb_tables() ET information_schema KO: %v / %v", err, err2)
		}
		rows = rows2
	}
	defer rows.Close()
	fmt.Printf("  %-40s %12s\n", "table_name", "est_rows")
	fmt.Printf("  %s\n", strings.Repeat("-", 56))
	for rows.Next() {
		var name string
		var estSize int64
		if err := rows.Scan(&name, &estSize); err != nil {
			return err
		}
		fmt.Printf("  %-40s %12d\n", name, estSize)
	}
	return rows.Err()
}

// scanLegacyForCSRColumns inspecte information_schema pour trouver des colonnes
// potentiellement liées au CSR (csr*, rank*, skill*, rating*, mmr*) à travers
// toutes les tables du backup attaché.
func scanLegacyForCSRColumns(db *sql.DB) error {
	rows, err := db.Query(`
		SELECT table_name, column_name, data_type
		FROM information_schema.columns
		WHERE table_catalog = 'legacy'
		  AND (lower(column_name) LIKE '%csr%'
		    OR lower(column_name) LIKE '%rank%'
		    OR lower(column_name) LIKE '%skill%'
		    OR lower(column_name) LIKE '%rating%'
		    OR lower(column_name) LIKE '%mmr%'
		    OR lower(column_name) LIKE '%tier%')
		ORDER BY table_name, column_name`)
	if err != nil {
		return err
	}
	defer rows.Close()
	fmt.Printf("  %-25s %-30s %s\n", "table", "column", "type")
	fmt.Printf("  %s\n", strings.Repeat("-", 70))
	any := false
	for rows.Next() {
		var t, c, dt string
		if err := rows.Scan(&t, &c, &dt); err != nil {
			return err
		}
		fmt.Printf("  %-25s %-30s %s\n", t, c, dt)
		any = true
	}
	if !any {
		fmt.Println("  (aucune colonne suspecte trouvée)")
	}
	return rows.Err()
}

func describeLegacyTable(db *sql.DB, table string) error {
	rows, err := db.Query(fmt.Sprintf(`DESCRIBE legacy.%s`, table))
	if err != nil {
		return err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return err
	}
	fmt.Printf("Schéma legacy.%s :\n", table)
	for rows.Next() {
		vals := make([]sql.NullString, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return err
		}
		var name, typ string
		if len(vals) >= 1 {
			name = vals[0].String
		}
		if len(vals) >= 2 {
			typ = vals[1].String
		}
		fmt.Printf("  - %-20s %s\n", name, typ)
	}
	return rows.Err()
}
