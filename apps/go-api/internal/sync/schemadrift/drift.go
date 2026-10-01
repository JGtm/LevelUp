// Package schemadrift — LE SOIN DE SCHÉMA DEVIENT BRUYANT (2026-08-05).
//
// POURQUOI. sync.EnsurePlayerSchema soigne les player DB à chaque OpenPlayerDB
// (ADD COLUMN / CREATE ... IF NOT EXISTS). Ce soin était SILENCIEUX : pendant des mois il
// a masqué la dérive de la chaîne de migrations (l'AUTORITÉ de schéma des player DB),
// jusqu'au bug prod career_progression du 2026-08-05 — invisible en prod (DB re-soignées
// à chaque boot), fatal sur toute DB FRAÎCHE ou reconstruite.
//
// CE QUE FAIT CE PACKAGE. Il photographie le schéma AVANT et APRÈS le soin et journalise
// en WARN (`schema_drift_healed`) chaque objet RÉELLEMENT créé, et chaque index RÉELLEMENT
// retiré (lot B-C5, 2026-09-27) — jamais les no-ops. Une
// player DB à jour n'émet donc RIEN ; une DB en retard sur la chaîne de migrations se
// signale bruyamment à chaque ouverture, avec l'objet exact qui manquait.
//
// CE QUE CE PACKAGE NE FAIT PAS. Il ne RETIRE pas le soin : les player DB legacy en
// dépendent encore. Retirer le soin exigera une CAMPAGNE DE CONVERGENCE préalable (faire
// tourner la chaîne complète sur toutes les player DB existantes, puis constater zéro
// `schema_drift_healed` sur une fenêtre d'observation). Les WARN émis ici sont
// précisément l'instrument de mesure de cette campagne.
//
// EMPLACEMENT : sous-package cohésif plutôt qu'un fichier de plus à la racine du
// god-package internal/sync/ (ratchet archlint.TestSyncRootPackageFrozen, ADR 0027).
package schemadrift

import (
	"context"
	"database/sql"
	"log/slog"
)

// Object identifie un objet de schéma observable dans une DB DuckDB.
// Table est la table porteuse (vide pour une séquence) ; Name est le nom de l'objet,
// ou le nom de la colonne quand Kind vaut "column".
type Object struct {
	Kind  string
	Table string
	Name  string
}

// catalogQueries — introspection du catalogue DuckDB, une requête par famille d'objets.
// Chaque requête retourne (kind, table, name) dans cet ordre.
var catalogQueries = []string{
	`SELECT 'table', table_name, table_name FROM duckdb_tables() WHERE schema_name = 'main'`,
	`SELECT 'view', view_name, view_name FROM duckdb_views() WHERE schema_name = 'main' AND NOT internal`,
	`SELECT 'index', table_name, index_name FROM duckdb_indexes() WHERE schema_name = 'main'`,
	`SELECT 'sequence', '', sequence_name FROM duckdb_sequences() WHERE schema_name = 'main'`,
	`SELECT 'column', table_name, column_name FROM duckdb_columns() WHERE schema_name = 'main'`,
}

// Snapshot photographie les objets de schéma de la DB. Retourne nil si l'introspection
// échoue — la détection de dérive est alors DÉSACTIVÉE pour cet appel (jamais bloquante :
// le soin de schéma doit rester possible), mais l'échec est JOURNALISÉ, jamais avalé.
func Snapshot(ctx context.Context, db *sql.DB) map[Object]struct{} {
	out := make(map[Object]struct{}, 256)
	for _, q := range catalogQueries {
		if err := collect(ctx, db, q, out); err != nil {
			slog.ErrorContext(ctx,
				"schemadrift.Snapshot: introspection du catalogue échouée — détection de dérive désactivée pour ce boot",
				"err", err, "query", q)
			return nil
		}
	}
	return out
}

// collect exécute une requête d'introspection et accumule ses lignes.
func collect(ctx context.Context, db *sql.DB, query string, out map[Object]struct{}) error {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var obj Object
		if err := rows.Scan(&obj.Kind, &obj.Table, &obj.Name); err != nil {
			return err
		}
		out[obj] = struct{}{}
	}
	return rows.Err()
}

// Report compare l'état APRÈS le soin à l'état `before` et journalise en WARN
// `schema_drift_healed` chaque objet réellement créé (action=created) et chaque index
// réellement retiré (action=dropped). No-op si `before` est nil
// (introspection indisponible, déjà journalisée) ou si rien n'a changé — cas NOMINAL
// d'une DB à jour. `healedBy` nomme le soin responsable (ex. "sync.EnsurePlayerSchema").
//
// Les colonnes d'une table CRÉÉE par le soin ne sont pas énumérées : l'événement utile
// est la table, pas ses 15 colonnes. Seule une colonne ajoutée à une table PRÉ-EXISTANTE
// est une dérive de colonne.
func Report(ctx context.Context, db *sql.DB, before map[Object]struct{}, healedBy string) {
	if before == nil {
		return
	}
	after := Snapshot(ctx, db)
	if after == nil {
		return
	}
	dbPath := currentDatabasePath(ctx, db)
	for obj := range after {
		if _, existed := before[obj]; existed {
			continue
		}
		if obj.Kind == "column" && !tableExistedBefore(before, obj.Table) {
			continue // table entièrement créée : déjà signalée comme telle
		}
		warnHealed(ctx, obj, "created", dbPath, healedBy)
	}
	reportRetiredIndexes(ctx, before, after, dbPath, healedBy)
}

// reportRetiredIndexes journalise chaque index présent AVANT le soin et absent APRÈS : le
// soin retire les index ART retirés (MSR, PSA — convergence D-4 du plan backlog 2026-09-26)
// qu'un binaire plus ancien aurait recréés. Ce retrait est une action réelle : il se
// journalise comme une création (lot B-C5), jamais sur une base à jour. Seuls les INDEX
// sont comparés dans ce sens : c'est la suppression que porte le DDL de soin (DROP INDEX IF
// EXISTS). Une conversion append-only d'une table legacy, qui la reconstruit, y apparaît
// aussi, à raison : c'est une action réelle.
func reportRetiredIndexes(ctx context.Context, before, after map[Object]struct{}, dbPath, healedBy string) {
	for obj := range before {
		if obj.Kind != "index" {
			continue
		}
		if _, stillThere := after[obj]; stillThere {
			continue
		}
		warnHealed(ctx, obj, "dropped", dbPath, healedBy)
	}
}

// warnHealed émet le WARN schema_drift_healed d'un objet créé ou retiré par le soin.
func warnHealed(ctx context.Context, obj Object, action, dbPath, healedBy string) {
	slog.WarnContext(ctx, "schema_drift_healed",
		"action", action,
		"object_kind", obj.Kind,
		"table", obj.Table,
		"object", obj.Name,
		"db", dbPath,
		"authority", "migrations",
		"healed_by", healedBy)
}

// tableExistedBefore indique si la table portait déjà des objets avant le soin.
func tableExistedBefore(before map[Object]struct{}, table string) bool {
	_, ok := before[Object{Kind: "table", Table: table, Name: table}]
	return ok
}

// currentDatabasePath retourne le chemin du fichier DuckDB attaché (vide pour :memory:),
// pour identifier la DB soignée dans les logs. Best-effort : un échec dégrade la clé `db`
// en chaîne vide APRÈS journalisation, jamais un échec de boot.
func currentDatabasePath(ctx context.Context, db *sql.DB) string {
	var path string
	err := db.QueryRowContext(ctx,
		`SELECT COALESCE(path, '') FROM duckdb_databases() WHERE database_name = current_database()`).Scan(&path)
	if err != nil {
		slog.DebugContext(ctx, "schemadrift: chemin de la DB indisponible", "err", err)
		return ""
	}
	return path
}
