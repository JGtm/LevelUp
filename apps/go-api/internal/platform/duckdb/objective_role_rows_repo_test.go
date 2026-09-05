//go:build cgo

// Package duckdb — objective_role_rows_repo_test.go : LoadObjectiveRoleRows,
// les lignes projetées par rôle depuis narrative.ObjectiveRoleColumns (chantier
// session-usage S2). Trois propriétés :
//   - la PROJECTION suit la classification narrative — en particulier
//     `flag_grabs` (hors tables de poids) ne compte dans AUCUN rôle, et les
//     ticks de scoring (unité non additive) non plus ;
//   - la lecture passe par la vue `_latest` (ADR 0026) ;
//   - la DÉGRADATION : vue absente ⇒ nil + nil, jamais d'échec de page.
package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/analysis/narrative"
)

func TestObjectiveRoleRows_ProjectionNarrativeEtFlagGrabsExclu(t *testing.T) {
	repo, db := newObjectiveStatsRepoOnMem(t, true)
	insertObjectiveRow(t, db, "m1", "x1", map[string]any{
		"flag_captures": 2, "flag_steals": 1, "flag_grabs": 5,
		"flag_returns": 1, "flag_carriers_killed": 2,
		"time_as_flag_carrier_seconds": 12.5,
	})
	// Zones KOTH (ticks > 0) : les ticks ne comptent dans AUCUN rôle (unité).
	insertObjectiveRow(t, db, "m2", "x1", map[string]any{
		"zone_captures": 3, "zone_secures": 1, "zone_scoring_ticks": 40,
		"time_in_zones_seconds": 55.0,
	})
	// Ligne sans aucun bloc objectif : famille NULL, hors résultat.
	insertObjectiveRow(t, db, "m3", "x1", map[string]any{})

	rows, err := repo.LoadObjectiveRoleRows(context.Background(), []string{"m1", "m2", "m3"})
	if err != nil {
		t.Fatalf("LoadObjectiveRoleRows: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("%d lignes, attendu 2 (la ligne sans famille est écartée) : %+v", len(rows), rows)
	}
	byMatch := map[string]int{}
	for i := range rows {
		byMatch[rows[i].MatchID] = i
	}
	ctf := rows[byMatch["m1"]]
	if ctf.Family != narrative.FamilyCTF {
		t.Errorf("famille m1 = %q, attendu ctf", ctf.Family)
	}
	// PRENDRE = captures (2) + vols (1) — flag_grabs (5) est HORS rôle : s'il
	// comptait, take vaudrait 8 et la part « prendre » du témoin changerait.
	if ctf.Take != 3 {
		t.Errorf("take m1 = %v, attendu 3 (flag_grabs exclu)", ctf.Take)
	}
	if ctf.Defend != 3 {
		t.Errorf("defend m1 = %v, attendu 3 (retours 1 + porteurs abattus 2)", ctf.Defend)
	}
	if ctf.HoldSeconds != 12.5 {
		t.Errorf("hold m1 = %v, attendu 12.5", ctf.HoldSeconds)
	}
	koth := rows[byMatch["m2"]]
	if koth.Family != narrative.FamilyZonesKOTH {
		t.Errorf("famille m2 = %q, attendu zones_koth (ticks > 0)", koth.Family)
	}
	if koth.Take != 3 || koth.Defend != 1 || koth.HoldSeconds != 55 {
		t.Errorf("(take, defend, hold) m2 = (%v, %v, %v), attendu (3, 1, 55) — ticks hors rôles",
			koth.Take, koth.Defend, koth.HoldSeconds)
	}
}

func TestObjectiveRoleRows_LectureParLaVueLatest(t *testing.T) {
	repo, db := newObjectiveStatsRepoOnMem(t, true)
	if _, err := db.Exec(`
		INSERT INTO match_objective_stats (match_id, xuid, flag_captures, written_at)
			VALUES ('m1', 'x1', 1, TIMESTAMP '2026-01-01 00:00:00');
		INSERT INTO match_objective_stats (match_id, xuid, flag_captures, written_at)
			VALUES ('m1', 'x1', 4, TIMESTAMP '2026-01-02 00:00:00');
	`); err != nil {
		t.Fatalf("seed append-only: %v", err)
	}
	rows, err := repo.LoadObjectiveRoleRows(context.Background(), []string{"m1"})
	if err != nil {
		t.Fatalf("LoadObjectiveRoleRows: %v", err)
	}
	if len(rows) != 1 || rows[0].Take != 4 {
		t.Errorf("rows = %+v, attendu UNE ligne take=4 (dernière version, pas 1+4)", rows)
	}
}

func TestObjectiveRoleRows_ScopeVideEtVueAbsente(t *testing.T) {
	repo, _ := newObjectiveStatsRepoOnMem(t, true)
	if rows, err := repo.LoadObjectiveRoleRows(context.Background(), nil); err != nil || rows != nil {
		t.Errorf("scope vide : (%v, %v), attendu (nil, nil) sans requête", rows, err)
	}
	repoNu, _ := newObjectiveStatsRepoOnMem(t, false)
	rows, err := repoNu.LoadObjectiveRoleRows(context.Background(), []string{"m1"})
	if err != nil || rows != nil {
		t.Errorf("vue absente : (%v, %v), attendu (nil, nil) — dégradation best-effort", rows, err)
	}
}
