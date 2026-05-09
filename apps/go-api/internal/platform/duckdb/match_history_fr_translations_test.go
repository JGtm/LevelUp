//go:build integration

// match_history_fr_translations_test.go — tests d'intégration de
// applyMatchHistoryFRTranslations (cf. thought_log 2026-05-09 root cause P2).
//
// Couvre spécifiquement le cas où pair_name brut est vide en DB et
// asset_translations[pair_id, fr-FR] retourne l'EN raw "Arena:CTF on X" — le
// helper analysis.ResolvePairNameFR doit re-normaliser et re-lookup mode_name_tr.
package duckdb

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/domain"

	_ "github.com/duckdb/duckdb-go/v2"
)

func setupMetadataWithModeTranslations(t *testing.T) *DB {
	t.Helper()
	sqlDB, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	db := &DB{sqlDB: sqlDB, path: ":memory:"}

	ctx := context.Background()
	for _, q := range []string{
		`CREATE TABLE mode_name_tr (mode_en VARCHAR, lang VARCHAR, name VARCHAR, PRIMARY KEY (mode_en, lang))`,
		`CREATE TABLE asset_translations (asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR, name VARCHAR, PRIMARY KEY (asset_id, asset_type, lang))`,
	} {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatalf("schema: %v", err)
		}
	}

	// mode_name_tr : seeds critiques.
	for _, kv := range [][3]string{
		{"CTF", "fr", "Capture du drapeau"},
		{"Strongholds", "fr", "Bases"},
		{"Slayer", "fr", "Assassin"},
		{"Team Slayer", "fr", "Assassin en équipe"},
		{"Neutral Flag CTF", "fr", "Drapeau neutre"},
	} {
		if _, err := db.Exec(ctx, `INSERT INTO mode_name_tr (mode_en, lang, name) VALUES (?, ?, ?)`, kv[0], kv[1], kv[2]); err != nil {
			t.Fatalf("seed mode_name_tr: %v", err)
		}
	}

	// asset_translations : pair_id corrompu — toutes les langues retournent l'EN raw.
	pairID := "bd1457cc-corrupted-pair"
	for _, lang := range []string{"en-US", "fr", "fr-FR", "de-DE"} {
		_, _ = db.Exec(ctx, `INSERT INTO asset_translations (asset_id, asset_type, lang, name) VALUES (?, ?, ?, ?)`,
			pairID, "pair", lang, "Arena:CTF on Shiro")
	}
	// pair_id correct — fr-FR a la traduction.
	pairIDOK := "ok-pair-001"
	_, _ = db.Exec(ctx, `INSERT INTO asset_translations (asset_id, asset_type, lang, name) VALUES (?, ?, ?, ?)`,
		pairIDOK, "pair", "fr-FR", "Capture du drapeau sur Aquarius")
	_, _ = db.Exec(ctx, `INSERT INTO asset_translations (asset_id, asset_type, lang, name) VALUES (?, ?, ?, ?)`,
		pairIDOK, "pair", "en-US", "Arena:CTF on Aquarius")

	return db
}

func TestApplyMatchHistoryFRTranslations_HandlesCorruptedAssetTranslations(t *testing.T) {
	ctx := context.Background()
	meta := setupMetadataWithModeTranslations(t)
	pdb := &PlayerDB{Metadata: meta}

	// Cas reproduisant le bug observé sur Chocoboflor :
	corruptedID := "bd1457cc-corrupted-pair"
	pairNameOK := "Arena:CTF on Aquarius"
	pairNameFROK := pairNameOK // COALESCE(NULL, EN)
	emptyPairName := ""
	pairNameFRPlaceholder := "Arena:CTF on Shiro" // ce que COALESCE renvoie si pair_name_fr stocké en EN

	rows := []domain.MatchHistoryRawRow{
		// Cas A : pair_name brut renseigné, traduction directe via mode_name_tr.
		{
			MatchID:    "match-A",
			PairName:   &pairNameOK,
			PairNameFR: &pairNameFROK,
		},
		// Cas B (root cause P2) : pair_name vide, asset_translations a l'EN raw,
		// le helper doit re-normaliser et trouver "CTF" → "Capture du drapeau".
		{
			MatchID:    "match-B",
			PairName:   &emptyPairName,
			PairNameFR: &pairNameFRPlaceholder,
			PairID:     &corruptedID,
		},
	}

	applyMatchHistoryFRTranslations(ctx, pdb, rows)

	if got := derefString(rows[0].PairNameFR); got != "Capture du drapeau" {
		t.Errorf("Cas A (pair_name OK): PairNameFR = %q, want %q", got, "Capture du drapeau")
	}
	if got := derefString(rows[1].PairNameFR); got != "Capture du drapeau" {
		t.Errorf("Cas B (asset corrompu): PairNameFR = %q, want %q", got, "Capture du drapeau")
	}
}

func TestFiltersRepo_applyModeFRTranslations_HandlesCorruptedAssetTranslations(t *testing.T) {
	ctx := context.Background()
	meta := setupMetadataWithModeTranslations(t)
	r := &FiltersRepo{pdb: &PlayerDB{Metadata: meta}}

	corruptedID := "bd1457cc-corrupted-pair"
	pairNameOK := "Arena:Strongholds on Live Fire"
	pairNameFROK := pairNameOK
	emptyPairName := ""
	pairNameFRPlaceholder := "Arena:Team Slayer on Bazaar - Forge"
	pairIDOther := "another-pair-id"
	_, _ = meta.Exec(ctx, `INSERT INTO asset_translations (asset_id, asset_type, lang, name) VALUES (?, 'pair', 'fr-FR', 'Arena:Team Slayer on Bazaar - Forge')`, pairIDOther)

	rows := []domain.FilterMatchRow{
		{
			MatchID:    "match-A",
			PairName:   &pairNameOK,
			PairNameFR: &pairNameFROK,
		},
		{
			MatchID:    "match-B",
			PairName:   &emptyPairName,
			PairNameFR: &pairNameFRPlaceholder,
			PairID:     &corruptedID,
		},
		{
			MatchID:    "match-C",
			PairName:   &emptyPairName,
			PairNameFR: &pairNameFRPlaceholder,
			PairID:     &pairIDOther,
		},
	}
	r.applyModeFRTranslations(ctx, rows)

	if got := derefString(rows[0].PairNameFR); got != "Bases" {
		t.Errorf("Cas A: PairNameFR = %q, want %q", got, "Bases")
	}
	// Cas B et C : tous deux corrompus → re-normaliser puis re-lookup.
	for i, want := range map[int]string{1: "Capture du drapeau", 2: "Assassin en équipe"} {
		if got := derefString(rows[i].PairNameFR); got != want {
			t.Errorf("Cas %s: PairNameFR = %q, want %q", rows[i].MatchID, got, want)
		}
	}
}
