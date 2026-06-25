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

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain/title"
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

// OpenSnapshotShared ouvre la version courante comme une DuckDB :memory: reconstruisant
// le SCHÉMA SHARED COMPLET (tables de base + tables globales + vues dérivées aux noms
// live : v_gamertag_lookup, v_match_full, v_killer_victim_full, v_weapon_kills,
// match_csrs_latest) → un SharedReader peut servir TOUTES les lectures shared depuis le
// snapshot, hors fenêtre RW. Retourne ErrNoSnapshot (aucune version) ou
// ErrSnapshotIncomplete (relation requise absente) → le caller dégrade vers live.
//
// Zéro per-joueur (shared = global). Réutilise analysis.GamertagLookupViewSQL (source
// unique du chokepoint gamertag — jamais de définition divergente).
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
	// Tables de base + globales : REQUISES (vue passthrough au nom de la table).
	required := append(append([]string{}, sharedSnapshotTables...), sharedSnapshotGlobalTables...)
	for _, tbl := range required {
		if err := createParquetViewStrict(ctx, db, tbl, sharedFile(tbl)); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	// Vues collapsed exportées : passthrough au nom de la vue live (v_weapon_kills,
	// match_csrs_latest) — le collapse a tourné côté live au moment du cut.
	for _, ve := range sharedSnapshotViewExports {
		if err := createParquetViewStrict(ctx, db, ve.view, sharedFile(ve.dest)); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	// Vues dérivées recréées sur les tables/vues ci-dessus (ordre : gamertag_lookup AVANT
	// killer_victim_full qui en dépend).
	derivedViews := []string{
		analysis.GamertagLookupViewSQL(),
		`CREATE VIEW v_match_full AS SELECT mr.* FROM match_registry mr`,
		// Copie exacte de migration/steps_shared.go (killer_victim_pairs + 2 jointures
		// gamertag). Le test de fidélité garde contre toute divergence.
		`CREATE VIEW v_killer_victim_full AS
			SELECT kvp.*, k.gamertag AS killer_gamertag, v.gamertag AS victim_gamertag
			FROM killer_victim_pairs kvp
			LEFT JOIN v_gamertag_lookup k ON kvp.killer_xuid = k.xuid
			LEFT JOIN v_gamertag_lookup v ON kvp.victim_xuid = v.xuid`,
	}
	for _, ddl := range derivedViews {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("snapshot read shared: vue dérivée: %w", err)
		}
	}
	return &SnapshotQuerier{DB: db, Version: version, closeFn: func() { _ = db.Close() }}, nil
}
