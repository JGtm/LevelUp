//go:build cgo

// Package wire — registry_actions_translation_cgo_test.go : les actions admin de traduction
// écrivent une ligne par langue saisie, sous la valeur de `lang` propre à chaque table
// (code de langue pour mode_name_tr, première étiquette Halo pour asset_translations).
//
// Base DuckDB réelle dans un répertoire temporaire (driver CGO requis).
package wire

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/platform/duckdb"
)

// newTranslationRegistry crée un metadata.duckdb minimal pour le titre et retourne le registre qui
// l'ouvre, plus un lecteur de (clé, langue) → nom sur la même base.
func newTranslationRegistry(t *testing.T) (*ServiceRegistry, func(query string, args ...any) []string) {
	t.Helper()
	const slug = "halo_infinite"
	root := t.TempDir()
	metaPath := title.NewPathResolver(root).MetadataDBPath(slug)
	if err := os.MkdirAll(filepath.Dir(metaPath), 0o755); err != nil {
		t.Fatalf("mkdir warehouse : %v", err)
	}
	setup, err := sql.Open("duckdb", metaPath)
	if err != nil {
		t.Fatalf("open metadata : %v", err)
	}
	for _, q := range []string{
		`CREATE TABLE mode_name_tr (mode_en VARCHAR NOT NULL, lang VARCHAR NOT NULL, name VARCHAR NOT NULL)`,
		`CREATE TABLE asset_translations (asset_id VARCHAR NOT NULL, asset_type VARCHAR NOT NULL, lang VARCHAR NOT NULL, name VARCHAR NOT NULL)`,
	} {
		if _, err := setup.Exec(q); err != nil {
			t.Fatalf("setup : %v", err)
		}
	}
	if _, err := setup.Exec("CHECKPOINT"); err != nil {
		t.Fatalf("checkpoint : %v", err)
	}
	if err := setup.Close(); err != nil {
		t.Fatalf("close setup : %v", err)
	}
	reg := &ServiceRegistry{cfg: &config.AppConfig{RepoRoot: root}}
	read := func(query string, args ...any) []string {
		db, err := duckdb.OpenReadWriteShared(metaPath)
		if err != nil {
			t.Fatalf("open lecture : %v", err)
		}
		defer db.Close() //nolint:errcheck // ref-count
		rows, err := db.SQLDb().Query(query, args...)
		if err != nil {
			t.Fatalf("query : %v", err)
		}
		defer rows.Close()
		var out []string
		for rows.Next() {
			var s string
			if err := rows.Scan(&s); err != nil {
				t.Fatalf("scan : %v", err)
			}
			out = append(out, s)
		}
		return out
	}
	return reg, read
}

func TestResolveModeTranslation_OneRowPerLanguage(t *testing.T) {
	reg, read := newTranslationRegistry(t)
	ctx := context.Background()

	res, err := reg.ResolveModeTranslation(ctx, "halo_infinite", domain.ModeTranslationRequest{
		ModeKey: "Slayer", Names: map[string]string{"fr": "Assassin", "en": "Slayer EN"},
	})
	if err != nil {
		t.Fatalf("ResolveModeTranslation : %v", err)
	}
	if res.Action != ops.ResolveActionCreated || res.ModeKey != "Slayer" {
		t.Fatalf("résultat = %+v", res)
	}
	got := read(`SELECT lang || '=' || name FROM mode_name_tr WHERE mode_en = ? ORDER BY lang`, "Slayer")
	if len(got) != 2 || got[0] != "en=Slayer EN" || got[1] != "fr=Assassin" {
		t.Fatalf("mode_name_tr = %v (attendu en=... et fr=..., lang = code de langue)", got)
	}

	// Une seule langue renseignée : une ligne mise à jour, l'autre intacte, action « updated ».
	res, err = reg.ResolveModeTranslation(ctx, "halo_infinite", domain.ModeTranslationRequest{
		ModeKey: "Slayer", Names: map[string]string{"fr": "Assassin v2"},
	})
	if err != nil {
		t.Fatalf("ResolveModeTranslation (maj) : %v", err)
	}
	if res.Action != ops.ResolveActionUpdated {
		t.Fatalf("action = %q (attendu updated)", res.Action)
	}
	got = read(`SELECT lang || '=' || name FROM mode_name_tr WHERE mode_en = ? ORDER BY lang`, "Slayer")
	if len(got) != 2 || got[0] != "en=Slayer EN" || got[1] != "fr=Assassin v2" {
		t.Fatalf("mode_name_tr après mise à jour = %v", got)
	}

	if _, err := reg.ResolveModeTranslation(ctx, "halo_infinite", domain.ModeTranslationRequest{
		ModeKey: "Slayer", Names: map[string]string{"zz": "?"},
	}); err == nil {
		t.Fatal("une langue inconnue seule doit être refusée")
	}
}

func TestResolveAssetTranslation_HaloTagPerLanguage(t *testing.T) {
	reg, read := newTranslationRegistry(t)
	ctx := context.Background()

	res, err := reg.ResolveAssetTranslation(ctx, "halo_infinite", domain.AssetTranslationRequest{
		AssetKind: "map", AssetID: "m-1", Names: map[string]string{"fr": "Aquarius", "en": "Aquarius EN"},
	})
	if err != nil {
		t.Fatalf("ResolveAssetTranslation : %v", err)
	}
	if res.Action != ops.ResolveActionCreated || len(res.Langs) != 2 || res.Langs[0] != "fr-FR" || res.Langs[1] != "en-US" {
		t.Fatalf("résultat = %+v (attendu langs fr-FR, en-US)", res)
	}
	got := read(`SELECT lang || '=' || name FROM asset_translations WHERE asset_id = ? AND asset_type = ? ORDER BY lang`, "m-1", "map")
	if len(got) != 2 || got[0] != "en-US=Aquarius EN" || got[1] != "fr-FR=Aquarius" {
		t.Fatalf("asset_translations = %v (attendu en-US et fr-FR)", got)
	}

	// Rejouer la même saisie est idempotent : mêmes lignes, action « updated », aucun doublon.
	res, err = reg.ResolveAssetTranslation(ctx, "halo_infinite", domain.AssetTranslationRequest{
		AssetKind: "map", AssetID: "m-1", Names: map[string]string{"fr": "Aquarius", "en": "Aquarius EN"},
	})
	if err != nil || res.Action != ops.ResolveActionUpdated {
		t.Fatalf("rejeu : %+v err=%v (attendu updated)", res, err)
	}
	if got := read(`SELECT name FROM asset_translations WHERE asset_id = ?`, "m-1"); len(got) != 2 {
		t.Fatalf("doublon après rejeu : %v", got)
	}

	if _, err := reg.ResolveAssetTranslation(ctx, "halo_infinite", domain.AssetTranslationRequest{
		AssetKind: "bogus", AssetID: "m-2", Names: map[string]string{"fr": "x"},
	}); err == nil {
		t.Fatal("un asset_kind invalide doit être refusé")
	}
}
