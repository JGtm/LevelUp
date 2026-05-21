// Package duckdb — art_probe.go : détection de corruption d'index ART
// (Adaptive Radix Tree) sur les tables à PK VARCHAR.
//
// Contexte : bug DuckDB documenté dans
// `docs/INCIDENT_2026-05-20_match_participants_index.md`. Pour certaines
// combinaisons de données, les requêtes `WHERE pk_col = ?` qui empruntent
// le filter pushdown via l'arbre ART retournent un sous-ensemble strict des
// rows réelles. Le workaround "WHERE pk_col concat empty-string equals ?" force un table-scan
// physique qui retourne les rows correctes.
//
// Ce module fournit un détecteur automatique : pour chaque table avec PK
// VARCHAR, échantillonne N valeurs PK et compare les COUNT(*) obtenus via
// les deux méthodes. Toute divergence est rapportée.
//
// Usage :
//   - cmd/diag_art_probe/ : outil ad-hoc CLI multi-DB
//   - cmd/server : filet de garde au boot (log WARN + métrique si divergence)
//   - Tests anti-régression : TestART_FilterPushdown_NoTruncation
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/observability"
)

// ARTDivergence décrit une divergence détectée entre le filter pushdown
// (via index ART) et le table-scan (via concat trick).
type ARTDivergence struct {
	Table        string // nom de la table (qualifié si nécessaire)
	PKColumn     string // première colonne PK testée
	SampleValue  string // valeur PK qui a déclenché la divergence
	CountIndexed int    // count via WHERE pk = ? (filter pushdown)
	CountScan    int    // count via WHERE pk || '' = ? (table-scan)
}

// ARTProbeReport agrège les résultats du probe.
type ARTProbeReport struct {
	TablesScanned int
	SamplesTested int
	Divergences   []ARTDivergence
	Duration      time.Duration
}

// HasDivergence retourne true si au moins une divergence a été détectée.
func (r *ARTProbeReport) HasDivergence() bool {
	return len(r.Divergences) > 0
}

// ProbeARTDivergences scanne toutes les tables de la DB courante qui ont
// une PK VARCHAR et compare COUNT(*) via filter pushdown vs table-scan.
//
// sampleSize : nombre de valeurs PK distinctes à tester par table. 5-10
// suffit en général ; le bug se déclenche selon la combinaison des données,
// pas selon une valeur unique.
//
// Schémas inspectés : `main` uniquement. Pour inspecter une DB attachée,
// utiliser ATTACH puis appeler ProbeARTDivergencesForSchema (à implémenter
// si besoin).
//
// Retourne nil si aucune table avec PK VARCHAR n'est trouvée.
func ProbeARTDivergences(ctx context.Context, db *sql.DB, sampleSize int) (*ARTProbeReport, error) {
	start := time.Now()
	report := &ARTProbeReport{}

	if sampleSize <= 0 {
		sampleSize = 5
	}

	tables, err := listVarcharPKTables(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("list tables: %w", err)
	}

	for _, tbl := range tables {
		report.TablesScanned++
		divergences, samples, err := probeTable(ctx, db, tbl, sampleSize)
		if err != nil {
			// Non-bloquant : on log et on continue pour ne pas masquer
			// les divergences sur les autres tables.
			slog.WarnContext(ctx, "art_probe: scan échoué pour table — skip",
				"table", tbl.name,
				"err", err)
			continue
		}
		report.SamplesTested += samples
		report.Divergences = append(report.Divergences, divergences...)
	}

	report.Duration = time.Since(start)
	return report, nil
}

// tableInfo : métadonnées sur une table candidate au probe.
type tableInfo struct {
	name     string // nom de la table
	pkColumn string // première colonne PK (typée VARCHAR)
}

// listVarcharPKTables énumère les tables du schéma main qui ont au moins
// une colonne PK de type VARCHAR. Pour les PK composites, on prend la
// première colonne (suffit pour démasquer la corruption — la racine ART
// est partagée).
func listVarcharPKTables(ctx context.Context, db *sql.DB) ([]tableInfo, error) {
	// information_schema.key_column_usage liste les colonnes PK ordonnées.
	// On joint avec information_schema.columns pour filtrer par type.
	rows, err := db.QueryContext(ctx, `
		SELECT
			kcu.table_name,
			kcu.column_name
		FROM information_schema.key_column_usage kcu
		JOIN information_schema.table_constraints tc
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_name = kcu.table_name
		JOIN information_schema.columns c
			ON c.table_name = kcu.table_name
			AND c.column_name = kcu.column_name
		WHERE tc.constraint_type = 'PRIMARY KEY'
			AND tc.table_schema = 'main'
			AND c.data_type = 'VARCHAR'
			AND kcu.ordinal_position = 1
		ORDER BY kcu.table_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []tableInfo
	for rows.Next() {
		var t tableInfo
		if err := rows.Scan(&t.name, &t.pkColumn); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// probeTable teste sampleSize valeurs PK et reporte les divergences.
// Retourne (divergences, samplesTested, err).
func probeTable(ctx context.Context, db *sql.DB, tbl tableInfo, sampleSize int) ([]ARTDivergence, int, error) {
	// 1. Échantillonner sampleSize valeurs PK distinctes.
	//
	// On utilise ORDER BY random() pour varier les valeurs testées entre runs.
	// LIMIT sampleSize. Le SELECT lui-même utilise filter pushdown sur la PK
	// mais le COUNT global (sans WHERE) ne dépend pas du bug ART.
	sampleQuery := fmt.Sprintf(
		`SELECT %s FROM %s WHERE %s IS NOT NULL ORDER BY random() LIMIT ?`,
		quoteIdent(tbl.pkColumn), quoteIdent(tbl.name), quoteIdent(tbl.pkColumn))
	rows, err := db.QueryContext(ctx, sampleQuery, sampleSize)
	if err != nil {
		return nil, 0, fmt.Errorf("sample: %w", err)
	}
	var samples []string
	for rows.Next() {
		var v sql.NullString
		if err := rows.Scan(&v); err != nil {
			rows.Close()
			return nil, 0, fmt.Errorf("scan sample: %w", err)
		}
		if v.Valid {
			samples = append(samples, v.String)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	// 2. Pour chaque valeur, comparer count via index vs scan.
	var divergences []ARTDivergence
	for _, val := range samples {
		var indexed, scan int
		indexedQuery := fmt.Sprintf(
			`SELECT COUNT(*) FROM %s WHERE %s = ?`,
			quoteIdent(tbl.name), quoteIdent(tbl.pkColumn))
		if err := db.QueryRowContext(ctx, indexedQuery, val).Scan(&indexed); err != nil {
			return nil, len(samples), fmt.Errorf("count indexed (%s=%q): %w", tbl.pkColumn, val, err)
		}
		scanQuery := fmt.Sprintf(
			`SELECT COUNT(*) FROM %s WHERE %s || '' = ?`,
			quoteIdent(tbl.name), quoteIdent(tbl.pkColumn))
		if err := db.QueryRowContext(ctx, scanQuery, val).Scan(&scan); err != nil {
			return nil, len(samples), fmt.Errorf("count scan (%s=%q): %w", tbl.pkColumn, val, err)
		}
		if indexed != scan {
			divergences = append(divergences, ARTDivergence{
				Table:        tbl.name,
				PKColumn:     tbl.pkColumn,
				SampleValue:  val,
				CountIndexed: indexed,
				CountScan:    scan,
			})
		}
	}
	return divergences, len(samples), nil
}

// quoteIdent quote un identifiant SQL en remplaçant " par "" (échappement
// DuckDB standard). Protection contre injection sur les noms de tables/colonnes
// lus depuis information_schema — théoriquement safe mais ceinture+bretelles.
func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// BootARTGuard est un filet de garde lancé au boot du serveur sur chacune des
// DBs critiques (shared_matches_v2, shared_social, metadata). Combine :
//   - ProbeARTDivergences (détection)
//   - slog WARN structuré par divergence
//   - métrique expvar `art_corruption_detected_<dbLabel>_<table>`
//   - métrique expvar `art_probe_runs_total_<dbLabel>` (pour visibilité boot)
//
// dbLabel : étiquette courte de la DB ("shared", "social", "metadata", ou
// "player_<gamertag>") — sert pour les clés expvar et les logs.
//
// Non-bloquant : retourne le rapport mais n'échoue jamais le boot. Si le probe
// lui-même erre (ex: information_schema indisponible), log ERROR + métrique
// `art_probe_errors_total_<dbLabel>` mais le serveur continue.
func BootARTGuard(ctx context.Context, db *sql.DB, dbLabel string, sampleSize int) *ARTProbeReport {
	observability.IncCounter("art_probe_runs_total_" + dbLabel)

	report, err := ProbeARTDivergences(ctx, db, sampleSize)
	if err != nil {
		slog.ErrorContext(ctx, "art_guard: probe échoué",
			"db", dbLabel,
			"err", err,
		)
		observability.IncCounter("art_probe_errors_total_" + dbLabel)
		return nil
	}

	if !report.HasDivergence() {
		slog.InfoContext(ctx, "art_guard: aucune corruption ART détectée",
			"db", dbLabel,
			"tables_scanned", report.TablesScanned,
			"samples_tested", report.SamplesTested,
			"duration_ms", report.Duration.Milliseconds(),
		)
		return report
	}

	// Divergences détectées : log un WARN par divergence pour traçabilité
	// fine + un WARN agrégé pour les dashboards.
	for _, d := range report.Divergences {
		slog.WarnContext(ctx, "art_guard: corruption ART détectée — rebuild table requis",
			"db", dbLabel,
			"table", d.Table,
			"pk_column", d.PKColumn,
			"sample_value", d.SampleValue,
			"count_indexed", d.CountIndexed,
			"count_scan", d.CountScan,
			"missing_rows", d.CountScan-d.CountIndexed,
		)
		// Une métrique par (db, table) pour permettre les alertes ciblées.
		observability.IncCounter(fmt.Sprintf("art_corruption_detected_%s_%s",
			dbLabel, sanitizeMetricKey(d.Table)))
	}
	slog.WarnContext(ctx, "art_guard: corruption ART agrégée",
		"db", dbLabel,
		"divergences_count", len(report.Divergences),
		"tables_scanned", report.TablesScanned,
		"samples_tested", report.SamplesTested,
		"duration_ms", report.Duration.Milliseconds(),
	)
	return report
}

// sanitizeMetricKey nettoie un identifiant pour usage dans une clé expvar
// (alphanumérique + underscore uniquement).
func sanitizeMetricKey(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}
