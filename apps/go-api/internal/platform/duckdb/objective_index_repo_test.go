//go:build cgo

// Package duckdb — objective_index_repo_test.go : couverture de
// LoadObjectiveIndexInputs / LoadObjectiveIndexInputsByGamertag (plan
// PLAN_AXE_OBJECTIFS_INDEX étape 4). Harnais partagé avec
// objective_stats_repo_test.go (vraies migrations shared sur :memory:).
package duckdb

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/analysis/narrative"
)

// insertParticipantRow insère une ligne match_participants minimale (jointure
// time_played du calcul d'index).
func insertParticipantRow(t *testing.T, db *sql.DB, matchID, xuid string, tpSeconds int) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO match_participants (match_id, xuid, gamertag, time_played_seconds)
		VALUES (?, ?, ?, ?)`, matchID, xuid, "GT_"+xuid, tpSeconds); err != nil {
		t.Fatalf("insert participant: %v", err)
	}
}

// TestObjectiveIndexRepo_WeightKeysSubsetOfSchema : les clés des tables de poids
// et de durées narrative sont TOUTES des colonnes réelles de
// match_objective_stats — le SELECT étant généré depuis ces tables, une clé
// fantôme casserait la requête en dégradation silencieuse (axe muet sans erreur).
func TestObjectiveIndexRepo_WeightKeysSubsetOfSchema(t *testing.T) {
	_, db := newObjectiveStatsRepoOnMem(t, true)

	rows, err := db.Query(`
		SELECT column_name FROM information_schema.columns
		WHERE table_name = 'match_objective_stats'`)
	if err != nil {
		t.Fatalf("information_schema: %v", err)
	}
	defer rows.Close()
	schema := map[string]bool{}
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			t.Fatalf("scan: %v", err)
		}
		schema[c] = true
	}

	for fam, weights := range narrative.ObjectiveFamilyActionWeights {
		for col := range weights {
			if !schema[col] {
				t.Errorf("famille %s : colonne d'action %q absente du schéma match_objective_stats", fam, col)
			}
		}
	}
	for fam, cols := range narrative.ObjectiveFamilyHoldColumns {
		for _, col := range cols {
			if !schema[col] {
				t.Errorf("famille %s : colonne de durée %q absente du schéma match_objective_stats", fam, col)
			}
		}
	}
	if !schema["times_selected_as_vip"] {
		t.Error("colonne times_selected_as_vip absente du schéma (dénominateur VIP)")
	}
}

// TestObjectiveIndexRepo_FamilySplitAndSums : classification par famille (dont le
// split KOTH/Strongholds par ligne via zone_scoring_ticks), sommes, n_f, durées
// et time_played joints.
func TestObjectiveIndexRepo_FamilySplitAndSums(t *testing.T) {
	repo, db := newObjectiveStatsRepoOnMem(t, true)

	// x1 : 2 matchs CTF + 1 Strongholds (ticks=0) + 1 KOTH (ticks>0) + 1 VIP.
	insertObjectiveRow(t, db, "m1", "x1", map[string]any{
		"flag_captures": 2, "flag_secures": 3, "time_as_flag_carrier_seconds": 30.0,
	})
	insertObjectiveRow(t, db, "m2", "x1", map[string]any{
		"flag_captures": 1, "flag_steals": 2, "time_as_flag_carrier_seconds": 10.0,
	})
	insertObjectiveRow(t, db, "m3", "x1", map[string]any{
		"zone_captures": 4, "zone_scoring_ticks": 0, "time_in_zones_seconds": 100.0,
	})
	insertObjectiveRow(t, db, "m4", "x1", map[string]any{
		"zone_captures": 2, "zone_scoring_ticks": 12, "time_in_zones_seconds": 80.0,
	})
	insertObjectiveRow(t, db, "m5", "x1", map[string]any{
		"vip_kills": 3, "kills_as_vip": 2, "times_selected_as_vip": 2, "time_as_vip_seconds": 60.0,
	})
	for _, mid := range []string{"m1", "m2", "m3", "m4", "m5"} {
		insertParticipantRow(t, db, mid, "x1", 600)
	}

	got, err := repo.LoadObjectiveIndexInputs(context.Background(),
		[]string{"m1", "m2", "m3", "m4", "m5"}, []string{"x1"})
	if err != nil {
		t.Fatalf("LoadObjectiveIndexInputs: %v", err)
	}
	in := got["x1"]
	if in == nil {
		t.Fatal("input x1 absent")
	}

	ctf := in[narrative.FamilyCTF]
	if ctf.Matches != 2 || ctf.TimePlayedSeconds != 1200 {
		t.Errorf("ctf n=%d tp=%v, want n=2 tp=1200", ctf.Matches, ctf.TimePlayedSeconds)
	}
	if ctf.ColumnSums["flag_captures"] != 3 || ctf.ColumnSums["flag_secures"] != 3 || ctf.ColumnSums["flag_steals"] != 2 {
		t.Errorf("ctf sums = %v", ctf.ColumnSums)
	}
	if ctf.HoldSeconds != 40 {
		t.Errorf("ctf hold = %v, want 40", ctf.HoldSeconds)
	}

	sh := in[narrative.FamilyZonesStrongholds]
	if sh.Matches != 1 || sh.ColumnSums["zone_captures"] != 4 || sh.HoldSeconds != 100 {
		t.Errorf("strongholds = %+v", sh)
	}
	if _, hasTicks := sh.ColumnSums["zone_scoring_ticks"]; hasTicks {
		t.Error("strongholds ne doit pas porter zone_scoring_ticks (poids KOTH uniquement)")
	}

	koth := in[narrative.FamilyZonesKOTH]
	if koth.Matches != 1 || koth.ColumnSums["zone_scoring_ticks"] != 12 || koth.HoldSeconds != 80 {
		t.Errorf("koth = %+v", koth)
	}

	vip := in[narrative.FamilyVIP]
	if vip.Matches != 1 || vip.TimesSelectedAsVIP != 2 || vip.HoldSeconds != 60 {
		t.Errorf("vip = %+v", vip)
	}
	if vip.ColumnSums[narrative.ObjectiveColKillsAsVIP] != 2 || vip.ColumnSums["vip_kills"] != 3 {
		t.Errorf("vip sums = %v", vip.ColumnSums)
	}

	// Chaîne complète : l'input alimente ComputeObjectiveIndex sans erreur.
	raw, nObj := narrative.ComputeObjectiveIndex(in)
	if nObj != 5 || raw <= 0 {
		t.Errorf("ComputeObjectiveIndex = (%v, %d), want nObj=5 et raw > 0", raw, nObj)
	}
}

// TestObjectiveIndexRepo_TimePlayedFilter : les lignes à time_played ≤ 30 s sont
// exclues (population alignée sur la calibration P80) ; scope match/xuid respecté.
func TestObjectiveIndexRepo_TimePlayedFilter(t *testing.T) {
	repo, db := newObjectiveStatsRepoOnMem(t, true)

	insertObjectiveRow(t, db, "m1", "x1", map[string]any{"flag_captures": 1})
	insertParticipantRow(t, db, "m1", "x1", 25) // ≤ 30 s → exclu
	insertObjectiveRow(t, db, "m2", "x1", map[string]any{"flag_captures": 5})
	insertParticipantRow(t, db, "m2", "x1", 300)
	// Hors scope match.
	insertObjectiveRow(t, db, "m_out", "x1", map[string]any{"flag_captures": 50})
	insertParticipantRow(t, db, "m_out", "x1", 300)
	// Hors scope xuid.
	insertObjectiveRow(t, db, "m2", "x2", map[string]any{"flag_captures": 7})
	insertParticipantRow(t, db, "m2", "x2", 300)

	got, err := repo.LoadObjectiveIndexInputs(context.Background(),
		[]string{"m1", "m2"}, []string{"x1"})
	if err != nil {
		t.Fatalf("LoadObjectiveIndexInputs: %v", err)
	}
	ctf := got["x1"][narrative.FamilyCTF]
	if ctf.Matches != 1 || ctf.ColumnSums["flag_captures"] != 5 {
		t.Errorf("ctf = %+v, want n=1 flag_captures=5 (ligne ≤30s et hors scope exclues)", ctf)
	}
	if _, ok := got["x2"]; ok {
		t.Error("x2 hors scope xuid présent dans le résultat")
	}
}

// TestObjectiveIndexRepo_ByGamertag : résolution via shared.xuid_aliases (coéquipier
// non suivi, sans player DB).
func TestObjectiveIndexRepo_ByGamertag(t *testing.T) {
	repo, db := newObjectiveStatsRepoOnMem(t, true)

	if _, err := db.Exec(`INSERT INTO xuid_aliases (xuid, gamertag) VALUES ('x9', 'MateGT')`); err != nil {
		t.Fatalf("seed alias: %v", err)
	}
	insertObjectiveRow(t, db, "m1", "x9", map[string]any{"skull_grabs": 4, "time_as_skull_carrier_seconds": 20.0})
	insertParticipantRow(t, db, "m1", "x9", 480)

	got, err := repo.LoadObjectiveIndexInputsByGamertag(context.Background(), []string{"m1"}, "MateGT")
	if err != nil {
		t.Fatalf("LoadObjectiveIndexInputsByGamertag: %v", err)
	}
	odd := got[narrative.FamilyOddball]
	if odd.Matches != 1 || odd.ColumnSums["skull_grabs"] != 4 || odd.HoldSeconds != 20 {
		t.Errorf("oddball = %+v", odd)
	}

	// Gamertag inconnu → input vide, pas d'erreur.
	empty, err := repo.LoadObjectiveIndexInputsByGamertag(context.Background(), []string{"m1"}, "Inconnu")
	if err != nil || len(empty) != 0 {
		t.Errorf("gamertag inconnu = (%v, %v), want (vide, nil)", empty, err)
	}
}

// TestObjectiveIndexRepo_MissingViewDegrades : vue absente (DB non migrée / titre
// sans capability) → map vide + err nil, l'axe Objectif est simplement retiré.
func TestObjectiveIndexRepo_MissingViewDegrades(t *testing.T) {
	repo, _ := newObjectiveStatsRepoOnMem(t, false)

	got, err := repo.LoadObjectiveIndexInputs(context.Background(), []string{"m1"}, []string{"x1"})
	if err != nil {
		t.Fatalf("dégradation attendue silencieuse, err = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %v, want map vide", got)
	}
}
