package assets

import (
	"context"
	"path/filepath"
	"testing"
)

// catalogFixture crée une metadata.duckdb minimale : un titre, une médaille, une carte.
func catalogFixture(t *testing.T) *DuckDBIndexStore {
	t.Helper()
	store := NewDuckDBIndexStore(filepath.Join(t.TempDir(), "metadata.duckdb"))
	t.Cleanup(store.Close)
	db, err := store.openDB()
	if err != nil {
		t.Fatalf("openDB: %v", err)
	}
	ctx := context.Background()
	for _, stmt := range []string{
		`CREATE TABLE medal_definitions (medal_name_id BIGINT)`,
		`INSERT INTO medal_definitions VALUES (42)`,
		`CREATE TABLE maps_catalog (title_slug VARCHAR, map_asset_id VARCHAR)`,
		`INSERT INTO maps_catalog VALUES ('halo_infinite', 'map-known')`,
		`CREATE TABLE asset_translations (asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR, name VARCHAR)`,
		`INSERT INTO asset_translations VALUES ('map-translated', 'map', 'en', 'Recharge')`,
		`INSERT INTO asset_translations VALUES ('pl-1', 'playlist', 'en', 'Quick Play')`,
	} {
		if _, err := db.Exec(ctx, stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	return store
}

func TestMetadataCatalog_Classify(t *testing.T) {
	store := catalogFixture(t)
	titles := map[string]bool{"halo_infinite": true}
	cat := NewMetadataCatalog(store, func(id string) bool { return titles[id] })

	cases := []struct {
		name string
		ref  Ref
		want Verdict
	}{
		{"médaille connue", Ref{Kind: KindMedalImage, TitleID: "halo_infinite", ID: "42"}, VerdictKnown},
		{"médaille inconnue", Ref{Kind: KindMedalImage, TitleID: "halo_infinite", ID: "999"}, VerdictUnknown},
		{"médaille non numérique", Ref{Kind: KindMedalImage, TitleID: "halo_infinite", ID: "abc"}, VerdictUnknown},
		{"titre inconnu", Ref{Kind: KindMedalImage, TitleID: "halo_999", ID: "42"}, VerdictUnknown},
		{"carte du catalogue", Ref{Kind: KindMapImage, TitleID: "halo_infinite", ID: "map-known"}, VerdictKnown},
		{"carte traduite", Ref{Kind: KindMapImage, TitleID: "halo_infinite", ID: "map-translated"}, VerdictKnown},
		{"carte inconnue", Ref{Kind: KindMapImage, TitleID: "halo_infinite", ID: "map-invented"}, VerdictUnknown},
		{"identifiant d'une autre sorte d'asset", Ref{Kind: KindMapImage, TitleID: "halo_infinite", ID: "pl-1"}, VerdictUnknown},
		{"badge de défi : pas de catalogue", Ref{Kind: KindChallengeBadge, TitleID: "halo_infinite", ID: "combat/x"}, VerdictUnverifiable},
		{"image de passe : pas de catalogue", Ref{Kind: KindBPTrackImage, TitleID: "halo_infinite", ID: "/a.png"}, VerdictUnverifiable},
		{"image de joueur : pas de catalogue", Ref{Kind: KindSpartanEmblem, TitleID: "halo_infinite", ID: "x.png"}, VerdictUnverifiable},
		{"kind sans titre connu", Ref{Kind: KindSpartanEmblem, TitleID: "halo_999", ID: "x.png"}, VerdictUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := cat.Classify(context.Background(), tc.ref); got != tc.want {
				t.Fatalf("Classify(%s) = %v, attendu %v", tc.ref, got, tc.want)
			}
		})
	}
}

// Une table absente (métadonnées d'un titre sans médailles) ne fait pas planter la
// classification : l'identifiant reste inconnu.
func TestMetadataCatalog_MissingTableIsUnknown(t *testing.T) {
	store := NewDuckDBIndexStore(filepath.Join(t.TempDir(), "empty.duckdb"))
	t.Cleanup(store.Close)
	cat := NewMetadataCatalog(store, func(string) bool { return true })
	ref := Ref{Kind: KindMedalImage, TitleID: "halo_infinite", ID: "42"}
	if got := cat.Classify(context.Background(), ref); got != VerdictUnknown {
		t.Fatalf("Classify = %v, attendu VerdictUnknown", got)
	}
}
