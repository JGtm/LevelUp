//go:build integration

package migration

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// oldPlayerCSRSnapshotsDDL — schéma EXACT de player_csr_snapshots AVANT la conversion
// append-only (état pré-2026-05-24, tiré de git show 37264462f^:internal/sync/schema.go) :
// PK composite (playlist_id, season_id), SANS colonnes id/written_at. C'est le schéma qui fait
// échouer le BIND de player_csr_snapshots_latest dans playerSchemaSQL (findings M2/M3).
const oldPlayerCSRSnapshotsDDL = `
CREATE TABLE player_csr_snapshots (
	playlist_id                   VARCHAR NOT NULL,
	playlist_name                 VARCHAR,
	queue                         VARCHAR,
	input                         VARCHAR,
	season_id                     VARCHAR NOT NULL,
	current_value                 FLOAT,
	current_tier                  VARCHAR,
	current_sub_tier              SMALLINT,
	current_measurement_remaining INTEGER,
	season_value                  FLOAT,
	season_tier                   VARCHAR,
	season_sub_tier               SMALLINT,
	alltime_value                 FLOAT,
	alltime_tier                  VARCHAR,
	alltime_sub_tier              SMALLINT,
	fetched_at                    TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
	PRIMARY KEY (playlist_id, season_id)
)`

// pcsLatestViewDDL — vue player_csr_snapshots_latest telle qu'émise par playerSchemaSQL
// (sync/schema.go). Rejouée ici pour PROUVER que, après réparation, son BIND réussit (id +
// written_at présents) — c'est exactement l'instruction qui plantait sur l'ancien schéma.
const pcsLatestViewDDL = `
CREATE OR REPLACE VIEW player_csr_snapshots_latest AS
	SELECT * FROM player_csr_snapshots
	QUALIFY ROW_NUMBER() OVER (PARTITION BY playlist_id, season_id ORDER BY written_at DESC, id DESC) = 1`

// TestPlayerCSRSnapshotsAppendOnly_LegacySwap verrouille C1 : sur l'ANCIEN schéma
// player_csr_snapshots (sans id/written_at), EnsurePlayerCSRSnapshotsAppendOnly convertit en
// append-only sans perte, la vue _latest bind ensuite, et la réparation est idempotente.
func TestPlayerCSRSnapshotsAppendOnly_LegacySwap(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(oldPlayerCSRSnapshotsDDL); err != nil {
		t.Fatalf("legacy csr snapshots DDL: %v", err)
	}
	seed := []struct {
		playlist, season string
		val              float64
	}{
		{"ranked-arena", "csrseason12", 1450},
		{"ranked-arena", "csrseason13-2", 1502},
		{"ranked-tactical", "csrseason13-2", 1333},
	}
	for _, r := range seed {
		if _, err := db.Exec(
			`INSERT INTO player_csr_snapshots (playlist_id, season_id, current_value) VALUES (?, ?, ?)`,
			r.playlist, r.season, r.val); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	// Sanity : sur l'ancien schéma, la vue ne peut PAS bind (colonnes written_at/id absentes).
	if _, err := db.Exec(pcsLatestViewDDL); err == nil {
		t.Fatal("attendu : bind de player_csr_snapshots_latest DEVRAIT échouer sur l'ancien schéma")
	}

	// Réparation (C1).
	if err := EnsurePlayerCSRSnapshotsAppendOnly(db); err != nil {
		t.Fatalf("EnsurePlayerCSRSnapshotsAppendOnly (ancien schéma): %v", err)
	}

	// id + written_at ajoutés, PK technique posée.
	for _, col := range []string{"id", "written_at"} {
		ok, err := columnExists(db, "player_csr_snapshots", col)
		if err != nil || !ok {
			t.Fatalf("colonne %q absente après conversion (err=%v)", col, err)
		}
	}
	if ok, _ := hasPrimaryKey(db, "player_csr_snapshots"); !ok {
		t.Error("player_csr_snapshots devrait avoir une PK (id technique) après conversion")
	}

	// Zéro perte : 3 lignes physiques et 3 via la vue _latest (1 par clé fonctionnelle).
	if _, err := db.Exec(pcsLatestViewDDL); err != nil {
		t.Fatalf("la vue _latest devrait bind après réparation: %v", err)
	}
	var phys, latest int
	_ = db.QueryRow(`SELECT COUNT(*) FROM player_csr_snapshots`).Scan(&phys)
	_ = db.QueryRow(`SELECT COUNT(*) FROM player_csr_snapshots_latest`).Scan(&latest)
	if phys != 3 || latest != 3 {
		t.Fatalf("rows: physique=%d latest=%d, want 3/3 (zéro perte)", phys, latest)
	}

	// Idempotence : re-run → aucun échec, aucune re-conversion, aucune perte.
	if err := EnsurePlayerCSRSnapshotsAppendOnly(db); err != nil {
		t.Fatalf("EnsurePlayerCSRSnapshotsAppendOnly (idempotence): %v", err)
	}
	var physAfter int
	_ = db.QueryRow(`SELECT COUNT(*) FROM player_csr_snapshots`).Scan(&physAfter)
	if physAfter != 3 {
		t.Fatalf("idempotence: physique=%d après 2e run, want 3 (pas de duplication/perte)", physAfter)
	}
	ok, err := tableExists(db, "player_csr_snapshots__appendonly")
	if err != nil {
		t.Fatalf("check orphan: %v", err)
	}
	if ok {
		t.Error("table orpheline player_csr_snapshots__appendonly ne devrait pas subsister")
	}
}

// TestPlayerCSRSnapshotsAppendOnly_FreshDBNoop : sur une DB vierge (table absente), la
// réparation est un no-op sans erreur (le schéma append-only est créé par playerSchemaSQL).
func TestPlayerCSRSnapshotsAppendOnly_FreshDBNoop(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	if err := EnsurePlayerCSRSnapshotsAppendOnly(db); err != nil {
		t.Fatalf("EnsurePlayerCSRSnapshotsAppendOnly (table absente) devrait être no-op: %v", err)
	}
	if ok, _ := tableExists(db, "player_csr_snapshots"); ok {
		t.Error("la réparation ne doit PAS créer la table sur DB vierge (c'est le rôle de playerSchemaSQL)")
	}
}
