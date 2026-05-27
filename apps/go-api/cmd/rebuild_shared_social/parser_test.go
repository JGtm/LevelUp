package main

import "testing"

func TestDiffCounts_ZeroDelta(t *testing.T) {
	baseline := []tableCount{
		{"media_files", 121},
		{"media_likes", 1},
		{"match_favorites", 3},
	}
	final := []tableCount{
		{"media_files", 121},
		{"media_likes", 1},
		{"match_favorites", 3},
	}
	if d := diffCounts(baseline, final); len(d) != 0 {
		t.Errorf("aucune différence attendue, got %d diffs: %+v", len(d), d)
	}
}

func TestDiffCounts_DataLoss(t *testing.T) {
	baseline := []tableCount{{"media_likes", 5}}
	final := []tableCount{{"media_likes", 2}}
	d := diffCounts(baseline, final)
	if len(d) != 1 {
		t.Fatalf("attendu 1 diff, got %d", len(d))
	}
	if d[0].name != "media_likes" {
		t.Errorf("attendu media_likes, got %q", d[0].name)
	}
}

func TestDiffCounts_LegacyBakIgnored(t *testing.T) {
	// Les tables media_match_associations_bak_* sont des backups manuels
	// legacy. Elles doivent être ignorées car le rebuild ne les recrée pas.
	baseline := []tableCount{
		{"media_files", 100},
		{"media_match_associations_bak_20260426T103338Z", 66},
	}
	final := []tableCount{
		{"media_files", 100},
	}
	d := diffCounts(baseline, final)
	if len(d) != 0 {
		t.Errorf("legacy bak doit être ignorée, got %d diffs: %+v", len(d), d)
	}
}

func TestDiffCounts_NewTableFromMigration(t *testing.T) {
	// Une table créée par les migrations (ex: nouvelle table dans une
	// version récente) doit être signalée comme "nouvelle".
	baseline := []tableCount{{"media_files", 100}}
	final := []tableCount{
		{"media_files", 100},
		{"new_table_v2", 0},
	}
	d := diffCounts(baseline, final)
	if len(d) != 1 {
		t.Fatalf("1 diff attendue, got %d: %+v", len(d), d)
	}
	if d[0].name != "new_table_v2" {
		t.Errorf("attendu new_table_v2, got %q", d[0].name)
	}
}

func TestDiffCounts_TableDisappeared(t *testing.T) {
	// Une table présente en baseline qui n'est pas dans final (et qui n'est
	// PAS un legacy bak_*) doit être signalée — ne JAMAIS perdre de la
	// data user-facing en silence.
	baseline := []tableCount{
		{"media_files", 100},
		{"player_records", 50},
	}
	final := []tableCount{
		{"media_files", 100},
	}
	d := diffCounts(baseline, final)
	if len(d) != 1 {
		t.Fatalf("1 diff attendue, got %d", len(d))
	}
	if d[0].name != "player_records" {
		t.Errorf("attendu player_records, got %q", d[0].name)
	}
}
