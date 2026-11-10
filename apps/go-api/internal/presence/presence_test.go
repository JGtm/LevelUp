package presence

import (
	"encoding/json"
	"testing"
)

// =============================================================================
// EventParser tests
// =============================================================================

func TestParsePresencePayload_Online_HaloInfinite(t *testing.T) {
	raw := json.RawMessage(`{
		"xuid": "1234567890123456",
		"presenceState": "Online",
		"presenceDetails": [{
			"titleid": "1144039928",
			"titleName": "Halo Infinite",
			"isGame": true,
			"isPrimary": true,
			"device": "PC",
			"state": "Active"
		}]
	}`)
	event, err := ParsePresencePayload(raw, "fallback-xuid")
	if err != nil {
		t.Fatalf("ParsePresencePayload() error = %v", err)
	}
	if event.XUID != "1234567890123456" {
		t.Errorf("XUID = %q, want %q", event.XUID, "1234567890123456")
	}
	if event.PresenceState != "Online" {
		t.Errorf("PresenceState = %q, want %q", event.PresenceState, "Online")
	}
	if event.PresenceDetail == nil {
		t.Fatal("PresenceDetail is nil")
	}
	if event.PresenceDetail.TitleID != "1144039928" {
		t.Errorf("TitleID = %q, want %q", event.PresenceDetail.TitleID, "1144039928")
	}
	if event.PresenceDetail.TitleName != "Halo Infinite" {
		t.Errorf("TitleName = %q", event.PresenceDetail.TitleName)
	}
	if !event.PresenceDetail.IsGame {
		t.Error("IsGame should be true")
	}
}

func TestParsePresencePayload_Offline(t *testing.T) {
	raw := json.RawMessage(`{"presenceState": "Offline"}`)
	event, err := ParsePresencePayload(raw, "xuid-fallback")
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if event.XUID != "xuid-fallback" {
		t.Errorf("XUID = %q, want fallback", event.XUID)
	}
	if event.PresenceState != "Offline" {
		t.Errorf("PresenceState = %q", event.PresenceState)
	}
	if event.PresenceDetail != nil {
		t.Error("PresenceDetail should be nil for offline")
	}
}

func TestParsePresencePayload_MultipleDetails_PrimaryFirst(t *testing.T) {
	raw := json.RawMessage(`{
		"xuid": "X",
		"presenceState": "Online",
		"presenceDetails": [
			{"titleid": "999", "titleName": "Xbox App", "isGame": false, "isPrimary": true, "state": "Active"},
			{"titleid": "1144039928", "titleName": "Halo Infinite", "isGame": true, "isPrimary": true, "state": "Active"},
			{"titleid": "730", "titleName": "CS2", "isGame": true, "isPrimary": false, "state": "Inactive"}
		]
	}`)
	event, err := ParsePresencePayload(raw, "")
	if err != nil {
		t.Fatal(err)
	}
	if event.PresenceDetail == nil {
		t.Fatal("PresenceDetail nil")
	}
	if event.PresenceDetail.TitleID != "1144039928" {
		t.Errorf("got TitleID %q, want 1144039928 (first isPrimary && isGame)", event.PresenceDetail.TitleID)
	}
}

func TestParsePresencePayload_FallbackToFirstGame(t *testing.T) {
	raw := json.RawMessage(`{
		"xuid": "X",
		"presenceState": "Online",
		"presenceDetails": [
			{"titleid": "999", "titleName": "Xbox App", "isGame": false, "isPrimary": true},
			{"titleid": "1144039928", "titleName": "Halo", "isGame": true, "isPrimary": false}
		]
	}`)
	event, err := ParsePresencePayload(raw, "")
	if err != nil {
		t.Fatal(err)
	}
	if event.PresenceDetail == nil {
		t.Fatal("PresenceDetail nil — should fallback to first isGame")
	}
	if event.PresenceDetail.TitleID != "1144039928" {
		t.Errorf("TitleID = %q", event.PresenceDetail.TitleID)
	}
}

func TestParsePresencePayload_InvalidJSON(t *testing.T) {
	_, err := ParsePresencePayload(json.RawMessage(`{invalid`), "x")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParsePresencePayload_EmptyDetails(t *testing.T) {
	raw := json.RawMessage(`{"xuid":"X","presenceState":"Online","presenceDetails":[]}`)
	event, err := ParsePresencePayload(raw, "")
	if err != nil {
		t.Fatal(err)
	}
	if event.PresenceDetail != nil {
		t.Error("PresenceDetail should be nil for empty details")
	}
}

// Format /titles/<TID> + nonce observé en prod 2026-05-25 :
// state + devices[].titles[]. JGtm joue à Halo Infinite ; on doit
// extraire le titre actif depuis devices[0].titles[0].
func TestParsePresencePayload_DevicesFormat_Active(t *testing.T) {
	raw := json.RawMessage(`{"xuid":"2603225103667623","state":"Online","devices":[{"type":"WindowsOneCore","titles":[{"id":"2043073184","name":"Halo Infinite","placement":"Full","state":"Active","lastModified":"2026-05-25T19:53:30.5573644"}]}]}`)
	event, err := ParsePresencePayload(raw, "fallback")
	if err != nil {
		t.Fatalf("ParsePresencePayload() error = %v", err)
	}
	if event.XUID != "2603225103667623" {
		t.Errorf("XUID = %q, want 2603225103667623", event.XUID)
	}
	if event.PresenceState != "Online" {
		t.Errorf("PresenceState = %q, want Online", event.PresenceState)
	}
	if event.PresenceDetail == nil {
		t.Fatal("PresenceDetail nil (parser n'a pas reconnu le format devices[])")
	}
	if event.PresenceDetail.TitleID != "2043073184" {
		t.Errorf("TitleID = %q, want 2043073184", event.PresenceDetail.TitleID)
	}
	if event.PresenceDetail.TitleName != "Halo Infinite" {
		t.Errorf("TitleName = %q, want Halo Infinite", event.PresenceDetail.TitleName)
	}
	if event.PresenceDetail.State != "Active" {
		t.Errorf("State = %q, want Active", event.PresenceDetail.State)
	}
	if !event.PresenceDetail.IsGame {
		t.Error("IsGame should be true (topic /titles/<TID> implique game)")
	}
	if !event.PresenceDetail.IsPrimary {
		t.Error("IsPrimary should be true (placement Full)")
	}
	if event.PresenceDetail.Device != "WindowsOneCore" {
		t.Errorf("Device = %q, want WindowsOneCore", event.PresenceDetail.Device)
	}
}

// Snapshot Offline avec lastSeen : on parse OK mais PresenceDetail reste nil
// (pas de devices[].titles[] actifs).
func TestParsePresencePayload_DevicesFormat_OfflineLastSeen(t *testing.T) {
	raw := json.RawMessage(`{"xuid":"2558964982616721","state":"Offline","lastSeen":{"deviceType":"Win32","titleId":"2043073184","titleName":"Halo Infinite","timestamp":"2026-04-13T21:10:46.8592228"}}`)
	event, err := ParsePresencePayload(raw, "fallback")
	if err != nil {
		t.Fatalf("ParsePresencePayload() error = %v", err)
	}
	if event.PresenceState != "Offline" {
		t.Errorf("PresenceState = %q, want Offline", event.PresenceState)
	}
	if event.PresenceDetail != nil {
		t.Error("PresenceDetail should be nil for lastSeen-only payload")
	}
}

// Fix PR2 2026-05-26 : le bloc `lastSeen` du payload Offline doit être
// extrait dans event.LastSeen pour exposition UI.
func TestParsePresencePayload_LastSeenExtracted(t *testing.T) {
	raw := json.RawMessage(`{"xuid":"X","state":"Offline","lastSeen":{"deviceType":"Win32","titleId":"2043073184","titleName":"Halo Infinite","timestamp":"2026-04-13T21:10:46.8592228"}}`)
	event, err := ParsePresencePayload(raw, "")
	if err != nil {
		t.Fatalf("ParsePresencePayload() error = %v", err)
	}
	if event.LastSeen == nil {
		t.Fatal("LastSeen nil — devrait extraire le bloc lastSeen")
	}
	if event.LastSeen.TitleName != "Halo Infinite" {
		t.Errorf("LastSeen.TitleName = %q, want Halo Infinite", event.LastSeen.TitleName)
	}
	if event.LastSeen.TitleID != "2043073184" {
		t.Errorf("LastSeen.TitleID = %q", event.LastSeen.TitleID)
	}
	if event.LastSeen.DeviceType != "Win32" {
		t.Errorf("LastSeen.DeviceType = %q", event.LastSeen.DeviceType)
	}
	// Timestamp parseable comme UTC.
	if event.LastSeen.Timestamp.IsZero() {
		t.Error("LastSeen.Timestamp non parsée")
	}
	if event.LastSeen.Timestamp.Year() != 2026 {
		t.Errorf("LastSeen.Timestamp.Year = %d, want 2026", event.LastSeen.Timestamp.Year())
	}
}

// Sans bloc lastSeen, event.LastSeen reste nil (ne crash pas).
func TestParsePresencePayload_NoLastSeen(t *testing.T) {
	raw := json.RawMessage(`{"xuid":"X","state":"Online","devices":[{"type":"WindowsOneCore","titles":[{"id":"2043073184","name":"Halo Infinite","state":"Active"}]}]}`)
	event, err := ParsePresencePayload(raw, "")
	if err != nil {
		t.Fatalf("ParsePresencePayload() error = %v", err)
	}
	if event.LastSeen != nil {
		t.Error("LastSeen devrait être nil quand payload n'a pas de bloc lastSeen")
	}
}

// Timestamp avec Z (UTC explicite) — supporté.
func TestParsePresencePayload_LastSeenWithZSuffix(t *testing.T) {
	raw := json.RawMessage(`{"xuid":"X","state":"Offline","lastSeen":{"titleName":"Halo Infinite","timestamp":"2026-04-13T21:10:46Z"}}`)
	event, err := ParsePresencePayload(raw, "")
	if err != nil {
		t.Fatalf("ParsePresencePayload() error = %v", err)
	}
	if event.LastSeen == nil || event.LastSeen.Timestamp.IsZero() {
		t.Fatal("Timestamp avec suffixe Z devrait être parsé")
	}
}
