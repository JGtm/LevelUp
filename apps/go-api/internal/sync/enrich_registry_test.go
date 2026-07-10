//go:build integration

package sync

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// setupMetaWithTranslations crée une metadata DB en-mémoire avec un seed
// asset_translations pour reproduire les cas réels :
//   - playlist UUID-A : pas de traduction → enrichissement no-op (UUID brut conservé)
//   - playlist UUID-B : traduction en-US présente → enrichissement override
//   - map UUID-C : traduction en-US présente → override
func setupMetaWithTranslations(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`CREATE TABLE asset_translations (
		asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR, name VARCHAR,
		PRIMARY KEY (asset_id, asset_type, lang))`); err != nil {
		t.Fatalf("schema: %v", err)
	}
	for _, kv := range [][4]string{
		{"playlist-known-uuid", "playlist", "en-US", "Quick Play"},
		{"map-known-uuid", "map", "en-US", "Aquarius"},
		{"pair-known-uuid", "pair", "en-US", "Arena:Slayer on Aquarius"},
		{"gv-known-uuid", "game_variant", "en-US", "Slayer"},
		// playlist-unknown-uuid / pair-absent-uuid n'ont aucune traduction.
	} {
		if _, err := db.Exec(`INSERT INTO asset_translations VALUES (?, ?, ?, ?)`,
			kv[0], kv[1], kv[2], kv[3]); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	return db
}

func TestEnrichRegistryFromMetadata_OverridesUUIDFallback(t *testing.T) {
	ctx := context.Background()
	meta := setupMetaWithTranslations(t)

	playlistKnown := "playlist-known-uuid"
	mapKnown := "map-known-uuid"
	playlistUnknown := "playlist-unknown-uuid"

	// Cas A : PlaylistName == PlaylistID (fallback UUID via coalesceStrPtr)
	//   → doit être remplacé par "Quick Play" depuis asset_translations.
	row := &MatchRegistryRow{
		PlaylistID:   strPtrNonEmpty(playlistKnown),
		PlaylistName: strPtrNonEmpty(playlistKnown), // UUID fallback à corriger
		MapID:        strPtrNonEmpty(mapKnown),
		MapName:      strPtrNonEmpty(mapKnown),
	}
	if err := EnrichRegistryFromMetadata(ctx, meta, row); err != nil {
		t.Fatalf("err: %v", err)
	}
	if got := derefSyncStr(row.PlaylistName); got != "Quick Play" {
		t.Errorf("PlaylistName = %q, want %q", got, "Quick Play")
	}
	if got := derefSyncStr(row.MapName); got != "Aquarius" {
		t.Errorf("MapName = %q, want %q", got, "Aquarius")
	}

	// Cas B : PlaylistName déjà rempli avec un vrai nom → on préserve.
	rowB := &MatchRegistryRow{
		PlaylistID:   strPtrNonEmpty(playlistKnown),
		PlaylistName: strPtrNonEmpty("Quick Play"),
	}
	if err := EnrichRegistryFromMetadata(ctx, meta, rowB); err != nil {
		t.Fatalf("err: %v", err)
	}
	if got := derefSyncStr(rowB.PlaylistName); got != "Quick Play" {
		t.Errorf("préservation: PlaylistName = %q, want %q", got, "Quick Play")
	}

	// Cas C : PlaylistID inconnu en asset_translations → on conserve l'UUID
	//   (fallback historique) au lieu d'écrire NULL.
	rowC := &MatchRegistryRow{
		PlaylistID:   strPtrNonEmpty(playlistUnknown),
		PlaylistName: strPtrNonEmpty(playlistUnknown),
	}
	if err := EnrichRegistryFromMetadata(ctx, meta, rowC); err != nil {
		t.Fatalf("err: %v", err)
	}
	if got := derefSyncStr(rowC.PlaylistName); got != playlistUnknown {
		t.Errorf("UUID inconnu: PlaylistName = %q, want UUID préservé %q", got, playlistUnknown)
	}

	// Cas D : PlaylistName nil (extractPublicName a retourné "" et coalesce a
	// fait son boulot) → on doit aussi enrichir.
	rowD := &MatchRegistryRow{
		PlaylistID:   strPtrNonEmpty(playlistKnown),
		PlaylistName: nil,
	}
	if err := EnrichRegistryFromMetadata(ctx, meta, rowD); err != nil {
		t.Fatalf("err: %v", err)
	}
	if got := derefSyncStr(rowD.PlaylistName); got != "Quick Play" {
		t.Errorf("name nil: PlaylistName = %q, want %q", got, "Quick Play")
	}
}

// TestEnrichRegistryFromMetadata_ConstructsPairFromParts couvre le cas du
// nouveau contenu Halo : la PAIRE est absente d'asset_translations (pair_name
// reste un GUID), MAIS le game_variant et la map y sont. Le fallback doit
// construire "{mode} on {map}" pour ne jamais exposer un GUID comme libellé.
func TestEnrichRegistryFromMetadata_ConstructsPairFromParts(t *testing.T) {
	ctx := context.Background()
	meta := setupMetaWithTranslations(t)

	pairAbsent := "pair-absent-uuid" // pas de ligne asset_translations[pair]
	row := &MatchRegistryRow{
		PairID:          strPtrNonEmpty(pairAbsent),
		PairName:        strPtrNonEmpty(pairAbsent), // GUID fallback
		GameVariantID:   strPtrNonEmpty("gv-known-uuid"),
		GameVariantName: strPtrNonEmpty("gv-known-uuid"), // sera résolu en "Slayer"
		MapID:           strPtrNonEmpty("map-known-uuid"),
		MapName:         strPtrNonEmpty("map-known-uuid"), // sera résolu en "Aquarius"
	}
	if err := EnrichRegistryFromMetadata(ctx, meta, row); err != nil {
		t.Fatalf("err: %v", err)
	}
	if got := derefSyncStr(row.PairName); got != "Slayer on Aquarius" {
		t.Errorf("PairName construit = %q, want %q", got, "Slayer on Aquarius")
	}
	// La construction ne doit PAS s'appliquer quand la vraie traduction existe :
	// pair-known-uuid → "Arena:Slayer on Aquarius" (pas la construction).
	rowKnown := &MatchRegistryRow{
		PairID:          strPtrNonEmpty("pair-known-uuid"),
		PairName:        strPtrNonEmpty("pair-known-uuid"),
		GameVariantID:   strPtrNonEmpty("gv-known-uuid"),
		GameVariantName: strPtrNonEmpty("gv-known-uuid"),
		MapID:           strPtrNonEmpty("map-known-uuid"),
		MapName:         strPtrNonEmpty("map-known-uuid"),
	}
	if err := EnrichRegistryFromMetadata(ctx, meta, rowKnown); err != nil {
		t.Fatalf("err: %v", err)
	}
	if got := derefSyncStr(rowKnown.PairName); got != "Arena:Slayer on Aquarius" {
		t.Errorf("PairName traduit = %q, want %q (asset_translations prioritaire)", got, "Arena:Slayer on Aquarius")
	}
}

func TestEnrichRegistryFromMetadata_NilDB_NoError(t *testing.T) {
	row := &MatchRegistryRow{
		PlaylistID:   strPtrNonEmpty("uuid-x"),
		PlaylistName: strPtrNonEmpty("uuid-x"),
	}
	if err := EnrichRegistryFromMetadata(context.Background(), nil, row); err != nil {
		t.Fatalf("nil DB should be no-op, got err: %v", err)
	}
	// Préservation du fallback UUID si pas de metadata.
	if got := derefSyncStr(row.PlaylistName); got != "uuid-x" {
		t.Errorf("PlaylistName = %q, want %q (no override sans metadata)", got, "uuid-x")
	}
}

func TestEnrichRegistryFromMetadata_TableMissing_NoError(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	// Pas de CREATE TABLE asset_translations.
	row := &MatchRegistryRow{
		PlaylistID:   strPtrNonEmpty("uuid-x"),
		PlaylistName: strPtrNonEmpty("uuid-x"),
	}
	if err := EnrichRegistryFromMetadata(context.Background(), db, row); err != nil {
		t.Fatalf("table missing should be no-op, got err: %v", err)
	}
}

func TestNeedsRegistryNameOverride(t *testing.T) {
	id := "abcd-1234"
	tests := []struct {
		name string
		val  *string
		want bool
	}{
		{"nil", nil, true},
		{"empty", strPtrNonEmpty(""), true},
		{"whitespace", strPtrNonEmpty("   "), true},
		{"== id (fallback UUID)", strPtrNonEmpty(id), true},
		{"== id case insensitive", strPtrNonEmpty("ABCD-1234"), true},
		{"vrai nom", strPtrNonEmpty("Quick Play"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := needsRegistryNameOverride(tt.val, id); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func derefSyncStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
