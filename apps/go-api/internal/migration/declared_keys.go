package migration

// declared_keys.go — le schéma de RÉFÉRENCE d'un cycle de migrations : ce que le même jeu
// d'étapes produit sur une base vierge, en mémoire. Il dit, table par table, quelles clés
// (PRIMARY KEY, UNIQUE) le schéma canonique déclare et quels défauts portent ses colonnes. La
// réparation des clés perdues (declared_keys_repair.go) le compare à la base réelle.
//
// Une seule construction par jeu d'étapes et par processus (cache) : au démarrage du serveur,
// toutes les bases joueur d'un titre partagent la même référence.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"levelup/go-api/pkg/duckdbkeys"
)

// referenceTable : clés et colonnes d'une table du schéma de référence.
type referenceTable struct {
	keys []duckdbkeys.Key
	cols map[string]duckdbkeys.Column
}

// keyReference : tables du schéma de référence, par nom.
type keyReference struct {
	tables map[string]referenceTable
}

// referenceCache : une référence (ou l'échec de sa construction) par signature de cycle.
var referenceCache = struct {
	sync.Mutex
	byCycle map[string]referenceResult
}{byCycle: map[string]referenceResult{}}

type referenceResult struct {
	ref *keyReference
	err error
}

// referenceTrusted : la référence ne vaut que si le cycle porte le jeu COMPLET d'étapes du
// titre. Un titre qui possède son jeu pour cette cible l'a par construction ; sinon le jeu par
// défaut n'est complet qu'avec les étapes title-owned (provider posé). Sans elles (outil CLI
// qui n'a pas câblé titleseams), la référence ignorerait des étapes qui ont façonné la base —
// par exemple une reconstruction SANS clé voulue contre le bug ART — et la « réparation »
// reposerait une clé retirée à dessein.
func referenceTrusted(slug string, target TargetDB) bool {
	if set, ok := migrationSetFor(slug); ok && set.ownsTarget(target) {
		return true
	}
	return titleStepsProvider != nil
}

// cycleSignature identifie un jeu d'étapes : titre, cible et noms des étapes dans l'ordre.
func cycleSignature(slug string, target TargetDB, steps []Migration) string {
	names := make([]string, len(steps))
	for i := range steps {
		names[i] = steps[i].Name
	}
	return slug + "|" + string(target) + "|" + strings.Join(names, ",")
}

// referenceFor rend la référence du cycle, construite au premier appel puis lue du cache.
// steps doit être déjà trié dans l'ordre canonique (celui que runSteps vient d'appliquer).
func referenceFor(ctx context.Context, slug string, target TargetDB, steps []Migration) (*keyReference, error) {
	sig := cycleSignature(slug, target, steps)
	referenceCache.Lock()
	defer referenceCache.Unlock()
	if r, ok := referenceCache.byCycle[sig]; ok {
		return r.ref, r.err
	}
	start := time.Now()
	ref, err := buildReference(ctx, slug, steps)
	referenceCache.byCycle[sig] = referenceResult{ref: ref, err: err}
	if err == nil {
		slog.DebugContext(ctx, "migration: schéma de référence construit",
			"title", slug, "target", target, "tables", len(ref.tables),
			"duration_ms", time.Since(start).Milliseconds())
	}
	return ref, err
}

// buildReference applique les étapes à une base vierge en mémoire (DDL et backfills hors API,
// sans journal du runner) et relève clés et colonnes de chaque table.
func buildReference(ctx context.Context, slug string, steps []Migration) (*keyReference, error) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("référence: ouverture: %w", err)
	}
	defer func() { _ = db.Close() }()
	if err := ensureMigrationTable(ctx, db); err != nil {
		return nil, fmt.Errorf("référence: registre: %w", err)
	}
	if err := ensureTitleSchemaVersionTable(ctx, db); err != nil {
		return nil, fmt.Errorf("référence: version: %w", err)
	}
	for i := range steps {
		m := steps[i]
		if !m.appliesToTitle(slug) {
			continue
		}
		if err := m.ApplySchema(db); err != nil {
			return nil, fmt.Errorf("référence: étape %s: %w", m.Name, err)
		}
		if _, err := db.ExecContext(ctx,
			`INSERT INTO schema_migrations (title_slug, name, description, schema_done, backfill_done)
			 VALUES (?, ?, ?, TRUE, TRUE)`, slug, m.Name, m.Description); err != nil {
			return nil, fmt.Errorf("référence: registre %s: %w", m.Name, err)
		}
		if m.ApplyBackfill != nil && !m.RequiresAPI {
			if err := m.ApplyBackfill(db); err != nil {
				slog.DebugContext(ctx, "migration: backfill de la référence en échec (sans effet sur le schéma)",
					"name", m.Name, "err", err)
			}
		}
	}
	return readReference(ctx, db)
}

// readReference relève les tables de base de db avec leurs clés et leurs colonnes.
func readReference(ctx context.Context, db *sql.DB) (*keyReference, error) {
	names, err := mainBaseTables(ctx, db)
	if err != nil {
		return nil, err
	}
	ref := &keyReference{tables: make(map[string]referenceTable, len(names))}
	for _, name := range names {
		keys, err := duckdbkeys.TableKeys(ctx, db, name)
		if err != nil {
			return nil, err
		}
		cols, err := duckdbkeys.TableColumns(ctx, db, name)
		if err != nil {
			return nil, err
		}
		byName := make(map[string]duckdbkeys.Column, len(cols))
		for _, c := range cols {
			byName[c.Name] = c
		}
		ref.tables[name] = referenceTable{keys: keys, cols: byName}
	}
	return ref, nil
}

// mainBaseTables liste les tables de base (ni vues, ni temporaires) du schéma main de la base
// courante.
func mainBaseTables(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT table_name FROM duckdb_tables()
		WHERE database_name = current_database() AND schema_name = 'main' AND NOT temporary
		ORDER BY table_name`)
	if err != nil {
		return nil, fmt.Errorf("tables de la base: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("tables de la base: lecture: %w", err)
		}
		out = append(out, name)
	}
	return out, rows.Err()
}
