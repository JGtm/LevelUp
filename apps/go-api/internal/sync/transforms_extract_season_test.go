// Package sync — transforms_extract_season_test.go : tests Phase 9.5
//
// Vérifie l'extraction du champ SeasonID depuis le payload Halo (matchInfo["SeasonId"]).
// Phase 9.5 du plan pipeline CSR : sans ce champ populé à l'écriture, la migration
// shared_backfill_is_ranked_and_season serait à re-rejouer à chaque nouveau sync.
package sync

import (
	"testing"
)

func TestExtractRegistry_SeasonIDFromPayload(t *testing.T) {
	j := minimalMatchJSON()
	mi := j["MatchInfo"].(map[string]any)
	mi["SeasonId"] = "CsrSeason13-1"

	row, err := ExtractRegistry(j, "TestSyncer")
	if err != nil {
		t.Fatalf("ExtractRegistry: %v", err)
	}
	if row.SeasonID == nil || *row.SeasonID != "CsrSeason13-1" {
		t.Errorf("SeasonID = %v, want pointer to \"CsrSeason13-1\"", row.SeasonID)
	}
}

func TestExtractRegistry_SeasonIDAbsentPayload(t *testing.T) {
	// Pas de champ SeasonId dans matchInfo → SeasonID nil (sera populé par la
	// migration backfill via dérivation start_time).
	row, err := ExtractRegistry(minimalMatchJSON(), "TestSyncer")
	if err != nil {
		t.Fatalf("ExtractRegistry: %v", err)
	}
	if row.SeasonID != nil {
		t.Errorf("SeasonID should be nil when SeasonId absent ; got %q", *row.SeasonID)
	}
}

func TestExtractRegistry_SeasonIDEmptyString(t *testing.T) {
	// Halo peut retourner SeasonId="" (drift API). Doit être traité comme absent.
	j := minimalMatchJSON()
	mi := j["MatchInfo"].(map[string]any)
	mi["SeasonId"] = ""

	row, err := ExtractRegistry(j, "TestSyncer")
	if err != nil {
		t.Fatalf("ExtractRegistry: %v", err)
	}
	if row.SeasonID != nil {
		t.Errorf("SeasonID should be nil when SeasonId is empty string ; got %q", *row.SeasonID)
	}
}

func TestExtractRegistry_SeasonIDWrongType(t *testing.T) {
	// Halo retourne accidentellement un nombre (drift API). Type-assertion
	// silencieusement ignore (return ""), résultat = nil (no panic).
	j := minimalMatchJSON()
	mi := j["MatchInfo"].(map[string]any)
	mi["SeasonId"] = 42.0

	row, err := ExtractRegistry(j, "TestSyncer")
	if err != nil {
		t.Fatalf("ExtractRegistry: %v", err)
	}
	if row.SeasonID != nil {
		t.Errorf("SeasonID should be nil when SeasonId is wrong type ; got %q", *row.SeasonID)
	}
}
