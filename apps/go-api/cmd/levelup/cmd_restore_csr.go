// cmd_restore_csr.go — sous-commande restore-csr.
//
// Ajoute à match_skill_rank les CSR par match d'un backup DuckDB legacy (typiquement
// `shared_matches_v2.duckdb` extrait d'une archive) que la base joueur n'a pas encore.
// match_skill_rank est append-only (ADR 0026) : la commande n'efface ni ne modifie aucune
// ligne, elle INSÈRE par persist.AppendOnlyLUSRPersister. Un LUSR présent sur le même match
// reste en place : la vue match_skill_rank_latest fait gagner le CSR (priorité CSR > LUSR par
// match_id). Idempotente : un match qui a déjà une ligne CSR n'est pas réécrit.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/dblease"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
)

// restoreCSRLeaseTimeout — attente maximale du bail d'écriture de la base joueur.
const restoreCSRLeaseTimeout = 30 * time.Second

type restoreCSROptions struct {
	gamertag  string
	titleSlug string
	backup    string
	dryRun    bool
}

func parseRestoreCSRFlags(args []string) (restoreCSROptions, error) {
	fs := flag.NewFlagSet("restore-csr", flag.ExitOnError)
	var o restoreCSROptions
	fs.StringVar(&o.gamertag, "gamertag", "", "Gamertag du joueur (obligatoire)")
	fs.StringVar(&o.titleSlug, "title", titlePkg.DefaultSlug, "Slug du titre (ex: halo_infinite)")
	fs.StringVar(&o.backup, "backup", "", "Path vers le .duckdb extrait du backup (obligatoire)")
	fs.BoolVar(&o.dryRun, "dry-run", false, "Inspecter le schéma et compter sans écrire")
	if err := fs.Parse(args); err != nil {
		return o, err
	}
	if o.gamertag == "" || o.backup == "" {
		return o, fmt.Errorf("--gamertag et --backup sont obligatoires")
	}
	if _, err := os.Stat(o.backup); err != nil {
		return o, fmt.Errorf("backup introuvable %s: %w", o.backup, err)
	}
	return o, nil
}

func runRestoreCSR(cfg *config.AppConfig, args []string) error {
	ctx := context.Background()
	o, err := parseRestoreCSRFlags(args)
	if err != nil {
		return err
	}
	playerDBPath := titlePkg.NewPathResolver(cfg.RepoRoot).PlayerDBPath(o.titleSlug, o.gamertag)
	if _, err := os.Stat(playerDBPath); err != nil {
		return fmt.Errorf("player DB introuvable %s: %w", playerDBPath, err)
	}
	fmt.Printf("Player DB    : %s\n", playerDBPath)
	fmt.Printf("Backup legacy: %s\n", o.backup)
	fmt.Printf("dry-run      : %t\n", o.dryRun)

	if !o.dryRun {
		if err := applyMigrationsOnDB(playerDBPath, migration.TargetPlayer); err != nil {
			return fmt.Errorf("migrations player: %w", err)
		}
	}

	// Porte unique des ouvertures en écriture : une base joueur y a ses séquences alignées
	// avant toute écriture (internal/platform/duckdb/physical_open.go).
	handle, err := duckdbpkg.OpenReadWrite(playerDBPath)
	if err != nil {
		return fmt.Errorf("ouverture player DB (serveur LevelUp arrêté ?): %w", err)
	}
	defer func() { _ = handle.Close() }()
	db := handle.SQLDb()

	if err := attachLegacyBackup(ctx, db, o.backup); err != nil {
		return err
	}
	defer detachLegacyBackup(ctx, db)

	sourceTable, nCSR, err := inspectLegacyBackup(db)
	if err != nil {
		return err
	}
	if o.dryRun {
		fmt.Println("[dry-run] aucune écriture")
		return nil
	}

	w, err := dblease.AcquireWriter(db, playerDBPath, dblease.KindPlayer, restoreCSRLeaseTimeout)
	if err != nil {
		return fmt.Errorf("bail d'écriture %s: %w", playerDBPath, err)
	}
	defer w.Release()
	inserted, err := restoreLegacyCSR(ctx, db, w, sourceTable)
	if err != nil {
		return err
	}
	fmt.Println("Restauration terminée")
	fmt.Printf("   CSR legacy lus         : %d\n", nCSR)
	fmt.Printf("   CSR insérés            : %d\n", inserted)
	fmt.Printf("   CSR déjà présents      : %d\n", nCSR-inserted)
	return nil
}

// attachLegacyBackup attache le backup en LECTURE sous l'alias `legacy`.
func attachLegacyBackup(ctx context.Context, db *sql.DB, backup string) error {
	backupEsc := strings.ReplaceAll(backup, "'", "''")
	if _, err := db.ExecContext(ctx, fmt.Sprintf("ATTACH '%s' AS legacy (READ_ONLY)", backupEsc)); err != nil {
		return fmt.Errorf("ATTACH backup: %w", err)
	}
	return nil
}

func detachLegacyBackup(ctx context.Context, db *sql.DB) {
	if _, err := db.ExecContext(ctx, "DETACH legacy"); err != nil {
		slog.WarnContext(ctx, "restore-csr: DETACH legacy", "err", err)
	}
}

// inspectLegacyBackup trouve la table CSR du backup attaché et en imprime le schéma ; sans
// table CSR, imprime le schéma complet du backup et rend une erreur.
func inspectLegacyBackup(db *sql.DB) (string, int, error) {
	sourceTable, nCSR, err := findLegacyCSRTable(db)
	if err != nil {
		return "", 0, fmt.Errorf("inspection backup: %w", err)
	}
	if sourceTable == "" {
		fmt.Println("Aucun candidat n'a fourni de CSR. Listing complet du schéma legacy :")
		if listErr := listLegacyTables(db); listErr != nil {
			return "", 0, fmt.Errorf("list tables: %w", listErr)
		}
		fmt.Println()
		fmt.Println("Inspection des colonnes 'csr*' / 'rank*' / 'skill*' dans toutes les tables :")
		if scanErr := scanLegacyForCSRColumns(db); scanErr != nil {
			return "", 0, fmt.Errorf("scan colonnes: %w", scanErr)
		}
		return "", 0, fmt.Errorf("aucune table CSR trouvée (essais: match_skill_rank, match_stats, match_csr_snapshots)")
	}
	fmt.Printf("Source CSR   : legacy.%s (%d lignes 'CSR')\n", sourceTable, nCSR)
	if err := describeLegacyTable(db, sourceTable); err != nil {
		return "", 0, fmt.Errorf("describe legacy.%s: %w", sourceTable, err)
	}
	return sourceTable, nCSR, nil
}

// restoreLegacyCSR insère, pour chaque match de legacy.<sourceTable> porteur d'un CSR et sans
// ligne CSR dans match_skill_rank, une ligne CSR (une seule par match : un doublon du backup
// est départagé par la valeur la plus haute, choix déterministe). Lecture sur db, écriture INSERT pure par le persisteur append-only sous le bail w.
// Rend le nombre de lignes insérées.
func restoreLegacyCSR(ctx context.Context, db *sql.DB, w *dblease.LeasedWriter, sourceTable string) (int, error) {
	rows, err := loadMissingLegacyCSR(ctx, db, sourceTable)
	if err != nil {
		return 0, err
	}
	if err := persist.NewAppendOnlyLUSRPersister(w).Persist(ctx, rows); err != nil {
		return 0, fmt.Errorf("INSERT CSR: %w", err)
	}
	return len(rows), nil
}

// loadMissingLegacyCSR lit les CSR du backup absents de match_skill_rank (rating_type CSR).
func loadMissingLegacyCSR(ctx context.Context, db *sql.DB, sourceTable string) ([]persist.LUSRRatingInsert, error) {
	q := fmt.Sprintf(`
		SELECT l.match_id, l.rating_value, l.rating_deviation, l.tier, l.sub_tier,
		       l.tier_label, l.rating_delta, l.playlist_group, l.start_time
		FROM legacy.%s l
		WHERE l.rating_type = 'CSR'
		  AND NOT EXISTS (
			SELECT 1 FROM match_skill_rank m
			WHERE m.match_id = l.match_id AND m.rating_type = 'CSR')
		QUALIFY ROW_NUMBER() OVER (
			PARTITION BY l.match_id ORDER BY l.rating_value DESC NULLS LAST, l.rating_deviation) = 1
		ORDER BY l.match_id`, sourceTable)
	rs, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("lecture CSR legacy.%s: %w", sourceTable, err)
	}
	defer func() { _ = rs.Close() }()
	var out []persist.LUSRRatingInsert
	for rs.Next() {
		r, err := scanLegacyCSRRow(rs)
		if err != nil {
			return nil, fmt.Errorf("scan CSR legacy.%s: %w", sourceTable, err)
		}
		out = append(out, r)
	}
	return out, rs.Err()
}

// scanLegacyCSRRow convertit une ligne CSR du backup ; sub_tier absent vaut 0 et
// playlist_group absent vaut « ranked », comme à l'écriture d'origine.
func scanLegacyCSRRow(rs *sql.Rows) (persist.LUSRRatingInsert, error) {
	var (
		matchID            string
		value, deviation   sql.NullFloat64
		tier, label, group sql.NullString
		subTier            sql.NullInt64
		delta              sql.NullFloat64
		startTime          sql.NullTime
	)
	if err := rs.Scan(&matchID, &value, &deviation, &tier, &subTier, &label, &delta, &group, &startTime); err != nil {
		return persist.LUSRRatingInsert{}, err
	}
	sub := 0
	if subTier.Valid {
		sub = int(subTier.Int64)
	}
	r := persist.LUSRRatingInsert{
		MatchID:         matchID,
		RatingType:      "CSR",
		RatingValue:     value.Float64,
		RatingDeviation: deviation.Float64,
		SubTier:         &sub,
		PlaylistGroup:   "ranked",
	}
	if tier.Valid {
		r.Tier = &tier.String
	}
	if label.Valid {
		r.TierLabel = &label.String
	}
	if delta.Valid {
		r.RatingDelta = &delta.Float64
	}
	if group.Valid && group.String != "" {
		r.PlaylistGroup = group.String
	}
	if startTime.Valid {
		r.StartTime = &startTime.Time
	}
	return r, nil
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
