// Package ops — data_quality_cgo_test.go : comptages/listes d'inconnus et
// upserts de résolution metadata sur DuckDB :memory: (driver CGO requis).
package ops

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openDQTestShared(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open shared :memory: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	for _, q := range []string{
		`CREATE TABLE match_registry (
			match_id VARCHAR, start_time TIMESTAMP, start_time_utc TIMESTAMPTZ,
			playlist_id VARCHAR, playlist_name VARCHAR,
			map_id VARCHAR, map_name VARCHAR,
			pair_id VARCHAR, pair_name VARCHAR,
			game_variant_id VARCHAR, game_variant_name VARCHAR,
			backfill_completed BIGINT DEFAULT 0,
			events_loaded BOOLEAN DEFAULT FALSE)`,
		`CREATE TABLE highlight_events (match_id VARCHAR)`,
		`CREATE TABLE weapon_kills (match_id VARCHAR)`,
		`CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR)`,
		`CREATE TABLE xuid_aliases (xuid VARCHAR, gamertag VARCHAR)`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("setup shared: %v", err)
		}
	}
	return db
}

func openDQTestMeta(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open meta :memory: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	for _, q := range []string{
		`CREATE TABLE mode_name_tr (mode_en VARCHAR, lang VARCHAR, name VARCHAR)`,
		`CREATE TABLE asset_translations (asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR, name VARCHAR)`,
		`CREATE TABLE playlists_catalog (title_slug VARCHAR, playlist_asset_id VARCHAR)`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("setup meta: %v", err)
		}
	}
	return db
}

// seedDQMatch insère un match minimal. pairName "" → NULL.
func seedDQMatch(t *testing.T, db *sql.DB, id, playlistID, playlistName, pairID, pairName string, bits int) {
	t.Helper()
	nullable := func(s string) any {
		if s == "" {
			return nil
		}
		return s
	}
	if _, err := db.Exec(`
		INSERT INTO match_registry (match_id, start_time_utc, playlist_id, playlist_name, pair_id, pair_name, backfill_completed)
		VALUES (?, now(), ?, ?, ?, ?, ?)`,
		id, nullable(playlistID), nullable(playlistName), nullable(pairID), nullable(pairName), bits); err != nil {
		t.Fatalf("seed match %s: %v", id, err)
	}
}

// TestDataQuality_CountsAndLists : chaque catégorie d'inconnu est comptée et
// listée — UUID bruts (name == id), mode non traduit FR, playlist hors
// catalogue, xuid orphelin, lying bits.
func TestDataQuality_CountsAndLists(t *testing.T) {
	ctx := context.Background()
	shared := openDQTestShared(t)
	meta := openDQTestMeta(t)

	// m1 : playlist UUID brut (name == id) + pair traduit FR connu.
	seedDQMatch(t, shared, "m1", "pl-uuid-1", "pl-uuid-1", "p1", "Arena:Slayer on Bazaar", 0)
	// m2 : pair non traduit ("Husky Raid"), playlist au catalogue.
	seedDQMatch(t, shared, "m2", "pl-known", "Quick Play", "p2", "Husky Raid CTF on Empyrean", 0)
	// m3 : lying bits (bits posés, tables vides) — pair traduit.
	seedDQMatch(t, shared, "m3", "pl-known", "Quick Play", "p1", "Arena:Slayer on Bazaar", dqBitEvents|dqBitWeaponKills)

	if _, err := meta.Exec(`INSERT INTO mode_name_tr VALUES ('Slayer', 'fr', 'Assassin')`); err != nil {
		t.Fatal(err)
	}
	if _, err := meta.Exec(`INSERT INTO playlists_catalog VALUES ('halo_infinite', 'pl-known')`); err != nil {
		t.Fatal(err)
	}
	// Participant sans alias → xuid orphelin ; un bot est exclu.
	for _, q := range []string{
		`INSERT INTO match_participants VALUES ('m1', 'x-orphan')`,
		`INSERT INTO match_participants VALUES ('m1', 'bid(1.0)')`,
	} {
		if _, err := shared.Exec(q); err != nil {
			t.Fatal(err)
		}
	}

	counts, err := CountDataQuality(ctx, shared, meta, "halo_infinite")
	if err != nil {
		t.Fatalf("CountDataQuality: %v", err)
	}
	if counts.RawUUIDPlaylists != 1 || counts.RawUUIDPairs != 0 {
		t.Errorf("raw uuids: playlists=%d pairs=%d (attendu 1, 0)", counts.RawUUIDPlaylists, counts.RawUUIDPairs)
	}
	if counts.UntranslatedModes != 1 {
		t.Errorf("UntranslatedModes = %d (attendu 1 — Husky Raid CTF)", counts.UntranslatedModes)
	}
	// pl-uuid-1 ET pl-known… pl-uuid-1 n'est pas au catalogue non plus.
	if counts.OrphanPlaylists != 1 {
		t.Errorf("OrphanPlaylists = %d (attendu 1 — pl-uuid-1)", counts.OrphanPlaylists)
	}
	if counts.OrphanXUIDs != 1 {
		t.Errorf("OrphanXUIDs = %d (attendu 1 — bots exclus)", counts.OrphanXUIDs)
	}
	if counts.LyingBitsEvents != 1 || counts.LyingBitsWeapons != 1 {
		t.Errorf("lying bits = %d/%d (attendu 1/1)", counts.LyingBitsEvents, counts.LyingBitsWeapons)
	}
	if counts.RawUUIDTotal() != 1 {
		t.Errorf("RawUUIDTotal = %d (attendu 1)", counts.RawUUIDTotal())
	}

	// Listes.
	raw, err := ListDataQualityIssues(ctx, shared, meta, "halo_infinite", "raw_uuids", 10)
	if err != nil || len(raw) != 1 || raw[0].AssetKind != "playlist" || raw[0].ID != "pl-uuid-1" {
		t.Errorf("raw_uuids = %+v, err=%v (attendu 1 playlist pl-uuid-1)", raw, err)
	}
	if raw[0].LastSeen == "" {
		t.Error("raw_uuids LastSeen vide (timestamp canonique attendu)")
	}

	modes, err := ListDataQualityIssues(ctx, shared, meta, "halo_infinite", "untranslated_modes", 10)
	if err != nil || len(modes) != 1 {
		t.Fatalf("untranslated_modes = %+v, err=%v (attendu 1)", modes, err)
	}
	if modes[0].ID == "" || modes[0].Label != "Husky Raid CTF on Empyrean" || modes[0].Occurrences != 1 {
		t.Errorf("untranslated mode inattendu : %+v", modes[0])
	}

	orphPl, err := ListDataQualityIssues(ctx, shared, meta, "halo_infinite", "orphan_playlists", 10)
	if err != nil || len(orphPl) != 1 || orphPl[0].ID != "pl-uuid-1" {
		t.Errorf("orphan_playlists = %+v, err=%v", orphPl, err)
	}

	orphX, err := ListDataQualityIssues(ctx, shared, meta, "halo_infinite", "orphan_xuids", 10)
	if err != nil || len(orphX) != 1 || orphX[0].ID != "x-orphan" {
		t.Errorf("orphan_xuids = %+v, err=%v", orphX, err)
	}

	if _, err := ListDataQualityIssues(ctx, shared, meta, "halo_infinite", "kind_inconnu", 10); err == nil {
		t.Error("kind inconnu doit retourner une erreur")
	}
}

// openDQTestMetaH5Like : metadata SANS mode_name_tr ni playlists_catalog —
// c'est le schéma metadata PROPRE d'un titre comme Halo 5 (PMT-9 : ses
// référentiels vivent dans asset_translations, pas dans les tables HINF).
func openDQTestMetaH5Like(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open meta h5-like :memory: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(
		`CREATE TABLE asset_translations (asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR, name VARCHAR)`,
	); err != nil {
		t.Fatalf("setup meta h5-like: %v", err)
	}
	return db
}

// TestDataQuality_MetaWithoutHINFReferentials : régression du bug local H5
// (2026-07-10) — GET data-quality?title=halo_5 renvoyait data_quality_error
// (« internal error ») parce que listUntranslatedModes/listOrphanPlaylists
// requêtaient mode_name_tr/playlists_catalog, ABSENTES du schéma metadata H5
// (Catalog Error → tout l'endpoint en 500). Attendu : détecteurs non
// applicables → compteurs à 0, PAS d'erreur — les autres détecteurs (shared)
// continuent de compter.
func TestDataQuality_MetaWithoutHINFReferentials(t *testing.T) {
	ctx := context.Background()
	shared := openDQTestShared(t)
	meta := openDQTestMetaH5Like(t)

	// Un mode non traduit + une playlist hors catalogue + un UUID brut : seuls
	// les détecteurs APPLICABLES doivent compter.
	seedDQMatch(t, shared, "m1", "pl-1", "Team Slayer", "p1", "Slayer on Truth", 0)
	seedDQMatch(t, shared, "m2", "11111111-2222-3333-4444-555555555555", "11111111-2222-3333-4444-555555555555", "", "", 0)

	counts, err := CountDataQuality(ctx, shared, meta, "halo_5")
	if err != nil {
		t.Fatalf("CountDataQuality doit dégrader proprement sans mode_name_tr/playlists_catalog, err=%v", err)
	}
	if counts.UntranslatedModes != 0 {
		t.Errorf("UntranslatedModes = %d, attendu 0 (détecteur non applicable)", counts.UntranslatedModes)
	}
	if counts.OrphanPlaylists != 0 {
		t.Errorf("OrphanPlaylists = %d, attendu 0 (détecteur non applicable)", counts.OrphanPlaylists)
	}
	if counts.RawUUIDPlaylists != 1 {
		t.Errorf("RawUUIDPlaylists = %d, attendu 1 (détecteur shared toujours actif)", counts.RawUUIDPlaylists)
	}

	// Les listes détaillées des kinds non applicables répondent vide, sans erreur.
	for _, kind := range []string{"untranslated_modes", "orphan_playlists"} {
		items, lerr := ListDataQualityIssues(ctx, shared, meta, "halo_5", kind, 10)
		if lerr != nil {
			t.Errorf("ListDataQualityIssues(%s) : err=%v, attendu dégradation propre", kind, lerr)
		}
		if len(items) != 0 {
			t.Errorf("ListDataQualityIssues(%s) = %d items, attendu 0", kind, len(items))
		}
	}
}

// TestDataQuality_TranslatedModeNotListed : une fois la traduction posée via
// UpsertModeTranslation, le mode sort de la liste (boucle de résolution
// complète : liste → action → liste vide).
func TestDataQuality_TranslatedModeNotListed(t *testing.T) {
	ctx := context.Background()
	shared := openDQTestShared(t)
	meta := openDQTestMeta(t)
	seedDQMatch(t, shared, "m1", "", "", "p1", "Husky Raid CTF on Empyrean", 0)

	before, err := listUntranslatedModes(ctx, shared, meta, 0)
	if err != nil || len(before) != 1 {
		t.Fatalf("avant résolution : %+v, err=%v", before, err)
	}
	modeEN := before[0].ID

	action, err := UpsertModeTranslation(ctx, meta, modeEN, "fr", "Husky Raid CTF")
	if err != nil || action != ResolveActionCreated {
		t.Fatalf("upsert: action=%q err=%v", action, err)
	}

	after, err := listUntranslatedModes(ctx, shared, meta, 0)
	if err != nil || len(after) != 0 {
		t.Fatalf("après résolution : %+v, err=%v (attendu vide)", after, err)
	}

	// Deuxième écriture = update (SELECT-then-UPDATE, pas de doublon).
	action, err = UpsertModeTranslation(ctx, meta, modeEN, "fr", "Husky Raid CTF v2")
	if err != nil || action != ResolveActionUpdated {
		t.Fatalf("re-upsert: action=%q err=%v", action, err)
	}
	var n int
	if err := meta.QueryRow(`SELECT COUNT(*) FROM mode_name_tr WHERE mode_en = ? AND lang = 'fr'`, modeEN).Scan(&n); err != nil || n != 1 {
		t.Fatalf("doublon mode_name_tr : n=%d err=%v", n, err)
	}
}

// TestDataQuality_UpsertAssetTranslation : created puis updated, sans doublon
// (la table test n'a volontairement PAS de PK — le pattern SELECT-then-UPDATE
// doit fonctionner sur les tables prebuilt sans contrainte).
func TestDataQuality_UpsertAssetTranslation(t *testing.T) {
	ctx := context.Background()
	meta := openDQTestMeta(t)

	action, err := UpsertAssetTranslation(ctx, meta, "pair", "p-uuid", "fr-FR", "Fiesta : Assassin")
	if err != nil || action != ResolveActionCreated {
		t.Fatalf("create: action=%q err=%v", action, err)
	}
	action, err = UpsertAssetTranslation(ctx, meta, "pair", "p-uuid", "fr-FR", "Fiesta : Assassin v2")
	if err != nil || action != ResolveActionUpdated {
		t.Fatalf("update: action=%q err=%v", action, err)
	}
	var n int
	var name string
	if err := meta.QueryRow(`SELECT COUNT(*), MAX(name) FROM asset_translations WHERE asset_id='p-uuid'`).Scan(&n, &name); err != nil {
		t.Fatal(err)
	}
	if n != 1 || name != "Fiesta : Assassin v2" {
		t.Fatalf("asset_translations: n=%d name=%q", n, name)
	}

	// Champs vides refusés.
	if _, err := UpsertAssetTranslation(ctx, meta, "pair", "", "fr-FR", "x"); err == nil {
		t.Error("asset_id vide doit être refusé")
	}
}
