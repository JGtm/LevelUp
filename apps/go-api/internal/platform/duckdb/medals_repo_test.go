//go:build integration

package duckdb

import (
	"context"
	"testing"
)

// seedMedalsCatalogSchema crée medal_definitions (avec difficulty_index, interrogé
// par MedalsRepo.ListAllMedals) + medal_translations (LEFT JOIN toléré vide).
func seedMedalsCatalogSchema(t *testing.T, db *DB) {
	t.Helper()
	ctx := context.Background()
	stmts := []string{
		`CREATE TABLE medal_definitions (
			medal_name_id    BIGINT PRIMARY KEY,
			name_en          VARCHAR DEFAULT '',
			name_fr          VARCHAR DEFAULT '',
			description_en   VARCHAR DEFAULT '',
			description_fr   VARCHAR DEFAULT '',
			difficulty       VARCHAR DEFAULT '',
			medal_type       VARCHAR DEFAULT '',
			difficulty_index TINYINT,
			personal_score   INTEGER DEFAULT 0
		)`,
		`CREATE TABLE medal_translations (
			medal_name_id BIGINT, lang VARCHAR, name VARCHAR DEFAULT '', description VARCHAR DEFAULT ''
		)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(ctx, s); err != nil {
			t.Fatalf("seedMedalsCatalogSchema: %v", err)
		}
	}
}

func TestMedalsRepo_ListAllMedals(t *testing.T) {
	db := openMemDB(t)
	seedMedalsCatalogSchema(t, db)
	ctx := context.Background()

	if _, err := db.Exec(ctx,
		`INSERT INTO medal_definitions
		    (medal_name_id, name_en, difficulty, medal_type, difficulty_index, personal_score)
		 VALUES
		    (100, 'Killjoy', 'Normal', 'skill', 0, 5),
		    (200, 'Double Kill', 'Heroic', 'multikill', 1, 0)`,
	); err != nil {
		t.Fatalf("insert medals: %v", err)
	}
	// Médaille custom : difficulty vide (→ COALESCE 'Normal'), medal_type vide,
	// difficulty_index NULL (→ 0). Doit rester dans le catalogue (aucune perdue).
	if _, err := db.Exec(ctx,
		`INSERT INTO medal_definitions (medal_name_id, name_en) VALUES (9000000001, 'Avenger')`,
	); err != nil {
		t.Fatalf("insert custom medal: %v", err)
	}

	rows, err := NewMedalsRepo(&PlayerDB{Metadata: db}).ListAllMedals(ctx, "en")
	if err != nil {
		t.Fatalf("ListAllMedals: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("attendu 3 médailles (catalogue complet), got %d", len(rows))
	}
	// Tri par medal_name_id : 100, 200, 9000000001.
	if rows[0].MedalID != 100 || rows[0].Label != "Killjoy" || rows[0].Difficulty != "Normal" ||
		rows[0].MedalType != "skill" || rows[0].DifficultyIndex != 0 || rows[0].PersonalScore != 5 {
		t.Errorf("médaille 100 mal résolue : %+v", rows[0])
	}
	if rows[1].MedalID != 200 || rows[1].DifficultyIndex != 1 || rows[1].MedalType != "multikill" {
		t.Errorf("médaille 200 mal résolue : %+v", rows[1])
	}
	// Custom : difficulty coalescé 'Normal', medal_type vide, difficulty_index 0.
	if rows[2].MedalID != 9000000001 || rows[2].Difficulty != "Normal" || rows[2].MedalType != "" || rows[2].DifficultyIndex != 0 {
		t.Errorf("médaille custom mal résolue : %+v", rows[2])
	}
}

func TestMedalsRepo_LoadMedalTotals(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	const targetXUID = "xuid_medals_777"

	inserts := []struct {
		medalID int64
		matchID string
		count   int
	}{
		{100, "m1", 3},
		{100, "m2", 2}, // même médaille, autre match → SUM = 5
		{200, "m1", 1},
	}
	for _, in := range inserts {
		// On n'insere QUE medal_name_id : c'est la seule colonne reelle de
		// shared.medals_earned (migration steps_shared_core.go). La colonne medal_id
		// du schema de test est fictive ; l'ancien Q36a la lisait par erreur, d'ou le
		// « 0/167 obtenues » en prod. Ce test verrouille la lecture sur medal_name_id.
		if _, err := pdb.Player.Exec(ctx,
			`INSERT INTO shared.medals_earned (medal_name_id, xuid, match_id, count) VALUES (?,?,?,?)`,
			in.medalID, targetXUID, in.matchID, in.count); err != nil {
			t.Fatalf("insert medals_earned: %v", err)
		}
	}

	totals, err := NewMedalsRepo(pdb).LoadMedalTotals(ctx, targetXUID)
	if err != nil {
		t.Fatalf("LoadMedalTotals: %v", err)
	}
	byID := map[int64]int{}
	for _, r := range totals {
		byID[r.MedalID] = r.TotalCount
	}
	if byID[100] != 5 {
		t.Errorf("médaille 100 total = %d, want 5 (SUM sur 2 matchs)", byID[100])
	}
	if byID[200] != 1 {
		t.Errorf("médaille 200 total = %d, want 1", byID[200])
	}
}
