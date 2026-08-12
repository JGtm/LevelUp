// Package ops — snapshot_read.go : couche de LECTURE des snapshots immuables.
//
// OpenSnapshotShared ouvre la version COURANTE du snapshot d'un titre comme une base
// DuckDB :memory: reconstruisant le SCHÉMA SHARED COMPLET (tables de base matérialisées
// + toutes les vues aux noms live). Conséquence : une requête de lecture shared existante
// tourne dessus SANS réécriture, sur des fichiers Parquet immuables → ZÉRO contact avec
// la base RW, donc jamais stallée par une fenêtre d'écriture du B-swap (c'est l'intérêt).
// Utilisé par sync.SnapshotPreferredSharedReader (scoped MatchView), avec fallback live.
package ops

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
)

// ErrNoSnapshot : aucune version de snapshot active pour ce titre (CURRENT.json absent /
// version 0). Le caller doit dégrader vers la lecture live.
var ErrNoSnapshot = errors.New("snapshot: aucune version active")

// ErrSnapshotIncomplete : la version active n'a pas toutes les relations requises pour
// reconstruire le schéma shared complet (fixture partielle / titre sans certaines
// tables). Le caller doit dégrader vers la lecture live (jamais servir un schéma partiel).
var ErrSnapshotIncomplete = errors.New("snapshot: schéma shared incomplet")

// CurrentSnapshotVersion retourne le numéro de version active d'un titre (0 si aucune).
// Lecture légère de CURRENT.json — sert au cache versionné d'un SharedReader snapshot.
func CurrentSnapshotVersion(paths *title.PathResolver, titleSlug string) (int64, error) {
	if paths == nil {
		return 0, fmt.Errorf("CurrentSnapshotVersion: paths nil")
	}
	if titleSlug == "" {
		titleSlug = title.DefaultSlug
	}
	return readCurrent(paths.SnapshotCurrentManifestPath(titleSlug))
}

// SnapshotQuerier expose la version courante du snapshot comme une DuckDB :memory'
// interrogeable. Fermer après usage (libère le handle :memory:).
type SnapshotQuerier struct {
	DB      *sql.DB
	Version int64
	closeFn func()
}

// Close libère le handle :memory: du snapshot.
func (q *SnapshotQuerier) Close() {
	if q != nil && q.closeFn != nil {
		q.closeFn()
	}
}

// createParquetViewStrict crée la vue read_parquet et ERRE si le fichier est absent
// (relation REQUISE). Utilisé par OpenSnapshotShared où un schéma partiel = fallback live.
func createParquetViewStrict(ctx context.Context, db *sql.DB, viewName, file string) error {
	if _, err := os.Stat(file); err != nil {
		return fmt.Errorf("%w: %s", ErrSnapshotIncomplete, viewName)
	}
	stmt := fmt.Sprintf("CREATE VIEW %s AS SELECT * FROM read_parquet(%s)",
		viewName, sqlQuote(filepath.ToSlash(file)))
	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("snapshot read: vue %s: %w", viewName, err)
	}
	return nil
}

// sharedSnapshotRequiredTables : tables de base REQUISES pour reconstruire le schéma
// shared COMPLET (sinon ErrSnapshotIncomplete → fallback live global). Matérialisées en
// TABLES :memory: (pas vues) pour que les fonctions canoniques de création de vues
// (CREATE TABLE IF NOT EXISTS + CREATE VIEW) s'appliquent sans conflit de nom.
func sharedSnapshotRequiredTables() []string {
	out := append([]string{}, sharedSnapshotTables...)
	out = append(out, sharedSnapshotMatchKeyedRaw...) // weapon_kills, match_csrs, match_objective_stats (raw)
	out = append(out, sharedSnapshotGlobalTables...)  // xuid_aliases
	return out
}

// OpenSnapshotShared ouvre la version courante comme une DuckDB :memory: reconstruisant
// le SCHÉMA SHARED COMPLET — toutes les tables de base + TOUTES les vues aux noms live
// (v_gamertag_lookup, v_match_full, v_weapon_kills, match_kill_events_latest,
// match_csrs_latest, match_objective_stats_latest, mv_player_matches) → un SharedReader
// peut servir TOUTES les lectures shared de l'app depuis le snapshot, hors fenêtre RW.
// Retourne ErrNoSnapshot (aucune version) ou ErrSnapshotIncomplete (table requise
// absente) → le caller dégrade vers live (jamais de schéma partiel servi : soit le
// schéma complet, soit le live).
//
// Zéro divergence : les vues sont créées par les MÊMES fonctions canoniques que le boot/
// migrations (migration.ApplyResolutionViews + ApplyMvPlayerMatchesView, qui réutilisent
// analysis.GamertagLookupViewSQL — source unique du chokepoint gamertag ;
// migration.MatchObjectiveStatsLatestViewSQL pour match_objective_stats_latest). Seules
// v_weapon_kills / match_csrs_latest sont inlinées (DDL court, gardé par le test de
// fidélité). Le test de contrat TestOpenSnapshotShared_SchemaContract_integration liste
// les relations que les requêtes MatchView (Q12/Q13...) attendent de ce schéma.
func OpenSnapshotShared(ctx context.Context, paths *title.PathResolver, titleSlug string) (*SnapshotQuerier, error) {
	if paths == nil {
		return nil, fmt.Errorf("OpenSnapshotShared: paths nil")
	}
	if titleSlug == "" {
		titleSlug = title.DefaultSlug
	}
	version, err := readCurrent(paths.SnapshotCurrentManifestPath(titleSlug))
	if err != nil {
		return nil, err
	}
	if version == 0 {
		return nil, ErrNoSnapshot
	}
	versionDir := paths.SnapshotVersionDir(titleSlug, version)
	sharedFile := func(name string) string { return filepath.Join(versionDir, "shared", name+".parquet") }

	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("snapshot read shared: open :memory:: %w", err)
	}
	// Matérialise les tables de base (REQUISES) depuis les Parquet.
	for _, tbl := range sharedSnapshotRequiredTables() {
		if err := materializeParquetTable(ctx, db, tbl, sharedFile(tbl)); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	// Recrée TOUTES les vues shared via les fonctions canoniques (zéro divergence).
	if err := migration.ApplyResolutionViews(db); err != nil { // v_gamertag_lookup, v_match_full, killer_victim_pairs (compat)
		_ = db.Close()
		return nil, fmt.Errorf("snapshot read shared: resolution views: %w", err)
	}
	if err := migration.ApplyMvPlayerMatchesView(db); err != nil { // mv_player_matches
		_ = db.Close()
		return nil, fmt.Errorf("snapshot read shared: mv_player_matches: %w", err)
	}
	// v_weapon_kills (DENSE_RANK) + match_csrs_latest (QUALIFY) : DDL inline aligné sur
	// migration/steps_shared_append_only_weapon_kills.go et sync/schema.go.
	// match_objective_stats_latest : DDL canonique partagé (Q12 la LEFT JOIN depuis v7.2 —
	// son absence du schéma snapshot rendait TOUT match du cut scoreboard_empty).
	for _, ddl := range []string{
		`CREATE OR REPLACE VIEW v_weapon_kills AS
			SELECT * EXCLUDE (rk) FROM (
				SELECT *, COALESCE(reconciled_as, weapon_id) AS effective_weapon_id,
				       DENSE_RANK() OVER (PARTITION BY match_id, xuid ORDER BY generation_id DESC) AS rk
				FROM weapon_kills
			) WHERE rk = 1`,
		`CREATE OR REPLACE VIEW match_csrs_latest AS
			SELECT * FROM match_csrs
			QUALIFY ROW_NUMBER() OVER (PARTITION BY match_id, xuid ORDER BY written_at DESC, id DESC) = 1`,
		migration.MatchObjectiveStatsLatestViewSQL("match_objective_stats"),
	} {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("snapshot read shared: vue append-only: %w", err)
		}
	}
	// match_kill_events_latest : par la fonction canonique de la migration (zéro
	// divergence — c'est elle qui porte la sémantique de sélection de passe). Sur le
	// :memory:, la table matérialisée depuis le Parquet est déjà là : les CREATE ... IF
	// NOT EXISTS sont des no-ops, seule la vue (OR REPLACE) est posée. Ajoutée le
	// 2026-08-12 : Q21b/Q21c (arme + assistant du kill feed) la lisent depuis le lot
	// portage POC — sans elle, MatchView servi du snapshot rendait ces lectures vides.
	if err := migration.EnsureMatchKillEvents(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("snapshot read shared: match_kill_events_latest: %w", err)
	}
	return &SnapshotQuerier{DB: db, Version: version, closeFn: func() { _ = db.Close() }}, nil
}

// materializeParquetTable crée une TABLE :memory: `name` matérialisant le Parquet `file`.
// Erre ErrSnapshotIncomplete si le fichier est absent (table requise → fallback live).
func materializeParquetTable(ctx context.Context, db *sql.DB, name, file string) error {
	if _, err := os.Stat(file); err != nil {
		return fmt.Errorf("%w: %s", ErrSnapshotIncomplete, name)
	}
	stmt := fmt.Sprintf("CREATE TABLE %s AS SELECT * FROM read_parquet(%s)", name, sqlQuote(filepath.ToSlash(file)))
	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("snapshot read shared: matérialise %s: %w", name, err)
	}
	return nil
}
