package haloclient

import "testing"

// Échantillon réduit de la vraie réponse discovery-infiniteugc (validée 2026-06-12).
const playlistConfigSample = `{
  "PublicName": "Quick Play",
  "RotationEntries": [
    {"AssetId": "0f856108-1e62-4c8e-babb-8c05f7f7753c", "VersionId": "ver-a", "Metadata": {"Weight": 4.17}},
    {"AssetId": "5949df91-77db-498a-9254-760fe1bd4291", "VersionId": "ver-b", "Metadata": {"Weight": 2.0}},
    {"AssetId": "", "VersionId": "skip", "Metadata": {"Weight": 9}}
  ],
  "CustomData": {"PlaylistEntries": [{"MapModePairAssetId": "ignore-si-rotation-presente"}]}
}`

func TestParsePlaylistConfig_RotationEntries(t *testing.T) {
	cfg, err := parsePlaylistConfig([]byte(playlistConfigSample))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cfg.PublicName != "Quick Play" {
		t.Errorf("PublicName = %q, want Quick Play", cfg.PublicName)
	}
	if len(cfg.Entries) != 2 { // l'entrée AssetId vide est ignorée
		t.Fatalf("entries = %d, want 2", len(cfg.Entries))
	}
	if cfg.Entries[0].MapModePairAssetID != "0f856108-1e62-4c8e-babb-8c05f7f7753c" {
		t.Errorf("entry0 id = %q", cfg.Entries[0].MapModePairAssetID)
	}
	if cfg.Entries[0].VersionID != "ver-a" {
		t.Errorf("entry0 version = %q, want ver-a", cfg.Entries[0].VersionID)
	}
	if cfg.Entries[0].Weight != 4.17 {
		t.Errorf("entry0 weight = %v, want 4.17", cfg.Entries[0].Weight)
	}
}

func TestParsePlaylistConfig_FallbackCustomData(t *testing.T) {
	const sample = `{"PublicName":"X","CustomData":{"PlaylistEntries":[{"MapModePairAssetId":"abc"}]}}`
	cfg, err := parsePlaylistConfig([]byte(sample))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(cfg.Entries) != 1 || cfg.Entries[0].MapModePairAssetID != "abc" {
		t.Fatalf("fallback CustomData KO: %+v", cfg.Entries)
	}
}

func TestParsePlaylistConfig_LocalizedName(t *testing.T) {
	const sample = `{"PublicName":{"value":"Arène classée"},"RotationEntries":[]}`
	cfg, err := parsePlaylistConfig([]byte(sample))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cfg.PublicName != "Arène classée" {
		t.Errorf("PublicName localisé = %q", cfg.PublicName)
	}
}
