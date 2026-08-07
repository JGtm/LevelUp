//go:build cgo

// cmd/seed-assists-model — calcule les coefficients de régression expected_assists
// par mode de jeu et les stocke dans metadata.duckdb (table assists_model_coefs).
//
// Formule : expected_assists = slope × (personal_score + shots_hit) + intercept
// calibrée sur les données historiques de shared_matches_v2.duckdb.
//
// Un enregistrement '__global__' sert de fallback pour les modes sans données.
//
// Usage (depuis apps/go-api/) :
//
//	make run-seed-assists-model
//	# ou directement :
//	PATH=/c/msys64/ucrt64/bin:$PATH CC=gcc CGO_ENABLED=1 go run -tags cgo ./cmd/seed-assists-model/
//
// Idempotent : INSERT OR REPLACE.
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

const minSamples = 30

func main() {
	root := filepath.Join("..", "..")
	sharedPath := filepath.Join(root, "data", "titles", "halo_infinite", "warehouse", "shared_matches_v2.duckdb")
	metaPath := filepath.Join(root, "data", "titles", "halo_infinite", "warehouse", "metadata.duckdb")

	for _, p := range []string{sharedPath, metaPath} {
		if _, err := os.Stat(p); err != nil {
			log.Fatalf("fichier introuvable %s: %v", p, err)
		}
	}

	shared, err := sql.Open("duckdb", sharedPath+"?access_mode=read_only")
	if err != nil {
		log.Fatalf("open shared: %v", err)
	}
	defer shared.Close()

	rows, err := computeCoefs(shared)
	if err != nil {
		log.Fatalf("calcul coefs: %v", err)
	}
	fmt.Printf("Modes calculés : %d (+ 1 global)\n", len(rows)-1)

	meta, err := sql.Open("duckdb", metaPath)
	if err != nil {
		log.Fatalf("open metadata: %v", err)
	}
	defer meta.Close()

	if err := migration.RunForDB(meta, migration.TargetMetadata); err != nil {
		log.Fatalf("migration metadata: %v", err)
	}

	if err := upsertCoefs(meta, rows); err != nil {
		log.Fatalf("upsert coefs: %v", err)
	}

	var total int
	if err := meta.QueryRow("SELECT COUNT(*) FROM assists_model_coefs").Scan(&total); err != nil {
		log.Fatalf("count: %v", err)
	}
	fmt.Printf("assists_model_coefs : %d lignes en base\n", total)
}

type coefRow struct {
	mode      string
	slope     float64
	intercept float64
	r2        float64
	n         int
}

func computeCoefs(db *sql.DB) ([]coefRow, error) {
	const q = `
		WITH base AS (
			SELECT
				p.assists,
				p.personal_score,
				p.shots_hit,
				r.game_variant_name AS mode
			FROM match_participants p
			JOIN match_registry r ON r.match_id = p.match_id
			WHERE p.kills >= 0
			  AND p.shots_hit >= 0 AND NOT isnan(p.shots_hit)
			  AND p.personal_score >= 0
		)
		SELECT
			mode,
			regr_slope(assists,     personal_score + shots_hit) AS slope,
			regr_intercept(assists, personal_score + shots_hit) AS intercept,
			regr_r2(assists,        personal_score + shots_hit) AS r2,
			COUNT(*) AS n
		FROM base
		GROUP BY mode
		HAVING COUNT(*) >= ?
		UNION ALL
		SELECT
			'__global__',
			regr_slope(assists,     personal_score + shots_hit),
			regr_intercept(assists, personal_score + shots_hit),
			regr_r2(assists,        personal_score + shots_hit),
			COUNT(*)
		FROM base
		ORDER BY n DESC
	`

	rows, err := db.Query(q, minSamples)
	if err != nil {
		return nil, fmt.Errorf("query coefs: %w", err)
	}
	defer rows.Close()

	var out []coefRow
	for rows.Next() {
		var c coefRow
		if err := rows.Scan(&c.mode, &c.slope, &c.intercept, &c.r2, &c.n); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func upsertCoefs(db *sql.DB, rows []coefRow) error {
	// computed_at en UTC EXPLICITE : `NOW()` nu rend un TIMESTAMPTZ que DuckDB coerce
	// vers cette colonne TIMESTAMP naive par le fuseau de SESSION — la valeur partirait
	// donc a l'heure locale du poste qui seed, alors que l'ecrivain applicatif
	// (sync.upsertAssistsModels) pose `time.Now().UTC()` sur la meme colonne.
	const q = `
		INSERT OR REPLACE INTO assists_model_coefs
			(game_variant_name, slope, intercept, r2, n_samples, computed_at)
		VALUES (?, ?, ?, ?, ?, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))
	`
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	stmt, err := tx.Prepare(q)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, c := range rows {
		if _, err := stmt.Exec(c.mode, c.slope, c.intercept, c.r2, c.n); err != nil {
			return fmt.Errorf("insert %q: %w", c.mode, err)
		}
		fmt.Printf("  %-40s slope=%.6f intercept=%.4f r2=%.3f n=%d\n",
			c.mode, c.slope, c.intercept, c.r2, c.n)
	}
	return tx.Commit()
}
