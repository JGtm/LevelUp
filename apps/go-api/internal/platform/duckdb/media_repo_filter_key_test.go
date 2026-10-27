package duckdb

import (
	"testing"

	"levelup/go-api/internal/i18n"
)

// TestToMediaFileRows_ClesDeFiltreIndependantesDeLaLangue : l'affichage (MapName, ModeName) suit la langue
// de la requête, la clé de filtre (MapID, ModeFilterKey) non : le filtre du dépôt compare le libellé en
// langue par défaut (mediaRowMatchesMode) ou l'identifiant de la carte (mediaRowMatchesMap).
func TestToMediaFileRows_ClesDeFiltreIndependantesDeLaLangue(t *testing.T) {
	// Titre sans pair : le mode vient du libellé résolu par asset_translations (game_variant).
	row := mediaEnrichedRow{
		Cand: mediaCandidateRow{FilePath: "/clips/a.mp4", FileName: "a.mp4", Kind: "video"},
		Match: &mediaMatchRegistryInfo{
			MatchID:          "m1",
			MapID:            "map-1",
			MapName:          i18n.Text{i18n.Default: "Pic de la Foudre", i18n.Fallback: "Lightning Peak"},
			ModeNameFallback: i18n.Text{i18n.Default: "Capture du drapeau", i18n.Fallback: "Capture the Flag"},
		},
	}
	for _, tc := range []struct {
		loc                    i18n.Locale
		wantMap, wantMode, key string
	}{
		{i18n.Default, "Pic de la Foudre", "Capture du drapeau", "Capture du drapeau"},
		{i18n.Fallback, "Lightning Peak", "Capture the Flag", "Capture du drapeau"},
	} {
		got := toMediaFileRows([]mediaEnrichedRow{row}, tc.loc)
		if len(got) != 1 {
			t.Fatalf("%s : %d lignes", tc.loc, len(got))
		}
		r := got[0]
		if r.MapName == nil || *r.MapName != tc.wantMap {
			t.Errorf("%s : MapName = %v, want %q", tc.loc, r.MapName, tc.wantMap)
		}
		if r.ModeName == nil || *r.ModeName != tc.wantMode {
			t.Errorf("%s : ModeName = %v, want %q", tc.loc, r.ModeName, tc.wantMode)
		}
		if r.MapID == nil || *r.MapID != "map-1" {
			t.Errorf("%s : MapID = %v, want map-1 (identique dans toutes les langues)", tc.loc, r.MapID)
		}
		if r.ModeFilterKey == nil || *r.ModeFilterKey != tc.key {
			t.Errorf("%s : ModeFilterKey = %v, want %q (libelle en langue par defaut)", tc.loc, r.ModeFilterKey, tc.key)
		}
	}
}
