//go:build integration

// api_wire_stores_test.go : les dépôts nés du déplacement du SQL de `api/wire` (accès aux bases d'un titre,
// expansion des playlists, registre des rejeux, couverture d'arme), contre de vraies bases migrées. Chaque test fixe ce
// que la requête déplacée rend : mêmes lignes qu'avant le déplacement.
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"testing"

	titlepkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/port"
)

func TestTitleBases_ReadSharedEtMetadata(t *testing.T) {
	root := t.TempDir()
	pr := titlepkg.NewPathResolver(root)
	slug := titlepkg.DefaultSlug
	b := TitleBases{Paths: pr, Log: slog.Default()}

	// Base partagée absente : erreur nommée, aucun handle.
	if _, _, err := b.ReadShared(context.Background(), slug); err == nil {
		t.Fatal("ReadShared sans fichier : erreur attendue")
	}
	if db, closeMeta := b.Metadata(context.Background(), slug); db != nil {
		t.Fatal("Metadata sans fichier : nil attendu")
	} else {
		closeMeta()
	}

	migratedFile(t, pr.SharedDBPath(slug), migration.TargetShared)
	migratedFile(t, pr.MetadataDBPath(slug), migration.TargetMetadata)
	h, err := b.Open(context.Background(), slug)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer h.Close()
	if h.Shared == nil || h.Meta == nil {
		t.Fatalf("handles = %v / %v, attendus non nil", h.Shared, h.Meta)
	}
	var n int
	if err := h.Shared.QueryRow(`SELECT COUNT(*) FROM match_registry`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("lecture shared : %d, %v", n, err)
	}
}

func TestPlaylistsSeenReader_PlaylistsDistinctesAvecVersion(t *testing.T) {
	root := t.TempDir()
	pr := titlepkg.NewPathResolver(root)
	slug := titlepkg.DefaultSlug
	migratedFile(t, pr.SharedDBPath(slug), migration.TargetShared,
		`INSERT INTO match_registry (match_id, playlist_id, playlist_version_id) VALUES
			('m1', 'pl-1', 'v1'), ('m2', 'pl-1', 'v1'), ('m3', 'pl-2', 'v7'),
			('m4', NULL, 'v1'), ('m5', '', 'v1'), ('m6', 'pl-3', NULL), ('m7', 'pl-4', '')`)

	got, err := PlaylistsSeenReader{Bases: TitleBases{Paths: pr, Log: slog.Default()}}.
		PlaylistsSeenInMatches(context.Background(), slug)
	if err != nil {
		t.Fatalf("PlaylistsSeenInMatches: %v", err)
	}
	seen := map[port.PlaylistRef]bool{}
	for _, p := range got {
		seen[p] = true
	}
	want := map[port.PlaylistRef]bool{{ID: "pl-1", Version: "v1"}: true, {ID: "pl-2", Version: "v7"}: true}
	if len(got) != 2 || !seen[port.PlaylistRef{ID: "pl-1", Version: "v1"}] || !seen[port.PlaylistRef{ID: "pl-2", Version: "v7"}] {
		t.Fatalf("playlists = %v, attendu %v", got, want)
	}
}

func TestReplayRegistryRepo_IdentiteEtModesNonGeres(t *testing.T) {
	root := t.TempDir()
	pr := titlepkg.NewPathResolver(root)
	slug := titlepkg.DefaultSlug
	migratedFile(t, pr.SharedDBPath(slug), migration.TargetShared,
		`INSERT INTO match_registry (match_id, map_name, map_id, is_firefight) VALUES
			('edc5daf6-0df8-4eb0-b578-b8eb892ce184', ' Cole Protocol ', 'b', TRUE),
			('39910eb1-8c01-4e26-bc60-965cb2dcc789', 'Recharge', ' a ', FALSE),
			('7b7b7b7b-0000-4000-8000-000000000000', 'Sans drapeau', NULL, NULL)`)
	migratedFile(t, pr.MetadataDBPath(slug), migration.TargetMetadata,
		`INSERT INTO asset_translations (asset_type, asset_id, lang, name) VALUES ('map', 'a', 'en-US', 'Recharge EN')`)
	h, err := TitleBases{Paths: pr, Log: slog.Default()}.Open(context.Background(), slug)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer h.Close()
	repo := NewReplayRegistryRepo(h.Shared, h.Meta)
	ctx := context.Background()

	hits, err := repo.MatchesByIDOrPrefix(ctx, "39910eb1")
	if err != nil || len(hits) != 1 {
		t.Fatalf("préfixe : %v, %v", hits, err)
	}
	if h0 := hits[0]; h0.ID != "39910eb1-8c01-4e26-bc60-965cb2dcc789" || h0.RawMapName != "Recharge" ||
		h0.MapID != "a" || h0.UnmanagedMode {
		t.Errorf("ligne = %+v", h0)
	}
	if hits, _ := repo.MatchesByIDOrPrefix(ctx, "edc5daf6"); len(hits) != 1 || !hits[0].UnmanagedMode || hits[0].RawMapName != "Cole Protocol" {
		t.Errorf("Firefight = %+v, attendu mode non géré, nom sans espaces", hits)
	}
	if en, err := repo.EnglishMapName(ctx, "a"); err != nil || en != "Recharge EN" {
		t.Errorf("nom EN = %q, %v", en, err)
	}
	if en, err := repo.EnglishMapName(ctx, "zz"); err == nil || en != "" {
		t.Errorf("carte inconnue : %q, %v (attendu une erreur de lecture, nom vide)", en, err)
	}
	if en, err := NewReplayRegistryRepo(h.Shared, nil).EnglishMapName(ctx, "a"); err != nil || en != "" {
		t.Errorf("sans metadata : %q, %v (attendu vide, sans erreur)", en, err)
	}
	ids, err := repo.UnmanagedModeMatchIDs(ctx)
	if err != nil || len(ids) != 1 || ids[0] != "edc5daf6-0df8-4eb0-b578-b8eb892ce184" {
		t.Errorf("modes non gérés = %v, %v", ids, err)
	}
}

func TestWeaponCoverageRepo_FragsEtRegistre(t *testing.T) {
	root := t.TempDir()
	pr := titlepkg.NewPathResolver(root)
	migratedFile(t, pr.MetadataDBPath("halo_infinite"), migration.TargetMetadata,
		`INSERT INTO weapons (title_slug, weapon_key, name) VALUES ('halo_infinite', 'br75', 'BR75'), ('halo_5', 'ar', 'AR')`)
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	for _, q := range []string{
		`CREATE TABLE w (effective_weapon_id UBIGINT)`,
		`INSERT INTO w VALUES (0), (1), (2), (10), (10), (10), (20)`,
		`CREATE VIEW v_weapon_kills AS SELECT effective_weapon_id FROM w`,
		`CREATE TABLE p (source_tag UINTEGER, kills INTEGER)`,
		`INSERT INTO p VALUES (7, 3), (7, 4), (9, 1), (NULL, 5)`,
		`CREATE VIEW match_kill_pairs_latest AS SELECT source_tag, kills FROM p`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	meta, closeMeta := TitleBases{Paths: pr, Log: slog.Default()}.Metadata(context.Background(), "halo_infinite")
	defer closeMeta()
	repo := NewWeaponCoverageRepo(db, meta)
	ctx := context.Background()

	if !repo.HasMetadata() || NewWeaponCoverageRepo(db, nil).HasMetadata() {
		t.Error("HasMetadata incohérent")
	}
	byID, err := repo.KillsByWeaponID(ctx)
	if err != nil {
		t.Fatalf("KillsByWeaponID: %v", err)
	}
	got := map[int64]int{}
	for _, c := range byID {
		got[c.WeaponID] = c.Kills
	}
	if len(got) != 2 || got[10] != 3 || got[20] != 1 {
		t.Errorf("frags par arme = %v, attendu {10:3, 20:1} (sentinelles 0/1/2 exclues)", got)
	}
	bySource, err := repo.KillsBySourceTag(ctx)
	if err != nil {
		t.Fatalf("KillsBySourceTag: %v", err)
	}
	srcs := map[uint32]int{}
	for _, c := range bySource {
		srcs[c.SourceTag] = c.Kills
	}
	if len(srcs) != 2 || srcs[7] != 7 || srcs[9] != 1 {
		t.Errorf("frags par source = %v, attendu {7:7, 9:1} (source NULL exclue)", srcs)
	}
	keys, err := repo.RegistryWeaponKeys(ctx, "halo_infinite")
	// Le registre livré par les migrations porte déjà les armes du titre ; la clé semée s'y ajoute, celle de
	// l'autre titre n'y est pas.
	if err != nil || !keys["br75"] || !keys["hinf_br75"] || keys["ar"] {
		t.Errorf("clés du registre (%d) = %v, %v", len(keys), keys, err)
	}
}

func TestSignalUnreadableRow_CompteParTitre(t *testing.T) {
	const compteur = "wire_rows_scan_failed_total"
	avant := observability.LoadCounterT("halo_5", compteur)
	signalUnreadableRow(context.Background(), "halo_5", "test", errors.New("ligne illisible"))
	if d := observability.LoadCounterT("halo_5", compteur) - avant; d != 1 {
		t.Fatalf("compteur +%d, want +1", d)
	}
}
