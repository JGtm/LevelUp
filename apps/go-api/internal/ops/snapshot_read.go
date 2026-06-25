// Package ops — snapshot_read.go : couche de LECTURE des snapshots immuables (Phase 3
// du PLAN_DURABILITE_SNAPSHOT_IMMUABLE).
//
// OpenSnapshotForPlayer ouvre la version COURANTE du snapshot d'un titre comme une base
// DuckDB :memory: dont les VUES read_parquet portent les MÊMES noms que les tables/vues
// des DB live (faits shared globaux + dérivés `_latest` du joueur). Conséquence : une
// requête de lecture existante tourne dessus SANS réécriture — et peut même joindre
// shared + dérivés en une seule requête (impossible en live où ce sont 2 DB séparées).
//
// Lecture pure read_parquet sur des fichiers immuables → ZÉRO contact avec la base RW,
// donc jamais stallée par une fenêtre d'écriture du B-swap (c'est tout l'intérêt).
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

// OpenSnapshotForPlayer ouvre la version courante pour un joueur (faits shared globaux +
// dérivés `_latest` de CE joueur). Retourne ErrNoSnapshot si aucune version active.
func OpenSnapshotForPlayer(ctx context.Context, paths *title.PathResolver, titleSlug, gamertag string) (*SnapshotQuerier, error) {
	if paths == nil {
		return nil, fmt.Errorf("OpenSnapshotForPlayer: paths nil")
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

	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("snapshot read: open :memory:: %w", err)
	}
	// Faits shared (globaux) : vue au nom de la table live.
	for _, tbl := range sharedSnapshotTables {
		file := filepath.Join(versionDir, "shared", tbl+".parquet")
		if err := createSnapshotView(ctx, db, tbl, file); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	// Dérivés ancrés (CE joueur) : vue au nom de la vue live `<table>_latest`.
	safe := sanitizeSnapshotFilename(gamertag)
	for _, spec := range derivedSnapshotSpecs {
		file := filepath.Join(versionDir, "derived", spec.name, safe+".parquet")
		if err := createSnapshotView(ctx, db, spec.name+"_latest", file); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	return &SnapshotQuerier{DB: db, Version: version, closeFn: func() { _ = db.Close() }}, nil
}

// createSnapshotView crée une VUE read_parquet `viewName` sur `file`. Si le fichier est
// absent (table/joueur sans données ready), aucune vue n'est créée — une requête sur
// cette relation échouera proprement (Catalog Error), à charge du caller de dégrader.
func createSnapshotView(ctx context.Context, db *sql.DB, viewName, file string) error {
	if _, err := os.Stat(file); err != nil {
		return nil // fichier absent → pas de vue (dégradation au niveau requête)
	}
	return createParquetViewStrict(ctx, db, viewName, file)
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
	out = append(out, sharedSnapshotMatchKeyedRaw...) // weapon_kills, match_csrs (raw)
	out = append(out, sharedSnapshotGlobalTables...)  // xuid_aliases
	return out
}

// OpenSnapshotShared ouvre la version courante comme une DuckDB :memory: reconstruisant
// le SCHÉMA SHARED COMPLET — toutes les tables de base + TOUTES les vues aux noms live
// (v_gamertag_lookup, v_match_full, v_killer_victim_full, v_weapon_kills,
// match_csrs_latest, mv_player_matches) → un SharedReader peut servir TOUTES les lectures
// shared de l'app depuis le snapshot, hors fenêtre RW. Retourne ErrNoSnapshot (aucune
// version) ou ErrSnapshotIncomplete (table requise absente) → le caller dégrade vers live
// (jamais de schéma partiel servi : soit le schéma complet, soit le live).
//
// Zéro divergence : les vues sont créées par les MÊMES fonctions canoniques que le boot/
// migrations (migration.ApplyResolutionViews + ApplyMvPlayerMatchesView, qui réutilisent
// analysis.GamertagLookupViewSQL — source unique du chokepoint gamertag). Seules
// v_weapon_kills / match_csrs_latest sont inlinées (DDL court, gardé par le test de
// fidélité).
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
	if err := migration.ApplyResolutionViews(db); err != nil { // v_gamertag_lookup, v_match_full, v_killer_victim_full
		_ = db.Close()
		return nil, fmt.Errorf("snapshot read shared: resolution views: %w", err)
	}
	if err := migration.ApplyMvPlayerMatchesView(db); err != nil { // mv_player_matches
		_ = db.Close()
		return nil, fmt.Errorf("snapshot read shared: mv_player_matches: %w", err)
	}
	// v_weapon_kills (DENSE_RANK) + match_csrs_latest (QUALIFY) : DDL inline aligné sur
	// migration/steps_shared_append_only_weapon_kills.go et sync/schema.go.
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
	} {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("snapshot read shared: vue append-only: %w", err)
		}
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
