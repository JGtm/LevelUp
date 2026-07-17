package prestigetuning

import (
	"context"
	"database/sql"
	"fmt"
)

// collect.go — accès LECTURE SEULE à une player DB (stats.duckdb). Deux requêtes
// d'agrégation GROUP BY (aucune écriture) :
//   - metricWindowQuery : jointure prestige_telemetry ⋈ challenge → comptes par
//     (source, métrique, fenêtre). Attribue created/completed/expired/abandoned à
//     la métrique du défi.
//   - acceptanceQuery : agrégation sans jointure → created/rejected par source,
//     pour l'acceptance (les rejets ne sont pas persistés dans challenge).
//
// Le *sql.DB doit provenir de duckdb.OpenReadForQuery (modèle mono-process :
// jamais d'ouverture RW ici).

// metricWindowQueryTmpl : %s = expression source (pt.source ou littéral 'unknown'
// pour une DB legacy sans la colonne source, cf. migration prestige_add_source_columns_v1).
const metricWindowQueryTmpl = `
	SELECT
		%s                                                               AS src,
		c.metric                                                         AS metric,
		c.window_type                                                    AS window_type,
		COALESCE(c.window_value, '')                                     AS window_value,
		SUM(CASE WHEN pt.event_type = 'created'   THEN 1 ELSE 0 END)     AS created,
		SUM(CASE WHEN pt.event_type = 'completed' THEN 1 ELSE 0 END)     AS completed,
		SUM(CASE WHEN pt.event_type = 'expired'   THEN 1 ELSE 0 END)     AS expired,
		SUM(CASE WHEN pt.event_type = 'abandoned' THEN 1 ELSE 0 END)     AS abandoned
	FROM prestige_telemetry pt
	JOIN challenge c ON pt.challenge_id = c.id
	GROUP BY src, c.metric, c.window_type, COALESCE(c.window_value, '')
	ORDER BY src, metric, window_type, window_value`

// acceptanceQueryTmpl : %s = expression source (source ou littéral 'unknown').
const acceptanceQueryTmpl = `
	SELECT
		%s                                                               AS src,
		SUM(CASE WHEN event_type = 'created'    THEN 1 ELSE 0 END)        AS created,
		SUM(CASE WHEN event_type LIKE 'rejected%%' THEN 1 ELSE 0 END)     AS rejected
	FROM prestige_telemetry
	GROUP BY src
	ORDER BY src`

// CollectFromDB agrège la télémétrie d'une player DB. Best-effort et tolérant aux
// DBs legacy : si la table prestige_telemetry est absente, retourne un résultat
// vide sans erreur (le joueur n'a simplement pas de télémétrie). Si la colonne
// source est absente (DB antérieure à la migration prestige_add_source_columns_v1),
// les événements sont agrégés sous "unknown" plutôt que de perdre le joueur. Une
// vraie erreur de requête (autre cause) remonte pour être tracée (jamais avalée).
func CollectFromDB(ctx context.Context, db *sql.DB) ([]MetricWindowCount, []SourceAcceptance, error) {
	present, hasSource, err := probeTelemetry(ctx, db)
	if err != nil {
		return nil, nil, fmt.Errorf("probe prestige_telemetry schema: %w", err)
	}
	if !present {
		return nil, nil, nil // DB legacy sans télémétrie : rien à agréger.
	}

	srcJoined := "'unknown'"
	srcFlat := "'unknown'"
	if hasSource {
		srcJoined = "COALESCE(NULLIF(pt.source, ''), 'unknown')"
		srcFlat = "COALESCE(NULLIF(source, ''), 'unknown')"
	}

	counts, err := queryMetricWindows(ctx, db, fmt.Sprintf(metricWindowQueryTmpl, srcJoined))
	if err != nil {
		return nil, nil, fmt.Errorf("metric/window aggregation: %w", err)
	}
	accept, err := queryAcceptance(ctx, db, fmt.Sprintf(acceptanceQueryTmpl, srcFlat))
	if err != nil {
		return nil, nil, fmt.Errorf("acceptance aggregation: %w", err)
	}
	return counts, accept, nil
}

// probeTelemetry inspecte information_schema : la table prestige_telemetry
// existe-t-elle, et porte-t-elle la colonne source ?
func probeTelemetry(ctx context.Context, db *sql.DB) (present, hasSource bool, err error) {
	rows, err := db.QueryContext(ctx,
		`SELECT column_name FROM information_schema.columns
		 WHERE table_name = 'prestige_telemetry'`)
	if err != nil {
		return false, false, err
	}
	defer rows.Close()
	for rows.Next() {
		var col string
		if err := rows.Scan(&col); err != nil {
			return false, false, err
		}
		present = true
		if col == "source" {
			hasSource = true
		}
	}
	return present, hasSource, rows.Err()
}

func queryMetricWindows(ctx context.Context, db *sql.DB, query string) ([]MetricWindowCount, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []MetricWindowCount
	for rows.Next() {
		var c MetricWindowCount
		if err := rows.Scan(&c.Source, &c.Metric, &c.WindowType, &c.WindowValue,
			&c.Created, &c.Completed, &c.Expired, &c.Abandoned); err != nil {
			return nil, fmt.Errorf("scan metric/window row: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func queryAcceptance(ctx context.Context, db *sql.DB, query string) ([]SourceAcceptance, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SourceAcceptance
	for rows.Next() {
		var a SourceAcceptance
		if err := rows.Scan(&a.Source, &a.Created, &a.Rejected); err != nil {
			return nil, fmt.Errorf("scan acceptance row: %w", err)
		}
		a.AcceptanceRate = rate(a.Created, a.Created+a.Rejected)
		out = append(out, a)
	}
	return out, rows.Err()
}
