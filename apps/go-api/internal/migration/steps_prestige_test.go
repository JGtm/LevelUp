//go:build integration

// Tests d'intégration des migrations Prestige (Phase 1).
// Vérifie : application sur DB vierge, idempotence, présence des tables.

package migration

import (
	"database/sql"
	"testing"
)

// hasTable vérifie qu'une table est présente dans information_schema.
func hasTable(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var n int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'main' AND table_name = ?",
		name,
	).Scan(&n)
	if err != nil {
		t.Fatalf("hasTable %s: %v", name, err)
	}
	return n > 0
}

// hasColumn vérifie qu'une colonne est présente sur une table.
func hasColumn(t *testing.T, db *sql.DB, table, column string) bool {
	t.Helper()
	got, err := columnExists(db, table, column)
	if err != nil {
		t.Fatalf("hasColumn %s.%s: %v", table, column, err)
	}
	return got
}

// ─────────────────────────────────────────────────────────────────────────────
// shared_social — tables Prestige (events, user_prestige, squad, squad_challenge)
// ─────────────────────────────────────────────────────────────────────────────

func TestPrestige_SharedSocialMigration_CreatesTables(t *testing.T) {
	db := openMemDB(t)

	if err := RunForDB(db, TargetSharedSocial); err != nil {
		t.Fatalf("RunForDB(SharedSocial): %v", err)
	}

	expected := []string{
		"prestige_events",
		"user_prestige",
		"squad",
		"squad_member",
		"squad_challenge",
		"squad_challenge_participant",
	}
	for _, table := range expected {
		if !hasTable(t, db, table) {
			t.Errorf("table absente après migration: %s", table)
		}
	}

	// Sanity check : colonnes critiques
	if !hasColumn(t, db, "prestige_events", "pp_amount") {
		t.Error("prestige_events.pp_amount manquante")
	}
	if !hasColumn(t, db, "user_prestige", "total_pp") {
		t.Error("user_prestige.total_pp manquante")
	}
	if !hasColumn(t, db, "squad_challenge_participant", "chosen_tier") {
		t.Error("squad_challenge_participant.chosen_tier manquante")
	}
}

func TestPrestige_SharedSocialMigration_Idempotent(t *testing.T) {
	db := openMemDB(t)

	if err := RunForDB(db, TargetSharedSocial); err != nil {
		t.Fatalf("1ère passe: %v", err)
	}
	count1 := countMigrations(t, db)

	if err := RunForDB(db, TargetSharedSocial); err != nil {
		t.Fatalf("2ème passe: %v", err)
	}
	count2 := countMigrations(t, db)

	if err := RunForDB(db, TargetSharedSocial); err != nil {
		t.Fatalf("3ème passe: %v", err)
	}
	count3 := countMigrations(t, db)

	if count1 != count2 || count2 != count3 {
		t.Errorf("idempotence violée: passe1=%d passe2=%d passe3=%d", count1, count2, count3)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// player (stats.duckdb) — tables Prestige côté joueur
// ─────────────────────────────────────────────────────────────────────────────

func TestPrestige_PlayerMigration_CreatesTables(t *testing.T) {
	db := openMemDB(t)

	if err := RunForDB(db, TargetPlayer); err != nil {
		t.Fatalf("RunForDB(Player): %v", err)
	}

	expected := []string{
		"arc",
		"challenge",
		"moment_card",
		"prestige_telemetry",
		"baseline_state",
	}
	for _, table := range expected {
		if !hasTable(t, db, table) {
			t.Errorf("table absente après migration: %s", table)
		}
	}

	// Sanity check : colonnes critiques sur challenge (gros schéma)
	criticalCols := []string{
		"arc_id", "position", "metric", "target", "target_per_member",
		"window_type", "cadence", "eval_type", "mode", "tier", "data_tier",
		"status", "is_private", "last_palier_recompute_at",
	}
	for _, col := range criticalCols {
		if !hasColumn(t, db, "challenge", col) {
			t.Errorf("challenge.%s manquante", col)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// metadata — tables Prestige (templates + preset arcs)
// ─────────────────────────────────────────────────────────────────────────────

func TestPrestige_MetadataMigration_CreatesTables(t *testing.T) {
	db := openMemDB(t)

	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("RunForDB(Metadata): %v", err)
	}

	expected := []string{
		"challenge_template",
		"preset_arc",
		"preset_arc_step",
	}
	for _, table := range expected {
		if !hasTable(t, db, table) {
			t.Errorf("table absente après migration: %s", table)
		}
	}

	// Sanity check : colonnes critiques sur challenge_template
	criticalCols := []string{
		"title_slug", "metric", "cadence", "eval_type", "mode_filter",
		"normal_target", "heroic_target", "legendary_target", "mythic_target",
		"label_en", "label_fr",
	}
	for _, col := range criticalCols {
		if !hasColumn(t, db, "challenge_template", col) {
			t.Errorf("challenge_template.%s manquante", col)
		}
	}
}
