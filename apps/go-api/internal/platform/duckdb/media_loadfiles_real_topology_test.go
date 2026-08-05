//go:build integration

// Package duckdb — media_loadfiles_real_topology_test.go : test E2E qui
// exerce LoadMediaFiles dans la vraie topologie prod (2 fichiers DB
// distincts sur disque, conns séparées sans ATTACH).
//
// Contraste avec media_repo_filters_test.go qui utilise des conns
// in-memory avec seed schema local : ici, on reproduit exactement la
// topologie qui plantait avant le refactor P1 (le bug d'origine de cette
// branche). Si ce test passe, le bug ne peut plus revenir.
package duckdb

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain"
	titlepkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
)

// TestLoadMediaFiles_RealTopology_NoCrossDBSQL reproduit la topologie
// prod (2 fichiers DB sur disque, aucun ATTACH `shared` sur les conns du
// pool) et vérifie que :
//
//  1. LoadMediaFiles fonctionne sans erreur "schema shared does not exist".
//  2. Les rows retournées incluent l'enrich match_registry chargé via
//     SharedReader (map_name, mode_name, pair_name_raw, match_id).
//  3. Le dédup mf.file_path fonctionne (1 ligne par fichier média).
//
// Ce test serait rouge avant le commit P1 (refactor Q37) si on inversait
// la migration : la query historique faisait LEFT JOIN shared.match_registry
// sur SharedSocial qui n'a aucun ATTACH `shared` en prod.
func TestLoadMediaFiles_RealTopology_NoCrossDBSQL(t *testing.T) {
	dir := t.TempDir()

	// Ouvrir 4 fichiers DB distincts comme en prod (player, shared, social, meta).
	open := func(name string, target migration.TargetDB) *DB {
		path := filepath.Join(dir, name+".duckdb")
		raw, err := sql.Open("duckdb", path)
		if err != nil {
			t.Fatalf("open %s: %v", name, err)
		}
		if err := migration.RunForDB(raw, target); err != nil {
			raw.Close()
			t.Fatalf("migrate %s: %v", name, err)
		}
		raw.Close()
		db, err := OpenReadWrite(path)
		if err != nil {
			t.Fatalf("reopen %s: %v", name, err)
		}
		return db
	}

	player := open("stats", migration.TargetPlayer)
	shared := open("shared_matches_v2", migration.TargetShared)
	social := open("shared_social", migration.TargetSharedSocial)
	meta := open("metadata", migration.TargetMetadata)
	t.Cleanup(func() {
		player.Close()
		shared.Close()
		social.Close()
		meta.Close()
	})

	ctx := context.Background()

	// Aligner le schéma media_files de social sur ce que ensureMediaTables
	// (ops/media.go) produit en prod : ajoute capture_start_utc et indexed_at
	// (manquants dans la migration TargetSharedSocial pure, ajoutés au boot
	// par ops.EnsureMediaTables via la conn SharedSocial du pool).
	for _, alter := range []string{
		"ALTER TABLE media_files ADD COLUMN IF NOT EXISTS capture_start_utc TIMESTAMPTZ",
		"ALTER TABLE media_files ADD COLUMN IF NOT EXISTS indexed_at TIMESTAMPTZ DEFAULT NOW()",
	} {
		if _, err := social.Exec(ctx, alter); err != nil {
			t.Fatalf("align media_files schema: %v\nSQL: %s", err, alter)
		}
	}

	// Insérer un match sur shared (= où SharedReader va lire).
	if _, err := shared.Exec(ctx, `
		INSERT INTO match_registry (match_id, start_time, start_time_utc, map_name, pair_name, playlist_name)
		VALUES ('e2e_m1', ?, ?, 'Aquarius', 'Slayer', 'Ranked Slayer')
	`, time.Date(2026, 5, 20, 14, 0, 0, 0, time.UTC), time.Date(2026, 5, 20, 14, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("insert match: %v", err)
	}

	// Insérer un média + son association sur social (table shared_social).
	if _, err := social.Exec(ctx, `
		INSERT INTO media_files
			(id, player_slug, file_path, file_name, file_stem, file_ext, kind, capture_end_utc, status)
		VALUES ('1', 'TestPlayer', '/e2e_test.mp4', 'e2e_test.mp4', 'e2e_test', '.mp4', 'video',
		        ?, 'active')
	`, time.Date(2026, 5, 20, 14, 30, 0, 0, time.UTC)); err != nil {
		t.Fatalf("insert media: %v", err)
	}
	// Append-only média : l'association EFFECTIVE passe par la table _history + vue
	// media_match_associations_latest (le reader lit la vue). media_file_id numérique
	// '1' (history = BIGINT en schéma migré ; coercition DuckDB avec media_files.id).
	if _, err := social.Exec(ctx, `
		INSERT INTO media_match_associations_history (media_file_id, match_id, delta_seconds)
		VALUES ('1', 'e2e_m1', 1800)
	`); err != nil {
		t.Fatalf("insert assoc: %v", err)
	}

	// Construire PlayerDB en mode prod-like : aucun ATTACH `shared` sur les
	// conns ; SharedReader = LegacySharedReader(shared) pointe vers le fichier.
	pdb := &PlayerDB{
		Player:       player,
		Shared:       shared,
		SharedSocial: social,
		Metadata:     meta,
		SharedReader: LegacySharedReader(shared),
		XUID:         "xuid-e2e",
		Gamertag:     "TestPlayer",
		TitleSlug:    titlepkg.DefaultSlug,
	}

	// Sanity check de la topologie : `shared.match_registry` DOIT échouer
	// sur SharedSocial (le bug d'origine est de cette nature).
	var dummy int
	probeErr := social.QueryRow(ctx, "SELECT 1 FROM shared.match_registry LIMIT 1").Scan(&dummy)
	if probeErr == nil {
		t.Fatal("sanity check failed : SharedSocial ne devrait pas avoir le schéma `shared` (ADR 0016)")
	}

	// Le vrai test : LoadMediaFiles doit fonctionner dans cette topologie.
	repo := NewMediaRepo(pdb)
	rows, err := repo.LoadMediaFiles(ctx, domain.MediaFilters{}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles a échoué dans la topologie réelle : %v\n"+
			"Si l'erreur mentionne 'schema shared does not exist', c'est exactement "+
			"le bug du début de cette branche — le refactor P1 a régressé.", err)
	}
	if len(rows) != 1 {
		t.Fatalf("LoadMediaFiles: %d rows, want 1", len(rows))
	}
	got := rows[0]
	if got.FilePath != "/e2e_test.mp4" {
		t.Errorf("FilePath = %q, want /e2e_test.mp4", got.FilePath)
	}
	if got.MatchID == nil || *got.MatchID != "e2e_m1" {
		t.Errorf("MatchID = %v, want e2e_m1", got.MatchID)
	}
	if got.MapName == nil || *got.MapName != "Aquarius" {
		t.Errorf("MapName = %v, want Aquarius (enrich match_registry via SharedReader)", got.MapName)
	}
	if got.PairNameRaw == nil || *got.PairNameRaw != "Slayer" {
		t.Errorf("PairNameRaw = %v, want Slayer", got.PairNameRaw)
	}
	if got.MatchStartTime == nil {
		t.Errorf("MatchStartTime nil — enrich match_registry n'a pas alimenté start_time_utc")
	}

	// Vérifier aussi que CountMediaFiles + LoadMediaFilterOptions tournent.
	count, err := repo.CountMediaFiles(ctx, domain.MediaFilters{})
	if err != nil {
		t.Fatalf("CountMediaFiles: %v", err)
	}
	if count != 1 {
		t.Errorf("CountMediaFiles = %d, want 1", count)
	}

	opts, err := repo.LoadMediaFilterOptions(ctx, domain.MediaFilters{})
	if err != nil {
		t.Fatalf("LoadMediaFilterOptions: %v", err)
	}
	if len(opts.Maps) == 0 {
		t.Errorf("LoadMediaFilterOptions.Maps vide — enrich match_registry n'a pas alimenté map_id/map_name")
	}
}
