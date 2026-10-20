//go:build integration

// Package duckdb — lookups_degrades_journalises_test.go : E5.3 du plan backlog 2026-10-16. Une
// résolution best-effort (libellé d'arme, image de carte, modes connus) qui rend un résultat vide
// après une erreur de lecture le DIT dans le journal avant de se dégrader. Une table absente
// (schéma non migré) reste un DEBUG.
package duckdb

import (
	"context"
	"testing"
)

// metaSansColonnes : une metadata dont les tables existent mais n'ont pas les colonnes lues
// (panne de lecture qui n'est PAS une table absente).
func metaSansColonnes(t *testing.T, tables ...string) *DB {
	t.Helper()
	meta := openMemDB(t)
	for _, tbl := range tables {
		if _, err := meta.Exec(context.Background(), "CREATE TABLE "+tbl+" (inutile VARCHAR)"); err != nil {
			t.Fatalf("fixture %s : %v", tbl, err)
		}
	}
	return meta
}

func TestResolveWeaponLabelsOnly_LectureEnEchec_EstJournalisee(t *testing.T) {
	buf := captureSlog(t)
	got := resolveWeaponLabelsOnly(context.Background(), metaSansColonnes(t, "weapon_labels"), []int64{1, 2})
	if len(got) != 0 {
		t.Errorf("aucun libellé attendu, obtenu %v", got)
	}
	if !hasWarnContaining(t, buf, "weapon_labels", "lecture") {
		t.Errorf("WARN attendu sur la lecture de weapon_labels ; logs=%s", buf.String())
	}
}

func TestResolveWeaponLabelsOnly_TableAbsente_RestePrudente(t *testing.T) {
	buf := captureSlog(t)
	_ = resolveWeaponLabelsOnly(context.Background(), openMemDB(t), []int64{1})
	if hasWarnContaining(t, buf, "weapon_labels") || hasLogLevelContaining(t, buf, "ERROR", "weapon_labels") {
		t.Errorf("une table absente n'est pas une panne (DEBUG) ; logs=%s", buf.String())
	}
	if !hasLogLevelContaining(t, buf, "DEBUG", "weapon_labels") {
		t.Errorf("la dégradation doit laisser une trace DEBUG ; logs=%s", buf.String())
	}
}

func TestLookupMapImageURL_LectureEnEchec_EstJournalisee(t *testing.T) {
	buf := captureSlog(t)
	pdb := &PlayerDB{Metadata: metaSansColonnes(t, "map_images_registry"), TitleSlug: "halo_infinite"}
	if got := NewMatchViewRepo(pdb, pTestXUID).lookupMapImageURL(context.Background(), "map-1"); got != nil {
		t.Errorf("aucune image attendue, obtenu %q", *got)
	}
	if !hasWarnContaining(t, buf, "image de carte") {
		t.Errorf("WARN attendu sur la lecture du registre d'images ; logs=%s", buf.String())
	}
}

func TestLoadKnownModesEN_LectureEnEchec_EstJournalisee(t *testing.T) {
	buf := captureSlog(t)
	if got := loadKnownModesEN(context.Background(), metaSansColonnes(t, "mode_name_tr")); got != nil {
		t.Errorf("aucun mode attendu, obtenu %v", got)
	}
	if !hasWarnContaining(t, buf, "loadKnownModesEN") {
		t.Errorf("WARN attendu ; logs=%s", buf.String())
	}
}

func TestLoadKnownModesEN_TableAbsente_RestePrudente(t *testing.T) {
	buf := captureSlog(t)
	_ = loadKnownModesEN(context.Background(), openMemDB(t))
	if hasWarnContaining(t, buf, "loadKnownModesEN") {
		t.Errorf("une table absente n'est pas une panne ; logs=%s", buf.String())
	}
	if !hasLogLevelContaining(t, buf, "DEBUG", "loadKnownModesEN") {
		t.Errorf("la dégradation doit laisser une trace DEBUG ; logs=%s", buf.String())
	}
}
