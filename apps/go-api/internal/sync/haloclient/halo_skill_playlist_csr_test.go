package haloclient

import "testing"

// TestParsePlaylistCSR_Ranked vérifie le parsing d'une réponse classée pour le
// bon xuid (et l'ignorance des autres joueurs / ResultCode != 0).
func TestParsePlaylistCSR_Ranked(t *testing.T) {
	body := []byte(`{"Value":[
		{"Id":"xuid(999)","ResultCode":1,"Result":{}},
		{"Id":"xuid(123)","ResultCode":0,"Result":{
			"Current":{"Value":1234,"Tier":"Gold","SubTier":3,"MeasurementMatchesRemaining":0},
			"SeasonMax":{"Value":1300,"Tier":"Platinum","SubTier":1},
			"AllTimeMax":{"Value":1450,"Tier":"Diamond","SubTier":2}}}]}`)

	csr, err := parsePlaylistCSR(body, "pl-arena", "123")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if csr == nil {
		t.Fatal("csr nil, attendu une entrée pour xuid 123")
	}
	if csr.PlaylistID != "pl-arena" {
		t.Errorf("playlist_id = %q, attendu pl-arena", csr.PlaylistID)
	}
	if csr.Current.Tier != "Gold" || csr.Current.SubTier != 3 || csr.Current.Value != 1234 {
		t.Errorf("current = %+v", csr.Current)
	}
	if csr.AllTime.Tier != "Diamond" {
		t.Errorf("alltime tier = %q, attendu Diamond", csr.AllTime.Tier)
	}
}

// TestParsePlaylistCSR_NoEntry : aucune entrée pour le xuid → (nil, nil), la
// lecture catalogue-first synthétisera "Non classé".
func TestParsePlaylistCSR_NoEntry(t *testing.T) {
	body := []byte(`{"Value":[{"Id":"xuid(999)","ResultCode":0,"Result":{"Current":{"Tier":"Onyx"}}}]}`)
	csr, err := parsePlaylistCSR(body, "pl-arena", "123")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if csr != nil {
		t.Errorf("attendu nil (xuid absent), obtenu %+v", csr)
	}
}

// TestParsePlaylistCSR_Unranked : entrée présente mais non classée (placement) →
// renvoyée telle quelle (MeasurementMatchesRemaining porté).
func TestParsePlaylistCSR_Unranked(t *testing.T) {
	body := []byte(`{"Value":[{"Id":"xuid(123)","ResultCode":0,"Result":{
		"Current":{"Value":0,"Tier":"","MeasurementMatchesRemaining":5}}}]}`)
	csr, err := parsePlaylistCSR(body, "pl-arena", "123")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if csr == nil {
		t.Fatal("csr nil, attendu une entrée placement")
	}
	if csr.Current.Tier != "" || csr.Current.MeasurementMatchesRemaining != 5 {
		t.Errorf("placement mal parsé: %+v", csr.Current)
	}
}
