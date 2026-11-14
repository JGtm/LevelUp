//go:build integration

package migrations

import (
	"testing"

	"levelup/go-api/internal/migration"
)

// Les records de précision rangés en pourcentage deviennent des ratios par des lignes neuves ; les
// autres métriques, les valeurs déjà en ratio et celles au-dessus de 100 ne bougent pas, aucune ligne
// n'est supprimée.
func TestAccuracyRecordsToRatio(t *testing.T) {
	db := freshProvision(t, migration.TargetSharedSocial)
	for _, r := range []struct {
		xuid, metric, period string
		value                float64
	}{
		{"x1", "accuracy", "all_time", 73.333336},
		{"x1", "accuracy", "30d", 69.64},
		{"x2", "accuracy", "all_time", 0.61},
		{"x3", "accuracy", "all_time", 120},
		{"x1", "kda", "all_time", 3.5},
	} {
		if _, err := db.Exec(`INSERT INTO player_records_history (xuid, metric, period, value, previous_value)
			VALUES (?, ?, ?, ?, ?)`, r.xuid, r.metric, r.period, r.value, r.value-1); err != nil {
			t.Fatalf("seed %v: %v", r, err)
		}
	}
	if _, err := db.Exec(`DELETE FROM schema_migrations WHERE name = 'shared_social_accuracy_records_ratio_v1'`); err != nil {
		t.Fatalf("retrait du step du registre: %v", err)
	}

	if err := migration.RunForDB(db, migration.TargetSharedSocial); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}

	latest := func(xuid, metric, period string) float64 {
		var v float64
		if err := db.QueryRow(`SELECT value FROM player_records_latest WHERE xuid=? AND metric=? AND period=?`,
			xuid, metric, period).Scan(&v); err != nil {
			t.Fatalf("latest %s/%s/%s: %v", xuid, metric, period, err)
		}
		return v
	}
	if v := latest("x1", "accuracy", "all_time"); v < 0.7333 || v > 0.7334 {
		t.Errorf("x1 all_time = %v, attendu 0,73333", v)
	}
	if v := latest("x1", "accuracy", "30d"); v != 0.6964 {
		t.Errorf("x1 30d = %v, attendu 0,6964", v)
	}
	if v := latest("x2", "accuracy", "all_time"); v != 0.61 {
		t.Errorf("x2 (déjà en ratio) = %v, attendu 0,61", v)
	}
	if v := latest("x3", "accuracy", "all_time"); v != 120 {
		t.Errorf("x3 (au-dessus de 100) = %v, attendu 120 intact", v)
	}
	if v := latest("x1", "kda", "all_time"); v != 3.5 {
		t.Errorf("kda = %v, attendu 3,5 intact", v)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM player_records_history`).Scan(&n); err != nil || n != 7 {
		t.Errorf("player_records_history = %d lignes (err %v), attendu 7 : 5 d'origine + 2 conversions", n, err)
	}
}
